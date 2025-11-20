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
