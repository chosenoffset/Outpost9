// Package turn provides turn-based game management.
// It handles turn order, action processing, and state transitions.
package turn

import (
	"math/rand"

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

	// Callbacks
	OnTurnStart   func(turnNumber int)
	OnTurnEnd     func(turnNumber int)
	OnEntityTurn  func(e *entity.Entity)
	OnCombat      func(result *CombatResult)
	OnEntityDeath func(e *entity.Entity)
	OnMessage     func(msg string)

	// Map interaction
	IsWalkable  func(x, y int) bool
	GetEntityAt func(x, y int) *entity.Entity
}

// NewManager creates a new turn manager
func NewManager(rng *rand.Rand) *Manager {
	return &Manager{
		entities:   make([]*entity.Entity, 0),
		turnNumber: 0,
		phase:      PhasePlayerInput,
		roller:     dice.NewRoller(rng),
	}
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

// ProcessPlayerAction handles a player action and advances the turn
func (m *Manager) ProcessPlayerAction(action Action) bool {
	if m.phase != PhasePlayerInput {
		return false
	}

	if m.player == nil || !m.player.CanAct() {
		return false
	}

	m.phase = PhasePlayerAction

	// Execute the action
	success := m.executeAction(action)

	if success {
		m.player.EndTurn()
		// After player acts, enemies get their turn
		m.phase = PhaseEnemyTurn
		m.processEnemyTurns()
		m.endTurn()
	} else {
		// Action failed, go back to player input
		m.phase = PhasePlayerInput
	}

	return success
}

// executeAction executes a single action
func (m *Manager) executeAction(action Action) bool {
	switch action.Type {
	case ActionMove:
		return m.executeMove(action)
	case ActionAttack:
		return m.executeAttack(action)
	case ActionWait:
		return true // Waiting always succeeds
	case ActionInteract:
		return true // Interaction handled elsewhere
	default:
		return false
	}
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
