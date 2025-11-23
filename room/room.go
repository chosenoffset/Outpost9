package room

import (
	"encoding/json"
	"fmt"
	"os"

	"chosenoffset.com/outpost9/furnishing"
)

// ConnectionPoint represents a door or exit point in a room
type ConnectionPoint struct {
	X         int    `json:"x"`         // Grid X position
	Y         int    `json:"y"`         // Grid Y position
	Direction string `json:"direction"` // "north", "south", "east", "west"
	Type      string `json:"type"`      // "door", "corridor", "entrance", "exit"
}

// SkillReveal represents information revealed by a skill check
type SkillReveal struct {
	Skill      string `json:"skill"`       // Skill required (e.g., "perception", "investigation")
	Difficulty int    `json:"difficulty"`  // DC for the check
	Text       string `json:"text"`        // Text revealed on success
	OneTime    bool   `json:"one_time"`    // Only reveal once per game
	Tag        string `json:"tag"`         // Identifier for tracking if revealed
}

// RoomNarrative contains all narrative text for a room
type RoomNarrative struct {
	EntryText    string        `json:"entry_text"`    // Text shown on first entry
	ReturnText   string        `json:"return_text"`   // Text shown when returning to room
	SearchText   string        `json:"search_text"`   // Text revealed when actively searching
	Atmosphere   string        `json:"atmosphere"`    // Ambient description (sounds, smells, mood)
	SkillReveals []SkillReveal `json:"skill_reveals"` // Skill-based discoveries
	DangerHint   string        `json:"danger_hint"`   // Hint about dangers (shown if enemies present)
	SafeText     string        `json:"safe_text"`     // Text when room is cleared of enemies
}

// RoomDefinition represents a single room template that can be used in level generation
type RoomDefinition struct {
	Name        string                               `json:"name"`         // Room identifier
	Description string                               `json:"description"`  // Optional description
	Type        string                               `json:"type"`         // "entrance", "corridor", "chamber", "exit", etc.
	Tags        []string                             `json:"tags"`         // Additional categorization
	Width       int                                  `json:"width"`        // Room width in tiles
	Height      int                                  `json:"height"`       // Room height in tiles
	Tiles       [][]string                           `json:"tiles"`        // 2D array of tile names [y][x]
	Connections []ConnectionPoint                    `json:"connections"`  // Available connection points
	Furnishings []furnishing.RoomFurnishingPlacement `json:"furnishings"`  // Furnishings to place in this room
	SpawnWeight int                                  `json:"spawn_weight"` // Weight for random selection (higher = more likely)
	MinCount    int                                  `json:"min_count"`    // Minimum number of times this room should appear
	MaxCount    int                                  `json:"max_count"`    // Maximum number of times this room can appear (0 = unlimited)
	Narrative   *RoomNarrative                       `json:"narrative"`    // Narrative content for this room
}

// RoomLibrary holds a collection of room definitions
type RoomLibrary struct {
	Name        string            `json:"name"`        // Library name
	Description string            `json:"description"` // Optional description
	AtlasPath   string            `json:"atlas_path"`  // Path to atlas.json used by these rooms
	TileSize    int               `json:"tile_size"`   // Size of tiles in pixels
	FloorTile   string            `json:"floor_tile"`  // Default floor tile
	Rooms       []*RoomDefinition `json:"rooms"`       // All room definitions
}

// Validate checks if a room definition is valid
func (r *RoomDefinition) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("room name is required")
	}
	if r.Width <= 0 || r.Height <= 0 {
		return fmt.Errorf("room %s: width and height must be positive", r.Name)
	}
	if len(r.Tiles) != r.Height {
		return fmt.Errorf("room %s: tiles array height (%d) doesn't match height (%d)", r.Name, len(r.Tiles), r.Height)
	}
	for y, row := range r.Tiles {
		if len(row) != r.Width {
			return fmt.Errorf("room %s: row %d width (%d) doesn't match width (%d)", r.Name, y, len(row), r.Width)
		}
	}
	for i, conn := range r.Connections {
		if conn.X < 0 || conn.X >= r.Width || conn.Y < 0 || conn.Y >= r.Height {
			return fmt.Errorf("room %s: connection %d at (%d,%d) is out of bounds", r.Name, i, conn.X, conn.Y)
		}
		if conn.Direction != "north" && conn.Direction != "south" && conn.Direction != "east" && conn.Direction != "west" {
			return fmt.Errorf("room %s: connection %d has invalid direction '%s'", r.Name, i, conn.Direction)
		}
	}
	return nil
}

// GetConnectionsByDirection returns all connections in a specific direction
func (r *RoomDefinition) GetConnectionsByDirection(direction string) []ConnectionPoint {
	var result []ConnectionPoint
	for _, conn := range r.Connections {
		if conn.Direction == direction {
			result = append(result, conn)
		}
	}
	return result
}

// GetConnectionsByType returns all connections of a specific type
func (r *RoomDefinition) GetConnectionsByType(connType string) []ConnectionPoint {
	var result []ConnectionPoint
	for _, conn := range r.Connections {
		if conn.Type == connType {
			result = append(result, conn)
		}
	}
	return result
}

// HasTag checks if the room has a specific tag
func (r *RoomDefinition) HasTag(tag string) bool {
	for _, t := range r.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// LoadRoomLibrary loads a room library from a JSON file
func LoadRoomLibrary(path string) (*RoomLibrary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read room library file: %w", err)
	}

	var library RoomLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		return nil, fmt.Errorf("failed to parse room library JSON: %w", err)
	}

	// Validate all rooms
	for _, room := range library.Rooms {
		if err := room.Validate(); err != nil {
			return nil, err
		}
	}

	return &library, nil
}

// GetRoomsByType returns all rooms of a specific type
func (l *RoomLibrary) GetRoomsByType(roomType string) []*RoomDefinition {
	var result []*RoomDefinition
	for _, room := range l.Rooms {
		if room.Type == roomType {
			result = append(result, room)
		}
	}
	return result
}

// GetRoomsByTag returns all rooms with a specific tag
func (l *RoomLibrary) GetRoomsByTag(tag string) []*RoomDefinition {
	var result []*RoomDefinition
	for _, room := range l.Rooms {
		if room.HasTag(tag) {
			result = append(result, room)
		}
	}
	return result
}

// GetRoomByName finds a room by its name
func (l *RoomLibrary) GetRoomByName(name string) *RoomDefinition {
	for _, room := range l.Rooms {
		if room.Name == name {
			return room
		}
	}
	return nil
}
