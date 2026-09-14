package app

func (g *Game) syncAssignmentDraft(force bool) {
	band := g.selected()
	g.syncInterbreedFocus(band)
	g.workforce.Sync(band, force)
}

func (g *Game) editAssignmentDraft(delta int) {
	if g.workforce.Edit(delta) {
		g.showNotice("Workforce draft changed — total must equal 100%; A applies, D discards")
	}
}

func (g *Game) applyAssignmentDraft() {
	if !g.workforce.Dirty() {
		return
	}
	command, ready := g.workforce.Command()
	if !ready {
		g.showNotice("Workforce allocation must total exactly 100%")
		return
	}
	if g.apply(command) {
		g.syncAssignmentDraft(true)
		g.showNotice("Workforce allocation applied")
	}
}

func (g *Game) discardAssignmentDraft() {
	if g.workforce.Discard() {
		g.showNotice("Workforce changes discarded")
	}
}
