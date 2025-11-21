package placeholders

import (
	"fmt"
	"image"
	"image/color"
	"os"
)

// GenerateBaseTilesAtlas creates the base_tiles.png atlas
// This matches the simplified structure in atlas.json
func GenerateBaseTilesAtlas() *image.RGBA {
	// Simple structure:
	// Row 0: floor (0,0), floor_alt1 (16,0), floor_alt2 (32,0)
	// Row 1: wall (0,16), empty, empty

	tiles := make([]*image.RGBA, 6)

	// Row 0: Floors (0-2)
	tiles[0] = CreateSolidTile(ColorPalette.FloorMetal1)
	tiles[1] = CreateSolidTile(ColorPalette.FloorMetal2)
	tiles[2] = CreatePatternedTile(ColorPalette.FloorGrating, Darken(ColorPalette.FloorGrating, 0.7), "grid")

	// Row 1: Wall (3)
	tiles[3] = CreateBorderedTile(ColorPalette.WallMetal, Darken(ColorPalette.WallMetal, 0.3), 1)
	tiles[4] = nil
	tiles[5] = nil

	return CreateAtlas(tiles, 3) // 3 columns, 2 rows
}

// GenerateObjectTilesAtlas creates the object_tiles.png atlas
// This matches the structure defined in Art/objects_layer.json
func GenerateObjectTilesAtlas() *image.RGBA {
	// According to objects_layer.json, we need these tiles:
	// Row 0: desk_west, desk_east, chair_north, chair_south
	// Row 1: computer_terminal, console_left, console_center, console_right
	// Row 2: locker_closed, locker_open, crate, generator

	tiles := make([]*image.RGBA, 12)

	// Row 0: Furniture (0-3)
	tiles[0] = CreateDesk("west")
	tiles[1] = CreateDesk("east")
	tiles[2] = CreateChair("north")
	tiles[3] = CreateChair("south")

	// Row 1: Electronics (4-7)
	tiles[4] = CreateTerminal()
	tiles[5] = CreateConsole("left")
	tiles[6] = CreateConsole("center")
	tiles[7] = CreateConsole("right")

	// Row 2: Storage and machinery (8-11)
	tiles[8] = CreateLocker(false)
	tiles[9] = CreateLocker(true)
	tiles[10] = CreateCrate()
	tiles[11] = CreateGenerator()

	return CreateAtlas(tiles, 4) // 4 columns, 3 rows
}

// Wall creation helpers
func CreateWallCorner(direction string) *image.RGBA {
	img := CreateSolidTile(ColorPalette.WallMetal)
	borderColor := Lighten(ColorPalette.WallMetal, 0.3)

	// Add corner highlighting
	switch direction {
	case "nw":
		// Top and left edges
		for i := 0; i < TileSize; i++ {
			img.Set(i, 0, borderColor)
			img.Set(0, i, borderColor)
		}
	case "ne":
		// Top and right edges
		for i := 0; i < TileSize; i++ {
			img.Set(i, 0, borderColor)
			img.Set(TileSize-1, i, borderColor)
		}
	case "sw":
		// Bottom and left edges
		for i := 0; i < TileSize; i++ {
			img.Set(i, TileSize-1, borderColor)
			img.Set(0, i, borderColor)
		}
	case "se":
		// Bottom and right edges
		for i := 0; i < TileSize; i++ {
			img.Set(i, TileSize-1, borderColor)
			img.Set(TileSize-1, i, borderColor)
		}
	}

	return img
}

func CreateWallSegment(direction string) *image.RGBA {
	img := CreateSolidTile(ColorPalette.WallMetal)
	borderColor := Lighten(ColorPalette.WallMetal, 0.3)
	panelColor := ColorPalette.WallPanel

	// Add directional panels
	switch direction {
	case "n", "s":
		// Horizontal panel
		for y := 6; y < 10; y++ {
			for x := 2; x < TileSize-2; x++ {
				img.Set(x, y, panelColor)
			}
		}
		// Edge highlight
		edge := 0
		if direction == "s" {
			edge = TileSize - 1
		}
		for i := 0; i < TileSize; i++ {
			img.Set(i, edge, borderColor)
		}
	case "w", "e":
		// Vertical panel
		for x := 6; x < 10; x++ {
			for y := 2; y < TileSize-2; y++ {
				img.Set(x, y, panelColor)
			}
		}
		// Edge highlight
		edge := 0
		if direction == "e" {
			edge = TileSize - 1
		}
		for i := 0; i < TileSize; i++ {
			img.Set(edge, i, borderColor)
		}
	}

	return img
}

func CreateWallCenter() *image.RGBA {
	img := CreateSolidTile(ColorPalette.WallMetal)
	panelColor := ColorPalette.WallPanel

	// Add a centered panel
	for y := 4; y < 12; y++ {
		for x := 4; x < 12; x++ {
			img.Set(x, y, panelColor)
		}
	}

	return img
}

// Object creation helpers
func CreateDesk(direction string) *image.RGBA {
	img := CreateBorderedTile(ColorPalette.DeskWood, Darken(ColorPalette.DeskWood, 0.7), 2)

	// Add a small screen/object on desk (scaled for tile size)
	screenColor := color.RGBA{100, 150, 200, 255}
	quarter := TileSize / 4
	half := TileSize / 2
	threeQuarter := 3 * TileSize / 4

	if direction == "west" {
		for y := quarter; y < half+quarter/2; y++ {
			for x := half; x < threeQuarter; x++ {
				img.Set(x, y, screenColor)
			}
		}
	} else {
		for y := quarter; y < half+quarter/2; y++ {
			for x := quarter; x < half; x++ {
				img.Set(x, y, screenColor)
			}
		}
	}

	return img
}

func CreateChair(direction string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	chairColor := ColorPalette.ChairMetal

	// Scaled positions for chair
	eighth := TileSize / 8
	quarter := TileSize / 4
	threeEighths := 3 * TileSize / 8
	fiveEighths := 5 * TileSize / 8
	threeQuarter := 3 * TileSize / 4
	sevenEighths := 7 * TileSize / 8

	// Simple chair shape - seat and back
	switch direction {
	case "north":
		// Back at top
		for y := eighth; y < quarter; y++ {
			for x := quarter; x < threeQuarter; x++ {
				img.Set(x, y, chairColor)
			}
		}
		// Seat
		for y := threeEighths; y < fiveEighths; y++ {
			for x := threeEighths; x < fiveEighths+eighth; x++ {
				img.Set(x, y, chairColor)
			}
		}
	case "south":
		// Seat
		for y := threeEighths; y < fiveEighths; y++ {
			for x := threeEighths; x < fiveEighths+eighth; x++ {
				img.Set(x, y, chairColor)
			}
		}
		// Back at bottom
		for y := threeQuarter-eighth; y < sevenEighths; y++ {
			for x := quarter; x < threeQuarter; x++ {
				img.Set(x, y, chairColor)
			}
		}
	}

	return img
}

func CreateTerminal() *image.RGBA {
	img := CreateBorderedTile(ColorPalette.Terminal, Lighten(ColorPalette.Terminal, 0.3), 2)

	// Add a screen effect (scaled)
	screenColor := Lighten(ColorPalette.Terminal, 0.5)
	margin := TileSize / 8
	for y := margin; y < TileSize-margin; y++ {
		for x := margin; x < TileSize-margin; x++ {
			if y%2 == 0 { // Scanline effect
				img.Set(x, y, screenColor)
			}
		}
	}

	return img
}

func CreateConsole(position string) *image.RGBA {
	img := CreateSolidTile(ColorPalette.Console)
	buttonColor := Lighten(ColorPalette.Console, 0.4)

	// Scaled button positions
	third := TileSize / 3
	twoThirds := 2 * TileSize / 3
	quarter := TileSize / 4
	half := TileSize / 2
	threeQuarter := 3 * TileSize / 4
	buttonSize := TileSize / 8

	// Add buttons/lights based on position
	buttons := []image.Point{}
	switch position {
	case "left":
		buttons = []image.Point{{twoThirds, third}, {twoThirds, twoThirds}}
	case "center":
		buttons = []image.Point{{third, half}, {twoThirds, half}}
	case "right":
		buttons = []image.Point{{third, third}, {third, twoThirds}}
	}

	for _, p := range buttons {
		for dy := 0; dy < buttonSize; dy++ {
			for dx := 0; dx < buttonSize; dx++ {
				img.Set(p.X+dx, p.Y+dy, buttonColor)
			}
		}
	}

	// Suppress unused variable warnings
	_ = quarter
	_ = threeQuarter

	return img
}

func CreateLocker(open bool) *image.RGBA {
	lockerColor := ColorPalette.Locker
	if open {
		lockerColor = Darken(lockerColor, 0.7)
	}

	borderWidth := TileSize / 8
	img := CreateBorderedTile(lockerColor, Darken(ColorPalette.Locker, 0.5), borderWidth)

	// Add a handle (scaled)
	handleColor := color.RGBA{200, 200, 210, 255}
	handleY := TileSize / 2 - TileSize/16
	handleX := TileSize * 2 / 3
	handleSize := TileSize / 8
	for y := handleY; y < handleY+handleSize; y++ {
		for x := handleX; x < handleX+handleSize; x++ {
			img.Set(x, y, handleColor)
		}
	}

	return img
}

func CreateCrate() *image.RGBA {
	borderWidth := TileSize / 8
	img := CreateBorderedTile(ColorPalette.Crate, Darken(ColorPalette.Crate, 0.6), borderWidth)

	// Add cross pattern (scaled)
	crossColor := Darken(ColorPalette.Crate, 0.4)
	margin := TileSize / 8
	center := TileSize / 2
	for i := margin; i < TileSize-margin; i++ {
		img.Set(i, center, crossColor)
		img.Set(center, i, crossColor)
	}

	return img
}

func CreateGenerator() *image.RGBA {
	borderWidth := TileSize / 16
	img := CreateBorderedTile(ColorPalette.Generator, Darken(ColorPalette.Generator, 0.5), borderWidth)

	// Add some "energy" indicators (scaled)
	lightColor := color.RGBA{255, 255, 0, 255}
	quarter := TileSize / 4
	threeQuarter := 3 * TileSize / 4
	lights := []image.Point{{quarter, quarter}, {threeQuarter, quarter}, {quarter, threeQuarter}, {threeQuarter, threeQuarter}}

	lightSize := TileSize / 16
	for _, p := range lights {
		for dy := 0; dy < lightSize; dy++ {
			for dx := 0; dx < lightSize; dx++ {
				img.Set(p.X+dx, p.Y+dy, lightColor)
			}
		}
	}

	// Add center vent pattern (scaled)
	ventColor := Darken(ColorPalette.Generator, 0.7)
	threeEighths := 3 * TileSize / 8
	fiveEighths := 5 * TileSize / 8
	for y := threeEighths; y < fiveEighths; y++ {
		for x := threeEighths; x < fiveEighths; x++ {
			if x%2 == y%2 {
				img.Set(x, y, ventColor)
			}
		}
	}

	return img
}

// GenerateEntitiesAtlas creates the entities.png atlas
// Contains player, enemies, items, and effects
func GenerateEntitiesAtlas() *image.RGBA {
	// Layout:
	// Row 0: player_idle, player_walk_1, player_walk_2, player_walk_3
	// Row 1: enemy_basic, enemy_elite, enemy_boss, enemy_turret
	// Row 2: item_health, item_ammo, item_key, item_weapon
	// Row 3: projectile_bullet, projectile_plasma, effect_flash, effect_impact

	tiles := make([]*image.RGBA, 16)

	// Row 0: Player sprites (0-3)
	tiles[0] = CreatePlayerSprite()
	tiles[1] = CreatePlayerSprite() // Walk frame 1 (same for now)
	tiles[2] = CreatePlayerSprite() // Walk frame 2
	tiles[3] = CreatePlayerSprite() // Walk frame 3

	// Row 1: Enemy sprites (4-7)
	tiles[4] = CreateEnemySprite("basic")
	tiles[5] = CreateEnemySprite("elite")
	tiles[6] = CreateEnemySprite("boss")
	tiles[7] = CreateEnemySprite("turret")

	// Row 2: Item sprites (8-11)
	tiles[8] = CreateItemSprite("health")
	tiles[9] = CreateItemSprite("ammo")
	tiles[10] = CreateItemSprite("key")
	tiles[11] = CreateItemSprite("weapon")

	// Row 3: Projectile and effect sprites (12-15)
	tiles[12] = CreateProjectileSprite("bullet")
	tiles[13] = CreateProjectileSprite("plasma")
	tiles[14] = CreateEffectSprite("flash")
	tiles[15] = CreateEffectSprite("impact")

	return CreateAtlas(tiles, 4) // 4 columns, 4 rows
}

// Entity sprite creators
func CreatePlayerSprite() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	// Draw player as a rounded rectangle (body) - scaled
	bodyColor := ColorPalette.Player
	outlineColor := Darken(bodyColor, 0.5)

	// Scaled positions
	eighth := TileSize / 8
	quarter := TileSize / 4
	threeEighths := 3 * TileSize / 8
	half := TileSize / 2
	fiveEighths := 5 * TileSize / 8
	threeQuarter := 3 * TileSize / 4
	sevenEighths := 7 * TileSize / 8

	// Body (simplified humanoid shape)
	for y := quarter; y < sevenEighths; y++ {
		for x := threeEighths; x < fiveEighths+eighth; x++ {
			img.Set(x, y, bodyColor)
		}
	}

	// Head
	for y := eighth; y < quarter+eighth; y++ {
		for x := threeEighths+eighth/2; x < fiveEighths+eighth/2; x++ {
			img.Set(x, y, bodyColor)
		}
	}

	// Draw simple outline using border color
	for x := threeEighths; x < fiveEighths+eighth; x++ {
		img.Set(x, quarter, outlineColor) // Shoulders
		img.Set(x, sevenEighths-1, outlineColor) // Feet
	}

	// Suppress unused variable warning
	_ = half
	_ = threeQuarter

	return img
}

func CreateEnemySprite(enemyType string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	var enemyColor color.RGBA
	var sizeRatio float64 // Ratio of TileSize

	switch enemyType {
	case "basic":
		enemyColor = ColorPalette.EnemyBasic
		sizeRatio = 0.6
	case "elite":
		enemyColor = ColorPalette.EnemyElite
		sizeRatio = 0.7
	case "boss":
		enemyColor = color.RGBA{255, 100, 0, 255} // Orange
		sizeRatio = 0.85
	case "turret":
		enemyColor = color.RGBA{150, 150, 150, 255} // Gray
		sizeRatio = 0.75
	}

	// Draw as a hostile-looking shape (angular)
	center := TileSize / 2
	radius := int(float64(TileSize) * sizeRatio / 2)

	// Simple diamond/angular shape
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			dx := abs(x - center)
			dy := abs(y - center)

			if dx+dy <= radius {
				img.Set(x, y, enemyColor)
			} else if dx+dy <= radius+2 {
				img.Set(x, y, Darken(enemyColor, 0.5))
			}
		}
	}

	// Add hostile "eyes" or markers (scaled)
	eyeColor := color.RGBA{255, 255, 0, 255}
	eyeOffset := TileSize / 8
	eyeY := center - TileSize/16
	if enemyType != "turret" {
		img.Set(center-eyeOffset, eyeY, eyeColor)
		img.Set(center+eyeOffset, eyeY, eyeColor)
	}

	return img
}

func CreateItemSprite(itemType string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	var itemColor color.RGBA
	var symbol string

	switch itemType {
	case "health":
		itemColor = color.RGBA{0, 255, 0, 255} // Green
		symbol = "plus"
	case "ammo":
		itemColor = color.RGBA{255, 180, 0, 255} // Orange
		symbol = "box"
	case "key":
		itemColor = color.RGBA{255, 215, 0, 255} // Gold
		symbol = "key"
	case "weapon":
		itemColor = ColorPalette.Weapon
		symbol = "gun"
	}

	// Scaled positions
	quarter := TileSize / 4
	threeEighths := 3 * TileSize / 8
	half := TileSize / 2
	fiveEighths := 5 * TileSize / 8
	threeQuarter := 3 * TileSize / 4

	// Draw item with symbol
	switch symbol {
	case "plus":
		// Medical cross (scaled)
		crossWidth := TileSize / 8
		for y := quarter; y < threeQuarter; y++ {
			for x := half - crossWidth; x < half + crossWidth; x++ {
				img.Set(x, y, itemColor)
			}
		}
		for x := quarter; x < threeQuarter; x++ {
			for y := half - crossWidth; y < half + crossWidth; y++ {
				img.Set(x, y, itemColor)
			}
		}
	case "box":
		// Ammo box (scaled)
		for y := threeEighths; y < fiveEighths+quarter/2; y++ {
			for x := threeEighths; x < fiveEighths+quarter/2; x++ {
				if y == threeEighths || y == fiveEighths+quarter/2-1 ||
					x == threeEighths || x == fiveEighths+quarter/2-1 {
					img.Set(x, y, Darken(itemColor, 0.5))
				} else {
					img.Set(x, y, itemColor)
				}
			}
		}
	case "key":
		// Simple key shape (scaled)
		keyWidth := TileSize / 16
		for y := threeEighths; y < fiveEighths; y++ {
			for dx := 0; dx < keyWidth; dx++ {
				img.Set(half-keyWidth/2+dx, y, itemColor)
			}
		}
		for x := half - keyWidth/2; x < fiveEighths + quarter/2; x++ {
			for dy := 0; dy < keyWidth; dy++ {
				img.Set(x, fiveEighths-1+dy, itemColor)
			}
		}
		// Key head
		for y := threeEighths - keyWidth; y < threeEighths + keyWidth; y++ {
			for x := half - keyWidth; x < half + keyWidth; x++ {
				img.Set(x, y, itemColor)
			}
		}
	case "gun":
		// Simple gun shape (scaled)
		gunWidth := TileSize / 16
		for x := threeEighths; x < fiveEighths + quarter/2; x++ {
			for dy := 0; dy < gunWidth; dy++ {
				img.Set(x, half+dy, itemColor)
			}
		}
		for y := threeEighths + quarter/2; y < fiveEighths; y++ {
			for dx := 0; dx < gunWidth; dx++ {
				img.Set(fiveEighths+dx, y, itemColor)
			}
		}
	}

	return img
}

func CreateProjectileSprite(projType string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	var projColor color.RGBA
	center := TileSize / 2

	switch projType {
	case "bullet":
		projColor = color.RGBA{255, 255, 0, 255} // Yellow
		// Small bullet (scaled)
		bulletSize := TileSize / 8
		for y := center - bulletSize/2; y < center + bulletSize/2; y++ {
			for x := center - bulletSize/2; x < center + bulletSize; x++ {
				img.Set(x, y, projColor)
			}
		}
	case "plasma":
		projColor = color.RGBA{0, 255, 255, 255} // Cyan
		// Larger plasma ball (scaled)
		radius := TileSize / 4
		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				dx := x - center
				dy := y - center
				if dx*dx+dy*dy <= radius*radius {
					img.Set(x, y, projColor)
				}
			}
		}
	}

	return img
}

func CreateEffectSprite(effectType string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	center := TileSize / 2
	eighth := TileSize / 8

	switch effectType {
	case "flash":
		// Muzzle flash - star burst (scaled)
		flashColor := color.RGBA{255, 255, 100, 255}
		// Main cross
		for i := 0; i < TileSize; i++ {
			img.Set(i, center, flashColor)
			img.Set(center, i, flashColor)
		}
		// Diagonal rays
		for i := center - eighth; i <= center+eighth; i++ {
			for j := 0; j < TileSize; j++ {
				img.Set(j, i, flashColor)
				img.Set(i, j, flashColor)
			}
		}
	case "impact":
		// Impact spark (scaled)
		impactColor := color.RGBA{255, 100, 0, 255}
		radius := TileSize / 4
		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				dx := abs(x - center)
				dy := abs(y - center)
				if dx+dy <= radius {
					img.Set(x, y, impactColor)
				}
			}
		}
	}

	return img
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GenerateAndSave generates all atlases and saves them to the specified game directory
func GenerateAndSave(gameDir string) error {
	fmt.Println("Generating placeholder atlases...")

	assetsDir := fmt.Sprintf("%s/assets", gameDir)

	// Ensure assets directory exists
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Generate base tiles
	baseAtlas := GenerateBaseTilesAtlas()
	if err := SavePNG(baseAtlas, fmt.Sprintf("%s/base_tiles.png", assetsDir)); err != nil {
		return fmt.Errorf("failed to save base_tiles.png: %w", err)
	}
	fmt.Printf("✓ Generated %s/base_tiles.png (%dx%d pixels, 3x2 tiles @ %dpx)\n",
		assetsDir, 3*TileSize, 2*TileSize, TileSize)

	// Generate object tiles
	objectAtlas := GenerateObjectTilesAtlas()
	if err := SavePNG(objectAtlas, fmt.Sprintf("%s/object_tiles.png", assetsDir)); err != nil {
		return fmt.Errorf("failed to save object_tiles.png: %w", err)
	}
	fmt.Printf("✓ Generated %s/object_tiles.png (%dx%d pixels, 4x3 tiles @ %dpx)\n",
		assetsDir, 4*TileSize, 3*TileSize, TileSize)

	// Generate entities
	entitiesAtlas := GenerateEntitiesAtlas()
	if err := SavePNG(entitiesAtlas, fmt.Sprintf("%s/entities.png", assetsDir)); err != nil {
		return fmt.Errorf("failed to save entities.png: %w", err)
	}
	fmt.Printf("✓ Generated %s/entities.png (%dx%d pixels, 4x4 tiles @ %dpx)\n",
		assetsDir, 4*TileSize, 4*TileSize, TileSize)

	fmt.Println("Placeholder atlases generated successfully!")
	return nil
}
