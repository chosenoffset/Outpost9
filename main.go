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
	A, B Point
}

func castShadow(viewerPos Point, seg Segment, maxDistance float64, tileSize int) []Point {
	dirA := Point{
		X: seg.A.X - viewerPos.X,
		Y: seg.A.Y - viewerPos.Y,
	}
	dirB := Point{
		X: seg.B.X - viewerPos.X,
		Y: seg.B.Y - viewerPos.Y,
	}

	lenA := math.Sqrt(dirA.X*dirA.X + dirA.Y*dirA.Y)
	lenB := math.Sqrt(dirB.X*dirB.X + dirB.Y*dirB.Y)

	if lenA < 0.001 || lenB < 0.001 {
		return nil
	}

	// Offset the shadow start to be behind the wall (so the wall is visible)
	shadowOffset := float64(tileSize)

	// Push the start points away from viewer by the offset
	offsetA := Point{
		X: seg.A.X + (dirA.X/lenA)*shadowOffset,
		Y: seg.A.Y + (dirA.Y/lenA)*shadowOffset,
	}
	offsetB := Point{
		X: seg.B.X + (dirB.X/lenB)*shadowOffset,
		Y: seg.B.Y + (dirB.Y/lenB)*shadowOffset,
	}

	// Extend to max distance from the offset points
	extendedA := Point{
		X: offsetA.X + (dirA.X/lenA)*maxDistance,
		Y: offsetA.Y + (dirA.Y/lenA)*maxDistance,
	}
	extendedB := Point{
		X: offsetB.X + (dirB.X/lenB)*maxDistance,
		Y: offsetB.Y + (dirB.Y/lenB)*maxDistance,
	}

	return []Point{offsetA, offsetB, extendedB, extendedA}
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
	tileScale    float64 // Scale factor for rendering tiles (TileSize / atlas tile size)
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
		// Only cast shadows for walls facing away from player
		if g.isFacingPlayer(wall, g.player.Pos) {
			shadowPoly := castShadow(g.player.Pos, wall, maxDist, g.gameMap.Data.RenderTileSize)
			if shadowPoly != nil {
				// Draw solid black shadow
				g.drawPolygon(shadowMask, shadowPoly, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	// Step 4: Draw the shadow mask on top of the world
	screen.DrawImage(shadowMask, &ebiten.DrawImageOptions{})

	// Step 5: Draw player character on top of everything
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

	renderTileSize := g.gameMap.Data.RenderTileSize

	for y := 0; y < g.gameMap.Data.Height; y++ {
		for x := 0; x < g.gameMap.Data.Width; x++ {
			tileName, err := g.gameMap.GetTileAt(x, y)
			if err != nil {
				continue
			}

			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			subImg := g.gameMap.Atlas.GetTileSubImage(tile)

			screenX := float64(x * renderTileSize)
			screenY := float64(y * renderTileSize)

			opts := &ebiten.DrawImageOptions{}
			// Scale the tile to match the game's render tile size
			opts.GeoM.Scale(g.tileScale, g.tileScale)
			opts.GeoM.Translate(screenX, screenY)

			screen.DrawImage(subImg, opts)
		}
	}
}

func (g *Game) drawPolygon(dst *ebiten.Image, points []Point, c color.RGBA) {
	if len(points) < 3 {
		return
	}

	if g.whiteImg == nil {
		g.whiteImg = ebiten.NewImage(1, 1)
		g.whiteImg.Fill(color.White)
	}

	vertices := make([]ebiten.Vertex, len(points))
	for i, p := range points {
		vertices[i] = ebiten.Vertex{
			DstX:   float32(p.X),
			DstY:   float32(p.Y),
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(c.R) / 255,
			ColorG: float32(c.G) / 255,
			ColorB: float32(c.B) / 255,
			ColorA: float32(c.A) / 255,
		}
	}

	indices := make([]uint16, (len(points)-2)*3)
	for i := 0; i < len(points)-2; i++ {
		indices[i*3] = 0
		indices[i*3+1] = uint16(i + 1)
		indices[i*3+2] = uint16(i + 2)
	}

	dst.DrawTriangles(vertices, indices, g.whiteImg, nil)
}

func (g *Game) isFacingPlayer(seg Segment, playerPos Point) bool {
	// Check if wall segment is facing away from player (should cast shadow)
	dx1 := seg.B.X - seg.A.X
	dy1 := seg.B.Y - seg.A.Y
	dx2 := playerPos.X - seg.A.X
	dy2 := playerPos.Y - seg.A.Y

	cross := dx1*dy2 - dy1*dx2
	return cross < 0
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenWidth, g.screenHeight
}

// Helper function to create wall segments from map data
func createWallSegmentsFromMap(gameMap *maploader.Map) []Segment {
	var segments []Segment

	mapWidth := gameMap.Data.Width
	mapHeight := gameMap.Data.Height
	renderTileSize := float64(gameMap.Data.RenderTileSize)

	// For each tile that blocks sight, create segments for its edges
	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			if gameMap.BlocksSight(x, y) {
				tileX := float64(x) * renderTileSize
				tileY := float64(y) * renderTileSize

				// Check each edge and create segment if it borders non-blocking tile
				// Top edge
				if y == 0 || !gameMap.BlocksSight(x, y-1) {
					segments = append(segments, Segment{
						A: Point{tileX, tileY},
						B: Point{tileX + renderTileSize, tileY},
					})
				}
				// Right edge
				if x == mapWidth-1 || !gameMap.BlocksSight(x+1, y) {
					segments = append(segments, Segment{
						A: Point{tileX + renderTileSize, tileY},
						B: Point{tileX + renderTileSize, tileY + renderTileSize},
					})
				}
				// Bottom edge
				if y == mapHeight-1 || !gameMap.BlocksSight(x, y+1) {
					segments = append(segments, Segment{
						A: Point{tileX + renderTileSize, tileY + renderTileSize},
						B: Point{tileX, tileY + renderTileSize},
					})
				}
				// Left edge
				if x == 0 || !gameMap.BlocksSight(x-1, y) {
					segments = append(segments, Segment{
						A: Point{tileX, tileY + renderTileSize},
						B: Point{tileX, tileY},
					})
				}
			}
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

	log.Printf("Loaded map: %s (%dx%d, tiles: %dpx→%dpx)",
		gameMap.Data.Name,
		gameMap.Data.Width,
		gameMap.Data.Height,
		gameMap.Data.TileSize,
		gameMap.Data.RenderTileSize)

	// Calculate tile scale factor (render tile size / atlas tile size)
	tileScale := float64(gameMap.Data.RenderTileSize) / float64(gameMap.Data.TileSize)

	// Generate wall segments from map data
	walls := createWallSegmentsFromMap(gameMap)

	log.Printf("Generated %d wall segments", len(walls))

	game := &Game{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		gameMap:      gameMap,
		walls:        walls,
		tileScale:    tileScale,
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
