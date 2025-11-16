# Room-Based Procedural Level Generation

This document explains the new room-based procedural generation system for Outpost 9.

## Overview

Outpost 9 now supports procedurally generated levels! Instead of manually creating entire level files, you can define individual room templates and let the game combine them randomly to create unique outposts for each playthrough.

## How It Works

### 1. Room Libraries

A **Room Library** is a JSON file that contains:
- A collection of room definitions (templates)
- Shared configuration (atlas path, tile size, default floor tile)
- Metadata about each room (type, tags, spawn weights)

### 2. Room Definitions

Each room is a template that includes:
- **Tile data**: A 2D array of tiles (like a mini level)
- **Dimensions**: Width and height
- **Connection points**: Doors or exits where rooms can connect
- **Metadata**: Type, tags, spawn weight, min/max occurrences

### 3. Procedural Generation

When you select a room library from the main menu, the game:
1. Loads all room definitions
2. Selects rooms based on spawn weights and constraints
3. Places rooms in the level
4. Connects rooms via their connection points
5. Generates a complete playable level

## File Structure

### Room Library Format (`rooms.json`)

```json
{
  "name": "Example Room Library",
  "description": "A collection of rooms for procedural generation",
  "atlas_path": "data/Example/atlas.json",
  "tile_size": 16,
  "floor_tile": "f1x",
  "rooms": [
    {
      "name": "entrance_small",
      "description": "Small entrance room",
      "type": "entrance",
      "tags": ["start", "small"],
      "width": 7,
      "height": 5,
      "spawn_weight": 10,
      "min_count": 1,
      "max_count": 1,
      "connections": [
        {
          "x": 6,
          "y": 2,
          "direction": "east",
          "type": "door"
        }
      ],
      "tiles": [
        ["nwt", "ntx", "ntx", "ntx", "ntx", "ntx", "net"],
        ["nwb", "nbx", "nbx", "nbx", "nbx", "nbx", "neb"],
        ["wwx", "", "", "", "", "", ""],
        ["swt", "stx", "stx", "stx", "stx", "stx", "set"],
        ["swb", "sbx", "sbx", "sbx", "sbx", "sbx", "seb"]
      ]
    }
  ]
}
```

### Room Properties

#### Required Fields
- `name`: Unique identifier for the room
- `width`, `height`: Dimensions in tiles
- `tiles`: 2D array of tile names [y][x]
- `type`: Room category (entrance, corridor, chamber, junction, exit)

#### Optional Fields
- `description`: Human-readable description
- `tags`: Array of strings for categorization
- `spawn_weight`: Higher = more likely to appear (default: 1)
- `min_count`: Minimum times this room must appear (default: 0)
- `max_count`: Maximum times this room can appear (0 = unlimited)
- `connections`: Array of connection points

#### Connection Points

Connection points define where rooms can link together:

```json
{
  "x": 6,          // Grid X position
  "y": 2,          // Grid Y position
  "direction": "east",  // north, south, east, west
  "type": "door"   // door, corridor, entrance, exit
}
```

## Room Types

The system recognizes several room types:

- **entrance**: Starting room (usually min_count: 1, max_count: 1)
- **corridor**: Connecting hallways
- **chamber**: Main gameplay rooms
- **junction**: Intersection/crossroads rooms
- **exit**: End rooms (for future use)

## Generator Configuration

The procedural generator can be configured in `main.go`:

```go
config := room.GeneratorConfig{
    MinRooms:     5,      // Minimum number of rooms
    MaxRooms:     10,     // Maximum number of rooms
    Seed:         0,      // Random seed (0 = random each time)
    ConnectAll:   true,   // Ensure all rooms are connected
    AllowOverlap: false,  // Allow rooms to overlap
}
```

## Example Rooms

The `rooms.json` file includes several example rooms:

1. **entrance_small**: 7x5 entrance room (always appears once)
2. **corridor_horizontal**: 5x3 horizontal hallway
3. **corridor_vertical**: 3x5 vertical hallway
4. **chamber_small**: 5x4 small room
5. **chamber_medium**: 8x6 medium room with multiple doors
6. **chamber_large**: 10x8 large open area
7. **room_with_interior**: 11x8 complex room with interior walls
8. **junction_cross**: 5x5 four-way intersection

## Using Procedural Generation

### From the Main Menu

1. Run the game
2. Select "Example" game
3. You'll see both traditional levels and procedural options:
   - `[PROCEDURAL] rooms.json` - Purple colored
   - Traditional level files - Gray colored
4. Select the procedural option and press SPACE

Each time you start a procedural level, you'll get a different layout!

### Creating Your Own Rooms

1. Copy `data/Example/rooms.json` as a template
2. Design rooms using your atlas tiles
3. Define connection points where rooms should link
4. Set spawn weights and constraints
5. Test different configurations

## Tips for Room Design

### Good Room Design
- Keep rooms small to medium sized (5x5 to 15x15)
- Always include at least one connection point
- Use appropriate spawn weights (rare rooms: 1-5, common rooms: 10-20)
- Set min_count=1 for entrance rooms
- Add variety with different room types

### Connection Points
- Place connections on room edges (not corners)
- Match connection directions to actual openings in tiles
- Use consistent connection types
- Consider multiple connections for larger rooms

### Balancing
- Higher spawn_weight = appears more often
- Use max_count to limit special rooms
- Mix small and large rooms for variety
- Include corridors to connect chambers

## Technical Details

### Code Structure

The room-based system is implemented across several packages:

- **`room/room.go`**: Room definitions, library loading, validation
- **`room/generator.go`**: Procedural generation algorithm
- **`maploader/maploader.go`**: Integration with existing map system
- **`gamescanner/scanner.go`**: Detection of room library files
- **`menu/menu.go`**: UI for selecting procedural vs static levels

### Generation Algorithm

Current implementation uses a simple linear placement:
1. Select rooms based on weights and constraints
2. Place rooms left-to-right, wrapping to new rows
3. Generate final tile grid
4. Find player spawn in entrance room

**Note**: The connection system is prepared for future enhancement. Currently, rooms are placed without explicit connection matching, but all the infrastructure is in place to implement intelligent room linking.

## Future Enhancements

Potential improvements to the system:

1. **Smart Room Connection**: Use connection points to snap rooms together
2. **Corridors Between Rooms**: Generate connecting hallways automatically
3. **Room Rotation**: Rotate rooms 90/180/270 degrees for more variety
4. **Biome Support**: Different room libraries for different themes
5. **Enemy Placement**: Spawn enemies based on room tags
6. **Item Distribution**: Place items in rooms automatically
7. **Graph-Based Generation**: Use graph algorithms for better layouts
8. **Difficulty Scaling**: Harder rooms further from entrance

## Troubleshooting

### "No entrance rooms found in library"
- Ensure at least one room has `"type": "entrance"`
- Check that the room library has a `"rooms"` array

### Rooms look disconnected
- This is expected with the current linear placement
- Future updates will implement proper room connection

### Player spawns in wrong location
- Check that your entrance room is properly marked
- Verify tiles in the entrance room are walkable

### Tiles not rendering
- Verify all tile names in rooms exist in the atlas
- Check that `atlas_path` points to correct atlas file
- Ensure `tile_size` matches your atlas configuration

## Example Workflow

1. **Design a room** in your level editor or by hand
2. **Extract the tile array** into a room definition
3. **Mark connection points** where doors/exits exist
4. **Add to room library** with appropriate metadata
5. **Test generation** by running the game
6. **Adjust spawn weights** based on how often you want it to appear
7. **Iterate** until you have a good variety of rooms

## Resources

- Original level format: `data/Example/level1.json`
- Example room library: `data/Example/rooms.json`
- Atlas definition: `data/Example/atlas.json`
- Main documentation: `data/Example/README.md`

Happy room designing! Each playthrough of your procedurally generated outpost will be unique!
