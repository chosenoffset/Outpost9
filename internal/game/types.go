package game

import (
	"chosenoffset.com/outpost9/internal/core/shadows"
)

// Player represents the player's physical state in the world.
type Player struct {
	Pos   shadows.Point
	Speed float64
	// Grid position for turn-based movement
	GridX, GridY int
}

// Camera tracks the viewport position for scrolling large levels.
type Camera struct {
	X, Y float64 // Camera position (top-left corner of viewport in world coords)
}

// Message represents an on-screen message that fades over time.
type Message struct {
	Text     string
	TimeLeft float64 // Seconds remaining
	MaxTime  float64 // Initial duration
}
