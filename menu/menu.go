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

// Selection represents a game and level selection from the menu.
type Selection struct {
	GameDir       string
	LevelFile     string
	IsRoomLibrary bool // If true, LevelFile is a room library to generate from
}

// MainMenu represents the main menu screen.
type MainMenu struct {
	games          []gamescanner.GameEntry
	selectedGame   int
	selectedLevel  int
	renderer       renderer.Renderer
	input          renderer.InputManager
	screenWidth    int
	screenHeight   int
	lastMouseClick bool
}

// NewMainMenu creates a new main menu.
func NewMainMenu(games []gamescanner.GameEntry, r renderer.Renderer, input renderer.InputManager, width, height int) *MainMenu {
	return &MainMenu{
		games:          games,
		selectedGame:   0,
		selectedLevel:  0,
		renderer:       r,
		input:          input,
		screenWidth:    width,
		screenHeight:   height,
		lastMouseClick: false,
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
				m.selectedLevel = 0
				break
			}

			// Check level clicks if this game is selected
			if i == m.selectedGame {
				levelY := gameY + 30

				// Check room library clicks first
				for j := range game.RoomLibraries {
					libRect := rect{x: 70, y: levelY + j*entryHeight, w: 280, h: 25}
					if pointInRect(mouseX, mouseY, libRect) {
						m.selectedLevel = j
						// If clicking on a library, don't start yet
						break
					}
				}

				// Update levelY to account for room libraries
				levelY += len(game.RoomLibraries) * entryHeight

				// Check level clicks
				for j := range game.Levels {
					levelRect := rect{x: 70, y: levelY + j*entryHeight, w: 280, h: 25}
					if pointInRect(mouseX, mouseY, levelRect) {
						m.selectedLevel = len(game.RoomLibraries) + j
						// If clicking on a level, don't start yet
						break
					}
				}

				// Check start button click
				startBtnY := levelY + len(game.Levels)*entryHeight + 10
				startBtnRect := rect{x: 70, y: startBtnY, w: 200, h: 30}
				if pointInRect(mouseX, mouseY, startBtnRect) {
					// Determine if selected item is a room library or level
					if m.selectedLevel < len(game.RoomLibraries) {
						return true, Selection{
							GameDir:       game.Dir,
							LevelFile:     game.RoomLibraries[m.selectedLevel],
							IsRoomLibrary: true,
						}
					} else {
						levelIndex := m.selectedLevel - len(game.RoomLibraries)
						return true, Selection{
							GameDir:       game.Dir,
							LevelFile:     game.Levels[levelIndex],
							IsRoomLibrary: false,
						}
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
			// Determine if selected item is a room library or level
			if m.selectedLevel < len(game.RoomLibraries) {
				return true, Selection{
					GameDir:       game.Dir,
					LevelFile:     game.RoomLibraries[m.selectedLevel],
					IsRoomLibrary: true,
				}
			} else {
				levelIndex := m.selectedLevel - len(game.RoomLibraries)
				if levelIndex < len(game.Levels) {
					return true, Selection{
						GameDir:       game.Dir,
						LevelFile:     game.Levels[levelIndex],
						IsRoomLibrary: false,
					}
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

		totalItems := len(game.Levels) + len(game.RoomLibraries)
		gameName := fmt.Sprintf("%s (%d items)", game.Name, totalItems)
		m.renderer.DrawText(screen, gameName, 50, currentY, gameColor, 1.5)
		currentY += 30

		// Draw levels and room libraries if selected
		if isSelected {
			itemIndex := 0

			// Draw room libraries first
			for j, library := range game.RoomLibraries {
				itemSelected := itemIndex == m.selectedLevel
				itemColor := color.RGBA{200, 150, 255, 255} // Purple for procedural
				if itemSelected {
					itemColor = color.RGBA{255, 255, 100, 255}
					// Draw selection indicator
					m.renderer.DrawText(screen, ">", 50, currentY, itemColor, 1.2)
				}

				itemText := fmt.Sprintf("  [PROCEDURAL] %s", library)
				m.renderer.DrawText(screen, itemText, 70, currentY, itemColor, 1.2)
				currentY += entryHeight
				itemIndex++
			}

			// Draw levels
			for j, level := range game.Levels {
				itemSelected := itemIndex == m.selectedLevel
				itemColor := color.RGBA{180, 180, 180, 255}
				if itemSelected {
					itemColor = color.RGBA{255, 255, 100, 255}
					// Draw selection indicator
					m.renderer.DrawText(screen, ">", 50, currentY, itemColor, 1.2)
				}

				levelText := fmt.Sprintf("  %s", level)
				m.renderer.DrawText(screen, levelText, 70, currentY, itemColor, 1.2)
				currentY += entryHeight
				itemIndex++
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
