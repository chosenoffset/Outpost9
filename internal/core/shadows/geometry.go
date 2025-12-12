package shadows

import "math"

// IsFacingPoint checks if a segment is facing towards a given point
// Uses cross product to determine if the point is on the "front" side of the segment
func IsFacingPoint(seg Segment, point Point) bool {
	dx1 := seg.B.X - seg.A.X
	dy1 := seg.B.Y - seg.A.Y
	dx2 := point.X - seg.A.X
	dy2 := point.Y - seg.A.Y

	cross := dx1*dy2 - dy1*dx2
	// Return true if point is on the positive side (segment is facing point)
	return cross > 0
}

// PointInPolygon tests if a point is inside a polygon using ray casting algorithm
func PointInPolygon(point Point, polygon []Point) bool {
	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i].X, polygon[i].Y
		xj, yj := polygon[j].X, polygon[j].Y

		if ((yi > point.Y) != (yj > point.Y)) &&
			(point.X < (xj-xi)*(point.Y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}

	return inside
}

// Distance calculates the Euclidean distance between two points
func Distance(a, b Point) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	return math.Sqrt(dx*dx + dy*dy)
}
