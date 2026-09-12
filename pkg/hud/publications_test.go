package hud

import (
	"image"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestWelcomeNotePresentsContextThenPlayingSteps(t *testing.T) {
	body := noteBody(ui.CampaignOverviewFieldNote())
	context, play := strings.Index(body, "HISTORICAL CONTEXT"), strings.Index(body, "HOW TO PLAY")
	if !strings.HasPrefix(body, "[color=9fb1ae]HISTORICAL CONTEXT") || play <= context {
		t.Fatalf("welcome should begin with historical context, then playing guidance: %q", body)
	}
	for _, obsolete := range []string{"SUMMARY", "GAME ABSTRACTION", "]HINT["} {
		if strings.Contains(body, obsolete) {
			t.Fatalf("welcome retained redundant section %q", obsolete)
		}
	}
	previous := play
	for _, label := range []string{"1. Tab", "2. With the Move", "3. Enter", "4. Space", "When done reading", "SOURCES"} {
		position := strings.Index(body, label)
		if position <= previous {
			t.Fatalf("missing or out-of-order guidance %q: %q", label, body)
		}
		previous = position
	}
}

// Check rendered pixels, not just theme configuration: ebitenui v0.7.3
// silently ignores TextOpts.LinkColor and otherwise paints these links blue.
func TestPublicationLinksRenderInReadableGold(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.NotesMode = NotesExpanded
	state.Note = ui.CampaignOverviewFieldNote()
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	panel.Draw(screen)
	view := panel.handles.notesScroll.ViewRect()
	pixels := make([]byte, 4*1280*720)
	screen.ReadPixels(pixels)
	gold, blue := 0, 0
	for y := view.Min.Y; y < view.Max.Y; y++ {
		for x := view.Min.X; x < view.Max.X; x++ {
			offset := 4 * (y*1280 + x)
			r, g, b := pixels[offset], pixels[offset+1], pixels[offset+2]
			if r > 180 && g > 140 && b < 120 {
				gold++
			}
			if b > 150 && r < 80 && g < 80 {
				blue++
			}
		}
	}
	if gold == 0 || blue != 0 {
		t.Fatalf("publication pixels: gold=%d, unreadable blue=%d", gold, blue)
	}
}

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
