package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"log"
	"math"

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
		screenWidth:  gm.screenWidth,
		screenHeight: gm.screenHeight,
		gameMap:      gameMap,
		walls:        walls,
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
	// WASD movement with collision detection
	moveSpeed := g.player.Speed

	// Player hitbox radius (half of sprite size, slightly smaller for better feel)
	playerRadius := 6.0

	// Calculate intended movement
	var dx, dy float64
	if g.inputMgr.IsKeyPressed(renderer.KeyW) {
		dy -= moveSpeed
	}
	if g.inputMgr.IsKeyPressed(renderer.KeyS) {
		dy += moveSpeed
	}
	if g.inputMgr.IsKeyPressed(renderer.KeyA) {
		dx -= moveSpeed
	}
	if g.inputMgr.IsKeyPressed(renderer.KeyD) {
		dx += moveSpeed
	}

	// Try to move horizontally first (allows wall sliding)
	if dx != 0 {
		newX := g.player.Pos.X + dx
		if g.canMoveTo(newX, g.player.Pos.Y, playerRadius) {
			g.player.Pos.X = newX
		}
	}

	// Then try to move vertically
	if dy != 0 {
		newY := g.player.Pos.Y + dy
		if g.canMoveTo(g.player.Pos.X, newY, playerRadius) {
			g.player.Pos.Y = newY
		}
	}

	// Keep player in level bounds
	tileSize := float64(g.gameMap.Data.TileSize)
	maxX := float64(g.gameMap.Data.Width) * tileSize
	maxY := float64(g.gameMap.Data.Height) * tileSize

	if g.player.Pos.X < playerRadius {
		g.player.Pos.X = playerRadius
	}
	if g.player.Pos.X > maxX-playerRadius {
		g.player.Pos.X = maxX - playerRadius
	}
	if g.player.Pos.Y < playerRadius {
		g.player.Pos.Y = playerRadius
	}
	if g.player.Pos.Y > maxY-playerRadius {
		g.player.Pos.Y = maxY - playerRadius
	}

	return nil
}

// canMoveTo checks if the player can move to the specified position
func (g *Game) canMoveTo(x, y, radius float64) bool {
	if g.gameMap == nil {
		return true
	}

	tileSize := float64(g.gameMap.Data.TileSize)

	// Check multiple points around the player's hitbox
	// This ensures we can't clip through walls at any angle
	checkPoints := []struct{ dx, dy float64 }{
		{0, 0},                         // Center
		{radius, 0},                    // Right
		{-radius, 0},                   // Left
		{0, radius},                    // Bottom
		{0, -radius},                   // Top
		{radius * 0.7, radius * 0.7},   // Bottom-right
		{-radius * 0.7, radius * 0.7},  // Bottom-left
		{radius * 0.7, -radius * 0.7},  // Top-right
		{-radius * 0.7, -radius * 0.7}, // Top-left
	}

	for _, offset := range checkPoints {
		checkX := x + offset.dx
		checkY := y + offset.dy

		// Convert to tile coordinates
		tileX := int(checkX / tileSize)
		tileY := int(checkY / tileSize)

		// Check bounds
		if tileX < 0 || tileX >= g.gameMap.Data.Width ||
			tileY < 0 || tileY >= g.gameMap.Data.Height {
			return false
		}

		// Check if tile is walkable
		if !g.gameMap.IsWalkable(tileX, tileY) {
			return false
		}
	}

	return true
}

func (g *Game) Draw(screen renderer.Image) {
	// MULTI-PASS RENDERING FOR PIXEL-PERFECT SHADOWS

	// Step 1: Draw ONLY floors to the screen
	g.drawFloorsOnly(screen)

	// Step 2: Render walls to wallTexture for shader input
	g.wallTexture.Clear()
	g.drawWallsToTexture(g.wallTexture)

	// Step 3: Apply shadow shader - darkens occluded areas
	if ebitenImg, ok := screen.(*ebitenrenderer.EbitenImage); ok && g.shadowShader != nil {
		ebitenScreen := ebitenImg.GetEbitenImage()

		// DEBUG: Log player position (throttle to ~1/sec to avoid spam)
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			log.Printf("DEBUG PlayerPos: X=%.2f Y=%.2f (screen: %dx%d)",
				g.player.Pos.X, g.player.Pos.Y, g.screenWidth, g.screenHeight)
		}

		opts := &ebiten.DrawRectShaderOptions{}
		opts.Uniforms = map[string]interface{}{
			"PlayerPos":   []float32{float32(g.player.Pos.X), float32(g.player.Pos.Y)},
			"MaxDistance": float32(g.screenWidth + g.screenHeight),
		}
		opts.Images[0] = g.wallTexture
		ebitenScreen.DrawRectShader(g.screenWidth, g.screenHeight, g.shadowShader, opts)
	}

	// Step 4: Draw walls that have clear line of sight ON TOP of shadows
	g.drawVisibleWalls(screen)

	// Step 5: Draw player character on top of everything
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

func (g *Game) drawFloorsOnly(screen renderer.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize

	// Draw floor layer only (fills entire map with floor tile)
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
		return
	}

	tileSize := g.gameMap.Data.TileSize
	drawnTiles := make(map[string]bool) // Track which tiles we've drawn

	// Use wall segments to identify all wall tiles
	for _, wall := range g.walls {
		// Get tiles covered by this segment
		tilesToDraw := wall.TilesCovered
		if len(tilesToDraw) == 0 {
			// Fallback to single tile if TilesCovered is empty
			tilesToDraw = []shadows.Coord{{X: wall.TileX, Y: wall.TileY}}
		}

		for _, tileCoord := range tilesToDraw {
			tileKey := fmt.Sprintf("%d,%d", tileCoord.X, tileCoord.Y)
			if drawnTiles[tileKey] {
				continue // Already drew this tile
			}
			drawnTiles[tileKey] = true

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

			// Extract underlying ebiten.Image to draw wall tiles
			if ebitenImg, ok := subImg.(*ebitenrenderer.EbitenImage); ok {
				ebitenSubImg := ebitenImg.GetEbitenImage()

				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(screenX, screenY)
				texture.DrawImage(ebitenSubImg, opts)
			}
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

			// Special case: Check if this is a corner wall
			// A corner has walls on two perpendicular adjacent sides
			isCorner, adjacentWalls := g.isCornerWall(tileCoord.X, tileCoord.Y)

			var anyVisible bool

			if isCorner {
				// For corner walls: visible if ANY adjacent wall forming the corner is visible
				// This prevents flickering as corners are stable when either wall is visible
				anyVisible = false
				for _, adjCoord := range adjacentWalls {
					adjKey := fmt.Sprintf("%d,%d", adjCoord.X, adjCoord.Y)
					if drawnTiles[adjKey] {
						// Adjacent wall is already drawn, so it's visible
						anyVisible = true
						break
					}
				}

				// If adjacent walls haven't been checked yet, fall back to normal sampling
				if !anyVisible {
					// Check a few sample points on the corner tile itself
					tileBaseX := float64(tileCoord.X * tileSize)
					tileBaseY := float64(tileCoord.Y * tileSize)
					tileSizeFloat := float64(tileSize)

					cornerSamples := []shadows.Point{
						{tileBaseX + 4, tileBaseY + 4},
						{tileBaseX + tileSizeFloat - 4, tileBaseY + 4},
						{tileBaseX + 4, tileBaseY + tileSizeFloat - 4},
						{tileBaseX + tileSizeFloat - 4, tileBaseY + tileSizeFloat - 4},
					}

					for _, point := range cornerSamples {
						if !g.isPointInShadow(point, tileCoord.X, tileCoord.Y) {
							anyVisible = true
							break
						}
					}
				}
			} else {
				// Normal wall: sample multiple points
				tileBaseX := float64(tileCoord.X * tileSize)
				tileBaseY := float64(tileCoord.Y * tileSize)
				tileSizeFloat := float64(tileSize)

				samplePoints := []shadows.Point{
					// Center
					{tileBaseX + tileSizeFloat/2, tileBaseY + tileSizeFloat/2},

					// 4 corners (inset 2px)
					{tileBaseX + 2, tileBaseY + 2},
					{tileBaseX + tileSizeFloat - 2, tileBaseY + 2},
					{tileBaseX + 2, tileBaseY + tileSizeFloat - 2},
					{tileBaseX + tileSizeFloat - 2, tileBaseY + tileSizeFloat - 2},

					// Edge midpoints (inset 2px)
					{tileBaseX + tileSizeFloat/2, tileBaseY + 2},
					{tileBaseX + tileSizeFloat/2, tileBaseY + tileSizeFloat - 2},
					{tileBaseX + 2, tileBaseY + tileSizeFloat/2},
					{tileBaseX + tileSizeFloat - 2, tileBaseY + tileSizeFloat/2},

					// Quarter points
					{tileBaseX + tileSizeFloat/4, tileBaseY + tileSizeFloat/4},
					{tileBaseX + 3*tileSizeFloat/4, tileBaseY + tileSizeFloat/4},
					{tileBaseX + tileSizeFloat/4, tileBaseY + 3*tileSizeFloat/4},
					{tileBaseX + 3*tileSizeFloat/4, tileBaseY + 3*tileSizeFloat/4},
				}

				anyVisible = false
				for _, point := range samplePoints {
					if !g.isPointInShadow(point, tileCoord.X, tileCoord.Y) {
						anyVisible = true
						break
					}
				}
			}

			if !anyVisible {
				continue // Not visible, don't draw
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

func (g *Game) isCornerWall(tileX, tileY int) (bool, []shadows.Coord) {
	// Check if this wall tile is a corner (has walls on 2 perpendicular adjacent sides)
	// Returns true if corner, and the list of adjacent walls forming the corner

	adjacentWalls := []shadows.Coord{}

	// Check 4 cardinal directions
	checkWall := func(x, y int) bool {
		tileName, err := g.gameMap.GetTileAt(x, y)
		if err != nil || tileName == "" {
			return false
		}
		if tile, ok := g.gameMap.Atlas.GetTile(tileName); ok {
			if blocksSight, ok := tile.Properties["blocks_sight"].(bool); ok && blocksSight {
				return true
			}
		}
		return false
	}

	hasNorth := checkWall(tileX, tileY-1)
	hasSouth := checkWall(tileX, tileY+1)
	hasEast := checkWall(tileX+1, tileY)
	hasWest := checkWall(tileX-1, tileY)

	// Check for perpendicular pairs (corners)
	if (hasNorth && hasEast) || (hasNorth && hasWest) || (hasSouth && hasEast) || (hasSouth && hasWest) {
		// This is a corner - collect the adjacent walls
		if hasNorth {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX, Y: tileY - 1})
		}
		if hasSouth {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX, Y: tileY + 1})
		}
		if hasEast {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX + 1, Y: tileY})
		}
		if hasWest {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX - 1, Y: tileY})
		}
		return true, adjacentWalls
	}

	return false, nil
}

func (g *Game) isPointInShadow(point shadows.Point, ignoreTileX, ignoreTileY int) bool {
	// Pixel-perfect raycasting to match shader behavior
	// Cast ray from player to point and check if it hits a wall
	// ignoreTileX/Y: The tile we're checking visibility for (don't count it as an occluder)

	dx := point.X - g.player.Pos.X
	dy := point.Y - g.player.Pos.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance < 1.0 {
		return false // Player position is never in shadow
	}

	// Build ignore set: the target tile plus orthogonally adjacent wall tiles
	// This prevents walls in a line from blocking each other (hallway effect)
	ignoreSet := make(map[string]bool)
	ignoreSet[fmt.Sprintf("%d,%d", ignoreTileX, ignoreTileY)] = true

	// Check 4 orthogonal neighbors and add to ignore set if they're walls
	checkAndIgnore := func(x, y int) {
		tileName, err := g.gameMap.GetTileAt(x, y)
		if err == nil && tileName != "" {
			if tile, ok := g.gameMap.Atlas.GetTile(tileName); ok {
				if blocksSight, ok := tile.Properties["blocks_sight"].(bool); ok && blocksSight {
					ignoreSet[fmt.Sprintf("%d,%d", x, y)] = true
				}
			}
		}
	}

	checkAndIgnore(ignoreTileX-1, ignoreTileY) // West
	checkAndIgnore(ignoreTileX+1, ignoreTileY) // East
	checkAndIgnore(ignoreTileX, ignoreTileY-1) // North
	checkAndIgnore(ignoreTileX, ignoreTileY+1) // South

	// Normalize direction
	dirX := dx / distance
	dirY := dy / distance

	// Sample along the ray (matching shader logic)
	for t := 1.0; t < distance-0.5; t += 1.0 {
		sampleX := g.player.Pos.X + dirX*t
		sampleY := g.player.Pos.Y + dirY*t

		// Check if this sample point is inside any wall tile
		tileX := int(sampleX / float64(g.gameMap.Data.TileSize))
		tileY := int(sampleY / float64(g.gameMap.Data.TileSize))

		tileKey := fmt.Sprintf("%d,%d", tileX, tileY)

		// Skip tiles in our ignore set
		if ignoreSet[tileKey] {
			continue
		}

		// Check if this tile is a wall
		tileName, err := g.gameMap.GetTileAt(tileX, tileY)
		if err == nil && tileName != "" {
			// Check if it's actually a wall (has blocks_sight property)
			if tile, ok := g.gameMap.Atlas.GetTile(tileName); ok {
				if blocksSight, ok := tile.Properties["blocks_sight"].(bool); ok && blocksSight {
					return true // Hit a wall before reaching the point
				}
			}
		}
	}

	return false // Clear line of sight
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
