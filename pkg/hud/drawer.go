package hud

import (
	"fmt"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const drawerEventLines = 2

func noteBody(note render.FieldNote) string {
	parts := []string{"[color=9fb1ae]SUMMARY[/color] · " + note.Introduction}
	if note.Context != "" {
		parts = append(parts, "[color=9fb1ae]HISTORICAL CONTEXT[/color] · "+note.Context)
	}
	if note.GameEffect != "" {
		parts = append(parts, "[color=9fb1ae]GAME ABSTRACTION[/color] · "+note.GameEffect)
	}
	if note.Hint != "" {
		parts = append(parts, "[color=9fb1ae]HINT[/color] · "+note.Hint)
	}
	if note.References != "" {
		parts = append(parts, "[color=9fb1ae]SOURCES[/color] · "+note.References)
	}
	return strings.Join(parts, "\n")
}

func eventLine(event gameapi.Event) string {
	return fmt.Sprintf("T%d · %s · %s", event.Turn, event.Kind, event.Summary)
}

// truncateToWidth returns value unchanged when it already fits maxWidthPx
// against face, and otherwise the longest rune prefix (plus an ellipsis)
// that does. The hidden drawer bar spans the full map width and its event
// text shares that width with the notes control, so the cap has to reflect
// actual glyph widths rather than a fixed rune count that was only ever
// correct for one control layout.
func truncateToWidth(value string, maxWidthPx float64, face *text.Face) string {
	if maxWidthPx <= 0 {
		return ""
	}
	if width, _ := text.Measure(value, *face, 0); width <= maxWidthPx {
		return value
	}
	runes := []rune(value)
	for n := len(runes) - 1; n > 0; n-- {
		candidate := string(runes[:n]) + "…"
		if width, _ := text.Measure(candidate, *face, 0); width <= maxWidthPx {
			return candidate
		}
	}
	return "…"
}

// newestEvents returns up to limit events, newest first.
func newestEvents(events []gameapi.Event, limit int) []gameapi.Event {
	result := make([]gameapi.Event, 0, limit)
	for index := len(events) - 1; index >= 0 && len(result) < limit; index-- {
		result = append(result, events[index])
	}
	return result
}

// drawerHeight is the drawer's DIP height for a mode; hidden is the
// full-width bar rather than nothing, so the camera and tile picking still
// leave room for it (spec §6).
func drawerHeight(mode NotesMode) float64 {
	switch mode {
	case NotesCompact:
		return drawerCompactH
	case NotesExpanded:
		return drawerExpandedH
	default:
		return drawerHiddenH
	}
}

// buildDrawer places the Field Notes drawer over the bottom of the map
// (spec §4) with its edge tab, or the full-width hidden bar in its place
// when hidden.
func (p *Panel) buildDrawer(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	events := newestEvents(state.Frame.Events, drawerEventLines)
	if state.NotesMode == NotesHidden {
		return p.buildHiddenDrawerBar(state, events)
	}
	height := drawerHeight(state.NotesMode)
	root := widget.NewContainer(widget.ContainerOpts.Layout(fixedLayout{}),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(mapLeft, mapBottom-height-drawerTabH, mapRight-mapLeft, height+drawerTabH))))
	tabRow := t.rowOf(4, widget.WidgetOpts.LayoutData(p.rect(mapRight-2*drawerTabW-8, mapBottom-height-drawerTabH, 2*drawerTabW, drawerTabH)))
	moreLabel, moreMode := "▲ more", NotesExpanded
	if state.NotesMode == NotesExpanded {
		moreLabel, moreMode = "▼ less", NotesCompact
	}
	more := t.button(moreLabel, 9, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSetNotesMode, Notes: moreMode}) })
	p.handles.drawerMore = more
	tab := t.button("hide notes · F", 9, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSetNotesMode, Notes: NotesHidden}) })
	p.handles.drawerTab = tab
	tabRow.AddChild(more, tab)
	root.AddChild(tabRow)

	background := colorDrawer
	heading := "FIELD NOTES"
	headingColor := colorGoldDeep
	if state.Note.Topic != "" {
		heading += " · " + state.Note.Topic
	}
	if state.Note.Celebration {
		background, headingColor = colorCelebrate, colorGold
		heading = "BREAKTHROUGH · " + state.Note.Topic
	}
	body := t.column(4, t.insets(6, 12, 12, 6), t.solid(background),
		widget.WidgetOpts.LayoutData(p.rect(mapLeft, mapBottom-height, mapRight-mapLeft, height)))
	body.AddChild(t.label(heading, 10, headingColor))
	noteHeight := height - 24 - 14*float64(min(len(events), drawerEventLines)+1) - 12
	area := widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(mapRight-mapLeft-24), t.px(noteHeight)))),
		widget.TextAreaOpts.FontFace(t.face(9)),
		widget.TextAreaOpts.FontColor(colorText),
		widget.TextAreaOpts.ProcessBBCode(true),
		widget.TextAreaOpts.Text(noteBody(state.Note)),
		widget.TextAreaOpts.ScrollContainerImage(&widget.ScrollContainerImage{Idle: t.solid(background), Mask: t.solid(background)}),
	)
	p.handles.notesArea = area
	body.AddChild(area)
	body.AddChild(t.label("RECENT EVENTS", 8, headingColor))
	if len(events) == 0 {
		body.AddChild(t.label("No campaign events yet.", 8.5, colorDim))
	}
	for _, event := range events {
		kind := event.Kind
		line := widget.NewButton(
			widget.ButtonOpts.Image(&widget.ButtonImage{Idle: t.solid(background), Hover: t.solid(colorRowOpen), Pressed: t.solid(colorRow)}),
			widget.ButtonOpts.Text(eventLine(event), t.face(8.5), t.buttonText(colorDim)),
			widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentFocusEvent, Event: kind}) }),
			widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
		)
		p.handles.events = append(p.handles.events, line)
		body.AddChild(line)
	}
	root.AddChild(body)
	return root
}

// hiddenBarControlLabel is the always-present right-hand control on the
// hidden drawer bar. Unlike the old edge tab it never carries the event or
// the BREAKTHROUGH marker — both now belong to the event button on the left
// — so its width is fixed and known up front, which is what
// hiddenBarEventBudgetPx below subtracts from the full map width.
const hiddenBarControlLabel = "▲ notes · F"

// hiddenBarPaddingDIP and hiddenBarSpacingDIP size the bar's own RowLayout,
// so the event-text width budget can subtract them precisely instead of
// guessing at a generous constant.
const (
	hiddenBarPaddingDIP = 8.0
	hiddenBarSpacingDIP = 8.0
)

// buildHiddenDrawerBar is the hidden-mode replacement for the small
// right-aligned edge tab: a full-width single-line bar sharing the drawer's
// left edge, so it can show a whole event line rather than a 42-rune
// fragment. The newest event sits on the left as its own button (clicking
// anywhere on the bar reopens the drawer, so the event text has to be
// clickable too, not a label); the notes control stays on the right.
func (p *Panel) buildHiddenDrawerBar(state State, events []gameapi.Event) widget.PreferredSizeLocateableWidget {
	t := p.theme
	background, accent := colorDrawer, colorGoldDeep
	if state.Note.Celebration {
		background, accent = colorCelebrate, colorGold
	}
	open := func() { p.emit(Intent{Kind: IntentSetNotesMode, Notes: NotesCompact}) }

	bar := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(t.px(hiddenBarSpacingDIP)),
			widget.RowLayoutOpts.Padding(t.insets(2, hiddenBarPaddingDIP, hiddenBarPaddingDIP, 2)),
		)),
		widget.ContainerOpts.BackgroundImage(t.solid(background)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(
			p.rect(mapLeft, mapBottom-drawerHiddenH, mapRight-mapLeft, drawerHiddenH))),
	)

	label := ""
	if state.Note.Celebration {
		label = "BREAKTHROUGH"
	}
	if len(events) > 0 {
		truncated := truncateToWidth(eventLine(events[0]), p.hiddenBarEventBudgetPx(t), t.face(9))
		if label != "" {
			label += " · " + truncated
		} else {
			label = truncated
		}
	}
	event := t.button(label, 9, accent, accent, func() { open() })
	event.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	p.handles.drawerBarEvent = event

	control := t.button(hiddenBarControlLabel, 9, colorGoldDeep, colorGoldDeep, func() { open() })
	control.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionEnd}
	p.handles.drawerTab = control

	bar.AddChild(event, control)
	p.handles.drawerBar = bar
	return bar
}

// hiddenBarEventBudgetPx is the render-pixel width left for the hidden bar's
// event text once the fixed-label control, the row's own spacing, and its
// padding are subtracted from the full map width. Computing it rather than
// guessing means a font, control-label, or scale change cannot silently
// start clipping the control off the right edge.
func (p *Panel) hiddenBarEventBudgetPx(t *theme) float64 {
	controlWidth, _ := text.Measure(hiddenBarControlLabel, *t.face(9), 0)
	padding := t.insets(3, 8, 8, 3) // matches theme.button's TextPadding
	controlWidth += float64(padding.Left + padding.Right + 2*t.px(1))
	total := float64(t.px(mapRight - mapLeft))
	reserve := controlWidth + float64(t.px(hiddenBarSpacingDIP)) + float64(2*t.px(hiddenBarPaddingDIP))
	return total - reserve
}
