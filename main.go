package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"log"

	"chosenoffset.com/outpost9/atlas"
	"chosenoffset.com/outpost9/gamescanner"
	"chosenoffset.com/outpost9/maploader"
	"chosenoffset.com/outpost9/menu"
	"chosenoffset.com/outpost9/renderer"
	ebitenrenderer "chosenoffset.com/outpost9/renderer/ebiten"
	"chosenoffset.com/outpost9/room"
	"chosenoffset.com/outpost9/shadows"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed shaders/sight_shadows.kage
var shadowShaderSrc []byte

type Player struct {
	Pos   shadows.Point
	Speed float64
}

// GameManager handles the overall game state, including menu and gameplay.
type GameManager struct {
	screenWidth  int
	screenHeight int
	state        menu.GameState
	mainMenu     *menu.MainMenu
	game         *Game
	renderer     renderer.Renderer
	inputMgr     renderer.InputManager
	loader       renderer.ResourceLoader
}

func (gm *GameManager) Update() error {
	switch gm.state {
	case menu.StateMainMenu:
		selected, selection := gm.mainMenu.Update()
		if selected {
			// Load the selected game
			if err := gm.loadGame(selection); err != nil {
				log.Printf("Failed to load game: %v", err)
				return err
			}
			gm.state = menu.StatePlaying
		}
	case menu.StatePlaying:
		if gm.game != nil {
			// Check for ESC to return to menu
			if gm.inputMgr.IsKeyPressed(renderer.KeyEscape) {
				gm.state = menu.StateMainMenu
			}
			return gm.game.Update()
		}
	}
	return nil
}

func (gm *GameManager) Draw(screen renderer.Image) {
	switch gm.state {
	case menu.StateMainMenu:
		gm.mainMenu.Draw(screen)
	case menu.StatePlaying:
		if gm.game != nil {
			gm.game.Draw(screen)
		}
	}
}

func (gm *GameManager) Layout(outsideWidth, outsideHeight int) (int, int) {
	return gm.screenWidth, gm.screenHeight
}

func (gm *GameManager) loadGame(selection menu.Selection) error {
	libraryPath := fmt.Sprintf("data/%s/%s", selection.GameDir, selection.RoomLibraryFile)

	// Load procedurally generated level from room library
	log.Printf("Loading room library: %s", libraryPath)

	config := room.GeneratorConfig{
		MinRooms:     5,
		MaxRooms:     10,
		Seed:         0, // Use random seed each time
		ConnectAll:   true,
		AllowOverlap: false,
	}

	gameMap, err := maploader.LoadMapFromRoomLibrary(libraryPath, config, gm.loader)
	if err != nil {
		return fmt.Errorf("failed to generate map from room library: %w", err)
	}

	log.Printf("Generated procedural map: %s (%dx%d, tile size: %dpx)",
		gameMap.Data.Name,
		gameMap.Data.Width,
		gameMap.Data.Height,
		gameMap.Data.TileSize)

	// Generate wall segments from map data
	walls := shadows.CreateWallSegmentsFromMap(gameMap)
	log.Printf("Generated %d wall segments", len(walls))

	// Load entities atlas for player, enemies, items, etc.
	entitiesAtlasPath := fmt.Sprintf("data/%s/entities.json", selection.GameDir)
	entitiesAtlas, err := atlas.LoadAtlas(entitiesAtlasPath, gm.loader)
	if err != nil {
		log.Printf("Warning: Failed to load entities atlas: %v", err)
		// Continue without entities atlas - will use fallback rendering
	}

	// Extract player sprite if atlas loaded successfully
	var playerSprite renderer.Image
	if entitiesAtlas != nil {
		playerSprite, err = entitiesAtlas.GetTileSubImageByName("player_idle")
		if err != nil {
			log.Printf("Warning: Failed to get player sprite: %v", err)
		}
	}

	// Initialize shadow shader
	shadowShader, err := ebiten.NewShader(shadowShaderSrc)
	if err != nil {
		return fmt.Errorf("failed to create shadow shader: %w", err)
	}

	// Create wall texture render target
	wallTexture := ebiten.NewImage(gm.screenWidth, gm.screenHeight)

	gm.game = &Game{
		screenWidth:     gm.screenWidth,
		screenHeight:    gm.screenHeight,
		gameMap:         gameMap,
		walls:           walls,
		player: Player{
			Pos:   shadows.Point{X: gameMap.Data.PlayerSpawn.X, Y: gameMap.Data.PlayerSpawn.Y},
			Speed: 3.0,
		},
		renderer:        gm.renderer,
		inputMgr:        gm.inputMgr,
		entitiesAtlas:   entitiesAtlas,
		playerSpriteImg: playerSprite,
		shadowShader:    shadowShader,
		wallTexture:     wallTexture,
	}

	return nil
}

type Game struct {
	screenWidth     int
	screenHeight    int
	gameMap         *maploader.Map
	walls           []shadows.Segment
	player          Player
	whiteImg        renderer.Image
	renderer        renderer.Renderer
	inputMgr        renderer.InputManager
	entitiesAtlas   *atlas.Atlas
	playerSpriteImg renderer.Image
	shadowShader    *ebiten.Shader
	wallTexture     *ebiten.Image // Render target containing just walls
}

func (g *Game) Update() error {
	// WASD movement
	moveSpeed := g.player.Speed

	if g.inputMgr.IsKeyPressed(renderer.KeyW) {
		g.player.Pos.Y -= moveSpeed
	}
	if g.inputMgr.IsKeyPressed(renderer.KeyS) {
		g.player.Pos.Y += moveSpeed
	}
	if g.inputMgr.IsKeyPressed(renderer.KeyA) {
		g.player.Pos.X -= moveSpeed
	}
	if g.inputMgr.IsKeyPressed(renderer.KeyD) {
		g.player.Pos.X += moveSpeed
	}

	// Keep player in bounds
	if g.player.Pos.X < 0 {
		g.player.Pos.X = 0
	}
	if g.player.Pos.X > float64(g.screenWidth) {
		g.player.Pos.X = float64(g.screenWidth)
	}
	if g.player.Pos.Y < 0 {
		g.player.Pos.Y = 0
	}
	if g.player.Pos.Y > float64(g.screenHeight) {
		g.player.Pos.Y = float64(g.screenHeight)
	}

	return nil
}

func (g *Game) Draw(screen renderer.Image) {
	// Step 1: Draw all tiles (floor and walls) to the screen
	g.drawTiles(screen)

	// Step 2: Render ONLY walls to wallTexture for shader input
	// The shader will sample this texture to detect wall pixels
	g.wallTexture.Clear()

	// DEBUG: Draw a test rectangle to verify texture rendering works
	testImg := ebiten.NewImage(100, 100)
	testImg.Fill(color.RGBA{255, 255, 255, 255}) // White with full alpha
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(200, 200) // Draw at 200,200
	g.wallTexture.DrawImage(testImg, opts)

	g.drawWallsToTexture(g.wallTexture)

	// DEBUG: Draw wall texture directly to screen to see what's in it
	if ebitenImg, ok := screen.(*ebitenrenderer.EbitenImage); ok {
		ebitenScreen := ebitenImg.GetEbitenImage()
		drawOpts := &ebiten.DrawImageOptions{}
		drawOpts.ColorScale.ScaleAlpha(0.5)
		ebitenScreen.DrawImage(g.wallTexture, drawOpts)
	}

	// Step 3: Apply shadow shader - does pixel-perfect raycasting on GPU
	// DISABLED FOR DEBUG
	/*
	// Each pixel checks: is there a wall between me and the player?
	// Extract underlying ebiten.Image to use shader (renderer abstraction doesn't support shaders yet)
	if ebitenImg, ok := screen.(*ebitenrenderer.EbitenImage); ok && g.shadowShader != nil {
		ebitenScreen := ebitenImg.GetEbitenImage()

		// Shader options with player position uniform
		opts := &ebiten.DrawRectShaderOptions{}
		opts.Uniforms = map[string]interface{}{
			"PlayerPos":   []float32{float32(g.player.Pos.X), float32(g.player.Pos.Y)},
			"MaxDistance": float32(g.screenWidth + g.screenHeight),
		}
		opts.Images[0] = g.wallTexture

		// Draw shader over entire screen
		ebitenScreen.DrawRectShader(g.screenWidth, g.screenHeight, g.shadowShader, opts)
	}
	*/

	// Step 4: Draw player character on top of shadows
	if g.playerSpriteImg != nil {
		// Draw player sprite centered on player position
		spriteSize := 16.0 // Tile size from atlas
		opts := &renderer.DrawImageOptions{}
		opts.GeoM = renderer.NewGeoM()
		opts.GeoM.Translate(g.player.Pos.X-spriteSize/2, g.player.Pos.Y-spriteSize/2)
		screen.DrawImage(g.playerSpriteImg, opts)
	} else {
		// Fallback to circle if sprite not loaded
		g.renderer.FillCircle(screen,
			float32(g.player.Pos.X),
			float32(g.player.Pos.Y),
			8,
			color.RGBA{255, 255, 100, 255})

		g.renderer.StrokeCircle(screen,
			float32(g.player.Pos.X),
			float32(g.player.Pos.Y),
			8,
			2,
			color.RGBA{200, 200, 50, 255})
	}
}

func (g *Game) drawTiles(screen renderer.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize

	// Pass 1: Draw floor layer (fills entire map with floor tile)
	if g.gameMap.Data.FloorTile != "" {
		floorTile, ok := g.gameMap.Atlas.GetTile(g.gameMap.Data.FloorTile)
		if ok {
			floorImg := g.gameMap.Atlas.GetTileSubImage(floorTile)
			for y := 0; y < g.gameMap.Data.Height; y++ {
				for x := 0; x < g.gameMap.Data.Width; x++ {
					screenX := float64(x * tileSize)
					screenY := float64(y * tileSize)

					opts := &renderer.DrawImageOptions{}
					opts.GeoM = renderer.NewGeoM()
					opts.GeoM.Translate(screenX, screenY)
					screen.DrawImage(floorImg, opts)
				}
			}
		}
	}

	// Pass 2: Draw walls/objects layer (only non-empty tiles)
	for y := 0; y < g.gameMap.Data.Height; y++ {
		for x := 0; x < g.gameMap.Data.Width; x++ {
			tileName, err := g.gameMap.GetTileAt(x, y)
			if err != nil {
				continue
			}

			// Skip empty tiles (let floor show through)
			if tileName == "" {
				continue
			}

			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			subImg := g.gameMap.Atlas.GetTileSubImage(tile)

			screenX := float64(x * tileSize)
			screenY := float64(y * tileSize)

			opts := &renderer.DrawImageOptions{}
			opts.GeoM = renderer.NewGeoM()
			opts.GeoM.Translate(screenX, screenY)

			screen.DrawImage(subImg, opts)
		}
	}
}

// drawWallsToTexture renders only wall tiles (sight-blocking) to a texture for shader input
// The shader will sample this texture's alpha channel to detect walls during raycasting
func (g *Game) drawWallsToTexture(texture *ebiten.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		log.Println("DEBUG: gameMap or Atlas is nil")
		return
	}

	tileSize := g.gameMap.Data.TileSize
	wallCount := 0

	// Draw only sight-blocking tiles to the texture
	for y := 0; y < g.gameMap.Data.Height; y++ {
		for x := 0; x < g.gameMap.Data.Width; x++ {
			// Check if this tile blocks sight
			if !g.gameMap.BlocksSight(x, y) {
				continue
			}

			wallCount++

			// Get the wall tile
			tileName, err := g.gameMap.GetTileAt(x, y)
			if err != nil || tileName == "" {
				log.Printf("DEBUG: Wall at %d,%d has empty tileName or error: %v", x, y, err)
				continue
			}

			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				log.Printf("DEBUG: Failed to get tile '%s' from atlas", tileName)
				continue
			}

			// Store for potential later use
			_ = g.gameMap.Atlas.GetTileSubImage(tile)

			screenX := float64(x * tileSize)
			screenY := float64(y * tileSize)

			// DEBUG: Draw solid rectangles instead of actual tiles to verify positions
			wallRect := ebiten.NewImage(tileSize, tileSize)
			wallRect.Fill(color.RGBA{255, 0, 255, 255}) // Magenta with full alpha
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(screenX, screenY)
			texture.DrawImage(wallRect, opts)

			// Log first wall
			if wallCount == 1 {
				log.Printf("DEBUG: First wall '%s' at tile (%d,%d) -> screen (%.0f,%.0f), tileSize: %d",
					tileName, x, y, screenX, screenY, tileSize)
			}
		}
	}

	log.Printf("DEBUG: drawWallsToTexture rendered %d wall tiles", wallCount)
}

func (g *Game) drawVisibleWalls(screen renderer.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize

	// Draw wall tiles that are visible (not in shadow)
	// Check each wall tile to see if it's obscured by shadows
	drawnTiles := make(map[string]bool) // Track which tiles we've already drawn

	for _, wall := range g.walls {
		if shadows.IsFacingPoint(wall, g.player.Pos) {
			// Iterate through all tiles covered by this segment (for merged segments)
			tilesToDraw := wall.TilesCovered
			if len(tilesToDraw) == 0 {
				// Fallback to single tile if TilesCovered is empty (shouldn't happen with new code)
				tilesToDraw = []shadows.Coord{{X: wall.TileX, Y: wall.TileY}}
			}

			for _, tileCoord := range tilesToDraw {
				tileKey := fmt.Sprintf("%d,%d", tileCoord.X, tileCoord.Y)
				if drawnTiles[tileKey] {
					continue // Already drew this tile
				}

				// Check if this tile is in shadow by testing if the tile center is visible
				tileCenterX := float64(tileCoord.X)*float64(tileSize) + float64(tileSize)/2
				tileCenterY := float64(tileCoord.Y)*float64(tileSize) + float64(tileSize)/2

				if g.isPointInShadow(shadows.Point{tileCenterX, tileCenterY}) {
					continue // This wall is in shadow, don't redraw it
				}

				// Get the wall tile at this position
				tileName, err := g.gameMap.GetTileAt(tileCoord.X, tileCoord.Y)
				if err != nil || tileName == "" {
					continue
				}

				tile, ok := g.gameMap.Atlas.GetTile(tileName)
				if !ok {
					continue
				}

				subImg := g.gameMap.Atlas.GetTileSubImage(tile)

				screenX := float64(tileCoord.X * tileSize)
				screenY := float64(tileCoord.Y * tileSize)

				opts := &renderer.DrawImageOptions{}
				opts.GeoM = renderer.NewGeoM()
				opts.GeoM.Translate(screenX, screenY)
				screen.DrawImage(subImg, opts)

				drawnTiles[tileKey] = true
			}
		}
	}
}

func (g *Game) isPointInShadow(point shadows.Point) bool {
	// With visibility polygon approach: a point is in shadow if it's NOT visible
	// Compute visibility polygon and check if point is inside it
	maxDist := float64(g.screenWidth + g.screenHeight)
	visibilityPolygon := shadows.ComputeVisibilityPolygon(g.player.Pos, g.walls, maxDist)

	// Point is in shadow if it's OUTSIDE the visibility polygon
	return !shadows.PointInPolygon(point, visibilityPolygon)
}

// extendToScreenEdge extends a ray from 'from' through 'to' until it hits a screen edge
func (g *Game) extendToScreenEdge(from, to shadows.Point, maxDist float64) shadows.Point {
	// Direction vector
	dx := to.X - from.X
	dy := to.Y - from.Y

	// Normalize and extend
	length := maxDist * 2.0
	if dx != 0 || dy != 0 {
		currentLen := (dx*dx + dy*dy)
		if currentLen > 0.001 {
			scale := length / currentLen
			dx *= scale
			dy *= scale
		}
	}

	return shadows.Point{
		X: from.X + dx,
		Y: from.Y + dy,
	}
}

func (g *Game) drawPolygon(dst renderer.Image, points []shadows.Point, c color.RGBA) {
	if len(points) < 3 {
		return
	}

	// Convert points to float32 path
	path := vector.Path{}
	path.MoveTo(float32(points[0].X), float32(points[0].Y))
	for i := 1; i < len(points); i++ {
		path.LineTo(float32(points[i].X), float32(points[i].Y))
	}
	path.Close()

	// Fill with anti-aliasing disabled to avoid edge artifacts
	ebitenVertexes, indexes := path.AppendVerticesAndIndicesForFilling(nil, nil)

	if g.whiteImg == nil {
		g.whiteImg = g.renderer.NewImage(1, 1)
		g.whiteImg.Fill(color.White)
	}

	// Convert ebiten vertices to renderer vertices and apply color
	vertexes := make([]renderer.Vertex, len(ebitenVertexes))
	for i := range ebitenVertexes {
		vertexes[i] = renderer.Vertex{
			DstX:   ebitenVertexes[i].DstX,
			DstY:   ebitenVertexes[i].DstY,
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(c.R) / 255,
			ColorG: float32(c.G) / 255,
			ColorB: float32(c.B) / 255,
			ColorA: float32(c.A) / 255,
		}
	}

	opts := &renderer.DrawTrianglesOptions{
		AntiAlias: false,
	}
	dst.DrawTriangles(vertexes, indexes, g.whiteImg, opts)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenWidth, g.screenHeight
}

func main() {
	screenWidth := 800
	screenHeight := 600

	// Initialize the renderer backend (ebiten)
	rend := ebitenrenderer.NewRenderer()
	inputMgr := ebitenrenderer.NewInputManager()
	loader := ebitenrenderer.NewResourceLoader()
	engine := ebitenrenderer.NewEngine()

	// Scan data directory for available games
	log.Println("Scanning data directory for available games...")
	games, err := gamescanner.ScanDataDirectory("data")
	if err != nil {
		log.Fatalf("Failed to scan data directory: %v", err)
	}

	// Create the main menu
	mainMenu := menu.NewMainMenu(games, rend, inputMgr, screenWidth, screenHeight)

	// Create the game manager
	gameManager := &GameManager{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		state:        menu.StateMainMenu,
		mainMenu:     mainMenu,
		renderer:     rend,
		inputMgr:     inputMgr,
		loader:       loader,
	}

	// Set up the window
	engine.SetWindowSize(screenWidth, screenHeight)
	engine.SetWindowTitle("Outpost9 - Main Menu")

	log.Println("Starting game...")
	if err := engine.RunGame(gameManager); err != nil {
		log.Fatal(err)
	}
}
