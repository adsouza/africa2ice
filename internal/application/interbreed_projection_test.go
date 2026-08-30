package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// coLocatedService returns a service whose first sapiens band shares a tile with
// an archaic band. The scenario starts them regions apart, so the pairing is
// arranged through a save round trip rather than by playing hundreds of turns.
func coLocatedService(t *testing.T) (*GameService, gameapi.BandID, gameapi.BandID) {
	t.Helper()
	service, err := NewGameService(0x9e3779b97f4a7c15)
	if err != nil {
		t.Fatal(err)
	}
	save, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	sapiens, archaic := -1, -1
	for index, band := range save.Bands {
		if band.Species == uint8(domain.HomoSapiens) && sapiens < 0 {
			sapiens = index
		}
		if band.Species != uint8(domain.HomoSapiens) && archaic < 0 {
			archaic = index
		}
	}
	if sapiens < 0 || archaic < 0 {
		t.Fatal("scenario lacks both a sapiens and an archaic band")
	}
	save.Bands[sapiens].TileID = save.Bands[archaic].TileID
	world, err := save.RestoreWorld()
	if err != nil {
		t.Fatalf("co-located save rejected: %v", err)
	}
	service.world = world
	return service, gameapi.BandID(save.Bands[sapiens].ID), gameapi.BandID(save.Bands[archaic].ID)
}

func bandInFrame(t *testing.T, frame *gameapi.Frame, id gameapi.BandID) gameapi.Band {
	t.Helper()
	for _, band := range frame.Bands {
		if band.ID == id {
			return band
		}
	}
	t.Fatalf("band %d is absent from the frame", id)
	return gameapi.Band{}
}

// TestAcceptedInterbreedIntentIsProjected covers the field the HUD needs to show
// the player what they chose before the turn resolves it. Without it the action
// is silent in both directions: nothing marks it available, nothing confirms it
// happened.
func TestAcceptedInterbreedIntentIsProjected(t *testing.T) {
	service, sapiens, archaic := coLocatedService(t)

	frame, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	actor := bandInFrame(t, frame, sapiens)
	if actor.HasInterbreedTarget {
		t.Fatal("band reports an accepted target before any command")
	}
	if len(actor.InterbreedCandidateIDs) == 0 {
		t.Fatal("co-located sapiens band projects no interbreed candidate")
	}

	if _, err := service.Apply(gameapi.Interbreed{BandID: sapiens, TargetBandID: archaic}); err != nil {
		t.Fatal(err)
	}
	frame, err = service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	actor = bandInFrame(t, frame, sapiens)
	if !actor.HasInterbreedTarget {
		t.Fatal("accepted interbreed intent is not projected")
	}
	if actor.InterbreedTargetID != archaic {
		t.Fatalf("projected target = %d, want %d", actor.InterbreedTargetID, archaic)
	}
	if !actor.SpatialActionUsed {
		t.Fatal("an accepted interbreed must spend the band's spatial action")
	}
}

// TestInterbreedCandidatesAreOfferedOnlyToSapiensActors keeps the projection
// honest about who may act: the domain rejects an archaic actor, so offering the
// archaic band a candidate list would advertise an action that cannot succeed.
func TestInterbreedCandidatesAreOfferedOnlyToSapiensActors(t *testing.T) {
	service, _, archaic := coLocatedService(t)
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if candidates := bandInFrame(t, frame, archaic).InterbreedCandidateIDs; len(candidates) != 0 {
		t.Fatalf("archaic band was offered %d interbreed candidates", len(candidates))
	}
}
