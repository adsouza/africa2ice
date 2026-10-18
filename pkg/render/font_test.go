package render

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

func TestOutlineFontRejectsInvalidFont(t *testing.T) {
	if _, err := NewOutlineFont([]byte("not a font")); err == nil {
		t.Fatal("invalid embedded font accepted")
	}
}

func TestOutlineFontReusesFacesAndRetainsThemAcrossCacheEviction(t *testing.T) {
	source, err := NewOutlineFont(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	face := source.Face(14)
	if source.Face(14) != face {
		t.Fatal("repeated size discarded the face and its glyph cache")
	}
	width, height := text.Measure("Africa 2 Ice: café — 100%", face, 0)
	if width <= 0 || height <= 0 {
		t.Fatalf("invalid text dimensions: %v x %v", width, height)
	}
	if larger, _ := text.Measure("Africa 2 Ice: café — 100%", source.Face(28), 0); larger <= width {
		t.Fatal("physical pixel size did not scale the font")
	}
	// A widget may still hold a face after repeated resizes evict it from the
	// source cache. Eviction must not invalidate that live widget's face.
	for size := 1.0; size <= 256; size++ {
		source.Face(size)
	}
	if len(source.faces) >= 256 {
		t.Fatal("resizing retained every old font size")
	}
	if got, _ := text.Measure("Africa 2 Ice: café — 100%", face, 0); got != width {
		t.Fatalf("evicted face width = %v, want %v", got, width)
	}
}
