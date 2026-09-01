package render

import "math"

const (
	PresentationWidth    = 1280.0
	PresentationHeight   = 720.0
	MaxRenderScale       = 2.0
	MinViewportWidthDIP  = PresentationWidth
	MinViewportHeightDIP = PresentationHeight
)

// Viewport is presentation-only state. It converts the window's logical DIPs
// into the bounded physical render surface and never enters a frame or save.
type Viewport struct {
	LogicalWidthDIP  float64
	LogicalHeightDIP float64
	RenderWidthPx    int
	RenderHeightPx   int
	RenderScale      float64
	ScaleX           float64
	ScaleY           float64
	ViewportRevision uint64
}

// NextViewport derives the next immutable viewport. Transient minimization
// retains the last valid value; before any valid layout it returns a safe 1×1
// surface without claiming a revision.
func NextViewport(previous Viewport, outsideWidthDIP, outsideHeightDIP, rawScale float64) Viewport {
	if !positiveFinite(outsideWidthDIP) || !positiveFinite(outsideHeightDIP) {
		if previous.RenderWidthPx > 0 && previous.RenderHeightPx > 0 {
			return previous
		}
		return Viewport{RenderWidthPx: 1, RenderHeightPx: 1, RenderScale: 1, ScaleX: 1, ScaleY: 1}
	}
	if !positiveFinite(rawScale) {
		rawScale = 1
	}
	renderScale := min(max(rawScale, 1), MaxRenderScale)
	renderWidth := max(1, int(math.Ceil(outsideWidthDIP*renderScale)))
	renderHeight := max(1, int(math.Ceil(outsideHeightDIP*renderScale)))
	next := Viewport{
		LogicalWidthDIP: outsideWidthDIP, LogicalHeightDIP: outsideHeightDIP,
		RenderWidthPx: renderWidth, RenderHeightPx: renderHeight, RenderScale: renderScale,
		ScaleX: float64(renderWidth) / outsideWidthDIP, ScaleY: float64(renderHeight) / outsideHeightDIP,
		ViewportRevision: previous.ViewportRevision,
	}
	if sameViewportGeometry(previous, next) {
		return previous
	}
	next.ViewportRevision++
	return next
}

func (viewport Viewport) SupportsGameplay() bool {
	return viewport.LogicalWidthDIP >= MinViewportWidthDIP && viewport.LogicalHeightDIP >= MinViewportHeightDIP
}

func (viewport Viewport) DIPToRender(xDIP, yDIP float64) (float64, float64) {
	return xDIP * viewport.ScaleX, yDIP * viewport.ScaleY
}

func (viewport Viewport) RenderToDIP(xPx, yPx float64) (float64, float64) {
	if viewport.ScaleX <= 0 || viewport.ScaleY <= 0 {
		return 0, 0
	}
	return xPx / viewport.ScaleX, yPx / viewport.ScaleY
}

// PresentationTransform aspect-fits the fixed presentation surface inside
// the physical render target. Letterboxing is presentation-only and mouse
// input uses the exact inverse transform.
type PresentationTransform struct {
	Scale   float64
	OffsetX float64
	OffsetY float64
}

func FitPresentation(renderWidth, renderHeight int) PresentationTransform {
	if renderWidth <= 0 || renderHeight <= 0 {
		return PresentationTransform{Scale: 1}
	}
	scale := min(float64(renderWidth)/PresentationWidth, float64(renderHeight)/PresentationHeight)
	return PresentationTransform{
		Scale:   scale,
		OffsetX: (float64(renderWidth) - PresentationWidth*scale) / 2,
		OffsetY: (float64(renderHeight) - PresentationHeight*scale) / 2,
	}
}

func (transform PresentationTransform) LogicalToRender(x, y float64) (float64, float64) {
	return transform.OffsetX + x*transform.Scale, transform.OffsetY + y*transform.Scale
}

func (transform PresentationTransform) RenderToLogical(x, y float64) (float64, float64, bool) {
	if transform.Scale <= 0 {
		return 0, 0, false
	}
	logicalX := (x - transform.OffsetX) / transform.Scale
	logicalY := (y - transform.OffsetY) / transform.Scale
	return logicalX, logicalY, logicalX >= 0 && logicalX < PresentationWidth && logicalY >= 0 && logicalY < PresentationHeight
}

func sameViewportGeometry(first, second Viewport) bool {
	return first.LogicalWidthDIP == second.LogicalWidthDIP && first.LogicalHeightDIP == second.LogicalHeightDIP &&
		first.RenderWidthPx == second.RenderWidthPx && first.RenderHeightPx == second.RenderHeightPx &&
		first.RenderScale == second.RenderScale && first.ScaleX == second.ScaleX && first.ScaleY == second.ScaleY
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
