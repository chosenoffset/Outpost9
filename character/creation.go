// Package character creation system
package character

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"chosenoffset.com/outpost9/dice"
)

// CreationState tracks the current phase of character creation
type CreationState int

const (
	StateSelectMethod CreationState = iota
	StateEnterName
	StateRollStats
	StateAssignStats
	StateReview
	StateComplete
)

// CreationManager handles the character creation process
type CreationManager struct {
	template       *CharacterTemplate
	character      *Character
	roller         *dice.Roller
	state          CreationState
	selectedMethod string
	screenWidth    int
	screenHeight   int

	// UI state
	focusedField  int
	nameInput     string
	cursorVisible bool
	cursorTimer   int
	message       string
	messageTimer  int

	// Stats display
	statRolls          map[string]*dice.RollResult
	unassignedStats    []int
	assignmentMap      map[int]string // unassigned index -> stat ID
	selectedUnassigned int

	// Completion callback
	onComplete func(*Character)
}

// NewCreationManager creates a new character creation manager
func NewCreationManager(template *CharacterTemplate, width, height int) *CreationManager {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	return &CreationManager{
		template:           template,
		roller:             dice.NewRoller(rng),
		state:              StateSelectMethod,
		screenWidth:        width,
		screenHeight:       height,
		statRolls:          make(map[string]*dice.RollResult),
		assignmentMap:      make(map[int]string),
		selectedUnassigned: -1,
	}
}

// SetOnComplete sets the callback for when character creation finishes
func (cm *CreationManager) SetOnComplete(callback func(*Character)) {
	cm.onComplete = callback
}

// GetCharacter returns the character being created
func (cm *CreationManager) GetCharacter() *Character {
	return cm.character
}

// GetState returns the current creation state
func (cm *CreationManager) GetState() CreationState {
	return cm.state
}

// Update handles input and state changes
func (cm *CreationManager) Update() error {
	// Update cursor blink
	cm.cursorTimer++
	if cm.cursorTimer > 30 {
		cm.cursorTimer = 0
		cm.cursorVisible = !cm.cursorVisible
	}

	// Update message timer
	if cm.messageTimer > 0 {
		cm.messageTimer--
		if cm.messageTimer == 0 {
			cm.message = ""
		}
	}

	switch cm.state {
	case StateSelectMethod:
		return cm.updateSelectMethod()
	case StateEnterName:
		return cm.updateEnterName()
	case StateRollStats:
		return cm.updateRollStats()
	case StateAssignStats:
		return cm.updateAssignStats()
	case StateReview:
		return cm.updateReview()
	}

	return nil
}

func (cm *CreationManager) updateSelectMethod() error {
	methods := cm.template.GenerationMethods
	if len(methods) == 0 {
		// No methods defined, use default roll
		cm.selectedMethod = ""
		cm.state = StateEnterName
		return nil
	}

	// Navigate with up/down
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		cm.focusedField--
		if cm.focusedField < 0 {
			cm.focusedField = len(methods) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		cm.focusedField++
		if cm.focusedField >= len(methods) {
			cm.focusedField = 0
		}
	}

	// Select with enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		cm.selectedMethod = methods[cm.focusedField].ID
		cm.state = StateEnterName
		cm.focusedField = 0
	}

	return nil
}

func (cm *CreationManager) updateEnterName() error {
	// Handle text input
	chars := ebiten.AppendInputChars(nil)
	for _, c := range chars {
		if len(cm.nameInput) < 20 { // Max name length
			cm.nameInput += string(c)
		}
	}

	// Handle backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(cm.nameInput) > 0 {
		cm.nameInput = cm.nameInput[:len(cm.nameInput)-1]
	}

	// Proceed with enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if len(cm.nameInput) == 0 {
			cm.showMessage("Please enter a name")
			return nil
		}

		// Create character and move to next state
		cm.character = NewCharacter(cm.template)
		cm.character.Name = cm.nameInput

		// Check if we need to roll or assign stats
		method := cm.template.GetMethod(cm.selectedMethod)
		if method != nil && len(method.Array) > 0 {
			// Standard array - go to assignment
			cm.unassignedStats = make([]int, len(method.Array))
			copy(cm.unassignedStats, method.Array)
			cm.state = StateAssignStats
		} else {
			// Rolling - go to roll stats
			cm.state = StateRollStats
		}
		cm.focusedField = 0
	}

	// Go back with escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		cm.state = StateSelectMethod
		cm.focusedField = 0
	}

	return nil
}

func (cm *CreationManager) updateRollStats() error {
	stats := cm.getGenerableStats()

	// Navigate with up/down
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		cm.focusedField--
		if cm.focusedField < 0 {
			cm.focusedField = len(stats)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		cm.focusedField++
		if cm.focusedField > len(stats) {
			cm.focusedField = 0
		}
	}

	// Roll stat with enter or space
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if cm.focusedField < len(stats) {
			// Roll individual stat
			stat := stats[cm.focusedField]
			cm.rollStat(stat)
		} else {
			// "Roll All" button
			cm.rollAllStats()
		}
	}

	// Roll all with R key
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		cm.rollAllStats()
	}

	// Proceed with Tab or when all rolled
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if cm.allStatsRolled() {
			cm.applyRolls()
			cm.state = StateReview
			cm.focusedField = 0
		} else {
			cm.showMessage("Roll all stats first")
		}
	}

	// Go back with escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		cm.state = StateEnterName
		cm.focusedField = 0
	}

	return nil
}

func (cm *CreationManager) updateAssignStats() error {
	stats := cm.getGenerableStats()

	// Navigate stats with up/down
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		cm.focusedField--
		if cm.focusedField < 0 {
			cm.focusedField = len(stats) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		cm.focusedField++
		if cm.focusedField >= len(stats) {
			cm.focusedField = 0
		}
	}

	// Navigate unassigned values with left/right
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		cm.selectedUnassigned--
		if cm.selectedUnassigned < 0 {
			cm.selectedUnassigned = len(cm.unassignedStats) - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		cm.selectedUnassigned++
		if cm.selectedUnassigned >= len(cm.unassignedStats) {
			cm.selectedUnassigned = 0
		}
	}

	// Assign with enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if cm.selectedUnassigned >= 0 && cm.selectedUnassigned < len(cm.unassignedStats) {
			stat := stats[cm.focusedField]
			// Check if this stat already has a value assigned
			currentAssigned := -1
			for idx, statID := range cm.assignmentMap {
				if statID == stat.ID {
					currentAssigned = idx
					break
				}
			}
			if currentAssigned >= 0 {
				// Swap assignments
				delete(cm.assignmentMap, currentAssigned)
			}
			// Assign the selected value to this stat
			cm.assignmentMap[cm.selectedUnassigned] = stat.ID
		}
	}

	// Proceed with Tab when all assigned
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if len(cm.assignmentMap) == len(stats) {
			cm.applyAssignments()
			cm.state = StateReview
			cm.focusedField = 0
		} else {
			cm.showMessage("Assign all stats first")
		}
	}

	// Go back with escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		cm.state = StateEnterName
		cm.focusedField = 0
	}

	return nil
}

func (cm *CreationManager) updateReview() error {
	// Navigate with up/down (for potential re-roll options)
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		cm.focusedField--
		if cm.focusedField < 0 {
			cm.focusedField = 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		cm.focusedField++
		if cm.focusedField > 1 {
			cm.focusedField = 0
		}
	}

	// Confirm with enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if cm.focusedField == 0 {
			// Accept
			cm.state = StateComplete
			if cm.onComplete != nil {
				cm.onComplete(cm.character)
			}
		} else {
			// Start over
			cm.resetCreation()
		}
	}

	// Go back with escape
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		method := cm.template.GetMethod(cm.selectedMethod)
		if method != nil && len(method.Array) > 0 {
			cm.state = StateAssignStats
		} else {
			cm.state = StateRollStats
		}
		cm.focusedField = 0
	}

	return nil
}

func (cm *CreationManager) getGenerableStats() []*StatDefinition {
	var stats []*StatDefinition
	for i := range cm.template.Stats {
		stat := &cm.template.Stats[i]
		if stat.Formula == "" && !stat.Hidden {
			stats = append(stats, stat)
		}
	}
	return stats
}

func (cm *CreationManager) rollStat(stat *StatDefinition) {
	method := cm.template.GetMethod(cm.selectedMethod)

	var expression *dice.StatExpression

	// Check for method override
	if method != nil && method.Overrides != nil {
		if override, ok := method.Overrides[stat.ID]; ok {
			expression = override
		}
	}

	// Fall back to method default
	if expression == nil && method != nil && method.Default != nil {
		expression = method.Default
	}

	// Fall back to stat generation
	if expression == nil && stat.Generation != nil {
		expression = stat.Generation
	}

	// Fall back to 3d6
	if expression == nil {
		expression = &dice.StatExpression{
			Type:       dice.ExprRoll,
			Expression: "3d6",
		}
	}

	result, err := expression.Evaluate(cm.roller)
	if err != nil {
		cm.showMessage(fmt.Sprintf("Error rolling %s: %v", stat.Name, err))
		return
	}

	cm.statRolls[stat.ID] = result
}

func (cm *CreationManager) rollAllStats() {
	for _, stat := range cm.getGenerableStats() {
		cm.rollStat(stat)
	}
}

func (cm *CreationManager) allStatsRolled() bool {
	stats := cm.getGenerableStats()
	for _, stat := range stats {
		if _, ok := cm.statRolls[stat.ID]; !ok {
			return false
		}
	}
	return true
}

func (cm *CreationManager) applyRolls() {
	for statID, result := range cm.statRolls {
		cm.character.Stats[statID] = &StatValue{
			StatID:    statID,
			Value:     result.Total,
			BaseValue: result.Total,
			RollInfo:  result,
		}
	}
}

func (cm *CreationManager) applyAssignments() {
	for idx, statID := range cm.assignmentMap {
		value := cm.unassignedStats[idx]
		cm.character.Stats[statID] = &StatValue{
			StatID:    statID,
			Value:     value,
			BaseValue: value,
		}
	}
}

func (cm *CreationManager) resetCreation() {
	cm.character = nil
	cm.statRolls = make(map[string]*dice.RollResult)
	cm.assignmentMap = make(map[int]string)
	cm.selectedUnassigned = -1
	cm.nameInput = ""
	cm.state = StateSelectMethod
	cm.focusedField = 0
}

func (cm *CreationManager) showMessage(msg string) {
	cm.message = msg
	cm.messageTimer = 120 // 2 seconds at 60fps
}

// IsComplete returns true if character creation is finished
func (cm *CreationManager) IsComplete() bool {
	return cm.state == StateComplete
}

// Draw renders the character creation UI
func (cm *CreationManager) Draw(dst *ebiten.Image) {
	// Draw title
	title := "Character Creation"
	ebitenutil.DebugPrintAt(dst, title, cm.screenWidth/2-len(title)*3, 20)

	switch cm.state {
	case StateSelectMethod:
		cm.drawSelectMethod(dst)
	case StateEnterName:
		cm.drawEnterName(dst)
	case StateRollStats:
		cm.drawRollStats(dst)
	case StateAssignStats:
		cm.drawAssignStats(dst)
	case StateReview:
		cm.drawReview(dst)
	}

	// Draw message if any
	if cm.message != "" {
		msgX := cm.screenWidth/2 - len(cm.message)*3
		msgY := cm.screenHeight - 60
		ebitenutil.DebugPrintAt(dst, cm.message, msgX, msgY)
	}

	// Draw navigation help
	cm.drawNavHelp(dst)
}

func (cm *CreationManager) drawSelectMethod(dst *ebiten.Image) {
	methods := cm.template.GenerationMethods

	ebitenutil.DebugPrintAt(dst, "Select Generation Method:", 50, 60)

	y := 100
	for i, method := range methods {
		prefix := "  "
		if i == cm.focusedField {
			prefix = "> "
		}

		text := fmt.Sprintf("%s%s", prefix, method.Name)
		ebitenutil.DebugPrintAt(dst, text, 50, y)

		if method.Description != "" {
			ebitenutil.DebugPrintAt(dst, "    "+method.Description, 50, y+15)
			y += 15
		}

		y += 25
	}
}

func (cm *CreationManager) drawEnterName(dst *ebiten.Image) {
	ebitenutil.DebugPrintAt(dst, "Enter Character Name:", 50, 60)

	// Draw input box
	boxX := 50
	boxY := 90
	boxWidth := 200
	boxHeight := 25

	// Box background
	drawRect(dst, boxX, boxY, boxWidth, boxHeight, 30, 30, 40)
	// Box border
	drawRectOutline(dst, boxX, boxY, boxWidth, boxHeight, 100, 100, 120)

	// Draw name text
	displayText := cm.nameInput
	if cm.cursorVisible {
		displayText += "_"
	}
	ebitenutil.DebugPrintAt(dst, displayText, boxX+5, boxY+6)
}

func (cm *CreationManager) drawRollStats(dst *ebiten.Image) {
	stats := cm.getGenerableStats()

	ebitenutil.DebugPrintAt(dst, "Roll Your Stats:", 50, 60)
	ebitenutil.DebugPrintAt(dst, "(Press Enter/Space to roll, R for all)", 50, 78)

	y := 110
	for i, stat := range stats {
		prefix := "  "
		if i == cm.focusedField {
			prefix = "> "
		}

		// Stat name
		abbr := stat.Abbreviation
		if abbr == "" {
			abbr = stat.Name[:3]
		}
		text := fmt.Sprintf("%s%-12s (%s)", prefix, stat.Name, abbr)
		ebitenutil.DebugPrintAt(dst, text, 50, y)

		// Roll result
		if result, ok := cm.statRolls[stat.ID]; ok {
			valueText := fmt.Sprintf("%3d", result.Total)
			ebitenutil.DebugPrintAt(dst, valueText, 220, y)

			// Breakdown
			if result.Breakdown != "" {
				ebitenutil.DebugPrintAt(dst, result.Breakdown, 260, y)
			}
		} else {
			ebitenutil.DebugPrintAt(dst, " --", 220, y)
		}

		y += 22
	}

	// Roll all button
	prefix := "  "
	if cm.focusedField == len(stats) {
		prefix = "> "
	}
	ebitenutil.DebugPrintAt(dst, prefix+"[Roll All]", 50, y+10)

	// Show continue hint if all rolled
	if cm.allStatsRolled() {
		ebitenutil.DebugPrintAt(dst, "Press Tab to continue", 50, y+40)
	}
}

func (cm *CreationManager) drawAssignStats(dst *ebiten.Image) {
	stats := cm.getGenerableStats()

	ebitenutil.DebugPrintAt(dst, "Assign Your Stats:", 50, 60)
	ebitenutil.DebugPrintAt(dst, "(Use arrows to navigate, Enter to assign)", 50, 78)

	// Draw available values
	ebitenutil.DebugPrintAt(dst, "Available:", 50, 100)
	x := 130
	for i, val := range cm.unassignedStats {
		// Check if assigned
		assigned := false
		for idx := range cm.assignmentMap {
			if idx == i {
				assigned = true
				break
			}
		}

		prefix := " "
		if i == cm.selectedUnassigned {
			prefix = "["
		}
		suffix := " "
		if i == cm.selectedUnassigned {
			suffix = "]"
		}

		text := fmt.Sprintf("%s%d%s", prefix, val, suffix)
		if assigned {
			text = fmt.Sprintf("%s*%s", prefix, suffix)
		}
		ebitenutil.DebugPrintAt(dst, text, x, 100)
		x += 30
	}

	// Draw stats
	y := 140
	for i, stat := range stats {
		prefix := "  "
		if i == cm.focusedField {
			prefix = "> "
		}

		text := fmt.Sprintf("%s%-12s:", prefix, stat.Name)
		ebitenutil.DebugPrintAt(dst, text, 50, y)

		// Show assigned value
		for idx, statID := range cm.assignmentMap {
			if statID == stat.ID {
				ebitenutil.DebugPrintAt(dst, fmt.Sprintf("%3d", cm.unassignedStats[idx]), 180, y)
				break
			}
		}

		y += 22
	}

	// Show continue hint if all assigned
	if len(cm.assignmentMap) == len(stats) {
		ebitenutil.DebugPrintAt(dst, "Press Tab to continue", 50, y+20)
	}
}

func (cm *CreationManager) drawReview(dst *ebiten.Image) {
	stats := cm.getGenerableStats()

	ebitenutil.DebugPrintAt(dst, fmt.Sprintf("Review: %s", cm.character.Name), 50, 60)

	y := 100
	for _, stat := range stats {
		abbr := stat.Abbreviation
		if abbr == "" {
			abbr = stat.Name[:3]
		}

		value := 0
		if sv := cm.character.GetStat(stat.ID); sv != nil {
			value = sv.Value
		}

		text := fmt.Sprintf("%-12s (%s): %3d", stat.Name, abbr, value)
		ebitenutil.DebugPrintAt(dst, text, 50, y)
		y += 20
	}

	// Options
	y += 30
	prefix := "  "
	if cm.focusedField == 0 {
		prefix = "> "
	}
	ebitenutil.DebugPrintAt(dst, prefix+"Accept Character", 50, y)

	prefix = "  "
	if cm.focusedField == 1 {
		prefix = "> "
	}
	ebitenutil.DebugPrintAt(dst, prefix+"Start Over", 50, y+20)
}

func (cm *CreationManager) drawNavHelp(dst *ebiten.Image) {
	y := cm.screenHeight - 30
	help := ""

	switch cm.state {
	case StateSelectMethod:
		help = "Up/Down: Select   Enter: Confirm"
	case StateEnterName:
		help = "Type name   Enter: Continue   Esc: Back"
	case StateRollStats:
		help = "Up/Down: Select   Enter/Space: Roll   R: Roll All   Tab: Next"
	case StateAssignStats:
		help = "Up/Down: Stat   Left/Right: Value   Enter: Assign   Tab: Next"
	case StateReview:
		help = "Up/Down: Select   Enter: Confirm   Esc: Back"
	}

	ebitenutil.DebugPrintAt(dst, help, 20, y)
}

// Helper drawing functions
func drawRect(dst *ebiten.Image, x, y, w, h int, r, g, b uint8) {
	rect := ebiten.NewImage(w, h)
	rect.Fill(color.RGBA{r, g, b, 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(rect, op)
}

func drawRectOutline(dst *ebiten.Image, x, y, w, h int, r, g, b uint8) {
	clr := color.RGBA{r, g, b, 255}
	// Top
	line := ebiten.NewImage(w, 1)
	line.Fill(clr)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(line, op)
	// Bottom
	op.GeoM.Reset()
	op.GeoM.Translate(float64(x), float64(y+h-1))
	dst.DrawImage(line, op)
	// Left
	line = ebiten.NewImage(1, h)
	line.Fill(clr)
	op.GeoM.Reset()
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(line, op)
	// Right
	op.GeoM.Reset()
	op.GeoM.Translate(float64(x+w-1), float64(y))
	dst.DrawImage(line, op)
}
