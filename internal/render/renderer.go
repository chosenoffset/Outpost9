package render

import (
	"image"
	"image/color"
)

// Shader represents a compiled shader program.
type Shader interface {
	// Dispose releases shader resources.
	Dispose()
}

// ShaderCompiler compiles shaders from source code.
type ShaderCompiler interface {
	// CompileShader compiles shader source code into a Shader.
	CompileShader(src []byte) (Shader, error)
}

// DrawRectShaderOptions contains options for drawing with a shader.
type DrawRectShaderOptions struct {
	// Images are the source images for the shader (up to 4).
	Images [4]Image
	// Uniforms are the shader uniform values.
	Uniforms map[string]interface{}
}

// Renderer is the main rendering interface that abstracts the underlying
// graphics engine. This allows swapping rendering backends without changing
// game logic.
type Renderer interface {
	// Image operations
	NewImage(width, height int) Image

	// Vector operations (for drawing shapes)
	FillCircle(dst Image, x, y, radius float32, clr color.Color)
	StrokeCircle(dst Image, x, y, radius float32, strokeWidth float32, clr color.Color)

	// Text operations
	DrawText(dst Image, text string, x, y int, clr color.Color, scale float64)
	MeasureText(text string, scale float64) (width, height int)

	// Shader operations
	CompileShader(src []byte) (Shader, error)
}

// Image represents a renderable image surface that can be drawn to or drawn from.
// It abstracts the underlying image implementation.
type Image interface {
	// Properties
	Bounds() image.Rectangle
	Size() (width, height int)

	// Sub-image extraction
	SubImage(r image.Rectangle) Image

	// Fill operations
	Fill(clr color.Color)
	Clear()

	// Drawing operations
	DrawImage(src Image, opts *DrawImageOptions)
	DrawTriangles(vertices []Vertex, indices []uint16, img Image, opts *DrawTrianglesOptions)

	// Shader operations
	DrawRectShader(width, height int, shader Shader, opts *DrawRectShaderOptions)

	// Resource management
	Dispose()
}

// DrawImageOptions contains options for drawing an image.
type DrawImageOptions struct {
	GeoM GeoM
}

// GeoM represents a geometric transformation matrix.
type GeoM interface {
	// Translate shifts the image by (tx, ty).
	Translate(tx, ty float64)

	// Scale scales the image by (sx, sy).
	Scale(sx, sy float64)

	// Rotate rotates the image by the given angle in radians.
	Rotate(angle float64)

	// Reset resets the matrix to identity.
	Reset()
}

// NewGeoM creates a new geometric transformation matrix.
// This is implemented by the specific renderer backend.
var NewGeoM func() GeoM

// DrawTrianglesOptions contains options for drawing triangles.
type DrawTrianglesOptions struct {
	AntiAlias bool
}

// Vertex represents a vertex for triangle rendering.
type Vertex struct {
	DstX   float32
	DstY   float32
	SrcX   float32
	SrcY   float32
	ColorR float32
	ColorG float32
	ColorB float32
	ColorA float32
}

// InputManager handles input from the user (keyboard, mouse, etc).
type InputManager interface {
	IsKeyPressed(key Key) bool
	IsKeyJustPressed(key Key) bool
	GetCursorPosition() (x, y int)
	IsMouseButtonPressed(button MouseButton) bool
}

// Key represents a keyboard key.
type Key int

// Key constants for common keys
const (
	KeyW Key = iota
	KeyA
	KeyS
	KeyD
	KeyE // Interact key
	KeyL // Light toggle key
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeySpace
	KeyEscape
)

// MouseButton represents a mouse button.
type MouseButton int

// Mouse button constants
const (
	MouseButtonLeft MouseButton = iota
	MouseButtonRight
	MouseButtonMiddle
)

// ResourceLoader handles loading resources like images from disk.
type ResourceLoader interface {
	LoadImage(path string) (Image, error)
}

// Game represents the game interface that the engine will call.
// This is typically implemented by the main game struct.
type Game interface {
	// Update updates the game logic. It is called every tick (typically 60 times per second).
	Update() error

	// Draw draws the game screen. It is called every frame.
	Draw(screen Image)

	// Layout accepts the outside size (e.g., window size) and returns the logical screen size.
	// The logical screen size is used for rendering and input coordinates.
	Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int)
}

// Engine represents the game engine that manages the game loop and window.
type Engine interface {
	// SetWindowSize sets the window size in pixels.
	SetWindowSize(width, height int)

	// SetWindowTitle sets the window title.
	SetWindowTitle(title string)

	// SetWindowResizable enables or disables window resizing.
	SetWindowResizable(resizable bool)

	// RunGame runs the game loop with the provided game.
	// This is a blocking call that runs until the game ends.
	RunGame(game Game) error
}
