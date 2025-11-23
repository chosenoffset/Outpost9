// Package action provides a data-driven action system for the simulation.
// Actions are loaded from JSON files and define what entities can do each turn.
package action

import (
	"encoding/json"
	"fmt"
	"os"
)

// TargetingType defines how an action selects its target
type TargetingType string

const (
	TargetNone      TargetingType = "none"      // No target needed (self-buffs, wait)
	TargetSelf      TargetingType = "self"      // Targets the actor
	TargetDirection TargetingType = "direction" // Pick a direction (movement)
	TargetEntity    TargetingType = "entity"    // Select a specific entity
	TargetTile      TargetingType = "tile"      // Select a specific tile
	TargetAdjacent  TargetingType = "adjacent"  // Any adjacent tile/entity
)

// ActionCategory groups related actions together
type ActionCategory string

const (
	CategoryMovement   ActionCategory = "movement"
	CategoryCombat     ActionCategory = "combat"
	CategoryStealth    ActionCategory = "stealth"
	CategoryPerception ActionCategory = "perception"
	CategoryInteract   ActionCategory = "interact"
	CategoryUtility    ActionCategory = "utility"
)

// Targeting defines how an action acquires its target
type Targeting struct {
	Type     TargetingType `json:"type"`
	Range    int           `json:"range,omitempty"`     // Max range in tiles (0 = unlimited for type)
	MinRange int           `json:"min_range,omitempty"` // Minimum range (for ranged attacks)
	ArcAngle int           `json:"arc_angle,omitempty"` // For cone effects (degrees)
	Radius   int           `json:"radius,omitempty"`    // For area effects
}

// Requirement defines a condition that must be met to use an action
type Requirement struct {
	Type     string `json:"type"`               // "skill", "item", "status", "context"
	ID       string `json:"id,omitempty"`       // Skill ID, item ID, etc.
	MinValue int    `json:"min_value,omitempty"` // Minimum skill level, etc.
	Value    string `json:"value,omitempty"`    // For context checks like "in_shadow"
}

// Effect defines what happens when an action is executed
type Effect struct {
	Type       string `json:"type"`                  // "damage", "heal", "buff", "debuff", "move", "status", etc.
	Value      string `json:"value,omitempty"`       // Dice expression or fixed value
	Target     string `json:"target,omitempty"`      // "self", "target", "area"
	Stat       string `json:"stat,omitempty"`        // For buff/debuff effects
	Status     string `json:"status,omitempty"`      // For status effects
	Duration   int    `json:"duration,omitempty"`    // Turns the effect lasts
	DamageType string `json:"damage_type,omitempty"` // "physical", "fire", etc.
}

// SkillCheck defines an optional skill check for the action
type SkillCheck struct {
	Skill      string `json:"skill"`                 // Skill to check
	OpposedBy  string `json:"opposed_by,omitempty"`  // Defender's opposing skill
	Difficulty int    `json:"difficulty,omitempty"`  // Fixed DC if not opposed
	OnFail     string `json:"on_fail,omitempty"`     // What happens on failure
	Determines string `json:"determines,omitempty"`  // What success level affects
}

// Action defines a single action that can be taken
type Action struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Category    ActionCategory `json:"category"`

	// Cost
	APCost   int `json:"ap_cost"`             // Action points required
	AmmoCost int `json:"ammo_cost,omitempty"` // Ammo consumed (for ranged)

	// Noise (for stealth system)
	Noise int `json:"noise,omitempty"` // How much noise this action makes

	// Targeting
	Targeting Targeting `json:"targeting"`

	// Requirements to use this action
	Requirements []Requirement `json:"requirements,omitempty"`

	// Effects when action succeeds
	Effects []Effect `json:"effects,omitempty"`

	// Optional skill check
	SkillCheck *SkillCheck `json:"skill_check,omitempty"`

	// Combat modifiers (for attack actions)
	AttackModifier int `json:"attack_modifier,omitempty"` // Bonus/penalty to hit
	DamageModifier int `json:"damage_modifier,omitempty"` // Bonus/penalty to damage

	// UI hints
	Hotkey      string `json:"hotkey,omitempty"`       // Suggested keyboard shortcut
	IconName    string `json:"icon_name,omitempty"`    // Sprite name for icon
	ActionVerb  string `json:"action_verb,omitempty"`  // "attacks", "sneaks", etc.
	TargetVerb  string `json:"target_verb,omitempty"`  // "toward", "at", etc.
}

// ActionLibrary holds all loaded actions
type ActionLibrary struct {
	Actions     map[string]*Action            // All actions by ID
	Categories  map[ActionCategory][]*Action  // Actions grouped by category
	ActionOrder []string                      // Ordered list of action IDs (for stable UI)
}

// ActionsFile is the JSON file structure
type ActionsFile struct {
	Actions []Action `json:"actions"`
}

// LoadActionLibrary loads actions from a JSON file
func LoadActionLibrary(path string) (*ActionLibrary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read actions file: %w", err)
	}

	var file ActionsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("failed to parse actions file: %w", err)
	}

	library := &ActionLibrary{
		Actions:     make(map[string]*Action),
		Categories:  make(map[ActionCategory][]*Action),
		ActionOrder: make([]string, 0, len(file.Actions)),
	}

	for i := range file.Actions {
		action := &file.Actions[i]
		library.Actions[action.ID] = action
		library.Categories[action.Category] = append(library.Categories[action.Category], action)
		library.ActionOrder = append(library.ActionOrder, action.ID)
	}

	return library, nil
}

// GetAction returns an action by ID
func (lib *ActionLibrary) GetAction(id string) *Action {
	return lib.Actions[id]
}

// GetActionsByCategory returns all actions in a category
func (lib *ActionLibrary) GetActionsByCategory(category ActionCategory) []*Action {
	return lib.Categories[category]
}

// GetAllActions returns all actions in stable order
func (lib *ActionLibrary) GetAllActions() []*Action {
	result := make([]*Action, 0, len(lib.ActionOrder))
	for _, id := range lib.ActionOrder {
		if action, ok := lib.Actions[id]; ok {
			result = append(result, action)
		}
	}
	return result
}

// DefaultLibrary creates a library with built-in default actions
// These can be overridden by data files
func DefaultLibrary() *ActionLibrary {
	library := &ActionLibrary{
		Actions:     make(map[string]*Action),
		Categories:  make(map[ActionCategory][]*Action),
		ActionOrder: make([]string, 0),
	}

	// Built-in actions that are always available
	defaults := []Action{
		{
			ID:          "move",
			Name:        "Move",
			Description: "Walk to an adjacent tile",
			Category:    CategoryMovement,
			APCost:      1,
			Noise:       2,
			Targeting:   Targeting{Type: TargetDirection, Range: 1},
			Effects:     []Effect{{Type: "move", Value: "1"}},
			Hotkey:      "m",
			ActionVerb:  "moves",
			TargetVerb:  "toward",
		},
		{
			ID:          "wait",
			Name:        "Wait",
			Description: "Pass time without acting",
			Category:    CategoryUtility,
			APCost:      1,
			Noise:       0,
			Targeting:   Targeting{Type: TargetNone},
			Effects:     []Effect{{Type: "pass_time"}},
			Hotkey:      ".",
			ActionVerb:  "waits",
		},
	}

	for i := range defaults {
		action := &defaults[i]
		library.Actions[action.ID] = action
		library.Categories[action.Category] = append(library.Categories[action.Category], action)
		library.ActionOrder = append(library.ActionOrder, action.ID)
	}

	return library
}

// MergeLibrary adds actions from another library, overwriting duplicates
// New actions are added in the order they appear in the other library
func (lib *ActionLibrary) MergeLibrary(other *ActionLibrary) {
	// Add new actions in order from other library
	for _, id := range other.ActionOrder {
		action := other.Actions[id]
		if action == nil {
			continue
		}

		// Remove from old category if exists
		if existing, ok := lib.Actions[id]; ok {
			lib.removeFromCategory(existing)
		} else {
			// New action - add to order
			lib.ActionOrder = append(lib.ActionOrder, id)
		}

		lib.Actions[id] = action
		lib.Categories[action.Category] = append(lib.Categories[action.Category], action)
	}
}

func (lib *ActionLibrary) removeFromCategory(action *Action) {
	actions := lib.Categories[action.Category]
	for i, a := range actions {
		if a.ID == action.ID {
			lib.Categories[action.Category] = append(actions[:i], actions[i+1:]...)
			return
		}
	}
}
