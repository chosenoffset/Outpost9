// Package entity provides a turn-based entity system for players, enemies, and NPCs.
// All entities exist on a grid and take turns moving and acting.
package entity

import (
	"chosenoffset.com/outpost9/character"
	"chosenoffset.com/outpost9/dice"
)

// EntityType identifies the kind of entity
type EntityType string

const (
	TypePlayer EntityType = "player"
	TypeEnemy  EntityType = "enemy"
	TypeNPC    EntityType = "npc"
)

// Faction determines hostility relationships
type Faction string

const (
	FactionPlayer  Faction = "player"
	FactionEnemy   Faction = "enemy"
	FactionNeutral Faction = "neutral"
)

// Direction represents cardinal directions for movement/facing
type Direction int

const (
	DirNone Direction = iota
	DirNorth
	DirSouth
	DirEast
	DirWest
	DirNorthEast
	DirNorthWest
	DirSouthEast
	DirSouthWest
)

// DirectionDelta returns the x,y delta for a direction
func (d Direction) Delta() (int, int) {
	switch d {
	case DirNorth:
		return 0, -1
	case DirSouth:
		return 0, 1
	case DirEast:
		return 1, 0
	case DirWest:
		return -1, 0
	case DirNorthEast:
		return 1, -1
	case DirNorthWest:
		return -1, -1
	case DirSouthEast:
		return 1, 1
	case DirSouthWest:
		return -1, 1
	default:
		return 0, 0
	}
}

// Entity represents any creature or character in the game world
type Entity struct {
	ID      string     // Unique identifier
	Name    string     // Display name
	Type    EntityType // player, enemy, npc
	Faction Faction    // For determining hostility

	// Position (grid coordinates)
	X, Y int

	// Movement
	Facing  Direction // Direction entity is facing
	Speed   int       // Tiles per turn (usually 1)
	CanMove bool      // Can this entity move?
	Flying  bool      // Can fly over obstacles?

	// Combat stats (can be overridden by Character)
	MaxHP     int
	CurrentHP int
	Attack    int    // Base attack bonus
	Defense   int    // Base defense/AC
	Damage    string // Damage dice expression (e.g., "1d6+2")

	// Visual
	SpriteName string // Name of sprite in atlas

	// AI (for enemies/NPCs)
	AIType     string // AI behavior type
	AggroRange int    // Range at which entity becomes hostile

	// Turn state
	HasActed     bool // Has this entity acted this turn?
	ActionPoints int  // AP for this turn (usually 1 for move OR attack)
	MaxAP        int  // Max AP per turn

	// Link to full character (for player or complex NPCs)
	Character *character.Character
}

// NewEntity creates a basic entity
func NewEntity(id, name string, entityType EntityType) *Entity {
	return &Entity{
		ID:           id,
		Name:         name,
		Type:         entityType,
		Faction:      FactionNeutral,
		Speed:        1,
		CanMove:      true,
		MaxHP:        10,
		CurrentHP:    10,
		Attack:       0,
		Defense:      10,
		Damage:       "1d4",
		MaxAP:        1,
		ActionPoints: 1,
	}
}

// NewPlayerEntity creates a player entity from a character
func NewPlayerEntity(char *character.Character, x, y int) *Entity {
	e := &Entity{
		ID:           "player",
		Name:         char.Name,
		Type:         TypePlayer,
		Faction:      FactionPlayer,
		X:            x,
		Y:            y,
		Speed:        1,
		CanMove:      true,
		MaxAP:        1,
		ActionPoints: 1,
		SpriteName:   "player_idle",
		Character:    char,
	}

	// Pull stats from character if available
	if char != nil {
		// HP from Constitution
		if con := char.GetStatTotal("constitution"); con > 0 {
			conMod := (con - 10) / 2
			e.MaxHP = 10 + conMod
			e.CurrentHP = e.MaxHP
		}
		// Defense from Dexterity
		if dex := char.GetStatTotal("dexterity"); dex > 0 {
			dexMod := (dex - 10) / 2
			e.Defense = 10 + dexMod
		}
		// Attack from Strength
		if str := char.GetStatTotal("strength"); str > 0 {
			strMod := (str - 10) / 2
			e.Attack = strMod
			if strMod >= 0 {
				e.Damage = "1d6+" + string(rune('0'+strMod))
			} else {
				e.Damage = "1d6" + string(rune('0'+strMod))
			}
		}
	} else {
		e.MaxHP = 20
		e.CurrentHP = 20
		e.Defense = 10
		e.Attack = 0
		e.Damage = "1d6"
	}

	return e
}

// IsAlive returns true if the entity has HP remaining
func (e *Entity) IsAlive() bool {
	return e.CurrentHP > 0
}

// TakeDamage applies damage to the entity
func (e *Entity) TakeDamage(amount int) {
	e.CurrentHP -= amount
	if e.CurrentHP < 0 {
		e.CurrentHP = 0
	}
}

// Heal restores HP to the entity
func (e *Entity) Heal(amount int) {
	e.CurrentHP += amount
	if e.CurrentHP > e.MaxHP {
		e.CurrentHP = e.MaxHP
	}
}

// StartTurn resets the entity for a new turn
func (e *Entity) StartTurn() {
	e.HasActed = false
	e.ActionPoints = e.MaxAP
}

// EndTurn marks the entity as having finished their turn
func (e *Entity) EndTurn() {
	e.HasActed = true
	e.ActionPoints = 0
}

// CanAct returns true if the entity can still act this turn
func (e *Entity) CanAct() bool {
	return e.IsAlive() && e.ActionPoints > 0
}

// DistanceTo calculates Manhattan distance to another entity
func (e *Entity) DistanceTo(other *Entity) int {
	dx := e.X - other.X
	dy := e.Y - other.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// IsAdjacent returns true if entities are next to each other
func (e *Entity) IsAdjacent(other *Entity) bool {
	return e.DistanceTo(other) == 1
}

// IsHostileTo returns true if this entity is hostile to another
func (e *Entity) IsHostileTo(other *Entity) bool {
	if e.Faction == FactionNeutral || other.Faction == FactionNeutral {
		return false
	}
	return e.Faction != other.Faction
}

// RollDamage rolls the entity's damage dice
func (e *Entity) RollDamage(roller *dice.Roller) int {
	result, err := roller.Roll(e.Damage)
	if err != nil {
		return 1 // Fallback minimum damage
	}
	return result.Total
}
