package atlas

// This file contains example code showing how to integrate the atlas system
// into your main game. This is for reference only and not compiled.

/*
Example integration into main.go:

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yourusername/Outpost9/atlas"
	"log"
)

type Game struct {
	// Existing fields...
	playerX, playerY float64
	mapWidth, mapHeight int
	tiles [][]Tile

	// Add atlas manager
	atlasManager *atlas.Manager

	// Optional: cache tile names for each map position
	tileNames [][]string // [y][x] = tile name for rendering
}

func NewGame() *Game {
	g := &Game{
		mapWidth:  13,
		mapHeight: 10,
		playerX:   400,
		playerY:   300,
	}

	// Initialize atlas manager
	g.atlasManager = atlas.NewManager()

	// Load atlas configurations
	if err := g.atlasManager.LoadAtlasConfig("data/atlases/base_layer.json"); err != nil {
		log.Printf("Warning: Failed to load base layer atlas: %v", err)
		// Fall back to vector rendering if atlas fails to load
	}

	if err := g.atlasManager.LoadAtlasConfig("data/atlases/objects_layer.json"); err != nil {
		log.Printf("Warning: Failed to load objects layer atlas: %v", err)
	}

	// Initialize the map
	g.initializeMap()

	return g
}

func (g *Game) initializeMap() {
	g.tiles = make([][]Tile, g.mapHeight)
	g.tileNames = make([][]string, g.mapHeight)

	for y := 0; y < g.mapHeight; y++ {
		g.tiles[y] = make([]Tile, g.mapWidth)
		g.tileNames[y] = make([]string, g.mapWidth)

		for x := 0; x < g.mapWidth; x++ {
			// Determine tile type (floor or wall)
			isWall := false

			// Perimeter walls
			if x == 0 || x == g.mapWidth-1 || y == 0 || y == g.mapHeight-1 {
				isWall = true
			}

			// Interior obstacles (example: 4x4 block in the center)
			if y >= 3 && y <= 6 && x >= 5 && x <= 8 {
				isWall = true
			}

			if isWall {
				g.tiles[y][x].Type = TileWall
				// Assign appropriate wall tile name based on neighbors
				// For simplicity, using a generic wall tile
				g.tileNames[y][x] = "wall_center"
			} else {
				g.tiles[y][x].Type = TileFloor
				g.tileNames[y][x] = "floor_metal_01"
			}
		}
	}

	// Optional: improve wall tile selection based on neighbors
	g.updateWallTiles()
}

func (g *Game) updateWallTiles() {
	// This function assigns the correct wall tile based on neighbors
	// (corners, edges, etc.)
	for y := 0; y < g.mapHeight; y++ {
		for x := 0; x < g.mapWidth; x++ {
			if g.tiles[y][x].Type != TileWall {
				continue
			}

			// Check neighbors
			hasNorth := y > 0 && g.tiles[y-1][x].Type == TileWall
			hasSouth := y < g.mapHeight-1 && g.tiles[y+1][x].Type == TileWall
			hasWest := x > 0 && g.tiles[y][x-1].Type == TileWall
			hasEast := x < g.mapWidth-1 && g.tiles[y][x+1].Type == TileWall

			// Determine tile name based on neighbors
			if !hasNorth && !hasWest {
				g.tileNames[y][x] = "wall_nw_corner"
			} else if !hasNorth && !hasEast {
				g.tileNames[y][x] = "wall_ne_corner"
			} else if !hasSouth && !hasWest {
				g.tileNames[y][x] = "wall_sw_corner"
			} else if !hasSouth && !hasEast {
				g.tileNames[y][x] = "wall_se_corner"
			} else if !hasNorth {
				g.tileNames[y][x] = "wall_north"
			} else if !hasSouth {
				g.tileNames[y][x] = "wall_south"
			} else if !hasWest {
				g.tileNames[y][x] = "wall_west"
			} else if !hasEast {
				g.tileNames[y][x] = "wall_east"
			} else {
				g.tileNames[y][x] = "wall_center"
			}
		}
	}
}

func (g *Game) drawTilesWithAtlas(screen *ebiten.Image) {
	const TileSize = 64

	// Try to get the base layer atlas
	baseAtlas, hasAtlas := g.atlasManager.GetAtlasByLayer("base")

	for y := 0; y < g.mapHeight; y++ {
		for x := 0; x < g.mapWidth; x++ {
			screenX := float64(x * TileSize)
			screenY := float64(y * TileSize)

			// Get the tile name for this position
			tileName := g.tileNames[y][x]

			if hasAtlas {
				// Draw using atlas
				if err := baseAtlas.DrawTile(screen, tileName, screenX, screenY); err != nil {
					// Fallback to colored rectangle if tile not found
					g.drawFallbackTile(screen, x, y)
				}
			} else {
				// No atlas loaded, use fallback rendering
				g.drawFallbackTile(screen, x, y)
			}
		}
	}
}

func (g *Game) drawFallbackTile(screen *ebiten.Image, x, y int) {
	// This is the original vector-based rendering as a fallback
	const TileSize = 64
	screenX := float64(x * TileSize)
	screenY := float64(y * TileSize)

	tile := g.tiles[y][x]

	var tileColor color.RGBA
	switch tile.Type {
	case TileFloor:
		tileColor = color.RGBA{80, 80, 80, 255}
	case TileWall:
		tileColor = color.RGBA{60, 80, 120, 255}
	}

	vector.FillRect(screen, float32(screenX), float32(screenY), TileSize, TileSize, tileColor, false)
	borderColor := color.RGBA{40, 40, 40, 255}
	vector.StrokeRect(screen, float32(screenX), float32(screenY), TileSize, TileSize, 1, borderColor, false)
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw tiles using atlas (with fallback)
	g.drawTilesWithAtlas(screen)

	// Draw shadows (existing code)
	g.drawShadows(screen)

	// Draw player (existing code)
	g.drawPlayer(screen)
}

// Optional: Add objects to the map
type GameObject struct {
	X, Y     float64
	TileName string
	Layer    string
}

func (g *Game) drawObjects(screen *ebiten.Image) {
	// Example: draw objects from an objects list
	for _, obj := range g.objects {
		if err := g.atlasManager.DrawTile(screen, obj.Layer, obj.TileName, obj.X, obj.Y); err != nil {
			log.Printf("Failed to draw object: %v", err)
		}
	}
}
*/
