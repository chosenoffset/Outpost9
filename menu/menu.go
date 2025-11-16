package menu

import (
	"fmt"
	"image/color"

	"chosenoffset.com/outpost9/gamescanner"
	"chosenoffset.com/outpost9/renderer"
)

// GameState represents the current state of the game.
type GameState int

const (
	StateMainMenu GameState = iota
	StatePlaying
)

// Selection represents a game and room library selection from the menu.
type Selection struct {
	GameDir         string
	RoomLibraryFile string
}

// MainMenu represents the main menu screen.
type MainMenu struct {
	games           []gamescanner.GameEntry
	selectedGame    int
	selectedLibrary int
	renderer        renderer.Renderer
	input           renderer.InputManager
	screenWidth     int
	screenHeight    int
	lastMouseClick  bool
}

// NewMainMenu creates a new main menu.
func NewMainMenu(games []gamescanner.GameEntry, r renderer.Renderer, input renderer.InputManager, width, height int) *MainMenu {
	return &MainMenu{
		games:           games,
		selectedGame:    0,
		selectedLibrary: 0,
		renderer:        r,
		input:           input,
		screenWidth:     width,
		screenHeight:    height,
		lastMouseClick:  false,
	}
}

// Update updates the menu state based on user input.
// Returns true if a game was selected, false otherwise.
func (m *MainMenu) Update() (selected bool, selection Selection) {
	if len(m.games) == 0 {
		return false, Selection{}
	}

	mouseX, mouseY := m.input.GetCursorPosition()
	mousePressed := m.input.IsMouseButtonPressed(renderer.MouseButtonLeft)

	// Detect mouse click (button pressed this frame but not last frame)
	mouseClicked := mousePressed && !m.lastMouseClick
	m.lastMouseClick = mousePressed

	if mouseClicked {
		// Check if click is on a game entry
		startY := 100
		entryHeight := 30
		gameListY := startY

		for i, game := range m.games {
			gameY := gameListY + i*(entryHeight+40)
			gameRect := rect{x: 50, y: gameY, w: 300, h: 25}

			if pointInRect(mouseX, mouseY, gameRect) {
				m.selectedGame = i
				m.selectedLibrary = 0
				break
			}

			// Check library clicks if this game is selected
			if i == m.selectedGame {
				libraryY := gameY + 30

				// Check room library clicks
				for j := range game.RoomLibraries {
					libRect := rect{x: 70, y: libraryY + j*entryHeight, w: 280, h: 25}
					if pointInRect(mouseX, mouseY, libRect) {
						m.selectedLibrary = j
						// If clicking on a library, don't start yet
						break
					}
				}

				// Check start button click
				startBtnY := libraryY + len(game.RoomLibraries)*entryHeight + 10
				startBtnRect := rect{x: 70, y: startBtnY, w: 200, h: 30}
				if pointInRect(mouseX, mouseY, startBtnRect) {
					return true, Selection{
						GameDir:         game.Dir,
						RoomLibraryFile: game.RoomLibraries[m.selectedLibrary],
					}
				}
			}
		}
	}

	// Keyboard navigation
	if m.input.IsKeyPressed(renderer.KeyUp) {
		// TODO: Add debouncing for keyboard input
	}
	if m.input.IsKeyPressed(renderer.KeyDown) {
		// TODO: Add debouncing for keyboard input
	}
	if m.input.IsKeyPressed(renderer.KeySpace) {
		// Start selected game
		if len(m.games) > 0 && m.selectedGame < len(m.games) {
			game := m.games[m.selectedGame]
			if m.selectedLibrary < len(game.RoomLibraries) {
				return true, Selection{
					GameDir:         game.Dir,
					RoomLibraryFile: game.RoomLibraries[m.selectedLibrary],
				}
			}
		}
	}

	return false, Selection{}
}

// Draw renders the menu to the screen.
func (m *MainMenu) Draw(screen renderer.Image) {
	// Clear screen with dark background
	screen.Fill(color.RGBA{20, 20, 30, 255})

	// Draw title
	titleColor := color.RGBA{255, 255, 255, 255}
	m.renderer.DrawText(screen, "OUTPOST 9", 50, 30, titleColor, 3.0)
	m.renderer.DrawText(screen, "Select a Game", 50, 70, titleColor, 1.5)

	if len(m.games) == 0 {
		noGamesColor := color.RGBA{255, 100, 100, 255}
		m.renderer.DrawText(screen, "No games found in data directory!", 50, 120, noGamesColor, 1.2)
		m.renderer.DrawText(screen, "Please add game data to the 'data' folder.", 50, 145, noGamesColor, 1.0)
		return
	}

	// Draw game list
	startY := 100
	entryHeight := 30
	currentY := startY

	for i, game := range m.games {
		isSelected := i == m.selectedGame

		// Draw game name
		gameColor := color.RGBA{200, 200, 255, 255}
		if isSelected {
			gameColor = color.RGBA{100, 255, 100, 255}
		}

		numLibraries := len(game.RoomLibraries)
		gameName := fmt.Sprintf("%s (%d room libraries)", game.Name, numLibraries)
		m.renderer.DrawText(screen, gameName, 50, currentY, gameColor, 1.5)
		currentY += 30

		// Draw room libraries if selected
		if isSelected {
			// Draw room libraries
			for j, library := range game.RoomLibraries {
				librarySelected := j == m.selectedLibrary
				libraryColor := color.RGBA{200, 150, 255, 255} // Purple for procedural
				if librarySelected {
					libraryColor = color.RGBA{255, 255, 100, 255}
					// Draw selection indicator
					m.renderer.DrawText(screen, ">", 50, currentY, libraryColor, 1.2)
				}

				libraryText := fmt.Sprintf("  %s", library)
				m.renderer.DrawText(screen, libraryText, 70, currentY, libraryColor, 1.2)
				currentY += entryHeight
			}

			// Draw start button
			currentY += 10
			startBtnColor := color.RGBA{100, 255, 100, 255}
			m.renderer.DrawText(screen, "[Press SPACE or Click to Start]", 70, currentY, startBtnColor, 1.2)
			currentY += 40
		} else {
			currentY += 10
		}
	}

	// Draw instructions
	instructionY := m.screenHeight - 60
	instructionColor := color.RGBA{150, 150, 150, 255}
	m.renderer.DrawText(screen, "Click on a game to expand, then click a level to select it.", 20, instructionY, instructionColor, 1.0)
	m.renderer.DrawText(screen, "Press SPACE or click the start button to begin.", 20, instructionY+20, instructionColor, 1.0)
}

// Helper types and functions

type rect struct {
	x, y, w, h int
}

func pointInRect(px, py int, r rect) bool {
	return px >= r.x && px <= r.x+r.w && py >= r.y && py <= r.y+r.h
}
