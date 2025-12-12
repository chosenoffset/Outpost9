package shadows

// Point represents a 2D point in space
type Point struct {
	X, Y float64
}

// Coord represents a tile coordinate
type Coord struct {
	X, Y int
}

// Segment represents a wall segment that can cast shadows
type Segment struct {
	A, B         Point
	TileX        int // Grid coordinates of the first tile this segment belongs to (for compatibility)
	TileY        int
	TilesCovered []Coord // All tiles this segment covers (for merged segments)
	EdgeType     string  // "top", "bottom", "left", "right"
}
