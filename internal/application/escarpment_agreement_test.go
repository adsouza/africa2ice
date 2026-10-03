package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// gameapi.EscarpmentBlocks re-implements domain.Grid.EscarpmentBlocks over the
// frame's explored escarpments, because presentation and verification cannot
// see the domain. This package can see both, so it is where the two copies are
// held together: with every tile explored they must agree on every ordinary
// one-tile move, and under fog the frame may hide a cliff but never invent one.
func TestFrameEscarpmentRuleAgreesWithDomain(t *testing.T) {
	world, err := domain.NewWorld(1)
	if err != nil {
		t.Fatal(err)
	}
	fogged, err := projectFrame(world, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	state, err := world.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	for id := range domain.TileCount {
		state.ExploredTiles[id/64] |= uint64(1) << (id % 64)
	}
	explored, err := domain.RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := projectFrame(explored, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	grid := explored.Grid()
	blocked, diagonalBlocked, hiddenByFog := 0, 0, 0
	for from := range domain.TileCount {
		x, y, _ := domain.TileXY(domain.TileID(from))
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				to, err := domain.TileIDAt(x+dx, y+dy)
				if err != nil || dx == 0 && dy == 0 {
					continue
				}
				want := grid.EscarpmentBlocks(domain.TileID(from), to)
				if got := gameapi.EscarpmentBlocks(frame, gameapi.TileID(from), gameapi.TileID(to)); got != want {
					t.Fatalf("explored move %d -> %d: frame rule %v, domain rule %v", from, to, got, want)
				}
				underFog := gameapi.EscarpmentBlocks(fogged, gameapi.TileID(from), gameapi.TileID(to))
				if underFog && !want {
					t.Fatalf("fogged move %d -> %d: frame rule invents a cliff the domain does not have", from, to)
				}
				if want {
					blocked++
					if dx != 0 && dy != 0 {
						diagonalBlocked++
					}
					if !underFog {
						hiddenByFog++
					}
				}
			}
		}
	}
	// Each count must be non-zero or a branch of the rule went unexercised.
	if blocked == 0 || diagonalBlocked == 0 || hiddenByFog == 0 {
		t.Fatalf("blocked %d, diagonal %d, hidden by fog %d: every case must occur", blocked, diagonalBlocked, hiddenByFog)
	}
}
