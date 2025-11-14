package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"

	"chosenoffset.com/outpost9/maploader"
	"chosenoffset.com/outpost9/renderer"
	ebitenrenderer "chosenoffset.com/outpost9/renderer/ebiten"
	"chosenoffset.com/outpost9/shadows"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Player struct {
	Pos   shadows.Point
	Speed float64
}

type Game struct {
	screenWidth  int
	screenHeight int
	gameMap      *maploader.Map
	walls        []shadows.Segment
	player       Player
	whiteImg     renderer.Image
	renderer     renderer.Renderer
	inputMgr     renderer.InputManager
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
	// Step 1: Draw all tiles (the world)
	g.drawTiles(screen)

	// Step 2: Create a shadow mask
	shadowMask := g.renderer.NewImage(g.screenWidth, g.screenHeight)
	// Start transparent (no shadows)
	shadowMask.Fill(color.RGBA{0, 0, 0, 0})

	// Step 3: Draw shadow volumes onto the mask
	maxDist := float64(g.screenWidth + g.screenHeight)

	for _, wall := range g.walls {
		// Determine if this segment should cast a shadow based on player position
		tileSize := float64(g.gameMap.Data.TileSize)
		tileCenterX := float64(wall.TileX)*tileSize + tileSize/2.0
		tileCenterY := float64(wall.TileY)*tileSize + tileSize/2.0

		// Determine player direction relative to tile
		playerAbove := g.player.Pos.Y < tileCenterY
		playerBelow := g.player.Pos.Y > tileCenterY
		playerLeft := g.player.Pos.X < tileCenterX
		playerRight := g.player.Pos.X > tileCenterX

		// Determine if this is a main wall shadow or a corner shadow
		isMainShadow := false
		isCornerShadow := false

		switch wall.EdgeType {
		case "top":
			// Top edge exposed
			if playerAbove {
				isMainShadow = true // Straight shadow going down
			} else if playerLeft || playerRight {
				isCornerShadow = true // Angled corner shadow
			}
		case "bottom":
			// Bottom edge exposed
			if playerBelow {
				isMainShadow = true // Straight shadow going up
			} else if playerLeft || playerRight {
				isCornerShadow = true // Angled corner shadow
			}
		case "left":
			// Left edge exposed
			if playerLeft {
				isMainShadow = true // Straight shadow going right
			} else if playerAbove || playerBelow {
				isCornerShadow = true // Angled corner shadow
			}
		case "right":
			// Right edge exposed
			if playerRight {
				isMainShadow = true // Straight shadow going left
			} else if playerAbove || playerBelow {
				isCornerShadow = true // Angled corner shadow
			}
		}

		if isMainShadow || isCornerShadow {
			shadowPoly := shadows.CastShadow(g.player.Pos, wall, maxDist, g.gameMap.Data.TileSize, g.gameMap, isCornerShadow)
			if shadowPoly != nil {
				// Draw solid black shadow
				g.drawPolygon(shadowMask, shadowPoly, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	// Step 4: Draw the shadow mask on top of the world
	screen.DrawImage(shadowMask, nil)

	// Step 5: Redraw wall tiles that face the player (so they're visible above shadows)
	g.drawVisibleWalls(screen)

	// Step 6: Draw player character on top of everything
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
			tileKey := fmt.Sprintf("%d,%d", wall.TileX, wall.TileY)
			if drawnTiles[tileKey] {
				continue // Already drew this tile
			}

			// Check if this tile is in shadow by testing if the tile center is visible
			tileCenterX := float64(wall.TileX)*float64(tileSize) + float64(tileSize)/2
			tileCenterY := float64(wall.TileY)*float64(tileSize) + float64(tileSize)/2

			if g.isPointInShadow(shadows.Point{tileCenterX, tileCenterY}) {
				continue // This wall is in shadow, don't redraw it
			}

			// Get the wall tile at this segment's position
			tileName, err := g.gameMap.GetTileAt(wall.TileX, wall.TileY)
			if err != nil || tileName == "" {
				continue
			}

			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			subImg := g.gameMap.Atlas.GetTileSubImage(tile)

			screenX := float64(wall.TileX * tileSize)
			screenY := float64(wall.TileY * tileSize)

			opts := &renderer.DrawImageOptions{}
			opts.GeoM = renderer.NewGeoM()
			opts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(subImg, opts)

			drawnTiles[tileKey] = true
		}
	}
}

func (g *Game) isPointInShadow(point shadows.Point) bool {
	// Check if a point is in shadow by testing against all shadow-casting walls
	maxDist := float64(g.screenWidth + g.screenHeight)

	for _, wall := range g.walls {
		if !shadows.IsFacingPoint(wall, g.player.Pos) {
			continue
		}

		// Use false for isCornerShadow in point-in-shadow testing
		shadowPoly := shadows.CastShadow(g.player.Pos, wall, maxDist, g.gameMap.Data.TileSize, g.gameMap, false)
		if shadowPoly != nil && shadows.PointInPolygon(point, shadowPoly) {
			return true
		}
	}

	return false
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
	// Command-line flags
	gameDir := flag.String("game", "Example", "Game data directory (e.g., Example, Outpost9)")
	levelFile := flag.String("level", "level1.json", "Level file to load")
	flag.Parse()

	screenWidth := 800
	screenHeight := 600

	// Initialize the renderer backend (ebiten)
	rend := ebitenrenderer.NewRenderer()
	inputMgr := ebitenrenderer.NewInputManager()
	loader := ebitenrenderer.NewResourceLoader()
	engine := ebitenrenderer.NewEngine()

	// Construct the level path
	levelPath := fmt.Sprintf("data/%s/%s", *gameDir, *levelFile)

	// Load the map from JSON
	log.Printf("Loading level: %s", levelPath)
	gameMap, err := maploader.LoadMap(levelPath, loader)
	if err != nil {
		log.Fatalf("Failed to load map: %v", err)
	}

	log.Printf("Loaded map: %s (%dx%d, tile size: %dpx)",
		gameMap.Data.Name,
		gameMap.Data.Width,
		gameMap.Data.Height,
		gameMap.Data.TileSize)

	// Generate wall segments from map data
	walls := shadows.CreateWallSegmentsFromMap(gameMap)

	log.Printf("Generated %d wall segments", len(walls))

	game := &Game{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		gameMap:      gameMap,
		walls:        walls,
		player: Player{
			Pos:   shadows.Point{X: gameMap.Data.PlayerSpawn.X, Y: gameMap.Data.PlayerSpawn.Y},
			Speed: 3.0,
		},
		renderer: rend,
		inputMgr: inputMgr,
	}

	engine.SetWindowSize(screenWidth, screenHeight)
	windowTitle := fmt.Sprintf("Outpost9 [%s] - WASD to move", *gameDir)
	engine.SetWindowTitle(windowTitle)

	log.Printf("Starting game...")
	if err := engine.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
