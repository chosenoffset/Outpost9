package game

import (
	"log"

	"chosenoffset.com/outpost9/internal/action"
	"chosenoffset.com/outpost9/internal/character"
	"chosenoffset.com/outpost9/internal/core/gamestate"
	"chosenoffset.com/outpost9/internal/core/shadows"
	"chosenoffset.com/outpost9/internal/entity"
	"chosenoffset.com/outpost9/internal/entity/turn"
	"chosenoffset.com/outpost9/internal/interaction"
	"chosenoffset.com/outpost9/internal/inventory"
	"chosenoffset.com/outpost9/internal/render"
	"chosenoffset.com/outpost9/internal/render/lighting"
	"chosenoffset.com/outpost9/internal/roominfo"
	"chosenoffset.com/outpost9/internal/ui/hud"
	"chosenoffset.com/outpost9/internal/ui/narrative"
	"chosenoffset.com/outpost9/internal/world/atlas"
	"chosenoffset.com/outpost9/internal/world/maploader"
)

// Game holds all game state and logic.
type Game struct {
	ScreenWidth  int
	ScreenHeight int
	GameMap      *maploader.Map
	Walls        []shadows.Segment
	Player       Player
	Camera       Camera
	WhiteImg     render.Image
	Renderer     render.Renderer
	InputMgr     render.InputManager
	EntitiesAtlas   *atlas.Atlas
	ObjectsAtlas    *atlas.Atlas
	PlayerSpriteImg render.Image
	ShadowShader    render.Shader
	WallTexture     render.Image
	LightingShader  render.Shader
	LightingManager *lighting.Manager
	SceneTexture    render.Image

	// Interaction system
	InteractionEngine *interaction.Engine
	GameState         *gamestate.GameState
	Inventory         *inventory.Inventory

	// Player character
	PlayerChar *character.Character

	// Turn-based system
	TurnManager   *turn.Manager
	PlayerEntity  *entity.Entity
	EntityLibrary *entity.EntityLibrary

	// Action system
	ActionLibrary *action.ActionLibrary

	// Narrative panel
	NarrativePanel *narrative.Panel
	SceneGenerator *narrative.SceneGenerator
	TurnNarrator   *narrative.TurnNarrator
	ProseGenerator *narrative.ProseGenerator

	// Room tracking
	RoomTracker *roominfo.RoomTracker

	// Player action tracking for prose
	LastPlayerAction    string
	LastPlayerDirection string

	// HUD
	GameHUD *hud.HUD

	// Layout dimensions (split screen)
	MapViewWidth int
	PanelWidth   int

	// UI state
	Messages         []Message
	InteractHint     string
	InteractCooldown float64

	// Debug
	FrameCount int
}

// Update handles game logic updates.
func (g *Game) Update() error {
	// Delta time for timers (assuming 60 FPS)
	dt := 1.0 / 60.0

	// Update message timers
	g.updateMessages(dt)

	// Update interaction cooldown
	if g.InteractCooldown > 0 {
		g.InteractCooldown -= dt
	}

	// Handle input when it's player's turn
	if g.TurnManager != nil && g.TurnManager.IsPlayerTurn() && g.PlayerEntity != nil {
		// Direct movement with WASD only
		var dir entity.Direction
		if g.InputMgr.IsKeyJustPressed(render.KeyW) {
			dir = entity.DirNorth
		} else if g.InputMgr.IsKeyJustPressed(render.KeyS) {
			dir = entity.DirSouth
		} else if g.InputMgr.IsKeyJustPressed(render.KeyA) {
			dir = entity.DirWest
		} else if g.InputMgr.IsKeyJustPressed(render.KeyD) {
			dir = entity.DirEast
		}

		if dir != entity.DirNone {
			// Check if in direction selection mode for narrative panel
			if g.NarrativePanel != nil && g.NarrativePanel.GetInputMode() == narrative.ModeSelectDirection {
				g.NarrativePanel.Update()
			} else {
				// Direct movement
				moveAction := g.ActionLibrary.GetAction("move")
				if moveAction != nil && g.PlayerEntity.CanAffordAP(moveAction.APCost) {
					g.LastPlayerAction = "move"
					g.LastPlayerDirection = DirectionName(dir)
					g.TurnManager.ProcessDataAction(moveAction, dir, 0, 0)
					g.SyncPlayerPosition()
					g.UpdateNarrativePanel()
				}
			}
		} else {
			// Handle narrative panel for non-movement actions
			if g.NarrativePanel != nil {
				g.NarrativePanel.Update()
			}
		}

		// End turn with Space key
		if g.InputMgr.IsKeyJustPressed(render.KeySpace) {
			g.TurnManager.EndPlayerTurn()
			g.SyncPlayerPosition()
			g.UpdateNarrativePanel()
		}

		// Toggle player light with L key
		if g.InputMgr.IsKeyJustPressed(render.KeyL) {
			if g.LightingManager != nil {
				wasOn := g.LightingManager.IsPlayerLightOn()
				g.LightingManager.EnablePlayerLight(!wasOn)
				if !wasOn {
					g.ShowMessage("Light source activated")
				} else {
					g.ShowMessage("Light source deactivated")
				}
			}
		}
	}

	// Update camera to follow player
	g.UpdateCamera()

	// Update player light position
	if g.LightingManager != nil {
		g.LightingManager.UpdatePlayerLightPosition(g.Player.Pos.X, g.Player.Pos.Y)
	}

	// Handle interactions (E key) - legacy support
	g.UpdateInteractions()

	return nil
}

// Layout returns the game's logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.ScreenWidth, g.ScreenHeight
}

// SyncPlayerPosition updates the pixel position from grid position.
func (g *Game) SyncPlayerPosition() {
	if g.PlayerEntity == nil {
		return
	}
	tileSize := float64(g.GameMap.Data.TileSize)
	g.Player.GridX = g.PlayerEntity.X
	g.Player.GridY = g.PlayerEntity.Y
	g.Player.Pos.X = float64(g.PlayerEntity.X)*tileSize + tileSize/2
	g.Player.Pos.Y = float64(g.PlayerEntity.Y)*tileSize + tileSize/2

	if g.RoomTracker != nil {
		g.RoomTracker.UpdatePlayerPosition(g.PlayerEntity.X, g.PlayerEntity.Y)
	}
}

func (g *Game) updateMessages(dt float64) {
	var active []Message
	for _, msg := range g.Messages {
		msg.TimeLeft -= dt
		if msg.TimeLeft > 0 {
			active = append(active, msg)
		}
	}
	g.Messages = active
}

// ShowMessage adds a new message to be displayed on screen.
func (g *Game) ShowMessage(text string) {
	g.Messages = append(g.Messages, Message{
		Text:     text,
		TimeLeft: 3.0,
		MaxTime:  3.0,
	})

	if g.NarrativePanel != nil && g.TurnManager != nil {
		g.NarrativePanel.AddMessage(text, g.TurnManager.GetTurnNumber())
	}

	log.Printf("Message: %s", text)
}

// DirectionName returns a string name for a direction.
func DirectionName(dir entity.Direction) string {
	switch dir {
	case entity.DirNorth:
		return "north"
	case entity.DirSouth:
		return "south"
	case entity.DirEast:
		return "east"
	case entity.DirWest:
		return "west"
	default:
		return ""
	}
}

// UpdateNarrativePanel updates the scene description and available actions.
func (g *Game) UpdateNarrativePanel() {
	if g.NarrativePanel == nil || g.SceneGenerator == nil || g.PlayerEntity == nil {
		return
	}
	g.NarrativePanel.SetAP(g.PlayerEntity.ActionPoints, g.PlayerEntity.MaxAP)
	ctx := g.BuildSceneContext()
	description := g.SceneGenerator.GenerateDescription(ctx)
	g.NarrativePanel.SetSceneDescription(description)
	actions := g.BuildAvailableActions()
	g.NarrativePanel.SetAvailableActions(actions)
}

// UpdateCamera updates the camera to follow the player.
func (g *Game) UpdateCamera() {
	if g.GameMap == nil {
		return
	}
	// Center camera on player
	g.Camera.X = g.Player.Pos.X - float64(g.MapViewWidth)/2
	g.Camera.Y = g.Player.Pos.Y - float64(g.ScreenHeight)/2

	// Clamp camera to map bounds
	mapWidth := float64(g.GameMap.Data.Width * g.GameMap.Data.TileSize)
	mapHeight := float64(g.GameMap.Data.Height * g.GameMap.Data.TileSize)

	if g.Camera.X < 0 {
		g.Camera.X = 0
	}
	if g.Camera.Y < 0 {
		g.Camera.Y = 0
	}
	if g.Camera.X > mapWidth-float64(g.MapViewWidth) {
		g.Camera.X = mapWidth - float64(g.MapViewWidth)
	}
	if g.Camera.Y > mapHeight-float64(g.ScreenHeight) {
		g.Camera.Y = mapHeight - float64(g.ScreenHeight)
	}
}

// UpdateInteractions handles interaction key presses.
func (g *Game) UpdateInteractions() {
	// Placeholder - interaction logic will be added
}

// BuildSceneContext builds the context for scene generation.
func (g *Game) BuildSceneContext() *narrative.SceneContext {
	// Placeholder - returns minimal context
	return &narrative.SceneContext{}
}

// BuildAvailableActions builds the list of available actions.
func (g *Game) BuildAvailableActions() []*narrative.ActionChoice {
	// Placeholder - returns empty list
	return nil
}
