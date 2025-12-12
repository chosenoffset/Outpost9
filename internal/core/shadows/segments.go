package shadows

import (
	"chosenoffset.com/outpost9/internal/world/atlas"
	"chosenoffset.com/outpost9/internal/world/maploader"
)

// CreateWallSegmentsFromMap generates wall segments from map data using a holistic contour-based approach
// Instead of creating per-tile segments, this extracts the perimeter of contiguous sight-blocking regions
// and merges colinear segments for cleaner, more efficient shadows
func CreateWallSegmentsFromMap(gameMap *maploader.Map) []Segment {
	mapWidth := gameMap.Data.Width
	mapHeight := gameMap.Data.Height
	tileSize := float64(gameMap.Data.TileSize)

	// Step 1: Find all contiguous regions of sight-blocking tiles
	regions := findContiguousRegions(gameMap, mapWidth, mapHeight)

	// Step 2: Extract perimeter segments for each region
	var allSegments []Segment
	for _, region := range regions {
		perimeter := extractPerimeterSegments(region, gameMap, tileSize)
		allSegments = append(allSegments, perimeter...)
	}

	// Step 3: Merge colinear segments to create longer wall segments
	mergedSegments := mergeColinearSegments(allSegments)

	return mergedSegments
}

// findContiguousRegions identifies all connected regions of sight-blocking tiles
func findContiguousRegions(gameMap *maploader.Map, width, height int) [][]Coord {
	visited := make(map[Coord]bool)
	var regions [][]Coord

	// Scan the entire map for unvisited sight-blocking tiles
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			coord := Coord{X: x, Y: y}
			if visited[coord] || !gameMap.BlocksSight(x, y) {
				continue
			}

			// Found an unvisited sight-blocking tile - flood fill to find the entire region
			region := floodFill(gameMap, coord, width, height, visited)
			if len(region) > 0 {
				regions = append(regions, region)
			}
		}
	}

	return regions
}

// floodFill performs BFS to find all connected sight-blocking tiles
func floodFill(gameMap *maploader.Map, start Coord, width, height int, visited map[Coord]bool) []Coord {
	var region []Coord
	queue := []Coord{start}
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		region = append(region, current)

		// Check all 4-connected neighbors (no diagonals)
		neighbors := []Coord{
			{X: current.X, Y: current.Y - 1}, // North
			{X: current.X + 1, Y: current.Y}, // East
			{X: current.X, Y: current.Y + 1}, // South
			{X: current.X - 1, Y: current.Y}, // West
		}

		for _, neighbor := range neighbors {
			// Check bounds
			if neighbor.X < 0 || neighbor.X >= width || neighbor.Y < 0 || neighbor.Y >= height {
				continue
			}

			// Check if already visited or doesn't block sight
			if visited[neighbor] || !gameMap.BlocksSight(neighbor.X, neighbor.Y) {
				continue
			}

			visited[neighbor] = true
			queue = append(queue, neighbor)
		}
	}

	return region
}

// extractPerimeterSegments finds all exposed edges of a region
func extractPerimeterSegments(region []Coord, gameMap *maploader.Map, tileSize float64) []Segment {
	var segments []Segment

	// Convert region to a set for fast lookup
	regionSet := make(map[Coord]bool)
	for _, coord := range region {
		regionSet[coord] = true
	}

	// For each tile in the region, check which edges are exposed
	for _, coord := range region {
		x, y := coord.X, coord.Y
		tileX := float64(x) * tileSize
		tileY := float64(y) * tileSize

		// Get the tile definition to check visual_bounds
		tileDef, err := gameMap.GetTileDefAt(x, y)
		tileName := ""
		if err == nil {
			tileName = tileDef.Name
		}

		// Get visual bounds (where actual pixels are within the tile)
		bounds := getVisualBounds(tileDef, tileSize)

		// Calculate actual pixel boundaries within this tile
		// These are the positions of the actual wall pixels, not the tile grid
		pixelLeft := tileX + bounds.left
		pixelRight := tileX + bounds.right
		pixelTop := tileY + bounds.top
		pixelBottom := tileY + bounds.bottom

		// Check each edge - create segment if it borders a non-blocking area
		// Segments are positioned at the visual bounds, not tile edges

		// Top edge (top of the wall pixels)
		northCoord := Coord{X: x, Y: y - 1}
		hasWallAbove := regionSet[northCoord]
		shouldCreateTopEdge := !regionSet[northCoord]

		// Skip top edge for bottom tiles that have walls above
		if shouldCreateTopEdge && !(hasWallAbove && (tileName == "nwb" || tileName == "nbx" || tileName == "neb" ||
			tileName == "swb" || tileName == "sbx" || tileName == "seb")) {
			segments = append(segments, Segment{
				A:            Point{pixelLeft, pixelTop},
				B:            Point{pixelRight, pixelTop},
				TileX:        x,
				TileY:        y,
				TilesCovered: []Coord{{X: x, Y: y}},
				EdgeType:     "top",
			})
		}

		// Right edge (right side of wall pixels)
		if !regionSet[Coord{X: x + 1, Y: y}] {
			segments = append(segments, Segment{
				A:            Point{pixelRight, pixelTop},
				B:            Point{pixelRight, pixelBottom},
				TileX:        x,
				TileY:        y,
				TilesCovered: []Coord{{X: x, Y: y}},
				EdgeType:     "right",
			})
		}

		// Bottom edge (bottom of wall pixels)
		if !regionSet[Coord{X: x, Y: y + 1}] {
			segments = append(segments, Segment{
				A:            Point{pixelRight, pixelBottom},
				B:            Point{pixelLeft, pixelBottom},
				TileX:        x,
				TileY:        y,
				TilesCovered: []Coord{{X: x, Y: y}},
				EdgeType:     "bottom",
			})
		}

		// Left edge (left side of wall pixels)
		if !regionSet[Coord{X: x - 1, Y: y}] {
			segments = append(segments, Segment{
				A:            Point{pixelLeft, pixelBottom},
				B:            Point{pixelLeft, pixelTop},
				TileX:        x,
				TileY:        y,
				TilesCovered: []Coord{{X: x, Y: y}},
				EdgeType:     "left",
			})
		}
	}

	return segments
}

// visualBounds represents the actual pixel boundaries within a tile
type visualBounds struct {
	top, bottom, left, right float64
}

// getVisualBounds extracts visual bounds from tile definition, or returns defaults
func getVisualBounds(tileDef *atlas.TileDefinition, tileSize float64) visualBounds {
	// Default: entire tile is the wall
	bounds := visualBounds{
		top:    0,
		bottom: tileSize,
		left:   0,
		right:  tileSize,
	}

	if tileDef == nil {
		return bounds
	}

	// Try to get visual_bounds property
	visualBoundsProp, ok := tileDef.GetTileProperty("visual_bounds")
	if !ok {
		return bounds
	}

	// Parse visual_bounds map
	boundsMap, ok := visualBoundsProp.(map[string]interface{})
	if !ok {
		return bounds
	}

	// Extract each boundary value
	if top, ok := boundsMap["top"].(float64); ok {
		bounds.top = top
	}
	if bottom, ok := boundsMap["bottom"].(float64); ok {
		bounds.bottom = bottom
	}
	if left, ok := boundsMap["left"].(float64); ok {
		bounds.left = left
	}
	if right, ok := boundsMap["right"].(float64); ok {
		bounds.right = right
	}

	return bounds
}

// mergeColinearSegments combines adjacent parallel segments into longer segments
func mergeColinearSegments(segments []Segment) []Segment {
	if len(segments) == 0 {
		return segments
	}

	merged := make([]bool, len(segments))
	var result []Segment

	for i := 0; i < len(segments); i++ {
		if merged[i] {
			continue
		}

		current := segments[i]
		merged[i] = true

		// Try to extend this segment by merging with adjacent colinear segments
		extended := true
		for extended {
			extended = false

			for j := 0; j < len(segments); j++ {
				if merged[j] || i == j {
					continue
				}

				other := segments[j]

				// Check if segments can be merged
				if canMergeSegments(current, other) {
					current = mergeSegments(current, other)
					merged[j] = true
					extended = true
					break
				}
			}
		}

		result = append(result, current)
	}

	return result
}

// canMergeSegments checks if two segments are adjacent and colinear
func canMergeSegments(seg1, seg2 Segment) bool {
	// Must be the same edge type
	if seg1.EdgeType != seg2.EdgeType {
		return false
	}

	epsilon := 0.001

	switch seg1.EdgeType {
	case "top", "bottom":
		// Horizontal segments - must have same Y coordinate
		if abs(seg1.A.Y-seg2.A.Y) > epsilon {
			return false
		}
		// Must be adjacent (one ends where other begins)
		return abs(seg1.B.X-seg2.A.X) < epsilon || abs(seg1.A.X-seg2.B.X) < epsilon

	case "left", "right":
		// Vertical segments - must have same X coordinate
		if abs(seg1.A.X-seg2.A.X) > epsilon {
			return false
		}
		// Must be adjacent (one ends where other begins)
		return abs(seg1.B.Y-seg2.A.Y) < epsilon || abs(seg1.A.Y-seg2.B.Y) < epsilon
	}

	return false
}

// mergeSegments combines two adjacent colinear segments into one
func mergeSegments(seg1, seg2 Segment) Segment {
	result := seg1

	switch seg1.EdgeType {
	case "top", "bottom":
		// Horizontal - extend X range
		minX := min(seg1.A.X, seg1.B.X, seg2.A.X, seg2.B.X)
		maxX := max(seg1.A.X, seg1.B.X, seg2.A.X, seg2.B.X)
		result.A.X = minX
		result.B.X = maxX

	case "left", "right":
		// Vertical - extend Y range
		minY := min(seg1.A.Y, seg1.B.Y, seg2.A.Y, seg2.B.Y)
		maxY := max(seg1.A.Y, seg1.B.Y, seg2.A.Y, seg2.B.Y)
		result.A.Y = minY
		result.B.Y = maxY
	}

	// Merge the tiles covered by both segments
	result.TilesCovered = append(result.TilesCovered, seg2.TilesCovered...)

	return result
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(vals ...float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	minVal := vals[0]
	for _, v := range vals[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func max(vals ...float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	maxVal := vals[0]
	for _, v := range vals[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}
