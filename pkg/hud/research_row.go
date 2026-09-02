package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/ebitenui/ebitenui/widget"
)

var techShortNames = [gameapi.TechCount]string{"Fire", "Haft", "Plants", "Clothes", "Cordage", "Camp", "Medicine", "Traps", "Navigation"}

func missingPrerequisites(option gameapi.ResearchOption, acquired uint16) string {
	label := ""
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		if option.PrerequisiteMask&^acquired&(1<<technology) == 0 {
			continue
		}
		if label != "" {
			label += " + "
		}
		label += techShortNames[technology]
	}
	return label
}

// buildResearchBody lists the nine technologies (spec §4.2); available ones
// are buttons, the rest explain their state.
func (p *Panel) buildResearchBody(_ State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	body := t.column(3, t.insets(6, 24, 10, 8), solid(colorRowOpen), stretch())
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		option := band.ResearchOptions[technology]
		progress := fmt.Sprintf("%.0f/%.0f", band.ResearchProgress[technology], option.Cost)
		label := fmt.Sprintf("%d  %-20s %s", int(technology)+1, technology.String(), progress)
		var status string
		border, textColor := colorPanelEdge, colorText
		switch {
		case option.Acquired:
			status, border, textColor = "learned", colorGreen, colorGreen
		case option.Current:
			status, border, textColor = "current", colorGold, colorGold
		case option.Available && band.Species == gameapi.HomoSapiens:
			status = "available"
		case band.Species == gameapi.ArchaicHominin:
			status, textColor = "computer", colorDim
		default:
			status, textColor = "needs "+missingPrerequisites(option, band.AcquiredTech), colorDim
		}
		tech := technology
		button := widget.NewButton(
			widget.ButtonOpts.Image(t.buttonImages(border)),
			widget.ButtonOpts.Text(label+" · "+status, t.face(9.5), t.buttonText(textColor)),
			widget.ButtonOpts.TextPadding(t.insets(3, 8, 8, 3)),
			widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentChooseResearch, Tech: tech}) }),
			widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
		)
		button.GetWidget().Disabled = option.Acquired || !option.Available || band.Species != gameapi.HomoSapiens
		p.handles.research[technology] = button
		body.AddChild(button)
	}
	body.AddChild(t.label("Up/Down highlight · Enter chooses · keys 1–9", 8.5, colorDim))
	return body
}
