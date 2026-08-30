package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestCampaignEndSceneClassifiesDestinationBreadth(t *testing.T) {
	tests := []struct {
		name       string
		regions    []gameapi.Region
		wantPhrase string
	}{
		{name: "one", regions: []gameapi.Region{gameapi.Frangistan}, wantPhrase: "Successful dispersal"},
		{name: "three", regions: []gameapi.Region{gameapi.Frangistan, gameapi.SouthAsia, gameapi.Sahul}, wantPhrase: "Broad dispersal — 3 of five"},
		{name: "five", regions: campaignDestinations[:], wantPhrase: "Complete destination coverage"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ending := CampaignEndScene(&gameapi.Frame{
				CampaignResult:            gameapi.Victory,
				Turn:                      400,
				YearBP:                    20_000,
				SapiensEstablishedRegions: test.regions,
			})
			if !ending.Visible || ending.DestinationCount != len(test.regions) || !strings.Contains(ending.Epilogue, test.wantPhrase) {
				t.Fatalf("ending = %#v", ending)
			}
		})
	}
}

func TestCampaignEndSceneSummarizesTerminalFrameInStableOrder(t *testing.T) {
	ending := CampaignEndScene(&gameapi.Frame{
		CampaignResult: gameapi.Extinction,
		Turn:           178,
		YearBP:         34_400,
		SapiensEstablishedRegions: []gameapi.Region{
			gameapi.Sahul, gameapi.EastAfrica, gameapi.Frangistan, gameapi.YellowRiverBasin, gameapi.SouthAsia,
		},
		Bands: []gameapi.Band{
			{Species: gameapi.ArchaicHominin, Population: 17},
			{Species: gameapi.ArchaicHominin, Population: 11},
		},
	})
	if ending.Title != "HOMO SAPIENS EXTINCT" || ending.ArchaicPopulation != 28 || ending.ArchaicBands != 2 || ending.RegionsEstablished != 5 {
		t.Fatalf("ending summary = %#v", ending)
	}
	wantDestinations := "Frangistan · South Asia · Yellow River Basin\nSahul"
	if ending.Destinations != wantDestinations {
		t.Fatalf("destinations = %q, want %q", ending.Destinations, wantDestinations)
	}
}

func TestCampaignEndSceneHidesForOngoingCampaign(t *testing.T) {
	if ending := CampaignEndScene(&gameapi.Frame{CampaignResult: gameapi.Ongoing}); ending.Visible {
		t.Fatalf("ongoing ending = %#v", ending)
	}
}

func TestCampaignEndSceneExplainsTurn400DispersalFailure(t *testing.T) {
	ending := CampaignEndScene(&gameapi.Frame{
		CampaignResult: gameapi.DispersalFailed,
		Turn:           400,
		YearBP:         20_000,
		Bands:          []gameapi.Band{{Species: gameapi.HomoSapiens, Population: 42}},
	})
	if ending.Title != "DISPERSAL FAILED" || !strings.Contains(ending.Subtitle, "20,000 BP") || ending.Destinations != "None" {
		t.Fatalf("dispersal-failed ending = %#v", ending)
	}
}
