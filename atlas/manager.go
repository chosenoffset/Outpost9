package atlas

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Manager manages multiple sprite atlases organized by layer
type Manager struct {
	atlasesByLayer map[string]*Atlas // Atlases organized by layer name
	atlasesByName  map[string]*Atlas // Atlases organized by atlas name
}

// NewManager creates a new atlas manager
func NewManager() *Manager {
	return &Manager{
		atlasesByLayer: make(map[string]*Atlas),
		atlasesByName:  make(map[string]*Atlas),
	}
}

// LoadAtlasConfig loads an atlas from a config file and registers it
func (m *Manager) LoadAtlasConfig(configPath string) error {
	atlas, err := LoadAtlas(configPath)
	if err != nil {
		return err
	}

	return m.RegisterAtlas(atlas)
}

// RegisterAtlas registers a loaded atlas with the manager
func (m *Manager) RegisterAtlas(atlas *Atlas) error {
	if atlas.Config.Layer == "" {
		return fmt.Errorf("atlas layer cannot be empty")
	}

	if atlas.Config.Name == "" {
		return fmt.Errorf("atlas name cannot be empty")
	}

	// Check for duplicate layer (one atlas per layer)
	if existing, exists := m.atlasesByLayer[atlas.Config.Layer]; exists {
		return fmt.Errorf("layer %s already has an atlas registered: %s", atlas.Config.Layer, existing.Config.Name)
	}

	m.atlasesByLayer[atlas.Config.Layer] = atlas
	m.atlasesByName[atlas.Config.Name] = atlas

	return nil
}

// GetAtlasByLayer returns the atlas for a specific layer
func (m *Manager) GetAtlasByLayer(layer string) (*Atlas, bool) {
	atlas, ok := m.atlasesByLayer[layer]
	return atlas, ok
}

// GetAtlasByName returns an atlas by its name
func (m *Manager) GetAtlasByName(name string) (*Atlas, bool) {
	atlas, ok := m.atlasesByName[name]
	return atlas, ok
}

// DrawTile draws a tile from a specific layer at screen coordinates
func (m *Manager) DrawTile(screen *ebiten.Image, layer, tileName string, x, y float64) error {
	atlas, ok := m.GetAtlasByLayer(layer)
	if !ok {
		return fmt.Errorf("no atlas found for layer: %s", layer)
	}

	return atlas.DrawTile(screen, tileName, x, y)
}

// DrawTileWithOptions draws a tile with custom options from a specific layer
func (m *Manager) DrawTileWithOptions(screen *ebiten.Image, layer, tileName string, opts *ebiten.DrawImageOptions) error {
	atlas, ok := m.GetAtlasByLayer(layer)
	if !ok {
		return fmt.Errorf("no atlas found for layer: %s", layer)
	}

	return atlas.DrawTileWithOptions(screen, tileName, opts)
}

// GetTile retrieves a tile definition from a specific layer
func (m *Manager) GetTile(layer, tileName string) (*TileDefinition, error) {
	atlas, ok := m.GetAtlasByLayer(layer)
	if !ok {
		return nil, fmt.Errorf("no atlas found for layer: %s", layer)
	}

	tile, ok := atlas.GetTile(tileName)
	if !ok {
		return nil, fmt.Errorf("tile %s not found in layer %s", tileName, layer)
	}

	return tile, nil
}

// GetLayers returns all registered layer names
func (m *Manager) GetLayers() []string {
	layers := make([]string, 0, len(m.atlasesByLayer))
	for layer := range m.atlasesByLayer {
		layers = append(layers, layer)
	}
	return layers
}
