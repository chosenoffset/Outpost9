package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Point struct {
	X, Y float64
}

type Segment struct {
	A, B Point
}

const (
	TileSize  = 64
	TileFloor = 0
	TileWall  = 1
)

type Tile struct {
	Type int
}

func castShadow(viewerPos Point, seg Segment, maxDistance float64) []Point {
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

	extendedA := Point{
		X: seg.A.X + (dirA.X/lenA)*maxDistance,
		Y: seg.A.Y + (dirA.Y/lenA)*maxDistance,
	}
	extendedB := Point{
		X: seg.B.X + (dirB.X/lenB)*maxDistance,
		Y: seg.B.Y + (dirB.Y/lenB)*maxDistance,
	}

	return []Point{seg.A, seg.B, extendedB, extendedA}
}

type Player struct {
	Pos   Point
	Speed float64
}

type Game struct {
	screenWidth  int
	screenHeight int
	mapWidth     int
	mapHeight    int
	tiles        [][]Tile
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
		// Only cast shadows for walls facing away from player
		if g.isFacingPlayer(wall, g.player.Pos) {
			shadowPoly := castShadow(g.player.Pos, wall, maxDist)
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
	for y := 0; y < g.mapHeight; y++ {
		for x := 0; x < g.mapWidth; x++ {
			tile := g.tiles[y][x]

			screenX := float32(x * TileSize)
			screenY := float32(y * TileSize)

			var tileColor color.RGBA

			switch tile.Type {
			case TileFloor:
				tileColor = color.RGBA{80, 80, 80, 255} // Gray floor
			case TileWall:
				tileColor = color.RGBA{60, 80, 120, 255} // Blue wall
			}

			// Draw filled tile
			vector.FillRect(screen,
				screenX, screenY,
				TileSize, TileSize,
				tileColor, false)

			// Draw tile border for visual clarity
			vector.StrokeRect(screen,
				screenX, screenY,
				TileSize, TileSize,
				1,
				color.RGBA{40, 40, 40, 255}, false)
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

// Helper function to create wall segments from tile positions
func createWallSegmentsFromTiles(tiles [][]Tile, mapWidth, mapHeight int) []Segment {
	var segments []Segment

	// For each wall tile, create segments for its edges
	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			if tiles[y][x].Type == TileWall {
				tileX := float64(x * TileSize)
				tileY := float64(y * TileSize)

				// Check each edge and create segment if it borders non-wall
				// Top edge
				if y == 0 || tiles[y-1][x].Type != TileWall {
					segments = append(segments, Segment{
						A: Point{tileX, tileY},
						B: Point{tileX + TileSize, tileY},
					})
				}
				// Right edge
				if x == mapWidth-1 || tiles[y][x+1].Type != TileWall {
					segments = append(segments, Segment{
						A: Point{tileX + TileSize, tileY},
						B: Point{tileX + TileSize, tileY + TileSize},
					})
				}
				// Bottom edge
				if y == mapHeight-1 || tiles[y+1][x].Type != TileWall {
					segments = append(segments, Segment{
						A: Point{tileX + TileSize, tileY + TileSize},
						B: Point{tileX, tileY + TileSize},
					})
				}
				// Left edge
				if x == 0 || tiles[y][x-1].Type != TileWall {
					segments = append(segments, Segment{
						A: Point{tileX, tileY + TileSize},
						B: Point{tileX, tileY},
					})
				}
			}
		}
	}

	return segments
}

func main() {
	screenWidth := 800
	screenHeight := 600
	mapWidth := screenWidth / TileSize
	mapHeight := screenHeight / TileSize

	// Create a simple tile map
	tiles := make([][]Tile, mapHeight)
	for y := 0; y < mapHeight; y++ {
		tiles[y] = make([]Tile, mapWidth)
		for x := 0; x < mapWidth; x++ {
			tiles[y][x] = Tile{Type: TileFloor}
		}
	}

	// Add some wall tiles to create a room with obstacles
	// Outer walls
	for x := 0; x < mapWidth; x++ {
		tiles[0][x].Type = TileWall
		tiles[mapHeight-1][x].Type = TileWall
	}
	for y := 0; y < mapHeight; y++ {
		tiles[y][0].Type = TileWall
		tiles[y][mapWidth-1].Type = TileWall
	}

	// Central obstacle
	for y := 3; y <= 5; y++ {
		for x := 5; x <= 8; x++ {
			tiles[y][x].Type = TileWall
		}
	}

	// Additional obstacles
	tiles[2][2].Type = TileWall
	tiles[2][3].Type = TileWall
	tiles[7][9].Type = TileWall
	tiles[8][9].Type = TileWall

	// Generate wall segments from tiles
	walls := createWallSegmentsFromTiles(tiles, mapWidth, mapHeight)

	game := &Game{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		mapWidth:     mapWidth,
		mapHeight:    mapHeight,
		tiles:        tiles,
		walls:        walls,
		player: Player{
			Pos:   Point{200, 300},
			Speed: 3.0,
		},
	}

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Nox-style Tile-Based Shadows - WASD to move")
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
