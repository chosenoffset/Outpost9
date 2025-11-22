// Package hud provides a data-driven heads-up display for showing player stats,
// combat info, and other game information during gameplay.
package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"chosenoffset.com/outpost9/character"
	"chosenoffset.com/outpost9/entity"
)

// HUDConfig defines what to display in the HUD
type HUDConfig struct {
	ShowStats      bool     `json:"show_stats"`      // Show character stats
	ShowHP         bool     `json:"show_hp"`         // Show HP bar
	ShowTurnInfo   bool     `json:"show_turn_info"`  // Show turn number
	ShowPosition   bool     `json:"show_position"`   // Show grid position
	StatCategories []string `json:"stat_categories"` // Which categories to show (empty = all)
	CompactMode    bool     `json:"compact_mode"`    // Use compact single-line display
	Position       string   `json:"position"`        // "top-left", "top-right", "bottom-left", "bottom-right"
	Opacity        float64  `json:"opacity"`         // Background opacity (0-1)
}

// DefaultConfig returns a sensible default HUD configuration
func DefaultConfig() *HUDConfig {
	return &HUDConfig{
		ShowStats:    true,
		ShowHP:       true,
		ShowTurnInfo: true,
		ShowPosition: false,
		CompactMode:  false,
		Position:     "top-left",
		Opacity:      0.7,
	}
}

// HUD manages the heads-up display
type HUD struct {
	config       *HUDConfig
	screenWidth  int
	screenHeight int

	// Data sources
	playerEntity *entity.Entity
	playerChar   *character.Character
	template     *character.CharacterTemplate

	// Turn info
	turnNumber int

	// Cached layout
	panelWidth  int
	panelHeight int
}

// New creates a new HUD with the given configuration
func New(config *HUDConfig, screenWidth, screenHeight int) *HUD {
	if config == nil {
		config = DefaultConfig()
	}
	return &HUD{
		config:       config,
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		panelWidth:   180,
	}
}

// SetPlayer sets the player entity and character to display
func (h *HUD) SetPlayer(playerEntity *entity.Entity, playerChar *character.Character) {
	h.playerEntity = playerEntity
	h.playerChar = playerChar
	if playerChar != nil {
		h.template = playerChar.GetTemplate()
	}
}

// SetTurnNumber updates the displayed turn number
func (h *HUD) SetTurnNumber(turn int) {
	h.turnNumber = turn
}

// SetScreenSize updates the screen dimensions
func (h *HUD) SetScreenSize(width, height int) {
	h.screenWidth = width
	h.screenHeight = height
}

// Draw renders the HUD to the screen
func (h *HUD) Draw(screen *ebiten.Image) {
	if h.playerEntity == nil {
		return
	}

	// Calculate position based on config
	x, y := h.calculatePosition()

	// Draw panel background
	h.drawPanel(screen, x, y)

	// Current Y offset for drawing
	currentY := y + 8

	// Draw player name
	if h.playerEntity.Name != "" {
		h.drawText(screen, h.playerEntity.Name, x+8, currentY, color.RGBA{255, 255, 200, 255})
		currentY += 16
	}

	// Draw HP bar
	if h.config.ShowHP {
		currentY = h.drawHPBar(screen, x+8, currentY)
		currentY += 8
	}

	// Draw divider
	h.drawDivider(screen, x+4, currentY, h.panelWidth-8)
	currentY += 8

	// Draw stats by category
	if h.config.ShowStats && h.template != nil {
		currentY = h.drawStats(screen, x+8, currentY)
	}

	// Draw turn info
	if h.config.ShowTurnInfo {
		h.drawDivider(screen, x+4, currentY, h.panelWidth-8)
		currentY += 8
		h.drawText(screen, fmt.Sprintf("Turn: %d", h.turnNumber), x+8, currentY, color.RGBA{180, 180, 180, 255})
		currentY += 16
	}

	// Draw position
	if h.config.ShowPosition {
		posText := fmt.Sprintf("Pos: %d, %d", h.playerEntity.X, h.playerEntity.Y)
		h.drawText(screen, posText, x+8, currentY, color.RGBA{150, 150, 150, 255})
		currentY += 16
	}
}

// calculatePosition returns the top-left corner of the HUD panel
func (h *HUD) calculatePosition() (int, int) {
	padding := 10

	switch h.config.Position {
	case "top-right":
		return h.screenWidth - h.panelWidth - padding, padding
	case "bottom-left":
		return padding, h.screenHeight - h.panelHeight - padding
	case "bottom-right":
		return h.screenWidth - h.panelWidth - padding, h.screenHeight - h.panelHeight - padding
	default: // "top-left"
		return padding, padding
	}
}

// drawPanel draws the semi-transparent background panel
func (h *HUD) drawPanel(screen *ebiten.Image, x, y int) {
	// Calculate panel height dynamically based on content
	height := h.calculatePanelHeight()
	h.panelHeight = height

	// Create panel with transparency
	alpha := uint8(h.config.Opacity * 255)
	panelColor := color.RGBA{20, 20, 30, alpha}

	panel := ebiten.NewImage(h.panelWidth, height)
	panel.Fill(panelColor)

	// Draw border
	borderColor := color.RGBA{60, 60, 80, alpha}
	for i := 0; i < h.panelWidth; i++ {
		panel.Set(i, 0, borderColor)
		panel.Set(i, height-1, borderColor)
	}
	for i := 0; i < height; i++ {
		panel.Set(0, i, borderColor)
		panel.Set(h.panelWidth-1, i, borderColor)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(panel, op)
}

// calculatePanelHeight calculates the height needed for all HUD elements
func (h *HUD) calculatePanelHeight() int {
	height := 16 // Padding

	// Name
	if h.playerEntity != nil && h.playerEntity.Name != "" {
		height += 16
	}

	// HP bar
	if h.config.ShowHP {
		height += 24
	}

	// Divider
	height += 8

	// Stats
	if h.config.ShowStats && h.template != nil {
		// Count visible stats
		statCount := 0
		for _, stat := range h.template.Stats {
			if stat.Hidden {
				continue
			}
			if len(h.config.StatCategories) > 0 && !h.categoryIncluded(stat.Category) {
				continue
			}
			statCount++
		}
		height += statCount * 14
		height += 8 // Extra spacing
	}

	// Turn info
	if h.config.ShowTurnInfo {
		height += 24
	}

	// Position
	if h.config.ShowPosition {
		height += 16
	}

	return height + 8 // Bottom padding
}

// categoryIncluded checks if a category should be shown
func (h *HUD) categoryIncluded(category string) bool {
	if len(h.config.StatCategories) == 0 {
		return true
	}
	for _, cat := range h.config.StatCategories {
		if cat == category {
			return true
		}
	}
	return false
}

// drawHPBar draws the player's HP bar
func (h *HUD) drawHPBar(screen *ebiten.Image, x, y int) int {
	if h.playerEntity == nil {
		return y
	}

	barWidth := h.panelWidth - 24
	barHeight := 12

	// Background
	bg := ebiten.NewImage(barWidth, barHeight)
	bg.Fill(color.RGBA{60, 20, 20, 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(bg, op)

	// Health fill
	if h.playerEntity.MaxHP > 0 {
		healthPct := float64(h.playerEntity.CurrentHP) / float64(h.playerEntity.MaxHP)
		if healthPct > 0 {
			fillWidth := int(float64(barWidth) * healthPct)
			if fillWidth < 1 {
				fillWidth = 1
			}

			// Color based on health percentage
			var fillColor color.RGBA
			if healthPct > 0.6 {
				fillColor = color.RGBA{50, 180, 50, 255} // Green
			} else if healthPct > 0.3 {
				fillColor = color.RGBA{200, 180, 50, 255} // Yellow
			} else {
				fillColor = color.RGBA{200, 50, 50, 255} // Red
			}

			fill := ebiten.NewImage(fillWidth, barHeight-2)
			fill.Fill(fillColor)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x+1), float64(y+1))
			screen.DrawImage(fill, op)
		}
	}

	// HP text
	hpText := fmt.Sprintf("%d/%d", h.playerEntity.CurrentHP, h.playerEntity.MaxHP)
	textX := x + barWidth/2 - len(hpText)*3
	h.drawText(screen, hpText, textX, y, color.RGBA{255, 255, 255, 255})

	return y + barHeight + 4
}

// drawStats draws all character stats organized by category
func (h *HUD) drawStats(screen *ebiten.Image, x, y int) int {
	if h.playerChar == nil || h.template == nil {
		return y
	}

	// Group stats by category
	categories := make(map[string][]*character.StatDefinition)
	categoryOrder := make([]string, 0)

	for i := range h.template.Stats {
		stat := &h.template.Stats[i]
		if stat.Hidden {
			continue
		}
		if len(h.config.StatCategories) > 0 && !h.categoryIncluded(stat.Category) {
			continue
		}

		cat := stat.Category
		if cat == "" {
			cat = "general"
		}

		if _, exists := categories[cat]; !exists {
			categoryOrder = append(categoryOrder, cat)
		}
		categories[cat] = append(categories[cat], stat)
	}

	// Draw stats by category
	for _, catID := range categoryOrder {
		stats := categories[catID]

		// Get category display name
		catName := catID
		if cat := h.template.GetCategory(catID); cat != nil {
			catName = cat.Name
		}

		// Draw category header (only if multiple categories)
		if len(categoryOrder) > 1 {
			h.drawText(screen, catName+":", x, y, color.RGBA{150, 150, 180, 255})
			y += 14
		}

		// Draw each stat
		for _, stat := range stats {
			y = h.drawStat(screen, x, y, stat)
		}

		y += 4 // Spacing between categories
	}

	return y
}

// drawStat draws a single stat line
func (h *HUD) drawStat(screen *ebiten.Image, x, y int, stat *character.StatDefinition) int {
	// Get abbreviation or short name
	label := stat.Abbreviation
	if label == "" {
		if len(stat.Name) > 3 {
			label = stat.Name[:3]
		} else {
			label = stat.Name
		}
	}

	// Get value
	value := 0
	if sv := h.playerChar.GetStat(stat.ID); sv != nil {
		value = sv.GetTotal()
	}

	// Calculate modifier for D&D-style stats
	mod := (value - 10) / 2
	modStr := ""
	if stat.Category == "attributes" {
		if mod >= 0 {
			modStr = fmt.Sprintf(" (+%d)", mod)
		} else {
			modStr = fmt.Sprintf(" (%d)", mod)
		}
	}

	// Format: "STR: 16 (+3)"
	text := fmt.Sprintf("%-4s %2d%s", label+":", value, modStr)

	// Color based on value (for attributes typically 3-18)
	var textColor color.RGBA
	if stat.Category == "attributes" {
		if value >= 16 {
			textColor = color.RGBA{100, 255, 100, 255} // High - green
		} else if value >= 12 {
			textColor = color.RGBA{200, 200, 200, 255} // Above average - white
		} else if value >= 9 {
			textColor = color.RGBA{180, 180, 150, 255} // Average - tan
		} else {
			textColor = color.RGBA{255, 150, 100, 255} // Low - orange
		}
	} else {
		textColor = color.RGBA{200, 200, 200, 255}
	}

	h.drawText(screen, text, x, y, textColor)
	return y + 14
}

// drawDivider draws a horizontal line
func (h *HUD) drawDivider(screen *ebiten.Image, x, y, width int) {
	line := ebiten.NewImage(width, 1)
	line.Fill(color.RGBA{80, 80, 100, 200})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(line, op)
}

// drawText draws text with a shadow for readability
func (h *HUD) drawText(screen *ebiten.Image, text string, x, y int, clr color.RGBA) {
	// Shadow
	ebitenutil.DebugPrintAt(screen, text, x+1, y+1)
	// Main text (debug print is always white, but we can use it for now)
	ebitenutil.DebugPrintAt(screen, text, x, y)
}
