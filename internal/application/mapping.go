package application

import (
	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func mapSpecies(value domain.Species) gameapi.Species {
	switch value {
	case domain.HomoSapiens:
		return gameapi.HomoSapiens
	case domain.ArchaicHominin:
		return gameapi.ArchaicHominin
	default:
		panic("unmapped species")
	}
}

func mapBiome(value domain.Biome) gameapi.Biome {
	switch value {
	case domain.RiverineWoodland:
		return gameapi.RiverineWoodland
	case domain.Savanna:
		return gameapi.Savanna
	case domain.CoastalShrubland:
		return gameapi.CoastalShrubland
	case domain.MountainousHighlands:
		return gameapi.MountainousHighlands
	case domain.SemiAridDesert:
		return gameapi.SemiAridDesert
	case domain.GlacialTundra:
		return gameapi.GlacialTundra
	default:
		panic("unmapped biome")
	}
}

func mapSeason(value domain.Season) gameapi.Season {
	switch value {
	case domain.SeasonWarm:
		return gameapi.SeasonWarm
	case domain.SeasonCooling:
		return gameapi.SeasonCooling
	case domain.SeasonCold:
		return gameapi.SeasonCold
	case domain.SeasonWarming:
		return gameapi.SeasonWarming
	default:
		panic("unmapped season")
	}
}

func mapEra(value domain.CampaignEra) gameapi.CampaignEra {
	switch value {
	case domain.EraEarly:
		return gameapi.EraEarly
	case domain.EraMiddle:
		return gameapi.EraMiddle
	case domain.EraLate:
		return gameapi.EraLate
	case domain.EraFinal:
		return gameapi.EraFinal
	default:
		panic("unmapped era")
	}
}

func mapRegion(value domain.Region) gameapi.Region {
	switch value {
	case domain.EastAfrica:
		return gameapi.EastAfrica
	case domain.RestOfAfrica:
		return gameapi.RestOfAfrica
	case domain.Arabia:
		return gameapi.Arabia
	case domain.Levant:
		return gameapi.Levant
	case domain.Frangistan:
		return gameapi.Frangistan
	case domain.CentralAsia:
		return gameapi.CentralAsia
	case domain.SouthAsia:
		return gameapi.SouthAsia
	case domain.SoutheastAsia:
		return gameapi.SoutheastAsia
	case domain.EastAsia:
		return gameapi.EastAsia
	case domain.YellowRiverBasin:
		return gameapi.YellowRiverBasin
	case domain.Sahul:
		return gameapi.Sahul
	case domain.Siberia:
		return gameapi.Siberia
	case domain.Beringia:
		return gameapi.Beringia
	default:
		panic("unmapped region")
	}
}

func mapResult(value domain.CampaignResult) gameapi.CampaignResult {
	switch value {
	case domain.CampaignOngoing:
		return gameapi.Ongoing
	case domain.CampaignVictory:
		return gameapi.Victory
	case domain.CampaignExtinction:
		return gameapi.Extinction
	case domain.CampaignDispersalFailed:
		return gameapi.DispersalFailed
	default:
		panic("unmapped campaign result")
	}
}

func mapEpoch(value domain.ClimateEpoch) gameapi.ClimateEpoch {
	switch value {
	case domain.HumidOptimum:
		return gameapi.HumidOptimum
	case domain.AridTransition:
		return gameapi.AridTransition
	case domain.GlacialMaximum:
		return gameapi.GlacialMaximum
	default:
		panic("unmapped climate epoch")
	}
}

func mapTech(value domain.Technology) gameapi.Tech {
	switch value {
	case domain.Firecraft:
		return gameapi.Firecraft
	case domain.HaftedTools:
		return gameapi.HaftedTools
	case domain.PlantKnowledge:
		return gameapi.PlantKnowledge
	case domain.TailoredClothing:
		return gameapi.TailoredClothing
	case domain.CordageAndNets:
		return gameapi.CordageAndNets
	case domain.Campcraft:
		return gameapi.Campcraft
	case domain.MedicinalKnowledge:
		return gameapi.MedicinalKnowledge
	case domain.Trapping:
		return gameapi.Trapping
	case domain.CoastalNavigation:
		return gameapi.CoastalNavigation
	default:
		panic("unmapped technology")
	}
}

func unmapTech(value gameapi.Tech) (domain.Technology, bool) {
	switch value {
	case gameapi.Firecraft:
		return domain.Firecraft, true
	case gameapi.HaftedTools:
		return domain.HaftedTools, true
	case gameapi.PlantKnowledge:
		return domain.PlantKnowledge, true
	case gameapi.TailoredClothing:
		return domain.TailoredClothing, true
	case gameapi.CordageAndNets:
		return domain.CordageAndNets, true
	case gameapi.Campcraft:
		return domain.Campcraft, true
	case gameapi.MedicinalKnowledge:
		return domain.MedicinalKnowledge, true
	case gameapi.Trapping:
		return domain.Trapping, true
	case gameapi.CoastalNavigation:
		return domain.CoastalNavigation, true
	default:
		return 0, false
	}
}

func mapTrait(value domain.HeritableTrait) gameapi.HeritableTrait {
	switch value {
	case domain.ColdAdaptation:
		return gameapi.ColdAdaptation
	case domain.HighAltitudeAdaptation:
		return gameapi.HighAltitudeAdaptation
	case domain.InnateImmuneReactivity:
		return gameapi.InnateImmuneReactivity
	case domain.AridClimateAdaptation:
		return gameapi.AridClimateAdaptation
	case domain.PigmentationLevel:
		return gameapi.PigmentationLevel
	case domain.FattyAcidMetabolism:
		return gameapi.FattyAcidMetabolism
	default:
		panic("unmapped trait")
	}
}

func mapFaunaGroup(value domain.FaunaGroup) gameapi.FaunaGroup {
	switch value {
	case domain.SmallGame:
		return gameapi.SmallGame
	case domain.MediumGame:
		return gameapi.MediumGame
	case domain.LargeGame:
		return gameapi.LargeGame
	case domain.Megafauna:
		return gameapi.Megafauna
	case domain.InshoreAquatic:
		return gameapi.InshoreAquatic
	case domain.PelagicAquatic:
		return gameapi.PelagicAquatic
	default:
		panic("unmapped fauna group")
	}
}
