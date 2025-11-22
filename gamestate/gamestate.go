// Package gamestate provides global game state management including flags, counters, and variables.
// This is used by the interaction system to track game progress and conditions.
package gamestate

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// GameState holds all persistent game state data
type GameState struct {
	mu sync.RWMutex

	// Flags are boolean values (e.g., "door_unlocked", "boss_defeated")
	Flags map[string]bool `json:"flags"`

	// Counters are integer values (e.g., "enemies_killed", "gold_collected")
	Counters map[string]int `json:"counters"`

	// Strings are string variables (e.g., "current_quest", "player_name")
	Strings map[string]string `json:"strings"`

	// ObjectStates tracks the current state of each interactable object
	// Key is object ID (e.g., "chest_1", "door_main")
	ObjectStates map[string]string `json:"object_states"`

	// TriggeredInteractions tracks single-use interactions that have been triggered
	// Key is "objectID:interactionID"
	TriggeredInteractions map[string]bool `json:"triggered_interactions"`
}

// New creates a new empty GameState
func New() *GameState {
	return &GameState{
		Flags:                 make(map[string]bool),
		Counters:              make(map[string]int),
		Strings:               make(map[string]string),
		ObjectStates:          make(map[string]string),
		TriggeredInteractions: make(map[string]bool),
	}
}

// --- Flag operations ---

// GetFlag returns the value of a flag (false if not set)
func (gs *GameState) GetFlag(name string) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.Flags[name]
}

// SetFlag sets a flag to a specific value
func (gs *GameState) SetFlag(name string, value bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.Flags[name] = value
}

// ToggleFlag flips a flag's value and returns the new value
func (gs *GameState) ToggleFlag(name string) bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.Flags[name] = !gs.Flags[name]
	return gs.Flags[name]
}

// --- Counter operations ---

// GetCounter returns the value of a counter (0 if not set)
func (gs *GameState) GetCounter(name string) int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.Counters[name]
}

// SetCounter sets a counter to a specific value
func (gs *GameState) SetCounter(name string, value int) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.Counters[name] = value
}

// IncrementCounter adds delta to a counter (can be negative)
func (gs *GameState) IncrementCounter(name string, delta int) int {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.Counters[name] += delta
	return gs.Counters[name]
}

// --- String operations ---

// GetString returns the value of a string variable (empty if not set)
func (gs *GameState) GetString(name string) string {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.Strings[name]
}

// SetString sets a string variable
func (gs *GameState) SetString(name string, value string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.Strings[name] = value
}

// --- Object state operations ---

// GetObjectState returns the current state of an object
func (gs *GameState) GetObjectState(objectID string) string {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.ObjectStates[objectID]
}

// SetObjectState sets the state of an object
func (gs *GameState) SetObjectState(objectID string, state string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.ObjectStates[objectID] = state
}

// --- Triggered interaction tracking ---

// IsInteractionTriggered checks if a single-use interaction has been triggered
func (gs *GameState) IsInteractionTriggered(objectID, interactionID string) bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", objectID, interactionID)
	return gs.TriggeredInteractions[key]
}

// MarkInteractionTriggered marks a single-use interaction as triggered
func (gs *GameState) MarkInteractionTriggered(objectID, interactionID string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	key := fmt.Sprintf("%s:%s", objectID, interactionID)
	gs.TriggeredInteractions[key] = true
}

// --- Serialization ---

// Save writes the game state to a file
func (gs *GameState) Save(filepath string) error {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	data, err := json.MarshalIndent(gs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize game state: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write game state file: %w", err)
	}

	return nil
}

// Load reads the game state from a file
func Load(filepath string) (*GameState, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read game state file: %w", err)
	}

	gs := New()
	if err := json.Unmarshal(data, gs); err != nil {
		return nil, fmt.Errorf("failed to parse game state: %w", err)
	}

	return gs, nil
}

// Reset clears all game state (for new game)
func (gs *GameState) Reset() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.Flags = make(map[string]bool)
	gs.Counters = make(map[string]int)
	gs.Strings = make(map[string]string)
	gs.ObjectStates = make(map[string]string)
	gs.TriggeredInteractions = make(map[string]bool)
}

// Clone creates a deep copy of the game state (useful for save states)
func (gs *GameState) Clone() *GameState {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	clone := New()
	for k, v := range gs.Flags {
		clone.Flags[k] = v
	}
	for k, v := range gs.Counters {
		clone.Counters[k] = v
	}
	for k, v := range gs.Strings {
		clone.Strings[k] = v
	}
	for k, v := range gs.ObjectStates {
		clone.ObjectStates[k] = v
	}
	for k, v := range gs.TriggeredInteractions {
		clone.TriggeredInteractions[k] = v
	}
	return clone
}

// Debug returns a string representation of the game state for debugging
func (gs *GameState) Debug() string {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	return fmt.Sprintf("GameState{Flags: %d, Counters: %d, Strings: %d, ObjectStates: %d, Triggered: %d}",
		len(gs.Flags), len(gs.Counters), len(gs.Strings), len(gs.ObjectStates), len(gs.TriggeredInteractions))
}
