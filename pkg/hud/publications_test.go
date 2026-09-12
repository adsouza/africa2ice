package hud

import (
	"image"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestPublicationLinkClickRespectsDrawerAndModals(t *testing.T) {
	const url = "https://doi.org/10.1038/nature09710"
	for _, scenario := range []string{"visible", "clipped", "menu", "shortcuts", "bands", "ending"} {
		t.Run(scenario, func(t *testing.T) {
			panel := New()
			state := testState(testFrame(1), 1)
			state.NotesMode = NotesExpanded
			state.Note = ui.CampaignOverviewFieldNote()
			switch scenario {
			case "menu":
				state.Overlay.Scene = ui.SceneMenu
			case "shortcuts":
				state.ShortcutsOpen = true
			case "bands":
				state.BandListOpen = true
			case "ending":
				state.Ending.Visible = true
			}
			panel.Update(state)
			screen := ebiten.NewImage(1280, 720)
			defer screen.Deallocate()
			panel.Draw(screen)
			note := panel.handles.noteText
			if note == nil || !strings.Contains(note.Label, "[link="+url+"]Reich et al. (2010)[/link]") {
				t.Fatal("drawer lacks linked publication text")
			}
			view := panel.handles.notesScroll.ViewRect()
			point := view.Min.Add(image.Pt(1, 1))
			if scenario == "clipped" {
				point.Y = view.Max.Y + 1
			}
			offset := point.Sub(note.GetWidget().Rect.Min)
			note.LinkClickedEvent.Fire(&widget.LinkEventArgs{Text: note, Id: url, OffsetX: offset.X, OffsetY: offset.Y})
			intents := panel.Update(state)
			if scenario == "visible" {
				if len(intents) != 1 || intents[0].Kind != IntentOpenPublication || intents[0].URL != url {
					t.Fatalf("publication click = %+v", intents)
				}
			} else if len(intents) != 0 {
				t.Fatalf("invisible or obscured citation emitted %+v", intents)
			}
		})
	}
}
