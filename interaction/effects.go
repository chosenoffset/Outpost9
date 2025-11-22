package interaction

import (
	"fmt"
	"strings"
)

// EffectContext provides all the data needed to execute effects
type EffectContext struct {
	// Object being interacted with
	ObjectID    string
	ObjectState string

	// Callbacks to modify game state (set by the game engine)
	SetObjectState func(objectID string, newState string)

	// Game state modification
	GameState GameStateMutator

	// Inventory modification
	Inventory InventoryMutator

	// UI/feedback callbacks
	ShowMessage func(message string)
	PlaySound   func(soundName string)

	// World manipulation
	SpawnEntity  func(entityType string, x, y int)
	RemoveObject func(objectID string)
	TeleportPlayer func(x, y int)

	// For resolving targets relative to current object
	ResolveTarget func(targetID string) string // Returns resolved object ID
}

// GameStateMutator interface for modifying game flags/state
type GameStateMutator interface {
	GameStateProvider
	SetFlag(name string, value bool)
	SetCounter(name string, value int)
	IncrementCounter(name string, delta int) int
	SetString(name string, value string)
}

// InventoryMutator interface for modifying player inventory
type InventoryMutator interface {
	InventoryProvider
	AddItem(itemName string, count int) int
	RemoveItem(itemName string, count int) bool
	ClearItem(itemName string)
}

// EffectExecutor is a function that executes a specific effect type
type EffectExecutor func(effect *Effect, ctx *EffectContext) error

// effectRegistry maps effect type names to executor functions
var effectRegistry = map[string]EffectExecutor{}

// RegisterEffect registers a custom effect executor
func RegisterEffect(name string, executor EffectExecutor) {
	effectRegistry[name] = executor
}

// ExecuteEffect executes a single effect
func ExecuteEffect(effect *Effect, ctx *EffectContext) error {
	executor, ok := effectRegistry[effect.Type]
	if !ok {
		return fmt.Errorf("unknown effect type: %s", effect.Type)
	}
	return executor(effect, ctx)
}

// ExecuteEffects executes all effects in order
func ExecuteEffects(effects []Effect, ctx *EffectContext) error {
	for i, effect := range effects {
		if err := ExecuteEffect(&effect, ctx); err != nil {
			return fmt.Errorf("effect %d (%s): %w", i, effect.Type, err)
		}
	}
	return nil
}

// Helper functions for extracting values from effects

func getEffectStringValue(effect *Effect) string {
	if s, ok := effect.Value.(string); ok {
		return s
	}
	return ""
}

func getEffectIntValue(effect *Effect) int {
	switch v := effect.Value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	}
	return 0
}

func getEffectFloatValue(effect *Effect) float64 {
	switch v := effect.Value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func getEffectArgString(effect *Effect, key string) string {
	if effect.Args == nil {
		return ""
	}
	if v, ok := effect.Args[key].(string); ok {
		return v
	}
	return ""
}

func getEffectArgInt(effect *Effect, key string) int {
	if effect.Args == nil {
		return 0
	}
	switch v := effect.Args[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	}
	return 0
}

func getEffectArgBool(effect *Effect, key string) bool {
	if effect.Args == nil {
		return false
	}
	if v, ok := effect.Args[key].(bool); ok {
		return v
	}
	return false
}

// resolveTarget returns the target object ID, defaulting to the current object
func resolveTarget(effect *Effect, ctx *EffectContext) string {
	if effect.Target == "" || strings.EqualFold(effect.Target, "self") {
		return ctx.ObjectID
	}
	if ctx.ResolveTarget != nil {
		return ctx.ResolveTarget(effect.Target)
	}
	return effect.Target
}

// Built-in effect executors

func init() {
	// set_state: Change the state of an object
	// Usage: {"type": "set_state", "value": "open"}
	// Or for another object: {"type": "set_state", "value": "open", "target": "door_1"}
	RegisterEffect("set_state", func(e *Effect, ctx *EffectContext) error {
		newState := getEffectStringValue(e)
		targetID := resolveTarget(e, ctx)
		if ctx.SetObjectState != nil {
			ctx.SetObjectState(targetID, newState)
		}
		return nil
	})

	// show_message: Display a message to the player
	// Usage: {"type": "show_message", "value": "You found a key!"}
	RegisterEffect("show_message", func(e *Effect, ctx *EffectContext) error {
		message := getEffectStringValue(e)
		if ctx.ShowMessage != nil && message != "" {
			ctx.ShowMessage(message)
		}
		return nil
	})

	// give_item: Add an item to player inventory
	// Usage: {"type": "give_item", "value": "gold", "args": {"amount": 50}}
	// Or simple: {"type": "give_item", "value": "iron_key"}
	RegisterEffect("give_item", func(e *Effect, ctx *EffectContext) error {
		itemName := getEffectStringValue(e)
		amount := getEffectArgInt(e, "amount")
		if amount <= 0 {
			amount = 1
		}
		if ctx.Inventory != nil && itemName != "" {
			ctx.Inventory.AddItem(itemName, amount)
		}
		return nil
	})

	// remove_item: Remove an item from player inventory
	// Usage: {"type": "remove_item", "value": "iron_key"}
	// Or with amount: {"type": "remove_item", "value": "gold", "args": {"amount": 10}}
	RegisterEffect("remove_item", func(e *Effect, ctx *EffectContext) error {
		itemName := getEffectStringValue(e)
		amount := getEffectArgInt(e, "amount")
		if amount <= 0 {
			amount = 1
		}
		if ctx.Inventory != nil && itemName != "" {
			ctx.Inventory.RemoveItem(itemName, amount)
		}
		return nil
	})

	// set_flag: Set a game flag to true
	// Usage: {"type": "set_flag", "value": "door_unlocked"}
	RegisterEffect("set_flag", func(e *Effect, ctx *EffectContext) error {
		flagName := getEffectStringValue(e)
		if ctx.GameState != nil && flagName != "" {
			ctx.GameState.SetFlag(flagName, true)
		}
		return nil
	})

	// clear_flag: Set a game flag to false
	// Usage: {"type": "clear_flag", "value": "alarm_active"}
	RegisterEffect("clear_flag", func(e *Effect, ctx *EffectContext) error {
		flagName := getEffectStringValue(e)
		if ctx.GameState != nil && flagName != "" {
			ctx.GameState.SetFlag(flagName, false)
		}
		return nil
	})

	// set_counter: Set a counter to a specific value
	// Usage: {"type": "set_counter", "value": "switches_activated", "args": {"amount": 0}}
	RegisterEffect("set_counter", func(e *Effect, ctx *EffectContext) error {
		counterName := getEffectStringValue(e)
		amount := getEffectArgInt(e, "amount")
		if ctx.GameState != nil && counterName != "" {
			ctx.GameState.SetCounter(counterName, amount)
		}
		return nil
	})

	// increment_counter: Add to a counter
	// Usage: {"type": "increment_counter", "value": "enemies_killed"}
	// Or with delta: {"type": "increment_counter", "value": "score", "args": {"delta": 100}}
	RegisterEffect("increment_counter", func(e *Effect, ctx *EffectContext) error {
		counterName := getEffectStringValue(e)
		delta := getEffectArgInt(e, "delta")
		if delta == 0 {
			delta = 1
		}
		if ctx.GameState != nil && counterName != "" {
			ctx.GameState.IncrementCounter(counterName, delta)
		}
		return nil
	})

	// decrement_counter: Subtract from a counter
	// Usage: {"type": "decrement_counter", "value": "lives"}
	RegisterEffect("decrement_counter", func(e *Effect, ctx *EffectContext) error {
		counterName := getEffectStringValue(e)
		delta := getEffectArgInt(e, "delta")
		if delta == 0 {
			delta = 1
		}
		if ctx.GameState != nil && counterName != "" {
			ctx.GameState.IncrementCounter(counterName, -delta)
		}
		return nil
	})

	// set_string: Set a string variable
	// Usage: {"type": "set_string", "value": "current_quest", "args": {"string": "find_exit"}}
	RegisterEffect("set_string", func(e *Effect, ctx *EffectContext) error {
		varName := getEffectStringValue(e)
		strValue := getEffectArgString(e, "string")
		if ctx.GameState != nil && varName != "" {
			ctx.GameState.SetString(varName, strValue)
		}
		return nil
	})

	// play_sound: Play a sound effect
	// Usage: {"type": "play_sound", "value": "chest_open"}
	RegisterEffect("play_sound", func(e *Effect, ctx *EffectContext) error {
		soundName := getEffectStringValue(e)
		if ctx.PlaySound != nil && soundName != "" {
			ctx.PlaySound(soundName)
		}
		return nil
	})

	// spawn_entity: Create an entity at a position
	// Usage: {"type": "spawn_entity", "value": "skeleton", "args": {"x": 5, "y": 3}}
	RegisterEffect("spawn_entity", func(e *Effect, ctx *EffectContext) error {
		entityType := getEffectStringValue(e)
		x := getEffectArgInt(e, "x")
		y := getEffectArgInt(e, "y")
		if ctx.SpawnEntity != nil && entityType != "" {
			ctx.SpawnEntity(entityType, x, y)
		}
		return nil
	})

	// remove_object: Remove an object from the world
	// Usage: {"type": "remove_object"} // removes self
	// Or: {"type": "remove_object", "target": "barrier_1"}
	RegisterEffect("remove_object", func(e *Effect, ctx *EffectContext) error {
		targetID := resolveTarget(e, ctx)
		if ctx.RemoveObject != nil {
			ctx.RemoveObject(targetID)
		}
		return nil
	})

	// teleport_player: Move the player to a new position
	// Usage: {"type": "teleport_player", "args": {"x": 10, "y": 20}}
	RegisterEffect("teleport_player", func(e *Effect, ctx *EffectContext) error {
		x := getEffectArgInt(e, "x")
		y := getEffectArgInt(e, "y")
		if ctx.TeleportPlayer != nil {
			ctx.TeleportPlayer(x, y)
		}
		return nil
	})

	// unlock: Convenience effect to unlock a door/container (sets state to "unlocked")
	// Usage: {"type": "unlock"} or {"type": "unlock", "target": "door_1"}
	RegisterEffect("unlock", func(e *Effect, ctx *EffectContext) error {
		targetID := resolveTarget(e, ctx)
		if ctx.SetObjectState != nil {
			ctx.SetObjectState(targetID, "unlocked")
		}
		return nil
	})

	// lock: Convenience effect to lock a door/container (sets state to "locked")
	// Usage: {"type": "lock"} or {"type": "lock", "target": "door_1"}
	RegisterEffect("lock", func(e *Effect, ctx *EffectContext) error {
		targetID := resolveTarget(e, ctx)
		if ctx.SetObjectState != nil {
			ctx.SetObjectState(targetID, "locked")
		}
		return nil
	})

	// open: Convenience effect to open something (sets state to "open")
	// Usage: {"type": "open"}
	RegisterEffect("open", func(e *Effect, ctx *EffectContext) error {
		targetID := resolveTarget(e, ctx)
		if ctx.SetObjectState != nil {
			ctx.SetObjectState(targetID, "open")
		}
		return nil
	})

	// close: Convenience effect to close something (sets state to "closed")
	// Usage: {"type": "close"}
	RegisterEffect("close", func(e *Effect, ctx *EffectContext) error {
		targetID := resolveTarget(e, ctx)
		if ctx.SetObjectState != nil {
			ctx.SetObjectState(targetID, "closed")
		}
		return nil
	})

	// noop: Do nothing (useful for placeholder or testing)
	// Usage: {"type": "noop"}
	RegisterEffect("noop", func(e *Effect, ctx *EffectContext) error {
		return nil
	})
}
