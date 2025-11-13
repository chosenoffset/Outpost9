package atlas

import (
	"encoding/json"
	"fmt"
	"image"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// TileDefinition defines a single tile within an atlas
type TileDefinition struct {
	Name        string                 `json:"name"`        // Semantic name (e.g., "nw_wall_corner")
	AtlasX      int                    `json:"atlas_x"`     // X position in atlas (in tiles)
	AtlasY      int                    `json:"atlas_y"`     // Y position in atlas (in tiles)
	Properties  map[string]interface{} `json:"properties"`  // Custom properties (collision, type, etc.)
}

// AtlasConfig defines the JSON configuration for a sprite atlas
type AtlasConfig struct {
	Name       string            `json:"name"`        // Atlas name
	Layer      string            `json:"layer"`       // Layer this atlas belongs to (e.g., "base", "objects")
	ImagePath  string            `json:"image_path"`  // Path to the atlas image file
	TileWidth  int               `json:"tile_width"`  // Width of each tile in pixels
	TileHeight int               `json:"tile_height"` // Height of each tile in pixels
	Tiles      []TileDefinition  `json:"tiles"`       // Array of tile definitions
}

// Atlas represents a loaded sprite atlas
type Atlas struct {
	Config      *AtlasConfig
	Image       *ebiten.Image
	TilesByName map[string]*TileDefinition // Quick lookup by name
}

// LoadAtlas loads a sprite atlas from a JSON configuration file
func LoadAtlas(configPath string) (*Atlas, error) {
	// Read the JSON configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read atlas config %s: %w", configPath, err)
	}

	// Parse the JSON
	var config AtlasConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse atlas config %s: %w", configPath, err)
	}

	// Validate configuration
	if config.TileWidth <= 0 || config.TileHeight <= 0 {
		return nil, fmt.Errorf("invalid tile dimensions: %dx%d", config.TileWidth, config.TileHeight)
	}

	if config.ImagePath == "" {
		return nil, fmt.Errorf("image_path is required in atlas config")
	}

	// Load the atlas image
	img, _, err := ebitenutil.NewImageFromFile(config.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load atlas image %s: %w", config.ImagePath, err)
	}

	// Build the name lookup map
	tilesByName := make(map[string]*TileDefinition)
	for i := range config.Tiles {
		tile := &config.Tiles[i]
		if tile.Name != "" {
			tilesByName[tile.Name] = tile
		}
	}

	atlas := &Atlas{
		Config:      &config,
		Image:       img,
		TilesByName: tilesByName,
	}

	return atlas, nil
}

// GetTile returns a tile definition by name
func (a *Atlas) GetTile(name string) (*TileDefinition, bool) {
	tile, ok := a.TilesByName[name]
	return tile, ok
}

// GetTileSubImage returns the sub-image for a specific tile
func (a *Atlas) GetTileSubImage(tile *TileDefinition) *ebiten.Image {
	x := tile.AtlasX * a.Config.TileWidth
	y := tile.AtlasY * a.Config.TileHeight
	w := a.Config.TileWidth
	h := a.Config.TileHeight

	rect := image.Rect(x, y, x+w, y+h)
	return a.Image.SubImage(rect).(*ebiten.Image)
}

// GetTileSubImageByName returns the sub-image for a tile by name
func (a *Atlas) GetTileSubImageByName(name string) (*ebiten.Image, error) {
	tile, ok := a.GetTile(name)
	if !ok {
		return nil, fmt.Errorf("tile not found: %s", name)
	}
	return a.GetTileSubImage(tile), nil
}

// DrawTile draws a specific tile at the given screen coordinates
func (a *Atlas) DrawTile(screen *ebiten.Image, tileName string, x, y float64) error {
	tile, ok := a.GetTile(tileName)
	if !ok {
		return fmt.Errorf("tile not found: %s", tileName)
	}

	subImg := a.GetTileSubImage(tile)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(x, y)
	screen.DrawImage(subImg, opts)

	return nil
}

// DrawTileDef draws a tile definition at the given screen coordinates
func (a *Atlas) DrawTileDef(screen *ebiten.Image, tile *TileDefinition, x, y float64) {
	subImg := a.GetTileSubImage(tile)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(x, y)
	screen.DrawImage(subImg, opts)
}

// DrawTileWithOptions draws a tile with custom DrawImageOptions
func (a *Atlas) DrawTileWithOptions(screen *ebiten.Image, tileName string, opts *ebiten.DrawImageOptions) error {
	tile, ok := a.GetTile(tileName)
	if !ok {
		return fmt.Errorf("tile not found: %s", tileName)
	}

	subImg := a.GetTileSubImage(tile)
	screen.DrawImage(subImg, opts)

	return nil
}

// GetTileProperty retrieves a property from a tile definition
func (td *TileDefinition) GetTileProperty(key string) (interface{}, bool) {
	if td.Properties == nil {
		return nil, false
	}
	val, ok := td.Properties[key]
	return val, ok
}

// GetTilePropertyBool retrieves a boolean property
func (td *TileDefinition) GetTilePropertyBool(key string, defaultVal bool) bool {
	val, ok := td.GetTileProperty(key)
	if !ok {
		return defaultVal
	}
	if boolVal, ok := val.(bool); ok {
		return boolVal
	}
	return defaultVal
}

// GetTilePropertyString retrieves a string property
func (td *TileDefinition) GetTilePropertyString(key string, defaultVal string) string {
	val, ok := td.GetTileProperty(key)
	if !ok {
		return defaultVal
	}
	if strVal, ok := val.(string); ok {
		return strVal
	}
	return defaultVal
}

// GetTilePropertyInt retrieves an integer property
func (td *TileDefinition) GetTilePropertyInt(key string, defaultVal int) int {
	val, ok := td.GetTileProperty(key)
	if !ok {
		return defaultVal
	}
	// JSON numbers are float64
	if floatVal, ok := val.(float64); ok {
		return int(floatVal)
	}
	return defaultVal
}
