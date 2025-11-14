# Outpost 9: Top-Down 2D Game with Dynamic LOS Shadows (Like Nox) (Go/Ebitengine)
A sci-fi rougelike with dynamic LOS shadows similar to the game Nox.

The project is in very early stages, rebuilding from an initial project that was lost to an SSD failure.
Star the project and follow along as the game takes shape!


## Demo
![alt text](OutpostLOS.gif)

## Quick Start
```bash
git clone https://github.com/chosenoffset/Outpost9.git
cd Outpost9
go run .
```

Use WASD to move around the example dungeon. The game features dynamic line-of-sight shadows that cast from walls in real-time.

### Command-Line Options

```bash
# Run the example game (default)
go run .

# Specify a different game data directory
go run . -game Example
go run . -game Outpost9

# Specify a different level file
go run . -game Example -level level2.json

# Get help
go run . -help
```

## Features
- **Raycasted shadows** - Real-time dynamic line-of-sight system inspired by Nox
- **Data-driven sprite atlases** - Define sprites in JSON with semantic naming
- **Data-driven level loading** - Create levels with JSON configuration
- **Tile property system** - Attach custom properties to tiles (walkable, blocks sight, etc.)
- **Multi-layer support** - Separate atlases for different render layers

## Architecture

### Sprite Atlas System (`atlas/`)
Load and render sprites from sprite sheets with JSON configuration:
- Define tiles by grid position with semantic names
- Attach custom properties for gameplay logic
- Support for multiple atlas layers
- See `atlas/README.md` for complete documentation

### Map Loader (`maploader/`)
Load levels from JSON with automatic atlas integration:
- 2D tile arrays define level layouts
- Query tile properties (walkable, blocks sight, etc.)
- Automatic wall segment generation for shadow casting
- See `data/Example/README.md` for usage guide

### Example Game (`data/Example/`)
A complete working example showing:
- Dungeon tileset with floors and walls
- Example level layout
- Shadow casting with data-driven walls
- Player movement and collision

## Creating Your Own Content

See `data/Example/README.md` for a complete guide on creating your own levels and sprite atlases.


![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)