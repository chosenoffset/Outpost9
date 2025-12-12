package room

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"chosenoffset.com/outpost9/internal/world/furnishing"
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

	// Validate connectivity and remove unreachable rooms
	placedRooms = g.removeUnreachableRooms(tiles, placedRooms, levelWidth, levelHeight)

	// Find player spawn
	playerSpawn := g.findPlayerSpawn(placedRooms)

	// Place furnishings (only for reachable rooms)
	placedFurnishings := g.placeFurnishings(placedRooms)

	level := &GeneratedLevel{
		Name:              "Procedurally Generated Dungeon",
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

	// Add rooms with min_count requirements (only if not already satisfied)
	for _, room := range g.library.Rooms {
		if room.MinCount > 0 {
			// Only add more if we haven't reached the minimum yet
			for usageCount[room.Name] < room.MinCount {
				if len(selected) < numRooms {
					selected = append(selected, room)
					usageCount[room.Name]++
				} else {
					break
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

	fmt.Printf("DEBUG: Placed entrance %s at (%d,%d), size %dx%d\n",
		rooms[0].Name, 2, 2, rooms[0].Width, rooms[0].Height)
	if len(rooms[0].Connections) > 0 {
		conn := rooms[0].Connections[0]
		fmt.Printf("DEBUG:   East door at local (%d,%d) = world (%d,%d)\n",
			conn.X, conn.Y, 2+conn.X, 2+conn.Y)
	}

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
// Also ensures the room is within valid coordinate bounds (no negative positions)
func (g *Generator) canPlaceRoom(roomDef *RoomDefinition, x, y int, occupied map[string]bool) bool {
	// Check for negative coordinates - rooms must be fully within valid bounds
	// This prevents tiles from being clipped when the level is generated
	if x < 0 || y < 0 {
		return false
	}

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

// tryPlaceRoom attempts to place a room connected to an existing room via corridor
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

				// Place room with a small corridor gap (2 tiles) to avoid double walls
				corridorLength := 2
				var newRoomX, newRoomY int
				switch connDir {
				case "east":
					newRoomX = connX + 1 + corridorLength - newConn.X
					newRoomY = connY - newConn.Y
				case "west":
					newRoomX = connX - 1 - corridorLength - newConn.X
					newRoomY = connY - newConn.Y
				case "south":
					newRoomX = connX - newConn.X
					newRoomY = connY + 1 + corridorLength - newConn.Y
				case "north":
					newRoomX = connX - newConn.X
					newRoomY = connY - 1 - corridorLength - newConn.Y
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

					// Generate corridor between the two doors
					newConnX := newRoomX + newConn.X
					newConnY := newRoomY + newConn.Y
					corridor := g.generateCorridor(connX, connY, connDir, newConnX, newConnY, oppositeDir, occupied)

					// Debug: Log the connection
					fmt.Printf("DEBUG: Placing %s at (%d,%d) connected to %s via %s door with corridor\n",
						roomDef.Name, newRoomX, newRoomY, existingRoom.Room.Name, connDir)
					fmt.Printf("DEBUG:   Existing door at world (%d,%d), new door at world (%d,%d)\n",
						connX, connY, newConnX, newConnY)

					return newRoom, corridor
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
						newRoomX = connX - dist - newConn.X
						newRoomY = connY - newConn.Y
					case "south":
						newRoomX = connX - newConn.X
						newRoomY = connY + dist - newConn.Y
					case "north":
						newRoomX = connX - newConn.X
						newRoomY = connY - dist - newConn.Y
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
		fmt.Printf("DEBUG: Writing tiles for %s at (%d,%d)\n", room.Name, placedRoom.X, placedRoom.Y)
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
		// Debug: Log door tiles
		for i, conn := range room.Connections {
			doorWorldX := placedRoom.X + conn.X
			doorWorldY := placedRoom.Y + conn.Y
			if doorWorldY >= 0 && doorWorldY < height && doorWorldX >= 0 && doorWorldX < width {
				fmt.Printf("DEBUG:   Connection %d (%s) at world (%d,%d) = tile '%s'\n",
					i, conn.Direction, doorWorldX, doorWorldY, tiles[doorWorldY][doorWorldX])
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

	// Third pass: close off unused doors (floor tiles adjacent to void)
	g.closeUnusedDoors(tiles, width, height)

	// Debug: Validate room north walls
	g.validateRoomWalls(tiles, rooms, width, height)

	return tiles
}

// validateRoomWalls checks if room walls were written correctly to the tile grid
func (g *Generator) validateRoomWalls(tiles [][]string, rooms []*PlacedRoom, width, height int) {
	for _, placedRoom := range rooms {
		room := placedRoom.Room

		// Check north wall (row 0)
		northWallMissing := true
		for rx := 0; rx < room.Width; rx++ {
			worldX := placedRoom.X + rx
			worldY := placedRoom.Y // Row 0 = north wall

			if worldY < 0 || worldY >= height || worldX < 0 || worldX >= width {
				continue
			}

			expectedTile := room.Tiles[0][rx]
			actualTile := tiles[worldY][worldX]

			if expectedTile == "wall" && actualTile == "wall" {
				northWallMissing = false
			}

			if expectedTile != actualTile {
				fmt.Printf("DEBUG MISMATCH: Room %s at (%d,%d): tile[%d][%d] expected '%s' but found '%s'\n",
					room.Name, placedRoom.X, placedRoom.Y, worldY, worldX, expectedTile, actualTile)
			}
		}

		if northWallMissing && room.Height > 2 {
			fmt.Printf("DEBUG WARNING: Room %s at (%d,%d) has NO walls in its north row!\n",
				room.Name, placedRoom.X, placedRoom.Y)
			// Print the north row for debugging
			fmt.Printf("  Expected north row: %v\n", room.Tiles[0])
			row := make([]string, room.Width)
			for rx := 0; rx < room.Width; rx++ {
				worldX := placedRoom.X + rx
				worldY := placedRoom.Y
				if worldY >= 0 && worldY < height && worldX >= 0 && worldX < width {
					row[rx] = tiles[worldY][worldX]
				} else {
					row[rx] = "OOB"
				}
			}
			fmt.Printf("  Actual north row:   %v\n", row)
		}
	}
}

// closeUnusedDoors finds floor tiles that are adjacent to void and converts them to walls
func (g *Generator) closeUnusedDoors(tiles [][]string, width, height int) {
	// Find all floor tiles that are on the edge of rooms (adjacent to void)
	// and don't have a corridor connecting them
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if tiles[y][x] != "floor" {
				continue
			}

			// Check if this floor tile is adjacent to void in any cardinal direction
			adjacentToVoid := false
			var voidDir string
			if x > 0 && tiles[y][x-1] == "" {
				adjacentToVoid = true
				voidDir = "west"
			}
			if x < width-1 && tiles[y][x+1] == "" {
				adjacentToVoid = true
				voidDir = "east"
			}
			if y > 0 && tiles[y-1][x] == "" {
				adjacentToVoid = true
				voidDir = "north"
			}
			if y < height-1 && tiles[y+1][x] == "" {
				adjacentToVoid = true
				voidDir = "south"
			}

			if adjacentToVoid {
				// This is an unused door - convert to wall
				tiles[y][x] = "wall"
				fmt.Printf("DEBUG: Closing unused door at (%d,%d) facing %s\n", x, y, voidDir)
			}
		}
	}
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

// removeUnreachableRooms uses flood fill to find rooms not connected to the entrance
// and removes them from the level (converts their tiles to void)
func (g *Generator) removeUnreachableRooms(tiles [][]string, rooms []*PlacedRoom, width, height int) []*PlacedRoom {
	// Find the entrance room to start flood fill
	var startX, startY int
	var entranceRoom *PlacedRoom
	for _, room := range rooms {
		if room.Room.Type == "entrance" {
			// Start from center of entrance room
			startX = room.X + room.Room.Width/2
			startY = room.Y + room.Room.Height/2
			entranceRoom = room
			break
		}
	}

	if entranceRoom == nil && len(rooms) > 0 {
		// Fallback to first room
		startX = rooms[0].X + rooms[0].Room.Width/2
		startY = rooms[0].Y + rooms[0].Room.Height/2
	}

	// Flood fill to find all reachable floor tiles
	reachable := make(map[string]bool)
	g.floodFill(tiles, startX, startY, width, height, reachable)

	fmt.Printf("DEBUG: Flood fill found %d reachable tiles starting from (%d,%d)\n", len(reachable), startX, startY)

	// Check each room for reachability
	var reachableRooms []*PlacedRoom
	for _, room := range rooms {
		// A room is reachable if any of its floor tiles are in the reachable set
		isReachable := false
		for ry := 0; ry < room.Room.Height && !isReachable; ry++ {
			for rx := 0; rx < room.Room.Width && !isReachable; rx++ {
				worldX := room.X + rx
				worldY := room.Y + ry

				// Check if this is a floor tile
				if worldY >= 0 && worldY < height && worldX >= 0 && worldX < width {
					if tiles[worldY][worldX] == "floor" {
						key := fmt.Sprintf("%d,%d", worldX, worldY)
						if reachable[key] {
							isReachable = true
						}
					}
				}
			}
		}

		if isReachable {
			reachableRooms = append(reachableRooms, room)
		} else {
			// Remove unreachable room tiles from the grid
			fmt.Printf("DEBUG: Removing unreachable room %s at (%d,%d)\n", room.Room.Name, room.X, room.Y)
			for ry := 0; ry < room.Room.Height; ry++ {
				for rx := 0; rx < room.Room.Width; rx++ {
					worldX := room.X + rx
					worldY := room.Y + ry
					if worldY >= 0 && worldY < height && worldX >= 0 && worldX < width {
						tiles[worldY][worldX] = "" // Convert to void
					}
				}
			}
		}
	}

	fmt.Printf("DEBUG: %d/%d rooms are reachable\n", len(reachableRooms), len(rooms))
	return reachableRooms
}

// floodFill performs a flood fill from the starting position to find all reachable floor tiles
func (g *Generator) floodFill(tiles [][]string, startX, startY, width, height int, reachable map[string]bool) {
	// Use BFS for flood fill
	type point struct{ x, y int }
	queue := []point{{startX, startY}}

	// Directions: up, down, left, right
	dirs := []point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Skip if out of bounds
		if current.x < 0 || current.x >= width || current.y < 0 || current.y >= height {
			continue
		}

		key := fmt.Sprintf("%d,%d", current.x, current.y)

		// Skip if already visited
		if reachable[key] {
			continue
		}

		// Skip if not a walkable tile (only floor tiles are walkable)
		tile := tiles[current.y][current.x]
		if tile != "floor" {
			continue
		}

		// Mark as reachable
		reachable[key] = true

		// Add neighbors to queue
		for _, dir := range dirs {
			next := point{current.x + dir.x, current.y + dir.y}
			nextKey := fmt.Sprintf("%d,%d", next.x, next.y)
			if !reachable[nextKey] {
				queue = append(queue, next)
			}
		}
	}
}

// placeFurnishings places all furnishings defined in placed rooms
func (g *Generator) placeFurnishings(rooms []*PlacedRoom) []*furnishing.PlacedFurnishing {
	var placed []*furnishing.PlacedFurnishing

	// If no furnishing library is set, return empty list
	if g.furnishingLibrary == nil {
		return placed
	}

	// Track furnishing counts per type for unique ID generation
	furnishingCounts := make(map[string]int)

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

			// Generate unique ID for this furnishing instance
			furnishingID := furnishingPlacement.ID
			if furnishingID == "" {
				// Auto-generate ID: furnishing_name_index (e.g., "chest_0", "chest_1")
				count := furnishingCounts[furnishingDef.Name]
				furnishingID = fmt.Sprintf("%s_%d", furnishingDef.Name, count)
				furnishingCounts[furnishingDef.Name] = count + 1
			}

			// Determine initial state
			initialState := furnishingPlacement.State
			if initialState == "" {
				// Use default state from definition if available
				if furnishingDef.DefaultState != "" {
					initialState = furnishingDef.DefaultState
				} else {
					initialState = "default"
				}
			}

			// Create placed furnishing instance
			placedFurnishing := &furnishing.PlacedFurnishing{
				Definition: furnishingDef,
				ID:         furnishingID,
				X:          worldX,
				Y:          worldY,
				RoomID:     placedRoom.ID,
				State:      initialState,
			}

			placed = append(placed, placedFurnishing)
		}
	}

	return placed
}
