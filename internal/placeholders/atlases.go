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

	// Row 0: Stone dungeon floors (0-2)
	tiles[0] = CreateSolidTile(ColorPalette.FloorStone1)
	tiles[1] = CreateSolidTile(ColorPalette.FloorStone2)
	tiles[2] = CreatePatternedTile(ColorPalette.FloorCobble, Darken(ColorPalette.FloorCobble, 0.7), "grid")

	// Row 1: Stone wall (3)
	tiles[3] = CreateBorderedTile(ColorPalette.WallStone, Darken(ColorPalette.WallStone, 0.3), 1)
	tiles[4] = nil
	tiles[5] = nil

	return CreateAtlas(tiles, 3) // 3 columns, 2 rows
}

// GenerateObjectTilesAtlas creates the object_tiles.png atlas
// This matches the structure defined in Art/objects_layer.json
func GenerateObjectTilesAtlas() *image.RGBA {
	// Dungeon themed objects:
	// Row 0: stone_table (west), healing_fountain (east), skeleton_remains (north), wooden_stool (south)
	// Row 1: ancient_tome, torch_sconce_left, stone_altar, torch_sconce_right
	// Row 2: weapon_rack_closed, weapon_rack_open (potion_shelf), wooden_barrel, magical_brazier
	// Row 3: door_closed, door_open, treasure_chest_closed, treasure_chest_open

	tiles := make([]*image.RGBA, 16)

	// Row 0: Dungeon furniture (0-3)
	tiles[0] = CreateStoneTable("west")
	tiles[1] = CreateHealingFountain()
	tiles[2] = CreateSkeletonRemains()
	tiles[3] = CreateWoodenStool()

	// Row 1: Magic objects (4-7)
	tiles[4] = CreateAncientTome()
	tiles[5] = CreateTorchSconce("left")
	tiles[6] = CreateStoneAltar()
	tiles[7] = CreateTorchSconce("right")

	// Row 2: Storage (8-11)
	tiles[8] = CreateWeaponRack(false)
	tiles[9] = CreateWeaponRack(true)
	tiles[10] = CreateWoodenBarrel()
	tiles[11] = CreateMagicalBrazier()

	// Row 3: Doors and chests (12-15)
	tiles[12] = CreateDungeonDoor(false)
	tiles[13] = CreateDungeonDoor(true)
	tiles[14] = CreateTreasureChest(false)
	tiles[15] = CreateTreasureChest(true)

	return CreateAtlas(tiles, 4) // 4 columns, 4 rows
}

// Wall creation helpers
func CreateWallCorner(direction string) *image.RGBA {
	img := CreateSolidTile(ColorPalette.WallStone)
	borderColor := Lighten(ColorPalette.WallStone, 0.3)

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
	img := CreateSolidTile(ColorPalette.WallStone)
	borderColor := Lighten(ColorPalette.WallStone, 0.3)
	panelColor := ColorPalette.WallBrick

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
	img := CreateSolidTile(ColorPalette.WallStone)
	panelColor := ColorPalette.WallBrick

	// Add a centered panel
	for y := 4; y < 12; y++ {
		for x := 4; x < 12; x++ {
			img.Set(x, y, panelColor)
		}
	}

	return img
}

// Dungeon object creation helpers

// CreateStoneTable creates a stone table sprite
func CreateStoneTable(direction string) *image.RGBA {
	img := CreateBorderedTile(ColorPalette.TableWood, Darken(ColorPalette.TableWood, 0.5), 2)

	// Add some scratches/texture on the table
	scratchColor := Darken(ColorPalette.TableWood, 0.3)
	quarter := TileSize / 4
	half := TileSize / 2
	threeQuarter := 3 * TileSize / 4

	// Draw some scratches
	for i := quarter; i < threeQuarter; i++ {
		if i%3 == 0 {
			img.Set(i, half, scratchColor)
			img.Set(half, i, scratchColor)
		}
	}

	return img
}

// CreateHealingFountain creates a healing fountain sprite
func CreateHealingFountain() *image.RGBA {
	img := CreateBorderedTile(ColorPalette.TableWood, Darken(ColorPalette.TableWood, 0.5), 3)

	// Add glowing water in center
	waterColor := color.RGBA{100, 200, 255, 255}
	glowColor := color.RGBA{150, 230, 255, 255}
	quarter := TileSize / 4
	threeQuarter := 3 * TileSize / 4

	for y := quarter; y < threeQuarter; y++ {
		for x := quarter; x < threeQuarter; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, waterColor)
			} else {
				img.Set(x, y, glowColor)
			}
		}
	}

	return img
}

// CreateSkeletonRemains creates a skeleton sprite
func CreateSkeletonRemains() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	boneColor := ColorPalette.StoolWood
	darkBone := Darken(boneColor, 0.7)

	// Draw skull (circle at top)
	center := TileSize / 2
	skullRadius := TileSize / 6
	skullY := TileSize / 4

	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			dx := x - center
			dy := y - skullY
			if dx*dx+dy*dy <= skullRadius*skullRadius {
				img.Set(x, y, boneColor)
			}
		}
	}

	// Draw ribcage (lines below skull)
	ribStart := TileSize / 3
	ribEnd := 2 * TileSize / 3
	for y := ribStart + 4; y < ribEnd; y += 3 {
		for x := center - 4; x < center+4; x++ {
			img.Set(x, y, boneColor)
		}
	}

	// Eye sockets
	for dx := -2; dx <= 2; dx++ {
		img.Set(center-3+dx, skullY, darkBone)
		img.Set(center+3+dx, skullY, darkBone)
	}

	return img
}

// CreateWoodenStool creates a wooden stool sprite
func CreateWoodenStool() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Transparent background
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	woodColor := ColorPalette.Barrel
	darkWood := Darken(woodColor, 0.6)

	// Seat (top portion)
	quarter := TileSize / 4
	threeQuarter := 3 * TileSize / 4
	seatTop := TileSize / 3
	seatBottom := seatTop + TileSize/8

	for y := seatTop; y < seatBottom; y++ {
		for x := quarter; x < threeQuarter; x++ {
			img.Set(x, y, woodColor)
		}
	}

	// Legs
	legWidth := TileSize / 10
	for y := seatBottom; y < 3*TileSize/4; y++ {
		// Left leg
		for x := quarter + 2; x < quarter+2+legWidth; x++ {
			img.Set(x, y, darkWood)
		}
		// Right leg
		for x := threeQuarter - 2 - legWidth; x < threeQuarter-2; x++ {
			img.Set(x, y, darkWood)
		}
	}

	return img
}

// CreateAncientTome creates a glowing magical tome sprite
func CreateAncientTome() *image.RGBA {
	img := CreateBorderedTile(ColorPalette.Tome, Lighten(ColorPalette.Tome, 0.3), 2)

	// Add glowing runes
	runeColor := color.RGBA{180, 150, 255, 255}
	margin := TileSize / 6
	for y := margin; y < TileSize-margin; y++ {
		for x := margin; x < TileSize-margin; x++ {
			// Create a mystical pattern
			if (x+y)%4 == 0 || (x-y+TileSize)%4 == 0 {
				img.Set(x, y, runeColor)
			}
		}
	}

	return img
}

// CreateTorchSconce creates a wall torch sprite
func CreateTorchSconce(position string) *image.RGBA {
	img := CreateSolidTile(ColorPalette.WallBrick)
	flameColor := ColorPalette.TorchSconce
	brightFlame := Lighten(flameColor, 0.4)

	// Draw torch handle
	handleColor := Darken(ColorPalette.Barrel, 0.5)
	center := TileSize / 2
	handleTop := TileSize / 3
	handleBottom := 2 * TileSize / 3

	// Adjust position based on left/right
	offsetX := 0
	if position == "left" {
		offsetX = TileSize / 6
	} else {
		offsetX = -TileSize / 6
	}

	// Handle
	for y := handleTop; y < handleBottom; y++ {
		for dx := -1; dx <= 1; dx++ {
			img.Set(center+offsetX+dx, y, handleColor)
		}
	}

	// Flame
	flameY := handleTop - 2
	for dy := 0; dy < 6; dy++ {
		for dx := -3 + dy/2; dx <= 3-dy/2; dx++ {
			if dy < 3 {
				img.Set(center+offsetX+dx, flameY+dy, brightFlame)
			} else {
				img.Set(center+offsetX+dx, flameY+dy, flameColor)
			}
		}
	}

	return img
}

// CreateStoneAltar creates a stone altar sprite
func CreateStoneAltar() *image.RGBA {
	img := CreateBorderedTile(ColorPalette.WallStone, Darken(ColorPalette.WallStone, 0.4), 3)

	// Add mystical symbols
	symbolColor := color.RGBA{150, 100, 180, 255}
	center := TileSize / 2
	radius := TileSize / 4

	// Draw a simple pentagram-like pattern
	for angle := 0; angle < 360; angle += 72 {
		x := center + int(float64(radius)*0.7*float64(angle%2))
		y := center - radius/2 + (angle/72)*3
		if x >= 0 && x < TileSize && y >= 0 && y < TileSize {
			img.Set(x, y, symbolColor)
		}
	}

	// Center symbol
	for y := center - 2; y <= center+2; y++ {
		for x := center - 2; x <= center+2; x++ {
			if abs(x-center)+abs(y-center) <= 2 {
				img.Set(x, y, symbolColor)
			}
		}
	}

	return img
}

// CreateWeaponRack creates a weapon rack sprite
func CreateWeaponRack(empty bool) *image.RGBA {
	rackColor := ColorPalette.WeaponRack
	if empty {
		rackColor = Darken(rackColor, 0.7)
	}

	borderWidth := TileSize / 8
	img := CreateBorderedTile(rackColor, Darken(rackColor, 0.5), borderWidth)

	if !empty {
		// Add weapons on rack
		weaponColor := color.RGBA{180, 180, 190, 255}
		quarter := TileSize / 4
		threeQuarter := 3 * TileSize / 4

		// Sword shapes
		for y := quarter; y < threeQuarter; y++ {
			img.Set(quarter+2, y, weaponColor)
			img.Set(TileSize/2, y, weaponColor)
			img.Set(threeQuarter-2, y, weaponColor)
		}
	}

	return img
}

// CreateWoodenBarrel creates a wooden barrel sprite
func CreateWoodenBarrel() *image.RGBA {
	borderWidth := TileSize / 8
	img := CreateBorderedTile(ColorPalette.Barrel, Darken(ColorPalette.Barrel, 0.6), borderWidth)

	// Add barrel bands
	bandColor := color.RGBA{100, 80, 60, 255}
	bandPositions := []int{TileSize / 4, TileSize / 2, 3 * TileSize / 4}

	for _, y := range bandPositions {
		for x := borderWidth; x < TileSize-borderWidth; x++ {
			img.Set(x, y, bandColor)
			img.Set(x, y+1, bandColor)
		}
	}

	return img
}

// CreateMagicalBrazier creates a glowing brazier sprite
func CreateMagicalBrazier() *image.RGBA {
	// Base bowl
	bowlColor := color.RGBA{80, 70, 60, 255}
	img := CreateBorderedTile(bowlColor, Darken(bowlColor, 0.5), 2)

	// Add magical flames
	flameColors := []color.RGBA{
		{255, 200, 50, 255},  // Yellow
		{255, 150, 30, 255},  // Orange
		{255, 100, 20, 255},  // Red-orange
	}

	center := TileSize / 2
	flameHeight := TileSize / 2

	for y := TileSize/4; y < TileSize/4+flameHeight; y++ {
		// Flame narrows toward top
		progress := float64(y-TileSize/4) / float64(flameHeight)
		width := int(float64(TileSize/3) * (1.0 - progress*0.7))
		colorIdx := int(progress * float64(len(flameColors)-1))
		if colorIdx >= len(flameColors) {
			colorIdx = len(flameColors) - 1
		}

		for x := center - width; x <= center+width; x++ {
			if x >= 0 && x < TileSize {
				img.Set(x, y, flameColors[colorIdx])
			}
		}
	}

	return img
}

// CreateDungeonDoor creates a dungeon door sprite
func CreateDungeonDoor(open bool) *image.RGBA {
	doorColor := color.RGBA{90, 70, 50, 255} // Dark wood
	if open {
		doorColor = Darken(doorColor, 0.5)
	}

	img := CreateBorderedTile(doorColor, Darken(doorColor, 0.4), 2)

	if !open {
		// Add door details - iron bands
		bandColor := color.RGBA{70, 70, 80, 255}
		for x := 2; x < TileSize-2; x++ {
			img.Set(x, TileSize/4, bandColor)
			img.Set(x, TileSize/2, bandColor)
			img.Set(x, 3*TileSize/4, bandColor)
		}

		// Door handle
		handleColor := color.RGBA{150, 140, 100, 255}
		handleX := 3 * TileSize / 4
		handleY := TileSize / 2
		for dy := -2; dy <= 2; dy++ {
			for dx := -1; dx <= 1; dx++ {
				img.Set(handleX+dx, handleY+dy, handleColor)
			}
		}
	}

	return img
}

// CreateTreasureChest creates a treasure chest sprite
func CreateTreasureChest(open bool) *image.RGBA {
	chestColor := color.RGBA{120, 80, 40, 255} // Wood brown
	if open {
		chestColor = Darken(chestColor, 0.6)
	}

	img := CreateBorderedTile(chestColor, Darken(chestColor, 0.5), 2)

	// Gold trim
	goldColor := color.RGBA{255, 200, 50, 255}
	for x := 2; x < TileSize-2; x++ {
		img.Set(x, TileSize/3, goldColor)
		img.Set(x, 2*TileSize/3, goldColor)
	}

	if open {
		// Show gold inside
		for y := TileSize/3 + 2; y < 2*TileSize/3-2; y++ {
			for x := TileSize/4; x < 3*TileSize/4; x++ {
				if (x+y)%2 == 0 {
					img.Set(x, y, goldColor)
				}
			}
		}
	} else {
		// Lock
		lockColor := color.RGBA{180, 160, 80, 255}
		lockX := TileSize / 2
		lockY := TileSize / 2
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				img.Set(lockX+dx, lockY+dy, lockColor)
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
	fmt.Printf("✓ Generated %s/object_tiles.png (%dx%d pixels, 4x4 tiles @ %dpx)\n",
		assetsDir, 4*TileSize, 4*TileSize, TileSize)

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
