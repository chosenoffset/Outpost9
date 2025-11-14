package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"

	"chosenoffset.com/outpost9/maploader"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Point struct {
	X, Y float64
}

type Segment struct {
	A, B     Point
	TileX    int // Grid coordinates of the tile this segment belongs to
	TileY    int
	EdgeType string // "top", "bottom", "left", "right"
}

func castShadow(viewerPos Point, seg Segment, maxDistance float64, tileSize int, gameMap *maploader.Map, isCornerShadow bool) []Point {
	// Get the shadow start edge based on player position relative to the tile
	shadowStart := getShadowStartEdge(seg, tileSize, gameMap, viewerPos, isCornerShadow)

	// Calculate direction vectors from viewer to shadow start points
	dirA := Point{
		X: shadowStart.A.X - viewerPos.X,
		Y: shadowStart.A.Y - viewerPos.Y,
	}
	dirB := Point{
		X: shadowStart.B.X - viewerPos.X,
		Y: shadowStart.B.Y - viewerPos.Y,
	}

	lenA := math.Sqrt(dirA.X*dirA.X + dirA.Y*dirA.Y)
	lenB := math.Sqrt(dirB.X*dirB.X + dirB.Y*dirB.Y)

	if lenA < 0.001 || lenB < 0.001 {
		return nil
	}

	// Normalize direction vectors
	dirA.X /= lenA
	dirA.Y /= lenA
	dirB.X /= lenB
	dirB.Y /= lenB

	// Extend shadow rays far
	extendDist := maxDistance * 2

	extendedA := Point{
		X: shadowStart.A.X + dirA.X*extendDist,
		Y: shadowStart.A.Y + dirA.Y*extendDist,
	}
	extendedB := Point{
		X: shadowStart.B.X + dirB.X*extendDist,
		Y: shadowStart.B.Y + dirB.Y*extendDist,
	}

	return []Point{shadowStart.A, shadowStart.B, extendedB, extendedA}
}

func getShadowStartEdge(seg Segment, tileSize int, gameMap *maploader.Map, viewerPos Point, isCornerShadow bool) Segment {
	tileX := float64(seg.TileX) * float64(tileSize)
	tileY := float64(seg.TileY) * float64(tileSize)

	adjusted := seg

	if isCornerShadow {
		// Corner shadow: start from the EXPOSED edge itself (the outside corner edge)
		switch seg.EdgeType {
		case "top":
			// Top edge exposed - angled shadow starts FROM the top edge
			adjusted.A = Point{X: tileX + float64(tileSize), Y: tileY}
			adjusted.B = Point{X: tileX, Y: tileY}
		case "bottom":
			// Bottom edge exposed - angled shadow starts FROM the bottom edge
			adjusted.A = Point{X: tileX, Y: tileY + float64(tileSize)}
			adjusted.B = Point{X: tileX + float64(tileSize), Y: tileY + float64(tileSize)}
		case "left":
			// Left edge exposed - angled shadow starts FROM the left edge
			adjusted.A = Point{X: tileX, Y: tileY}
			adjusted.B = Point{X: tileX, Y: tileY + float64(tileSize)}
		case "right":
			// Right edge exposed - angled shadow starts FROM the right edge
			adjusted.A = Point{X: tileX + float64(tileSize), Y: tileY + float64(tileSize)}
			adjusted.B = Point{X: tileX + float64(tileSize), Y: tileY}
		}
	} else {
		// Main shadow: start from the OPPOSITE edge (far side of tile)
		switch seg.EdgeType {
		case "top":
			// Top edge is exposed - shadow starts from BOTTOM edge
			adjusted.A = Point{X: tileX, Y: tileY + float64(tileSize)}
			adjusted.B = Point{X: tileX + float64(tileSize), Y: tileY + float64(tileSize)}
		case "bottom":
			// Bottom edge is exposed - shadow starts from TOP edge
			adjusted.A = Point{X: tileX + float64(tileSize), Y: tileY}
			adjusted.B = Point{X: tileX, Y: tileY}
		case "left":
			// Left edge is exposed - shadow starts from RIGHT edge
			adjusted.A = Point{X: tileX + float64(tileSize), Y: tileY}
			adjusted.B = Point{X: tileX + float64(tileSize), Y: tileY + float64(tileSize)}
		case "right":
			// Right edge is exposed - shadow starts from LEFT edge
			adjusted.A = Point{X: tileX, Y: tileY + float64(tileSize)}
			adjusted.B = Point{X: tileX, Y: tileY}
		}
	}

	return adjusted
}

func getDefaultShadowOffset(seg Segment, tileSize int) Segment {
	adjusted := seg
	offset := float64(tileSize) / 2.0

	switch seg.EdgeType {
	case "top":
		adjusted.A.Y += offset
		adjusted.B.Y += offset
	case "bottom":
		adjusted.A.Y -= offset
		adjusted.B.Y -= offset
	case "left":
		adjusted.A.X += offset
		adjusted.B.X += offset
	case "right":
		adjusted.A.X -= offset
		adjusted.B.X -= offset
	}

	return adjusted
}

type Player struct {
	Pos   Point
	Speed float64
}

type Game struct {
	screenWidth  int
	screenHeight int
	gameMap      *maploader.Map
	walls        []Segment
	player       Player
	whiteImg     *ebiten.Image
}

func (g *Game) Update() error {
	// WASD movement
	moveSpeed := g.player.Speed

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.player.Pos.Y -= moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.player.Pos.Y += moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.player.Pos.X -= moveSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
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

func (g *Game) Draw(screen *ebiten.Image) {
	// Step 1: Draw all tiles (the world)
	g.drawTiles(screen)

	// Step 2: Create a shadow mask
	shadowMask := ebiten.NewImage(g.screenWidth, g.screenHeight)
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
			shadowPoly := castShadow(g.player.Pos, wall, maxDist, g.gameMap.Data.TileSize, g.gameMap, isCornerShadow)
			if shadowPoly != nil {
				// Draw solid black shadow
				g.drawPolygon(shadowMask, shadowPoly, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	// Step 4: Draw the shadow mask on top of the world
	screen.DrawImage(shadowMask, &ebiten.DrawImageOptions{})

	// Step 5: Redraw wall tiles that face the player (so they're visible above shadows)
	g.drawVisibleWalls(screen)

	// Step 6: Draw player character on top of everything
	vector.FillCircle(screen,
		float32(g.player.Pos.X),
		float32(g.player.Pos.Y),
		8,
		color.RGBA{255, 255, 100, 255},
		false)

	vector.StrokeCircle(screen,
		float32(g.player.Pos.X),
		float32(g.player.Pos.Y),
		8,
		2,
		color.RGBA{200, 200, 50, 255},
		false)
}

func (g *Game) drawTiles(screen *ebiten.Image) {
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

					opts := &ebiten.DrawImageOptions{}
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

			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(screenX, screenY)

			screen.DrawImage(subImg, opts)
		}
	}
}

func (g *Game) drawVisibleWalls(screen *ebiten.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize

	// Draw wall tiles that are visible (not in shadow)
	// Check each wall tile to see if it's obscured by shadows
	drawnTiles := make(map[string]bool) // Track which tiles we've already drawn

	for _, wall := range g.walls {
		if g.isFacingPlayer(wall, g.player.Pos) {
			tileKey := fmt.Sprintf("%d,%d", wall.TileX, wall.TileY)
			if drawnTiles[tileKey] {
				continue // Already drew this tile
			}

			// Check if this tile is in shadow by testing if the tile center is visible
			tileCenterX := float64(wall.TileX)*float64(tileSize) + float64(tileSize)/2
			tileCenterY := float64(wall.TileY)*float64(tileSize) + float64(tileSize)/2

			if g.isPointInShadow(Point{tileCenterX, tileCenterY}) {
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

			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(subImg, opts)

			drawnTiles[tileKey] = true
		}
	}
}

func (g *Game) isPointInShadow(point Point) bool {
	// Check if a point is in shadow by testing against all shadow-casting walls
	maxDist := float64(g.screenWidth + g.screenHeight)

	for _, wall := range g.walls {
		if !g.isFacingPlayer(wall, g.player.Pos) {
			continue
		}

		// Use false for isCornerShadow in point-in-shadow testing
		shadowPoly := castShadow(g.player.Pos, wall, maxDist, g.gameMap.Data.TileSize, g.gameMap, false)
		if shadowPoly != nil && g.pointInPolygon(point, shadowPoly) {
			return true
		}
	}

	return false
}

func (g *Game) pointInPolygon(point Point, polygon []Point) bool {
	// Ray casting algorithm to test if point is inside polygon
	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i].X, polygon[i].Y
		xj, yj := polygon[j].X, polygon[j].Y

		if ((yi > point.Y) != (yj > point.Y)) &&
			(point.X < (xj-xi)*(point.Y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}

	return inside
}

func (g *Game) drawPolygon(dst *ebiten.Image, points []Point, c color.RGBA) {
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
	vertexes, indexes := path.AppendVerticesAndIndicesForFilling(nil, nil)

	if g.whiteImg == nil {
		g.whiteImg = ebiten.NewImage(1, 1)
		g.whiteImg.Fill(color.White)
	}

	// Apply color to vertices
	for i := range vertexes {
		vertexes[i].SrcX = 0
		vertexes[i].SrcY = 0
		vertexes[i].ColorR = float32(c.R) / 255
		vertexes[i].ColorG = float32(c.G) / 255
		vertexes[i].ColorB = float32(c.B) / 255
		vertexes[i].ColorA = float32(c.A) / 255
	}

	opts := &ebiten.DrawTrianglesOptions{}
	opts.AntiAlias = false
	dst.DrawTriangles(vertexes, indexes, g.whiteImg, opts)
}

func calculateShadowOffset(seg Segment, tileSize int, gameMap *maploader.Map) float64 {
	// Get the tile definition to check for visual_bounds
	tileDef, err := gameMap.GetTileDefAt(seg.TileX, seg.TileY)
	if err != nil {
		return 2.0 // Default small offset
	}

	// Try to get visual_bounds from properties
	visualBounds, ok := tileDef.GetTileProperty("visual_bounds")
	if !ok {
		// No visual bounds defined, use small default offset
		return 2.0
	}

	// Parse visual_bounds (should be a map with top, bottom, left, right)
	bounds, ok := visualBounds.(map[string]interface{})
	if !ok {
		return 2.0
	}

	// Based on the edge type, calculate offset to the FAR side of the visual pixels
	// (the side away from the viewer, where the shadow should begin)
	switch seg.EdgeType {
	case "top":
		// For top edge: shadow starts at the BOTTOM of the visual wall
		// visual_bounds.bottom tells us where the wall pixels end
		if bottomVal, ok := bounds["bottom"].(float64); ok {
			// Offset from the top edge to the bottom of the visual wall
			return bottomVal + 1.0
		}
	case "bottom":
		// For bottom edge: shadow starts at the TOP of the visual wall
		// visual_bounds.top tells us where the wall pixels start
		if topVal, ok := bounds["top"].(float64); ok {
			// Calculate distance from bottom edge up to where wall starts
			distanceFromBottom := float64(tileSize) - topVal
			return distanceFromBottom + 1.0
		}
	case "left":
		// For left edge: shadow starts at the RIGHT of the visual wall
		if rightVal, ok := bounds["right"].(float64); ok {
			return rightVal + 1.0
		}
	case "right":
		// For right edge: shadow starts at the LEFT of the visual wall
		if leftVal, ok := bounds["left"].(float64); ok {
			distanceFromRight := float64(tileSize) - leftVal
			return distanceFromRight + 1.0
		}
	}

	// Default: small offset
	return 2.0
}

func (g *Game) isFacingPlayer(seg Segment, playerPos Point) bool {
	// Check if player is on the "front" side of the wall segment
	// The front side is determined by the cross product
	// We want to cast shadows only from walls the player can see
	dx1 := seg.B.X - seg.A.X
	dy1 := seg.B.Y - seg.A.Y
	dx2 := playerPos.X - seg.A.X
	dy2 := playerPos.Y - seg.A.Y

	cross := dx1*dy2 - dy1*dx2
	// Return true if player is on the positive side (wall is facing player)
	return cross > 0
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenWidth, g.screenHeight
}

// Helper function to create wall segments from map data
func createWallSegmentsFromMap(gameMap *maploader.Map) []Segment {
	var segments []Segment

	mapWidth := gameMap.Data.Width
	mapHeight := gameMap.Data.Height
	tileSize := float64(gameMap.Data.TileSize)

	// First pass: create individual tile edge segments
	type tempSegment struct {
		seg      Segment
		canMerge bool
	}
	var tempSegments []tempSegment

	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			if gameMap.BlocksSight(x, y) {
				tileX := float64(x) * tileSize
				tileY := float64(y) * tileSize

				// Get the tile name to check its type
				tileName, _ := gameMap.GetTileAt(x, y)

				// Check if there's a wall tile above this one
				hasWallAbove := y > 0 && gameMap.BlocksSight(x, y-1)

				// Check each edge and create segment if it borders non-blocking tile
				// Top edge - skip if this is a "bottom" tile with a wall above it
				shouldCreateTopEdge := (y == 0 || !gameMap.BlocksSight(x, y-1))

				// Skip top edge for bottom tiles (nwb, nbx, neb, swb, sbx, seb, etc) that have walls above
				if shouldCreateTopEdge && !(hasWallAbove && (tileName == "nwb" || tileName == "nbx" || tileName == "neb" ||
					tileName == "swb" || tileName == "sbx" || tileName == "seb")) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX, tileY},
							B:        Point{tileX + tileSize, tileY},
							TileX:    x,
							TileY:    y,
							EdgeType: "top",
						},
						canMerge: true,
					})
				}
				// Right edge
				if x == mapWidth-1 || !gameMap.BlocksSight(x+1, y) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX + tileSize, tileY},
							B:        Point{tileX + tileSize, tileY + tileSize},
							TileX:    x,
							TileY:    y,
							EdgeType: "right",
						},
						canMerge: true,
					})
				}
				// Bottom edge
				if y == mapHeight-1 || !gameMap.BlocksSight(x, y+1) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX + tileSize, tileY + tileSize},
							B:        Point{tileX, tileY + tileSize},
							TileX:    x,
							TileY:    y,
							EdgeType: "bottom",
						},
						canMerge: true,
					})
				}
				// Left edge
				if x == 0 || !gameMap.BlocksSight(x-1, y) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX, tileY + tileSize},
							B:        Point{tileX, tileY},
							TileX:    x,
							TileY:    y,
							EdgeType: "left",
						},
						canMerge: true,
					})
				}
			}
		}
	}

	// Second pass: merge adjacent colinear segments
	merged := make([]bool, len(tempSegments))

	for i := 0; i < len(tempSegments); i++ {
		if merged[i] || !tempSegments[i].canMerge {
			continue
		}

		current := tempSegments[i].seg

		// Try to find adjacent segments to merge with
		for j := i + 1; j < len(tempSegments); j++ {
			if merged[j] || !tempSegments[j].canMerge {
				continue
			}

			other := tempSegments[j].seg

			// Check if segments are adjacent and colinear
			// DON'T merge - keep segments separate for proper shadow alignment
			// Each tile needs its own shadow based on its visual bounds
			_ = other
		}

		segments = append(segments, current)
		merged[i] = true
	}

	// Add any unmerged segments
	for i, temp := range tempSegments {
		if !merged[i] {
			segments = append(segments, temp.seg)
		}
	}

	return segments
}

func main() {
	// Command-line flags
	gameDir := flag.String("game", "Example", "Game data directory (e.g., Example, Outpost9)")
	levelFile := flag.String("level", "level1.json", "Level file to load")
	flag.Parse()

	screenWidth := 800
	screenHeight := 600

	// Construct the level path
	levelPath := fmt.Sprintf("data/%s/%s", *gameDir, *levelFile)

	// Load the map from JSON
	log.Printf("Loading level: %s", levelPath)
	gameMap, err := maploader.LoadMap(levelPath)
	if err != nil {
		log.Fatalf("Failed to load map: %v", err)
	}

	log.Printf("Loaded map: %s (%dx%d, tile size: %dpx)",
		gameMap.Data.Name,
		gameMap.Data.Width,
		gameMap.Data.Height,
		gameMap.Data.TileSize)

	// Generate wall segments from map data
	walls := createWallSegmentsFromMap(gameMap)

	log.Printf("Generated %d wall segments", len(walls))

	game := &Game{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		gameMap:      gameMap,
		walls:        walls,
		player: Player{
			Pos:   Point{gameMap.Data.PlayerSpawn.X, gameMap.Data.PlayerSpawn.Y},
			Speed: 3.0,
		},
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	windowTitle := fmt.Sprintf("Outpost9 [%s] - WASD to move", *gameDir)
	ebiten.SetWindowTitle(windowTitle)

	log.Printf("Starting game...")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
