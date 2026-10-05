package app

import "github.com/adsouza/africa2ice/pkg/ui"

// Visiting camp is presentation navigation, including while an allocation
// draft is dirty. It neither spends an action nor publishes a new frame.
func (g *Game) openCamp() {
	if g.scenes.Current() != ui.SceneGameplay || g.shortcutsOpen || g.selected() == nil || g.endScene(g.frame).Visible {
		return
	}
	if _, accepted := g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneCamp)}); accepted {
		g.camp.Invalidate()
	}
}
