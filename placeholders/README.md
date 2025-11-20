# Placeholder Graphics System

This package provides procedurally generated placeholder graphics for Outpost-9, allowing you to develop gameplay systems without requiring final artwork.

## Overview

The placeholder system generates simple, color-coded sprite atlases at runtime. All graphics are created programmatically, so no art files are needed to get started.

## Generated Atlases

### 1. Base Layer (`Art/base_tiles.png`)
**Size:** 48x64 pixels (3x4 tiles, 16x16 each)

Contains floor and wall tiles for the sci-fi base:
- **Floors:** Metal floors (2 variants), grating
- **Walls:** 9 wall segments (corners, edges, center)
- **Color Scheme:** Grays and dark blue-grays

### 2. Objects Layer (`Art/object_tiles.png`)
**Size:** 64x48 pixels (4x3 tiles, 16x16 each)

Contains furniture and interactive objects:
- **Furniture:** Desks, chairs
- **Electronics:** Computer terminals, consoles
- **Storage:** Lockers, crates
- **Machinery:** Generators

### 3. Entities (`Art/entities.png`)
**Size:** 64x64 pixels (4x4 tiles, 16x16 each)

Contains player, enemies, items, and effects:
- **Row 0:** Player sprites (idle, walk animations)
- **Row 1:** Enemy types (basic, elite, boss, turret)
- **Row 2:** Items (health, ammo, key, weapon)
- **Row 3:** Projectiles and effects (bullets, plasma, muzzle flash, impact)

## Usage

### Generating Placeholder Graphics

```bash
# Generate all placeholder atlases
go run cmd/genplaceholders/main.go
```

This creates:
- `Art/base_tiles.png`
- `Art/object_tiles.png`
- `Art/entities.png`
- `Art/atlases/*.json` (atlas configuration files)

### Atlas Configuration

Each atlas has a corresponding JSON configuration file in `Art/atlases/`:
- `base_layer.json` - Base tile definitions
- `objects_layer.json` - Object tile definitions
- `entities.json` - Entity sprite definitions

### Integration

The main game automatically loads the entities atlas for player rendering:

```go
// Loads in main.go's loadGame() function
entitiesAtlas, err := atlas.LoadAtlas("Art/atlases/entities.json", loader)
playerSprite, err := entitiesAtlas.GetTileImage("player_idle")
```

## Color Palette

### Base Tiles
- Floor Metal: `#787882` (medium gray-blue)
- Floor Grating: `#50505A` (dark gray)
- Walls: `#3C3C46` - `#465A5A` (dark blue-grays)

### Entities
- **Player:** `#00FF64` (bright green) - Highly visible
- **Enemies:** `#FF3232` (red), `#C800C8` (magenta), `#FF6400` (orange)
- **Items:** `#FFD700` (gold), `#00FF00` (green health)

## Customization

### Adding New Tiles

Edit `placeholders/atlases.go` and add to the appropriate generator function:

```go
// Example: Add a new enemy type
tiles[8] = CreateEnemySprite("drone")
```

### Modifying Colors

Edit `placeholders/generator.go` and modify the `ColorPalette` struct:

```go
var ColorPalette = struct {
    Player color.RGBA
    // ... other colors
}{
    Player: color.RGBA{0, 255, 100, 255}, // Change player color
}
```

### Creating Custom Sprites

Use the provided helper functions:
- `CreateSolidTile(color)` - Simple filled tile
- `CreateBorderedTile(fill, border, width)` - Tile with border
- `CreatePatternedTile(base, pattern, type)` - Tile with pattern
- `CreateCircle(fill, outline)` - Circular sprite

## Replacing with Real Art

When you have final artwork:

1. Replace the PNG files in `Art/`
2. Update the atlas JSON configurations if tile positions change
3. **No code changes needed** - the game uses the same atlas loading system

The placeholder graphics use the exact same format as final artwork, so integration is seamless.

## Features

- ✅ No external dependencies - pure Go image generation
- ✅ Consistent color coding for easy identification
- ✅ Compatible with existing atlas system
- ✅ Automatic fallback rendering if assets fail to load
- ✅ Simple to modify and extend
- ✅ Professional appearance suitable for prototyping

## File Structure

```
placeholders/
├── generator.go      # Core sprite generation utilities
├── atlases.go        # Atlas-specific generators
└── README.md         # This file

cmd/genplaceholders/
└── main.go           # CLI tool to generate assets

Art/
├── base_tiles.png    # Generated base layer atlas
├── object_tiles.png  # Generated objects layer atlas
├── entities.png      # Generated entities atlas
└── atlases/
    ├── base_layer.json
    ├── objects_layer.json
    └── entities.json
```

## Next Steps

1. **Generate the assets:** Run `go run cmd/genplaceholders/main.go`
2. **Test in-game:** Run the game to see placeholder graphics in action
3. **Develop gameplay:** Build features without waiting for final art
4. **Replace incrementally:** Swap placeholders for real art as it becomes available

The placeholder system lets you focus on gameplay mechanics, with clear visual feedback for all game elements!
