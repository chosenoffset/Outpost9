package shadows

import (
	"math"
	"sort"
)

// ComputeVisibilityPolygon calculates what the viewer can see from their position
// Returns a polygon representing the visible area (everything outside is in shadow)
func ComputeVisibilityPolygon(viewerPos Point, segments []Segment, maxDistance float64) []Point {
	// Collect all unique endpoints (vertices) from wall segments
	vertices := collectVertices(segments)

	// For each vertex, we'll cast rays at angles slightly before and after it
	// This handles edge cases where the ray passes through a vertex
	var rays []ray
	for _, vertex := range vertices {
		angle := math.Atan2(vertex.Y-viewerPos.Y, vertex.X-viewerPos.X)

		// Add three rays: one at the exact angle, one slightly before, one slightly after
		epsilon := 0.0001
		rays = append(rays,
			ray{angle: angle - epsilon},
			ray{angle: angle},
			ray{angle: angle + epsilon},
		)
	}

	// Sort rays by angle
	sort.Slice(rays, func(i, j int) bool {
		return rays[i].angle < rays[j].angle
	})

	// For each ray, find the closest intersection point
	var visiblePoints []Point
	for _, r := range rays {
		// Ray direction
		dx := math.Cos(r.angle)
		dy := math.Sin(r.angle)

		// Find closest intersection
		closestDist := maxDistance
		closestPoint := Point{
			X: viewerPos.X + dx*maxDistance,
			Y: viewerPos.Y + dy*maxDistance,
		}

		// Check intersection with all segments
		for _, seg := range segments {
			if intersect, dist, point := raySegmentIntersection(viewerPos, dx, dy, seg); intersect {
				if dist < closestDist {
					closestDist = dist
					closestPoint = point
				}
			}
		}

		visiblePoints = append(visiblePoints, closestPoint)
	}

	return visiblePoints
}

// ray represents a ray cast from the viewer at a specific angle
type ray struct {
	angle float64
}

// collectVertices extracts all unique endpoint vertices from segments
func collectVertices(segments []Segment) []Point {
	vertexMap := make(map[Point]bool)

	for _, seg := range segments {
		vertexMap[seg.A] = true
		vertexMap[seg.B] = true
	}

	vertices := make([]Point, 0, len(vertexMap))
	for vertex := range vertexMap {
		vertices = append(vertices, vertex)
	}

	return vertices
}

// raySegmentIntersection checks if a ray intersects a line segment
// Returns: (intersects bool, distance float64, intersection point Point)
func raySegmentIntersection(origin Point, dx, dy float64, seg Segment) (bool, float64, Point) {
	// Ray: P = origin + t * (dx, dy) for t >= 0
	// Segment: Q = seg.A + u * (seg.B - seg.A) for 0 <= u <= 1

	// Segment direction
	segDX := seg.B.X - seg.A.X
	segDY := seg.B.Y - seg.A.Y

	// Solve: origin + t*(dx,dy) = seg.A + u*(segDX,segDY)
	// This is a 2x2 linear system

	denominator := dx*segDY - dy*segDX
	if math.Abs(denominator) < 1e-10 {
		// Ray and segment are parallel
		return false, 0, Point{}
	}

	// Calculate u and t
	diffX := seg.A.X - origin.X
	diffY := seg.A.Y - origin.Y

	u := (dx*diffY - dy*diffX) / denominator
	t := (segDX*diffY - segDY*diffX) / denominator

	// Check if intersection is within segment and in ray direction
	if u >= 0 && u <= 1 && t >= 0 {
		intersectionPoint := Point{
			X: origin.X + t*dx,
			Y: origin.Y + t*dy,
		}
		return true, t, intersectionPoint
	}

	return false, 0, Point{}
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
