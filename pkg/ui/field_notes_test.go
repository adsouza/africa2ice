package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestTechnologyFieldNotesCoverTheClosedCatalog(t *testing.T) {
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		note, ok := TechnologyFieldNote(technology, 7, 1)
		if !ok || note.Topic == "" || note.Introduction == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" {
			t.Fatalf("technology %d has incomplete Field Notes: %#v", technology, note)
		}
		for _, block := range []string{note.Introduction, note.Context, note.GameEffect, note.Hint} {
			for _, line := range strings.Split(block, "\n") {
				if len([]rune(line)) > 48 {
					t.Fatalf("technology %s has an overlong Field Notes line %q", technology, line)
				}
			}
		}
	}
	if _, ok := TechnologyFieldNote(gameapi.TechCount, 7, 1); ok {
		t.Fatal("out-of-range technology has Field Notes")
	}
}
