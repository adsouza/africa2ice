package render

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/opentype"
)

// OutlineFont supplies cached, unhinted faces for the bundled Go Regular and
// monochrome biome fonts. These need outline rasterization and pair kerning,
// but no complex-script shaping, bitmap or SVG color glyph support. GoXFace
// keeps those unused GoTextFace renderers out of the release WASM module.
// Like the renderer, OutlineFont is used only on the game loop goroutine.
type OutlineFont struct {
	font  *opentype.Font
	faces map[float64]text.Face
}

// NewOutlineFont parses an embedded TrueType or OpenType outline font.
func NewOutlineFont(data []byte) (*OutlineFont, error) {
	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}
	return &OutlineFont{font: parsed, faces: make(map[float64]text.Face)}, nil
}

// Face returns a face rasterized at size physical pixels. Reusing the face is
// essential: each GoXFace owns its glyph image cache. Bound the number of sizes
// retained so repeated window resizes cannot grow the cache indefinitely.
func (source *OutlineFont) Face(size float64) text.Face {
	if face, ok := source.faces[size]; ok {
		return face
	}
	face, err := opentype.NewFace(source.font, &opentype.FaceOptions{Size: size, DPI: 72})
	if err != nil {
		panic(err)
	}
	const maxCachedFontSizes = 64
	if len(source.faces) >= maxCachedFontSizes {
		clear(source.faces)
	}
	result := text.NewGoXFace(face)
	source.faces[size] = result
	return result
}
