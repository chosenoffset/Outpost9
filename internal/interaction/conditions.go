package interaction

import (
	"fmt"
	"math/rand"
	"strings"
)

// ConditionContext provides all the data needed to evaluate conditions
type ConditionContext struct {
	// Object state
	ObjectState string // Current state of the object being interacted with
	ObjectID    string // ID of the object

	// Player/game state (interfaces to avoid circular dependencies)
	GameState     GameStateProvider
	Inventory     InventoryProvider

	// Optional: item being used (for use_item trigger)
	UsedItem string
}

// GameStateProvider interface for accessing game flags/state
type GameStateProvider interface {
	GetFlag(name string) bool
	GetCounter(name string) int
	GetString(name string) string
}

// InventoryProvider interface for checking player inventory
type InventoryProvider interface {
	HasItem(itemName string) bool
	GetItemCount(itemName string) int
}

// ConditionEvaluator is a function that evaluates a specific condition type
type ConditionEvaluator func(condition *Condition, ctx *ConditionContext) (bool, error)

// conditionRegistry maps condition type names to evaluator functions
var conditionRegistry = map[string]ConditionEvaluator{}

// RegisterCondition registers a custom condition evaluator
func RegisterCondition(name string, evaluator ConditionEvaluator) {
	conditionRegistry[name] = evaluator
}

// EvaluateCondition evaluates a single condition
func EvaluateCondition(condition *Condition, ctx *ConditionContext) (bool, error) {
	evaluator, ok := conditionRegistry[condition.Type]
	if !ok {
		return false, fmt.Errorf("unknown condition type: %s", condition.Type)
	}

	result, err := evaluator(condition, ctx)
	if err != nil {
		return false, err
	}

	// Apply NOT modifier
	if condition.Not {
		result = !result
	}

	return result, nil
}

// EvaluateConditions evaluates all conditions (AND logic - all must be true)
func EvaluateConditions(conditions []Condition, ctx *ConditionContext) (bool, error) {
	for _, condition := range conditions {
		result, err := EvaluateCondition(&condition, ctx)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

// Helper functions for extracting values from conditions

func getStringValue(condition *Condition) string {
	if s, ok := condition.Value.(string); ok {
		return s
	}
	return ""
}

func getIntValue(condition *Condition) int {
	switch v := condition.Value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	}
	return 0
}

func getFloatValue(condition *Condition) float64 {
	switch v := condition.Value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func getArgString(condition *Condition, key string) string {
	if condition.Args == nil {
		return ""
	}
	if v, ok := condition.Args[key].(string); ok {
		return v
	}
	return ""
}

func getArgInt(condition *Condition, key string) int {
	if condition.Args == nil {
		return 0
	}
	switch v := condition.Args[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	}
	return 0
}

// Built-in condition evaluators

func init() {
	// state_equals: Check if object is in a specific state
	// Usage: {"type": "state_equals", "value": "closed"}
	RegisterCondition("state_equals", func(c *Condition, ctx *ConditionContext) (bool, error) {
		expected := getStringValue(c)
		return strings.EqualFold(ctx.ObjectState, expected), nil
	})

	// state_not_equals: Check if object is NOT in a specific state
	// Usage: {"type": "state_not_equals", "value": "broken"}
	RegisterCondition("state_not_equals", func(c *Condition, ctx *ConditionContext) (bool, error) {
		expected := getStringValue(c)
		return !strings.EqualFold(ctx.ObjectState, expected), nil
	})

	// has_item: Check if player has an item
	// Usage: {"type": "has_item", "value": "iron_key"}
	// Or with count: {"type": "has_item", "value": "gold", "args": {"min_count": 10}}
	RegisterCondition("has_item", func(c *Condition, ctx *ConditionContext) (bool, error) {
		if ctx.Inventory == nil {
			return false, nil
		}
		itemName := getStringValue(c)
		minCount := getArgInt(c, "min_count")
		if minCount <= 0 {
			minCount = 1
		}
		return ctx.Inventory.GetItemCount(itemName) >= minCount, nil
	})

	// flag_set: Check if a game flag is set (true)
	// Usage: {"type": "flag_set", "value": "door_unlocked"}
	RegisterCondition("flag_set", func(c *Condition, ctx *ConditionContext) (bool, error) {
		if ctx.GameState == nil {
			return false, nil
		}
		flagName := getStringValue(c)
		return ctx.GameState.GetFlag(flagName), nil
	})

	// flag_not_set: Check if a game flag is NOT set (false)
	// Usage: {"type": "flag_not_set", "value": "alarm_triggered"}
	RegisterCondition("flag_not_set", func(c *Condition, ctx *ConditionContext) (bool, error) {
		if ctx.GameState == nil {
			return true, nil // If no game state, flags are not set
		}
		flagName := getStringValue(c)
		return !ctx.GameState.GetFlag(flagName), nil
	})

	// counter_equals: Check if a counter equals a value
	// Usage: {"type": "counter_equals", "value": "switches_activated", "args": {"equals": 3}}
	RegisterCondition("counter_equals", func(c *Condition, ctx *ConditionContext) (bool, error) {
		if ctx.GameState == nil {
			return false, nil
		}
		counterName := getStringValue(c)
		expected := getArgInt(c, "equals")
		return ctx.GameState.GetCounter(counterName) == expected, nil
	})

	// counter_at_least: Check if a counter is >= a value
	// Usage: {"type": "counter_at_least", "value": "keys_collected", "args": {"min": 3}}
	RegisterCondition("counter_at_least", func(c *Condition, ctx *ConditionContext) (bool, error) {
		if ctx.GameState == nil {
			return false, nil
		}
		counterName := getStringValue(c)
		min := getArgInt(c, "min")
		return ctx.GameState.GetCounter(counterName) >= min, nil
	})

	// random_chance: Random percentage chance (0-100)
	// Usage: {"type": "random_chance", "value": 25} // 25% chance
	RegisterCondition("random_chance", func(c *Condition, ctx *ConditionContext) (bool, error) {
		chance := getFloatValue(c)
		if chance <= 0 {
			return false, nil
		}
		if chance >= 100 {
			return true, nil
		}
		return rand.Float64()*100 < chance, nil
	})

	// always: Always true (useful for default/fallback interactions)
	// Usage: {"type": "always"}
	RegisterCondition("always", func(c *Condition, ctx *ConditionContext) (bool, error) {
		return true, nil
	})

	// never: Always false (useful for disabled interactions)
	// Usage: {"type": "never"}
	RegisterCondition("never", func(c *Condition, ctx *ConditionContext) (bool, error) {
		return false, nil
	})

	// used_item: Check if a specific item is being used (for use_item trigger)
	// Usage: {"type": "used_item", "value": "lockpick"}
	RegisterCondition("used_item", func(c *Condition, ctx *ConditionContext) (bool, error) {
		expected := getStringValue(c)
		return strings.EqualFold(ctx.UsedItem, expected), nil
	})

	// string_equals: Check if a game string variable equals a value
	// Usage: {"type": "string_equals", "value": "current_quest", "args": {"equals": "find_key"}}
	RegisterCondition("string_equals", func(c *Condition, ctx *ConditionContext) (bool, error) {
		if ctx.GameState == nil {
			return false, nil
		}
		varName := getStringValue(c)
		expected := getArgString(c, "equals")
		return ctx.GameState.GetString(varName) == expected, nil
	})
}
