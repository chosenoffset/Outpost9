package furnishing

import (
	"encoding/json"
	"fmt"
	"os"

	"chosenoffset.com/outpost9/interaction"
)

// StateDefinition defines visual/behavioral properties for a specific state
type StateDefinition struct {
	TileName    string `json:"tile_name,omitempty"`    // Override tile for this state
	Walkable    *bool  `json:"walkable,omitempty"`     // Override walkability
	BlocksSight *bool  `json:"blocks_sight,omitempty"` // Override sight blocking
}

// FurnishingDefinition represents a template for objects that can be placed in rooms
type FurnishingDefinition struct {
	Name         string            `json:"name"`         // Unique identifier
	DisplayName  string            `json:"display_name"` // Human-readable name
	Description  string            `json:"description"`  // Description for interaction/inspection
	TileName     string            `json:"tile_name"`    // Default sprite tile to render
	Interactable bool              `json:"interactable"` // Can the player interact with this?
	Walkable     bool              `json:"walkable"`     // Can the player walk through it?
	Tags         []string          `json:"tags"`         // Categorization (e.g., "furniture", "container", "light_source")
	Properties   map[string]string `json:"properties"`   // Custom properties for game logic

	// Interaction system fields
	DefaultState string                     `json:"default_state,omitempty"` // Initial state (e.g., "closed")
	States       map[string]StateDefinition `json:"states,omitempty"`        // State-specific overrides
	Interactions []interaction.Interaction  `json:"interactions,omitempty"`  // Available interactions
}

// PlacedFurnishing represents an instance of a furnishing in a specific location
type PlacedFurnishing struct {
	Definition *FurnishingDefinition
	ID         string // Unique identifier for this instance (e.g., "chest_room1_0")
	X          int    // Grid X position (in tiles)
	Y          int    // Grid Y position (in tiles)
	RoomID     int    // Which room this belongs to (-1 for world-placed)
	State      string // Current state (e.g., "open", "closed", "broken")
}

// RoomFurnishingPlacement defines where to place a furnishing within a room template
type RoomFurnishingPlacement struct {
	FurnishingName string `json:"furnishing_name"` // Name of furnishing to place
	X              int    `json:"x"`               // Relative X position within room
	Y              int    `json:"y"`               // Relative Y position within room
	State          string `json:"state"`           // Initial state (optional)
	ID             string `json:"id,omitempty"`    // Optional custom ID for this placement
}

// FurnishingLibrary holds a collection of furnishing definitions
type FurnishingLibrary struct {
	Name        string                  `json:"name"`        // Library name
	Description string                  `json:"description"` // Optional description
	Furnishings []*FurnishingDefinition `json:"furnishings"` // All furnishing definitions
}

// Validate checks if a furnishing definition is valid
func (f *FurnishingDefinition) Validate() error {
	if f.Name == "" {
		return fmt.Errorf("furnishing name is required")
	}
	if f.TileName == "" {
		return fmt.Errorf("furnishing %s: tile_name is required", f.Name)
	}

	// Validate interactions if present
	for i, inter := range f.Interactions {
		if err := inter.Validate(); err != nil {
			return fmt.Errorf("furnishing %s, interaction %d: %w", f.Name, i, err)
		}
	}

	return nil
}

// HasTag checks if the furnishing has a specific tag
func (f *FurnishingDefinition) HasTag(tag string) bool {
	for _, t := range f.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetProperty retrieves a custom property value
func (f *FurnishingDefinition) GetProperty(key string) (string, bool) {
	val, ok := f.Properties[key]
	return val, ok
}

// GetStateDefinition returns the state definition for a given state, or nil
func (f *FurnishingDefinition) GetStateDefinition(state string) *StateDefinition {
	if f.States == nil {
		return nil
	}
	if def, ok := f.States[state]; ok {
		return &def
	}
	return nil
}

// --- PlacedFurnishing implements interaction.InteractableObject ---

// GetID returns the unique identifier for this furnishing instance
func (pf *PlacedFurnishing) GetID() string {
	return pf.ID
}

// GetState returns the current state of this furnishing
func (pf *PlacedFurnishing) GetState() string {
	return pf.State
}

// SetState changes the state of this furnishing
func (pf *PlacedFurnishing) SetState(state string) {
	pf.State = state
}

// GetInteractions returns all interactions defined for this furnishing
func (pf *PlacedFurnishing) GetInteractions() []interaction.Interaction {
	if pf.Definition == nil {
		return nil
	}
	return pf.Definition.Interactions
}

// GetStateDefinition returns the state definition for a given state
func (pf *PlacedFurnishing) GetStateDefinition(state string) *interaction.StateDefinition {
	if pf.Definition == nil {
		return nil
	}
	def := pf.Definition.GetStateDefinition(state)
	if def == nil {
		return nil
	}
	// Convert to interaction.StateDefinition
	return &interaction.StateDefinition{
		TileName:    def.TileName,
		Walkable:    def.Walkable,
		BlocksSight: def.BlocksSight,
	}
}

// GetCurrentTileName returns the tile name for the current state
func (pf *PlacedFurnishing) GetCurrentTileName() string {
	if pf.Definition == nil {
		return ""
	}

	// Check if current state has a tile override
	if pf.State != "" && pf.Definition.States != nil {
		if stateDef, ok := pf.Definition.States[pf.State]; ok {
			if stateDef.TileName != "" {
				return stateDef.TileName
			}
		}
	}

	// Fall back to default tile
	return pf.Definition.TileName
}

// IsWalkable returns whether the player can walk through this furnishing
func (pf *PlacedFurnishing) IsWalkable() bool {
	if pf.Definition == nil {
		return true
	}

	// Check if current state has a walkable override
	if pf.State != "" && pf.Definition.States != nil {
		if stateDef, ok := pf.Definition.States[pf.State]; ok {
			if stateDef.Walkable != nil {
				return *stateDef.Walkable
			}
		}
	}

	// Fall back to default
	return pf.Definition.Walkable
}

// IsInteractable returns whether this furnishing can be interacted with
func (pf *PlacedFurnishing) IsInteractable() bool {
	if pf.Definition == nil {
		return false
	}
	return pf.Definition.Interactable
}

// LoadFurnishingLibrary loads a furnishing library from a JSON file
func LoadFurnishingLibrary(path string) (*FurnishingLibrary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read furnishing library file: %w", err)
	}

	var library FurnishingLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		return nil, fmt.Errorf("failed to parse furnishing library JSON: %w", err)
	}

	// Validate all furnishings
	for _, furnishing := range library.Furnishings {
		if err := furnishing.Validate(); err != nil {
			return nil, err
		}
	}

	return &library, nil
}

// GetFurnishingByName finds a furnishing by its name
func (l *FurnishingLibrary) GetFurnishingByName(name string) *FurnishingDefinition {
	for _, furnishing := range l.Furnishings {
		if furnishing.Name == name {
			return furnishing
		}
	}
	return nil
}

// GetFurnishingsByTag returns all furnishings with a specific tag
func (l *FurnishingLibrary) GetFurnishingsByTag(tag string) []*FurnishingDefinition {
	var result []*FurnishingDefinition
	for _, furnishing := range l.Furnishings {
		if furnishing.HasTag(tag) {
			result = append(result, furnishing)
		}
	}
	return result
}

// GetInteractableFurnishings returns all interactable furnishings
func (l *FurnishingLibrary) GetInteractableFurnishings() []*FurnishingDefinition {
	var result []*FurnishingDefinition
	for _, furnishing := range l.Furnishings {
		if furnishing.Interactable {
			result = append(result, furnishing)
		}
	}
	return result
}
