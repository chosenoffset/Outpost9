package maploader

import (
	"encoding/json"
	"fmt"
	"os"

	"chosenoffset.com/outpost9/atlas"
)

// SpawnPoint defines a player or entity spawn location
type SpawnPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// MapData represents the loaded map configuration
type MapData struct {
	Name           string     `json:"name"`
	Width          int        `json:"width"`
	Height         int        `json:"height"`
	TileSize       int        `json:"tile_size"`        // Atlas sprite tile size (e.g., 16)
	RenderTileSize int        `json:"render_tile_size"` // Rendered tile size in game (e.g., 64)
	AtlasPath      string     `json:"atlas"`
	PlayerSpawn    SpawnPoint `json:"player_spawn"`
	Tiles          [][]string `json:"tiles"` // 2D array of tile names [y][x]
}

// Map represents a loaded map with its atlas
type Map struct {
	Data  *MapData
	Atlas *atlas.Atlas
}

// LoadMap loads a map from a JSON file and its associated atlas
func LoadMap(mapPath string) (*Map, error) {
	// Read the map JSON file
	data, err := os.ReadFile(mapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read map file %s: %w", mapPath, err)
	}

	// Parse the JSON
	var mapData MapData
	if err := json.Unmarshal(data, &mapData); err != nil {
		return nil, fmt.Errorf("failed to parse map file %s: %w", mapPath, err)
	}

	// Validate map data
	if err := validateMapData(&mapData); err != nil {
		return nil, fmt.Errorf("invalid map data in %s: %w", mapPath, err)
	}

	// Load the atlas
	atlasObj, err := atlas.LoadAtlas(mapData.AtlasPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load atlas %s: %w", mapData.AtlasPath, err)
	}

	gameMap := &Map{
		Data:  &mapData,
		Atlas: atlasObj,
	}

	return gameMap, nil
}

// validateMapData checks if the map data is valid
func validateMapData(data *MapData) error {
	if data.Width <= 0 || data.Height <= 0 {
		return fmt.Errorf("invalid map dimensions: %dx%d", data.Width, data.Height)
	}

	if data.TileSize <= 0 {
		return fmt.Errorf("invalid tile size: %d", data.TileSize)
	}

	if data.RenderTileSize <= 0 {
		return fmt.Errorf("invalid render tile size: %d", data.RenderTileSize)
	}

	if data.AtlasPath == "" {
		return fmt.Errorf("atlas path is required")
	}

	// Validate tiles array dimensions
	if len(data.Tiles) != data.Height {
		return fmt.Errorf("tiles array height mismatch: expected %d, got %d", data.Height, len(data.Tiles))
	}

	for y, row := range data.Tiles {
		if len(row) != data.Width {
			return fmt.Errorf("tiles array width mismatch at row %d: expected %d, got %d", y, data.Width, len(row))
		}
	}

	return nil
}

// GetTileAt returns the tile name at the given grid coordinates
func (m *Map) GetTileAt(x, y int) (string, error) {
	if x < 0 || x >= m.Data.Width || y < 0 || y >= m.Data.Height {
		return "", fmt.Errorf("coordinates out of bounds: (%d, %d)", x, y)
	}
	return m.Data.Tiles[y][x], nil
}

// GetTileDefAt returns the tile definition at the given grid coordinates
func (m *Map) GetTileDefAt(x, y int) (*atlas.TileDefinition, error) {
	tileName, err := m.GetTileAt(x, y)
	if err != nil {
		return nil, err
	}

	tile, ok := m.Atlas.GetTile(tileName)
	if !ok {
		return nil, fmt.Errorf("tile not found in atlas: %s", tileName)
	}

	return tile, nil
}

// IsWalkable returns whether the tile at the given coordinates is walkable
func (m *Map) IsWalkable(x, y int) bool {
	tile, err := m.GetTileDefAt(x, y)
	if err != nil {
		return false
	}
	return tile.GetTilePropertyBool("walkable", true)
}

// BlocksSight returns whether the tile at the given coordinates blocks line of sight
func (m *Map) BlocksSight(x, y int) bool {
	tile, err := m.GetTileDefAt(x, y)
	if err != nil {
		return false
	}
	return tile.GetTilePropertyBool("blocks_sight", false)
}

// GetTileType returns the type of tile at the given coordinates
func (m *Map) GetTileType(x, y int) string {
	tile, err := m.GetTileDefAt(x, y)
	if err != nil {
		return "unknown"
	}
	return tile.GetTilePropertyString("type", "unknown")
}
