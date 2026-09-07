package hud

import (
	"bytes"
	gimage "image"
	"image/color"
	"time"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

// Chrome palette. Map colors stay in pkg/render; these are panel-only. Only
// the colors and theme helpers a builder actually calls live here today:
// remaining spec §3 palette entries (colorQueued) are added alongside the
// first widget that needs them, so the unused linter never has to flag
// scaffolding with no caller.
var (
	colorPanel       = color.RGBA{R: 25, G: 35, B: 42, A: 238}
	colorPanelEdge   = color.RGBA{R: 58, G: 76, B: 82, A: 210}
	colorRow         = color.RGBA{R: 20, G: 29, B: 35, A: 255}
	colorRowOpen     = color.RGBA{R: 28, G: 40, B: 48, A: 255}
	colorTitle       = color.RGBA{R: 239, G: 220, B: 178, A: 255}
	colorText        = color.RGBA{R: 223, G: 229, B: 225, A: 255}
	colorDim         = color.RGBA{R: 159, G: 177, B: 174, A: 255}
	colorGold        = color.RGBA{R: 245, G: 202, B: 92, A: 255}
	colorGoldDeep    = color.RGBA{R: 203, G: 172, B: 104, A: 255}
	colorGreen       = color.RGBA{R: 121, G: 195, B: 137, A: 255}
	colorCyan        = color.RGBA{R: 87, G: 211, B: 211, A: 255}
	colorAmber       = color.RGBA{R: 237, G: 176, B: 84, A: 255}
	colorRed         = color.RGBA{R: 247, G: 137, B: 119, A: 255}
	colorInterbreed  = color.RGBA{R: 186, G: 148, B: 232, A: 255}
	colorButtonIdle  = color.RGBA{R: 35, G: 51, B: 58, A: 255}
	colorButtonHover = color.RGBA{R: 48, G: 68, B: 78, A: 255}
	colorButtonDown  = color.RGBA{R: 24, G: 36, B: 42, A: 255}
	colorDisabled    = color.RGBA{R: 92, G: 106, B: 109, A: 255}
	colorBlack       = color.RGBA{R: 17, G: 17, B: 17, A: 255}
	colorDrawer      = color.RGBA{R: 18, G: 28, B: 34, A: 240}
	colorCelebrate   = color.RGBA{R: 45, G: 39, B: 24, A: 255}
	colorGuide       = color.RGBA{R: 42, G: 36, B: 22, A: 255}
)

// theme owns the font source and the current presentation scale. Every size
// and inset the panel uses goes through px() or face(), so a viewport change
// only needs a rebuild with a new scale.
type theme struct {
	source  *text.GoTextFaceSource
	scale   float64
	faces   map[float64]*text.Face
	solids  map[color.RGBA]*image.NineSlice
	borders map[borderKey]*image.NineSlice
}

// borderKey identifies one bordered nine-slice. The width is already in
// render pixels, so a scale change produces different keys as well as
// clearing the cache.
type borderKey struct {
	body, border color.RGBA
	widthPx      int
}

func newTheme() *theme {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	theme := &theme{source: source, scale: 1}
	theme.resetCaches()
	return theme
}

func (t *theme) resetCaches() {
	t.faces = map[float64]*text.Face{}
	t.solids = map[color.RGBA]*image.NineSlice{}
	t.borders = map[borderKey]*image.NineSlice{}
}

func (t *theme) setScale(scale float64) {
	if scale <= 0 {
		scale = 1
	}
	if scale != t.scale {
		t.scale = scale
		t.resetCaches()
	}
}

// px converts a DIP length to render pixels, rounding to the nearest pixel.
func (t *theme) px(dip float64) int { return int(dip*t.scale + 0.5) }

// face returns a cached ebitenui font face for a DIP size at the current scale.
func (t *theme) face(sizeDIP float64) *text.Face {
	if face, ok := t.faces[sizeDIP]; ok {
		return face
	}
	var face text.Face = &text.GoTextFace{Source: t.source, Size: sizeDIP * t.scale}
	t.faces[sizeDIP] = &face
	return &face
}

func (t *theme) insets(top, left, right, bottom float64) *widget.Insets {
	return &widget.Insets{Top: t.px(top), Left: t.px(left), Right: t.px(right), Bottom: t.px(bottom)}
}

// solid and bordered memoize their nine-slices: a rebuild asks for the same
// handful of backgrounds from every builder, and each miss otherwise
// allocates a fresh image. The caches are cleared by setScale.
func (t *theme) solid(c color.RGBA) *image.NineSlice {
	if cached, ok := t.solids[c]; ok {
		return cached
	}
	slice := image.NewNineSliceColor(c)
	t.solids[c] = slice
	return slice
}

func (t *theme) bordered(body, border color.RGBA, widthPx int) *image.NineSlice {
	key := borderKey{body: body, border: border, widthPx: max(1, widthPx)}
	if cached, ok := t.borders[key]; ok {
		return cached
	}
	slice := image.NewBorderedNineSliceColor(body, border, key.widthPx)
	t.borders[key] = slice
	return slice
}

// buttonImages is the standard clickable look; border color varies by role.
func (t *theme) buttonImages(border color.RGBA) *widget.ButtonImage {
	width := t.px(1)
	return &widget.ButtonImage{
		Idle:     t.bordered(colorButtonIdle, border, width),
		Hover:    t.bordered(colorButtonHover, border, width),
		Pressed:  t.bordered(colorButtonDown, border, width),
		Disabled: t.bordered(colorRow, colorDisabled, width),
	}
}

func (t *theme) buttonText(idle color.RGBA) *widget.ButtonTextColor {
	return &widget.ButtonTextColor{Idle: idle, Hover: idle, Pressed: idle, Disabled: colorDisabled}
}

// button builds a labelled button that emits handler on click.
func (t *theme) button(label string, sizeDIP float64, border, textColor color.RGBA, handler func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Image(t.buttonImages(border)),
		widget.ButtonOpts.Text(label, t.face(sizeDIP), t.buttonText(textColor)),
		widget.ButtonOpts.TextPadding(t.insets(3, 8, 8, 3)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { handler() }),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.CursorHovered("pointer")),
	)
}

// label builds static text.
func (t *theme) label(value string, sizeDIP float64, textColor color.Color) *widget.Text {
	return widget.NewText(widget.TextOpts.Text(value, t.face(sizeDIP), textColor))
}

// rightLabel is t.label with its text right-aligned within the widget's own
// box via widget.TextOpts.Position — t.label always left-aligns, so a column
// of values that must line up on a shared right edge (rather than a shared
// left edge) needs this instead.
func (t *theme) rightLabel(value string, sizeDIP float64, textColor color.Color, opts ...widget.WidgetOpt) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(value, t.face(sizeDIP), textColor),
		widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(opts...),
	)
}

// wrapped builds text that wraps at a DIP width.
func (t *theme) wrapped(value string, sizeDIP float64, textColor color.Color, widthDIP float64) *widget.Text {
	return widget.NewText(widget.TextOpts.Text(value, t.face(sizeDIP), textColor), widget.TextOpts.MaxWidth(float64(t.px(widthDIP))))
}

// tooltipWidth bounds a tooltip's wrapped text, and tooltipDelay is how long
// the cursor must rest on a control before it explains itself. The pkg/ui copy
// runs to a full sentence, far wider than the panel unwrapped, and the delay
// is shorter than ebitenui's 800 ms default because these appear on dead
// controls, where the player is already asking why.
const (
	tooltipWidth = 240.0
	tooltipDelay = 350 * time.Millisecond
)

// tooltip is a hover explanation for one control. It opens below the control
// and right-aligned to it: the Move row's buttons sit against the right edge
// of the 1280 DIP presentation, so ebitenui's default cursor-following tooltip
// would run off screen.
func (t *theme) tooltip(message string) (*widget.ToolTip, *widget.Text) {
	label := t.wrapped(message, 9, colorText, tooltipWidth)
	content := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(t.bordered(colorPanel, colorGold, t.px(1))),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout(widget.AnchorLayoutOpts.Padding(t.insets(6, 8, 8, 6)))),
	)
	content.AddChild(label)
	return widget.NewToolTip(
		widget.ToolTipOpts.Content(content),
		widget.ToolTipOpts.Position(widget.TOOLTIP_POS_WIDGET),
		widget.ToolTipOpts.Delay(tooltipDelay),
		widget.ToolTipOpts.Offset(gimage.Point{Y: t.px(4)}),
		widget.ToolTipOpts.AnchorOriginHorizontal(widget.TOOLTIP_ANCHOR_END),
		widget.ToolTipOpts.ContentOriginHorizontal(widget.TOOLTIP_ANCHOR_END),
		widget.ToolTipOpts.AnchorOriginVertical(widget.TOOLTIP_ANCHOR_END),
		widget.ToolTipOpts.ContentOriginVertical(widget.TOOLTIP_ANCHOR_START),
	), label
}

// column is a vertical row layout container with DIP spacing and padding.
func (t *theme) column(spacingDIP float64, padding *widget.Insets, background *image.NineSlice, opts ...widget.WidgetOpt) *widget.Container {
	containerOpts := []widget.ContainerOpt{
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(t.px(spacingDIP)),
			widget.RowLayoutOpts.Padding(padding),
		)),
		widget.ContainerOpts.WidgetOpts(opts...),
	}
	if background != nil {
		containerOpts = append(containerOpts, widget.ContainerOpts.BackgroundImage(background))
	}
	return widget.NewContainer(containerOpts...)
}

// rowOf is a horizontal row layout container.
func (t *theme) rowOf(spacingDIP float64, opts ...widget.WidgetOpt) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(t.px(spacingDIP)),
		)),
		widget.ContainerOpts.WidgetOpts(opts...),
	)
}

// stretch is the row-layout data that makes a child fill the cross axis.
func stretch() widget.WidgetOpt {
	return widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})
}

func (t *theme) checkboxImages() *widget.CheckboxImage {
	makeImage := func(checked bool, ink color.RGBA) *image.NineSlice {
		size := t.px(18)
		img := ebiten.NewImage(size, size)
		img.Fill(colorButtonIdle)
		vector.StrokeRect(img, 1, 1, float32(size-2), float32(size-2), 1, ink, false)
		if checked {
			vector.StrokeLine(img, float32(size)*0.2, float32(size)*0.5, float32(size)*0.43, float32(size)*0.73, float32(t.px(2)), ink, true)
			vector.StrokeLine(img, float32(size)*0.43, float32(size)*0.73, float32(size)*0.8, float32(size)*0.25, float32(t.px(2)), ink, true)
		}
		return image.NewFixedNineSlice(img)
	}
	return &widget.CheckboxImage{Unchecked: makeImage(false, colorText), Checked: makeImage(true, colorGold), UncheckedDisabled: makeImage(false, colorDim), CheckedDisabled: makeImage(true, colorDim)}
}
