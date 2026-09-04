package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

// Workforce row column widths (spec: the five sliders form one aligned
// block). workforceLabelWidth pins every row's role-label column to the same
// width regardless of the label's own text length; workforceValueWidth does
// the same for the trailing percentage/worker-count column, right-aligned so
// it forms a clean right edge.
const (
	workforceLabelWidth = 84.0
	workforceValueWidth = 52.0
)

type workforceHandles struct {
	roleLabels [gameapi.AssignmentCount]*widget.Text
	sliders    [gameapi.AssignmentCount]*widget.Slider
	values     [gameapi.AssignmentCount]*widget.Text
	minus      [gameapi.AssignmentCount]*widget.Button
	plus       [gameapi.AssignmentCount]*widget.Button
	total      *widget.Text
	apply      *widget.Button
	discard    *widget.Button
	header     *widget.Button
}

// roleMarker is "› " for the selected role and blank padding otherwise, kept
// the same width either way so the role name column does not jitter.
func roleMarker(role, selected gameapi.WorkforceRole) string {
	if role == selected {
		return "› "
	}
	return "  "
}

// roleValueLabel is the percentage plus the worker count it represents,
// truncated the same way the simulation truncates (spec: no rounding up).
func roleValueLabel(population uint32, bp uint16) string {
	workers := int(population) * int(bp) / 10000
	return fmt.Sprintf("%d%% · %d", int(bp)/100, workers)
}

func workforceTotalLabel(draft WorkforceDraft) string {
	var total int
	for _, points := range draft.AllocationBP {
		total += int(points)
	}
	percent := total / 100
	switch {
	case percent > 100:
		return fmt.Sprintf("Total %d%% · reduce %d%% to apply", percent, percent-100)
	case percent < 100:
		return fmt.Sprintf("Total %d%% · add %d%% to apply", percent, 100-percent)
	default:
		return "Total 100%"
	}
}

// buildWorkforceBody is the open Workforce row (spec §4.3). The widgets it
// creates are refreshed in place by refreshWorkforce so a drag survives.
func (p *Panel) buildWorkforceBody(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	body := t.column(3, t.insets(6, 24, 10, 8), t.solid(colorRowOpen), stretch())
	draft := state.Workforce
	for role := gameapi.WorkforceRole(0); role < gameapi.AssignmentCount; role++ {
		current := role
		// A five-column grid, not a RowLayout: RowLayout's %-14s padding did
		// nothing in a proportional font, so every row's −, slider, + and
		// value started at a different x. Only the slider column (index 2)
		// stretches, so every slider shares the same left and right edges.
		line := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(5),
				widget.GridLayoutOpts.Spacing(t.px(6), 0),
				widget.GridLayoutOpts.Stretch([]bool{false, false, true, false, false}, nil),
			)),
			widget.ContainerOpts.WidgetOpts(stretch()),
		)
		roleLabel := widget.NewText(
			widget.TextOpts.Text(roleMarker(role, draft.SelectedRole)+ui.RoleShortLabel(role), t.face(9.5), colorText),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(workforceLabelWidth), 0)),
		)
		p.handles.workforce.roleLabels[role] = roleLabel
		line.AddChild(roleLabel)
		minus := t.button("−", 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentAdjustRole, Role: current, Delta: -100}) })
		p.handles.workforce.minus[role] = minus
		line.AddChild(minus)
		slider := widget.NewSlider(
			widget.SliderOpts.Orientation(widget.DirectionHorizontal),
			widget.SliderOpts.MinMax(0, 100),
			widget.SliderOpts.InitialCurrent(int(draft.AllocationBP[role])/100),
			widget.SliderOpts.Images(&widget.SliderTrackImage{Idle: t.solid(colorPanelEdge), Hover: t.solid(colorPanelEdge)}, t.buttonImages(colorGoldDeep)),
			widget.SliderOpts.FixedHandleSize(t.px(10)),
			widget.SliderOpts.TrackOffset(0),
			widget.SliderOpts.PageSizeFunc(func() int { return 5 }),
			widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
				delta := args.Current*100 - int(p.last.Workforce.AllocationBP[current])
				if delta != 0 {
					p.emit(Intent{Kind: IntentAdjustRole, Role: current, Delta: delta})
				}
			}),
			widget.SliderOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(120), t.px(12))),
		)
		p.handles.workforce.sliders[role] = slider
		line.AddChild(slider)
		plus := t.button("+", 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentAdjustRole, Role: current, Delta: 100}) })
		p.handles.workforce.plus[role] = plus
		line.AddChild(plus)
		value := t.rightLabel(roleValueLabel(draft.Population, draft.AllocationBP[role]), 9.5, colorText,
			widget.WidgetOpts.MinSize(t.px(workforceValueWidth), 0))
		p.handles.workforce.values[role] = value
		line.AddChild(value)
		body.AddChild(line)
	}
	total := t.label(workforceTotalLabel(draft), 9.5, colorText)
	p.handles.workforce.total = total
	body.AddChild(total)
	buttons := t.rowOf(6)
	apply := t.button("Apply · A / Enter", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentApplyWorkforce}) })
	discard := t.button("Discard · D", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentDiscardWorkforce}) })
	p.handles.workforce.apply, p.handles.workforce.discard = apply, discard
	buttons.AddChild(apply, discard)
	body.AddChild(buttons)
	body.AddChild(t.label("Up/Down pick a role · Left/Right or −/+ step 1% · Shift steps 5%", 8.5, colorDim))
	if band.Species == gameapi.ArchaicHominin {
		for role := range p.handles.workforce.sliders {
			p.handles.workforce.sliders[role].GetWidget().Disabled = true
			p.handles.workforce.minus[role].GetWidget().Disabled = true
			p.handles.workforce.plus[role].GetWidget().Disabled = true
		}
	}
	p.refreshWorkforce(state)
	return body
}

// refreshWorkforceHeader updates the row header's label in place. A button
// built in this very rebuild has not gone through ebitenui's Validate pass
// yet, so Text() still returns nil; the header already carries the current
// summary from its own construction in that case, so skipping is correct,
// not merely safe.
func refreshWorkforceHeader(header *widget.Button, draft WorkforceDraft) {
	if header == nil {
		return
	}
	text := header.Text()
	if text == nil {
		return
	}
	text.Label = fmt.Sprintf("%d  %-10s %s", int(ui.RowWorkforce)+1, ui.RowWorkforce.Title(), ui.WorkforceSummary(draft.AllocationBP, draft.Dirty))
}

// refreshWorkforce updates the row's dynamic parts from the draft without
// recreating widgets.
func (p *Panel) refreshWorkforce(state State) {
	p.refreshes++
	h := &p.handles.workforce
	draft := state.Workforce
	if h.total == nil {
		refreshWorkforceHeader(h.header, draft)
		return
	}
	for role := range h.sliders {
		if h.sliders[role] == nil {
			continue
		}
		if percent := int(draft.AllocationBP[role]) / 100; h.sliders[role].Current != percent {
			h.sliders[role].Current = percent
		}
		h.values[role].Label = roleValueLabel(draft.Population, draft.AllocationBP[gameapi.WorkforceRole(role)])
		h.roleLabels[role].Label = roleMarker(gameapi.WorkforceRole(role), draft.SelectedRole) + ui.RoleShortLabel(gameapi.WorkforceRole(role))
	}
	h.total.Label = workforceTotalLabel(draft)
	if draft.Valid {
		h.total.SetColor(colorText)
	} else {
		h.total.SetColor(colorRed)
	}
	h.apply.GetWidget().Disabled = !draft.Dirty || !draft.Valid
	h.discard.GetWidget().Disabled = !draft.Dirty
	refreshWorkforceHeader(h.header, draft)
}
