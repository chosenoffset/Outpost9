// Package narrative provides the narrative panel UI component.
// This displays scene descriptions and action choices in a text-adventure style.
package narrative

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"chosenoffset.com/outpost9/internal/action"
)

// Panel is the narrative/action selection UI
type Panel struct {
	// Dimensions
	X, Y          int
	Width, Height int

	// Content
	sceneText        []string        // Lines of scene description
	availableActions []*ActionChoice // Actions player can take
	selectedIndex    int             // Currently highlighted action
	actionLog        []LogEntry      // Recent action results

	// Player state for display
	currentAP int
	maxAP     int

	// State
	inputMode     InputMode      // What kind of input we're waiting for
	pendingAction *action.Action // Action waiting for target selection

	// Callbacks
	OnActionSelected func(action *action.Action, direction Direction)
	OnTargetSelected func(action *action.Action, targetX, targetY int)

	// Visual settings
	bgColor       color.RGBA
	textColor     color.RGBA
	selectedColor color.RGBA
	dimColor      color.RGBA
	lineHeight    int
	padding       int
}

// InputMode defines what input the panel is waiting for
type InputMode int

const (
	ModeSelectAction InputMode = iota // Selecting an action from the list
	ModeSelectDirection               // Selecting a direction for movement/attack
	ModeSelectTarget                  // Selecting a specific target
	ModeViewLog                       // Scrolling through action log
)

// Direction for directional actions
type Direction int

const (
	DirNone Direction = iota
	DirNorth
	DirSouth
	DirEast
	DirWest
	DirNorthEast
	DirNorthWest
	DirSouthEast
	DirSouthWest
)

// ActionChoice represents an action the player can select
type ActionChoice struct {
	Action    *action.Action
	Enabled   bool   // Can this action be used right now?
	Reason    string // Why it's disabled (if applicable)
	APDisplay string // e.g., "2 AP"
	Hotkey    string // Keyboard shortcut
}

// LogEntry is a single entry in the action log
type LogEntry struct {
	Text      string
	Lines     []string   // Wrapped lines for display
	Color     color.RGBA
	Timestamp int // Turn number
}

// NewPanel creates a new narrative panel
func NewPanel(x, y, width, height int) *Panel {
	return &Panel{
		X:             x,
		Y:             y,
		Width:         width,
		Height:        height,
		sceneText:     []string{},
		availableActions: []*ActionChoice{},
		actionLog:     []LogEntry{},
		inputMode:     ModeSelectAction,
		bgColor:       color.RGBA{20, 20, 30, 230},
		textColor:     color.RGBA{200, 200, 200, 255},
		selectedColor: color.RGBA{255, 255, 150, 255},
		dimColor:      color.RGBA{120, 120, 120, 255},
		lineHeight:    14,
		padding:       10,
	}
}

// SetSceneText updates the scene description
func (p *Panel) SetSceneText(lines []string) {
	p.sceneText = lines
}

// SetSceneDescription sets scene text from a single string (auto-wraps)
func (p *Panel) SetSceneDescription(text string) {
	p.sceneText = p.wrapText(text, p.Width-p.padding*2)
}

// SetAvailableActions updates the action choices
func (p *Panel) SetAvailableActions(choices []*ActionChoice) {
	p.availableActions = choices
	p.selectedIndex = 0
	p.inputMode = ModeSelectAction

	// Find first enabled action
	for i, choice := range choices {
		if choice.Enabled {
			p.selectedIndex = i
			break
		}
	}
}

// SetAP updates the displayed action points
func (p *Panel) SetAP(current, max int) {
	p.currentAP = current
	p.maxAP = max
}

// AddLogEntry adds a new entry to the action log
func (p *Panel) AddLogEntry(text string, clr color.RGBA, turn int) {
	// Wrap the text to fit in the panel
	wrappedLines := p.wrapText(text, p.Width-p.padding*2)

	p.actionLog = append(p.actionLog, LogEntry{
		Text:      text,
		Lines:     wrappedLines,
		Color:     clr,
		Timestamp: turn,
	})

	// Keep log size reasonable
	maxEntries := 50
	if len(p.actionLog) > maxEntries {
		p.actionLog = p.actionLog[len(p.actionLog)-maxEntries:]
	}
}

// AddMessage adds a simple message to the log
func (p *Panel) AddMessage(text string, turn int) {
	p.AddLogEntry(text, p.textColor, turn)
}

// AddCombatMessage adds a combat-related message
func (p *Panel) AddCombatMessage(text string, turn int) {
	p.AddLogEntry(text, color.RGBA{255, 200, 100, 255}, turn)
}

// AddSystemMessage adds a system message
func (p *Panel) AddSystemMessage(text string, turn int) {
	p.AddLogEntry(text, color.RGBA{150, 150, 255, 255}, turn)
}

// GetInputMode returns the current input mode
func (p *Panel) GetInputMode() InputMode {
	return p.inputMode
}

// Resize updates the panel dimensions when the window is resized
func (p *Panel) Resize(screenWidth, screenHeight, panelWidth int) {
	p.Width = panelWidth
	p.Height = screenHeight
	p.X = screenWidth - panelWidth
	p.Y = 0

	// Re-wrap scene text for new width
	if len(p.sceneText) > 0 {
		// We need to store the original text to re-wrap it
		// For now, we'll rewrap based on current text joined
		originalText := strings.Join(p.sceneText, " ")
		p.sceneText = p.wrapText(originalText, p.Width-p.padding*2)
	}

	// Re-wrap action log entries for new width
	for i := range p.actionLog {
		p.actionLog[i].Lines = p.wrapText(p.actionLog[i].Text, p.Width-p.padding*2)
	}
}

// SetDirectionMode switches to direction selection for an action
func (p *Panel) SetDirectionMode(act *action.Action) {
	p.pendingAction = act
	p.inputMode = ModeSelectDirection
}

// CancelSelection returns to action selection mode
func (p *Panel) CancelSelection() {
	p.pendingAction = nil
	p.inputMode = ModeSelectAction
}

// Update handles input and returns true if an action was triggered
func (p *Panel) Update() bool {
	switch p.inputMode {
	case ModeSelectAction:
		return p.updateActionSelection()
	case ModeSelectDirection:
		return p.updateDirectionSelection()
	case ModeSelectTarget:
		return p.updateTargetSelection()
	}
	return false
}

func (p *Panel) updateActionSelection() bool {
	// Navigate with Up/Down arrow keys (WASD is for movement)
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		p.moveSelection(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		p.moveSelection(1)
	}

	// Select with Enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if p.selectedIndex >= 0 && p.selectedIndex < len(p.availableActions) {
			choice := p.availableActions[p.selectedIndex]
			if choice.Enabled {
				return p.activateAction(choice.Action)
			}
		}
	}

	// Number keys for quick selection (1-9)
	for i := 0; i < 9 && i < len(p.availableActions); i++ {
		key := ebiten.Key(int(ebiten.Key1) + i)
		if inpututil.IsKeyJustPressed(key) {
			choice := p.availableActions[i]
			if choice.Enabled {
				return p.activateAction(choice.Action)
			}
		}
	}

	return false
}

func (p *Panel) updateDirectionSelection() bool {
	// Cancel with Escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.CancelSelection()
		return false
	}

	// Direction keys
	var dir Direction
	if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		dir = DirNorth
	} else if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		dir = DirSouth
	} else if inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		dir = DirWest
	} else if inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		dir = DirEast
	}

	if dir != DirNone && p.pendingAction != nil {
		if p.OnActionSelected != nil {
			p.OnActionSelected(p.pendingAction, dir)
		}
		p.inputMode = ModeSelectAction
		p.pendingAction = nil
		return true
	}

	return false
}

func (p *Panel) updateTargetSelection() bool {
	// Cancel with Escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.CancelSelection()
		return false
	}
	// Target selection is handled by the game (clicking on map, etc.)
	return false
}

func (p *Panel) moveSelection(delta int) {
	if len(p.availableActions) == 0 {
		return
	}

	// Find next enabled action
	newIndex := p.selectedIndex
	for i := 0; i < len(p.availableActions); i++ {
		newIndex += delta
		if newIndex < 0 {
			newIndex = len(p.availableActions) - 1
		} else if newIndex >= len(p.availableActions) {
			newIndex = 0
		}

		if p.availableActions[newIndex].Enabled {
			p.selectedIndex = newIndex
			return
		}
	}
}

func (p *Panel) activateAction(act *action.Action) bool {
	// Check if action needs direction selection
	if act.Targeting.Type == action.TargetDirection {
		p.SetDirectionMode(act)
		return false // Not complete yet
	}

	// For self-targeting or no-target actions, execute immediately
	if act.Targeting.Type == action.TargetNone || act.Targeting.Type == action.TargetSelf {
		if p.OnActionSelected != nil {
			p.OnActionSelected(act, DirNone)
		}
		return true
	}

	// For entity/tile targeting, switch to target mode
	// (This would be handled differently - clicking on map, etc.)
	if p.OnActionSelected != nil {
		p.OnActionSelected(act, DirNone)
	}
	return true
}

// Draw renders the panel
func (p *Panel) Draw(screen *ebiten.Image) {
	// Draw background
	bg := ebiten.NewImage(p.Width, p.Height)
	bg.Fill(p.bgColor)

	// Draw border
	borderColor := color.RGBA{60, 60, 80, 255}
	for i := 0; i < p.Width; i++ {
		bg.Set(i, 0, borderColor)
		bg.Set(i, p.Height-1, borderColor)
	}
	for i := 0; i < p.Height; i++ {
		bg.Set(0, i, borderColor)
		bg.Set(p.Width-1, i, borderColor)
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(p.X), float64(p.Y))
	screen.DrawImage(bg, opts)

	// Current Y position for drawing
	y := p.Y + p.padding

	// Draw AP display at the top
	y = p.drawAPDisplay(screen, y)
	y += p.lineHeight / 2

	// Draw divider after AP
	p.drawDivider(screen, y)
	y += 8

	// Draw scene description
	y = p.drawSceneText(screen, y)
	y += p.lineHeight // Extra spacing

	// Draw divider
	p.drawDivider(screen, y)
	y += 8

	// Draw action log (recent entries)
	y = p.drawActionLog(screen, y, 3) // Show last 3 entries
	y += p.lineHeight / 2

	// Draw another divider
	p.drawDivider(screen, y)
	y += 8

	// Draw available actions
	y = p.drawActions(screen, y)

	// Draw mode-specific UI
	if p.inputMode == ModeSelectDirection {
		p.drawDirectionPrompt(screen)
	}
}

func (p *Panel) drawAPDisplay(screen *ebiten.Image, startY int) int {
	y := startY

	// Draw AP header
	apLabel := "Action Points: "

	// Create visual AP bar using filled/empty circles
	var apDisplay string
	for i := 0; i < p.maxAP; i++ {
		if i < p.currentAP {
			apDisplay += "●" // Filled circle for available AP
		} else {
			apDisplay += "○" // Empty circle for spent AP
		}
	}

	// Also show numeric value
	apText := fmt.Sprintf("%s%s  (%d/%d)", apLabel, apDisplay, p.currentAP, p.maxAP)
	ebitenutil.DebugPrintAt(screen, apText, p.X+p.padding, y)
	y += p.lineHeight

	// Show controls hint
	controlsHint := "WASD:Move  ↑↓:Select  Enter:Confirm  Space:End Turn"
	ebitenutil.DebugPrintAt(screen, controlsHint, p.X+p.padding, y)
	y += p.lineHeight

	return y
}

func (p *Panel) drawSceneText(screen *ebiten.Image, startY int) int {
	y := startY
	for _, line := range p.sceneText {
		ebitenutil.DebugPrintAt(screen, line, p.X+p.padding, y)
		y += p.lineHeight
	}
	return y
}

func (p *Panel) drawActionLog(screen *ebiten.Image, startY int, maxEntries int) int {
	y := startY

	// Show most recent entries
	start := len(p.actionLog) - maxEntries
	if start < 0 {
		start = 0
	}

	for i := start; i < len(p.actionLog); i++ {
		entry := p.actionLog[i]
		// Draw each wrapped line of the entry
		if len(entry.Lines) > 0 {
			for _, line := range entry.Lines {
				ebitenutil.DebugPrintAt(screen, line, p.X+p.padding, y)
				y += p.lineHeight
			}
		} else {
			// Fallback to original text if no wrapped lines
			ebitenutil.DebugPrintAt(screen, entry.Text, p.X+p.padding, y)
			y += p.lineHeight
		}
	}

	return y
}

func (p *Panel) drawActions(screen *ebiten.Image, startY int) int {
	y := startY

	// Header
	headerText := "Actions (↑↓ to select, Enter to confirm):"
	if p.inputMode == ModeSelectDirection {
		headerText = "Select Direction (WASD, ESC to cancel):"
	}
	ebitenutil.DebugPrintAt(screen, headerText, p.X+p.padding, y)
	y += p.lineHeight + 4

	// Action list
	for i, choice := range p.availableActions {
		prefix := "  "
		if i == p.selectedIndex && p.inputMode == ModeSelectAction {
			prefix = "> "
		}

		// Format: "> [1] Move (1 AP)"
		numKey := fmt.Sprintf("[%d] ", i+1)
		if i >= 9 {
			numKey = "    "
		}

		text := fmt.Sprintf("%s%s%s (%s)", prefix, numKey, choice.Action.Name, choice.APDisplay)

		if !choice.Enabled {
			text += " - " + choice.Reason
		}

		ebitenutil.DebugPrintAt(screen, text, p.X+p.padding, y)
		y += p.lineHeight
	}

	return y
}

func (p *Panel) drawDivider(screen *ebiten.Image, y int) {
	divider := ebiten.NewImage(p.Width-p.padding*2, 1)
	divider.Fill(color.RGBA{60, 60, 80, 200})
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(p.X+p.padding), float64(y))
	screen.DrawImage(divider, opts)
}

func (p *Panel) drawDirectionPrompt(screen *ebiten.Image) {
	// Draw a prompt at the bottom of the panel
	promptY := p.Y + p.Height - p.lineHeight*2 - p.padding
	prompt := "Press direction key (WASD) or ESC to cancel"
	ebitenutil.DebugPrintAt(screen, prompt, p.X+p.padding, promptY)
}

// wrapText wraps text to fit within a given width (approximate)
func (p *Panel) wrapText(text string, maxWidth int) []string {
	// Rough approximation: 6 pixels per character
	charsPerLine := maxWidth / 6
	if charsPerLine < 20 {
		charsPerLine = 20
	}

	words := strings.Fields(text)
	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 > charsPerLine {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// Helper to convert hotkey string to ebiten key
func hotkeyToKey(hotkey string) ebiten.Key {
	if len(hotkey) != 1 {
		return ebiten.Key(-1)
	}

	ch := hotkey[0]
	switch {
	case ch >= 'a' && ch <= 'z':
		return ebiten.Key(int(ebiten.KeyA) + int(ch-'a'))
	case ch >= 'A' && ch <= 'Z':
		return ebiten.Key(int(ebiten.KeyA) + int(ch-'A'))
	case ch >= '0' && ch <= '9':
		return ebiten.Key(int(ebiten.Key0) + int(ch-'0'))
	case ch == '.':
		return ebiten.KeyPeriod
	case ch == ',':
		return ebiten.KeyComma
	case ch == ' ':
		return ebiten.KeySpace
	}

	return ebiten.Key(-1)
}
