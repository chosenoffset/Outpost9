// Package entity - enemy and NPC definitions loaded from data files
package entity

import (
	"encoding/json"
	"fmt"
	"os"
)

// EntityDefinition defines an enemy or NPC type that can be spawned
type EntityDefinition struct {
	ID          string     `json:"id"`                    // Unique identifier
	Name        string     `json:"name"`                  // Display name
	Type        EntityType `json:"type"`                  // enemy, npc
	Faction     Faction    `json:"faction,omitempty"`     // Faction (defaults based on type)
	Description string     `json:"description,omitempty"` // Lore/description

	// Stats
	HP      int    `json:"hp"`                // Max hit points
	Attack  int    `json:"attack,omitempty"`  // Attack bonus
	Defense int    `json:"defense,omitempty"` // Defense/AC
	Damage  string `json:"damage,omitempty"`  // Damage dice (e.g., "1d6+2")
	Speed   int    `json:"speed,omitempty"`   // Tiles per turn (default 1)

	// Behavior
	AIType     string `json:"ai_type,omitempty"`     // AI behavior type
	AggroRange int    `json:"aggro_range,omitempty"` // Detection range
	CanMove    bool   `json:"can_move"`              // Can this entity move?
	Flying     bool   `json:"flying,omitempty"`      // Can fly over obstacles?

	// Visual
	SpriteName string `json:"sprite_name"` // Sprite in atlas

	// Spawning
	SpawnWeight int      `json:"spawn_weight,omitempty"` // Relative spawn chance
	MinLevel    int      `json:"min_level,omitempty"`    // Minimum dungeon level to appear
	MaxLevel    int      `json:"max_level,omitempty"`    // Maximum dungeon level (0 = no limit)
	Tags        []string `json:"tags,omitempty"`         // Tags for filtering (e.g., "undead", "boss")

	// Loot
	Experience int         `json:"experience,omitempty"` // XP reward
	LootTable  []LootEntry `json:"loot_table,omitempty"` // Items dropped on death
}

// LootEntry defines a possible item drop
type LootEntry struct {
	ItemID   string  `json:"item_id"`
	Chance   float64 `json:"chance"`    // 0.0 to 1.0
	MinCount int     `json:"min_count"` // Minimum quantity
	MaxCount int     `json:"max_count"` // Maximum quantity
}

// EntityLibrary contains all entity definitions for a game
type EntityLibrary struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Enemies     []EntityDefinition `json:"enemies"`
	NPCs        []EntityDefinition `json:"npcs,omitempty"`

	// Lookup maps
	enemiesByID map[string]*EntityDefinition
	npcsByID    map[string]*EntityDefinition
}

// LoadEntityLibrary loads entity definitions from a JSON file
func LoadEntityLibrary(path string) (*EntityLibrary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read entity library: %w", err)
	}

	var library EntityLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		return nil, fmt.Errorf("failed to parse entity library: %w", err)
	}

	library.buildLookupMaps()
	return &library, nil
}

func (lib *EntityLibrary) buildLookupMaps() {
	lib.enemiesByID = make(map[string]*EntityDefinition)
	for i := range lib.Enemies {
		def := &lib.Enemies[i]
		// Set defaults
		if def.Type == "" {
			def.Type = TypeEnemy
		}
		if def.Faction == "" {
			def.Faction = FactionEnemy
		}
		if def.Speed == 0 {
			def.Speed = 1
		}
		if def.Damage == "" {
			def.Damage = "1d4"
		}
		if def.Defense == 0 {
			def.Defense = 10
		}
		lib.enemiesByID[def.ID] = def
	}

	lib.npcsByID = make(map[string]*EntityDefinition)
	for i := range lib.NPCs {
		def := &lib.NPCs[i]
		if def.Type == "" {
			def.Type = TypeNPC
		}
		if def.Faction == "" {
			def.Faction = FactionNeutral
		}
		lib.npcsByID[def.ID] = def
	}
}

// GetEnemy returns an enemy definition by ID
func (lib *EntityLibrary) GetEnemy(id string) *EntityDefinition {
	return lib.enemiesByID[id]
}

// GetNPC returns an NPC definition by ID
func (lib *EntityLibrary) GetNPC(id string) *EntityDefinition {
	return lib.npcsByID[id]
}

// GetAllEnemies returns all enemy definitions
func (lib *EntityLibrary) GetAllEnemies() []*EntityDefinition {
	result := make([]*EntityDefinition, 0, len(lib.Enemies))
	for i := range lib.Enemies {
		result = append(result, &lib.Enemies[i])
	}
	return result
}

// GetEnemiesWithTag returns enemies that have a specific tag
func (lib *EntityLibrary) GetEnemiesWithTag(tag string) []*EntityDefinition {
	var result []*EntityDefinition
	for i := range lib.Enemies {
		def := &lib.Enemies[i]
		for _, t := range def.Tags {
			if t == tag {
				result = append(result, def)
				break
			}
		}
	}
	return result
}

// GetEnemiesForLevel returns enemies appropriate for a dungeon level
func (lib *EntityLibrary) GetEnemiesForLevel(level int) []*EntityDefinition {
	var result []*EntityDefinition
	for i := range lib.Enemies {
		def := &lib.Enemies[i]
		if def.MinLevel <= level && (def.MaxLevel == 0 || def.MaxLevel >= level) {
			result = append(result, def)
		}
	}
	return result
}

// SpawnEntity creates a new entity instance from a definition
func (def *EntityDefinition) SpawnEntity(id string, x, y int) *Entity {
	return &Entity{
		ID:           id,
		Name:         def.Name,
		Type:         def.Type,
		Faction:      def.Faction,
		X:            x,
		Y:            y,
		Speed:        def.Speed,
		CanMove:      def.CanMove,
		Flying:       def.Flying,
		MaxHP:        def.HP,
		CurrentHP:    def.HP,
		Attack:       def.Attack,
		Defense:      def.Defense,
		Damage:       def.Damage,
		SpriteName:   def.SpriteName,
		AIType:       def.AIType,
		AggroRange:   def.AggroRange,
		MaxAP:        1,
		ActionPoints: 1,
	}
}
