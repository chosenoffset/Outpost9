// Package turn provides turn-based game management.
// It handles turn order, action processing, and state transitions.
package turn

import (
	"fmt"
	"math/rand"

	"chosenoffset.com/outpost9/action"
	"chosenoffset.com/outpost9/dice"
	"chosenoffset.com/outpost9/entity"
)

// Phase represents the current phase of a turn
type Phase int

const (
	PhasePlayerInput  Phase = iota // Waiting for player input
	PhasePlayerAction              // Processing player action
	PhaseEnemyTurn                 // Enemies taking their turns
	PhaseEndTurn                   // Cleanup phase
)

// ActionType represents the type of action being taken
type ActionType int

const (
	ActionNone ActionType = iota
	ActionMove
	ActionAttack
	ActionWait
	ActionUseItem
	ActionInteract
)

// Action represents an action to be taken by an entity
type Action struct {
	Type      ActionType
	Actor     *entity.Entity
	Target    *entity.Entity
	TargetX   int // For movement
	TargetY   int
	Direction entity.Direction
	ItemID    string
}

// CombatResult contains the outcome of a combat action
type CombatResult struct {
	Attacker    *entity.Entity
	Defender    *entity.Entity
	Hit         bool
	Damage      int
	AttackRoll  int
	DefenseRoll int
	Critical    bool
	Message     string
}

// Manager handles turn-based gameplay
type Manager struct {
	entities   []*entity.Entity
	player     *entity.Entity
	turnNumber int
	phase      Phase
	currentIdx int // Index of current entity in turn order
	roller     *dice.Roller

	// Action system
	actionLibrary *action.ActionLibrary

	// Callbacks
	OnTurnStart     func(turnNumber int)
	OnTurnEnd       func(turnNumber int)
	OnEntityTurn    func(e *entity.Entity)
	OnCombat        func(result *CombatResult)
	OnEntityDeath   func(e *entity.Entity)
	OnMessage       func(msg string)
	OnSceneUpdate   func() // Called when scene should be re-described
	OnAPChanged     func(current, max int) // Called when player AP changes

	// Map interaction
	IsWalkable  func(x, y int) bool
	GetEntityAt func(x, y int) *entity.Entity
}

// NewManager creates a new turn manager
func NewManager(rng *rand.Rand) *Manager {
	return &Manager{
		entities:      make([]*entity.Entity, 0),
		turnNumber:    0,
		phase:         PhasePlayerInput,
		roller:        dice.NewRoller(rng),
		actionLibrary: action.DefaultLibrary(),
	}
}

// SetActionLibrary sets the action library (merge with defaults)
func (m *Manager) SetActionLibrary(lib *action.ActionLibrary) {
	if m.actionLibrary == nil {
		m.actionLibrary = action.DefaultLibrary()
	}
	if lib != nil {
		m.actionLibrary.MergeLibrary(lib)
	}
}

// GetActionLibrary returns the action library
func (m *Manager) GetActionLibrary() *action.ActionLibrary {
	return m.actionLibrary
}

// SetPlayer sets the player entity
func (m *Manager) SetPlayer(player *entity.Entity) {
	m.player = player
	// Ensure player is in entities list
	found := false
	for _, e := range m.entities {
		if e == player {
			found = true
			break
		}
	}
	if !found {
		m.entities = append(m.entities, player)
	}
}

// AddEntity adds an entity to the turn manager
func (m *Manager) AddEntity(e *entity.Entity) {
	m.entities = append(m.entities, e)
}

// RemoveEntity removes an entity from the turn manager
func (m *Manager) RemoveEntity(e *entity.Entity) {
	for i, ent := range m.entities {
		if ent == e {
			m.entities = append(m.entities[:i], m.entities[i+1:]...)
			return
		}
	}
}

// GetEntities returns all entities
func (m *Manager) GetEntities() []*entity.Entity {
	return m.entities
}

// GetLivingEntities returns all living entities
func (m *Manager) GetLivingEntities() []*entity.Entity {
	var living []*entity.Entity
	for _, e := range m.entities {
		if e.IsAlive() {
			living = append(living, e)
		}
	}
	return living
}

// GetEnemies returns all living enemy entities
func (m *Manager) GetEnemies() []*entity.Entity {
	var enemies []*entity.Entity
	for _, e := range m.entities {
		if e.IsAlive() && e.Faction == entity.FactionEnemy {
			enemies = append(enemies, e)
		}
	}
	return enemies
}

// GetPhase returns the current phase
func (m *Manager) GetPhase() Phase {
	return m.phase
}

// GetTurnNumber returns the current turn number
func (m *Manager) GetTurnNumber() int {
	return m.turnNumber
}

// IsPlayerTurn returns true if it's the player's turn to input
func (m *Manager) IsPlayerTurn() bool {
	return m.phase == PhasePlayerInput
}

// StartNewTurn begins a new turn
func (m *Manager) StartNewTurn() {
	m.turnNumber++

	// Reset all entities for the new turn
	for _, e := range m.entities {
		e.StartTurn()
	}

	if m.OnTurnStart != nil {
		m.OnTurnStart(m.turnNumber)
	}

	m.phase = PhasePlayerInput
}

// ProcessPlayerAction handles a player action (partial AP spending)
// Returns true if action was successful
// Does NOT automatically end the turn - player can keep acting until out of AP
func (m *Manager) ProcessPlayerAction(act Action) bool {
	if m.phase != PhasePlayerInput {
		return false
	}

	if m.player == nil || !m.player.CanAct() {
		return false
	}

	m.phase = PhasePlayerAction

	// Execute the action
	success := m.executeAction(act)

	if success {
		// Notify AP change
		if m.OnAPChanged != nil {
			m.OnAPChanged(m.player.ActionPoints, m.player.MaxAP)
		}

		// Notify scene update
		if m.OnSceneUpdate != nil {
			m.OnSceneUpdate()
		}

		// Check if player is out of AP
		if m.player.ActionPoints <= 0 {
			m.EndPlayerTurn()
		} else {
			// Player can still act, go back to input phase
			m.phase = PhasePlayerInput
		}
	} else {
		// Action failed, go back to player input
		m.phase = PhasePlayerInput
	}

	return success
}

// EndPlayerTurn manually ends the player's turn (even with AP remaining)
func (m *Manager) EndPlayerTurn() {
	if m.player == nil {
		return
	}

	m.player.EndTurn()
	m.phase = PhaseEnemyTurn
	m.processEnemyTurns()
	m.endTurn()
}

// ProcessDataAction handles an action from the action library
func (m *Manager) ProcessDataAction(act *action.Action, dir entity.Direction, targetX, targetY int) bool {
	if m.phase != PhasePlayerInput || m.player == nil {
		return false
	}

	// Check if player can afford the AP
	if !m.player.CanAffordAP(act.APCost) {
		if m.OnMessage != nil {
			m.OnMessage(fmt.Sprintf("Not enough AP! Need %d, have %d", act.APCost, m.player.ActionPoints))
		}
		return false
	}

	m.phase = PhasePlayerAction

	// Execute based on action type
	success := false
	switch act.Category {
	case action.CategoryMovement:
		success = m.executeDataMove(act, dir)
	case action.CategoryCombat:
		success = m.executeDataAttack(act, dir, targetX, targetY)
	case action.CategoryUtility:
		success = m.executeDataUtility(act)
	default:
		// Generic action execution
		success = m.executeGenericAction(act, dir, targetX, targetY)
	}

	if success {
		// Spend the AP
		m.player.SpendAP(act.APCost)

		// Notify AP change
		if m.OnAPChanged != nil {
			m.OnAPChanged(m.player.ActionPoints, m.player.MaxAP)
		}

		// Notify scene update
		if m.OnSceneUpdate != nil {
			m.OnSceneUpdate()
		}

		// Check if player is out of AP
		if m.player.ActionPoints <= 0 {
			m.EndPlayerTurn()
		} else {
			m.phase = PhasePlayerInput
		}
	} else {
		m.phase = PhasePlayerInput
	}

	return success
}

// executeDataMove handles movement actions from the action library
func (m *Manager) executeDataMove(act *action.Action, dir entity.Direction) bool {
	if dir == entity.DirNone {
		return false
	}

	dx, dy := dir.Delta()
	newX := m.player.X + dx
	newY := m.player.Y + dy

	// Check if destination is walkable
	if m.IsWalkable != nil && !m.IsWalkable(newX, newY) {
		if m.OnMessage != nil {
			m.OnMessage("You can't move there.")
		}
		return false
	}

	// Check if another entity is there
	if m.GetEntityAt != nil {
		if blocker := m.GetEntityAt(newX, newY); blocker != nil && blocker.IsAlive() {
			// If hostile, convert to attack
			if m.player.IsHostileTo(blocker) {
				if m.OnMessage != nil {
					m.OnMessage("An enemy blocks your path!")
				}
				return false // Don't auto-attack, let player choose
			}
			if m.OnMessage != nil {
				m.OnMessage("Someone is in the way.")
			}
			return false
		}
	}

	// Execute the move
	m.player.X = newX
	m.player.Y = newY
	m.player.Facing = dir

	if m.OnMessage != nil {
		dirName := directionName(dir)
		m.OnMessage(fmt.Sprintf("You move %s.", dirName))
	}

	return true
}

// executeDataAttack handles combat actions from the action library
func (m *Manager) executeDataAttack(act *action.Action, dir entity.Direction, targetX, targetY int) bool {
	var target *entity.Entity

	// Find target based on direction or coordinates
	if dir != entity.DirNone {
		dx, dy := dir.Delta()
		targetX = m.player.X + dx
		targetY = m.player.Y + dy
	}

	if m.GetEntityAt != nil {
		target = m.GetEntityAt(targetX, targetY)
	}

	if target == nil || !target.IsAlive() {
		if m.OnMessage != nil {
			m.OnMessage("No target there.")
		}
		return false
	}

	// Check range
	dist := m.player.DistanceToPoint(targetX, targetY)
	if dist > act.Targeting.Range && act.Targeting.Range > 0 {
		if m.OnMessage != nil {
			m.OnMessage("Target is out of range!")
		}
		return false
	}

	// Execute attack using existing combat system
	oldAction := Action{
		Type:   ActionAttack,
		Actor:  m.player,
		Target: target,
	}

	return m.executeAttack(oldAction)
}

// executeDataUtility handles utility actions (wait, etc.)
func (m *Manager) executeDataUtility(act *action.Action) bool {
	// Check for specific utility actions
	for _, effect := range act.Effects {
		switch effect.Type {
		case "pass_time":
			if m.OnMessage != nil {
				m.OnMessage("You wait...")
			}
			return true
		}
	}

	return true
}

// executeGenericAction handles other action types
func (m *Manager) executeGenericAction(act *action.Action, dir entity.Direction, targetX, targetY int) bool {
	// Apply effects
	for _, effect := range act.Effects {
		switch effect.Type {
		case "pass_time":
			// Do nothing
		case "move":
			// Already handled by movement
		default:
			// Log unhandled effect
		}
	}

	if m.OnMessage != nil {
		m.OnMessage(fmt.Sprintf("You %s.", act.Name))
	}

	return true
}

// GetAvailableActions returns actions the player can currently take
func (m *Manager) GetAvailableActions() []*action.Action {
	if m.actionLibrary == nil || m.player == nil {
		return nil
	}

	var available []*action.Action
	for _, act := range m.actionLibrary.GetAllActions() {
		if m.player.CanAffordAP(act.APCost) {
			available = append(available, act)
		}
	}
	return available
}

// Helper function for direction names
func directionName(dir entity.Direction) string {
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
		return "somewhere"
	}
}

// executeAction executes a single action (spends AP)
func (m *Manager) executeAction(act Action) bool {
	// Determine AP cost based on action type
	apCost := 1 // Default cost
	switch act.Type {
	case ActionMove:
		apCost = 1
	case ActionAttack:
		apCost = 2 // Attacks cost more AP
	case ActionWait:
		apCost = 1
	case ActionInteract:
		apCost = 1
	}

	// Check if actor can afford the AP
	if act.Actor != nil && !act.Actor.CanAffordAP(apCost) {
		return false
	}

	// Execute the action
	success := false
	switch act.Type {
	case ActionMove:
		success = m.executeMove(act)
	case ActionAttack:
		success = m.executeAttack(act)
	case ActionWait:
		success = true // Waiting always succeeds
	case ActionInteract:
		success = true // Interaction handled elsewhere
	}

	// Spend AP on success
	if success && act.Actor != nil {
		act.Actor.SpendAP(apCost)
	}

	return success
}

// executeMove handles movement actions
func (m *Manager) executeMove(action Action) bool {
	actor := action.Actor
	if actor == nil {
		return false
	}

	dx, dy := action.Direction.Delta()
	newX := actor.X + dx
	newY := actor.Y + dy

	// Check if destination is walkable
	if m.IsWalkable != nil && !m.IsWalkable(newX, newY) {
		return false
	}

	// Check if another entity is there
	if m.GetEntityAt != nil {
		if blocker := m.GetEntityAt(newX, newY); blocker != nil && blocker.IsAlive() {
			// If hostile, convert to attack
			if actor.IsHostileTo(blocker) {
				action.Type = ActionAttack
				action.Target = blocker
				return m.executeAttack(action)
			}
			// Can't move through friendly entities
			return false
		}
	}

	// Execute the move
	actor.X = newX
	actor.Y = newY
	actor.Facing = action.Direction

	return true
}

// executeAttack handles attack actions
func (m *Manager) executeAttack(action Action) bool {
	attacker := action.Actor
	defender := action.Target

	if attacker == nil || defender == nil {
		return false
	}

	// Check range (must be adjacent for melee)
	if !attacker.IsAdjacent(defender) {
		if m.OnMessage != nil {
			m.OnMessage("Target is out of range!")
		}
		return false
	}

	// Roll attack: d20 + attack bonus vs defense
	attackRoll, _ := m.roller.Roll("1d20")
	totalAttack := attackRoll.Total + attacker.Attack

	result := &CombatResult{
		Attacker:    attacker,
		Defender:    defender,
		AttackRoll:  attackRoll.Total,
		DefenseRoll: defender.Defense,
	}

	// Check for critical hit (natural 20)
	result.Critical = attackRoll.Total == 20

	// Hit if attack >= defense, or critical
	result.Hit = totalAttack >= defender.Defense || result.Critical

	if result.Hit {
		// Roll damage
		damage := attacker.RollDamage(m.roller)
		if result.Critical {
			damage *= 2 // Double damage on crit
		}

		// Apply minimum 1 damage
		if damage < 1 {
			damage = 1
		}

		result.Damage = damage
		defender.TakeDamage(damage)

		if result.Critical {
			result.Message = attacker.Name + " critically hits " + defender.Name + "!"
		} else {
			result.Message = attacker.Name + " hits " + defender.Name + "."
		}

		// Check for death
		if !defender.IsAlive() {
			result.Message += " " + defender.Name + " is defeated!"
			if m.OnEntityDeath != nil {
				m.OnEntityDeath(defender)
			}
		}
	} else {
		result.Message = attacker.Name + " misses " + defender.Name + "."
	}

	if m.OnCombat != nil {
		m.OnCombat(result)
	}
	if m.OnMessage != nil {
		m.OnMessage(result.Message)
	}

	return true
}

// processEnemyTurns handles all enemy actions
func (m *Manager) processEnemyTurns() {
	for _, e := range m.entities {
		if e.Type == entity.TypeEnemy && e.IsAlive() && e.CanAct() {
			if m.OnEntityTurn != nil {
				m.OnEntityTurn(e)
			}
			m.processEnemyAI(e)
			e.EndTurn()
		}
	}
}

// processEnemyAI determines and executes an enemy's action
func (m *Manager) processEnemyAI(e *entity.Entity) {
	if m.player == nil || !m.player.IsAlive() {
		return
	}

	// Simple AI: move toward player and attack if adjacent
	if e.IsAdjacent(m.player) {
		// Attack the player
		action := Action{
			Type:   ActionAttack,
			Actor:  e,
			Target: m.player,
		}
		m.executeAttack(action)
	} else if e.CanMove {
		// Move toward player
		dir := m.getDirectionToward(e, m.player)
		if dir != entity.DirNone {
			action := Action{
				Type:      ActionMove,
				Actor:     e,
				Direction: dir,
			}
			m.executeMove(action)
		}
	}
}

// getDirectionToward calculates the direction from one entity toward another
func (m *Manager) getDirectionToward(from, to *entity.Entity) entity.Direction {
	dx := to.X - from.X
	dy := to.Y - from.Y

	// Prefer cardinal directions, try horizontal first
	if dx > 0 {
		if m.canMoveInDirection(from, entity.DirEast) {
			return entity.DirEast
		}
	} else if dx < 0 {
		if m.canMoveInDirection(from, entity.DirWest) {
			return entity.DirWest
		}
	}

	if dy > 0 {
		if m.canMoveInDirection(from, entity.DirSouth) {
			return entity.DirSouth
		}
	} else if dy < 0 {
		if m.canMoveInDirection(from, entity.DirNorth) {
			return entity.DirNorth
		}
	}

	// Try vertical if horizontal failed
	if dy > 0 && m.canMoveInDirection(from, entity.DirSouth) {
		return entity.DirSouth
	}
	if dy < 0 && m.canMoveInDirection(from, entity.DirNorth) {
		return entity.DirNorth
	}
	if dx > 0 && m.canMoveInDirection(from, entity.DirEast) {
		return entity.DirEast
	}
	if dx < 0 && m.canMoveInDirection(from, entity.DirWest) {
		return entity.DirWest
	}

	return entity.DirNone
}

// canMoveInDirection checks if an entity can move in a direction
func (m *Manager) canMoveInDirection(e *entity.Entity, dir entity.Direction) bool {
	dx, dy := dir.Delta()
	newX := e.X + dx
	newY := e.Y + dy

	if m.IsWalkable != nil && !m.IsWalkable(newX, newY) {
		return false
	}

	if m.GetEntityAt != nil {
		if blocker := m.GetEntityAt(newX, newY); blocker != nil && blocker.IsAlive() {
			return false
		}
	}

	return true
}

// endTurn finishes the current turn
func (m *Manager) endTurn() {
	m.phase = PhaseEndTurn

	if m.OnTurnEnd != nil {
		m.OnTurnEnd(m.turnNumber)
	}

	// Start a new turn
	m.StartNewTurn()
}

// GetEntityAtPosition finds an entity at the given position
func (m *Manager) GetEntityAtPosition(x, y int) *entity.Entity {
	for _, e := range m.entities {
		if e.X == x && e.Y == y && e.IsAlive() {
			return e
		}
	}
	return nil
}
