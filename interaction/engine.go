package interaction

import (
	"fmt"
	"log"
	"sort"
	"time"
)

// InteractableObject is the interface that objects must implement to be interactable
type InteractableObject interface {
	GetID() string
	GetState() string
	SetState(state string)
	GetInteractions() []Interaction
	GetStateDefinition(state string) *StateDefinition
}

// MessageHandler is called when a message should be displayed to the player
type MessageHandler func(message string)

// Engine processes interactions between the player and objects
type Engine struct {
	// Game state provider (for flags, counters, etc.)
	GameState GameStateMutator

	// Player inventory
	Inventory InventoryMutator

	// Message display callback
	OnMessage MessageHandler

	// Sound playback callback
	OnPlaySound func(soundName string)

	// World manipulation callbacks
	OnSpawnEntity    func(entityType string, x, y int)
	OnRemoveObject   func(objectID string)
	OnTeleportPlayer func(x, y int)

	// Object lookup for cross-object effects
	ObjectLookup func(objectID string) InteractableObject

	// Track cooldowns: objectID:interactionID -> time when cooldown ends
	cooldowns map[string]time.Time

	// Track single-use interactions: objectID:interactionID -> triggered
	triggered map[string]bool

	// Message queue for batched message display
	pendingMessages []string
}

// NewEngine creates a new interaction engine
func NewEngine() *Engine {
	return &Engine{
		cooldowns: make(map[string]time.Time),
		triggered: make(map[string]bool),
	}
}

// TryInteract attempts to trigger an interaction on an object
// Returns true if an interaction was successfully triggered
func (e *Engine) TryInteract(obj InteractableObject, trigger TriggerType, usedItem string) bool {
	if obj == nil {
		return false
	}

	interactions := obj.GetInteractions()
	if len(interactions) == 0 {
		return false
	}

	// Filter to matching trigger type
	var matching []Interaction
	for _, interaction := range interactions {
		if interaction.Trigger == trigger {
			matching = append(matching, interaction)
		}
	}

	if len(matching) == 0 {
		return false
	}

	// Sort by priority (higher first)
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].Priority > matching[j].Priority
	})

	// Build condition context
	condCtx := &ConditionContext{
		ObjectState: obj.GetState(),
		ObjectID:    obj.GetID(),
		GameState:   e.GameState,
		Inventory:   e.Inventory,
		UsedItem:    usedItem,
	}

	// Try each interaction until one succeeds
	for _, interaction := range matching {
		// Check if on cooldown
		cooldownKey := fmt.Sprintf("%s:%s", obj.GetID(), interaction.ID)
		if cooldownEnd, ok := e.cooldowns[cooldownKey]; ok {
			if time.Now().Before(cooldownEnd) {
				continue // Still on cooldown
			}
		}

		// Check if single-use and already triggered
		if interaction.SingleUse {
			if e.triggered[cooldownKey] {
				continue // Already triggered
			}
		}

		// Evaluate conditions
		pass, err := EvaluateConditions(interaction.Conditions, condCtx)
		if err != nil {
			log.Printf("Error evaluating conditions for %s: %v", obj.GetID(), err)
			continue
		}

		if !pass {
			continue // Conditions not met
		}

		// Execute effects
		if err := e.executeInteraction(obj, &interaction); err != nil {
			log.Printf("Error executing interaction on %s: %v", obj.GetID(), err)
			continue
		}

		// Mark cooldown
		if interaction.Cooldown > 0 {
			e.cooldowns[cooldownKey] = time.Now().Add(time.Duration(interaction.Cooldown * float64(time.Second)))
		}

		// Mark single-use
		if interaction.SingleUse {
			e.triggered[cooldownKey] = true
		}

		// Flush pending messages
		e.flushMessages()

		return true // Successfully triggered an interaction
	}

	return false
}

// executeInteraction runs all effects of an interaction
func (e *Engine) executeInteraction(obj InteractableObject, interaction *Interaction) error {
	// Build effect context
	effectCtx := &EffectContext{
		ObjectID:    obj.GetID(),
		ObjectState: obj.GetState(),

		SetObjectState: func(targetID, newState string) {
			if targetID == obj.GetID() || targetID == "self" || targetID == "" {
				obj.SetState(newState)
			} else if e.ObjectLookup != nil {
				if target := e.ObjectLookup(targetID); target != nil {
					target.SetState(newState)
				}
			}
		},

		GameState: e.GameState,
		Inventory: e.Inventory,

		ShowMessage: func(message string) {
			e.pendingMessages = append(e.pendingMessages, message)
		},

		PlaySound: e.OnPlaySound,

		SpawnEntity:    e.OnSpawnEntity,
		RemoveObject:   e.OnRemoveObject,
		TeleportPlayer: e.OnTeleportPlayer,

		ResolveTarget: func(targetID string) string {
			// Simple target resolution - just return as-is
			// Could be extended to support relative references like "nearest_door"
			return targetID
		},
	}

	return ExecuteEffects(interaction.Effects, effectCtx)
}

// flushMessages sends all pending messages to the handler
func (e *Engine) flushMessages() {
	if e.OnMessage == nil || len(e.pendingMessages) == 0 {
		return
	}

	// Combine messages or send individually based on preference
	for _, msg := range e.pendingMessages {
		e.OnMessage(msg)
	}
	e.pendingMessages = nil
}

// GetAvailableInteractions returns all interactions that could potentially trigger
// (used for UI hints like "Press E to open")
func (e *Engine) GetAvailableInteractions(obj InteractableObject, trigger TriggerType) []Interaction {
	if obj == nil {
		return nil
	}

	interactions := obj.GetInteractions()
	var available []Interaction

	condCtx := &ConditionContext{
		ObjectState: obj.GetState(),
		ObjectID:    obj.GetID(),
		GameState:   e.GameState,
		Inventory:   e.Inventory,
	}

	for _, interaction := range interactions {
		if interaction.Trigger != trigger {
			continue
		}

		// Check cooldown
		cooldownKey := fmt.Sprintf("%s:%s", obj.GetID(), interaction.ID)
		if cooldownEnd, ok := e.cooldowns[cooldownKey]; ok {
			if time.Now().Before(cooldownEnd) {
				continue
			}
		}

		// Check single-use
		if interaction.SingleUse && e.triggered[cooldownKey] {
			continue
		}

		// Check conditions
		pass, _ := EvaluateConditions(interaction.Conditions, condCtx)
		if pass {
			available = append(available, interaction)
		}
	}

	// Sort by priority
	sort.Slice(available, func(i, j int) bool {
		return available[i].Priority > available[j].Priority
	})

	return available
}

// GetInteractionHint returns a description of the first available interaction
// Returns empty string if no interaction is available
func (e *Engine) GetInteractionHint(obj InteractableObject, trigger TriggerType) string {
	available := e.GetAvailableInteractions(obj, trigger)
	if len(available) == 0 {
		return ""
	}
	return available[0].Description
}

// Reset clears all cooldowns and triggered states
func (e *Engine) Reset() {
	e.cooldowns = make(map[string]time.Time)
	e.triggered = make(map[string]bool)
	e.pendingMessages = nil
}

// CleanupCooldowns removes expired cooldowns (call periodically to prevent memory buildup)
func (e *Engine) CleanupCooldowns() {
	now := time.Now()
	for key, end := range e.cooldowns {
		if now.After(end) {
			delete(e.cooldowns, key)
		}
	}
}
