package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

type workforceHandles struct {
	sliders [gameapi.AssignmentCount]*widget.Slider
	values  [gameapi.AssignmentCount]*widget.Text
	minus   [gameapi.AssignmentCount]*widget.Button
	plus    [gameapi.AssignmentCount]*widget.Button
	total   *widget.Text
	apply   *widget.Button
	discard *widget.Button
	header  *widget.Button
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
		line := t.rowOf(6, stretch())
		marker := "  "
		if role == draft.SelectedRole {
			marker = "› "
		}
		line.AddChild(t.label(fmt.Sprintf("%s%-14s", marker, ui.RoleShortLabel(role)), 9.5, colorText))
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
			widget.SliderOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(120), t.px(12)), widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		)
		p.handles.workforce.sliders[role] = slider
		line.AddChild(slider)
		plus := t.button("+", 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentAdjustRole, Role: current, Delta: 100}) })
		p.handles.workforce.plus[role] = plus
		line.AddChild(plus)
		value := t.label(fmt.Sprintf("%3d%%", int(draft.AllocationBP[role])/100), 9.5, colorText)
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
		h.values[role].Label = fmt.Sprintf("%3d%%", int(draft.AllocationBP[role])/100)
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
