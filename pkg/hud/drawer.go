package hud

import (
	"fmt"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/ebitenui/ebitenui/widget"
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

// truncateRunes returns value unchanged when it has at most limit runes, and
// otherwise the first limit-1 runes plus an ellipsis. The hidden drawer tab
// sits in a fixed-width RowLayout that neither wraps nor clips, so a long
// event summary must be shortened before it reaches the button label.
func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

// newestEvents returns up to limit events, newest first.
func newestEvents(events []gameapi.Event, limit int) []gameapi.Event {
	result := make([]gameapi.Event, 0, limit)
	for index := len(events) - 1; index >= 0 && len(result) < limit; index-- {
		result = append(result, events[index])
	}
	return result
}

// drawerHeight is the drawer's DIP height for a mode; hidden is the tab only.
func drawerHeight(mode NotesMode) float64 {
	switch mode {
	case NotesCompact:
		return drawerCompactH
	case NotesExpanded:
		return drawerExpandedH
	default:
		return 0
	}
}

// buildDrawer places the Field Notes drawer over the bottom of the map
// (spec §4) with its edge tab, or just the tab when hidden.
func (p *Panel) buildDrawer(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	height := drawerHeight(state.NotesMode)
	root := widget.NewContainer(widget.ContainerOpts.Layout(fixedLayout{}),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(mapLeft, mapBottom-height-drawerTabH, mapRight-mapLeft, height+drawerTabH))))
	tabRow := t.rowOf(4, widget.WidgetOpts.LayoutData(p.rect(mapRight-2*drawerTabW-8, mapBottom-height-drawerTabH, 2*drawerTabW, drawerTabH)))
	events := newestEvents(state.Frame.Events, drawerEventLines)
	if state.NotesMode == NotesHidden {
		border, textColor := colorGoldDeep, colorGoldDeep
		label := "▴ notes · F"
		if state.Note.Celebration {
			border, textColor = colorGold, colorGold
			label = "BREAKTHROUGH · " + label
		}
		if len(events) > 0 {
			label += "  ·  " + truncateRunes(eventLine(events[0]), 42)
		}
		tab := t.button(label, 9, border, textColor, func() { p.emit(Intent{Kind: IntentSetNotesMode, Notes: NotesCompact}) })
		p.handles.drawerTab = tab
		tabRow.AddChild(tab)
		root.AddChild(tabRow)
		return root
	}
	moreLabel, moreMode := "▴ more", NotesExpanded
	if state.NotesMode == NotesExpanded {
		moreLabel, moreMode = "▾ less", NotesCompact
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
	body := t.column(4, t.insets(6, 12, 12, 6), solid(background),
		widget.WidgetOpts.LayoutData(p.rect(mapLeft, mapBottom-height, mapRight-mapLeft, height)))
	body.AddChild(t.label(heading, 10, headingColor))
	noteHeight := height - 24 - 14*float64(min(len(events), drawerEventLines)+1) - 12
	area := widget.NewTextArea(
		widget.TextAreaOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(mapRight-mapLeft-24), t.px(noteHeight)))),
		widget.TextAreaOpts.FontFace(t.face(9)),
		widget.TextAreaOpts.FontColor(colorText),
		widget.TextAreaOpts.ProcessBBCode(true),
		widget.TextAreaOpts.Text(noteBody(state.Note)),
		widget.TextAreaOpts.ScrollContainerImage(&widget.ScrollContainerImage{Idle: solid(background), Mask: solid(background)}),
	)
	body.AddChild(area)
	body.AddChild(t.label("RECENT EVENTS", 8, headingColor))
	if len(events) == 0 {
		body.AddChild(t.label("No campaign events yet.", 8.5, colorDim))
	}
	for _, event := range events {
		kind := event.Kind
		line := widget.NewButton(
			widget.ButtonOpts.Image(&widget.ButtonImage{Idle: solid(background), Hover: solid(colorRowOpen), Pressed: solid(colorRow)}),
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
