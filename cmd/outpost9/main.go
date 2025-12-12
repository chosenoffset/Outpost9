package main

import (
	"log"
	"os"

	"chosenoffset.com/outpost9/internal/game"
	"chosenoffset.com/outpost9/internal/gamescanner"
	ebitenrender "chosenoffset.com/outpost9/internal/render/ebiten"
	"chosenoffset.com/outpost9/internal/ui/menu"
)

func main() {
	screenWidth := 1280
	screenHeight := 800

	// Initialize the renderer backend (ebiten)
	renderer := ebitenrender.NewRenderer()
	inputMgr := ebitenrender.NewInputManager()
	loader := ebitenrender.NewResourceLoader()
	engine := ebitenrender.NewEngine()

	// Load shader source files
	shadowShaderSrc, err := os.ReadFile("shaders/sight_shadows.kage")
	if err != nil {
		log.Printf("Warning: Failed to load shadow shader: %v", err)
	}
	lightingShaderSrc, err := os.ReadFile("shaders/lighting.kage")
	if err != nil {
		log.Printf("Warning: Failed to load lighting shader: %v", err)
	}

	// Scan data directory for available games
	log.Println("Scanning data directory for available games...")
	games, err := gamescanner.ScanDataDirectory("data")
	if err != nil {
		log.Fatalf("Failed to scan data directory: %v", err)
	}

	// Create the main menu
	mainMenu := menu.NewMainMenu(games, renderer, inputMgr, screenWidth, screenHeight)

	// Create the game manager
	gameManager := game.NewManager(renderer, inputMgr, loader, screenWidth, screenHeight)
	gameManager.SetMainMenu(mainMenu)
	gameManager.SetShaderSources(shadowShaderSrc, lightingShaderSrc)

	// Set up the window
	engine.SetWindowSize(screenWidth, screenHeight)
	engine.SetWindowTitle("Outpost9 - Main Menu")
	engine.SetWindowResizable(true)

	log.Println("Starting game...")
	if err := engine.RunGame(gameManager); err != nil {
		log.Fatal(err)
	}
}
