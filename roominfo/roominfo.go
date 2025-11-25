// Package roominfo provides runtime tracking of room states and player location
package roominfo

import (
	"chosenoffset.com/outpost9/entity"
	"chosenoffset.com/outpost9/room"
)

// RoomEventType represents different room-related events
type RoomEventType int

const (
	RoomEntered RoomEventType = iota
	RoomExited
	RoomSearched
	SecretDiscovered
	RoomCleared // All enemies defeated
)

// RoomEvent represents a room-related event that occurred
type RoomEvent struct {
	Type     RoomEventType
	Room     *room.PlacedRoom
	RoomID   int
	IsFirst  bool   // First time entering this room
	Details  string // Additional context
	Revealed string // Text that was revealed (for discoveries)
}

// RoomState tracks the runtime state of a single placed room
type RoomState struct {
	RoomID          int               // ID of the PlacedRoom
	Visited         bool              // Has the player entered this room
	VisitCount      int               // Number of times visited
	Searched        bool              // Has the player searched this room
	RevealedSecrets map[string]bool   // Tags of secrets that have been revealed
	EnemiesCleared  bool              // Were all enemies in this room defeated
	CustomFlags     map[string]string // Custom room-specific flags
}

// NewRoomState creates a new room state for a placed room
func NewRoomState(roomID int) *RoomState {
	return &RoomState{
		RoomID:          roomID,
		Visited:         false,
		VisitCount:      0,
		Searched:        false,
		RevealedSecrets: make(map[string]bool),
		EnemiesCleared:  false,
		CustomFlags:     make(map[string]string),
	}
}

// MarkVisited marks the room as visited
func (rs *RoomState) MarkVisited() bool {
	wasFirst := !rs.Visited
	rs.Visited = true
	rs.VisitCount++
	return wasFirst
}

// MarkSearched marks the room as searched
func (rs *RoomState) MarkSearched() bool {
	wasFirst := !rs.Searched
	rs.Searched = true
	return wasFirst
}

// RevealSecret marks a secret as revealed
func (rs *RoomState) RevealSecret(tag string) bool {
	if rs.RevealedSecrets[tag] {
		return false // Already revealed
	}
	rs.RevealedSecrets[tag] = true
	return true
}

// IsSecretRevealed checks if a secret has been revealed
func (rs *RoomState) IsSecretRevealed(tag string) bool {
	return rs.RevealedSecrets[tag]
}

// RoomTracker manages player location and room states
type RoomTracker struct {
	level       *room.GeneratedLevel
	roomStates  map[int]*RoomState // Map of room ID to state
	currentRoom *room.PlacedRoom   // Room the player is currently in
	lastRoom    *room.PlacedRoom   // Previous room (for transition detection)

	// Callbacks for room events
	OnRoomEvent func(event RoomEvent)
}

// NewRoomTracker creates a new room tracker for a generated level
func NewRoomTracker(level *room.GeneratedLevel) *RoomTracker {
	tracker := &RoomTracker{
		level:      level,
		roomStates: make(map[int]*RoomState),
	}

	// Initialize room states for all placed rooms
	for _, placedRoom := range level.PlacedRooms {
		tracker.roomStates[placedRoom.ID] = NewRoomState(placedRoom.ID)
	}

	return tracker
}

// GetRoomAt returns the placed room at the given tile coordinates
func (rt *RoomTracker) GetRoomAt(x, y int) *room.PlacedRoom {
	for _, placedRoom := range rt.level.PlacedRooms {
		// Check if coordinates are within this room's bounds
		if x >= placedRoom.X && x < placedRoom.X+placedRoom.Room.Width &&
			y >= placedRoom.Y && y < placedRoom.Y+placedRoom.Room.Height {
			return placedRoom
		}
	}
	return nil // Player is in a corridor or undefined area
}

// GetRoomState returns the state for a room ID
func (rt *RoomTracker) GetRoomState(roomID int) *RoomState {
	return rt.roomStates[roomID]
}

// GetCurrentRoom returns the room the player is currently in
func (rt *RoomTracker) GetCurrentRoom() *room.PlacedRoom {
	return rt.currentRoom
}

// GetCurrentRoomState returns the state of the current room
func (rt *RoomTracker) GetCurrentRoomState() *RoomState {
	if rt.currentRoom == nil {
		return nil
	}
	return rt.roomStates[rt.currentRoom.ID]
}

// UpdatePlayerPosition updates tracking based on player position
// Returns a RoomEvent if a room transition occurred
func (rt *RoomTracker) UpdatePlayerPosition(x, y int) *RoomEvent {
	newRoom := rt.GetRoomAt(x, y)

	// Check if we've changed rooms
	if newRoom != rt.currentRoom {
		rt.lastRoom = rt.currentRoom
		rt.currentRoom = newRoom

		if newRoom != nil {
			// Player entered a room
			state := rt.roomStates[newRoom.ID]
			isFirst := state.MarkVisited()

			event := &RoomEvent{
				Type:    RoomEntered,
				Room:    newRoom,
				RoomID:  newRoom.ID,
				IsFirst: isFirst,
			}

			if rt.OnRoomEvent != nil {
				rt.OnRoomEvent(*event)
			}

			return event
		} else if rt.lastRoom != nil {
			// Player exited a room (into corridor)
			event := &RoomEvent{
				Type:   RoomExited,
				Room:   rt.lastRoom,
				RoomID: rt.lastRoom.ID,
			}

			if rt.OnRoomEvent != nil {
				rt.OnRoomEvent(*event)
			}

			return event
		}
	}

	return nil
}

// SearchCurrentRoom performs a search action in the current room
// Returns any revealed text and whether the search was successful
func (rt *RoomTracker) SearchCurrentRoom(player *entity.Entity) (string, bool) {
	if rt.currentRoom == nil {
		return "There's nothing particular to search in this corridor.", false
	}

	state := rt.roomStates[rt.currentRoom.ID]
	narrative := rt.currentRoom.Room.Narrative

	if narrative == nil {
		return "You search the area but find nothing of interest.", false
	}

	var result string
	foundSomething := false

	// First time search reveals search text
	if !state.Searched && narrative.SearchText != "" {
		result = narrative.SearchText
		foundSomething = true
		state.MarkSearched()
	}

	// Check skill reveals
	for _, reveal := range narrative.SkillReveals {
		// Skip if already revealed (for one-time reveals)
		if reveal.OneTime && state.IsSecretRevealed(reveal.Tag) {
			continue
		}

		// Check if player has the skill and can pass the check
		// TODO: Integrate with actual skill check system
		skillLevel := player.GetSkill(reveal.Skill)
		if skillLevel > 0 {
			// Simple check: skill level >= difficulty (can be made more complex)
			if skillLevel >= reveal.Difficulty {
				if state.RevealSecret(reveal.Tag) {
					if result != "" {
						result += " "
					}
					result += reveal.Text
					foundSomething = true

					// Fire discovery event
					if rt.OnRoomEvent != nil {
						rt.OnRoomEvent(RoomEvent{
							Type:     SecretDiscovered,
							Room:     rt.currentRoom,
							RoomID:   rt.currentRoom.ID,
							Revealed: reveal.Text,
							Details:  reveal.Skill,
						})
					}
				}
			}
		}
	}

	if !foundSomething {
		if state.Searched {
			return "You've already thoroughly searched this room.", false
		}
		return "You search carefully but find nothing new.", false
	}

	return result, true
}

// GetRoomDescription returns the appropriate description for the current room
func (rt *RoomTracker) GetRoomDescription(hasEnemies bool) string {
	if rt.currentRoom == nil {
		return "You are in a narrow corridor."
	}

	narrative := rt.currentRoom.Room.Narrative
	state := rt.roomStates[rt.currentRoom.ID]

	// Build description
	var description string

	// Entry text based on visit status
	if narrative != nil {
		if state.VisitCount == 1 && narrative.EntryText != "" {
			description = narrative.EntryText
		} else if narrative.ReturnText != "" {
			description = narrative.ReturnText
		}

		// Add atmosphere if available
		if narrative.Atmosphere != "" {
			if description != "" {
				description += " "
			}
			description += narrative.Atmosphere
		}

		// Add danger hint if enemies present
		if hasEnemies && narrative.DangerHint != "" {
			if description != "" {
				description += " "
			}
			description += narrative.DangerHint
		}

		// Add safe text if room was cleared
		if state.EnemiesCleared && !hasEnemies && narrative.SafeText != "" {
			if description != "" {
				description += " "
			}
			description += narrative.SafeText
		}
	}

	// Fallback to basic description
	if description == "" {
		description = rt.currentRoom.Room.Description
	}
	if description == "" {
		description = "You are in " + rt.currentRoom.Room.Name + "."
	}

	return description
}

// MarkRoomCleared marks the current room as cleared of enemies
func (rt *RoomTracker) MarkRoomCleared() {
	if rt.currentRoom == nil {
		return
	}

	state := rt.roomStates[rt.currentRoom.ID]
	if !state.EnemiesCleared {
		state.EnemiesCleared = true

		if rt.OnRoomEvent != nil {
			rt.OnRoomEvent(RoomEvent{
				Type:   RoomCleared,
				Room:   rt.currentRoom,
				RoomID: rt.currentRoom.ID,
			})
		}
	}
}

// GetRoomName returns the display name for the current room
func (rt *RoomTracker) GetRoomName() string {
	if rt.currentRoom == nil {
		return "a corridor"
	}

	// Format the room name for display
	name := rt.currentRoom.Room.Name
	if name == "" {
		return "an unknown room"
	}

	// Make it more readable (e.g., "chamber_small" -> "a small chamber")
	return formatRoomName(name, rt.currentRoom.Room.Type)
}

// formatRoomName converts internal room names to readable text
func formatRoomName(name, roomType string) string {
	// Use the type for a more natural description
	switch roomType {
	case "entrance":
		return "the entrance"
	case "corridor":
		return "a corridor"
	case "chamber":
		return "a chamber"
	case "storage":
		return "a storage room"
	default:
		return "a room"
	}
}

// IsInRoom returns whether the player is currently in a room (not corridor)
func (rt *RoomTracker) IsInRoom() bool {
	return rt.currentRoom != nil
}

// GetAllVisitedRooms returns all rooms that have been visited
func (rt *RoomTracker) GetAllVisitedRooms() []*room.PlacedRoom {
	var visited []*room.PlacedRoom
	for _, placedRoom := range rt.level.PlacedRooms {
		state := rt.roomStates[placedRoom.ID]
		if state != nil && state.Visited {
			visited = append(visited, placedRoom)
		}
	}
	return visited
}

// GetExplorationProgress returns the percentage of rooms visited
func (rt *RoomTracker) GetExplorationProgress() float64 {
	if len(rt.level.PlacedRooms) == 0 {
		return 0
	}

	visited := 0
	for _, state := range rt.roomStates {
		if state.Visited {
			visited++
		}
	}

	return float64(visited) / float64(len(rt.level.PlacedRooms)) * 100
}
