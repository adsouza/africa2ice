package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// TestMapEventKindCoversEveryDomainKind guards the projection's panic:
// mapEventKind rejects an unmapped kind rather than defaulting, so a kind
// added to the domain enum without a matching case crashes the game the first
// time that event reaches a snapshot rather than failing at build time.
func TestMapEventKindCoversEveryDomainKind(t *testing.T) {
	if int(domain.EventKindCount) != int(gameapi.EventKindCount) {
		t.Fatalf("domain has %d event kinds, gameapi has %d — the enums must move together", domain.EventKindCount, gameapi.EventKindCount)
	}
	for kind := domain.EventKind(0); kind < domain.EventKindCount; kind++ {
		mapped := mapEventKind(kind)
		if mapped.String() == "EventKind" {
			t.Errorf("kind %d maps to %d, which has no display name", kind, mapped)
		}
		if int(mapped) != int(kind) {
			t.Errorf("mapEventKind(%d) = %d, want %d — the enums have drifted apart", kind, mapped, kind)
		}
	}
}
