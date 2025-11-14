package atlas

import (
	"encoding/json"
	"os"
	"testing"
)

func TestAtlasConfigParsing(t *testing.T) {
	// Test JSON structure
	jsonData := `{
		"name": "test_atlas",
		"layer": "base",
		"image_path": "test.png",
		"tile_width": 64,
		"tile_height": 64,
		"tiles": [
			{
				"name": "floor_tile",
				"atlas_x": 0,
				"atlas_y": 0,
				"properties": {
					"blocks_sight": false,
					"walkable": true,
					"type": "floor"
				}
			},
			{
				"name": "wall_tile",
				"atlas_x": 1,
				"atlas_y": 0,
				"properties": {
					"blocks_sight": true,
					"walkable": false,
					"type": "wall"
				}
			}
		]
	}`

	var config AtlasConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify basic fields
	if config.Name != "test_atlas" {
		t.Errorf("Expected name 'test_atlas', got '%s'", config.Name)
	}

	if config.Layer != "base" {
		t.Errorf("Expected layer 'base', got '%s'", config.Layer)
	}

	if config.TileWidth != 64 {
		t.Errorf("Expected tile_width 64, got %d", config.TileWidth)
	}

	if config.TileHeight != 64 {
		t.Errorf("Expected tile_height 64, got %d", config.TileHeight)
	}

	// Verify tiles
	if len(config.Tiles) != 2 {
		t.Fatalf("Expected 2 tiles, got %d", len(config.Tiles))
	}

	// Check first tile
	floorTile := config.Tiles[0]
	if floorTile.Name != "floor_tile" {
		t.Errorf("Expected tile name 'floor_tile', got '%s'", floorTile.Name)
	}

	if floorTile.AtlasX != 0 || floorTile.AtlasY != 0 {
		t.Errorf("Expected atlas position (0, 0), got (%d, %d)", floorTile.AtlasX, floorTile.AtlasY)
	}

	// Check properties
	walkable := floorTile.GetTilePropertyBool("walkable", false)
	if !walkable {
		t.Error("Expected floor tile to be walkable")
	}

	blocksSight := floorTile.GetTilePropertyBool("blocks_sight", true)
	if blocksSight {
		t.Error("Expected floor tile to not block sight")
	}

	tileType := floorTile.GetTilePropertyString("type", "")
	if tileType != "floor" {
		t.Errorf("Expected type 'floor', got '%s'", tileType)
	}
}

func TestTileDefinitionProperties(t *testing.T) {
	tile := TileDefinition{
		Name:   "test_tile",
		AtlasX: 0,
		AtlasY: 0,
		Properties: map[string]interface{}{
			"bool_prop":   true,
			"string_prop": "test_value",
			"int_prop":    42.0, // JSON numbers are float64
		},
	}

	// Test bool property
	boolVal := tile.GetTilePropertyBool("bool_prop", false)
	if !boolVal {
		t.Error("Expected bool_prop to be true")
	}

	// Test string property
	strVal := tile.GetTilePropertyString("string_prop", "")
	if strVal != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", strVal)
	}

	// Test int property
	intVal := tile.GetTilePropertyInt("int_prop", 0)
	if intVal != 42 {
		t.Errorf("Expected 42, got %d", intVal)
	}

	// Test default values
	missingBool := tile.GetTilePropertyBool("missing", true)
	if !missingBool {
		t.Error("Expected default value true for missing property")
	}

	missingStr := tile.GetTilePropertyString("missing", "default")
	if missingStr != "default" {
		t.Errorf("Expected 'default', got '%s'", missingStr)
	}

	missingInt := tile.GetTilePropertyInt("missing", 99)
	if missingInt != 99 {
		t.Errorf("Expected 99, got %d", missingInt)
	}
}

func TestManagerLayerRegistration(t *testing.T) {
	manager := NewManager()

	// Create mock atlases
	atlas1 := &Atlas{
		Config: &AtlasConfig{
			Name:  "atlas1",
			Layer: "base",
		},
	}

	atlas2 := &Atlas{
		Config: &AtlasConfig{
			Name:  "atlas2",
			Layer: "objects",
		},
	}

	// Register atlases
	err := manager.RegisterAtlas(atlas1)
	if err != nil {
		t.Fatalf("Failed to register atlas1: %v", err)
	}

	err = manager.RegisterAtlas(atlas2)
	if err != nil {
		t.Fatalf("Failed to register atlas2: %v", err)
	}

	// Verify retrieval by layer
	retrieved, ok := manager.GetAtlasByLayer("base")
	if !ok {
		t.Fatal("Failed to retrieve atlas by layer 'base'")
	}
	if retrieved.Config.Name != "atlas1" {
		t.Errorf("Expected atlas 'atlas1', got '%s'", retrieved.Config.Name)
	}

	// Verify retrieval by name
	retrieved, ok = manager.GetAtlasByName("atlas2")
	if !ok {
		t.Fatal("Failed to retrieve atlas by name 'atlas2'")
	}
	if retrieved.Config.Layer != "objects" {
		t.Errorf("Expected layer 'objects', got '%s'", retrieved.Config.Layer)
	}

	// Test duplicate layer prevention
	atlas3 := &Atlas{
		Config: &AtlasConfig{
			Name:  "atlas3",
			Layer: "base", // Duplicate layer
		},
	}

	err = manager.RegisterAtlas(atlas3)
	if err == nil {
		t.Error("Expected error when registering duplicate layer")
	}

	// Test GetLayers
	layers := manager.GetLayers()
	if len(layers) != 2 {
		t.Errorf("Expected 2 layers, got %d", len(layers))
	}
}

func TestAtlasConfigValidation(t *testing.T) {
	// Create a temporary test config file
	tempFile, err := os.CreateTemp("", "atlas_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write invalid config (missing tile dimensions)
	invalidConfig := `{
		"name": "invalid",
		"layer": "test",
		"image_path": "nonexistent.png",
		"tile_width": 0,
		"tile_height": 0,
		"tiles": []
	}`

	if _, err := tempFile.Write([]byte(invalidConfig)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	// Try to load the invalid config
	_, err = LoadAtlas(tempFile.Name())
	if err == nil {
		t.Error("Expected error when loading atlas with invalid dimensions")
	}
}

func TestExampleConfigFiles(t *testing.T) {
	// Test that the example config files are valid JSON
	configs := []string{
		"../data/atlases/base_layer.json",
		"../data/atlases/objects_layer.json",
	}

	for _, configPath := range configs {
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Logf("Skipping %s (file not found, this is OK for unit tests)", configPath)
			continue
		}

		var config AtlasConfig
		err = json.Unmarshal(data, &config)
		if err != nil {
			t.Errorf("Failed to parse %s: %v", configPath, err)
			continue
		}

		// Basic validation
		if config.Name == "" {
			t.Errorf("%s: name is empty", configPath)
		}
		if config.Layer == "" {
			t.Errorf("%s: layer is empty", configPath)
		}
		if config.TileWidth <= 0 || config.TileHeight <= 0 {
			t.Errorf("%s: invalid tile dimensions", configPath)
		}
		if len(config.Tiles) == 0 {
			t.Errorf("%s: no tiles defined", configPath)
		}

		// Verify all tiles have names
		for i, tile := range config.Tiles {
			if tile.Name == "" {
				t.Errorf("%s: tile %d has no name", configPath, i)
			}
		}
	}
}
