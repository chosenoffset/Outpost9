package lighting

import (
	"fmt"
	"image/color"
	"strconv"

	"chosenoffset.com/outpost9/internal/world/furnishing"
)

// LightSource represents a single light source in the game world
type LightSource struct {
	X         float64     // World X position (in pixels)
	Y         float64     // World Y position (in pixels)
	Radius    float64     // Light radius (in pixels)
	Intensity float64     // Light intensity (0.0 to 1.0)
	Color     color.NRGBA // Light color
}

// Manager handles all light sources in the game
type Manager struct {
	lights         []LightSource
	ambientLight   float64 // Global ambient light level (0.0 = pitch black, 1.0 = fully lit)
	playerLight    *LightSource
	playerLightOn  bool
	furnishingLights map[string]*LightSource // Keyed by furnishing ID
}

// NewManager creates a new lighting manager
func NewManager() *Manager {
	return &Manager{
		lights:           make([]LightSource, 0),
		ambientLight:     0.15, // Low ambient light for better testing visibility
		playerLightOn:    false,
		furnishingLights: make(map[string]*LightSource),
	}
}

// SetAmbientLight sets the global ambient light level
func (m *Manager) SetAmbientLight(level float64) {
	m.ambientLight = level
}

// GetAmbientLight returns the current ambient light level
func (m *Manager) GetAmbientLight() float64 {
	return m.ambientLight
}

// SetPlayerLight configures the player's equipped light source
func (m *Manager) SetPlayerLight(x, y, radius, intensity float64, col color.NRGBA) {
	if m.playerLight == nil {
		m.playerLight = &LightSource{}
	}
	m.playerLight.X = x
	m.playerLight.Y = y
	m.playerLight.Radius = radius
	m.playerLight.Intensity = intensity
	m.playerLight.Color = col
}

// EnablePlayerLight turns on/off the player's light source
func (m *Manager) EnablePlayerLight(enabled bool) {
	m.playerLightOn = enabled
}

// IsPlayerLightOn returns whether the player's light is currently on
func (m *Manager) IsPlayerLightOn() bool {
	return m.playerLightOn
}

// UpdatePlayerLightPosition updates the player's light position (called each frame)
func (m *Manager) UpdatePlayerLightPosition(x, y float64) {
	if m.playerLight != nil {
		m.playerLight.X = x
		m.playerLight.Y = y
	}
}

// AddFurnishingLight adds a light source from a furnishing
func (m *Manager) AddFurnishingLight(furnishingID string, x, y int, tileSize int, def *furnishing.FurnishingDefinition) {
	// Check if this furnishing has the light_source tag
	if !def.HasTag("light_source") {
		return
	}

	// Parse light properties from furnishing properties
	radiusStr, hasRadius := def.GetProperty("light_radius")
	intensityStr, hasIntensity := def.GetProperty("light_intensity")

	if !hasRadius || !hasIntensity {
		fmt.Printf("WARNING: Furnishing %s has light_source tag but missing radius or intensity\n", def.Name)
		return
	}

	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		fmt.Printf("WARNING: Failed to parse light_radius for %s: %v\n", def.Name, err)
		return
	}

	intensity, err := strconv.ParseFloat(intensityStr, 64)
	if err != nil {
		fmt.Printf("WARNING: Failed to parse light_intensity for %s: %v\n", def.Name, err)
		return
	}

	// Parse light color (default to warm orange/yellow for torches)
	lightColor := color.NRGBA{255, 200, 100, 255} // Warm torch light
	if colorStr, hasColor := def.GetProperty("light_color"); hasColor {
		// Parse hex color (format: "RRGGBB")
		if len(colorStr) == 6 {
			var r, g, b uint8
			fmt.Sscanf(colorStr, "%02x%02x%02x", &r, &g, &b)
			lightColor = color.NRGBA{r, g, b, 255}
		}
	}

	// Convert grid position to world position (center of tile)
	worldX := float64(x*tileSize) + float64(tileSize)/2
	worldY := float64(y*tileSize) + float64(tileSize)/2

	light := &LightSource{
		X:         worldX,
		Y:         worldY,
		Radius:    radius,
		Intensity: intensity,
		Color:     lightColor,
	}

	m.furnishingLights[furnishingID] = light
	fmt.Printf("DEBUG: Added light %s at world pos (%.1f, %.1f) with radius=%.1f intensity=%.2f\n",
		furnishingID, worldX, worldY, radius, intensity)
}

// RemoveFurnishingLight removes a light source from a furnishing (e.g., if destroyed)
func (m *Manager) RemoveFurnishingLight(furnishingID string) {
	delete(m.furnishingLights, furnishingID)
}

// GetAllLights returns all active light sources
func (m *Manager) GetAllLights() []LightSource {
	lights := make([]LightSource, 0)

	// Add player light if enabled
	if m.playerLightOn && m.playerLight != nil {
		lights = append(lights, *m.playerLight)
	}

	// Add all furnishing lights
	for _, light := range m.furnishingLights {
		lights = append(lights, *light)
	}

	return lights
}

// ClearFurnishingLights removes all furnishing lights (called when loading new level)
func (m *Manager) ClearFurnishingLights() {
	m.furnishingLights = make(map[string]*LightSource)
}
