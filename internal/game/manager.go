package game

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

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
	"chosenoffset.com/outpost9/internal/ui/menu"
	"chosenoffset.com/outpost9/internal/ui/narrative"
	"chosenoffset.com/outpost9/internal/world/atlas"
	"chosenoffset.com/outpost9/internal/world/maploader"
	"chosenoffset.com/outpost9/internal/world/room"
)

// Manager handles the overall game state, including menu and gameplay.
type Manager struct {
	ScreenWidth  int
	ScreenHeight int
	State        menu.GameState
	MainMenu     *menu.MainMenu
	Game         *Game
	Renderer     render.Renderer
	InputMgr     render.InputManager
	Loader       render.ResourceLoader

	// Shader sources (passed in from main)
	ShadowShaderSrc   []byte
	LightingShaderSrc []byte

	// Character creation
	CharCreation     *character.CreationManager
	CharTemplate     *character.CharacterTemplate
	PendingSelection menu.Selection
}

// NewManager creates a new game manager.
func NewManager(r render.Renderer, input render.InputManager, loader render.ResourceLoader, width, height int) *Manager {
	return &Manager{
		ScreenWidth:  width,
		ScreenHeight: height,
		State:        menu.StateMainMenu,
		Renderer:     r,
		InputMgr:     input,
		Loader:       loader,
	}
}

// SetMainMenu sets the main menu.
func (m *Manager) SetMainMenu(mainMenu *menu.MainMenu) {
	m.MainMenu = mainMenu
}

// SetShaderSources sets the shader source code.
func (m *Manager) SetShaderSources(shadowSrc, lightingSrc []byte) {
	m.ShadowShaderSrc = shadowSrc
	m.LightingShaderSrc = lightingSrc
}

// Update updates the game state.
func (m *Manager) Update() error {
	switch m.State {
	case menu.StateMainMenu:
		selected, selection := m.MainMenu.Update()
		if selected {
			m.PendingSelection = selection
			charTemplatePath := fmt.Sprintf("data/%s/character.json", selection.GameDir)
			template, err := character.LoadCharacterTemplate(charTemplatePath)
			if err != nil {
				log.Printf("No character template found (%v), skipping character creation", err)
				if err := m.LoadGame(selection, nil); err != nil {
					log.Printf("Failed to load game: %v", err)
					return err
				}
				m.State = menu.StatePlaying
			} else {
				m.CharTemplate = template
				m.CharCreation = character.NewCreationManager(template, m.ScreenWidth, m.ScreenHeight)
				m.CharCreation.SetOnComplete(func(char *character.Character) {
					if err := m.LoadGame(m.PendingSelection, char); err != nil {
						log.Printf("Failed to load game: %v", err)
						return
					}
					m.State = menu.StatePlaying
				})
				m.State = menu.StateCharacterCreation
			}
		}
	case menu.StateCharacterCreation:
		if m.CharCreation != nil {
			if m.InputMgr.IsKeyPressed(render.KeyEscape) {
				m.State = menu.StateMainMenu
				m.CharCreation = nil
				return nil
			}
			return m.CharCreation.Update()
		}
	case menu.StatePlaying:
		if m.Game != nil {
			if m.InputMgr.IsKeyPressed(render.KeyEscape) {
				m.State = menu.StateMainMenu
			}
			return m.Game.Update()
		}
	}
	return nil
}

// Draw draws the current state.
func (m *Manager) Draw(screen render.Image) {
	switch m.State {
	case menu.StateMainMenu:
		m.MainMenu.Draw(screen)
	case menu.StateCharacterCreation:
		// Character creation needs special handling - it uses ebiten directly
		// This is a known limitation that needs future work
		screen.Fill(color.RGBA{20, 20, 40, 255})
		m.Renderer.DrawText(screen, "Character Creation (press ESC to return)", 50, 50, color.RGBA{255, 255, 255, 255}, 1.5)
	case menu.StatePlaying:
		if m.Game != nil {
			m.Game.Draw(screen)
		}
	}
}

// Layout handles window resize.
func (m *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth != m.ScreenWidth || outsideHeight != m.ScreenHeight {
		m.ScreenWidth = outsideWidth
		m.ScreenHeight = outsideHeight
		if m.MainMenu != nil {
			m.MainMenu.SetSize(outsideWidth, outsideHeight)
		}
		if m.Game != nil {
			m.Game.ScreenWidth = outsideWidth
			m.Game.ScreenHeight = outsideHeight
			m.Game.MapViewWidth = outsideWidth - m.Game.PanelWidth
			if m.Game.NarrativePanel != nil {
				m.Game.NarrativePanel.Resize(outsideWidth, outsideHeight, m.Game.PanelWidth)
			}
			m.Game.UpdateCamera()
		}
	}
	return outsideWidth, outsideHeight
}

// LoadGame loads a game from a room library selection.
func (m *Manager) LoadGame(selection menu.Selection, playerChar *character.Character) error {
	libraryPath := fmt.Sprintf("data/%s/%s", selection.GameDir, selection.RoomLibraryFile)
	log.Printf("Loading room library: %s", libraryPath)

	config := room.GeneratorConfig{
		MinRooms:     8,
		MaxRooms:     12,
		Seed:         0,
		ConnectAll:   true,
		AllowOverlap: false,
	}

	gameMap, err := maploader.LoadMapFromRoomLibrary(libraryPath, config, m.Loader)
	if err != nil {
		return fmt.Errorf("failed to generate map: %w", err)
	}

	log.Printf("Generated map: %s (%dx%d)", gameMap.Data.Name, gameMap.Data.Width, gameMap.Data.Height)

	walls := shadows.CreateWallSegmentsFromMap(gameMap)
	log.Printf("Generated %d wall segments", len(walls))

	// Load atlases
	entitiesAtlasPath := fmt.Sprintf("data/%s/entities.json", selection.GameDir)
	entitiesAtlas, err := atlas.LoadAtlas(entitiesAtlasPath, m.Loader)
	if err != nil {
		log.Printf("Warning: Failed to load entities atlas: %v", err)
	}

	objectsAtlasPath := fmt.Sprintf("data/%s/objects_layer.json", selection.GameDir)
	objectsAtlas, err := atlas.LoadAtlas(objectsAtlasPath, m.Loader)
	if err != nil {
		log.Printf("Warning: Failed to load objects atlas: %v", err)
	}

	var playerSprite render.Image
	if entitiesAtlas != nil {
		playerSprite, err = entitiesAtlas.GetTileSubImageByName("player_idle")
		if err != nil {
			log.Printf("Warning: Failed to get player sprite: %v", err)
		}
	}

	// Compile shaders
	var shadowShader, lightingShader render.Shader
	if m.ShadowShaderSrc != nil {
		shadowShader, err = m.Renderer.CompileShader(m.ShadowShaderSrc)
		if err != nil {
			return fmt.Errorf("failed to compile shadow shader: %w", err)
		}
	}
	if m.LightingShaderSrc != nil {
		lightingShader, err = m.Renderer.CompileShader(m.LightingShaderSrc)
		if err != nil {
			return fmt.Errorf("failed to compile lighting shader: %w", err)
		}
	}

	// Initialize lighting
	lightingMgr := lighting.NewManager()
	lightingMgr.SetPlayerLight(0, 0, 400.0, 1.0, color.NRGBA{255, 240, 200, 255})

	// Initialize game state
	gs := gamestate.New()
	inv := inventory.New()
	interactionEng := interaction.NewEngine()
	interactionEng.GameState = gs
	interactionEng.Inventory = inv

	// Calculate spawn position
	tileSize := gameMap.Data.TileSize
	spawnGridX := int(gameMap.Data.PlayerSpawn.X) / tileSize
	spawnGridY := int(gameMap.Data.PlayerSpawn.Y) / tileSize

	m.Game = &Game{
		ScreenWidth:       m.ScreenWidth,
		ScreenHeight:      m.ScreenHeight,
		GameMap:           gameMap,
		Walls:             walls,
		Player: Player{
			Pos:   shadows.Point{X: gameMap.Data.PlayerSpawn.X, Y: gameMap.Data.PlayerSpawn.Y},
			Speed: 3.0,
			GridX: spawnGridX,
			GridY: spawnGridY,
		},
		Renderer:          m.Renderer,
		InputMgr:          m.InputMgr,
		EntitiesAtlas:     entitiesAtlas,
		ObjectsAtlas:      objectsAtlas,
		PlayerSpriteImg:   playerSprite,
		ShadowShader:      shadowShader,
		LightingShader:    lightingShader,
		LightingManager:   lightingMgr,
		InteractionEngine: interactionEng,
		GameState:         gs,
		Inventory:         inv,
		PlayerChar:        playerChar,
		PanelWidth:        350,
		MapViewWidth:      m.ScreenWidth - 350,
	}

	m.Game.InteractionEngine.OnMessage = m.Game.ShowMessage

	// Load enemy library
	enemiesPath := fmt.Sprintf("data/%s/enemies.json", selection.GameDir)
	enemyLib, err := entity.LoadEntityLibrary(enemiesPath)
	if err != nil {
		log.Printf("Warning: Failed to load enemy library: %v", err)
	} else {
		m.Game.EntityLibrary = enemyLib
	}

	// Initialize turn manager
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	turnMgr := turn.NewManager(rng)
	playerEntity := entity.NewPlayerEntity(playerChar, spawnGridX, spawnGridY)
	turnMgr.SetPlayer(playerEntity)
	m.Game.PlayerEntity = playerEntity
	m.Game.TurnManager = turnMgr

	turnMgr.OnMessage = m.Game.ShowMessage
	turnMgr.IsWalkable = m.Game.IsTileWalkable
	turnMgr.GetEntityAt = turnMgr.GetEntityAtPosition
	turnMgr.OnTurnStart = func(turnNum int) {
		if m.Game.GameHUD != nil {
			m.Game.GameHUD.SetTurnNumber(turnNum)
		}
	}

	// Load actions
	actionsPath := fmt.Sprintf("data/%s/actions.json", selection.GameDir)
	actionLib, err := action.LoadActionLibrary(actionsPath)
	if err != nil {
		actionLib = action.DefaultLibrary()
	} else {
		defaults := action.DefaultLibrary()
		defaults.MergeLibrary(actionLib)
		actionLib = defaults
	}
	m.Game.ActionLibrary = actionLib
	turnMgr.SetActionLibrary(actionLib)

	// Initialize UI
	panelX := m.Game.MapViewWidth
	m.Game.NarrativePanel = narrative.NewPanel(panelX, 0, m.Game.PanelWidth, m.ScreenHeight)
	m.Game.SceneGenerator = narrative.NewSceneGenerator()
	m.Game.TurnNarrator = narrative.NewTurnNarrator()
	m.Game.ProseGenerator = narrative.NewProseGenerator(time.Now().UnixNano())

	// Initialize room tracker
	if gameMap.GeneratedLevel != nil {
		m.Game.RoomTracker = roominfo.NewRoomTracker(gameMap.GeneratedLevel)
		m.Game.RoomTracker.UpdatePlayerPosition(playerEntity.X, playerEntity.Y)
	}

	// Initialize HUD
	hudConfig := hud.DefaultConfig()
	hudConfig.StatCategories = []string{"attributes"}
	m.Game.GameHUD = hud.New(hudConfig, m.ScreenWidth, m.ScreenHeight)
	m.Game.GameHUD.SetPlayer(playerEntity, playerChar)
	m.Game.GameHUD.SetTurnNumber(1)

	// Start the game
	turnMgr.StartNewTurn()
	m.Game.UpdateNarrativePanel()

	log.Printf("Game loaded successfully")
	return nil
}

// IsTileWalkable checks if a tile is walkable.
func (g *Game) IsTileWalkable(x, y int) bool {
	if g.GameMap == nil {
		return false
	}
	return g.GameMap.IsWalkable(x, y)
}
