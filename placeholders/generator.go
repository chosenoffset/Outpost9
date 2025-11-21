package placeholders

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// TileSize is the standard size for placeholder tiles
const TileSize = 32

// ColorPalette defines colors for different tile types
var ColorPalette = struct {
	// Base tiles
	FloorMetal1  color.RGBA
	FloorMetal2  color.RGBA
	FloorGrating color.RGBA
	WallMetal    color.RGBA
	WallPanel    color.RGBA

	// Objects
	DeskWood   color.RGBA
	ChairMetal color.RGBA
	Terminal   color.RGBA
	Console    color.RGBA
	Locker     color.RGBA
	Crate      color.RGBA
	Generator  color.RGBA

	// Entities
	Player     color.RGBA
	EnemyBasic color.RGBA
	EnemyElite color.RGBA
	Item       color.RGBA
	Weapon     color.RGBA

	// UI
	Border     color.RGBA
	Background color.RGBA
}{
	// Base tiles - various grays and metals
	FloorMetal1:  color.RGBA{80, 85, 95, 255},   // Medium gray-blue
	FloorMetal2:  color.RGBA{70, 75, 85, 255},   // Darker gray-blue
	FloorGrating: color.RGBA{60, 65, 75, 255},   // Dark gray with slight blue
	WallMetal:    color.RGBA{140, 145, 155, 255}, // Much lighter gray-blue for walls
	WallPanel:    color.RGBA{120, 130, 140, 255}, // Steel blue-gray

	// Objects - distinct colors for easy identification
	DeskWood:   color.RGBA{139, 90, 60, 255},   // Brown
	ChairMetal: color.RGBA{180, 180, 190, 255}, // Light gray
	Terminal:   color.RGBA{0, 150, 200, 255},   // Cyan/blue
	Console:    color.RGBA{50, 120, 180, 255},  // Blue
	Locker:     color.RGBA{140, 140, 150, 255}, // Gray
	Crate:      color.RGBA{160, 120, 80, 255},  // Tan/brown
	Generator:  color.RGBA{200, 150, 0, 255},   // Gold/yellow

	// Entities - bright, easily visible colors
	Player:     color.RGBA{0, 255, 100, 255}, // Bright green
	EnemyBasic: color.RGBA{255, 50, 50, 255}, // Bright red
	EnemyElite: color.RGBA{200, 0, 200, 255}, // Magenta
	Item:       color.RGBA{255, 215, 0, 255}, // Gold
	Weapon:     color.RGBA{255, 140, 0, 255}, // Orange

	// UI
	Border:     color.RGBA{200, 200, 200, 255}, // Light gray
	Background: color.RGBA{40, 40, 45, 255},    // Very dark gray
}

// CreateSolidTile creates a simple solid-colored tile
func CreateSolidTile(col color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))
	draw.Draw(img, img.Bounds(), &image.Uniform{col}, image.Point{}, draw.Src)
	return img
}

// CreateBorderedTile creates a tile with a border
func CreateBorderedTile(fillColor, borderColor color.RGBA, borderWidth int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{fillColor}, image.Point{}, draw.Src)

	// Draw borders
	for i := 0; i < borderWidth; i++ {
		// Top and bottom borders
		for x := 0; x < TileSize; x++ {
			img.Set(x, i, borderColor)
			img.Set(x, TileSize-1-i, borderColor)
		}
		// Left and right borders
		for y := 0; y < TileSize; y++ {
			img.Set(i, y, borderColor)
			img.Set(TileSize-1-i, y, borderColor)
		}
	}

	return img
}

// CreatePatternedTile creates a tile with a simple pattern
func CreatePatternedTile(baseColor, patternColor color.RGBA, pattern string) *image.RGBA {
	img := CreateSolidTile(baseColor)

	switch pattern {
	case "grid":
		// Draw a grid pattern
		for i := 0; i < TileSize; i += 4 {
			for x := 0; x < TileSize; x++ {
				img.Set(x, i, patternColor)
				img.Set(i, x, patternColor)
			}
		}
	case "dots":
		// Draw dots (scaled for tile size)
		quarter := TileSize / 4
		threeQuarter := 3 * TileSize / 4
		dots := []image.Point{{quarter, quarter}, {threeQuarter, quarter}, {quarter, threeQuarter}, {threeQuarter, threeQuarter}}
		for _, p := range dots {
			// Draw 2x2 dots for 32x32 tiles
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					img.Set(p.X+dx, p.Y+dy, patternColor)
				}
			}
		}
	case "cross":
		// Draw a cross
		mid := TileSize / 2
		for i := 2; i < TileSize-2; i++ {
			img.Set(mid, i, patternColor)
			img.Set(i, mid, patternColor)
		}
	case "diagonal":
		// Draw diagonal lines
		for i := 0; i < TileSize; i++ {
			img.Set(i, i, patternColor)
			img.Set(i, TileSize-1-i, patternColor)
		}
	}

	return img
}

// CreateCircle creates a circular sprite (for entities)
func CreateCircle(fillColor, outlineColor color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, TileSize, TileSize))

	// Make background transparent
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	center := TileSize / 2
	radius := TileSize/2 - 2

	// Draw filled circle
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			dx := x - center
			dy := y - center
			distSq := dx*dx + dy*dy

			if distSq <= radius*radius {
				img.Set(x, y, fillColor)
			} else if distSq <= (radius+1)*(radius+1) {
				img.Set(x, y, outlineColor)
			}
		}
	}

	return img
}

// CreateAtlas creates a sprite atlas from multiple tiles
func CreateAtlas(tiles []*image.RGBA, columns int) *image.RGBA {
	tileCount := len(tiles)
	rows := (tileCount + columns - 1) / columns

	width := columns * TileSize
	height := rows * TileSize

	atlas := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with transparent background
	draw.Draw(atlas, atlas.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	// Copy each tile into the atlas
	for i, tile := range tiles {
		if tile == nil {
			continue
		}

		col := i % columns
		row := i / columns

		x := col * TileSize
		y := row * TileSize

		destRect := image.Rect(x, y, x+TileSize, y+TileSize)
		draw.Draw(atlas, destRect, tile, image.Point{}, draw.Src)
	}

	return atlas
}

// SavePNG saves an image to a PNG file
func SavePNG(img image.Image, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// Darken returns a darker version of a color
func Darken(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * factor),
		G: uint8(float64(c.G) * factor),
		B: uint8(float64(c.B) * factor),
		A: c.A,
	}
}

// Lighten returns a lighter version of a color
func Lighten(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) + (255-float64(c.R))*factor),
		G: uint8(float64(c.G) + (255-float64(c.G))*factor),
		B: uint8(float64(c.B) + (255-float64(c.B))*factor),
		A: c.A,
	}
}
