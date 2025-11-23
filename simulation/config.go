// Package simulation provides configuration for the game simulation rules.
// These rules are loaded from data files so each game can define its own mechanics.
package simulation

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds all simulation rules for a game
type Config struct {
	// Turn system
	TurnSystem TurnConfig `json:"turn_system"`

	// Combat rules
	Combat CombatConfig `json:"combat"`

	// Perception rules
	Perception PerceptionConfig `json:"perception"`

	// Movement rules
	Movement MovementConfig `json:"movement"`
}

// TurnConfig defines how turns and action points work
type TurnConfig struct {
	BaseAP        int    `json:"base_ap"`          // Base AP per turn (e.g., 4)
	APStatBonus   string `json:"ap_stat_bonus"`    // Stat that adds to AP (e.g., "dexterity")
	APFormula     string `json:"ap_formula"`       // Formula for bonus AP (e.g., "(stat - 10) / 4")
	MovementCost  int    `json:"movement_cost"`    // AP cost for basic movement
	WaitRecoverAP int    `json:"wait_recover_ap"`  // AP recovered by waiting (usually 0)
}

// CombatConfig defines combat mechanics
type CombatConfig struct {
	AttackFormula       string `json:"attack_formula"`        // e.g., "d20 + attack + weapon_mod"
	DefenseFormula      string `json:"defense_formula"`       // e.g., "10 + dex_mod + armor"
	CriticalThreshold   int    `json:"critical_threshold"`    // Natural roll for crit (e.g., 20)
	CriticalMultiplier  int    `json:"critical_multiplier"`   // Damage multiplier on crit
	MinimumDamage       int    `json:"minimum_damage"`        // Floor for damage (e.g., 1)

	// Unarmed combat
	UnarmedDamage  string `json:"unarmed_damage"`   // e.g., "1d3"
	UnarmedAPCost  int    `json:"unarmed_ap_cost"`  // AP for unarmed attack
	UnarmedRange   int    `json:"unarmed_range"`    // Range for unarmed (usually 1)

	// Range penalties
	RangePenalties RangePenalties `json:"range_penalties"`
}

// RangePenalties defines accuracy modifiers based on range
type RangePenalties struct {
	PointBlankRange    int `json:"point_blank_range"`    // Range considered point blank
	PointBlankModifier int `json:"point_blank_modifier"` // Modifier at point blank (can be negative)
	OptimalModifier    int `json:"optimal_modifier"`     // Modifier at optimal range
	PerTileOverOptimal int `json:"per_tile_over_optimal"` // Penalty per tile past optimal
}

// PerceptionConfig defines how entities perceive the world
type PerceptionConfig struct {
	// Vision
	BaseVisionRange int `json:"base_vision_range"` // Default vision range in tiles
	VisionConeAngle int `json:"vision_cone_angle"` // Angle of vision cone (degrees)

	// Hearing
	BaseHearingRange int `json:"base_hearing_range"` // Default hearing range

	// Detection modifiers
	MovingPenalty   int `json:"moving_penalty"`    // Penalty to stealth while moving
	SneakingBonus   int `json:"sneaking_bonus"`    // Bonus when actively sneaking
	ShadowBonus     int `json:"shadow_bonus"`      // Bonus when in shadow
	BehindBonus     int `json:"behind_bonus"`      // Bonus when behind target
}

// MovementConfig defines movement mechanics
type MovementConfig struct {
	DiagonalCost      float64 `json:"diagonal_cost"`       // Cost multiplier for diagonal (e.g., 1.5)
	DifficultTerrain  float64 `json:"difficult_terrain"`   // Cost multiplier for difficult terrain
	SneakCostMultiple float64 `json:"sneak_cost_multiple"` // AP multiplier for sneaking
}

// DefaultConfig returns sensible defaults for a fantasy roguelike
func DefaultConfig() *Config {
	return &Config{
		TurnSystem: TurnConfig{
			BaseAP:        4,
			APStatBonus:   "dexterity",
			APFormula:     "(stat - 10) / 4",
			MovementCost:  1,
			WaitRecoverAP: 0,
		},
		Combat: CombatConfig{
			AttackFormula:      "d20 + attack + weapon_mod",
			DefenseFormula:     "10 + dex_mod + armor",
			CriticalThreshold:  20,
			CriticalMultiplier: 2,
			MinimumDamage:      1,
			UnarmedDamage:      "1d3",
			UnarmedAPCost:      2,
			UnarmedRange:       1,
			RangePenalties: RangePenalties{
				PointBlankRange:    1,
				PointBlankModifier: -2,
				OptimalModifier:    0,
				PerTileOverOptimal: -1,
			},
		},
		Perception: PerceptionConfig{
			BaseVisionRange:  8,
			VisionConeAngle:  120,
			BaseHearingRange: 12,
			MovingPenalty:    -2,
			SneakingBonus:    4,
			ShadowBonus:      4,
			BehindBonus:      4,
		},
		Movement: MovementConfig{
			DiagonalCost:      1.5,
			DifficultTerrain:  2.0,
			SneakCostMultiple: 2.0,
		},
	}
}

// LoadConfig loads simulation config from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Return defaults if file doesn't exist
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read simulation config: %w", err)
	}

	config := DefaultConfig() // Start with defaults
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse simulation config: %w", err)
	}

	return config, nil
}

// CalculateAP calculates total AP for an entity based on stats
func (c *Config) CalculateAP(statValue int) int {
	base := c.TurnSystem.BaseAP

	// Simple formula: base + (stat - 10) / 4
	// This gives +1 AP for every 4 points above 10
	if c.TurnSystem.APStatBonus != "" && statValue > 0 {
		bonus := (statValue - 10) / 4
		if bonus < 0 {
			bonus = 0 // Don't penalize low stats
		}
		return base + bonus
	}

	return base
}
