package app

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// preferenceController owns asynchronous preference persistence. Presentation
// consumes accepted reads; write completions never reinstall old UI state.
type preferenceController struct {
	store   ui.UISettingsStore
	value   ui.UISettings
	loading bool
	// readRevision and writeRevision are separate spaces. A shared counter
	// let a write started during the initial read advance past the revision
	// that read was issued with, so its completion no longer matched and
	// loading never cleared -- which blocks every gameplay input.
	readRevision  uint64
	writeRevision uint64
	writeActive   bool
	pending       *ui.UISettings
}

type preferenceCompletion struct {
	installed bool
	notice    string
}

func newPreferenceController(store ui.UISettingsStore) (preferenceController, error) {
	controller := preferenceController{store: store, value: ui.DefaultUISettings()}
	if store == nil {
		return controller, nil
	}
	if err := store.BeginRead(1); err != nil {
		return controller, err
	}
	controller.loading = true
	controller.readRevision = 1
	return controller, nil
}

func (p *preferenceController) poll() []preferenceCompletion {
	if p.store == nil {
		return nil
	}
	var results []preferenceCompletion
	for _, completion := range p.store.Poll() {
		switch completion.Operation {
		case ui.UISettingsRead:
			if !p.loading || completion.Revision != p.readRevision {
				continue
			}
			p.loading = false
			result := preferenceCompletion{installed: true}
			if completion.Err != nil {
				// Keep the defaults the constructor seeded: a store is not
				// required to carry usable settings alongside its error, and a
				// zero record reads as silent audio with easy mode off.
				result.notice = "Preferences could not be loaded; using defaults"
			} else {
				p.value = ui.NormalizeUISettings(completion.Settings)
			}
			results = append(results, result)
		case ui.UISettingsWrite:
			if !p.writeActive || completion.Revision != p.writeRevision {
				continue
			}
			p.writeActive = false
			if completion.Err != nil {
				results = append(results, preferenceCompletion{notice: "Preferences could not be saved"})
			}
			if p.pending != nil {
				pending := *p.pending
				p.pending = nil
				if pending != completion.Settings {
					if err := p.startWrite(pending); err != nil {
						results = append(results, preferenceCompletion{notice: "Preferences could not be saved"})
					}
				}
			}
		}
	}
	return results
}

func (p *preferenceController) update(settings ui.UISettings) error {
	p.value = ui.NormalizeUISettings(settings)
	if p.store == nil {
		return nil
	}
	if p.writeActive {
		pending := p.value
		p.pending = &pending
		return nil
	}
	return p.startWrite(p.value)
}

func (p *preferenceController) startWrite(settings ui.UISettings) error {
	p.writeRevision++
	if err := p.store.BeginWrite(p.writeRevision, settings); err != nil {
		return err
	}
	p.writeActive = true
	return nil
}

func (g *Game) pollUISettings() {
	for _, completion := range g.preferences.poll() {
		if completion.installed {
			// Only an accepted read seeds the live guide from persisted preferences.
			g.guide = ui.NewGuideState(g.preferences.value.GuideDismissed)
			g.applyPresentationSettings()
			g.syncEasyMode()
		}
		if completion.notice != "" {
			g.showNotice(completion.notice)
		}
	}
}

func (g *Game) updateUISettings(settings ui.UISettings) {
	err := g.preferences.update(settings)
	g.applyPresentationSettings()
	if err != nil {
		g.showNotice("Preferences could not be saved")
	}
}

func (g *Game) applyPresentationSettings() {
	settings := g.preferences.value
	g.notesMode = notesModeFor(settings)
	g.sound.SetMaster(settings.MasterVolume, settings.Muted)
	g.scene.SetReducedMotion(settings.ReducedMotion)
}

// notesModeFor derives the drawer height state from the persisted preference
// pair, which remains the single source of truth across sessions.
func notesModeFor(settings ui.UISettings) hud.NotesMode {
	switch {
	case !settings.FieldNotesVisible:
		return hud.NotesHidden
	case settings.FieldNotesExpanded:
		return hud.NotesExpanded
	default:
		return hud.NotesCompact
	}
}

// setNotesMode is the one path that changes the drawer, so the F key and the
// drawer's own tab cannot disagree about what gets persisted.
func (g *Game) setNotesMode(mode hud.NotesMode) {
	if g.preferences.loading {
		g.showNotice("Loading preferences…")
		return
	}
	settings := g.preferences.value
	settings.FieldNotesVisible = mode != hud.NotesHidden
	if mode != hud.NotesHidden {
		settings.FieldNotesExpanded = mode == hud.NotesExpanded
	}
	g.updateUISettings(settings)
}

// toggleFieldNotes hides a visible drawer and restores the height the player
// last chose when showing it again.
func (g *Game) toggleFieldNotes() {
	if g.notesMode != hud.NotesHidden {
		g.setNotesMode(hud.NotesHidden)
		return
	}
	mode := hud.NotesCompact
	if g.preferences.value.FieldNotesExpanded {
		mode = hud.NotesExpanded
	}
	g.setNotesMode(mode)
}

func (g *Game) toggleMute() {
	if g.preferences.loading {
		g.showNotice("Loading preferences…")
		return
	}
	settings := g.preferences.value
	settings.Muted = !settings.Muted
	g.updateUISettings(settings)
	if settings.Muted {
		g.showNotice("Sound muted")
	} else {
		g.showNotice(fmt.Sprintf("Sound unmuted at %.0f%%", settings.MasterVolume*100))
	}
}

func (g *Game) toggleReducedMotion() {
	if g.preferences.loading {
		g.showNotice("Loading preferences…")
		return
	}
	settings := g.preferences.value
	settings.ReducedMotion = !settings.ReducedMotion
	g.updateUISettings(settings)
	if settings.ReducedMotion {
		g.showNotice("Fog shimmer disabled")
	} else {
		g.showNotice("Fog shimmer enabled")
	}
}

func (g *Game) adjustVolume(delta float64) {
	if g.preferences.loading {
		g.showNotice("Loading preferences…")
		return
	}
	settings := g.preferences.value
	settings.MasterVolume += delta
	g.updateUISettings(settings)
	g.showNotice(fmt.Sprintf("Master volume %.0f%%", g.preferences.value.MasterVolume*100))
}

func (g *Game) syncEasyMode() {
	if g.preferences.loading || g.frame == nil || g.frame.EasyMode == g.preferences.value.EasyMode {
		return
	}
	frame, err := g.port.Apply(gameapi.SetEasyMode{Enabled: g.preferences.value.EasyMode})
	if err != nil {
		g.showNotice(ui.ErrorMessageForMode(err, g.preferences.value.EasyMode))
		return
	}
	g.frame = frame
	g.publishFrame()
}

func (g *Game) toggleEasyMode() {
	if g.preferences.loading {
		return
	}
	settings := g.preferences.value
	settings.EasyMode = !settings.EasyMode
	frame, err := g.port.Apply(gameapi.SetEasyMode{Enabled: settings.EasyMode})
	if err != nil {
		g.showNotice(ui.ErrorMessageForMode(err, g.preferences.value.EasyMode))
		return
	}
	g.frame = frame
	g.publishFrame()
	g.updateUISettings(settings)
}
