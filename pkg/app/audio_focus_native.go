//go:build !js

package app

import "github.com/hajimehoshi/ebiten/v2"

func audioFocused() bool { return ebiten.IsFocused() }

func (g *Game) observeAudioFocus() func() { return func() {} }
