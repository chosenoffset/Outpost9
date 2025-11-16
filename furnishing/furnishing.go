package furnishing

import (
	"encoding/json"
	"fmt"
	"os"
)

// FurnishingDefinition represents a template for objects that can be placed in rooms
type FurnishingDefinition struct {
	Name         string            `json:"name"`         // Unique identifier
	DisplayName  string            `json:"display_name"` // Human-readable name
	Description  string            `json:"description"`  // Description for interaction/inspection
	TileName     string            `json:"tile_name"`    // Sprite tile to render
	Interactable bool              `json:"interactable"` // Can the player interact with this?
	Walkable     bool              `json:"walkable"`     // Can the player walk through it?
	Tags         []string          `json:"tags"`         // Categorization (e.g., "furniture", "container", "light_source")
	Properties   map[string]string `json:"properties"`   // Custom properties for game logic
}

// PlacedFurnishing represents an instance of a furnishing in a specific location
type PlacedFurnishing struct {
	Definition *FurnishingDefinition
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
