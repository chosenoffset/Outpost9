package game

import (
	"image/color"
	"log"

	"chosenoffset.com/outpost9/internal/render"
)

// Draw renders the game to the screen.
func (g *Game) Draw(screen render.Image) {
	w, h := screen.Size()

	// Ensure render textures exist and are the right size
	if g.SceneTexture == nil || needsResize(g.SceneTexture, w, h) {
		if g.SceneTexture != nil {
			g.SceneTexture.Dispose()
		}
		g.SceneTexture = g.Renderer.NewImage(w, h)
	}
	if g.WallTexture == nil || needsResize(g.WallTexture, w, h) {
		if g.WallTexture != nil {
			g.WallTexture.Dispose()
		}
		g.WallTexture = g.Renderer.NewImage(w, h)
	}

	// Step 1: Clear and render the scene to an offscreen texture
	g.SceneTexture.Clear()
	g.drawFloorsOnly(g.SceneTexture)
	g.drawFurnishings(g.SceneTexture)
	g.drawAllWalls(g.SceneTexture)
	g.drawEntities(g.SceneTexture)
	g.drawPlayer(g.SceneTexture)

	// Step 2: Render walls to wall texture for occlusion testing
	g.WallTexture.Clear()
	g.drawWallsToTexture(g.WallTexture)

	// Step 3: Apply lighting shader
	g.applyLightingShader(screen)

	// Step 4: Draw UI elements on top (unaffected by lighting)
	g.drawUI(screen)
	g.drawHUD(screen)
	g.drawNarrativePanel(screen)
}

func needsResize(img render.Image, w, h int) bool {
	bounds := img.Bounds()
	return bounds.Dx() != w || bounds.Dy() != h
}

func (g *Game) drawFloorsOnly(screen render.Image) {
	if g.GameMap == nil || g.GameMap.Atlas == nil {
		return
	}

	tileSize := g.GameMap.Data.TileSize
	for y := 0; y < g.GameMap.Data.Height; y++ {
		for x := 0; x < g.GameMap.Data.Width; x++ {
			tileName, err := g.GameMap.GetTileAt(x, y)
			if err != nil || tileName == "" {
				continue
			}

			tile, ok := g.GameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			// Only draw floors
			if tile.GetTilePropertyBool("walkable", false) {
				screenX := float64(x*tileSize) - g.Camera.X
				screenY := float64(y*tileSize) - g.Camera.Y
				g.GameMap.Atlas.DrawTileDef(screen, tile, screenX, screenY)
			}
		}
	}
}

func (g *Game) drawFurnishings(screen render.Image) {
	if g.GameMap == nil || g.ObjectsAtlas == nil {
		return
	}

	tileSize := g.GameMap.Data.TileSize
	for _, pf := range g.GameMap.Data.PlacedFurnishings {
		if pf.Definition == nil {
			continue
		}
		tile, ok := g.ObjectsAtlas.GetTile(pf.Definition.TileName)
		if !ok {
			continue
		}
		screenX := float64(pf.X*tileSize) - g.Camera.X
		screenY := float64(pf.Y*tileSize) - g.Camera.Y
		g.ObjectsAtlas.DrawTileDef(screen, tile, screenX, screenY)
	}
}

func (g *Game) drawAllWalls(screen render.Image) {
	if g.GameMap == nil || g.GameMap.Atlas == nil {
		return
	}

	tileSize := g.GameMap.Data.TileSize
	for y := 0; y < g.GameMap.Data.Height; y++ {
		for x := 0; x < g.GameMap.Data.Width; x++ {
			tileName, err := g.GameMap.GetTileAt(x, y)
			if err != nil || tileName == "" {
				continue
			}

			tile, ok := g.GameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			// Only draw walls (non-walkable tiles)
			if !tile.GetTilePropertyBool("walkable", false) {
				screenX := float64(x*tileSize) - g.Camera.X
				screenY := float64(y*tileSize) - g.Camera.Y
				g.GameMap.Atlas.DrawTileDef(screen, tile, screenX, screenY)
			}
		}
	}
}

func (g *Game) drawWallsToTexture(texture render.Image) {
	// Same as drawAllWalls but to the wall texture
	g.drawAllWalls(texture)
}

func (g *Game) drawEntities(screen render.Image) {
	if g.TurnManager == nil || g.EntitiesAtlas == nil || g.GameMap == nil {
		return
	}

	tileSize := g.GameMap.Data.TileSize
	for _, ent := range g.TurnManager.GetEnemies() {
		if !ent.IsAlive() {
			continue
		}

		screenX := float64(ent.X*tileSize) + float64(tileSize)/2 - g.Camera.X
		screenY := float64(ent.Y*tileSize) + float64(tileSize)/2 - g.Camera.Y

		// Try to get entity sprite from the entity's sprite name
		spriteName := ent.SpriteName
		if spriteName != "" {
			tile, ok := g.EntitiesAtlas.GetTile(spriteName)
			if ok {
				spriteSize := 32.0
				opts := &render.DrawImageOptions{}
				opts.GeoM = render.NewGeoM()
				opts.GeoM.Translate(screenX-spriteSize/2, screenY-spriteSize/2)
				img := g.EntitiesAtlas.GetTileSubImage(tile)
				if img != nil {
					screen.DrawImage(img, opts)
					continue
				}
			}
		}

		// Fallback to circle
		g.Renderer.FillCircle(screen, float32(screenX), float32(screenY), 12, color.RGBA{255, 100, 100, 255})
	}
}

func (g *Game) drawPlayer(screen render.Image) {
	playerScreenX := g.Player.Pos.X - g.Camera.X
	playerScreenY := g.Player.Pos.Y - g.Camera.Y

	if g.PlayerSpriteImg != nil {
		spriteSize := 32.0
		opts := &render.DrawImageOptions{}
		opts.GeoM = render.NewGeoM()
		opts.GeoM.Translate(playerScreenX-spriteSize/2, playerScreenY-spriteSize/2)
		screen.DrawImage(g.PlayerSpriteImg, opts)
	} else {
		g.Renderer.FillCircle(screen, float32(playerScreenX), float32(playerScreenY), 14, color.RGBA{255, 255, 100, 255})
		g.Renderer.StrokeCircle(screen, float32(playerScreenX), float32(playerScreenY), 14, 2, color.RGBA{200, 200, 50, 255})
	}
}

func (g *Game) applyLightingShader(screen render.Image) {
	if g.LightingShader == nil || g.LightingManager == nil {
		// No lighting - just copy scene to screen
		opts := &render.DrawImageOptions{}
		screen.DrawImage(g.SceneTexture, opts)
		return
	}

	lights := g.LightingManager.GetAllLights()
	g.FrameCount++
	if g.FrameCount <= 5 {
		log.Printf("DEBUG Frame %d: Rendering with %d lights", g.FrameCount, len(lights))
	}

	// Prepare shader uniforms
	const maxLights = 32
	var lightPositions [maxLights * 2]float32
	var lightProperties [maxLights * 4]float32
	var lightColors [maxLights * 3]float32

	numLights := len(lights)
	if numLights > maxLights {
		numLights = maxLights
	}

	for i := 0; i < numLights; i++ {
		light := lights[i]
		lightPositions[i*2] = float32(light.X)
		lightPositions[i*2+1] = float32(light.Y)
		lightProperties[i*4] = float32(light.Radius)
		lightProperties[i*4+1] = float32(light.Intensity)
		lightProperties[i*4+2] = 0.0
		lightProperties[i*4+3] = 1.0
		lightColors[i*3] = float32(light.Color.R) / 255.0
		lightColors[i*3+1] = float32(light.Color.G) / 255.0
		lightColors[i*3+2] = float32(light.Color.B) / 255.0
	}

	w, h := screen.Size()
	opts := &render.DrawRectShaderOptions{
		Uniforms: map[string]interface{}{
			"NumLights":       float32(numLights),
			"AmbientLight":    float32(g.LightingManager.GetAmbientLight()),
			"CameraOffset":    []float32{float32(g.Camera.X), float32(g.Camera.Y)},
			"LightPositions":  lightPositions[:],
			"LightProperties": lightProperties[:],
			"LightColors":     lightColors[:],
		},
	}
	opts.Images[0] = g.SceneTexture
	opts.Images[1] = g.WallTexture

	screen.DrawRectShader(w, h, g.LightingShader, opts)
}

func (g *Game) drawUI(screen render.Image) {
	// Draw on-screen messages
	y := 50.0
	for _, msg := range g.Messages {
		alpha := uint8(255 * (msg.TimeLeft / msg.MaxTime))
		g.Renderer.DrawText(screen, msg.Text, 20, int(y), color.RGBA{255, 255, 255, alpha}, 1.0)
		y += 20
	}
}

func (g *Game) drawHUD(screen render.Image) {
	// HUD drawing requires ebiten.Image - handled in main for now
	// TODO: Abstract HUD to use render.Image
}

func (g *Game) drawNarrativePanel(screen render.Image) {
	// NarrativePanel drawing requires ebiten.Image - handled in main for now
	// TODO: Abstract NarrativePanel to use render.Image
}
