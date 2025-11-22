// Package screen provides a data-driven UI screen system for menus,
// character creation, dialogs, and other UI elements.
package screen

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ElementType defines the type of UI element
type ElementType string

const (
	ElementLabel     ElementType = "label"     // Static text
	ElementButton    ElementType = "button"    // Clickable button
	ElementInput     ElementType = "input"     // Text input field
	ElementSelect    ElementType = "select"    // Dropdown/selection
	ElementStatRoll  ElementType = "stat_roll" // Stat with roll button
	ElementStatList  ElementType = "stat_list" // List of stats
	ElementDivider   ElementType = "divider"   // Visual divider
	ElementSpacer    ElementType = "spacer"    // Empty space
	ElementContainer ElementType = "container" // Container for other elements
)

// Alignment defines text/element alignment
type Alignment string

const (
	AlignLeft   Alignment = "left"
	AlignCenter Alignment = "center"
	AlignRight  Alignment = "right"
)

// Element defines a single UI element
type Element struct {
	ID         string         `json:"id"`                   // Unique identifier
	Type       ElementType    `json:"type"`                 // Element type
	Text       string         `json:"text,omitempty"`       // Display text
	X          int            `json:"x,omitempty"`          // X position (relative or absolute)
	Y          int            `json:"y,omitempty"`          // Y position (relative or absolute)
	Width      int            `json:"width,omitempty"`      // Width (0 = auto)
	Height     int            `json:"height,omitempty"`     // Height (0 = auto)
	Align      Alignment      `json:"align,omitempty"`      // Text alignment
	Color      string         `json:"color,omitempty"`      // Text/element color
	BGColor    string         `json:"bg_color,omitempty"`   // Background color
	FontScale  float64        `json:"font_scale,omitempty"` // Font size multiplier
	Visible    bool           `json:"visible"`              // Whether element is visible
	Enabled    bool           `json:"enabled"`              // Whether element is interactive
	Action     string         `json:"action,omitempty"`     // Action to trigger on activation
	Binding    string         `json:"binding,omitempty"`    // Data binding (e.g., "character.name")
	Options    []SelectOption `json:"options,omitempty"`    // Options for select elements
	Children   []Element      `json:"children,omitempty"`   // Child elements (for containers)
	Properties map[string]any `json:"properties,omitempty"` // Custom properties

	// Runtime state (not serialized)
	selected    bool
	inputValue  string
	selectIndex int
	hovered     bool
}

// SelectOption defines an option in a select element
type SelectOption struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
}

// Screen defines a complete UI screen
type Screen struct {
	ID         string    `json:"id"`                    // Unique identifier
	Name       string    `json:"name"`                  // Display name
	Title      string    `json:"title,omitempty"`       // Screen title
	Elements   []Element `json:"elements"`              // UI elements
	OnEnter    string    `json:"on_enter,omitempty"`    // Action on screen enter
	OnExit     string    `json:"on_exit,omitempty"`     // Action on screen exit
	Background string    `json:"background,omitempty"`  // Background color or image
	NextScreen string    `json:"next_screen,omitempty"` // Default next screen
	PrevScreen string    `json:"prev_screen,omitempty"` // Previous screen for back button

	// Runtime state
	focusIndex  int
	initialized bool
}

// ScreenFlow defines a sequence of screens
type ScreenFlow struct {
	ID          string   `json:"id"`                    // Flow identifier
	Name        string   `json:"name"`                  // Display name
	Description string   `json:"description,omitempty"` // Flow description
	Screens     []Screen `json:"screens"`               // Screens in order
	StartScreen string   `json:"start_screen"`          // First screen ID
	OnComplete  string   `json:"on_complete,omitempty"` // Action when flow completes

	// Lookup map
	screensByID map[string]*Screen
}

// LoadScreenFlow loads a screen flow from a JSON file
func LoadScreenFlow(path string) (*ScreenFlow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read screen flow: %w", err)
	}

	var flow ScreenFlow
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("failed to parse screen flow: %w", err)
	}

	flow.buildLookupMaps()
	return &flow, nil
}

// LoadScreenFlowFromFS loads a screen flow using an ebiten file system
func LoadScreenFlowFromFS(fs ebiten.FileSystem, path string) (*ScreenFlow, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open screen flow: %w", err)
	}
	defer file.Close()

	var flow ScreenFlow
	if err := json.NewDecoder(file).Decode(&flow); err != nil {
		return nil, fmt.Errorf("failed to parse screen flow: %w", err)
	}

	flow.buildLookupMaps()
	return &flow, nil
}

func (f *ScreenFlow) buildLookupMaps() {
	f.screensByID = make(map[string]*Screen)
	for i := range f.Screens {
		f.screensByID[f.Screens[i].ID] = &f.Screens[i]
		// Initialize elements with defaults
		for j := range f.Screens[i].Elements {
			initElement(&f.Screens[i].Elements[j])
		}
	}
}

func initElement(e *Element) {
	// Set defaults for unset fields
	if e.FontScale == 0 {
		e.FontScale = 1.0
	}
	if e.Type == "" {
		e.Type = ElementLabel
	}
	// Visible and Enabled default to true if not explicitly set
	// (JSON unmarshal sets them to false, so we handle this differently)

	// Initialize children
	for i := range e.Children {
		initElement(&e.Children[i])
	}
}

// GetScreen returns a screen by ID
func (f *ScreenFlow) GetScreen(id string) *Screen {
	return f.screensByID[id]
}

// GetStartScreen returns the starting screen
func (f *ScreenFlow) GetStartScreen() *Screen {
	return f.screensByID[f.StartScreen]
}

// --- Screen Manager ---

// ActionHandler is a function that handles UI actions
type ActionHandler func(action string, element *Element, manager *Manager) error

// DataProvider provides data for bindings
type DataProvider interface {
	GetValue(binding string) any
	SetValue(binding string, value any) error
}

// Manager manages screen rendering and interaction
type Manager struct {
	currentFlow    *ScreenFlow
	currentScreen  *Screen
	actionHandlers map[string]ActionHandler
	dataProvider   DataProvider
	screenWidth    int
	screenHeight   int

	// Input state
	lastMouseX, lastMouseY int
	mousePressed           bool
}

// NewManager creates a new screen manager
func NewManager(width, height int) *Manager {
	return &Manager{
		actionHandlers: make(map[string]ActionHandler),
		screenWidth:    width,
		screenHeight:   height,
	}
}

// SetFlow sets the current screen flow
func (m *Manager) SetFlow(flow *ScreenFlow) {
	m.currentFlow = flow
	if flow != nil {
		m.currentScreen = flow.GetStartScreen()
	}
}

// SetScreen sets the current screen by ID
func (m *Manager) SetScreen(screenID string) bool {
	if m.currentFlow == nil {
		return false
	}
	screen := m.currentFlow.GetScreen(screenID)
	if screen == nil {
		return false
	}
	m.currentScreen = screen
	return true
}

// GetCurrentScreen returns the current screen
func (m *Manager) GetCurrentScreen() *Screen {
	return m.currentScreen
}

// SetDataProvider sets the data provider for bindings
func (m *Manager) SetDataProvider(provider DataProvider) {
	m.dataProvider = provider
}

// RegisterAction registers a handler for an action
func (m *Manager) RegisterAction(action string, handler ActionHandler) {
	m.actionHandlers[action] = handler
}

// Update handles input and updates screen state
func (m *Manager) Update() error {
	if m.currentScreen == nil {
		return nil
	}

	// Get mouse position
	mx, my := ebiten.CursorPosition()
	m.lastMouseX = mx
	m.lastMouseY = my

	// Handle keyboard navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			m.focusPrev()
		} else {
			m.focusNext()
		}
	}

	// Handle enter key for focused element
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if elem := m.getFocusedElement(); elem != nil {
			m.activateElement(elem)
		}
	}

	// Handle mouse clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		m.handleClick(mx, my)
	}

	// Handle arrow keys for select elements
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		if elem := m.getFocusedElement(); elem != nil && elem.Type == ElementSelect {
			delta := 1
			if ebiten.IsKeyPressed(ebiten.KeyUp) {
				delta = -1
			}
			m.cycleSelect(elem, delta)
		}
	}

	// Handle text input for input elements
	if elem := m.getFocusedElement(); elem != nil && elem.Type == ElementInput {
		m.handleTextInput(elem)
	}

	return nil
}

func (m *Manager) focusNext() {
	elements := m.getInteractiveElements()
	if len(elements) == 0 {
		return
	}

	// Clear current focus
	for _, e := range elements {
		e.selected = false
	}

	m.currentScreen.focusIndex++
	if m.currentScreen.focusIndex >= len(elements) {
		m.currentScreen.focusIndex = 0
	}

	elements[m.currentScreen.focusIndex].selected = true
}

func (m *Manager) focusPrev() {
	elements := m.getInteractiveElements()
	if len(elements) == 0 {
		return
	}

	// Clear current focus
	for _, e := range elements {
		e.selected = false
	}

	m.currentScreen.focusIndex--
	if m.currentScreen.focusIndex < 0 {
		m.currentScreen.focusIndex = len(elements) - 1
	}

	elements[m.currentScreen.focusIndex].selected = true
}

func (m *Manager) getInteractiveElements() []*Element {
	var elements []*Element
	for i := range m.currentScreen.Elements {
		m.collectInteractive(&m.currentScreen.Elements[i], &elements)
	}
	return elements
}

func (m *Manager) collectInteractive(e *Element, list *[]*Element) {
	if !e.Visible {
		return
	}
	if e.Type == ElementButton || e.Type == ElementInput || e.Type == ElementSelect || e.Type == ElementStatRoll {
		if e.Enabled {
			*list = append(*list, e)
		}
	}
	for i := range e.Children {
		m.collectInteractive(&e.Children[i], list)
	}
}

func (m *Manager) getFocusedElement() *Element {
	elements := m.getInteractiveElements()
	if len(elements) == 0 || m.currentScreen.focusIndex >= len(elements) {
		return nil
	}
	return elements[m.currentScreen.focusIndex]
}

func (m *Manager) handleClick(x, y int) {
	for i := range m.currentScreen.Elements {
		if elem := m.findElementAt(&m.currentScreen.Elements[i], x, y); elem != nil {
			m.activateElement(elem)
			return
		}
	}
}

func (m *Manager) findElementAt(e *Element, x, y int) *Element {
	if !e.Visible || !e.Enabled {
		return nil
	}

	// Check if click is within element bounds
	if x >= e.X && x < e.X+e.Width && y >= e.Y && y < e.Y+e.Height {
		// Check children first
		for i := range e.Children {
			if child := m.findElementAt(&e.Children[i], x, y); child != nil {
				return child
			}
		}
		// Return this element if it's interactive
		if e.Type == ElementButton || e.Type == ElementInput || e.Type == ElementSelect || e.Type == ElementStatRoll {
			return e
		}
	}

	return nil
}

func (m *Manager) activateElement(e *Element) {
	if e == nil || e.Action == "" {
		return
	}

	if handler, ok := m.actionHandlers[e.Action]; ok {
		handler(e.Action, e, m)
	}
}

func (m *Manager) cycleSelect(e *Element, delta int) {
	if len(e.Options) == 0 {
		return
	}

	e.selectIndex += delta
	if e.selectIndex < 0 {
		e.selectIndex = len(e.Options) - 1
	} else if e.selectIndex >= len(e.Options) {
		e.selectIndex = 0
	}

	// Update binding if set
	if e.Binding != "" && m.dataProvider != nil {
		m.dataProvider.SetValue(e.Binding, e.Options[e.selectIndex].Value)
	}
}

func (m *Manager) handleTextInput(e *Element) {
	// Handle backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(e.inputValue) > 0 {
		e.inputValue = e.inputValue[:len(e.inputValue)-1]
		if e.Binding != "" && m.dataProvider != nil {
			m.dataProvider.SetValue(e.Binding, e.inputValue)
		}
	}

	// Handle character input
	chars := ebiten.AppendInputChars(nil)
	if len(chars) > 0 {
		e.inputValue += string(chars)
		if e.Binding != "" && m.dataProvider != nil {
			m.dataProvider.SetValue(e.Binding, e.inputValue)
		}
	}
}

// Draw renders the current screen
func (m *Manager) Draw(dst *ebiten.Image) {
	if m.currentScreen == nil {
		return
	}

	// Draw background
	if m.currentScreen.Background != "" {
		bgColor := parseColor(m.currentScreen.Background)
		dst.Fill(bgColor)
	}

	// Draw title
	if m.currentScreen.Title != "" {
		ebitenutil.DebugPrintAt(dst, m.currentScreen.Title, m.screenWidth/2-len(m.currentScreen.Title)*3, 20)
	}

	// Draw elements
	for i := range m.currentScreen.Elements {
		m.drawElement(dst, &m.currentScreen.Elements[i], 0, 0)
	}
}

func (m *Manager) drawElement(dst *ebiten.Image, e *Element, offsetX, offsetY int) {
	if !e.Visible {
		return
	}

	x := e.X + offsetX
	y := e.Y + offsetY

	switch e.Type {
	case ElementLabel:
		m.drawLabel(dst, e, x, y)
	case ElementButton:
		m.drawButton(dst, e, x, y)
	case ElementInput:
		m.drawInput(dst, e, x, y)
	case ElementSelect:
		m.drawSelect(dst, e, x, y)
	case ElementStatRoll:
		m.drawStatRoll(dst, e, x, y)
	case ElementStatList:
		m.drawStatList(dst, e, x, y)
	case ElementDivider:
		m.drawDivider(dst, e, x, y)
	case ElementSpacer:
		// Just takes up space
	case ElementContainer:
		// Draw children with offset
		for i := range e.Children {
			m.drawElement(dst, &e.Children[i], x, y)
		}
	}
}

func (m *Manager) drawLabel(dst *ebiten.Image, e *Element, x, y int) {
	text := e.Text

	// Handle binding
	if e.Binding != "" && m.dataProvider != nil {
		if val := m.dataProvider.GetValue(e.Binding); val != nil {
			text = fmt.Sprintf("%v", val)
		}
	}

	ebitenutil.DebugPrintAt(dst, text, x, y)
}

func (m *Manager) drawButton(dst *ebiten.Image, e *Element, x, y int) {
	// Draw button background
	width := e.Width
	if width == 0 {
		width = len(e.Text)*6 + 20
	}
	height := e.Height
	if height == 0 {
		height = 20
	}

	bgColor := color.RGBA{60, 60, 80, 255}
	if e.selected {
		bgColor = color.RGBA{80, 80, 120, 255}
	}

	drawRect(dst, x, y, width, height, bgColor)

	// Draw border if selected
	if e.selected {
		drawRectOutline(dst, x, y, width, height, color.RGBA{200, 200, 255, 255})
	}

	// Draw text centered
	textX := x + (width-len(e.Text)*6)/2
	textY := y + (height-13)/2
	ebitenutil.DebugPrintAt(dst, e.Text, textX, textY)
}

func (m *Manager) drawInput(dst *ebiten.Image, e *Element, x, y int) {
	width := e.Width
	if width == 0 {
		width = 200
	}
	height := e.Height
	if height == 0 {
		height = 20
	}

	// Draw background
	bgColor := color.RGBA{30, 30, 40, 255}
	drawRect(dst, x, y, width, height, bgColor)

	// Draw border
	borderColor := color.RGBA{100, 100, 120, 255}
	if e.selected {
		borderColor = color.RGBA{200, 200, 255, 255}
	}
	drawRectOutline(dst, x, y, width, height, borderColor)

	// Draw text
	text := e.inputValue
	if text == "" && e.Text != "" {
		text = e.Text // Placeholder
	}
	ebitenutil.DebugPrintAt(dst, text, x+4, y+4)

	// Draw cursor if selected
	if e.selected {
		cursorX := x + 4 + len(e.inputValue)*6
		ebitenutil.DebugPrintAt(dst, "_", cursorX, y+4)
	}
}

func (m *Manager) drawSelect(dst *ebiten.Image, e *Element, x, y int) {
	width := e.Width
	if width == 0 {
		width = 200
	}
	height := e.Height
	if height == 0 {
		height = 20
	}

	// Draw background
	bgColor := color.RGBA{40, 40, 50, 255}
	drawRect(dst, x, y, width, height, bgColor)

	// Draw border
	borderColor := color.RGBA{100, 100, 120, 255}
	if e.selected {
		borderColor = color.RGBA{200, 200, 255, 255}
	}
	drawRectOutline(dst, x, y, width, height, borderColor)

	// Draw current option
	var text string
	if len(e.Options) > 0 && e.selectIndex < len(e.Options) {
		text = e.Options[e.selectIndex].Label
	}
	ebitenutil.DebugPrintAt(dst, text, x+4, y+4)

	// Draw arrows
	ebitenutil.DebugPrintAt(dst, "<", x+width-20, y+4)
	ebitenutil.DebugPrintAt(dst, ">", x+width-10, y+4)
}

func (m *Manager) drawStatRoll(dst *ebiten.Image, e *Element, x, y int) {
	// Draw stat name
	ebitenutil.DebugPrintAt(dst, e.Text, x, y)

	// Draw value
	var value string
	if e.Binding != "" && m.dataProvider != nil {
		if val := m.dataProvider.GetValue(e.Binding); val != nil {
			value = fmt.Sprintf("%v", val)
		}
	}
	ebitenutil.DebugPrintAt(dst, value, x+120, y)

	// Draw roll button
	btnX := x + 160
	btnWidth := 50
	btnHeight := 16
	bgColor := color.RGBA{60, 60, 80, 255}
	if e.selected {
		bgColor = color.RGBA{80, 80, 120, 255}
	}
	drawRect(dst, btnX, y-2, btnWidth, btnHeight, bgColor)
	ebitenutil.DebugPrintAt(dst, "Roll", btnX+10, y)
}

func (m *Manager) drawStatList(dst *ebiten.Image, e *Element, x, y int) {
	// This is handled specially by the character creation system
	// Just draw a placeholder
	ebitenutil.DebugPrintAt(dst, "[Stat List]", x, y)
}

func (m *Manager) drawDivider(dst *ebiten.Image, e *Element, x, y int) {
	width := e.Width
	if width == 0 {
		width = m.screenWidth - 40
	}
	drawRect(dst, x, y+8, width, 1, color.RGBA{100, 100, 120, 255})
}

// Helper functions

func drawRect(dst *ebiten.Image, x, y, w, h int, clr color.Color) {
	rect := ebiten.NewImage(w, h)
	rect.Fill(clr)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(rect, op)
}

func drawRectOutline(dst *ebiten.Image, x, y, w, h int, clr color.Color) {
	// Top
	drawRect(dst, x, y, w, 1, clr)
	// Bottom
	drawRect(dst, x, y+h-1, w, 1, clr)
	// Left
	drawRect(dst, x, y, 1, h, clr)
	// Right
	drawRect(dst, x+w-1, y, 1, h, clr)
}

func parseColor(s string) color.Color {
	// Handle common color names
	switch s {
	case "black":
		return color.RGBA{0, 0, 0, 255}
	case "white":
		return color.RGBA{255, 255, 255, 255}
	case "red":
		return color.RGBA{255, 0, 0, 255}
	case "green":
		return color.RGBA{0, 255, 0, 255}
	case "blue":
		return color.RGBA{0, 0, 255, 255}
	case "gray", "grey":
		return color.RGBA{128, 128, 128, 255}
	case "dark":
		return color.RGBA{20, 20, 30, 255}
	default:
		// Try to parse as hex
		if len(s) == 7 && s[0] == '#' {
			var r, g, b uint8
			fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
			return color.RGBA{r, g, b, 255}
		}
		return color.RGBA{20, 20, 30, 255}
	}
}
