// Package character provides a data-driven character creation and stats system.
// All stat definitions, generation rules, and character templates are defined
// in JSON data files, making it easy to customize for different game systems.
package character

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"chosenoffset.com/outpost9/dice"
)

// StatDefinition defines a single stat/attribute
type StatDefinition struct {
	ID           string               `json:"id"`                     // Unique identifier (e.g., "strength")
	Name         string               `json:"name"`                   // Display name (e.g., "Strength")
	Abbreviation string               `json:"abbreviation,omitempty"` // Short form (e.g., "STR")
	Description  string               `json:"description,omitempty"`  // Description of the stat
	Category     string               `json:"category,omitempty"`     // Category grouping (e.g., "attributes", "skills")
	Generation   *dice.StatExpression `json:"generation,omitempty"`   // How to generate this stat
	BaseValue    int                  `json:"base_value,omitempty"`   // Default/starting value
	MinValue     int                  `json:"min_value,omitempty"`    // Minimum allowed value
	MaxValue     int                  `json:"max_value,omitempty"`    // Maximum allowed value
	DependsOn    []string             `json:"depends_on,omitempty"`   // Stats this depends on for calculation
	Formula      string               `json:"formula,omitempty"`      // Formula for derived stats
	Hidden       bool                 `json:"hidden,omitempty"`       // Whether to show in UI
	Order        int                  `json:"order,omitempty"`        // Display order within category
}

// StatCategory groups related stats together
type StatCategory struct {
	ID          string `json:"id"`                    // Unique identifier
	Name        string `json:"name"`                  // Display name
	Description string `json:"description,omitempty"` // Category description
	Order       int    `json:"order,omitempty"`       // Display order
}

// DerivedStatFormula defines how to calculate a derived stat
type DerivedStatFormula struct {
	StatID  string `json:"stat_id"` // The stat to calculate
	Formula string `json:"formula"` // Formula (e.g., "(strength - 10) / 2")
}

// GenerationMethod defines how character stats are generated overall
type GenerationMethod struct {
	ID          string                          `json:"id"`                    // Unique identifier
	Name        string                          `json:"name"`                  // Display name (e.g., "Standard Array", "Point Buy")
	Description string                          `json:"description,omitempty"` // Method description
	Default     *dice.StatExpression            `json:"default,omitempty"`     // Default expression for all stats
	Overrides   map[string]*dice.StatExpression `json:"overrides,omitempty"`   // Per-stat overrides
	PointPool   int                             `json:"point_pool,omitempty"`  // Points available for point buy
	Array       []int                           `json:"array,omitempty"`       // Fixed array of values to assign
}

// CharacterTemplate defines the complete character creation rules
type CharacterTemplate struct {
	Name              string               `json:"name"`                         // Template name
	Description       string               `json:"description,omitempty"`        // Template description
	Categories        []StatCategory       `json:"categories,omitempty"`         // Stat categories
	Stats             []StatDefinition     `json:"stats"`                        // All stat definitions
	DerivedStats      []DerivedStatFormula `json:"derived_stats,omitempty"`      // Calculated stats
	GenerationMethods []GenerationMethod   `json:"generation_methods,omitempty"` // Available generation methods
	DefaultMethod     string               `json:"default_method,omitempty"`     // Default generation method ID

	// Lookup maps (built after loading)
	statsByID      map[string]*StatDefinition
	methodsByID    map[string]*GenerationMethod
	categoriesByID map[string]*StatCategory
}

// LoadCharacterTemplate loads a character template from a JSON file
func LoadCharacterTemplate(path string) (*CharacterTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read character template: %w", err)
	}

	var template CharacterTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse character template: %w", err)
	}

	// Build lookup maps
	template.buildLookupMaps()

	return &template, nil
}

// LoadCharacterTemplateFromFS loads a character template using an ebiten file system
func LoadCharacterTemplateFromFS(fs ebiten.FileSystem, path string) (*CharacterTemplate, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open character template: %w", err)
	}
	defer file.Close()

	var template CharacterTemplate
	if err := json.NewDecoder(file).Decode(&template); err != nil {
		return nil, fmt.Errorf("failed to parse character template: %w", err)
	}

	template.buildLookupMaps()
	return &template, nil
}

func (t *CharacterTemplate) buildLookupMaps() {
	t.statsByID = make(map[string]*StatDefinition)
	for i := range t.Stats {
		t.statsByID[t.Stats[i].ID] = &t.Stats[i]
	}

	t.methodsByID = make(map[string]*GenerationMethod)
	for i := range t.GenerationMethods {
		t.methodsByID[t.GenerationMethods[i].ID] = &t.GenerationMethods[i]
	}

	t.categoriesByID = make(map[string]*StatCategory)
	for i := range t.Categories {
		t.categoriesByID[t.Categories[i].ID] = &t.Categories[i]
	}
}

// GetStat returns a stat definition by ID
func (t *CharacterTemplate) GetStat(id string) *StatDefinition {
	return t.statsByID[id]
}

// GetMethod returns a generation method by ID
func (t *CharacterTemplate) GetMethod(id string) *GenerationMethod {
	return t.methodsByID[id]
}

// GetCategory returns a category by ID
func (t *CharacterTemplate) GetCategory(id string) *StatCategory {
	return t.categoriesByID[id]
}

// GetStatsByCategory returns all stats in a given category
func (t *CharacterTemplate) GetStatsByCategory(categoryID string) []*StatDefinition {
	var stats []*StatDefinition
	for i := range t.Stats {
		if t.Stats[i].Category == categoryID {
			stats = append(stats, &t.Stats[i])
		}
	}
	return stats
}

// StatValue represents a single stat's current value and metadata
type StatValue struct {
	StatID    string           `json:"stat_id"`
	Value     int              `json:"value"`
	BaseValue int              `json:"base_value,omitempty"` // Original generated/assigned value
	Modifiers []StatModifier   `json:"modifiers,omitempty"`  // Active modifiers
	RollInfo  *dice.RollResult `json:"roll_info,omitempty"`  // How this value was generated
}

// StatModifier represents a temporary or permanent modifier to a stat
type StatModifier struct {
	Source      string `json:"source"`      // What's causing this modifier
	Value       int    `json:"value"`       // The modifier amount
	Description string `json:"description"` // Human-readable description
	Permanent   bool   `json:"permanent"`   // Is this a permanent modifier?
}

// GetTotal returns the total value including all modifiers
func (sv *StatValue) GetTotal() int {
	total := sv.Value
	for _, mod := range sv.Modifiers {
		total += mod.Value
	}
	return total
}

// Character represents a player or NPC character
type Character struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Template   string                `json:"template"` // Template ID this character uses
	Stats      map[string]*StatValue `json:"stats"`    // Stat values by ID
	Level      int                   `json:"level"`
	Experience int                   `json:"experience"`

	// Non-serialized
	templateRef *CharacterTemplate
}

// NewCharacter creates a new character with the given template
func NewCharacter(template *CharacterTemplate) *Character {
	return &Character{
		Template:    template.Name,
		Stats:       make(map[string]*StatValue),
		Level:       1,
		templateRef: template,
	}
}

// GenerateStats generates all stat values using the specified method
func (c *Character) GenerateStats(roller *dice.Roller, methodID string) error {
	if c.templateRef == nil {
		return fmt.Errorf("no template reference set")
	}

	method := c.templateRef.GetMethod(methodID)
	if method == nil {
		// Try to use the default method
		method = c.templateRef.GetMethod(c.templateRef.DefaultMethod)
	}

	for _, stat := range c.templateRef.Stats {
		// Skip derived stats - they'll be calculated later
		if stat.Formula != "" {
			continue
		}

		var expression *dice.StatExpression

		// Check for method-specific override
		if method != nil && method.Overrides != nil {
			if override, ok := method.Overrides[stat.ID]; ok {
				expression = override
			}
		}

		// Fall back to method default
		if expression == nil && method != nil && method.Default != nil {
			expression = method.Default
		}

		// Fall back to stat-specific generation
		if expression == nil && stat.Generation != nil {
			expression = stat.Generation
		}

		// Fall back to base value
		if expression == nil {
			c.Stats[stat.ID] = &StatValue{
				StatID:    stat.ID,
				Value:     stat.BaseValue,
				BaseValue: stat.BaseValue,
			}
			continue
		}

		// Apply min/max from stat definition
		if expression.Min == 0 && stat.MinValue > 0 {
			expression.Min = stat.MinValue
		}
		if expression.Max == 0 && stat.MaxValue > 0 {
			expression.Max = stat.MaxValue
		}

		// Generate the value
		result, err := expression.Evaluate(roller)
		if err != nil {
			return fmt.Errorf("failed to generate stat %s: %w", stat.ID, err)
		}

		c.Stats[stat.ID] = &StatValue{
			StatID:    stat.ID,
			Value:     result.Total,
			BaseValue: result.Total,
			RollInfo:  result,
		}
	}

	// Calculate derived stats
	c.CalculateDerivedStats()

	return nil
}

// SetStat sets a stat value directly
func (c *Character) SetStat(statID string, value int) {
	if c.Stats[statID] == nil {
		c.Stats[statID] = &StatValue{StatID: statID}
	}
	c.Stats[statID].Value = value
	c.Stats[statID].BaseValue = value
}

// GetStat returns a stat value, or nil if not set
func (c *Character) GetStat(statID string) *StatValue {
	return c.Stats[statID]
}

// GetStatTotal returns the total value of a stat (including modifiers)
func (c *Character) GetStatTotal(statID string) int {
	if sv := c.Stats[statID]; sv != nil {
		return sv.GetTotal()
	}
	return 0
}

// AddModifier adds a modifier to a stat
func (c *Character) AddModifier(statID string, modifier StatModifier) {
	if c.Stats[statID] == nil {
		c.Stats[statID] = &StatValue{StatID: statID}
	}
	c.Stats[statID].Modifiers = append(c.Stats[statID].Modifiers, modifier)
}

// CalculateDerivedStats recalculates all derived stats
func (c *Character) CalculateDerivedStats() {
	if c.templateRef == nil {
		return
	}

	for _, formula := range c.templateRef.DerivedStats {
		value := c.evaluateFormula(formula.Formula)
		c.Stats[formula.StatID] = &StatValue{
			StatID:    formula.StatID,
			Value:     value,
			BaseValue: value,
		}
	}
}

// evaluateFormula evaluates a simple formula for derived stats
// Supports: stat references, basic math (+, -, *, /)
// Example: "(strength - 10) / 2"
func (c *Character) evaluateFormula(formula string) int {
	// Simple formula evaluator - replace stat names with values
	// This is a basic implementation; can be expanded for complex formulas

	// For now, support simple patterns like "stat_id" or "(stat_id - N) / M"
	// This could be enhanced with a proper expression parser

	// Try to match common D&D modifier pattern: (stat - 10) / 2
	for _, stat := range c.templateRef.Stats {
		if c.Stats[stat.ID] != nil {
			// Replace stat ID with its value
			// This is simplistic and should be enhanced for production
			_ = formula // TODO: implement proper formula evaluation
		}
	}

	return 0 // Placeholder
}

// GetTemplate returns the character's template reference
func (c *Character) GetTemplate() *CharacterTemplate {
	return c.templateRef
}

// SetTemplate sets the character's template reference
func (c *Character) SetTemplate(template *CharacterTemplate) {
	c.templateRef = template
	c.Template = template.Name
}
