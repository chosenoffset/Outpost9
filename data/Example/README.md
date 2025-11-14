# Outpost9 Example Game Data

This directory contains example game data demonstrating the data-driven systems in Outpost9.

## Overview

The Example game shows how to use the sprite atlas and map loading systems to create a complete playable game without hardcoding any content.

## Files

### Assets
- **assets/DungeonTileset.png** - 16x16 pixel dungeon tileset sprite atlas
- **assets/tile_list_v1.7** - Tile metadata file listing all sprites, positions, and sizes

### Configuration
- **atlas.json** - Sprite atlas configuration defining floor and wall tiles
- **level1.json** - Example level layout using the defined tiles

## Running the Example

From the project root:
```bash
go run .
```

Use WASD keys to move the player character around the dungeon.

## Data-Driven Systems

### 1. Sprite Atlas System

The sprite atlas system (`atlas/` package) allows you to define sprite sheets using JSON configuration:

```json
{
  "name": "dungeon_tileset",
  "layer": "base",
  "image_path": "data/Example/assets/DungeonTileset.png",
  "tile_width": 16,
  "tile_height": 16,
  "tiles": [
    {
      "name": "floor_1",
      "atlas_x": 1,
      "atlas_y": 4,
      "properties": {
        "blocks_sight": false,
        "walkable": true,
        "type": "floor"
      }
    }
  ]
}
```

**Key Features:**
- Define tiles by their grid position in the atlas image
- Attach custom properties (gameplay flags, metadata, etc.)
- Reference tiles by semantic names instead of coordinates
- Support for multiple atlas layers (base, objects, effects, etc.)

### 2. Map Loading System

The map loading system (`maploader/` package) loads level layouts from JSON:

```json
{
  "name": "Example Level 1",
  "width": 13,
  "height": 10,
  "tile_size": 16,
  "render_tile_size": 64,
  "atlas": "data/Example/atlas.json",
  "player_spawn": {
    "x": 400,
    "y": 300
  },
  "tiles": [
    ["wall_top_left", "wall_top_mid", ...],
    ["wall_left", "floor_1", ...],
    ...
  ]
}
```

**Key Features:**
- 2D tile array defines the level layout
- References tiles by name from the atlas
- Configurable player spawn point
- Automatic atlas loading
- Query system for tile properties (walkable, blocks sight, etc.)

## Creating Your Own Content

### Step 1: Create Your Sprite Atlas

1. Create a PNG image with your sprites arranged in a grid
2. All tiles should be the same size (e.g., 16x16, 32x32, 64x64)
3. No spacing or padding between tiles

### Step 2: Configure the Atlas

Create a JSON file defining your sprites:

```json
{
  "name": "my_atlas",
  "layer": "base",
  "image_path": "path/to/your/sprites.png",
  "tile_width": 16,
  "tile_height": 16,
  "tiles": [
    {
      "name": "my_tile",
      "atlas_x": 0,
      "atlas_y": 0,
      "properties": {
        "blocks_sight": false,
        "walkable": true,
        "type": "floor"
      }
    }
  ]
}
```

**Coordinates:** `atlas_x` and `atlas_y` are in tile units (not pixels). For example, if your tiles are 16x16 and you want the tile at pixel position (32, 48), use `"atlas_x": 2, "atlas_y": 3"` (32/16=2, 48/16=3).

### Step 3: Create Your Level

Create a JSON file defining your level layout:

```json
{
  "name": "My Level",
  "width": 10,
  "height": 10,
  "tile_size": 16,
  "render_tile_size": 32,
  "atlas": "path/to/your/atlas.json",
  "player_spawn": {
    "x": 160,
    "y": 160
  },
  "tiles": [
    ["tile1", "tile2", "tile3", ...],
    ["tile4", "tile5", "tile6", ...],
    ...
  ]
}
```

**Field Descriptions:**
- `tile_size`: Size of tiles in the atlas PNG (e.g., 16 for 16x16 sprites)
- `render_tile_size`: Size to render each tile in the game (e.g., 64 for 64x64 game tiles)
- The system automatically scales: 16px atlas tiles → 64px game tiles (4x scale)
- Player spawn coordinates should be in pixels based on render_tile_size

The `tiles` array should have `height` rows, each with `width` tile names.

### Step 4: Load in Your Game

```go
gameMap, err := maploader.LoadMap("path/to/your/level.json")
if err != nil {
    log.Fatal(err)
}

// Access tile data
tileName, _ := gameMap.GetTileAt(x, y)
isWalkable := gameMap.IsWalkable(x, y)
blocksSight := gameMap.BlocksSight(x, y)

// Render tiles
for y := 0; y < gameMap.Data.Height; y++ {
    for x := 0; x < gameMap.Data.Width; x++ {
        tileName, _ := gameMap.GetTileAt(x, y)
        tile, _ := gameMap.Atlas.GetTile(tileName)
        gameMap.Atlas.DrawTileDef(screen, tile, float64(x*64), float64(y*64))
    }
}
```

## Tile Properties

You can attach any custom properties to tiles:

### Common Properties

| Property | Type | Description |
|----------|------|-------------|
| `blocks_sight` | bool | Whether the tile blocks line of sight for shadow casting |
| `walkable` | bool | Whether the player can walk through this tile |
| `type` | string | Semantic type ("floor", "wall", "door", etc.) |

### Custom Properties

Add any properties you need for your game logic:

```json
{
  "name": "lava_floor",
  "atlas_x": 5,
  "atlas_y": 3,
  "properties": {
    "walkable": true,
    "blocks_sight": false,
    "type": "floor",
    "damage": 10,
    "damage_type": "fire",
    "animated": true,
    "animation_frames": 4
  }
}
```

Access properties in code:

```go
tile, _ := gameMap.GetTileDefAt(x, y)

damage := tile.GetTilePropertyInt("damage", 0)
damageType := tile.GetTilePropertyString("damage_type", "")
isAnimated := tile.GetTilePropertyBool("animated", false)
```

## Tile Scaling

The system automatically scales atlas sprites to your desired render size:

- **Atlas tile size** (`tile_size`): Size of sprites in the PNG (e.g., 16x16)
- **Render tile size** (`render_tile_size`): Size to display in game (e.g., 64x64)

Example: 16px atlas tiles rendered at 64px = 4x scaling

```go
tileScale := float64(RenderTileSize) / float64(AtlasTileSize)
// 64 / 16 = 4x scale
```

**Benefits:**
- Use smaller source images (saves memory and file size)
- Render at any size you want
- Mix different tile sizes in different games
- No hardcoded sizes - fully configurable per level

## Architecture

```
main.go
  ├─ Loads map via maploader
  ├─ maploader.LoadMap()
  │   ├─ Parses level1.json
  │   └─ Loads atlas via atlas.LoadAtlas()
  │       └─ Parses atlas.json
  │           └─ Loads DungeonTileset.png
  │
  ├─ Generates wall segments from map data
  │   └─ Uses tile "blocks_sight" property
  │
  └─ Renders each frame
      ├─ Draws tiles using atlas.DrawTileDef()
      ├─ Draws shadows using wall segments
      └─ Draws player
```

## Extending the Example

### Adding More Tiles

1. Edit `atlas.json` to add new tile definitions
2. Reference the new tiles in `level1.json`

### Creating New Levels

1. Copy `level1.json` to `level2.json`
2. Modify the `tiles` array to create a new layout
3. Update main.go to load the new level

### Adding Multiple Layers

1. Create a second atlas (e.g., `objects_atlas.json`) with `"layer": "objects"`
2. Load both atlases using `atlas.Manager`
3. Render base layer first, then objects layer

See `atlas/README.md` for more details on the atlas system.

## License

This example data uses [0x72's DungeonTileset](https://0x72.itch.io/dungeontileset-ii) which is public domain (CC0).
