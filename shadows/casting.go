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

	// Build list of angles to cast rays at - ONLY toward wall vertices
	// This creates straight-edged, wall-aligned visibility (no curves)
	var angles []float64

	// Add angles for each wall vertex (with epsilon offsets for edge cases)
	epsilon := 0.0001
	for _, vertex := range vertices {
		angle := math.Atan2(vertex.Y-viewerPos.Y, vertex.X-viewerPos.X)
		angles = append(angles,
			angle-epsilon,
			angle,
			angle+epsilon,
		)
	}

	// Remove duplicate angles and sort
	angleMap := make(map[float64]bool)
	var uniqueAngles []float64
	for _, angle := range angles {
		// Normalize angle to [0, 2π)
		normalized := math.Mod(angle, 2.0*math.Pi)
		if normalized < 0 {
			normalized += 2.0 * math.Pi
		}
		if !angleMap[normalized] {
			angleMap[normalized] = true
			uniqueAngles = append(uniqueAngles, normalized)
		}
	}

	sort.Float64s(uniqueAngles)

	// For each angle, cast a ray and find the closest intersection
	var visiblePoints []Point
	for _, angle := range uniqueAngles {
		// Ray direction
		dx := math.Cos(angle)
		dy := math.Sin(angle)

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
