package shadows

import "chosenoffset.com/outpost9/maploader"

// CreateWallSegmentsFromMap generates wall segments from map data
// Each segment represents an exposed edge of a wall tile that can cast shadows
func CreateWallSegmentsFromMap(gameMap *maploader.Map) []Segment {
	var segments []Segment

	mapWidth := gameMap.Data.Width
	mapHeight := gameMap.Data.Height
	tileSize := float64(gameMap.Data.TileSize)

	// First pass: create individual tile edge segments
	type tempSegment struct {
		seg      Segment
		canMerge bool
	}
	var tempSegments []tempSegment

	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			if gameMap.BlocksSight(x, y) {
				tileX := float64(x) * tileSize
				tileY := float64(y) * tileSize

				// Get the tile name to check its type
				tileName, _ := gameMap.GetTileAt(x, y)

				// Check if there's a wall tile above this one
				hasWallAbove := y > 0 && gameMap.BlocksSight(x, y-1)

				// Check each edge and create segment if it borders non-blocking tile
				// Top edge - skip if this is a "bottom" tile with a wall above it
				shouldCreateTopEdge := (y == 0 || !gameMap.BlocksSight(x, y-1))

				// Skip top edge for bottom tiles (nwb, nbx, neb, swb, sbx, seb, etc) that have walls above
				if shouldCreateTopEdge && !(hasWallAbove && (tileName == "nwb" || tileName == "nbx" || tileName == "neb" ||
					tileName == "swb" || tileName == "sbx" || tileName == "seb")) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX, tileY},
							B:        Point{tileX + tileSize, tileY},
							TileX:    x,
							TileY:    y,
							EdgeType: "top",
						},
						canMerge: true,
					})
				}
				// Right edge
				if x == mapWidth-1 || !gameMap.BlocksSight(x+1, y) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX + tileSize, tileY},
							B:        Point{tileX + tileSize, tileY + tileSize},
							TileX:    x,
							TileY:    y,
							EdgeType: "right",
						},
						canMerge: true,
					})
				}
				// Bottom edge
				if y == mapHeight-1 || !gameMap.BlocksSight(x, y+1) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX + tileSize, tileY + tileSize},
							B:        Point{tileX, tileY + tileSize},
							TileX:    x,
							TileY:    y,
							EdgeType: "bottom",
						},
						canMerge: true,
					})
				}
				// Left edge
				if x == 0 || !gameMap.BlocksSight(x-1, y) {
					tempSegments = append(tempSegments, tempSegment{
						seg: Segment{
							A:        Point{tileX, tileY + tileSize},
							B:        Point{tileX, tileY},
							TileX:    x,
							TileY:    y,
							EdgeType: "left",
						},
						canMerge: true,
					})
				}
			}
		}
	}

	// Second pass: merge adjacent colinear segments
	merged := make([]bool, len(tempSegments))

	for i := 0; i < len(tempSegments); i++ {
		if merged[i] || !tempSegments[i].canMerge {
			continue
		}

		current := tempSegments[i].seg

		// Try to find adjacent segments to merge with
		for j := i + 1; j < len(tempSegments); j++ {
			if merged[j] || !tempSegments[j].canMerge {
				continue
			}

			other := tempSegments[j].seg

			// Check if segments are adjacent and colinear
			// DON'T merge - keep segments separate for proper shadow alignment
			// Each tile needs its own shadow based on its visual bounds
			_ = other
		}

		segments = append(segments, current)
		merged[i] = true
	}

	// Add any unmerged segments
	for i, temp := range tempSegments {
		if !merged[i] {
			segments = append(segments, temp.seg)
		}
	}

	return segments
}
