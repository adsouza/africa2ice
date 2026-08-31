package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

// Repository adapters may not import the domain, so they read BandSave.Species
// through these names. If the encoding and the domain enum ever disagree, every
// save slot would report its sapiens and archaic populations swapped, and the
// round trip would stay self-consistent while doing it.
func TestSavedSpeciesMatchesTheDomainEnum(t *testing.T) {
	if SavedHomoSapiens != uint8(domain.HomoSapiens) || SavedArchaicHominin != uint8(domain.ArchaicHominin) {
		t.Fatalf("saved species encoding drifted: sapiens=%d archaic=%d", SavedHomoSapiens, SavedArchaicHominin)
	}
	if SavedHomoSapiens == SavedArchaicHominin {
		t.Fatal("saved species encoding is not distinct")
	}
	if uint8(domain.SpeciesCount) != 2 {
		t.Fatalf("a species was added; PopulationBySpecies must classify it: count=%d", domain.SpeciesCount)
	}
}

func TestPopulationBySpeciesTotalsEachSide(t *testing.T) {
	save := SaveState{Bands: []BandSave{
		{Species: SavedHomoSapiens, Population: 30},
		{Species: SavedArchaicHominin, Population: 7},
		{Species: SavedHomoSapiens, Population: 12},
	}}
	sapiens, archaic := save.PopulationBySpecies()
	if sapiens != 42 || archaic != 7 {
		t.Fatalf("totals = sapiens %d, archaic %d; want 42 and 7", sapiens, archaic)
	}
	empty, emptyArchaic := SaveState{}.PopulationBySpecies()
	if empty != 0 || emptyArchaic != 0 {
		t.Fatalf("empty save totals = %d / %d", empty, emptyArchaic)
	}
}
