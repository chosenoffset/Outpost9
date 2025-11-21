package room

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"chosenoffset.com/outpost9/furnishing"
)

// UsedConnection tracks a connection point that has been used
type UsedConnection struct {
	RoomID      int
	ConnIdx     int
	ConnectedTo int // ID of room connected to (-1 if corridor)
}

// Corridor represents a generated corridor between rooms
type Corridor struct {
	Tiles []CorridorTile // Tiles that make up the corridor
}

// CorridorTile represents a single tile in a corridor
type CorridorTile struct {
	X       int
	Y       int
	IsFloor bool // true for floor, false for wall
}

// PlacedRoom represents a room instance placed in the level
type PlacedRoom struct {
	Room            *RoomDefinition
	X               int   // World position X (in tiles)
	Y               int   // World position Y (in tiles)
	ID              int   // Unique identifier for this instance
	Connected       bool  // Whether this room is connected to the main graph
	UsedConnections []int // Indices of connections that have been used
}

// GetUnusedConnections returns indices of connections that haven't been used yet
func (pr *PlacedRoom) GetUnusedConnections() []int {
	used := make(map[int]bool)
	for _, idx := range pr.UsedConnections {
		used[idx] = true
	}

	var unused []int
	for i := range pr.Room.Connections {
		if !used[i] {
			unused = append(unused, i)
		}
	}
	return unused
}

// GetWorldConnectionPoint returns the world coordinates of a connection point
func (pr *PlacedRoom) GetWorldConnectionPoint(connIdx int) (x, y int, direction string) {
	if connIdx < 0 || connIdx >= len(pr.Room.Connections) {
		return 0, 0, ""
	}
	conn := pr.Room.Connections[connIdx]
	return pr.X + conn.X, pr.Y + conn.Y, conn.Direction
}

// GeneratedLevel represents a procedurally generated level
type GeneratedLevel struct {
	Name              string                         // Level name
	Width             int                            // Total level width in tiles
	Height            int                            // Total level height in tiles
	TileSize          int                            // Tile size in pixels
	AtlasPath         string                         // Path to atlas
	FloorTile         string                         // Default floor tile
	Tiles             [][]string                     // Generated tile grid [y][x]
	PlacedRooms       []*PlacedRoom                  // All placed rooms
	Corridors         []*Corridor                    // Generated corridors
	PlacedFurnishings []*furnishing.PlacedFurnishing // All placed furnishings
	PlayerSpawn       PlayerSpawn                    // Player starting position
}

// PlayerSpawn represents the player's starting position
type PlayerSpawn struct {
	X int // X position in pixels
	Y int // Y position in pixels
}

// GeneratorConfig holds configuration for level generation
type GeneratorConfig struct {
	MinRooms     int   // Minimum number of rooms to generate
	MaxRooms     int   // Maximum number of rooms to generate
	LevelWidth   int   // Target level width in tiles (0 = auto)
	LevelHeight  int   // Target level height in tiles (0 = auto)
	Seed         int64 // Random seed (0 = use current time)
	ConnectAll   bool  // Ensure all rooms are connected
	AllowOverlap bool  // Allow rooms to overlap (not recommended)
}

// Generator handles procedural level generation
type Generator struct {
	library           *RoomLibrary
	furnishingLibrary *furnishing.FurnishingLibrary
	config            GeneratorConfig
	rng               *rand.Rand
}

// NewGenerator creates a new level generator
func NewGenerator(library *RoomLibrary, config GeneratorConfig) *Generator {
	seed := config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &Generator{
		library:           library,
		furnishingLibrary: nil, // Can be set later with SetFurnishingLibrary
		config:            config,
		rng:               rand.New(rand.NewSource(seed)),
	}
}

// SetFurnishingLibrary sets the furnishing library for the generator
func (g *Generator) SetFurnishingLibrary(furnishingLibrary *furnishing.FurnishingLibrary) {
	g.furnishingLibrary = furnishingLibrary
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

	// Place rooms in the level using connection-based placement
	placedRooms, corridors, err := g.placeRoomsConnected(roomsToPlace)
	if err != nil {
		return nil, err
	}

	// Calculate level bounds with padding for border walls
	levelWidth, levelHeight := g.calculateBoundsWithPadding(placedRooms, corridors)

	// Create tile grid with rooms, corridors, and border walls
	tiles := g.createTileGridWithCorridors(levelWidth, levelHeight, placedRooms, corridors)

	// Find player spawn
	playerSpawn := g.findPlayerSpawn(placedRooms)

	// Place furnishings
	placedFurnishings := g.placeFurnishings(placedRooms)

	level := &GeneratedLevel{
		Name:              "Procedurally Generated Outpost",
		Width:             levelWidth,
		Height:            levelHeight,
		TileSize:          g.library.TileSize,
		AtlasPath:         g.library.AtlasPath,
		FloorTile:         g.library.FloorTile,
		Tiles:             tiles,
		PlacedRooms:       placedRooms,
		Corridors:         corridors,
		PlacedFurnishings: placedFurnishings,
		PlayerSpawn:       playerSpawn,
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

// getOppositeDirection returns the opposite direction
func getOppositeDirection(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	}
	return ""
}

// placeRoomsConnected places rooms using connection-based placement
func (g *Generator) placeRoomsConnected(rooms []*RoomDefinition) ([]*PlacedRoom, []*Corridor, error) {
	if len(rooms) == 0 {
		return nil, nil, fmt.Errorf("no rooms to place")
	}

	var placed []*PlacedRoom
	var corridors []*Corridor

	// Track occupied tiles to prevent overlap
	occupied := make(map[string]bool)

	// Place the first room (entrance) at origin with some padding
	firstRoom := &PlacedRoom{
		Room:            rooms[0],
		X:               2, // Start with padding for border
		Y:               2,
		ID:              0,
		Connected:       true,
		UsedConnections: []int{},
	}
	placed = append(placed, firstRoom)
	g.markOccupied(occupied, firstRoom)

	// Place remaining rooms by connecting to existing rooms
	for i := 1; i < len(rooms); i++ {
		roomDef := rooms[i]
		placedRoom, corridor := g.tryPlaceRoom(roomDef, i, placed, occupied)

		if placedRoom != nil {
			placed = append(placed, placedRoom)
			g.markOccupied(occupied, placedRoom)
			if corridor != nil {
				corridors = append(corridors, corridor)
				g.markCorridorOccupied(occupied, corridor)
			}
		} else {
			// Couldn't connect this room, try placing it with a corridor from any available connection
			placedRoom, corridor = g.forcePlace(roomDef, i, placed, occupied)
			if placedRoom != nil {
				placed = append(placed, placedRoom)
				g.markOccupied(occupied, placedRoom)
				if corridor != nil {
					corridors = append(corridors, corridor)
					g.markCorridorOccupied(occupied, corridor)
				}
			}
		}
	}

	return placed, corridors, nil
}

// markOccupied marks all tiles of a room as occupied
func (g *Generator) markOccupied(occupied map[string]bool, room *PlacedRoom) {
	for dy := 0; dy < room.Room.Height; dy++ {
		for dx := 0; dx < room.Room.Width; dx++ {
			key := fmt.Sprintf("%d,%d", room.X+dx, room.Y+dy)
			occupied[key] = true
		}
	}
}

// markCorridorOccupied marks corridor tiles as occupied
func (g *Generator) markCorridorOccupied(occupied map[string]bool, corridor *Corridor) {
	for _, tile := range corridor.Tiles {
		key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
		occupied[key] = true
	}
}

// canPlaceRoom checks if a room can be placed at the given position without overlap
func (g *Generator) canPlaceRoom(roomDef *RoomDefinition, x, y int, occupied map[string]bool) bool {
	for dy := 0; dy < roomDef.Height; dy++ {
		for dx := 0; dx < roomDef.Width; dx++ {
			key := fmt.Sprintf("%d,%d", x+dx, y+dy)
			if occupied[key] {
				return false
			}
		}
	}
	return true
}

// tryPlaceRoom attempts to place a room connected to an existing room
func (g *Generator) tryPlaceRoom(roomDef *RoomDefinition, id int, placed []*PlacedRoom, occupied map[string]bool) (*PlacedRoom, *Corridor) {
	// Shuffle placed rooms to get variety
	shuffledPlaced := make([]*PlacedRoom, len(placed))
	copy(shuffledPlaced, placed)
	g.rng.Shuffle(len(shuffledPlaced), func(i, j int) {
		shuffledPlaced[i], shuffledPlaced[j] = shuffledPlaced[j], shuffledPlaced[i]
	})

	// Try to connect to each placed room
	for _, existingRoom := range shuffledPlaced {
		unusedConns := existingRoom.GetUnusedConnections()
		if len(unusedConns) == 0 {
			continue
		}

		// Shuffle unused connections
		g.rng.Shuffle(len(unusedConns), func(i, j int) {
			unusedConns[i], unusedConns[j] = unusedConns[j], unusedConns[i]
		})

		for _, connIdx := range unusedConns {
			connX, connY, connDir := existingRoom.GetWorldConnectionPoint(connIdx)
			oppositeDir := getOppositeDirection(connDir)

			// Find a compatible connection in the new room
			for newConnIdx, newConn := range roomDef.Connections {
				if newConn.Direction != oppositeDir {
					continue
				}

				// Calculate where to place the new room so connections align
				// The new room's connection point should be adjacent to the existing room's connection
				var newRoomX, newRoomY int
				switch connDir {
				case "east":
					newRoomX = connX + 1 - newConn.X
					newRoomY = connY - newConn.Y
				case "west":
					newRoomX = connX - 1 - newConn.X + 1 - roomDef.Width + newConn.X
					newRoomY = connY - newConn.Y
				case "south":
					newRoomX = connX - newConn.X
					newRoomY = connY + 1 - newConn.Y
				case "north":
					newRoomX = connX - newConn.X
					newRoomY = connY - 1 - newConn.Y + 1 - roomDef.Height + newConn.Y
				}

				// Check if we can place the room here
				if g.canPlaceRoom(roomDef, newRoomX, newRoomY, occupied) {
					newRoom := &PlacedRoom{
						Room:            roomDef,
						X:               newRoomX,
						Y:               newRoomY,
						ID:              id,
						Connected:       true,
						UsedConnections: []int{newConnIdx},
					}
					existingRoom.UsedConnections = append(existingRoom.UsedConnections, connIdx)
					return newRoom, nil
				}
			}
		}
	}

	return nil, nil
}

// forcePlace places a room with a corridor when direct connection isn't possible
func (g *Generator) forcePlace(roomDef *RoomDefinition, id int, placed []*PlacedRoom, occupied map[string]bool) (*PlacedRoom, *Corridor) {
	// Try to find a spot and connect with a corridor
	for _, existingRoom := range placed {
		unusedConns := existingRoom.GetUnusedConnections()
		if len(unusedConns) == 0 {
			continue
		}

		for _, connIdx := range unusedConns {
			connX, connY, connDir := existingRoom.GetWorldConnectionPoint(connIdx)
			oppositeDir := getOppositeDirection(connDir)

			// Look for compatible connections in the new room
			for newConnIdx, newConn := range roomDef.Connections {
				if newConn.Direction != oppositeDir {
					continue
				}

				// Try placing room at various distances with a corridor
				for dist := 3; dist <= 8; dist++ {
					var newRoomX, newRoomY int
					switch connDir {
					case "east":
						newRoomX = connX + dist - newConn.X
						newRoomY = connY - newConn.Y
					case "west":
						newRoomX = connX - dist - roomDef.Width + 1 + newConn.X - newConn.X
						newRoomY = connY - newConn.Y
					case "south":
						newRoomX = connX - newConn.X
						newRoomY = connY + dist - newConn.Y
					case "north":
						newRoomX = connX - newConn.X
						newRoomY = connY - dist - roomDef.Height + 1 + newConn.Y - newConn.Y
					}

					if newRoomX < 1 || newRoomY < 1 {
						continue
					}

					if g.canPlaceRoom(roomDef, newRoomX, newRoomY, occupied) {
						newRoom := &PlacedRoom{
							Room:            roomDef,
							X:               newRoomX,
							Y:               newRoomY,
							ID:              id,
							Connected:       true,
							UsedConnections: []int{newConnIdx},
						}

						// Generate corridor
						newConnX := newRoomX + newConn.X
						newConnY := newRoomY + newConn.Y
						corridor := g.generateCorridor(connX, connY, connDir, newConnX, newConnY, oppositeDir, occupied)

						if corridor != nil {
							existingRoom.UsedConnections = append(existingRoom.UsedConnections, connIdx)
							return newRoom, corridor
						}
					}
				}
			}
		}
	}

	// Last resort: place room somewhere and try to connect
	for _, existingRoom := range placed {
		unusedConns := existingRoom.GetUnusedConnections()
		if len(unusedConns) == 0 {
			continue
		}

		connIdx := unusedConns[0]
		connX, connY, connDir := existingRoom.GetWorldConnectionPoint(connIdx)

		// Try placing the new room in the general direction
		for dist := 5; dist <= 15; dist++ {
			for offset := -5; offset <= 5; offset++ {
				var newRoomX, newRoomY int
				switch connDir {
				case "east":
					newRoomX = connX + dist
					newRoomY = connY + offset
				case "west":
					newRoomX = connX - dist - roomDef.Width
					newRoomY = connY + offset
				case "south":
					newRoomX = connX + offset
					newRoomY = connY + dist
				case "north":
					newRoomX = connX + offset
					newRoomY = connY - dist - roomDef.Height
				}

				if newRoomX < 1 || newRoomY < 1 {
					continue
				}

				if g.canPlaceRoom(roomDef, newRoomX, newRoomY, occupied) {
					newRoom := &PlacedRoom{
						Room:            roomDef,
						X:               newRoomX,
						Y:               newRoomY,
						ID:              id,
						Connected:       true,
						UsedConnections: []int{},
					}

					// Find best connection point on new room
					var bestNewConnIdx int
					var bestNewConnX, bestNewConnY int
					bestDist := math.MaxFloat64

					for idx, conn := range roomDef.Connections {
						cx := newRoomX + conn.X
						cy := newRoomY + conn.Y
						d := math.Sqrt(float64((cx-connX)*(cx-connX) + (cy-connY)*(cy-connY)))
						if d < bestDist {
							bestDist = d
							bestNewConnIdx = idx
							bestNewConnX = cx
							bestNewConnY = cy
						}
					}

					if len(roomDef.Connections) > 0 {
						newRoom.UsedConnections = []int{bestNewConnIdx}
						oppositeDir := getOppositeDirection(connDir)
						corridor := g.generateCorridor(connX, connY, connDir, bestNewConnX, bestNewConnY, oppositeDir, occupied)
						if corridor != nil {
							existingRoom.UsedConnections = append(existingRoom.UsedConnections, connIdx)
							return newRoom, corridor
						}
					}
				}
			}
		}
	}

	return nil, nil
}

// generateCorridor generates a corridor between two connection points
func (g *Generator) generateCorridor(x1, y1 int, dir1 string, x2, y2 int, dir2 string, occupied map[string]bool) *Corridor {
	corridor := &Corridor{Tiles: []CorridorTile{}}

	// Determine the exit points from each room
	var startX, startY, endX, endY int
	switch dir1 {
	case "east":
		startX, startY = x1+1, y1
	case "west":
		startX, startY = x1-1, y1
	case "south":
		startX, startY = x1, y1+1
	case "north":
		startX, startY = x1, y1-1
	default:
		startX, startY = x1, y1
	}

	switch dir2 {
	case "east":
		endX, endY = x2+1, y2
	case "west":
		endX, endY = x2-1, y2
	case "south":
		endX, endY = x2, y2+1
	case "north":
		endX, endY = x2, y2-1
	default:
		endX, endY = x2, y2
	}

	// Generate L-shaped or straight corridor
	// First go horizontal, then vertical (or vice versa)
	currentX, currentY := startX, startY

	// Add floor tiles along the path
	// Go horizontal first
	dx := 0
	if endX > currentX {
		dx = 1
	} else if endX < currentX {
		dx = -1
	}

	for currentX != endX {
		g.addCorridorSection(corridor, currentX, currentY)
		currentX += dx
	}

	// Then go vertical
	dy := 0
	if endY > currentY {
		dy = 1
	} else if endY < currentY {
		dy = -1
	}

	for currentY != endY {
		g.addCorridorSection(corridor, currentX, currentY)
		currentY += dy
	}

	// Add final section
	g.addCorridorSection(corridor, currentX, currentY)

	return corridor
}

// addCorridorSection adds a 3-wide corridor section (floor with walls on sides)
func (g *Generator) addCorridorSection(corridor *Corridor, x, y int) {
	// Add floor tile
	corridor.Tiles = append(corridor.Tiles, CorridorTile{X: x, Y: y, IsFloor: true})
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

// calculateBoundsWithPadding determines level size with extra padding for border walls
func (g *Generator) calculateBoundsWithPadding(rooms []*PlacedRoom, corridors []*Corridor) (width, height int) {
	if len(rooms) == 0 {
		return 10, 10 // Minimum size
	}

	minX, minY := math.MaxInt32, math.MaxInt32
	maxX, maxY := 0, 0

	// Check rooms
	for _, room := range rooms {
		if room.X < minX {
			minX = room.X
		}
		if room.Y < minY {
			minY = room.Y
		}
		rightEdge := room.X + room.Room.Width
		bottomEdge := room.Y + room.Room.Height
		if rightEdge > maxX {
			maxX = rightEdge
		}
		if bottomEdge > maxY {
			maxY = bottomEdge
		}
	}

	// Check corridors
	for _, corridor := range corridors {
		for _, tile := range corridor.Tiles {
			if tile.X < minX {
				minX = tile.X
			}
			if tile.Y < minY {
				minY = tile.Y
			}
			if tile.X+1 > maxX {
				maxX = tile.X + 1
			}
			if tile.Y+1 > maxY {
				maxY = tile.Y + 1
			}
		}
	}

	// Add padding for border walls (2 tiles on each side)
	width = maxX + 2
	height = maxY + 2

	if g.config.LevelWidth > 0 && g.config.LevelWidth > width {
		width = g.config.LevelWidth
	}
	if g.config.LevelHeight > 0 && g.config.LevelHeight > height {
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

// createTileGridWithCorridors generates the tile grid including corridors and border walls
func (g *Generator) createTileGridWithCorridors(width, height int, rooms []*PlacedRoom, corridors []*Corridor) [][]string {
	// Initialize grid with empty tiles (void - will be black/not rendered)
	tiles := make([][]string, height)
	for y := 0; y < height; y++ {
		tiles[y] = make([]string, width)
		for x := 0; x < width; x++ {
			tiles[y][x] = "" // Empty = void (black, not rendered)
		}
	}

	// Place room tiles (rooms already have their walls defined in the tile data)
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

	// Place corridor floor tiles and add walls around them
	corridorFloors := make(map[string]bool)
	for _, corridor := range corridors {
		for _, tile := range corridor.Tiles {
			if tile.IsFloor {
				key := fmt.Sprintf("%d,%d", tile.X, tile.Y)
				corridorFloors[key] = true
			}
		}
	}

	// First pass: place corridor floors
	for _, corridor := range corridors {
		for _, tile := range corridor.Tiles {
			if tile.X >= 0 && tile.X < width && tile.Y >= 0 && tile.Y < height {
				if tile.IsFloor {
					// Only place floor if the tile is currently empty (don't overwrite room tiles)
					if tiles[tile.Y][tile.X] == "" {
						tiles[tile.Y][tile.X] = "floor"
					}
				}
			}
		}
	}

	// Second pass: add walls around corridor floors
	for _, corridor := range corridors {
		for _, tile := range corridor.Tiles {
			if !tile.IsFloor {
				continue
			}
			// Check all 8 neighbors
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := tile.X+dx, tile.Y+dy
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						// Add wall if the neighbor is empty void
						if tiles[ny][nx] == "" {
							tiles[ny][nx] = "wall"
						}
					}
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

// placeFurnishings places all furnishings defined in placed rooms
func (g *Generator) placeFurnishings(rooms []*PlacedRoom) []*furnishing.PlacedFurnishing {
	var placed []*furnishing.PlacedFurnishing

	// If no furnishing library is set, return empty list
	if g.furnishingLibrary == nil {
		return placed
	}

	for _, placedRoom := range rooms {
		room := placedRoom.Room

		// Process each furnishing placement in the room definition
		for _, furnishingPlacement := range room.Furnishings {
			// Look up the furnishing definition
			furnishingDef := g.furnishingLibrary.GetFurnishingByName(furnishingPlacement.FurnishingName)
			if furnishingDef == nil {
				// Skip if furnishing not found
				continue
			}

			// Calculate world position
			worldX := placedRoom.X + furnishingPlacement.X
			worldY := placedRoom.Y + furnishingPlacement.Y

			// Create placed furnishing instance
			placedFurnishing := &furnishing.PlacedFurnishing{
				Definition: furnishingDef,
				X:          worldX,
				Y:          worldY,
				RoomID:     placedRoom.ID,
				State:      furnishingPlacement.State,
			}

			// If state is empty, use default
			if placedFurnishing.State == "" {
				placedFurnishing.State = "default"
			}

			placed = append(placed, placedFurnishing)
		}
	}

	return placed
}
