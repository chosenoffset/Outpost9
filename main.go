package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"chosenoffset.com/outpost9/atlas"
	"chosenoffset.com/outpost9/character"
	"chosenoffset.com/outpost9/entity"
	"chosenoffset.com/outpost9/furnishing"
	"chosenoffset.com/outpost9/gamescanner"
	"chosenoffset.com/outpost9/gamestate"
	"chosenoffset.com/outpost9/hud"
	"chosenoffset.com/outpost9/interaction"
	"chosenoffset.com/outpost9/inventory"
	"chosenoffset.com/outpost9/maploader"
	"chosenoffset.com/outpost9/menu"
	"chosenoffset.com/outpost9/renderer"
	ebitenrenderer "chosenoffset.com/outpost9/renderer/ebiten"
	"chosenoffset.com/outpost9/room"
	"chosenoffset.com/outpost9/shadows"
	"chosenoffset.com/outpost9/turn"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed shaders/sight_shadows.kage
var shadowShaderSrc []byte

type Player struct {
	Pos   shadows.Point
	Speed float64
	// Grid position for turn-based movement
	GridX, GridY int
}

// Camera tracks the viewport position for scrolling large levels
type Camera struct {
	X, Y float64 // Camera position (top-left corner of viewport in world coords)
}

// GameManager handles the overall game state, including menu and gameplay.
type GameManager struct {
	screenWidth  int
	screenHeight int
	state        menu.GameState
	mainMenu     *menu.MainMenu
	game         *Game
	renderer     renderer.Renderer
	inputMgr     renderer.InputManager
	loader       renderer.ResourceLoader

	// Character creation
	charCreation     *character.CreationManager
	charTemplate     *character.CharacterTemplate
	pendingSelection menu.Selection // Store selection while character is being created
}

func (gm *GameManager) Update() error {
	switch gm.state {
	case menu.StateMainMenu:
		selected, selection := gm.mainMenu.Update()
		if selected {
			// Store selection and start character creation
			gm.pendingSelection = selection

			// Load character template for this game
			charTemplatePath := fmt.Sprintf("data/%s/character.json", selection.GameDir)
			template, err := character.LoadCharacterTemplate(charTemplatePath)
			if err != nil {
				log.Printf("No character template found (%v), skipping character creation", err)
				// No character template - go straight to game
				if err := gm.loadGame(selection, nil); err != nil {
					log.Printf("Failed to load game: %v", err)
					return err
				}
				gm.state = menu.StatePlaying
			} else {
				// Start character creation
				gm.charTemplate = template
				gm.charCreation = character.NewCreationManager(template, gm.screenWidth, gm.screenHeight)
				gm.charCreation.SetOnComplete(func(char *character.Character) {
					// Character created, now load the game
					if err := gm.loadGame(gm.pendingSelection, char); err != nil {
						log.Printf("Failed to load game: %v", err)
						return
					}
					gm.state = menu.StatePlaying
				})
				gm.state = menu.StateCharacterCreation
			}
		}
	case menu.StateCharacterCreation:
		if gm.charCreation != nil {
			// Check for ESC to return to menu
			if gm.inputMgr.IsKeyPressed(renderer.KeyEscape) {
				gm.state = menu.StateMainMenu
				gm.charCreation = nil
			}
			return gm.charCreation.Update()
		}
	case menu.StatePlaying:
		if gm.game != nil {
			// Check for ESC to return to menu
			if gm.inputMgr.IsKeyPressed(renderer.KeyEscape) {
				gm.state = menu.StateMainMenu
			}
			return gm.game.Update()
		}
	}
	return nil
}

func (gm *GameManager) Draw(screen renderer.Image) {
	switch gm.state {
	case menu.StateMainMenu:
		gm.mainMenu.Draw(screen)
	case menu.StateCharacterCreation:
		if gm.charCreation != nil {
			// Character creation uses ebiten.Image directly
			if ebitenScreen, ok := screen.(*ebitenrenderer.EbitenImage); ok {
				gm.charCreation.Draw(ebitenScreen.GetEbitenImage())
			}
		}
	case menu.StatePlaying:
		if gm.game != nil {
			gm.game.Draw(screen)
		}
	}
}

func (gm *GameManager) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Update screen dimensions when window is resized
	if outsideWidth != gm.screenWidth || outsideHeight != gm.screenHeight {
		gm.screenWidth = outsideWidth
		gm.screenHeight = outsideHeight
		// Update menu dimensions
		if gm.mainMenu != nil {
			gm.mainMenu.SetSize(outsideWidth, outsideHeight)
		}
		// Update game dimensions
		if gm.game != nil {
			gm.game.screenWidth = outsideWidth
			gm.game.screenHeight = outsideHeight
			// Recreate wall texture for new size
			gm.game.wallTexture = ebiten.NewImage(outsideWidth, outsideHeight)
		}
	}
	return outsideWidth, outsideHeight
}

func (gm *GameManager) loadGame(selection menu.Selection, playerChar *character.Character) error {
	libraryPath := fmt.Sprintf("data/%s/%s", selection.GameDir, selection.RoomLibraryFile)

	// Load procedurally generated level from room library
	log.Printf("Loading room library: %s", libraryPath)

	config := room.GeneratorConfig{
		MinRooms:     8,
		MaxRooms:     12,
		Seed:         0, // Use random seed each time
		ConnectAll:   true,
		AllowOverlap: false,
	}

	gameMap, err := maploader.LoadMapFromRoomLibrary(libraryPath, config, gm.loader)
	if err != nil {
		return fmt.Errorf("failed to generate map from room library: %w", err)
	}

	log.Printf("Generated procedural map: %s (%dx%d, tile size: %dpx)",
		gameMap.Data.Name,
		gameMap.Data.Width,
		gameMap.Data.Height,
		gameMap.Data.TileSize)

	// Generate wall segments from map data
	walls := shadows.CreateWallSegmentsFromMap(gameMap)
	log.Printf("Generated %d wall segments", len(walls))

	// Load entities atlas for player, enemies, items, etc.
	entitiesAtlasPath := fmt.Sprintf("data/%s/entities.json", selection.GameDir)
	entitiesAtlas, err := atlas.LoadAtlas(entitiesAtlasPath, gm.loader)
	if err != nil {
		log.Printf("Warning: Failed to load entities atlas: %v", err)
		// Continue without entities atlas - will use fallback rendering
	}

	// Load objects atlas for furnishings (crates, terminals, doors, etc.)
	objectsAtlasPath := fmt.Sprintf("data/%s/objects_layer.json", selection.GameDir)
	objectsAtlas, err := atlas.LoadAtlas(objectsAtlasPath, gm.loader)
	if err != nil {
		log.Printf("Warning: Failed to load objects atlas: %v", err)
		// Continue without objects atlas - furnishings won't render
	}

	// Extract player sprite if atlas loaded successfully
	var playerSprite renderer.Image
	if entitiesAtlas != nil {
		playerSprite, err = entitiesAtlas.GetTileSubImageByName("player_idle")
		if err != nil {
			log.Printf("Warning: Failed to get player sprite: %v", err)
		}
	}

	// Initialize shadow shader
	shadowShader, err := ebiten.NewShader(shadowShaderSrc)
	if err != nil {
		return fmt.Errorf("failed to create shadow shader: %w", err)
	}

	// Create wall texture render target
	wallTexture := ebiten.NewImage(gm.screenWidth, gm.screenHeight)

	// Initialize game state and inventory for interaction system
	gs := gamestate.New()
	inv := inventory.New()

	// Create the interaction engine
	interactionEng := interaction.NewEngine()
	interactionEng.GameState = gs
	interactionEng.Inventory = inv

	// Calculate grid position from pixel spawn position
	tileSize := gameMap.Data.TileSize
	spawnGridX := int(gameMap.Data.PlayerSpawn.X) / tileSize
	spawnGridY := int(gameMap.Data.PlayerSpawn.Y) / tileSize

	gm.game = &Game{
		screenWidth:  gm.screenWidth,
		screenHeight: gm.screenHeight,
		gameMap:      gameMap,
		walls:        walls,
		player: Player{
			Pos:   shadows.Point{X: gameMap.Data.PlayerSpawn.X, Y: gameMap.Data.PlayerSpawn.Y},
			Speed: 3.0,
			GridX: spawnGridX,
			GridY: spawnGridY,
		},
		renderer:          gm.renderer,
		inputMgr:          gm.inputMgr,
		entitiesAtlas:     entitiesAtlas,
		objectsAtlas:      objectsAtlas,
		playerSpriteImg:   playerSprite,
		shadowShader:      shadowShader,
		wallTexture:       wallTexture,
		interactionEngine: interactionEng,
		gameState:         gs,
		inventory:         inv,
		playerChar:        playerChar,
	}

	// Wire up interaction engine callbacks
	gm.game.interactionEngine.OnMessage = gm.game.showMessage
	gm.game.interactionEngine.ObjectLookup = gm.game.lookupFurnishing

	// Load enemy library
	enemiesPath := fmt.Sprintf("data/%s/enemies.json", selection.GameDir)
	enemyLib, err := entity.LoadEntityLibrary(enemiesPath)
	if err != nil {
		log.Printf("Warning: Failed to load enemy library: %v", err)
		// Continue without enemies
	} else {
		gm.game.entityLibrary = enemyLib
		log.Printf("Loaded enemy library with %d enemy types", len(enemyLib.Enemies))
	}

	// Initialize turn manager
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	turnMgr := turn.NewManager(rng)

	// Create player entity
	playerEntity := entity.NewPlayerEntity(playerChar, spawnGridX, spawnGridY)
	turnMgr.SetPlayer(playerEntity)
	gm.game.playerEntity = playerEntity
	gm.game.turnManager = turnMgr

	// Wire up turn manager callbacks
	turnMgr.OnMessage = gm.game.showMessage
	turnMgr.IsWalkable = gm.game.isTileWalkable
	turnMgr.GetEntityAt = turnMgr.GetEntityAtPosition
	turnMgr.OnTurnStart = func(turnNum int) {
		if gm.game.gameHUD != nil {
			gm.game.gameHUD.SetTurnNumber(turnNum)
		}
	}

	// Spawn some enemies
	gm.game.spawnEnemies()

	// Start the first turn
	turnMgr.StartNewTurn()

	// Initialize HUD
	hudConfig := hud.DefaultConfig()
	hudConfig.StatCategories = []string{"attributes"} // Only show main attributes
	gm.game.gameHUD = hud.New(hudConfig, gm.screenWidth, gm.screenHeight)
	gm.game.gameHUD.SetPlayer(playerEntity, playerChar)
	gm.game.gameHUD.SetTurnNumber(1)

	log.Printf("Interaction system initialized with %d furnishings",
		len(gameMap.Data.PlacedFurnishings))

	// Log character info if we have one
	if playerChar != nil {
		log.Printf("Player character: %s", playerChar.Name)
		for statID, statVal := range playerChar.Stats {
			log.Printf("  %s: %d", statID, statVal.Value)
		}
	}

	return nil
}

// Message represents an on-screen message that fades over time
type Message struct {
	Text     string
	TimeLeft float64 // Seconds remaining
	MaxTime  float64 // Initial duration
}

type Game struct {
	screenWidth     int
	screenHeight    int
	gameMap         *maploader.Map
	walls           []shadows.Segment
	player          Player
	camera          Camera // Camera for viewport scrolling
	whiteImg        renderer.Image
	renderer        renderer.Renderer
	inputMgr        renderer.InputManager
	entitiesAtlas   *atlas.Atlas
	objectsAtlas    *atlas.Atlas // Atlas for object/furnishing tiles
	playerSpriteImg renderer.Image
	shadowShader    *ebiten.Shader
	wallTexture     *ebiten.Image // Render target containing just walls

	// Interaction system
	interactionEngine *interaction.Engine
	gameState         *gamestate.GameState
	inventory         *inventory.Inventory

	// Player character
	playerChar *character.Character

	// Turn-based system
	turnManager   *turn.Manager
	playerEntity  *entity.Entity
	entityLibrary *entity.EntityLibrary

	// HUD
	gameHUD *hud.HUD

	// UI state
	messages         []Message
	interactHint     string  // Current interaction hint to display
	interactCooldown float64 // Prevent rapid E key presses
}

func (g *Game) Update() error {
	// Delta time for timers (assuming 60 FPS)
	dt := 1.0 / 60.0

	// Update message timers
	g.updateMessages(dt)

	// Update interaction cooldown
	if g.interactCooldown > 0 {
		g.interactCooldown -= dt
	}

	// Turn-based input handling
	if g.turnManager != nil && g.turnManager.IsPlayerTurn() {
		var dir entity.Direction

		// WASD or arrow keys for movement
		if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			dir = entity.DirNorth
		} else if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			dir = entity.DirSouth
		} else if inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			dir = entity.DirWest
		} else if inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			dir = entity.DirEast
		}

		// Process movement/attack
		if dir != entity.DirNone && g.playerEntity != nil {
			action := turn.Action{
				Type:      turn.ActionMove,
				Actor:     g.playerEntity,
				Direction: dir,
			}
			g.turnManager.ProcessPlayerAction(action)

			// Update pixel position from grid position
			g.syncPlayerPosition()
		}

		// Wait action (space or period)
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyPeriod) {
			action := turn.Action{
				Type:  turn.ActionWait,
				Actor: g.playerEntity,
			}
			g.turnManager.ProcessPlayerAction(action)
		}
	}

	// Update camera to follow player
	g.updateCamera()

	// Handle interactions (E key)
	g.updateInteractions()

	return nil
}

// syncPlayerPosition updates the pixel position from grid position
func (g *Game) syncPlayerPosition() {
	if g.playerEntity == nil {
		return
	}
	tileSize := float64(g.gameMap.Data.TileSize)
	g.player.GridX = g.playerEntity.X
	g.player.GridY = g.playerEntity.Y
	// Center the player in the tile
	g.player.Pos.X = float64(g.playerEntity.X)*tileSize + tileSize/2
	g.player.Pos.Y = float64(g.playerEntity.Y)*tileSize + tileSize/2
}

// updateMessages updates message timers and removes expired messages
func (g *Game) updateMessages(dt float64) {
	var active []Message
	for _, msg := range g.messages {
		msg.TimeLeft -= dt
		if msg.TimeLeft > 0 {
			active = append(active, msg)
		}
	}
	g.messages = active
}

// showMessage adds a new message to be displayed on screen
func (g *Game) showMessage(text string) {
	g.messages = append(g.messages, Message{
		Text:     text,
		TimeLeft: 3.0, // 3 second display
		MaxTime:  3.0,
	})
	log.Printf("Message: %s", text)
}

// updateInteractions handles E key presses and interaction hints
func (g *Game) updateInteractions() {
	if g.gameMap == nil || g.interactionEngine == nil {
		return
	}

	// Find nearby interactable furnishing
	nearby := g.getNearbyInteractable()

	// Update interaction hint
	if nearby != nil && g.interactionEngine != nil {
		hint := g.interactionEngine.GetInteractionHint(nearby, interaction.TriggerInteract)
		if hint != "" {
			g.interactHint = fmt.Sprintf("[E] %s", hint)
		} else {
			g.interactHint = "[E] Interact"
		}
	} else {
		g.interactHint = ""
	}

	// Handle E key press
	if g.inputMgr.IsKeyPressed(renderer.KeyE) && g.interactCooldown <= 0 {
		if nearby != nil {
			if g.interactionEngine.TryInteract(nearby, interaction.TriggerInteract, "") {
				g.interactCooldown = 0.3 // 300ms cooldown between interactions
			}
		}
	}
}

// getNearbyInteractable finds the closest interactable furnishing within range
func (g *Game) getNearbyInteractable() *furnishing.PlacedFurnishing {
	if g.gameMap == nil {
		return nil
	}

	interactRange := 40.0 // Pixels - interaction range
	var closest *furnishing.PlacedFurnishing
	closestDist := interactRange + 1

	tileSize := float64(g.gameMap.Data.TileSize)

	for _, placed := range g.gameMap.Data.PlacedFurnishings {
		if placed == nil || placed.Definition == nil || !placed.Definition.Interactable {
			continue
		}

		// Calculate center of furnishing
		fx := float64(placed.X)*tileSize + tileSize/2
		fy := float64(placed.Y)*tileSize + tileSize/2

		// Distance from player
		dx := fx - g.player.Pos.X
		dy := fy - g.player.Pos.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist < interactRange && dist < closestDist {
			closest = placed
			closestDist = dist
		}
	}

	return closest
}

// lookupFurnishing finds a furnishing by ID
func (g *Game) lookupFurnishing(id string) interaction.InteractableObject {
	if g.gameMap == nil {
		return nil
	}
	for _, placed := range g.gameMap.Data.PlacedFurnishings {
		if placed != nil && placed.ID == id {
			return placed
		}
	}
	return nil
}

// updateCamera centers the camera on the player, clamping to level bounds
func (g *Game) updateCamera() {
	if g.gameMap == nil {
		return
	}

	// Center camera on player
	g.camera.X = g.player.Pos.X - float64(g.screenWidth)/2
	g.camera.Y = g.player.Pos.Y - float64(g.screenHeight)/2

	// Clamp camera to level bounds
	tileSize := float64(g.gameMap.Data.TileSize)
	levelPixelWidth := float64(g.gameMap.Data.Width) * tileSize
	levelPixelHeight := float64(g.gameMap.Data.Height) * tileSize

	// Don't let camera go past level edges
	if g.camera.X < 0 {
		g.camera.X = 0
	}
	if g.camera.Y < 0 {
		g.camera.Y = 0
	}
	if g.camera.X > levelPixelWidth-float64(g.screenWidth) {
		g.camera.X = levelPixelWidth - float64(g.screenWidth)
	}
	if g.camera.Y > levelPixelHeight-float64(g.screenHeight) {
		g.camera.Y = levelPixelHeight - float64(g.screenHeight)
	}

	// Handle case where level is smaller than screen
	if levelPixelWidth < float64(g.screenWidth) {
		g.camera.X = (levelPixelWidth - float64(g.screenWidth)) / 2
	}
	if levelPixelHeight < float64(g.screenHeight) {
		g.camera.Y = (levelPixelHeight - float64(g.screenHeight)) / 2
	}
}

// canMoveTo checks if the player can move to the specified position
func (g *Game) canMoveTo(x, y, radius float64) bool {
	if g.gameMap == nil {
		return true
	}

	tileSize := float64(g.gameMap.Data.TileSize)

	// Check multiple points around the player's hitbox
	// This ensures we can't clip through walls at any angle
	checkPoints := []struct{ dx, dy float64 }{
		{0, 0},                         // Center
		{radius, 0},                    // Right
		{-radius, 0},                   // Left
		{0, radius},                    // Bottom
		{0, -radius},                   // Top
		{radius * 0.7, radius * 0.7},   // Bottom-right
		{-radius * 0.7, radius * 0.7},  // Bottom-left
		{radius * 0.7, -radius * 0.7},  // Top-right
		{-radius * 0.7, -radius * 0.7}, // Top-left
	}

	for _, offset := range checkPoints {
		checkX := x + offset.dx
		checkY := y + offset.dy

		// Convert to tile coordinates
		tileX := int(checkX / tileSize)
		tileY := int(checkY / tileSize)

		// Check bounds
		if tileX < 0 || tileX >= g.gameMap.Data.Width ||
			tileY < 0 || tileY >= g.gameMap.Data.Height {
			return false
		}

		// Check if tile is walkable
		if !g.gameMap.IsWalkable(tileX, tileY) {
			return false
		}
	}

	return true
}

// isTileWalkable checks if a tile is walkable for turn-based movement
func (g *Game) isTileWalkable(x, y int) bool {
	if g.gameMap == nil {
		return true
	}

	// Check bounds
	if x < 0 || x >= g.gameMap.Data.Width || y < 0 || y >= g.gameMap.Data.Height {
		return false
	}

	return g.gameMap.IsWalkable(x, y)
}

// spawnEnemies places enemies in the dungeon
func (g *Game) spawnEnemies() {
	if g.entityLibrary == nil || g.turnManager == nil {
		return
	}

	enemies := g.entityLibrary.GetEnemiesForLevel(1) // Start at level 1
	if len(enemies) == 0 {
		log.Printf("No enemies available for level 1")
		return
	}

	// Use a local random source for spawning
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Spawn a few enemies in walkable tiles away from player
	numEnemies := 3 + rng.Intn(5) // 3-7 enemies
	spawned := 0
	attempts := 0
	maxAttempts := numEnemies * 50

	for spawned < numEnemies && attempts < maxAttempts {
		attempts++

		// Pick random position
		x := rng.Intn(g.gameMap.Data.Width)
		y := rng.Intn(g.gameMap.Data.Height)

		// Check if walkable
		if !g.isTileWalkable(x, y) {
			continue
		}

		// Check distance from player (at least 5 tiles)
		dx := x - g.player.GridX
		dy := y - g.player.GridY
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		dist := dx + dy
		if dist < 5 {
			continue
		}

		// Check no entity already there
		if g.turnManager.GetEntityAtPosition(x, y) != nil {
			continue
		}

		// Pick random enemy type (weighted)
		totalWeight := 0
		for _, e := range enemies {
			totalWeight += e.SpawnWeight
		}

		roll := rng.Intn(totalWeight)
		var selectedDef *entity.EntityDefinition
		cumWeight := 0
		for _, e := range enemies {
			cumWeight += e.SpawnWeight
			if roll < cumWeight {
				selectedDef = e
				break
			}
		}

		if selectedDef == nil {
			selectedDef = enemies[0]
		}

		// Spawn the enemy
		enemyID := fmt.Sprintf("%s_%d", selectedDef.ID, spawned)
		enemy := selectedDef.SpawnEntity(enemyID, x, y)
		g.turnManager.AddEntity(enemy)

		log.Printf("Spawned %s at (%d, %d)", enemy.Name, x, y)
		spawned++
	}

	log.Printf("Spawned %d enemies", spawned)
}

func (g *Game) Draw(screen renderer.Image) {
	// MULTI-PASS RENDERING FOR PIXEL-PERFECT SHADOWS

	// Step 1: Draw ONLY floors to the screen
	g.drawFloorsOnly(screen)

	// Step 2: Draw furnishings/objects on top of floors
	g.drawFurnishings(screen)

	// Step 3: Render walls to wallTexture for shader input
	g.wallTexture.Clear()
	g.drawWallsToTexture(g.wallTexture)

	// Step 3: Apply shadow shader - darkens occluded areas
	/**if ebitenImg, ok := screen.(*ebitenrenderer.EbitenImage); ok && g.shadowShader != nil {
		ebitenScreen := ebitenImg.GetEbitenImage()

		// DEBUG: Log player position (throttle to ~1/sec to avoid spam)
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			log.Printf("DEBUG PlayerPos: X=%.2f Y=%.2f (screen: %dx%d)",
				g.player.Pos.X, g.player.Pos.Y, g.screenWidth, g.screenHeight)
		}

		opts := &ebiten.DrawRectShaderOptions{}
		opts.Uniforms = map[string]interface{}{
			"PlayerPos":   []float32{float32(g.player.Pos.X), float32(g.player.Pos.Y)},
			"MaxDistance": float32(g.screenWidth + g.screenHeight),
		}
		opts.Images[0] = g.wallTexture
		ebitenScreen.DrawRectShader(g.screenWidth, g.screenHeight, g.shadowShader, opts)
	}*/

	// Step 4: Draw walls that have clear line of sight ON TOP of shadows
	g.drawVisibleWalls(screen)

	// Step 5: Draw enemies
	g.drawEntities(screen)

	// Step 6: Draw player character on top of everything
	// Calculate player screen position with camera offset
	playerScreenX := g.player.Pos.X - g.camera.X
	playerScreenY := g.player.Pos.Y - g.camera.Y

	if g.playerSpriteImg != nil {
		// Draw player sprite centered on player position
		spriteSize := 32.0 // Tile size from atlas (32x32)
		opts := &renderer.DrawImageOptions{}
		opts.GeoM = renderer.NewGeoM()
		opts.GeoM.Translate(playerScreenX-spriteSize/2, playerScreenY-spriteSize/2)
		screen.DrawImage(g.playerSpriteImg, opts)
	} else {
		// Fallback to circle if sprite not loaded (larger for 32x32 tiles)
		g.renderer.FillCircle(screen,
			float32(playerScreenX),
			float32(playerScreenY),
			14,
			color.RGBA{255, 255, 100, 255})

		g.renderer.StrokeCircle(screen,
			float32(playerScreenX),
			float32(playerScreenY),
			14,
			2,
			color.RGBA{200, 200, 50, 255})
	}

	// Step 7: Draw UI elements (interaction hints, messages)
	g.drawUI(screen)

	// Step 8: Draw HUD
	g.drawHUD(screen)
}

// drawHUD renders the heads-up display
func (g *Game) drawHUD(screen renderer.Image) {
	if g.gameHUD == nil {
		return
	}

	// Get ebiten image from renderer
	ebitenImg, ok := screen.(*ebitenrenderer.EbitenImage)
	if !ok {
		return
	}

	g.gameHUD.Draw(ebitenImg.GetEbitenImage())
}

// drawEntities renders all non-player entities (enemies, NPCs)
func (g *Game) drawEntities(screen renderer.Image) {
	if g.turnManager == nil {
		return
	}

	tileSize := float64(g.gameMap.Data.TileSize)

	for _, e := range g.turnManager.GetEntities() {
		// Skip the player (drawn separately)
		if e.Type == entity.TypePlayer {
			continue
		}

		// Skip dead entities
		if !e.IsAlive() {
			continue
		}

		// Calculate screen position
		worldX := float64(e.X)*tileSize + tileSize/2
		worldY := float64(e.Y)*tileSize + tileSize/2
		screenX := worldX - g.camera.X
		screenY := worldY - g.camera.Y

		// Skip if off screen
		if screenX < -tileSize || screenX > float64(g.screenWidth)+tileSize ||
			screenY < -tileSize || screenY > float64(g.screenHeight)+tileSize {
			continue
		}

		// Try to get sprite from atlas
		var sprite renderer.Image
		if g.entitiesAtlas != nil && e.SpriteName != "" {
			sprite, _ = g.entitiesAtlas.GetTileSubImageByName(e.SpriteName)
		}

		if sprite != nil {
			// Draw sprite
			opts := &renderer.DrawImageOptions{}
			opts.GeoM = renderer.NewGeoM()
			opts.GeoM.Translate(screenX-tileSize/2, screenY-tileSize/2)
			screen.DrawImage(sprite, opts)
		} else {
			// Fallback: draw colored circle based on faction
			var clr color.RGBA
			switch e.Faction {
			case entity.FactionEnemy:
				clr = color.RGBA{255, 80, 80, 255} // Red for enemies
			case entity.FactionNeutral:
				clr = color.RGBA{100, 200, 100, 255} // Green for NPCs
			default:
				clr = color.RGBA{150, 150, 150, 255} // Gray for unknown
			}

			g.renderer.FillCircle(screen,
				float32(screenX),
				float32(screenY),
				12,
				clr)
		}

		// Draw HP bar above entity if damaged
		if e.CurrentHP < e.MaxHP {
			barWidth := 24.0
			barHeight := 4.0
			barX := screenX - barWidth/2
			barY := screenY - tileSize/2 - 8

			// Background (red)
			g.renderer.FillCircle(screen, float32(barX+barWidth/2), float32(barY+barHeight/2), float32(barWidth/2), color.RGBA{100, 0, 0, 255})

			// Health (green)
			healthPct := float64(e.CurrentHP) / float64(e.MaxHP)
			healthWidth := barWidth * healthPct
			if healthWidth > 0 {
				g.renderer.FillCircle(screen, float32(barX+healthWidth/2), float32(barY+barHeight/2), float32(healthWidth/2), color.RGBA{0, 200, 0, 255})
			}
		}
	}
}

// drawUI renders interaction hints and messages
func (g *Game) drawUI(screen renderer.Image) {
	ebitenImg, ok := screen.(*ebitenrenderer.EbitenImage)
	if !ok {
		return
	}
	ebitenScreen := ebitenImg.GetEbitenImage()

	// Draw interaction hint at bottom of screen
	if g.interactHint != "" {
		g.drawTextWithShadow(ebitenScreen, g.interactHint,
			float64(g.screenWidth)/2, float64(g.screenHeight)-40,
			color.RGBA{255, 255, 255, 255}, true)
	}

	// Draw messages from top
	y := 20.0
	for _, msg := range g.messages {
		alpha := uint8(255)
		if msg.TimeLeft < 0.5 {
			alpha = uint8(msg.TimeLeft / 0.5 * 255)
		}
		col := color.RGBA{255, 255, 200, alpha}
		g.drawTextWithShadow(ebitenScreen, msg.Text, float64(g.screenWidth)/2, y, col, true)
		y += 20
	}

	// Draw inventory hint in corner (if items exist)
	if g.inventory != nil && !g.inventory.IsEmpty() {
		items := g.inventory.GetAllItems()
		invText := "Inventory: "
		for i, item := range items {
			if i > 0 {
				invText += ", "
			}
			if item.Count > 1 {
				invText += fmt.Sprintf("%s x%d", item.ItemName, item.Count)
			} else {
				invText += item.ItemName
			}
		}
		g.drawTextWithShadow(ebitenScreen, invText, 10, 10, color.RGBA{200, 200, 200, 200}, false)
	}
}

// drawTextWithShadow draws text with a drop shadow for readability
func (g *Game) drawTextWithShadow(screen *ebiten.Image, str string, x, y float64, _ color.RGBA, centered bool) {
	// For centered text, approximate center position
	if centered {
		x -= float64(len(str)) * 3.0 // Rough approximation for debug font
	}

	// Use simple debug print (always works, no font loading required)
	ebitenutil.DebugPrintAt(screen, str, int(x), int(y))
}

func (g *Game) drawFloorsOnly(screen renderer.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize

	// Calculate visible tile range for culling
	startTileX := int(g.camera.X) / tileSize
	startTileY := int(g.camera.Y) / tileSize
	endTileX := (int(g.camera.X)+g.screenWidth)/tileSize + 1
	endTileY := (int(g.camera.Y)+g.screenHeight)/tileSize + 1

	// Clamp to level bounds
	if startTileX < 0 {
		startTileX = 0
	}
	if startTileY < 0 {
		startTileY = 0
	}
	if endTileX > g.gameMap.Data.Width {
		endTileX = g.gameMap.Data.Width
	}
	if endTileY > g.gameMap.Data.Height {
		endTileY = g.gameMap.Data.Height
	}

	// Only draw floor tiles where the tile data specifies a floor
	// Empty/void tiles are not rendered (stay black)
	for y := startTileY; y < endTileY; y++ {
		for x := startTileX; x < endTileX; x++ {
			tileName, err := g.gameMap.GetTileAt(x, y)
			if err != nil || tileName == "" {
				continue // Skip void/empty tiles
			}

			// Get tile definition to check if it's a floor type
			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			// Only draw if this is a walkable/floor tile (not a wall)
			if blocksSight, ok := tile.Properties["blocks_sight"].(bool); ok && blocksSight {
				continue // Skip walls - they'll be drawn separately
			}

			subImg := g.gameMap.Atlas.GetTileSubImage(tile)
			// Apply camera offset
			screenX := float64(x*tileSize) - g.camera.X
			screenY := float64(y*tileSize) - g.camera.Y

			opts := &renderer.DrawImageOptions{}
			opts.GeoM = renderer.NewGeoM()
			opts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(subImg, opts)
		}
	}
}

// drawFurnishings renders all placed furnishings/objects in the level
func (g *Game) drawFurnishings(screen renderer.Image) {
	if g.gameMap == nil || g.objectsAtlas == nil {
		return
	}

	// Check if there are any furnishings to draw
	if len(g.gameMap.Data.PlacedFurnishings) == 0 {
		return
	}

	tileSize := g.gameMap.Data.TileSize

	for _, placed := range g.gameMap.Data.PlacedFurnishings {
		if placed == nil || placed.Definition == nil {
			continue
		}

		// Calculate world position for visibility check
		worldX := float64(placed.X * tileSize)
		worldY := float64(placed.Y * tileSize)

		// Cull furnishings outside visible area
		if worldX+float64(tileSize) < g.camera.X || worldX > g.camera.X+float64(g.screenWidth) ||
			worldY+float64(tileSize) < g.camera.Y || worldY > g.camera.Y+float64(g.screenHeight) {
			continue
		}

		// Get the tile name based on current state (supports state-based sprites)
		tileName := placed.GetCurrentTileName()
		if tileName == "" {
			continue
		}

		// Look up the tile in the objects atlas
		tile, ok := g.objectsAtlas.GetTile(tileName)
		if !ok {
			// Tile not found in atlas - skip this furnishing
			log.Printf("Warning: tile '%s' not found in objects atlas for furnishing '%s' (state: %s)",
				tileName, placed.Definition.Name, placed.State)
			continue
		}

		// Get the tile sprite
		subImg := g.objectsAtlas.GetTileSubImage(tile)

		// Calculate screen position with camera offset
		screenX := worldX - g.camera.X
		screenY := worldY - g.camera.Y

		opts := &renderer.DrawImageOptions{}
		opts.GeoM = renderer.NewGeoM()
		opts.GeoM.Translate(screenX, screenY)
		screen.DrawImage(subImg, opts)
	}
}

func (g *Game) drawTiles(screen renderer.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize

	// Pass 1: Draw floor layer (fills entire map with floor tile)
	if g.gameMap.Data.FloorTile != "" {
		floorTile, ok := g.gameMap.Atlas.GetTile(g.gameMap.Data.FloorTile)
		if ok {
			floorImg := g.gameMap.Atlas.GetTileSubImage(floorTile)
			for y := 0; y < g.gameMap.Data.Height; y++ {
				for x := 0; x < g.gameMap.Data.Width; x++ {
					screenX := float64(x * tileSize)
					screenY := float64(y * tileSize)

					opts := &renderer.DrawImageOptions{}
					opts.GeoM = renderer.NewGeoM()
					opts.GeoM.Translate(screenX, screenY)
					screen.DrawImage(floorImg, opts)
				}
			}
		}
	}

	// Pass 2: Draw walls/objects layer (only non-empty tiles)
	for y := 0; y < g.gameMap.Data.Height; y++ {
		for x := 0; x < g.gameMap.Data.Width; x++ {
			tileName, err := g.gameMap.GetTileAt(x, y)
			if err != nil {
				continue
			}

			// Skip empty tiles (let floor show through)
			if tileName == "" {
				continue
			}

			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			subImg := g.gameMap.Atlas.GetTileSubImage(tile)

			screenX := float64(x * tileSize)
			screenY := float64(y * tileSize)

			opts := &renderer.DrawImageOptions{}
			opts.GeoM = renderer.NewGeoM()
			opts.GeoM.Translate(screenX, screenY)

			screen.DrawImage(subImg, opts)
		}
	}
}

// drawWallsToTexture renders only wall tiles (sight-blocking) to a texture for shader input
// The shader will sample this texture's alpha channel to detect walls during raycasting
func (g *Game) drawWallsToTexture(texture *ebiten.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize
	drawnTiles := make(map[string]bool) // Track which tiles we've drawn

	// Use wall segments to identify all wall tiles
	for _, wall := range g.walls {
		// Get tiles covered by this segment
		tilesToDraw := wall.TilesCovered
		if len(tilesToDraw) == 0 {
			// Fallback to single tile if TilesCovered is empty
			tilesToDraw = []shadows.Coord{{X: wall.TileX, Y: wall.TileY}}
		}

		for _, tileCoord := range tilesToDraw {
			tileKey := fmt.Sprintf("%d,%d", tileCoord.X, tileCoord.Y)
			if drawnTiles[tileKey] {
				continue // Already drew this tile
			}
			drawnTiles[tileKey] = true

			// Get the wall tile at this position
			tileName, err := g.gameMap.GetTileAt(tileCoord.X, tileCoord.Y)
			if err != nil || tileName == "" {
				continue
			}

			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			subImg := g.gameMap.Atlas.GetTileSubImage(tile)

			screenX := float64(tileCoord.X * tileSize)
			screenY := float64(tileCoord.Y * tileSize)

			// Extract underlying ebiten.Image to draw wall tiles
			if ebitenImg, ok := subImg.(*ebitenrenderer.EbitenImage); ok {
				ebitenSubImg := ebitenImg.GetEbitenImage()

				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(screenX, screenY)
				texture.DrawImage(ebitenSubImg, opts)
			}
		}
	}
}

func (g *Game) drawVisibleWalls(screen renderer.Image) {
	if g.gameMap == nil || g.gameMap.Atlas == nil {
		return
	}

	tileSize := g.gameMap.Data.TileSize
	tileSizeF := float64(tileSize)

	// Draw wall tiles that are visible (not in shadow)
	// Check each wall tile to see if it's obscured by shadows
	drawnTiles := make(map[string]bool) // Track which tiles we've already drawn

	for _, wall := range g.walls {
		// Iterate through all tiles covered by this segment (for merged segments)
		tilesToDraw := wall.TilesCovered
		if len(tilesToDraw) == 0 {
			// Fallback to single tile if TilesCovered is empty (shouldn't happen with new code)
			tilesToDraw = []shadows.Coord{{X: wall.TileX, Y: wall.TileY}}
		}

		for _, tileCoord := range tilesToDraw {
			// Cull tiles outside visible area
			worldX := float64(tileCoord.X) * tileSizeF
			worldY := float64(tileCoord.Y) * tileSizeF
			if worldX+tileSizeF < g.camera.X || worldX > g.camera.X+float64(g.screenWidth) ||
				worldY+tileSizeF < g.camera.Y || worldY > g.camera.Y+float64(g.screenHeight) {
				continue
			}

			tileKey := fmt.Sprintf("%d,%d", tileCoord.X, tileCoord.Y)
			if drawnTiles[tileKey] {
				continue // Already drew this tile
			}

			// Special case: Check if this is a corner wall
			// A corner has walls on two perpendicular adjacent sides
			isCorner, adjacentWalls := g.isCornerWall(tileCoord.X, tileCoord.Y)

			var anyVisible bool

			if isCorner {
				// For corner walls: visible if ANY adjacent wall forming the corner is visible
				// This prevents flickering as corners are stable when either wall is visible
				anyVisible = false
				for _, adjCoord := range adjacentWalls {
					adjKey := fmt.Sprintf("%d,%d", adjCoord.X, adjCoord.Y)
					if drawnTiles[adjKey] {
						// Adjacent wall is already drawn, so it's visible
						anyVisible = true
						break
					}
				}

				// If adjacent walls haven't been checked yet, fall back to normal sampling
				if !anyVisible {
					// Check a few sample points on the corner tile itself
					tileBaseX := float64(tileCoord.X * tileSize)
					tileBaseY := float64(tileCoord.Y * tileSize)
					tileSizeFloat := float64(tileSize)

					cornerSamples := []shadows.Point{
						{tileBaseX + 4, tileBaseY + 4},
						{tileBaseX + tileSizeFloat - 4, tileBaseY + 4},
						{tileBaseX + 4, tileBaseY + tileSizeFloat - 4},
						{tileBaseX + tileSizeFloat - 4, tileBaseY + tileSizeFloat - 4},
					}

					for _, point := range cornerSamples {
						if !g.isPointInShadow(point, tileCoord.X, tileCoord.Y) {
							anyVisible = true
							break
						}
					}
				}
			} else {
				// Normal wall: sample multiple points
				tileBaseX := float64(tileCoord.X * tileSize)
				tileBaseY := float64(tileCoord.Y * tileSize)
				tileSizeFloat := float64(tileSize)

				samplePoints := []shadows.Point{
					// Center
					{tileBaseX + tileSizeFloat/2, tileBaseY + tileSizeFloat/2},

					// 4 corners (inset 2px)
					{tileBaseX + 2, tileBaseY + 2},
					{tileBaseX + tileSizeFloat - 2, tileBaseY + 2},
					{tileBaseX + 2, tileBaseY + tileSizeFloat - 2},
					{tileBaseX + tileSizeFloat - 2, tileBaseY + tileSizeFloat - 2},

					// Edge midpoints (inset 2px)
					{tileBaseX + tileSizeFloat/2, tileBaseY + 2},
					{tileBaseX + tileSizeFloat/2, tileBaseY + tileSizeFloat - 2},
					{tileBaseX + 2, tileBaseY + tileSizeFloat/2},
					{tileBaseX + tileSizeFloat - 2, tileBaseY + tileSizeFloat/2},

					// Quarter points
					{tileBaseX + tileSizeFloat/4, tileBaseY + tileSizeFloat/4},
					{tileBaseX + 3*tileSizeFloat/4, tileBaseY + tileSizeFloat/4},
					{tileBaseX + tileSizeFloat/4, tileBaseY + 3*tileSizeFloat/4},
					{tileBaseX + 3*tileSizeFloat/4, tileBaseY + 3*tileSizeFloat/4},
				}

				anyVisible = false
				for _, point := range samplePoints {
					if !g.isPointInShadow(point, tileCoord.X, tileCoord.Y) {
						anyVisible = true
						break
					}
				}
			}

			/*if !anyVisible {
				continue // Not visible, don't draw
			}*/

			// Get the wall tile at this position
			tileName, err := g.gameMap.GetTileAt(tileCoord.X, tileCoord.Y)
			if err != nil || tileName == "" {
				continue
			}

			tile, ok := g.gameMap.Atlas.GetTile(tileName)
			if !ok {
				continue
			}

			subImg := g.gameMap.Atlas.GetTileSubImage(tile)

			// Apply camera offset
			screenX := float64(tileCoord.X*tileSize) - g.camera.X
			screenY := float64(tileCoord.Y*tileSize) - g.camera.Y

			opts := &renderer.DrawImageOptions{}
			opts.GeoM = renderer.NewGeoM()
			opts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(subImg, opts)

			drawnTiles[tileKey] = true
		}
	}
}

func (g *Game) isCornerWall(tileX, tileY int) (bool, []shadows.Coord) {
	// Check if this wall tile is a corner (has walls on 2 perpendicular adjacent sides)
	// Returns true if corner, and the list of adjacent walls forming the corner

	adjacentWalls := []shadows.Coord{}

	// Check 4 cardinal directions
	checkWall := func(x, y int) bool {
		tileName, err := g.gameMap.GetTileAt(x, y)
		if err != nil || tileName == "" {
			return false
		}
		if tile, ok := g.gameMap.Atlas.GetTile(tileName); ok {
			if blocksSight, ok := tile.Properties["blocks_sight"].(bool); ok && blocksSight {
				return true
			}
		}
		return false
	}

	hasNorth := checkWall(tileX, tileY-1)
	hasSouth := checkWall(tileX, tileY+1)
	hasEast := checkWall(tileX+1, tileY)
	hasWest := checkWall(tileX-1, tileY)

	// Check for perpendicular pairs (corners)
	if (hasNorth && hasEast) || (hasNorth && hasWest) || (hasSouth && hasEast) || (hasSouth && hasWest) {
		// This is a corner - collect the adjacent walls
		if hasNorth {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX, Y: tileY - 1})
		}
		if hasSouth {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX, Y: tileY + 1})
		}
		if hasEast {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX + 1, Y: tileY})
		}
		if hasWest {
			adjacentWalls = append(adjacentWalls, shadows.Coord{X: tileX - 1, Y: tileY})
		}
		return true, adjacentWalls
	}

	return false, nil
}

func (g *Game) isPointInShadow(point shadows.Point, ignoreTileX, ignoreTileY int) bool {
	// Pixel-perfect raycasting to match shader behavior
	// Cast ray from player to point and check if it hits a wall
	// ignoreTileX/Y: The tile we're checking visibility for (don't count it as an occluder)

	dx := point.X - g.player.Pos.X
	dy := point.Y - g.player.Pos.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance < 1.0 {
		return false // Player position is never in shadow
	}

	// Build ignore set: the target tile plus orthogonally adjacent wall tiles
	// This prevents walls in a line from blocking each other (hallway effect)
	ignoreSet := make(map[string]bool)
	ignoreSet[fmt.Sprintf("%d,%d", ignoreTileX, ignoreTileY)] = true

	// Check 4 orthogonal neighbors and add to ignore set if they're walls
	checkAndIgnore := func(x, y int) {
		tileName, err := g.gameMap.GetTileAt(x, y)
		if err == nil && tileName != "" {
			if tile, ok := g.gameMap.Atlas.GetTile(tileName); ok {
				if blocksSight, ok := tile.Properties["blocks_sight"].(bool); ok && blocksSight {
					ignoreSet[fmt.Sprintf("%d,%d", x, y)] = true
				}
			}
		}
	}

	checkAndIgnore(ignoreTileX-1, ignoreTileY) // West
	checkAndIgnore(ignoreTileX+1, ignoreTileY) // East
	checkAndIgnore(ignoreTileX, ignoreTileY-1) // North
	checkAndIgnore(ignoreTileX, ignoreTileY+1) // South

	// Normalize direction
	dirX := dx / distance
	dirY := dy / distance

	// Sample along the ray (matching shader logic)
	for t := 1.0; t < distance-0.5; t += 1.0 {
		sampleX := g.player.Pos.X + dirX*t
		sampleY := g.player.Pos.Y + dirY*t

		// Check if this sample point is inside any wall tile
		tileX := int(sampleX / float64(g.gameMap.Data.TileSize))
		tileY := int(sampleY / float64(g.gameMap.Data.TileSize))

		tileKey := fmt.Sprintf("%d,%d", tileX, tileY)

		// Skip tiles in our ignore set
		if ignoreSet[tileKey] {
			continue
		}

		// Check if this tile is a wall
		tileName, err := g.gameMap.GetTileAt(tileX, tileY)
		if err == nil && tileName != "" {
			// Check if it's actually a wall (has blocks_sight property)
			if tile, ok := g.gameMap.Atlas.GetTile(tileName); ok {
				if blocksSight, ok := tile.Properties["blocks_sight"].(bool); ok && blocksSight {
					return true // Hit a wall before reaching the point
				}
			}
		}
	}

	return false // Clear line of sight
}

// extendToScreenEdge extends a ray from 'from' through 'to' until it hits a screen edge
func (g *Game) extendToScreenEdge(from, to shadows.Point, maxDist float64) shadows.Point {
	// Direction vector
	dx := to.X - from.X
	dy := to.Y - from.Y

	// Normalize and extend
	length := maxDist * 2.0
	if dx != 0 || dy != 0 {
		currentLen := (dx*dx + dy*dy)
		if currentLen > 0.001 {
			scale := length / currentLen
			dx *= scale
			dy *= scale
		}
	}

	return shadows.Point{
		X: from.X + dx,
		Y: from.Y + dy,
	}
}

func (g *Game) drawPolygon(dst renderer.Image, points []shadows.Point, c color.RGBA) {
	if len(points) < 3 {
		return
	}

	// Convert points to float32 path
	path := vector.Path{}
	path.MoveTo(float32(points[0].X), float32(points[0].Y))
	for i := 1; i < len(points); i++ {
		path.LineTo(float32(points[i].X), float32(points[i].Y))
	}
	path.Close()

	// Fill with anti-aliasing disabled to avoid edge artifacts
	ebitenVertexes, indexes := path.AppendVerticesAndIndicesForFilling(nil, nil)

	if g.whiteImg == nil {
		g.whiteImg = g.renderer.NewImage(1, 1)
		g.whiteImg.Fill(color.White)
	}

	// Convert ebiten vertices to renderer vertices and apply color
	vertexes := make([]renderer.Vertex, len(ebitenVertexes))
	for i := range ebitenVertexes {
		vertexes[i] = renderer.Vertex{
			DstX:   ebitenVertexes[i].DstX,
			DstY:   ebitenVertexes[i].DstY,
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(c.R) / 255,
			ColorG: float32(c.G) / 255,
			ColorB: float32(c.B) / 255,
			ColorA: float32(c.A) / 255,
		}
	}

	opts := &renderer.DrawTrianglesOptions{
		AntiAlias: false,
	}
	dst.DrawTriangles(vertexes, indexes, g.whiteImg, opts)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Return current screen dimensions (GameManager handles resize)
	return g.screenWidth, g.screenHeight
}

func main() {
	screenWidth := 1280
	screenHeight := 800

	// Initialize the renderer backend (ebiten)
	rend := ebitenrenderer.NewRenderer()
	inputMgr := ebitenrenderer.NewInputManager()
	loader := ebitenrenderer.NewResourceLoader()
	engine := ebitenrenderer.NewEngine()

	// Enable window resizing and maximizing
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Scan data directory for available games
	log.Println("Scanning data directory for available games...")
	games, err := gamescanner.ScanDataDirectory("data")
	if err != nil {
		log.Fatalf("Failed to scan data directory: %v", err)
	}

	// Create the main menu
	mainMenu := menu.NewMainMenu(games, rend, inputMgr, screenWidth, screenHeight)

	// Create the game manager
	gameManager := &GameManager{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		state:        menu.StateMainMenu,
		mainMenu:     mainMenu,
		renderer:     rend,
		inputMgr:     inputMgr,
		loader:       loader,
	}

	// Set up the window
	engine.SetWindowSize(screenWidth, screenHeight)
	engine.SetWindowTitle("Outpost9 - Main Menu")

	log.Println("Starting game...")
	if err := engine.RunGame(gameManager); err != nil {
		log.Fatal(err)
	}
}
