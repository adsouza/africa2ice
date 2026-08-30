package ui

import (
	"fmt"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

var campaignDestinations = [...]gameapi.Region{
	gameapi.Frangistan,
	gameapi.SouthAsia,
	gameapi.YellowRiverBasin,
	gameapi.Sahul,
	gameapi.Beringia,
}

// CampaignEndScene derives terminal presentation content from an accepted
// frame. It never infers whether the campaign is over from the turn or bands.
func CampaignEndScene(frame *gameapi.Frame) render.EndScene {
	if frame == nil || frame.CampaignResult == gameapi.Ongoing {
		return render.EndScene{}
	}
	ending := render.EndScene{
		Visible:            true,
		Result:             frame.CampaignResult,
		Turn:               frame.Turn,
		YearBP:             frame.YearBP,
		RegionsEstablished: len(frame.SapiensEstablishedRegions),
	}
	for _, band := range frame.Bands {
		switch band.Species {
		case gameapi.HomoSapiens:
			ending.SapiensBands++
			ending.SapiensPopulation += uint64(band.Population)
		case gameapi.ArchaicHominin:
			ending.ArchaicBands++
			ending.ArchaicPopulation += uint64(band.Population)
		}
	}

	established := make(map[gameapi.Region]bool, len(frame.SapiensEstablishedRegions))
	for _, region := range frame.SapiensEstablishedRegions {
		established[region] = true
	}
	destinations := make([]string, 0, len(campaignDestinations))
	for _, region := range campaignDestinations {
		if established[region] {
			destinations = append(destinations, region.String())
		}
	}
	ending.DestinationCount = len(destinations)
	ending.Destinations = formatDestinationList(destinations)

	switch frame.CampaignResult {
	case gameapi.Victory:
		ending.Title = "DISPERSAL ACHIEVED"
		ending.Subtitle = "Homo sapiens endured to the campaign horizon\nwith a lasting foothold in a distant destination."
		switch len(destinations) {
		case 1:
			ending.Epilogue = "Successful dispersal — one destination established."
		case len(campaignDestinations):
			ending.Epilogue = "Complete destination coverage — all five established."
		default:
			ending.Epilogue = fmt.Sprintf("Broad dispersal — %d of five destinations established.", len(destinations))
		}
	case gameapi.Extinction:
		ending.Title = "HOMO SAPIENS EXTINCT"
		ending.Subtitle = fmt.Sprintf("The last sapiens band disappeared by %d BP.", frame.YearBP)
		if len(destinations) == 0 {
			ending.Epilogue = "No lasting foothold was established in a campaign destination."
		} else {
			ending.Epilogue = "Earlier dispersal achievements remain part of this campaign's history."
		}
	case gameapi.DispersalFailed:
		ending.Title = "DISPERSAL FAILED"
		ending.Subtitle = "Homo sapiens survived to 20,000 BP, but the campaign objective was not met."
		ending.Epilogue = "No lasting foothold was established in the five campaign destinations."
	}
	return ending
}

func formatDestinationList(destinations []string) string {
	if len(destinations) == 0 {
		return "None"
	}
	if len(destinations) <= 3 {
		return strings.Join(destinations, " · ")
	}
	return strings.Join(destinations[:3], " · ") + "\n" + strings.Join(destinations[3:], " · ")
}
