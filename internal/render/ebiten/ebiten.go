package ebiten

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"chosenoffset.com/outpost9/internal/render"
)

// EbitenRenderer implements the Renderer interface using Ebiten.
type EbitenRenderer struct{}

// init sets up the global functions for the ebiten render.
func init() {
	render.NewGeoM = func() render.GeoM {
		return NewGeoM()
	}
}

// NewRenderer creates a new Ebiten-based render.
func NewRenderer() render.Renderer {
	return &EbitenRenderer{}
}

// NewImage creates a new image with the given dimensions.
func (r *EbitenRenderer) NewImage(width, height int) render.Image {
	return &EbitenImage{img: ebiten.NewImage(width, height)}
}

// FillCircle draws a filled circle on the destination image.
func (r *EbitenRenderer) FillCircle(dst render.Image, x, y, radius float32, clr color.Color) {
	ebitenImg := dst.(*EbitenImage).img
	vector.DrawFilledCircle(ebitenImg, x, y, radius, clr, true)
}

// StrokeCircle draws a circle outline on the destination image.
func (r *EbitenRenderer) StrokeCircle(dst render.Image, x, y, radius float32, strokeWidth float32, clr color.Color) {
	ebitenImg := dst.(*EbitenImage).img
	vector.StrokeCircle(ebitenImg, x, y, radius, strokeWidth, clr, true)
}

// DrawText draws text on the destination image using the default font.
// Note: Color parameter is currently ignored, text is always white.
// Scale parameter adjusts the effective size (implemented via character spacing approximation).
func (r *EbitenRenderer) DrawText(dst render.Image, str string, x, y int, clr color.Color, scale float64) {
	ebitenImg := dst.(*EbitenImage).img

	// ebitenutil.DebugPrintAt uses a fixed font size, so we approximate scaling
	// by adjusting position. For now, we just use the base size.
	// TODO: Implement proper scaled text rendering with a font library
	ebitenutil.DebugPrintAt(ebitenImg, str, x, y)
}

// MeasureText measures the width and height of text with the given scale.
// This is an approximation based on the debug font's character size.
func (r *EbitenRenderer) MeasureText(str string, scale float64) (width, height int) {
	// Debug font is approximately 6x13 pixels per character
	charWidth := 6.0
	charHeight := 13.0
	return int(float64(len(str)) * charWidth * scale), int(charHeight * scale)
}

// CompileShader compiles shader source code into a Shader.
func (r *EbitenRenderer) CompileShader(src []byte) (render.Shader, error) {
	shader, err := ebiten.NewShader(src)
	if err != nil {
		return nil, err
	}
	return &EbitenShader{shader: shader}, nil
}

// EbitenShader wraps an ebiten.Shader to implement the render.Shader interface.
type EbitenShader struct {
	shader *ebiten.Shader
}

// Dispose releases shader resources.
func (s *EbitenShader) Dispose() {
	if s.shader != nil {
		s.shader.Dispose()
	}
}

// GetEbitenShader returns the underlying ebiten.Shader.
func (s *EbitenShader) GetEbitenShader() *ebiten.Shader {
	return s.shader
}

// EbitenImage wraps an ebiten.Image to implement the render.Image interface.
type EbitenImage struct {
	img *ebiten.Image
}

// Bounds returns the bounds of the image.
func (i *EbitenImage) Bounds() image.Rectangle {
	return i.img.Bounds()
}

// Size returns the width and height of the image.
func (i *EbitenImage) Size() (width, height int) {
	return i.img.Bounds().Dx(), i.img.Bounds().Dy()
}

// SubImage returns a sub-image of the image.
func (i *EbitenImage) SubImage(r image.Rectangle) render.Image {
	return &EbitenImage{img: i.img.SubImage(r).(*ebiten.Image)}
}

// Fill fills the entire image with the given color.
func (i *EbitenImage) Fill(clr color.Color) {
	i.img.Fill(clr)
}

// Clear clears the image to transparent.
func (i *EbitenImage) Clear() {
	i.img.Clear()
}

// Dispose releases the image resources.
func (i *EbitenImage) Dispose() {
	if i.img != nil {
		i.img.Dispose()
	}
}

// DrawImage draws the source image onto this image.
func (i *EbitenImage) DrawImage(src render.Image, opts *render.DrawImageOptions) {
	srcImg := src.(*EbitenImage).img

	if opts == nil {
		i.img.DrawImage(srcImg, nil)
		return
	}

	ebitenOpts := &ebiten.DrawImageOptions{}
	if opts.GeoM != nil {
		ebitenGeoM := opts.GeoM.(*EbitenGeoM)
		ebitenOpts.GeoM = ebitenGeoM.geoM
	}

	i.img.DrawImage(srcImg, ebitenOpts)
}

// DrawTriangles draws triangles on this image using the provided vertices.
func (i *EbitenImage) DrawTriangles(vertices []render.Vertex, indices []uint16, img render.Image, opts *render.DrawTrianglesOptions) {
	// Convert render.Vertex to ebiten.Vertex
	ebitenVertices := make([]ebiten.Vertex, len(vertices))
	for j, v := range vertices {
		ebitenVertices[j] = ebiten.Vertex{
			DstX:   v.DstX,
			DstY:   v.DstY,
			SrcX:   v.SrcX,
			SrcY:   v.SrcY,
			ColorR: v.ColorR,
			ColorG: v.ColorG,
			ColorB: v.ColorB,
			ColorA: v.ColorA,
		}
	}

	ebitenImg := img.(*EbitenImage).img

	if opts == nil {
		i.img.DrawTriangles(ebitenVertices, indices, ebitenImg, nil)
		return
	}

	ebitenOpts := &ebiten.DrawTrianglesOptions{
		AntiAlias: opts.AntiAlias,
	}

	i.img.DrawTriangles(ebitenVertices, indices, ebitenImg, ebitenOpts)
}

// DrawRectShader draws a rectangle using the specified shader.
func (i *EbitenImage) DrawRectShader(width, height int, shader render.Shader, opts *render.DrawRectShaderOptions) {
	ebitenShader := shader.(*EbitenShader).shader

	ebitenOpts := &ebiten.DrawRectShaderOptions{}
	if opts != nil {
		// Convert images
		for idx, img := range opts.Images {
			if img != nil {
				ebitenOpts.Images[idx] = img.(*EbitenImage).img
			}
		}
		// Copy uniforms
		ebitenOpts.Uniforms = opts.Uniforms
	}

	i.img.DrawRectShader(width, height, ebitenShader, ebitenOpts)
}

// GetEbitenImage returns the underlying ebiten.Image.
// This is useful for interop with ebiten-specific code.
func (i *EbitenImage) GetEbitenImage() *ebiten.Image {
	return i.img
}

// WrapEbitenImage wraps an existing ebiten.Image as a render.Image.
func WrapEbitenImage(img *ebiten.Image) render.Image {
	return &EbitenImage{img: img}
}

// EbitenGeoM wraps ebiten's GeoM to implement the render.GeoM interface.
type EbitenGeoM struct {
	geoM ebiten.GeoM
}

// NewGeoM creates a new geometric transformation matrix.
func NewGeoM() render.GeoM {
	return &EbitenGeoM{geoM: ebiten.GeoM{}}
}

// Translate shifts the image by (tx, ty).
func (g *EbitenGeoM) Translate(tx, ty float64) {
	g.geoM.Translate(tx, ty)
}

// Scale scales the image by (sx, sy).
func (g *EbitenGeoM) Scale(sx, sy float64) {
	g.geoM.Scale(sx, sy)
}

// Rotate rotates the image by the given angle in radians.
func (g *EbitenGeoM) Rotate(angle float64) {
	g.geoM.Rotate(angle)
}

// Reset resets the matrix to identity.
func (g *EbitenGeoM) Reset() {
	g.geoM.Reset()
}

// EbitenInputManager implements the InputManager interface using Ebiten.
type EbitenInputManager struct{}

// NewInputManager creates a new Ebiten-based input manager.
func NewInputManager() render.InputManager {
	return &EbitenInputManager{}
}

// IsKeyPressed returns whether the specified key is currently pressed.
func (m *EbitenInputManager) IsKeyPressed(key render.Key) bool {
	return ebiten.IsKeyPressed(keyToEbitenKey(key))
}

// IsKeyJustPressed returns whether the specified key was just pressed this frame.
func (m *EbitenInputManager) IsKeyJustPressed(key render.Key) bool {
	return inpututil.IsKeyJustPressed(keyToEbitenKey(key))
}

// GetCursorPosition returns the current cursor position.
func (m *EbitenInputManager) GetCursorPosition() (x, y int) {
	return ebiten.CursorPosition()
}

// IsMouseButtonPressed returns whether the specified mouse button is currently pressed.
func (m *EbitenInputManager) IsMouseButtonPressed(button render.MouseButton) bool {
	return ebiten.IsMouseButtonPressed(mouseButtonToEbiten(button))
}

// keyToEbitenKey converts a render.Key to an ebiten.Key.
func keyToEbitenKey(key render.Key) ebiten.Key {
	switch key {
	case render.KeyW:
		return ebiten.KeyW
	case render.KeyA:
		return ebiten.KeyA
	case render.KeyS:
		return ebiten.KeyS
	case render.KeyD:
		return ebiten.KeyD
	case render.KeyE:
		return ebiten.KeyE
	case render.KeyL:
		return ebiten.KeyL
	case render.KeyUp:
		return ebiten.KeyArrowUp
	case render.KeyDown:
		return ebiten.KeyArrowDown
	case render.KeyLeft:
		return ebiten.KeyArrowLeft
	case render.KeyRight:
		return ebiten.KeyArrowRight
	case render.KeySpace:
		return ebiten.KeySpace
	case render.KeyEscape:
		return ebiten.KeyEscape
	default:
		return 0
	}
}

// mouseButtonToEbiten converts a render.MouseButton to an ebiten.MouseButton.
func mouseButtonToEbiten(button render.MouseButton) ebiten.MouseButton {
	switch button {
	case render.MouseButtonLeft:
		return ebiten.MouseButtonLeft
	case render.MouseButtonRight:
		return ebiten.MouseButtonRight
	case render.MouseButtonMiddle:
		return ebiten.MouseButtonMiddle
	default:
		return ebiten.MouseButtonLeft
	}
}

// EbitenResourceLoader implements the ResourceLoader interface using Ebiten.
type EbitenResourceLoader struct{}

// NewResourceLoader creates a new Ebiten-based resource loader.
func NewResourceLoader() render.ResourceLoader {
	return &EbitenResourceLoader{}
}

// LoadImage loads an image from the specified file path.
func (l *EbitenResourceLoader) LoadImage(path string) (render.Image, error) {
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		return nil, err
	}
	return &EbitenImage{img: img}, nil
}

// EbitenEngine implements the Engine interface using Ebiten.
type EbitenEngine struct{}

// NewEngine creates a new Ebiten-based game engine.
func NewEngine() render.Engine {
	return &EbitenEngine{}
}

// SetWindowSize sets the window size in pixels.
func (e *EbitenEngine) SetWindowSize(width, height int) {
	ebiten.SetWindowSize(width, height)
}

// SetWindowTitle sets the window title.
func (e *EbitenEngine) SetWindowTitle(title string) {
	ebiten.SetWindowTitle(title)
}

// SetWindowResizable enables or disables window resizing.
func (e *EbitenEngine) SetWindowResizable(resizable bool) {
	if resizable {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	} else {
		ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	}
}

// RunGame runs the game loop with the provided game.
func (e *EbitenEngine) RunGame(game render.Game) error {
	return ebiten.RunGame(&gameAdapter{game: game})
}

// gameAdapter adapts a render.Game to ebiten.Game interface.
type gameAdapter struct {
	game render.Game
}

// Update implements ebiten.Game.
func (a *gameAdapter) Update() error {
	return a.game.Update()
}

// Draw implements ebiten.Game.
func (a *gameAdapter) Draw(screen *ebiten.Image) {
	a.game.Draw(&EbitenImage{img: screen})
}

// Layout implements ebiten.Game.
func (a *gameAdapter) Layout(outsideWidth, outsideHeight int) (int, int) {
	return a.game.Layout(outsideWidth, outsideHeight)
}
