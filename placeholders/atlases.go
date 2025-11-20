package placeholders

import (
	"fmt"
	"image"
	"image/color"
)

// GenerateBaseTilesAtlas creates the base_tiles.png atlas
// This matches the structure defined in Art/base_layer.json
func GenerateBaseTilesAtlas() *image.RGBA {
	// According to base_layer.json, we need these tiles:
	// Row 0: floor_metal_01, floor_metal_02, floor_grating
	// Row 1: wall_corner_nw, wall_n, wall_corner_ne
	// Row 2: wall_w, wall_center, wall_e
	// Row 3: wall_corner_sw, wall_s, wall_corner_se

	tiles := make([]*image.RGBA, 12)

	// Row 0: Floors (0-2)
	tiles[0] = CreateSolidTile(ColorPalette.FloorMetal1)
	tiles[1] = CreateSolidTile(ColorPalette.FloorMetal2)
	tiles[2] = CreatePatternedTile(ColorPalette.FloorGrating, Darken(ColorPalette.FloorGrating, 0.7), "grid")

	// Row 1: Top wall row (3-5)
	tiles[3] = CreateWallCorner("nw")
	tiles[4] = CreateWallSegment("n")
	tiles[5] = CreateWallCorner("ne")

	// Row 2: Middle wall row (6-8)
	tiles[6] = CreateWallSegment("w")
	tiles[7] = CreateWallCenter()
	tiles[8] = CreateWallSegment("e")

	// Row 3: Bottom wall row (9-11)
	tiles[9] = CreateWallCorner("sw")
	tiles[10] = CreateWallSegment("s")
	tiles[11] = CreateWallCorner("se")

	return CreateAtlas(tiles, 3) // 3 columns, 4 rows
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
	img := CreateBorderedTile(ColorPalette.DeskWood, Darken(ColorPalette.DeskWood, 0.7), 1)

	// Add a small screen/object on desk
	screenColor := color.RGBA{100, 150, 200, 255}
	if direction == "west" {
		for y := 5; y < 9; y++ {
			for x := 10; x < 14; x++ {
				img.Set(x, y, screenColor)
			}
		}
	} else {
		for y := 5; y < 9; y++ {
			for x := 2; x < 6; x++ {
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

	// Simple chair shape - seat and back
	switch direction {
	case "north":
		// Back at top
		for y := 2; y < 5; y++ {
			for x := 4; x < 12; x++ {
				img.Set(x, y, chairColor)
			}
		}
		// Seat
		for y := 6; y < 10; y++ {
			for x := 5; x < 11; x++ {
				img.Set(x, y, chairColor)
			}
		}
	case "south":
		// Seat
		for y := 6; y < 10; y++ {
			for x := 5; x < 11; x++ {
				img.Set(x, y, chairColor)
			}
		}
		// Back at bottom
		for y := 11; y < 14; y++ {
			for x := 4; x < 12; x++ {
				img.Set(x, y, chairColor)
			}
		}
	}

	return img
}

func CreateTerminal() *image.RGBA {
	img := CreateBorderedTile(ColorPalette.Terminal, Lighten(ColorPalette.Terminal, 0.3), 1)

	// Add a screen effect
	screenColor := Lighten(ColorPalette.Terminal, 0.5)
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
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

	// Add buttons/lights based on position
	buttons := []image.Point{}
	switch position {
	case "left":
		buttons = []image.Point{{10, 6}, {10, 10}}
	case "center":
		buttons = []image.Point{{6, 8}, {10, 8}}
	case "right":
		buttons = []image.Point{{6, 6}, {6, 10}}
	}

	for _, p := range buttons {
		for dy := 0; dy < 2; dy++ {
			for dx := 0; dx < 2; dx++ {
				img.Set(p.X+dx, p.Y+dy, buttonColor)
			}
		}
	}

	return img
}

func CreateLocker(open bool) *image.RGBA {
	lockerColor := ColorPalette.Locker
	if open {
		lockerColor = Darken(lockerColor, 0.7)
	}

	img := CreateBorderedTile(lockerColor, Darken(ColorPalette.Locker, 0.5), 2)

	// Add a handle
	handleColor := color.RGBA{200, 200, 210, 255}
	for y := 7; y < 9; y++ {
		for x := 11; x < 13; x++ {
			img.Set(x, y, handleColor)
		}
	}

	return img
}

func CreateCrate() *image.RGBA {
	img := CreateBorderedTile(ColorPalette.Crate, Darken(ColorPalette.Crate, 0.6), 2)

	// Add cross pattern
	crossColor := Darken(ColorPalette.Crate, 0.4)
	for i := 3; i < 13; i++ {
		img.Set(i, 8, crossColor)
		img.Set(8, i, crossColor)
	}

	return img
}

func CreateGenerator() *image.RGBA {
	img := CreateBorderedTile(ColorPalette.Generator, Darken(ColorPalette.Generator, 0.5), 1)

	// Add some "energy" indicators
	lightColor := color.RGBA{255, 255, 0, 255}
	lights := []image.Point{{4, 4}, {11, 4}, {4, 11}, {11, 11}}

	for _, p := range lights {
		img.Set(p.X, p.Y, lightColor)
	}

	// Add center vent pattern
	ventColor := Darken(ColorPalette.Generator, 0.7)
	for y := 6; y < 10; y++ {
		for x := 6; x < 10; x++ {
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

	// Draw player as a rounded rectangle (body)
	bodyColor := ColorPalette.Player
	outlineColor := Darken(bodyColor, 0.5)

	// Body (simplified humanoid shape)
	for y := 4; y < 14; y++ {
		for x := 5; x < 11; x++ {
			img.Set(x, y, bodyColor)
		}
	}

	// Head
	for y := 2; y < 5; y++ {
		for x := 6; x < 10; x++ {
			img.Set(x, y, bodyColor)
		}
	}

	// Outline
	outlinePoints := []image.Point{
		{6, 2}, {7, 2}, {8, 2}, {9, 2}, // Head top
		{5, 4}, {10, 4}, // Shoulders
		{5, 13}, {10, 13}, // Feet
	}

	for _, p := range outlinePoints {
		img.Set(p.X, p.Y, outlineColor)
	}

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
	var size int

	switch enemyType {
	case "basic":
		enemyColor = ColorPalette.EnemyBasic
		size = 10
	case "elite":
		enemyColor = ColorPalette.EnemyElite
		size = 11
	case "boss":
		enemyColor = color.RGBA{255, 100, 0, 255} // Orange
		size = 14
	case "turret":
		enemyColor = color.RGBA{150, 150, 150, 255} // Gray
		size = 12
	}

	// Draw as a hostile-looking shape (angular)
	center := TileSize / 2
	radius := size / 2

	// Simple diamond/angular shape
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			dx := abs(x - center)
			dy := abs(y - center)

			if dx+dy <= radius {
				img.Set(x, y, enemyColor)
			} else if dx+dy <= radius+1 {
				img.Set(x, y, Darken(enemyColor, 0.5))
			}
		}
	}

	// Add hostile "eyes" or markers
	eyeColor := color.RGBA{255, 255, 0, 255}
	if enemyType != "turret" {
		img.Set(center-2, center-1, eyeColor)
		img.Set(center+2, center-1, eyeColor)
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

	// Draw item with symbol
	switch symbol {
	case "plus":
		// Medical cross
		for x := 6; x < 10; x++ {
			for y := 4; y < 12; y++ {
				if (x >= 7 && x <= 8) || (y >= 7 && y <= 8) {
					img.Set(x, y, itemColor)
				}
			}
		}
	case "box":
		// Ammo box
		for y := 5; y < 11; y++ {
			for x := 5; x < 11; x++ {
				if y == 5 || y == 10 || x == 5 || x == 10 {
					img.Set(x, y, Darken(itemColor, 0.5))
				} else {
					img.Set(x, y, itemColor)
				}
			}
		}
	case "key":
		// Simple key shape
		for y := 6; y < 10; y++ {
			img.Set(7, y, itemColor)
		}
		for x := 7; x < 11; x++ {
			img.Set(x, 9, itemColor)
		}
		img.Set(8, 6, itemColor)
		img.Set(9, 6, itemColor)
	case "gun":
		// Simple gun shape
		for x := 5; x < 11; x++ {
			img.Set(x, 8, itemColor)
		}
		for y := 7; y < 10; y++ {
			img.Set(9, y, itemColor)
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

	switch projType {
	case "bullet":
		projColor = color.RGBA{255, 255, 0, 255} // Yellow
		// Small bullet
		for y := 7; y < 9; y++ {
			for x := 7; x < 10; x++ {
				img.Set(x, y, projColor)
			}
		}
	case "plasma":
		projColor = color.RGBA{0, 255, 255, 255} // Cyan
		// Larger plasma ball
		center := TileSize / 2
		for y := 5; y < 11; y++ {
			for x := 5; x < 11; x++ {
				dx := x - center
				dy := y - center
				if dx*dx+dy*dy <= 9 {
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

	switch effectType {
	case "flash":
		// Muzzle flash - star burst
		flashColor := color.RGBA{255, 255, 100, 255}
		for i := 0; i < TileSize; i++ {
			img.Set(i, center, flashColor)
			img.Set(center, i, flashColor)
			if i == center-2 || i == center+2 {
				for j := 0; j < TileSize; j++ {
					img.Set(j, i, flashColor)
					img.Set(i, j, flashColor)
				}
			}
		}
	case "impact":
		// Impact spark
		impactColor := color.RGBA{255, 100, 0, 255}
		for y := 4; y < 12; y++ {
			for x := 4; x < 12; x++ {
				dx := abs(x - center)
				dy := abs(y - center)
				if dx+dy <= 4 {
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

// GenerateAndSave generates all atlases and saves them
func GenerateAndSave() error {
	fmt.Println("Generating placeholder atlases...")

	// Generate base tiles
	baseAtlas := GenerateBaseTilesAtlas()
	if err := SavePNG(baseAtlas, "Art/base_tiles.png"); err != nil {
		return fmt.Errorf("failed to save base_tiles.png: %w", err)
	}
	fmt.Println("✓ Generated Art/base_tiles.png (48x64, 3x4 tiles)")

	// Generate object tiles
	objectAtlas := GenerateObjectTilesAtlas()
	if err := SavePNG(objectAtlas, "Art/object_tiles.png"); err != nil {
		return fmt.Errorf("failed to save object_tiles.png: %w", err)
	}
	fmt.Println("✓ Generated Art/object_tiles.png (64x48, 4x3 tiles)")

	// Generate entities
	entitiesAtlas := GenerateEntitiesAtlas()
	if err := SavePNG(entitiesAtlas, "Art/entities.png"); err != nil {
		return fmt.Errorf("failed to save entities.png: %w", err)
	}
	fmt.Println("✓ Generated Art/entities.png (64x64, 4x4 tiles)")

	fmt.Println("Placeholder atlases generated successfully!")
	return nil
}
