package room

import (
	"fmt"
	"math/rand"
	"time"
)

// PlacedRoom represents a room instance placed in the level
type PlacedRoom struct {
	Room      *RoomDefinition
	X         int // World position X (in tiles)
	Y         int // World position Y (in tiles)
	ID        int // Unique identifier for this instance
	Connected bool
}

// GeneratedLevel represents a procedurally generated level
type GeneratedLevel struct {
	Name        string         // Level name
	Width       int            // Total level width in tiles
	Height      int            // Total level height in tiles
	TileSize    int            // Tile size in pixels
	AtlasPath   string         // Path to atlas
	FloorTile   string         // Default floor tile
	Tiles       [][]string     // Generated tile grid [y][x]
	PlacedRooms []*PlacedRoom  // All placed rooms
	PlayerSpawn PlayerSpawn    // Player starting position
}

// PlayerSpawn represents the player's starting position
type PlayerSpawn struct {
	X int // X position in pixels
	Y int // Y position in pixels
}

// GeneratorConfig holds configuration for level generation
type GeneratorConfig struct {
	MinRooms     int     // Minimum number of rooms to generate
	MaxRooms     int     // Maximum number of rooms to generate
	LevelWidth   int     // Target level width in tiles (0 = auto)
	LevelHeight  int     // Target level height in tiles (0 = auto)
	Seed         int64   // Random seed (0 = use current time)
	ConnectAll   bool    // Ensure all rooms are connected
	AllowOverlap bool    // Allow rooms to overlap (not recommended)
}

// Generator handles procedural level generation
type Generator struct {
	library *RoomLibrary
	config  GeneratorConfig
	rng     *rand.Rand
}

// NewGenerator creates a new level generator
func NewGenerator(library *RoomLibrary, config GeneratorConfig) *Generator {
	seed := config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &Generator{
		library: library,
		config:  config,
		rng:     rand.New(rand.NewSource(seed)),
	}
}

// Generate creates a new procedurally generated level
func (g *Generator) Generate() (*GeneratedLevel, error) {
	// Determine number of rooms to generate
	numRooms := g.config.MinRooms
	if g.config.MaxRooms > g.config.MinRooms {
		numRooms += g.rng.Intn(g.config.MaxRooms - g.config.MinRooms + 1)
	}

	// Select rooms to place
	roomsToPlace, err := g.selectRooms(numRooms)
	if err != nil {
		return nil, err
	}

	// Place rooms in the level
	placedRooms, err := g.placeRooms(roomsToPlace)
	if err != nil {
		return nil, err
	}

	// Calculate level bounds
	levelWidth, levelHeight := g.calculateBounds(placedRooms)

	// Create tile grid
	tiles := g.createTileGrid(levelWidth, levelHeight, placedRooms)

	// Find player spawn
	playerSpawn := g.findPlayerSpawn(placedRooms)

	level := &GeneratedLevel{
		Name:        "Procedurally Generated Outpost",
		Width:       levelWidth,
		Height:      levelHeight,
		TileSize:    g.library.TileSize,
		AtlasPath:   g.library.AtlasPath,
		FloorTile:   g.library.FloorTile,
		Tiles:       tiles,
		PlacedRooms: placedRooms,
		PlayerSpawn: playerSpawn,
	}

	return level, nil
}

// selectRooms chooses which rooms to place in the level
func (g *Generator) selectRooms(numRooms int) ([]*RoomDefinition, error) {
	var selected []*RoomDefinition
	usageCount := make(map[string]int)

	// First, ensure we have required rooms
	// Start with an entrance
	entrances := g.library.GetRoomsByType("entrance")
	if len(entrances) == 0 {
		return nil, fmt.Errorf("no entrance rooms found in library")
	}
	entrance := entrances[g.rng.Intn(len(entrances))]
	selected = append(selected, entrance)
	usageCount[entrance.Name]++

	// Add rooms with min_count requirements
	for _, room := range g.library.Rooms {
		if room.MinCount > 0 {
			for i := 0; i < room.MinCount; i++ {
				if len(selected) < numRooms {
					selected = append(selected, room)
					usageCount[room.Name]++
				}
			}
		}
	}

	// Fill remaining slots with weighted random selection
	for len(selected) < numRooms {
		room := g.selectWeightedRoom(usageCount)
		if room == nil {
			break // No more valid rooms
		}
		selected = append(selected, room)
		usageCount[room.Name]++
	}

	return selected, nil
}

// selectWeightedRoom selects a room based on spawn weights
func (g *Generator) selectWeightedRoom(usageCount map[string]int) *RoomDefinition {
	// Calculate total weight of available rooms
	totalWeight := 0
	for _, room := range g.library.Rooms {
		// Check if room has reached max count
		if room.MaxCount > 0 && usageCount[room.Name] >= room.MaxCount {
			continue
		}
		weight := room.SpawnWeight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nil
	}

	// Select random room based on weight
	roll := g.rng.Intn(totalWeight)
	currentWeight := 0
	for _, room := range g.library.Rooms {
		if room.MaxCount > 0 && usageCount[room.Name] >= room.MaxCount {
			continue
		}
		weight := room.SpawnWeight
		if weight <= 0 {
			weight = 1
		}
		currentWeight += weight
		if roll < currentWeight {
			return room
		}
	}

	return nil
}

// placeRooms places selected rooms in the level using a simple linear layout
func (g *Generator) placeRooms(rooms []*RoomDefinition) ([]*PlacedRoom, error) {
	var placed []*PlacedRoom
	currentX := 0
	currentY := 0
	maxHeight := 0

	for i, room := range rooms {
		// Simple linear placement for now (we can make this more sophisticated later)
		placedRoom := &PlacedRoom{
			Room:      room,
			X:         currentX,
			Y:         currentY,
			ID:        i,
			Connected: true, // For now, assume all rooms are connected
		}
		placed = append(placed, placedRoom)

		// Move to next position
		currentX += room.Width
		if room.Height > maxHeight {
			maxHeight = room.Height
		}

		// Wrap to next row if needed (simple grid layout)
		if currentX > 40 { // Arbitrary wrap point
			currentX = 0
			currentY += maxHeight
			maxHeight = 0
		}
	}

	return placed, nil
}

// calculateBounds determines the total level size based on placed rooms
func (g *Generator) calculateBounds(rooms []*PlacedRoom) (width, height int) {
	if len(rooms) == 0 {
		return 10, 10 // Minimum size
	}

	maxX := 0
	maxY := 0

	for _, room := range rooms {
		rightEdge := room.X + room.Room.Width
		bottomEdge := room.Y + room.Room.Height

		if rightEdge > maxX {
			maxX = rightEdge
		}
		if bottomEdge > maxY {
			maxY = bottomEdge
		}
	}

	// Use config dimensions if specified, otherwise use calculated bounds
	width = maxX
	height = maxY

	if g.config.LevelWidth > 0 {
		width = g.config.LevelWidth
	}
	if g.config.LevelHeight > 0 {
		height = g.config.LevelHeight
	}

	return width, height
}

// createTileGrid generates the final tile grid from placed rooms
func (g *Generator) createTileGrid(width, height int, rooms []*PlacedRoom) [][]string {
	// Initialize grid with floor tiles
	tiles := make([][]string, height)
	for y := 0; y < height; y++ {
		tiles[y] = make([]string, width)
		for x := 0; x < width; x++ {
			tiles[y][x] = "" // Empty = will use floor_tile
		}
	}

	// Place room tiles
	for _, placedRoom := range rooms {
		room := placedRoom.Room
		for ry := 0; ry < room.Height; ry++ {
			for rx := 0; rx < room.Width; rx++ {
				worldX := placedRoom.X + rx
				worldY := placedRoom.Y + ry

				// Bounds check
				if worldX >= 0 && worldX < width && worldY >= 0 && worldY < height {
					tiles[worldY][worldX] = room.Tiles[ry][rx]
				}
			}
		}
	}

	return tiles
}

// findPlayerSpawn finds the player spawn point (in entrance room)
func (g *Generator) findPlayerSpawn(rooms []*PlacedRoom) PlayerSpawn {
	// Find entrance room
	for _, placedRoom := range rooms {
		if placedRoom.Room.Type == "entrance" {
			// Spawn in center of entrance room
			centerX := placedRoom.X + placedRoom.Room.Width/2
			centerY := placedRoom.Y + placedRoom.Room.Height/2

			return PlayerSpawn{
				X: centerX * g.library.TileSize,
				Y: centerY * g.library.TileSize,
			}
		}
	}

	// Fallback: spawn in first room
	if len(rooms) > 0 {
		centerX := rooms[0].X + rooms[0].Room.Width/2
		centerY := rooms[0].Y + rooms[0].Room.Height/2
		return PlayerSpawn{
			X: centerX * g.library.TileSize,
			Y: centerY * g.library.TileSize,
		}
	}

	// Ultimate fallback
	return PlayerSpawn{X: 100, Y: 100}
}
