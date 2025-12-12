# Sprite Atlas System

A data-driven sprite atlas loading and rendering system for Outpost9.

## Overview

The atlas system allows you to define sprite atlases using JSON configuration files, specifying tile sizes, source images, and individual tile definitions with semantic names and properties. This enables easy level editing and data-driven game content.

## Features

- **Data-driven configuration**: Define atlases in JSON format
- **Multiple layers**: Support for multiple atlas layers (base, objects, etc.)
- **Semantic tile naming**: Reference tiles by meaningful names (e.g., "nw_wall_corner")
- **Custom properties**: Attach arbitrary properties to tiles (collision, type, etc.)
- **Easy rendering**: Simple API for drawing tiles to the screen
- **Property accessors**: Helper methods for retrieving typed properties

## Usage

### 1. Creating an Atlas Configuration

Create a JSON file defining your sprite atlas:

```json
{
  "name": "base_tiles",
  "layer": "base",
  "image_path": "Art/base_tiles.png",
  "tile_width": 64,
  "tile_height": 64,
  "tiles": [
    {
      "name": "floor_metal_01",
      "atlas_x": 0,
      "atlas_y": 0,
      "properties": {
        "blocks_sight": false,
        "walkable": true,
        "type": "floor"
      }
    },
    {
      "name": "wall_nw_corner",
      "atlas_x": 0,
      "atlas_y": 1,
      "properties": {
        "blocks_sight": true,
        "walkable": false,
        "type": "wall"
      }
    }
  ]
}
```

### 2. Loading Atlases in Code

```go
import "chosenoffset.com/outpost9/internal/world/atlas"

// Create an atlas manager
manager := atlas.NewManager()

// Load atlas configurations
err := manager.LoadAtlasConfig("data/atlases/base_layer.json")
if err != nil {
    log.Fatal(err)
}

err = manager.LoadAtlasConfig("data/atlases/objects_layer.json")
if err != nil {
    log.Fatal(err)
}
```

### 3. Drawing Tiles

```go
// Draw a tile from the base layer
err := manager.DrawTile(screen, "base", "floor_metal_01", x, y)
if err != nil {
    log.Printf("Failed to draw tile: %v", err)
}

// Draw a tile with custom options
opts := &ebiten.DrawImageOptions{}
opts.GeoM.Translate(x, y)
err = manager.DrawTileWithOptions(screen, "objects", "desk_west", opts)
```

### 4. Accessing Tile Properties

```go
// Get a tile definition
tile, err := manager.GetTile("base", "wall_nw_corner")
if err != nil {
    log.Fatal(err)
}

// Access properties
blocksSight := tile.GetTilePropertyBool("blocks_sight", false)
tileType := tile.GetTilePropertyString("type", "unknown")
```

### 5. Direct Atlas Access

```go
// Get an atlas directly
atlas, ok := manager.GetAtlasByLayer("base")
if !ok {
    log.Fatal("Base layer not found")
}

// Get a tile from the atlas
tile, ok := atlas.GetTile("floor_metal_01")
if ok {
    // Draw the tile
    atlas.DrawTileDef(screen, tile, x, y)
}
```

## JSON Configuration Schema

### Atlas Configuration

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique name for the atlas |
| `layer` | string | Layer this atlas belongs to (e.g., "base", "objects") |
| `image_path` | string | Path to the PNG atlas image file |
| `tile_width` | int | Width of each tile in pixels |
| `tile_height` | int | Height of each tile in pixels |
| `tiles` | array | Array of tile definitions |

### Tile Definition

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Semantic name for the tile (e.g., "nw_wall_corner") |
| `atlas_x` | int | X position in the atlas (in tile units, not pixels) |
| `atlas_y` | int | Y position in the atlas (in tile units, not pixels) |
| `properties` | object | Custom properties (optional) |

### Common Properties

While properties are flexible, here are some commonly used ones:

| Property | Type | Description |
|----------|------|-------------|
| `blocks_sight` | bool | Whether the tile blocks line of sight |
| `walkable` | bool | Whether the tile can be walked through |
| `type` | string | Semantic type (e.g., "floor", "wall", "furniture") |
| `direction` | string | Directional facing (e.g., "north", "south") |
| `interactive` | bool | Whether the tile can be interacted with |

## Atlas Organization

### Recommended Layer Structure

- **base**: Floor and wall tiles that form the map structure
- **objects**: Furniture, machines, and other interactive objects
- **effects**: Visual effects, overlays, or temporary elements
- **ui**: UI elements that might be rendered in the game world

### Atlas Image Layout

Organize your atlas image as a grid where:
- Each tile occupies exactly `tile_width × tile_height` pixels
- Tiles are referenced by their grid position (0-indexed)
- Example for 64x64 tiles:
  ```
  [0,0] [1,0] [2,0] [3,0]
  [0,1] [1,1] [2,1] [3,1]
  [0,2] [1,2] [2,2] [3,2]
  ```

## Example: Integrating with Game Loop

```go
type Game struct {
    atlasManager *atlas.Manager
    // ... other fields
}

func (g *Game) Init() error {
    // Create atlas manager
    g.atlasManager = atlas.NewManager()

    // Load atlases
    if err := g.atlasManager.LoadAtlasConfig("data/atlases/base_layer.json"); err != nil {
        return err
    }
    if err := g.atlasManager.LoadAtlasConfig("data/atlases/objects_layer.json"); err != nil {
        return err
    }

    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    // Draw base layer
    for y := 0; y < g.mapHeight; y++ {
        for x := 0; x < g.mapWidth; x++ {
            tileName := g.getTileNameAt(x, y)
            screenX := float64(x * 64)
            screenY := float64(y * 64)

            g.atlasManager.DrawTile(screen, "base", tileName, screenX, screenY)
        }
    }

    // Draw objects layer
    for _, obj := range g.objects {
        g.atlasManager.DrawTile(screen, "objects", obj.TileName, obj.X, obj.Y)
    }
}
```

## Creating Atlas Images

1. Create a PNG image with dimensions that are multiples of your tile size
2. Arrange tiles in a grid layout
3. Ensure no spacing or padding between tiles
4. Save as PNG with transparency support
5. Reference tile positions in your JSON configuration

### Example Tool Workflow

You can use tools like:
- **Aseprite**: For pixel art and sprite sheet creation
- **TexturePacker**: For automatically packing sprites into atlases
- **Tiled**: For level editing with tileset support
- **GIMP/Photoshop**: For manual atlas creation

## API Reference

### Manager

- `NewManager()` - Create a new atlas manager
- `LoadAtlasConfig(configPath string)` - Load an atlas from JSON
- `RegisterAtlas(atlas *Atlas)` - Register a loaded atlas
- `GetAtlasByLayer(layer string)` - Get atlas by layer name
- `GetAtlasByName(name string)` - Get atlas by atlas name
- `DrawTile(screen, layer, tileName, x, y)` - Draw a tile
- `DrawTileWithOptions(screen, layer, tileName, opts)` - Draw with options
- `GetTile(layer, tileName)` - Get a tile definition
- `GetLayers()` - Get all registered layer names

### Atlas

- `LoadAtlas(configPath string)` - Load an atlas from JSON file
- `GetTile(name string)` - Get a tile definition by name
- `GetTileSubImage(tile *TileDefinition)` - Get the tile's sub-image
- `GetTileSubImageByName(name string)` - Get sub-image by tile name
- `DrawTile(screen, tileName, x, y)` - Draw a tile
- `DrawTileDef(screen, tile, x, y)` - Draw a tile definition
- `DrawTileWithOptions(screen, tileName, opts)` - Draw with options

### TileDefinition

- `GetTileProperty(key string)` - Get a property value
- `GetTilePropertyBool(key, defaultVal)` - Get boolean property
- `GetTilePropertyString(key, defaultVal)` - Get string property
- `GetTilePropertyInt(key, defaultVal)` - Get integer property

## License

Part of the Outpost9 project.
