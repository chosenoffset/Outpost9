// Package narrative scene generation
// This file contains logic to generate descriptive text based on the game state.
package narrative

import (
	"fmt"
	"sort"
	"strings"

	"chosenoffset.com/outpost9/entity"
)

// SceneContext provides the information needed to generate a scene description
type SceneContext struct {
	// Player info
	PlayerX, PlayerY int
	PlayerFacing     entity.Direction
	PlayerHP         int
	PlayerMaxHP      int
	PlayerAP         int
	PlayerMaxAP      int

	// Nearby entities
	NearbyEntities []*EntityInfo

	// Environment
	RoomName        string
	LightLevel      string // "bright", "dim", "dark"
	TerrainFeatures []string
	NearbyObjects   []string // Interactable objects

	// Exits/passages
	Exits []ExitInfo
}

// EntityInfo describes a nearby entity
type EntityInfo struct {
	Entity    *entity.Entity
	Distance  int
	Direction string // "north", "nearby", etc.
	Visible   bool
	Facing    string // Which way they're facing
	Status    string // "unaware", "alert", "hostile"
}

// ExitInfo describes an available exit
type ExitInfo struct {
	Direction   string
	Description string // "a dark corridor", "an open doorway"
}

// SceneGenerator creates narrative text from game state
type SceneGenerator struct {
	// Templates for various descriptions
	// These could be loaded from data files for customization
}

// NewSceneGenerator creates a new scene generator
func NewSceneGenerator() *SceneGenerator {
	return &SceneGenerator{}
}

// GenerateDescription creates a full scene description
func (sg *SceneGenerator) GenerateDescription(ctx *SceneContext) string {
	var parts []string

	// Location/environment
	if ctx.RoomName != "" {
		parts = append(parts, sg.describeLocation(ctx))
	}

	// Light conditions (if not bright)
	if ctx.LightLevel != "" && ctx.LightLevel != "bright" {
		parts = append(parts, sg.describeLighting(ctx.LightLevel))
	}

	// Nearby entities (most important)
	if len(ctx.NearbyEntities) > 0 {
		parts = append(parts, sg.describeEntities(ctx.NearbyEntities))
	}

	// Nearby objects
	if len(ctx.NearbyObjects) > 0 {
		parts = append(parts, sg.describeObjects(ctx.NearbyObjects))
	}

	// Exits
	if len(ctx.Exits) > 0 {
		parts = append(parts, sg.describeExits(ctx.Exits))
	}

	// Player status if relevant
	if ctx.PlayerHP < ctx.PlayerMaxHP/2 {
		parts = append(parts, sg.describePlayerStatus(ctx))
	}

	if len(parts) == 0 {
		return "You stand ready, considering your next move."
	}

	return strings.Join(parts, " ")
}

func (sg *SceneGenerator) describeLocation(ctx *SceneContext) string {
	return fmt.Sprintf("You are in %s.", ctx.RoomName)
}

func (sg *SceneGenerator) describeLighting(level string) string {
	switch level {
	case "dim":
		return "The light is dim here, shadows pool in the corners."
	case "dark":
		return "Darkness surrounds you, limiting your vision."
	case "shadowy":
		return "Deep shadows offer places to hide."
	default:
		return ""
	}
}

func (sg *SceneGenerator) describeEntities(entities []*EntityInfo) string {
	if len(entities) == 0 {
		return ""
	}

	// Sort by distance (closest first)
	sorted := make([]*EntityInfo, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Distance < sorted[j].Distance
	})

	var descriptions []string

	// Group by threat level
	var threats, neutrals []*EntityInfo
	for _, e := range sorted {
		if e.Entity.Faction == entity.FactionEnemy {
			threats = append(threats, e)
		} else if e.Entity.Faction != entity.FactionPlayer {
			neutrals = append(neutrals, e)
		}
	}

	// Describe threats first
	if len(threats) > 0 {
		descriptions = append(descriptions, sg.describeThreatGroup(threats))
	}

	// Then neutrals
	if len(neutrals) > 0 {
		descriptions = append(descriptions, sg.describeNeutralGroup(neutrals))
	}

	return strings.Join(descriptions, " ")
}

func (sg *SceneGenerator) describeThreatGroup(entities []*EntityInfo) string {
	if len(entities) == 0 {
		return ""
	}

	if len(entities) == 1 {
		e := entities[0]
		return sg.describeSingleThreat(e)
	}

	// Multiple threats
	var names []string
	for _, e := range entities {
		names = append(names, e.Entity.Name)
	}

	closest := entities[0]
	if closest.Distance <= 1 {
		return fmt.Sprintf("You are surrounded! %s are close by.", joinNames(names))
	} else if closest.Distance <= 3 {
		return fmt.Sprintf("%s lurk nearby, watching.", joinNames(names))
	}

	return fmt.Sprintf("You spot %s in the distance.", joinNames(names))
}

func (sg *SceneGenerator) describeSingleThreat(e *EntityInfo) string {
	name := e.Entity.Name

	// Include facing information if relevant
	facingInfo := ""
	if e.Facing != "" {
		facingInfo = fmt.Sprintf(", facing %s", e.Facing)
	}

	// Include status
	statusInfo := ""
	switch e.Status {
	case "unaware":
		statusInfo = " It hasn't noticed you."
	case "alert":
		statusInfo = " It seems alert."
	case "hostile":
		statusInfo = " It looks hostile!"
	}

	if e.Distance <= 1 {
		return fmt.Sprintf("A %s stands right next to you%s!%s", name, facingInfo, statusInfo)
	} else if e.Distance <= 2 {
		return fmt.Sprintf("A %s is very close, %s%s.%s", name, e.Direction, facingInfo, statusInfo)
	} else if e.Distance <= 4 {
		return fmt.Sprintf("A %s lurks %s%s.%s", name, e.Direction, facingInfo, statusInfo)
	}

	return fmt.Sprintf("You spot a %s to the %s%s.%s", name, e.Direction, facingInfo, statusInfo)
}

func (sg *SceneGenerator) describeNeutralGroup(entities []*EntityInfo) string {
	if len(entities) == 0 {
		return ""
	}

	if len(entities) == 1 {
		e := entities[0]
		return fmt.Sprintf("A %s is nearby to the %s.", e.Entity.Name, e.Direction)
	}

	var names []string
	for _, e := range entities {
		names = append(names, e.Entity.Name)
	}

	return fmt.Sprintf("Several figures are nearby: %s.", joinNames(names))
}

func (sg *SceneGenerator) describeObjects(objects []string) string {
	if len(objects) == 0 {
		return ""
	}

	if len(objects) == 1 {
		return fmt.Sprintf("You notice %s nearby.", objects[0])
	}

	return fmt.Sprintf("Nearby you see: %s.", joinNames(objects))
}

func (sg *SceneGenerator) describeExits(exits []ExitInfo) string {
	if len(exits) == 0 {
		return ""
	}

	var exitDescs []string
	for _, exit := range exits {
		if exit.Description != "" {
			exitDescs = append(exitDescs, fmt.Sprintf("%s to the %s", exit.Description, exit.Direction))
		} else {
			exitDescs = append(exitDescs, fmt.Sprintf("a passage %s", exit.Direction))
		}
	}

	if len(exitDescs) == 1 {
		return fmt.Sprintf("There is %s.", exitDescs[0])
	}

	return fmt.Sprintf("Exits lead %s.", joinNames(exitDescs))
}

func (sg *SceneGenerator) describePlayerStatus(ctx *SceneContext) string {
	hpPercent := float64(ctx.PlayerHP) / float64(ctx.PlayerMaxHP)

	if hpPercent < 0.25 {
		return "You are badly wounded and bleeding."
	} else if hpPercent < 0.5 {
		return "You are wounded."
	}

	return ""
}

// GenerateActionResult creates narrative text for an action's result
func (sg *SceneGenerator) GenerateActionResult(
	actorName string,
	actionVerb string,
	targetName string,
	success bool,
	details string,
) string {
	if targetName == "" {
		// Self-action or no target
		if success {
			return fmt.Sprintf("%s %s. %s", actorName, actionVerb, details)
		}
		return fmt.Sprintf("%s tries to %s but fails. %s", actorName, actionVerb, details)
	}

	// Action with target
	if success {
		return fmt.Sprintf("%s %s %s. %s", actorName, actionVerb, targetName, details)
	}
	return fmt.Sprintf("%s tries to %s %s but fails. %s", actorName, actionVerb, targetName, details)
}

// Helper functions

func joinNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}
	if len(names) == 2 {
		return names[0] + " and " + names[1]
	}

	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

// DirectionName returns a readable direction name
func DirectionName(dx, dy int) string {
	if dx == 0 && dy < 0 {
		return "north"
	}
	if dx == 0 && dy > 0 {
		return "south"
	}
	if dx > 0 && dy == 0 {
		return "east"
	}
	if dx < 0 && dy == 0 {
		return "west"
	}
	if dx > 0 && dy < 0 {
		return "northeast"
	}
	if dx < 0 && dy < 0 {
		return "northwest"
	}
	if dx > 0 && dy > 0 {
		return "southeast"
	}
	if dx < 0 && dy > 0 {
		return "southwest"
	}
	return "nearby"
}

// FacingName returns a readable facing direction
func FacingName(facing entity.Direction) string {
	switch facing {
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
		return ""
	}
}
