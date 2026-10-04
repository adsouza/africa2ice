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

// assertEnumBijection is the count sentinel for one domain-to-gameapi enum
// mapper. The two enums must have the same number of values, and the mapper
// must send each domain value to a distinct in-range gameapi value. Every
// mapper panics on an unmapped case, so a value added to one enum without the
// other otherwise surfaces as a crash the first time it is projected.
func assertEnumBijection[D, G ~uint8](t *testing.T, name string, domainCount D, gameCount G, mapper func(D) G) {
	t.Helper()
	if int(domainCount) != int(gameCount) {
		t.Errorf("%s: domain has %d values, gameapi has %d; the enums must move together", name, domainCount, gameCount)
		return
	}
	seen := map[G]D{}
	for value := D(0); value < domainCount; value++ {
		mapped := mapper(value)
		if mapped >= gameCount {
			t.Errorf("%s(%d) = %d, outside the gameapi enum", name, value, mapped)
		}
		if previous, duplicate := seen[mapped]; duplicate {
			t.Errorf("%s maps both %d and %d to %d", name, previous, value, mapped)
		}
		seen[mapped] = value
	}
}

func TestEveryEnumMapperIsACompleteBijection(t *testing.T) {
	assertEnumBijection(t, "mapSpecies", domain.SpeciesCount, gameapi.SpeciesCount, mapSpecies)
	assertEnumBijection(t, "mapBiome", domain.BiomeCount, gameapi.BiomeCount, mapBiome)
	assertEnumBijection(t, "mapSeason", domain.SeasonCount, gameapi.SeasonCount, mapSeason)
	assertEnumBijection(t, "mapEra", domain.CampaignEraCount, gameapi.CampaignEraCount, mapEra)
	assertEnumBijection(t, "mapRegion", domain.RegionCount, gameapi.RegionCount, mapRegion)
	assertEnumBijection(t, "mapResult", domain.CampaignResultCount, gameapi.CampaignResultCount, mapResult)
	assertEnumBijection(t, "mapEpoch", domain.ClimateEpochCount, gameapi.ClimateEpochCount, mapEpoch)
	assertEnumBijection(t, "mapTech", domain.TechCount, gameapi.TechCount, mapTech)
	assertEnumBijection(t, "mapTrait", domain.HeritableTraitCount, gameapi.HeritableTraitCount, mapTrait)
	assertEnumBijection(t, "mapFaunaGroup", domain.FaunaGroupCount, gameapi.FaunaGroupCount, mapFaunaGroup)
	assertEnumBijection(t, "mapEventKind", domain.EventKindCount, gameapi.EventKindCount, mapEventKind)
}

// unmapTech is the one inbound mapper: a research command names a gameapi
// technology. It must invert mapTech exactly and reject anything outside the
// enum rather than defaulting to Firecraft, technology zero.
func TestUnmapTechInvertsMapTechAndRejectsOutOfRange(t *testing.T) {
	for technology := domain.Technology(0); technology < domain.TechCount; technology++ {
		back, ok := unmapTech(mapTech(technology))
		if !ok || back != technology {
			t.Errorf("unmapTech(mapTech(%d)) = %d, %t", technology, back, ok)
		}
	}
	for _, invalid := range []gameapi.Tech{gameapi.TechCount, gameapi.TechCount + 1, ^gameapi.Tech(0)} {
		if technology, ok := unmapTech(invalid); ok {
			t.Errorf("unmapTech(%d) accepted as %d", invalid, technology)
		}
	}
}
