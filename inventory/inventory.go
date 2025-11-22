// Package inventory provides a simple item inventory system for players.
// Items are stored by name with quantities, suitable for both stackable and unique items.
package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
)

// Item represents a single item type in the inventory
type Item struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name,omitempty"`
	Description string            `json:"description,omitempty"`
	Stackable   bool              `json:"stackable"`
	MaxStack    int               `json:"max_stack,omitempty"` // 0 = unlimited
	Properties  map[string]string `json:"properties,omitempty"`
}

// InventorySlot represents an item and its quantity
type InventorySlot struct {
	ItemName string `json:"item_name"`
	Count    int    `json:"count"`
}

// Inventory holds all items for a player
type Inventory struct {
	mu sync.RWMutex

	// Slots maps item name to quantity
	Slots map[string]int `json:"slots"`

	// ItemDefinitions provides metadata about items (optional)
	// If not set, items are treated as simple named entities
	ItemDefinitions map[string]*Item `json:"-"`

	// MaxSlots limits total unique item types (0 = unlimited)
	MaxSlots int `json:"max_slots,omitempty"`

	// OnChange callback when inventory changes (for UI updates)
	OnChange func() `json:"-"`
}

// New creates a new empty inventory
func New() *Inventory {
	return &Inventory{
		Slots:           make(map[string]int),
		ItemDefinitions: make(map[string]*Item),
	}
}

// NewWithCapacity creates a new inventory with a slot limit
func NewWithCapacity(maxSlots int) *Inventory {
	inv := New()
	inv.MaxSlots = maxSlots
	return inv
}

// RegisterItem adds an item definition to the inventory system
func (inv *Inventory) RegisterItem(item *Item) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.ItemDefinitions[item.Name] = item
}

// GetItemDefinition returns the definition for an item, or nil if not defined
func (inv *Inventory) GetItemDefinition(name string) *Item {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.ItemDefinitions[name]
}

// HasItem checks if the inventory contains at least one of the named item
func (inv *Inventory) HasItem(itemName string) bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.Slots[itemName] > 0
}

// GetItemCount returns the quantity of an item (0 if not present)
func (inv *Inventory) GetItemCount(itemName string) int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.Slots[itemName]
}

// AddItem adds items to the inventory, returns actual amount added
func (inv *Inventory) AddItem(itemName string, count int) int {
	if count <= 0 {
		return 0
	}

	inv.mu.Lock()
	defer inv.mu.Unlock()

	// Check if we're adding a new item type and slots are limited
	_, exists := inv.Slots[itemName]
	if !exists && inv.MaxSlots > 0 && len(inv.Slots) >= inv.MaxSlots {
		return 0 // Inventory full
	}

	// Check max stack if item definition exists
	if def, ok := inv.ItemDefinitions[itemName]; ok && def.MaxStack > 0 {
		current := inv.Slots[itemName]
		room := def.MaxStack - current
		if count > room {
			count = room
		}
	}

	if count > 0 {
		inv.Slots[itemName] += count
		inv.notifyChange()
	}

	return count
}

// RemoveItem removes items from the inventory, returns true if successful
func (inv *Inventory) RemoveItem(itemName string, count int) bool {
	if count <= 0 {
		return true
	}

	inv.mu.Lock()
	defer inv.mu.Unlock()

	current := inv.Slots[itemName]
	if current < count {
		return false // Not enough items
	}

	inv.Slots[itemName] -= count
	if inv.Slots[itemName] <= 0 {
		delete(inv.Slots, itemName)
	}

	inv.notifyChange()
	return true
}

// ClearItem removes all of an item from the inventory
func (inv *Inventory) ClearItem(itemName string) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	delete(inv.Slots, itemName)
	inv.notifyChange()
}

// Clear removes all items from the inventory
func (inv *Inventory) Clear() {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.Slots = make(map[string]int)
	inv.notifyChange()
}

// GetAllItems returns a slice of all items and their quantities
func (inv *Inventory) GetAllItems() []InventorySlot {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	result := make([]InventorySlot, 0, len(inv.Slots))
	for name, count := range inv.Slots {
		result = append(result, InventorySlot{ItemName: name, Count: count})
	}

	// Sort by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].ItemName < result[j].ItemName
	})

	return result
}

// Count returns the number of unique item types in the inventory
func (inv *Inventory) Count() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return len(inv.Slots)
}

// TotalItems returns the total count of all items
func (inv *Inventory) TotalItems() int {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	total := 0
	for _, count := range inv.Slots {
		total += count
	}
	return total
}

// IsEmpty returns true if the inventory has no items
func (inv *Inventory) IsEmpty() bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return len(inv.Slots) == 0
}

// IsFull returns true if the inventory cannot accept new item types
func (inv *Inventory) IsFull() bool {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	return inv.MaxSlots > 0 && len(inv.Slots) >= inv.MaxSlots
}

// notifyChange calls the OnChange callback if set
func (inv *Inventory) notifyChange() {
	if inv.OnChange != nil {
		inv.OnChange()
	}
}

// --- Serialization ---

// Save writes the inventory to a file
func (inv *Inventory) Save(filepath string) error {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize inventory: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write inventory file: %w", err)
	}

	return nil
}

// Load reads an inventory from a file
func Load(filepath string) (*Inventory, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory file: %w", err)
	}

	inv := New()
	if err := json.Unmarshal(data, inv); err != nil {
		return nil, fmt.Errorf("failed to parse inventory: %w", err)
	}

	return inv, nil
}

// Clone creates a deep copy of the inventory
func (inv *Inventory) Clone() *Inventory {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	clone := New()
	clone.MaxSlots = inv.MaxSlots
	for k, v := range inv.Slots {
		clone.Slots[k] = v
	}
	// Note: ItemDefinitions are shared, not cloned
	clone.ItemDefinitions = inv.ItemDefinitions
	return clone
}

// Debug returns a string representation of the inventory
func (inv *Inventory) Debug() string {
	inv.mu.RLock()
	defer inv.mu.RUnlock()

	return fmt.Sprintf("Inventory{%d items, %d total}", len(inv.Slots), inv.TotalItems())
}

// --- Item Library ---

// ItemLibrary holds item definitions that can be shared across inventories
type ItemLibrary struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Items       map[string]*Item `json:"items"`
}

// LoadItemLibrary loads item definitions from a JSON file
func LoadItemLibrary(filepath string) (*ItemLibrary, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read item library: %w", err)
	}

	var library ItemLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		return nil, fmt.Errorf("failed to parse item library: %w", err)
	}

	if library.Items == nil {
		library.Items = make(map[string]*Item)
	}

	return &library, nil
}

// ApplyToInventory registers all items from the library to an inventory
func (lib *ItemLibrary) ApplyToInventory(inv *Inventory) {
	for name, item := range lib.Items {
		item.Name = name // Ensure name is set
		inv.RegisterItem(item)
	}
}
