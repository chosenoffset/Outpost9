# Outpost 9: Top-Down 2D Game with Dynamic LOS Shadows (Like Nox) (Go/Ebitengine)
A sci-fi roguelike with dynamic LOS shadows similar to the game Nox and procedurally generated levels.

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

Use WASD to move around the procedurally generated outpost. The game features dynamic line-of-sight shadows that cast from walls in real-time. Each playthrough generates a unique level layout!

## Features
- **Procedural level generation** - Each playthrough generates unique outpost layouts from room templates
- **Room-based design** - Define reusable room templates with connection points
- **Raycasted shadows** - Real-time dynamic line-of-sight system inspired by Nox
- **Data-driven sprite atlases** - Define sprites in JSON with semantic naming
- **Tile property system** - Attach custom properties to tiles (walkable, blocks sight, etc.)
- **Multi-layer support** - Separate atlases for different render layers

## Architecture

### Room-Based Procedural Generation (`room/`)
Generate unique levels from room templates:
- Define individual room templates with tiles and metadata
- Room libraries organize collections of rooms
- Procedural generator combines rooms with spawn weights and constraints
- Each playthrough creates a different layout
- See `data/Example/ROOM_GENERATION_README.md` for complete documentation

### Sprite Atlas System (`atlas/`)
Load and render sprites from sprite sheets with JSON configuration:
- Define tiles by grid position with semantic names
- Attach custom properties for gameplay logic
- Support for multiple atlas layers

### Map Loader (`maploader/`)
Generates levels from room libraries with automatic atlas integration:
- Procedural generation from room templates
- Query tile properties (walkable, blocks sight, etc.)
- Automatic wall segment generation for shadow casting
- See `data/Example/ROOM_GENERATION_README.md` for usage guide

### Example Game (`data/Example/`)
A complete working example showing:
- Dungeon tileset with floors and walls
- Room library with 8 diverse room types
- Shadow casting with data-driven walls
- Player movement and collision

## Creating Your Own Content

See `data/Example/ROOM_GENERATION_README.md` for a complete guide on creating your own room templates and designing procedurally generated levels.


![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)