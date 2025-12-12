// Package narrative prose.go provides dynamic prose generation for immersive gameplay
package narrative

import (
	"fmt"
	"math/rand"
	"strings"

	"chosenoffset.com/outpost9/internal/entity"
	"chosenoffset.com/outpost9/internal/world/furnishing"
)

// PositionalRelation describes a player's position relative to an object
type PositionalRelation int

const (
	RelNone PositionalRelation = iota
	RelBeside
	RelBehind    // Object is between player and enemies
	RelInFrontOf // Player is between object and enemies
	RelNear
)

// FurnishingContext describes a nearby furnishing and the player's relation to it
type FurnishingContext struct {
	Furnishing *furnishing.PlacedFurnishing
	Relation   PositionalRelation
	Distance   int
	Direction  string // "north", "south", etc.
}

// EnemyTurnAction describes what an enemy did during its turn
type EnemyTurnAction struct {
	Entity       *entity.Entity
	ActionType   string // "moved", "attacked", "approached", "patrolled", "waited"
	Direction    string // Direction of movement
	Distance     int    // Tiles moved
	TargetName   string // If attacking, who
	Damage       int    // If attacking, how much
	IsApproaching bool  // Moving toward player
	IsRetreating bool   // Moving away from player
}

// ProseContext contains all the context needed to generate dynamic prose
type ProseContext struct {
	// Player context
	PlayerAction     string // What the player just did
	PlayerDirection  string // Direction player moved/faced
	PlayerPosition   struct{ X, Y int }
	PlayerHP         int
	PlayerMaxHP      int
	PlayerIsHidden   bool
	PlayerIsSneaking bool

	// Furnishing context
	NearbyFurnishings []FurnishingContext
	CoverFurnishing   *FurnishingContext // If player is behind cover

	// Enemy context
	EnemyActions      []EnemyTurnAction
	VisibleEnemies    []*EntityInfo
	NearbyEnemyCount  int
	ClosestEnemyDist  int
	EnemiesApproaching bool
	EnemiesRetreating  bool

	// Room context
	RoomName        string
	RoomAtmosphere  string
	IsFirstVisit    bool
	RoomHasEnemies  bool
	RoomWasCleared  bool

	// Turn info
	TurnNumber int
}

// ProseGenerator creates varied, dynamic prose from game context
type ProseGenerator struct {
	rng *rand.Rand

	// Text variation pools
	movementVerbs     []string
	coverVerbs        []string
	enemyMoveVerbs    []string
	enemyApproachVerbs []string
	sensoryPhrases    map[string][]string // By enemy type
	transitionPhrases []string
	promptPhrases     []string
}

// NewProseGenerator creates a new prose generator
func NewProseGenerator(seed int64) *ProseGenerator {
	pg := &ProseGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
	pg.initTextPools()
	return pg
}

// initTextPools initializes the varied text pools
func (pg *ProseGenerator) initTextPools() {
	// Movement verbs for when player moves to cover
	pg.movementVerbs = []string{
		"move", "slip", "duck", "shift", "position yourself",
		"step", "edge", "slide", "press yourself",
	}

	// Verbs for taking cover
	pg.coverVerbs = []string{
		"hunching down behind", "crouching behind", "ducking behind",
		"pressing against", "taking cover behind", "positioning yourself behind",
		"slipping behind", "concealing yourself behind",
	}

	// Verbs for enemy movement
	pg.enemyMoveVerbs = []string{
		"moves", "shuffles", "stalks", "walks", "trudges",
		"creeps", "ambles", "advances", "proceeds",
	}

	// Verbs for enemies approaching
	pg.enemyApproachVerbs = []string{
		"approaches", "draws closer", "advances toward you",
		"closes the distance", "moves in your direction",
		"heads your way", "comes closer",
	}

	// Sensory phrases by enemy type
	pg.sensoryPhrases = map[string][]string{
		"rat": {
			"The acrid stench of their foulness reaches your nose.",
			"You catch a whiff of their musky, unpleasant odor.",
			"The scurrying of tiny clawed feet echoes in the corridor.",
			"A foul smell wafts from their direction.",
			"Their squeaking and chittering fills the air.",
		},
		"goblin": {
			"The goblin's guttural muttering reaches your ears.",
			"You hear harsh, guttural breathing.",
			"The stench of unwashed goblin assaults your senses.",
			"Crude snickering echoes through the chamber.",
			"The clanking of poorly-maintained gear accompanies its movements.",
		},
		"skeleton": {
			"Bones clatter with each movement.",
			"An unnatural cold emanates from the undead creature.",
			"The hollow eye sockets seem to scan the darkness.",
			"Rattling bones echo ominously.",
		},
		"zombie": {
			"The smell of rotting flesh precedes it.",
			"A low, mindless groan escapes its throat.",
			"The shuffling of decayed feet scrapes against stone.",
			"The stench of death hangs heavy in the air.",
		},
		"spider": {
			"Clicking mandibles echo eerily.",
			"Multiple legs tap against the stone floor.",
			"You feel the creeping sensation of being watched.",
			"Silken threads glint faintly in the dim light.",
		},
		"orc": {
			"Heavy footfalls shake the ground slightly.",
			"A brutish grunt echoes through the chamber.",
			"The smell of blood and sweat accompanies the creature.",
			"Crude weapons scrape against armor.",
		},
		"default": {
			"You sense danger nearby.",
			"Something stirs in the shadows.",
			"The air feels thick with menace.",
			"An uneasy feeling settles over you.",
		},
	}

	// Transition phrases between sections
	pg.transitionPhrases = []string{
		"", // Sometimes no transition
		"Meanwhile, ",
		"As you do, ",
		"From your position, you observe ",
		"You watch as ",
		"From here, you see ",
	}

	// Prompt phrases
	pg.promptPhrases = []string{
		"What do you do?",
		"Your move.",
		"How do you respond?",
		"What's your next move?",
		"The choice is yours.",
		"What will you do?",
		"You must decide.",
	}
}

// pickRandom returns a random element from a string slice
func (pg *ProseGenerator) pickRandom(options []string) string {
	if len(options) == 0 {
		return ""
	}
	return options[pg.rng.Intn(len(options))]
}

// getSensoryPhrase returns a sensory phrase for an enemy type
func (pg *ProseGenerator) getSensoryPhrase(enemyType string) string {
	// Normalize enemy type to lowercase for matching
	normalizedType := strings.ToLower(enemyType)

	// Check for partial matches
	for key, phrases := range pg.sensoryPhrases {
		if strings.Contains(normalizedType, key) {
			return pg.pickRandom(phrases)
		}
	}

	// Default sensory phrase
	return pg.pickRandom(pg.sensoryPhrases["default"])
}

// GenerateProse creates a dynamic prose paragraph from the context
func (pg *ProseGenerator) GenerateProse(ctx *ProseContext) string {
	var parts []string

	// 1. Player action description
	if playerDesc := pg.describePlayerAction(ctx); playerDesc != "" {
		parts = append(parts, playerDesc)
	}

	// 2. Enemy actions description
	if enemyDesc := pg.describeEnemyActions(ctx); enemyDesc != "" {
		parts = append(parts, enemyDesc)
	}

	// 3. Sensory details
	if sensory := pg.describeSensoryDetails(ctx); sensory != "" {
		parts = append(parts, sensory)
	}

	// 4. Action prompt
	parts = append(parts, pg.pickRandom(pg.promptPhrases))

	return strings.Join(parts, " ")
}

// describePlayerAction generates prose for what the player did
func (pg *ProseGenerator) describePlayerAction(ctx *ProseContext) string {
	// If player took cover behind something
	if ctx.CoverFurnishing != nil {
		coverVerb := pg.pickRandom(pg.coverVerbs)
		objName := pg.getFurnishingDisplayName(ctx.CoverFurnishing.Furnishing)
		return fmt.Sprintf("As you %s the %s,", coverVerb, objName)
	}

	// If player moved near a furnishing
	for _, fc := range ctx.NearbyFurnishings {
		if fc.Distance <= 1 {
			objName := pg.getFurnishingDisplayName(fc.Furnishing)
			switch fc.Relation {
			case RelBeside:
				return fmt.Sprintf("Standing beside the %s,", objName)
			case RelNear:
				return fmt.Sprintf("Near the %s,", objName)
			}
		}
	}

	// Generic movement
	if ctx.PlayerAction == "move" && ctx.PlayerDirection != "" {
		verb := pg.pickRandom(pg.movementVerbs)
		return fmt.Sprintf("You %s %s.", verb, ctx.PlayerDirection)
	}

	return ""
}

// describeEnemyActions generates prose for what enemies did
func (pg *ProseGenerator) describeEnemyActions(ctx *ProseContext) string {
	if len(ctx.EnemyActions) == 0 && len(ctx.VisibleEnemies) == 0 {
		return ""
	}

	var descriptions []string

	// Group enemies by their actions
	movers := []EnemyTurnAction{}
	approachers := []EnemyTurnAction{}

	for _, action := range ctx.EnemyActions {
		if action.IsApproaching {
			approachers = append(approachers, action)
		} else if action.ActionType == "moved" {
			movers = append(movers, action)
		}
	}

	// Describe enemies that moved (not approaching)
	if len(movers) > 0 {
		desc := pg.describeEnemyMovement(movers)
		if desc != "" {
			descriptions = append(descriptions, desc)
		}
	}

	// Describe enemies approaching (more urgent)
	if len(approachers) > 0 {
		desc := pg.describeEnemyApproach(approachers)
		if desc != "" {
			descriptions = append(descriptions, desc)
		}
	}

	// If no actions but enemies visible, describe their presence
	if len(descriptions) == 0 && len(ctx.VisibleEnemies) > 0 {
		desc := pg.describeVisibleEnemies(ctx.VisibleEnemies)
		if desc != "" {
			descriptions = append(descriptions, desc)
		}
	}

	if len(descriptions) == 0 {
		return ""
	}

	// Add transition
	transition := pg.pickRandom(pg.transitionPhrases)
	return transition + strings.Join(descriptions, " ")
}

// describeEnemyMovement describes enemies moving (not toward player)
func (pg *ProseGenerator) describeEnemyMovement(actions []EnemyTurnAction) string {
	if len(actions) == 0 {
		return ""
	}

	if len(actions) == 1 {
		action := actions[0]
		name := pg.getEntityDisplayName(action.Entity)
		verb := pg.pickRandom(pg.enemyMoveVerbs)

		if action.Direction != "" {
			return fmt.Sprintf("the %s %s %s.", name, verb, action.Direction)
		}
		return fmt.Sprintf("the %s %s about.", name, verb)
	}

	// Multiple enemies - check if they're following each other
	names := []string{}
	for _, action := range actions {
		names = append(names, pg.getEntityDisplayName(action.Entity))
	}

	if len(names) == 2 {
		return fmt.Sprintf("you see the %s continue on, followed by the %s.", names[0], names[1])
	}

	// Generic group movement
	return fmt.Sprintf("several creatures move in the distance.")
}

// describeEnemyApproach describes enemies approaching the player
func (pg *ProseGenerator) describeEnemyApproach(actions []EnemyTurnAction) string {
	if len(actions) == 0 {
		return ""
	}

	if len(actions) == 1 {
		action := actions[0]
		name := pg.getEntityDisplayName(action.Entity)
		verb := pg.pickRandom(pg.enemyApproachVerbs)
		return fmt.Sprintf("The %s %s!", name, verb)
	}

	// Multiple approaching
	return fmt.Sprintf("%d enemies are closing in!", len(actions))
}

// describeVisibleEnemies describes enemies that are visible but didn't act
func (pg *ProseGenerator) describeVisibleEnemies(enemies []*EntityInfo) string {
	if len(enemies) == 0 {
		return ""
	}

	// Group by distance
	close := []*EntityInfo{}   // <= 3 tiles
	medium := []*EntityInfo{}  // 4-6 tiles
	far := []*EntityInfo{}     // > 6 tiles

	for _, e := range enemies {
		if e.Distance <= 3 {
			close = append(close, e)
		} else if e.Distance <= 6 {
			medium = append(medium, e)
		} else {
			far = append(far, e)
		}
	}

	if len(close) > 0 {
		name := close[0].Entity.Name
		return fmt.Sprintf("A %s lurks dangerously close!", name)
	}

	if len(medium) > 0 {
		if len(medium) == 1 {
			return fmt.Sprintf("You spot a %s nearby to the %s.",
				medium[0].Entity.Name, medium[0].Direction)
		}
		return fmt.Sprintf("You spot %d enemies nearby.", len(medium))
	}

	if len(far) > 0 {
		return fmt.Sprintf("Movement in the distance catches your eye.")
	}

	return ""
}

// describeSensoryDetails adds atmospheric/sensory elements
func (pg *ProseGenerator) describeSensoryDetails(ctx *ProseContext) string {
	// Get sensory phrase based on nearby enemies
	if len(ctx.VisibleEnemies) > 0 {
		// Pick the closest enemy for sensory description
		closestEnemy := ctx.VisibleEnemies[0]
		for _, e := range ctx.VisibleEnemies {
			if e.Distance < closestEnemy.Distance {
				closestEnemy = e
			}
		}

		// Only add sensory details sometimes (not every turn)
		if pg.rng.Float32() < 0.6 { // 60% chance
			return pg.getSensoryPhrase(closestEnemy.Entity.Name)
		}
	}

	return ""
}

// getFurnishingDisplayName returns a readable name for a furnishing
func (pg *ProseGenerator) getFurnishingDisplayName(f *furnishing.PlacedFurnishing) string {
	if f == nil || f.Definition == nil {
		return "object"
	}

	if f.Definition.DisplayName != "" {
		return f.Definition.DisplayName
	}

	// Convert name like "wooden_barrel" to "wooden barrel"
	name := strings.ReplaceAll(f.Definition.Name, "_", " ")
	return name
}

// getEntityDisplayName returns a readable name for an entity
func (pg *ProseGenerator) getEntityDisplayName(e *entity.Entity) string {
	if e == nil {
		return "creature"
	}
	return strings.ToLower(e.Name)
}

// DeterminePositionalRelation determines the player's relation to a furnishing
// relative to enemy positions
func DeterminePositionalRelation(playerX, playerY int, furnX, furnY int, enemies []*EntityInfo) PositionalRelation {
	// Calculate if furnishing is between player and enemies
	if len(enemies) == 0 {
		// No enemies - just check distance
		dx := abs(playerX - furnX)
		dy := abs(playerY - furnY)
		if dx <= 1 && dy <= 1 {
			return RelBeside
		}
		if dx <= 2 && dy <= 2 {
			return RelNear
		}
		return RelNone
	}

	// Check if furnishing provides cover from enemies
	// Simple check: is furnishing between player and average enemy position?
	avgEnemyX, avgEnemyY := 0, 0
	for _, e := range enemies {
		avgEnemyX += e.Entity.X
		avgEnemyY += e.Entity.Y
	}
	avgEnemyX /= len(enemies)
	avgEnemyY /= len(enemies)

	// Vector from player to furnishing
	pfX := furnX - playerX
	pfY := furnY - playerY

	// Vector from player to enemies
	peX := avgEnemyX - playerX
	peY := avgEnemyY - playerY

	// If furnishing is adjacent and in the direction of enemies, it's cover
	if abs(pfX) <= 1 && abs(pfY) <= 1 {
		// Check if furnishing is between player and enemies
		if (pfX != 0 && sign(pfX) == sign(peX)) || (pfY != 0 && sign(pfY) == sign(peY)) {
			return RelBehind
		}
		return RelBeside
	}

	return RelNone
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}
