package shadows

// Point represents a 2D point in space
type Point struct {
	X, Y float64
}

// Segment represents a wall segment that can cast shadows
type Segment struct {
	A, B     Point
	TileX    int // Grid coordinates of the tile this segment belongs to
	TileY    int
	EdgeType string // "top", "bottom", "left", "right"
}
