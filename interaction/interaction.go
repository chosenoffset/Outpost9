// Package interaction provides a data-driven, game-agnostic interaction system.
// Interactions are defined in JSON and can be attached to furnishings, tiles, or entities.
package interaction

import (
	"encoding/json"
	"fmt"
)

// TriggerType defines what initiates an interaction
type TriggerType string

const (
	TriggerInteract TriggerType = "interact" // Player presses interact key (E)
	TriggerEnter    TriggerType = "enter"    // Player enters tile/area
	TriggerExit     TriggerType = "exit"     // Player exits tile/area
	TriggerUseItem  TriggerType = "use_item" // Player uses item on object
	TriggerAuto     TriggerType = "auto"     // Automatic when conditions met
)

// Condition represents a single condition that must be true for an interaction
type Condition struct {
	Type  string                 `json:"type"`            // Condition type (e.g., "state_equals", "has_item")
	Value interface{}            `json:"value,omitempty"` // Primary value for simple conditions
	Not   bool                   `json:"not,omitempty"`   // Negate this condition
	Args  map[string]interface{} `json:"args,omitempty"`  // Additional arguments for complex conditions
}

// Effect represents a single effect that occurs when an interaction is triggered
type Effect struct {
	Type   string                 `json:"type"`             // Effect type (e.g., "set_state", "give_item")
	Value  interface{}            `json:"value,omitempty"`  // Primary value for simple effects
	Target string                 `json:"target,omitempty"` // Target object (for effects that affect other objects)
	Args   map[string]interface{} `json:"args,omitempty"`   // Additional arguments for complex effects
}

// Interaction defines a complete interaction with trigger, conditions, and effects
type Interaction struct {
	ID          string      `json:"id,omitempty"`       // Optional unique identifier
	Trigger     TriggerType `json:"trigger"`            // What triggers this interaction
	Priority    int         `json:"priority,omitempty"` // Higher priority interactions are checked first
	Conditions  []Condition `json:"conditions"`         // All conditions must be true
	Effects     []Effect    `json:"effects"`            // Effects to execute when triggered
	Cooldown    float64     `json:"cooldown,omitempty"` // Seconds before can trigger again
	SingleUse   bool        `json:"single_use"`         // If true, can only trigger once
	Description string      `json:"description"`        // Human-readable description (shown to player)
}

// StateDefinition defines visual/behavioral states for an object
type StateDefinition struct {
	TileName    string            `json:"tile_name,omitempty"`   // Override tile for this state
	Walkable    *bool             `json:"walkable,omitempty"`    // Override walkability
	BlocksSight *bool             `json:"blocks_sight,omitempty"` // Override sight blocking
	Properties  map[string]string `json:"properties,omitempty"`  // State-specific properties
}

// InteractionSet is a collection of interactions attached to an object
type InteractionSet struct {
	DefaultState string                     `json:"default_state,omitempty"` // Starting state
	States       map[string]StateDefinition `json:"states,omitempty"`        // State definitions
	Interactions []Interaction              `json:"interactions"`            // All possible interactions
}

// Validate checks if an interaction is properly configured
func (i *Interaction) Validate() error {
	if i.Trigger == "" {
		return fmt.Errorf("interaction must have a trigger")
	}

	// Validate trigger type
	switch i.Trigger {
	case TriggerInteract, TriggerEnter, TriggerExit, TriggerUseItem, TriggerAuto:
		// Valid
	default:
		return fmt.Errorf("unknown trigger type: %s", i.Trigger)
	}

	if len(i.Effects) == 0 {
		return fmt.Errorf("interaction must have at least one effect")
	}

	return nil
}

// Clone creates a deep copy of an Interaction
func (i *Interaction) Clone() *Interaction {
	clone := &Interaction{
		ID:          i.ID,
		Trigger:     i.Trigger,
		Priority:    i.Priority,
		Cooldown:    i.Cooldown,
		SingleUse:   i.SingleUse,
		Description: i.Description,
	}

	// Deep copy conditions
	clone.Conditions = make([]Condition, len(i.Conditions))
	for idx, c := range i.Conditions {
		clone.Conditions[idx] = Condition{
			Type:  c.Type,
			Value: c.Value,
			Not:   c.Not,
		}
		if c.Args != nil {
			clone.Conditions[idx].Args = make(map[string]interface{})
			for k, v := range c.Args {
				clone.Conditions[idx].Args[k] = v
			}
		}
	}

	// Deep copy effects
	clone.Effects = make([]Effect, len(i.Effects))
	for idx, e := range i.Effects {
		clone.Effects[idx] = Effect{
			Type:   e.Type,
			Value:  e.Value,
			Target: e.Target,
		}
		if e.Args != nil {
			clone.Effects[idx].Args = make(map[string]interface{})
			for k, v := range e.Args {
				clone.Effects[idx].Args[k] = v
			}
		}
	}

	return clone
}

// ParseInteractionSet parses an InteractionSet from JSON data
func ParseInteractionSet(data []byte) (*InteractionSet, error) {
	var set InteractionSet
	if err := json.Unmarshal(data, &set); err != nil {
		return nil, fmt.Errorf("failed to parse interaction set: %w", err)
	}

	// Validate all interactions
	for i, interaction := range set.Interactions {
		if err := interaction.Validate(); err != nil {
			return nil, fmt.Errorf("interaction %d: %w", i, err)
		}
	}

	return &set, nil
}

// GetInteractionsForTrigger returns all interactions that match a trigger type
func (s *InteractionSet) GetInteractionsForTrigger(trigger TriggerType) []Interaction {
	var result []Interaction
	for _, interaction := range s.Interactions {
		if interaction.Trigger == trigger {
			result = append(result, interaction)
		}
	}
	return result
}

// GetStateDefinition returns the state definition for a given state name
func (s *InteractionSet) GetStateDefinition(state string) *StateDefinition {
	if s.States == nil {
		return nil
	}
	if def, ok := s.States[state]; ok {
		return &def
	}
	return nil
}
