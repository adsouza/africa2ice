package hud

import (
	"fmt"
	"image"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// drawerHiddenEventLines is how many events the hidden bar can show: it is
// one line tall, so exactly one. The compact and expanded drawers no longer
// use a fixed count — their events sit in a column whose height decides how
// many fit (see Panel.drawerEventRows).
const drawerHiddenEventLines = 1

// Drawer body geometry. The body is split into two side-by-side columns:
// the note on the left and the event log on the right, rather than the
// events stacked underneath. drawerEventsColW is sized from the widest
// event line the domain can produce (the Campanian-eruption summary
// measures ~301 DIP at 8.5), so a typical line never truncates; anything
// longer still goes through truncateToWidth against the column's own
// budget, so the width is safe by construction rather than by hope.
const (
	drawerEventsColW = 320.0
	drawerColGutter  = 12.0
	drawerBodyPadH   = 12.0
	drawerBodyPadV   = 6.0
	drawerColSpacing = 4.0
	// Font sizes the drawer's own layout arithmetic depends on. They are
	// named because eventRowsIn and eventColumnTextBudgetPx must measure the
	// same faces the widgets are actually built with.
	drawerHeadingSizeDIP = 10.0
	drawerEventSizeDIP   = 8.5
)

// notesContent wraps the Field Notes drawer body's wrapped Text and reports
// a fixed width plus a height floored at minHeightPx, mirroring scrollContent
// (see panel.go) with one addition: minHeightPx keeps the scroll container's
// own slot in the drawer's RowLayout column a stable size regardless of note
// length (RowLayoutData.MaxHeight on the scroll container clamps the other
// direction, so together the two make the slot exactly noteHeight whether
// the note is short or long). Height above the floor still reports the
// text's true preferred height, so the ScrollContainer's own scroll-extent
// math (which reads this same PreferredSize) sees the full note and can
// scroll to it.
type notesContent struct {
	*widget.Container
	widthPx, minHeightPx int
}

func (c notesContent) PreferredSize() (int, int) {
	_, height := c.Container.PreferredSize()
	if height < c.minHeightPx {
		height = c.minHeightPx
	}
	return c.widthPx, height
}

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
		parts = append(parts, "[color=9fb1ae]SOURCES[/color] · "+ui.FieldNoteReferenceMarkup(note.References))
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
	if state.NotesMode == NotesHidden {
		return p.buildHiddenDrawerBar(state, newestEvents(state.Frame.Events, drawerHiddenEventLines))
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
	// The body is a fixedLayout holding two absolutely-placed columns rather
	// than one vertical stack. A horizontal RowLayout would not do: its
	// RowLayoutData.Stretch stretches a child perpendicular to the layout
	// direction, so it offers no way to say "events take a fixed width, the
	// note takes the rest". The drawer root already places its children by
	// explicit rect, so the columns follow the same pattern.
	bodyTop := mapBottom - height
	body := widget.NewContainer(
		widget.ContainerOpts.Layout(fixedLayout{}),
		widget.ContainerOpts.BackgroundImage(t.solid(background)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(mapLeft, bodyTop, mapRight-mapLeft, height))),
	)
	p.handles.drawerBody = body

	colTop := bodyTop + drawerBodyPadV
	colHeight := height - 2*drawerBodyPadV
	eventsLeft := mapRight - drawerBodyPadH - drawerEventsColW
	notesLeft := mapLeft + drawerBodyPadH
	notesWidth := eventsLeft - drawerColGutter - notesLeft

	notes := t.column(drawerColSpacing, nil, nil,
		widget.WidgetOpts.LayoutData(p.rect(notesLeft, colTop, notesWidth, colHeight)))
	p.handles.notesColumn = notes
	notes.AddChild(t.label(heading, drawerHeadingSizeDIP, headingColor))

	// Both columns subtract the same heading height, so their bodies start at
	// the same y whatever the heading face measures. This has to go through
	// the face rather than the built widget: ebitenui panics on a Text's
	// PreferredSize before the UI has validated the tree, which a build pass
	// by definition has not.
	columnBodyPx := t.px(colHeight) - t.lineHeightPx(drawerHeadingSizeDIP) - t.px(drawerColSpacing)

	noteTextHolder := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionVertical))))
	noteText := widget.NewText(
		widget.TextOpts.Text(noteBody(state.Note), t.face(9), colorText),
		widget.TextOpts.ProcessBBCode(true),
		widget.TextOpts.LinkColor(&widget.TextLinkColor{Idle: colorGoldDeep, Hover: colorGold}),
		widget.TextOpts.LinkClickedHandler(func(args *widget.LinkEventArgs) {
			// Text's link hit testing does not clip to ancestor scroll views
			// or respect modal windows. Only accept visible gameplay clicks.
			point := args.Text.GetWidget().Rect.Min.Add(image.Pt(args.OffsetX, args.OffsetY))
			if state.Overlay.Scene == ui.SceneGameplay && !state.ShortcutsOpen && !state.BandListOpen && !state.Ending.Visible &&
				p.handles.notesScroll != nil && point.In(p.handles.notesScroll.ViewRect()) && ui.IsPublicationURL(args.Id) {
				p.emit(Intent{Kind: IntentOpenPublication, URL: args.Id})
			}
		}),
		widget.TextOpts.MaxWidth(float64(t.px(notesWidth))),
	)
	p.handles.noteText = noteText
	noteTextHolder.AddChild(noteText)
	// notesContent floors its reported height at the column body (the
	// drawer's budget for the note) so a short note still gives the scroll
	// container the same slot a long one does — RowLayoutData.MaxHeight below
	// only ever shrinks, never grows, so without this floor a short note
	// would report a smaller PreferredSize and leave the column ragged.
	content := notesContent{Container: noteTextHolder, widthPx: t.px(notesWidth), minHeightPx: columnBodyPx}
	scroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{Idle: t.solid(background), Disabled: t.solid(background), Mask: t.solid(background)}),
		widget.ScrollContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true, MaxHeight: columnBodyPx})),
	)
	p.handles.notesScroll = scroll
	p.wireScrollWheel(scroll, content)
	notes.AddChild(scroll)
	body.AddChild(notes)

	logColumn := t.column(drawerColSpacing, nil, nil,
		widget.WidgetOpts.LayoutData(p.rect(eventsLeft, colTop, drawerEventsColW, colHeight)))
	p.handles.eventsColumn = logColumn
	logColumn.AddChild(t.label("RECENT EVENTS", drawerHeadingSizeDIP, headingColor))
	events := newestEvents(state.Frame.Events, t.eventRowsIn(columnBodyPx))
	if len(events) == 0 {
		logColumn.AddChild(t.label("No campaign events yet.", drawerEventSizeDIP, colorDim))
	}
	budget := t.eventColumnTextBudgetPx()
	for _, event := range events {
		kind := event.Kind
		line := widget.NewButton(
			widget.ButtonOpts.Image(&widget.ButtonImage{Idle: t.solid(background), Hover: t.solid(colorRowOpen), Pressed: t.solid(colorRow)}),
			widget.ButtonOpts.Text(truncateToWidth(eventLine(event), budget, t.face(drawerEventSizeDIP)), t.face(drawerEventSizeDIP), t.buttonText(colorDim)),
			widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentFocusEvent, Event: kind}) }),
			widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
		)
		p.handles.events = append(p.handles.events, line)
		logColumn.AddChild(line)
	}
	body.AddChild(logColumn)
	root.AddChild(body)
	return root
}

// lineHeightPx is one line's height at sizeDIP, in render pixels. It measures
// the face directly because ebitenui widgets panic on PreferredSize until the
// UI has validated them, so a build pass cannot measure its own widgets.
func (t *theme) lineHeightPx(sizeDIP float64) int {
	_, height := text.Measure("Ag", *t.face(sizeDIP), 0)
	return int(height + 0.5)
}

// eventRowsIn reports how many event rows fit in bodyPx render pixels of
// column. The rows are built by widget.NewButton directly rather than by
// theme.button, so they carry no TextPadding and stand exactly one line
// tall; only the column's own spacing separates them.
func (t *theme) eventRowsIn(bodyPx int) int {
	rowPx, gapPx := t.lineHeightPx(drawerEventSizeDIP), t.px(drawerColSpacing)
	if rowPx+gapPx <= 0 {
		return 0
	}
	// n rows occupy n*rowPx + (n-1)*gapPx, so the trailing gap belongs to the
	// budget rather than being subtracted from every row.
	return max((bodyPx+gapPx)/(rowPx+gapPx), 0)
}

// eventColumnTextBudgetPx is the render-pixel width an event row's text has.
// The rows carry no padding of their own (see eventRowsIn), so the budget is
// the column width — but it still goes through truncateToWidth, so a longer
// summary or a larger face shortens the line instead of spilling it over the
// note column. Mirrors hiddenBarEventBudgetPx.
func (t *theme) eventColumnTextBudgetPx() float64 {
	return float64(t.px(drawerEventsColW))
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

	control := t.button(hiddenBarControlLabel, 9, accent, accent, func() { open() })
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
