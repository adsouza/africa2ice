// Package hud is the ebitenui-based gameplay chrome: the task-oriented right
// panel, the Field Notes drawer, the first-turn guide card, and the scene
// overlays. It renders a State value the application derives every tick and
// returns Intent values; it never touches the simulation port, storage, or
// saves, and every layout constant it owns is authored in DIPs and scaled at
// build time.
package hud

import _ "github.com/ebitenui/ebitenui"
