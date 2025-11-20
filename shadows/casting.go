package shadows

import (
	"math"

	"chosenoffset.com/outpost9/maploader"
)

// CastShadow projects a shadow from a wall segment based on viewer position
// Returns a polygon representing the shadow volume
func CastShadow(viewerPos Point, seg Segment, maxDistance float64, tileSize int, gameMap *maploader.Map, isCornerShadow bool) []Point {
	// Get the shadow start edge based on player position relative to the tile
	shadowStart := GetShadowStartEdge(seg, tileSize, gameMap, viewerPos, isCornerShadow)

	// Calculate direction vectors from viewer to shadow start points
	dirA := Point{
		X: shadowStart.A.X - viewerPos.X,
		Y: shadowStart.A.Y - viewerPos.Y,
	}
	dirB := Point{
		X: shadowStart.B.X - viewerPos.X,
		Y: shadowStart.B.Y - viewerPos.Y,
	}

	lenA := math.Sqrt(dirA.X*dirA.X + dirA.Y*dirA.Y)
	lenB := math.Sqrt(dirB.X*dirB.X + dirB.Y*dirB.Y)

	if lenA < 0.001 || lenB < 0.001 {
		return nil
	}

	// Normalize direction vectors
	dirA.X /= lenA
	dirA.Y /= lenA
	dirB.X /= lenB
	dirB.Y /= lenB

	// Extend shadow rays far
	extendDist := maxDistance * 2

	extendedA := Point{
		X: shadowStart.A.X + dirA.X*extendDist,
		Y: shadowStart.A.Y + dirA.Y*extendDist,
	}
	extendedB := Point{
		X: shadowStart.B.X + dirB.X*extendDist,
		Y: shadowStart.B.Y + dirB.Y*extendDist,
	}

	return []Point{shadowStart.A, shadowStart.B, extendedB, extendedA}
}

// GetShadowStartEdge determines where the shadow should start based on the viewer position
// and whether this is a main shadow or corner shadow
// For merged segments, this works with the segment's actual geometry
func GetShadowStartEdge(seg Segment, tileSize int, gameMap *maploader.Map, viewerPos Point, isCornerShadow bool) Segment {
	adjusted := seg

	if isCornerShadow {
		// Corner shadow: start from the EXPOSED edge itself
		// The segment already represents the exposed edge, so use it directly
		adjusted.A = seg.A
		adjusted.B = seg.B
	} else {
		// Main shadow: start from the OPPOSITE side
		// For horizontal/vertical segments, offset perpendicular to the edge
		offset := float64(tileSize)

		switch seg.EdgeType {
		case "top":
			// Top edge is exposed - shadow starts from bottom (offset down)
			adjusted.A = Point{X: seg.A.X, Y: seg.A.Y + offset}
			adjusted.B = Point{X: seg.B.X, Y: seg.B.Y + offset}
		case "bottom":
			// Bottom edge is exposed - shadow starts from top (offset up)
			adjusted.A = Point{X: seg.A.X, Y: seg.A.Y - offset}
			adjusted.B = Point{X: seg.B.X, Y: seg.B.Y - offset}
		case "left":
			// Left edge is exposed - shadow starts from right (offset right)
			adjusted.A = Point{X: seg.A.X + offset, Y: seg.A.Y}
			adjusted.B = Point{X: seg.B.X + offset, Y: seg.B.Y}
		case "right":
			// Right edge is exposed - shadow starts from left (offset left)
			adjusted.A = Point{X: seg.A.X - offset, Y: seg.A.Y}
			adjusted.B = Point{X: seg.B.X - offset, Y: seg.B.Y}
		}
	}

	return adjusted
}

// GetDefaultShadowOffset calculates a default shadow offset for a segment
// This shifts the shadow start point away from the edge
func GetDefaultShadowOffset(seg Segment, tileSize int) Segment {
	adjusted := seg
	offset := float64(tileSize) / 2.0

	switch seg.EdgeType {
	case "top":
		adjusted.A.Y += offset
		adjusted.B.Y += offset
	case "bottom":
		adjusted.A.Y -= offset
		adjusted.B.Y -= offset
	case "left":
		adjusted.A.X += offset
		adjusted.B.X += offset
	case "right":
		adjusted.A.X -= offset
		adjusted.B.X -= offset
	}

	return adjusted
}

// CalculateShadowOffset calculates shadow offset based on tile visual bounds properties
// This is used to align shadows with the visual representation of walls
func CalculateShadowOffset(seg Segment, tileSize int, gameMap *maploader.Map) float64 {
	// Get the tile definition to check for visual_bounds
	tileDef, err := gameMap.GetTileDefAt(seg.TileX, seg.TileY)
	if err != nil {
		return 2.0 // Default small offset
	}

	// Try to get visual_bounds from properties
	visualBounds, ok := tileDef.GetTileProperty("visual_bounds")
	if !ok {
		// No visual bounds defined, use small default offset
		return 2.0
	}

	// Parse visual_bounds (should be a map with top, bottom, left, right)
	bounds, ok := visualBounds.(map[string]interface{})
	if !ok {
		return 2.0
	}

	// Based on the edge type, calculate offset to the FAR side of the visual pixels
	// (the side away from the viewer, where the shadow should begin)
	switch seg.EdgeType {
	case "top":
		// For top edge: shadow starts at the BOTTOM of the visual wall
		// visual_bounds.bottom tells us where the wall pixels end
		if bottomVal, ok := bounds["bottom"].(float64); ok {
			// Offset from the top edge to the bottom of the visual wall
			return bottomVal + 1.0
		}
	case "bottom":
		// For bottom edge: shadow starts at the TOP of the visual wall
		// visual_bounds.top tells us where the wall pixels start
		if topVal, ok := bounds["top"].(float64); ok {
			// Calculate distance from bottom edge up to where wall starts
			distanceFromBottom := float64(tileSize) - topVal
			return distanceFromBottom + 1.0
		}
	case "left":
		// For left edge: shadow starts at the RIGHT of the visual wall
		if rightVal, ok := bounds["right"].(float64); ok {
			return rightVal + 1.0
		}
	case "right":
		// For right edge: shadow starts at the LEFT of the visual wall
		if leftVal, ok := bounds["left"].(float64); ok {
			distanceFromRight := float64(tileSize) - leftVal
			return distanceFromRight + 1.0
		}
	}

	// Default: small offset
	return 2.0
}
