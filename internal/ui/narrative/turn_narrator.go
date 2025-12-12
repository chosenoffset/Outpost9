// Package narrative turn_narrator.go provides turn-by-turn narrative generation
package narrative

import (
	"fmt"
	"strings"

	"chosenoffset.com/outpost9/internal/entity"
)

// TurnEventType represents different types of events that can occur during a turn
type TurnEventType int

const (
	EventMovement TurnEventType = iota
	EventAttack
	EventDamageDealt
	EventDamageTaken
	EventMiss
	EventDodge
	EventKill
	EventDeath
	EventRoomEnter
	EventRoomExit
	EventSearch
	EventDiscovery
	EventInteract
	EventItemPickup
	EventSkillCheck
	EventStatusEffect
	EventEnemyAction
	EventEnemySpotted
	EventWait
)

// TurnEvent represents a single event that occurred during a turn
type TurnEvent struct {
	Type        TurnEventType
	Actor       string // Name of the actor (player, enemy, etc.)
	Target      string // Name of the target (if applicable)
	Details     string // Additional details (damage amount, item name, etc.)
	Value       int    // Numeric value (damage, healing, etc.)
	Success     bool   // Whether the action succeeded
	IsCritical  bool   // Critical hit/success
	Direction   string // Direction of movement/action
	Description string // Pre-formatted description (optional override)
}

// TurnSummary collects all events from a single turn
type TurnSummary struct {
	TurnNumber     int
	Events         []TurnEvent
	PlayerMoved    bool
	PlayerActed    bool
	CombatOccurred bool
	RoomChanged    bool
	NewRoomName    string
}

// NewTurnSummary creates a new turn summary
func NewTurnSummary(turnNumber int) *TurnSummary {
	return &TurnSummary{
		TurnNumber: turnNumber,
		Events:     []TurnEvent{},
	}
}

// AddEvent adds an event to the turn summary
func (ts *TurnSummary) AddEvent(event TurnEvent) {
	ts.Events = append(ts.Events, event)

	// Update flags based on event type
	switch event.Type {
	case EventMovement:
		ts.PlayerMoved = true
	case EventAttack, EventDamageDealt, EventDamageTaken, EventKill, EventEnemyAction:
		ts.CombatOccurred = true
		ts.PlayerActed = true
	case EventRoomEnter:
		ts.RoomChanged = true
		ts.NewRoomName = event.Details
	case EventSearch, EventInteract, EventItemPickup:
		ts.PlayerActed = true
	}
}

// TurnNarrator generates narrative text from turn events
type TurnNarrator struct {
	currentSummary *TurnSummary

	// Customization
	UseVerboseDescriptions bool
	IncludeDirection       bool
}

// NewTurnNarrator creates a new turn narrator
func NewTurnNarrator() *TurnNarrator {
	return &TurnNarrator{
		UseVerboseDescriptions: true,
		IncludeDirection:       true,
	}
}

// StartTurn begins tracking a new turn
func (tn *TurnNarrator) StartTurn(turnNumber int) {
	tn.currentSummary = NewTurnSummary(turnNumber)
}

// GetCurrentSummary returns the current turn summary
func (tn *TurnNarrator) GetCurrentSummary() *TurnSummary {
	return tn.currentSummary
}

// RecordMovement records a movement event
func (tn *TurnNarrator) RecordMovement(actor string, direction entity.Direction, blocked bool) {
	dirName := directionToString(direction)
	event := TurnEvent{
		Type:      EventMovement,
		Actor:     actor,
		Direction: dirName,
		Success:   !blocked,
	}

	if blocked {
		event.Details = "blocked"
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordAttack records an attack event
func (tn *TurnNarrator) RecordAttack(attacker, target string, damage int, hit bool, critical bool) {
	eventType := EventAttack
	if !hit {
		eventType = EventMiss
	}

	event := TurnEvent{
		Type:       eventType,
		Actor:      attacker,
		Target:     target,
		Value:      damage,
		Success:    hit,
		IsCritical: critical,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordDamage records damage dealt or received
func (tn *TurnNarrator) RecordDamage(target string, damage int, source string, isPlayer bool) {
	eventType := EventDamageDealt
	if isPlayer {
		eventType = EventDamageTaken
	}

	event := TurnEvent{
		Type:   eventType,
		Actor:  source,
		Target: target,
		Value:  damage,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordKill records a kill event
func (tn *TurnNarrator) RecordKill(killer, victim string) {
	event := TurnEvent{
		Type:   EventKill,
		Actor:  killer,
		Target: victim,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordRoomEnter records entering a room
func (tn *TurnNarrator) RecordRoomEnter(roomName string, isFirstVisit bool) {
	event := TurnEvent{
		Type:    EventRoomEnter,
		Details: roomName,
		Success: isFirstVisit,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordSearch records a search action
func (tn *TurnNarrator) RecordSearch(location string, found bool, whatFound string) {
	event := TurnEvent{
		Type:    EventSearch,
		Details: whatFound,
		Success: found,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordDiscovery records a discovery (skill-based reveal)
func (tn *TurnNarrator) RecordDiscovery(skill string, description string) {
	event := TurnEvent{
		Type:    EventDiscovery,
		Actor:   skill,
		Details: description,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordEnemyAction records an enemy taking an action
func (tn *TurnNarrator) RecordEnemyAction(enemyName, actionType, target string, damage int) {
	event := TurnEvent{
		Type:    EventEnemyAction,
		Actor:   enemyName,
		Target:  target,
		Details: actionType,
		Value:   damage,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordEnemySpotted records spotting an enemy
func (tn *TurnNarrator) RecordEnemySpotted(enemyName, direction string, distance int) {
	event := TurnEvent{
		Type:      EventEnemySpotted,
		Actor:     enemyName,
		Direction: direction,
		Value:     distance,
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// RecordWait records the player waiting/ending turn
func (tn *TurnNarrator) RecordWait() {
	event := TurnEvent{
		Type:  EventWait,
		Actor: "You",
	}

	if tn.currentSummary != nil {
		tn.currentSummary.AddEvent(event)
	}
}

// GenerateTurnNarrative creates the narrative text for the current turn
func (tn *TurnNarrator) GenerateTurnNarrative() string {
	if tn.currentSummary == nil || len(tn.currentSummary.Events) == 0 {
		return ""
	}

	var parts []string

	// Generate narrative for each event
	for _, event := range tn.currentSummary.Events {
		text := tn.narrateEvent(event)
		if text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, " ")
}

// GenerateActionLog returns individual log entries for each event
func (tn *TurnNarrator) GenerateActionLog() []string {
	if tn.currentSummary == nil {
		return nil
	}

	var logs []string
	for _, event := range tn.currentSummary.Events {
		text := tn.narrateEvent(event)
		if text != "" {
			logs = append(logs, text)
		}
	}

	return logs
}

// narrateEvent generates narrative text for a single event
func (tn *TurnNarrator) narrateEvent(event TurnEvent) string {
	// Use pre-formatted description if provided
	if event.Description != "" {
		return event.Description
	}

	switch event.Type {
	case EventMovement:
		return tn.narrateMovement(event)
	case EventAttack:
		return tn.narrateAttack(event)
	case EventMiss:
		return tn.narrateMiss(event)
	case EventDamageDealt:
		return tn.narrateDamageDealt(event)
	case EventDamageTaken:
		return tn.narrateDamageTaken(event)
	case EventKill:
		return tn.narrateKill(event)
	case EventRoomEnter:
		return tn.narrateRoomEnter(event)
	case EventSearch:
		return tn.narrateSearch(event)
	case EventDiscovery:
		return tn.narrateDiscovery(event)
	case EventEnemyAction:
		return tn.narrateEnemyAction(event)
	case EventEnemySpotted:
		return tn.narrateEnemySpotted(event)
	case EventWait:
		return "You brace yourself."
	default:
		return ""
	}
}

func (tn *TurnNarrator) narrateMovement(event TurnEvent) string {
	if !event.Success {
		return fmt.Sprintf("You can't move %s - the way is blocked.", event.Direction)
	}

	if tn.IncludeDirection {
		return fmt.Sprintf("You move %s.", event.Direction)
	}
	return "You advance cautiously."
}

func (tn *TurnNarrator) narrateAttack(event TurnEvent) string {
	if event.IsCritical {
		return fmt.Sprintf("%s lands a devastating blow on %s for %d damage!", event.Actor, event.Target, event.Value)
	}
	return fmt.Sprintf("%s hits %s for %d damage.", event.Actor, event.Target, event.Value)
}

func (tn *TurnNarrator) narrateMiss(event TurnEvent) string {
	return fmt.Sprintf("%s's attack misses %s.", event.Actor, event.Target)
}

func (tn *TurnNarrator) narrateDamageDealt(event TurnEvent) string {
	return fmt.Sprintf("%s takes %d damage.", event.Target, event.Value)
}

func (tn *TurnNarrator) narrateDamageTaken(event TurnEvent) string {
	return fmt.Sprintf("You take %d damage from %s!", event.Value, event.Actor)
}

func (tn *TurnNarrator) narrateKill(event TurnEvent) string {
	if event.Actor == "You" {
		return fmt.Sprintf("You defeat the %s!", event.Target)
	}
	return fmt.Sprintf("The %s falls!", event.Target)
}

func (tn *TurnNarrator) narrateRoomEnter(event TurnEvent) string {
	if event.Success { // First visit
		return fmt.Sprintf("You enter %s for the first time.", event.Details)
	}
	return fmt.Sprintf("You return to %s.", event.Details)
}

func (tn *TurnNarrator) narrateSearch(event TurnEvent) string {
	if event.Success && event.Details != "" {
		return event.Details
	}
	return "You search the area."
}

func (tn *TurnNarrator) narrateDiscovery(event TurnEvent) string {
	if event.Details != "" {
		return fmt.Sprintf("[%s] %s", event.Actor, event.Details)
	}
	return "You notice something interesting."
}

func (tn *TurnNarrator) narrateEnemyAction(event TurnEvent) string {
	if event.Details == "attack" && event.Value > 0 {
		return fmt.Sprintf("The %s attacks %s for %d damage!", event.Actor, event.Target, event.Value)
	}
	if event.Details == "move" {
		return fmt.Sprintf("The %s moves closer.", event.Actor)
	}
	return fmt.Sprintf("The %s acts.", event.Actor)
}

func (tn *TurnNarrator) narrateEnemySpotted(event TurnEvent) string {
	if event.Value <= 2 {
		return fmt.Sprintf("A %s is right next to you!", event.Actor)
	}
	return fmt.Sprintf("You spot a %s to the %s.", event.Actor, event.Direction)
}

// GeneratePreparationText generates text about what the player should expect
func (tn *TurnNarrator) GeneratePreparationText(nearbyEnemies int, playerHP, maxHP int) string {
	var parts []string

	// Enemy warning
	if nearbyEnemies > 0 {
		if nearbyEnemies == 1 {
			parts = append(parts, "An enemy lurks nearby.")
		} else {
			parts = append(parts, fmt.Sprintf("%d enemies are nearby.", nearbyEnemies))
		}
	}

	// Health warning
	hpPercent := float64(playerHP) / float64(maxHP)
	if hpPercent < 0.25 {
		parts = append(parts, "You are critically wounded!")
	} else if hpPercent < 0.5 {
		parts = append(parts, "You are injured.")
	}

	if len(parts) == 0 {
		return "The way ahead looks clear."
	}

	return strings.Join(parts, " ")
}

// ClearTurn clears the current turn summary
func (tn *TurnNarrator) ClearTurn() {
	tn.currentSummary = nil
}

// HasEvents returns whether any events have been recorded this turn
func (tn *TurnNarrator) HasEvents() bool {
	return tn.currentSummary != nil && len(tn.currentSummary.Events) > 0
}

// Helper function to convert entity direction to string
func directionToString(dir entity.Direction) string {
	switch dir {
	case entity.DirNorth:
		return "north"
	case entity.DirSouth:
		return "south"
	case entity.DirEast:
		return "east"
	case entity.DirWest:
		return "west"
	case entity.DirNorthEast:
		return "northeast"
	case entity.DirNorthWest:
		return "northwest"
	case entity.DirSouthEast:
		return "southeast"
	case entity.DirSouthWest:
		return "southwest"
	default:
		return "forward"
	}
}
