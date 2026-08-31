package ui

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

var technologyFieldNotes = [gameapi.TechCount]struct {
	context, gameEffect, hint string
}{
	gameapi.Firecraft: {
		context:    "Controlled fire supported warmth,\ncooking, light, and safer camps.",
		gameEffect: "Capacity rises; exposure and some\npredation risks fall.",
		hint:       "Campcraft and Medicinal Knowledge\nboth build on Firecraft.",
	},
	gameapi.HaftedTools: {
		context:    "Binding worked stone to shafts made\nstronger, more versatile tools.",
		gameEffect: "Hunting improves; some predation\nrisks and injuries become less severe.",
		hint:       "It unlocks clothing, cordage, and\n—with Firecraft—Campcraft.",
	},
	gameapi.PlantKnowledge: {
		context:    "Foraging relies on learned seasonal\nknowledge of edible plants.",
		gameEffect: "Plant-food collection and ecological\ncapacity both improve.",
		hint:       "Combine it with Firecraft to unlock\nMedicinal Knowledge.",
	},
	gameapi.TailoredClothing: {
		context:    "Fitted hide garments protect more\neffectively than loosely draped skins.",
		gameEffect: "Capacity rises and exposure risk\nfalls substantially.",
		hint:       "This is especially useful as bands\nenter colder regions.",
	},
	gameapi.CordageAndNets: {
		context:    "Katanda points/fish date to ~90 ka;\nJerimalai pelagic catch to ~42 ka.",
		gameEffect: "Inshore fishing improves and pelagic\nresources become accessible.",
		hint:       "Shell hooks date to 23–16 ka; Lake\nCondah stone traps are post-campaign.",
	},
	gameapi.Campcraft: {
		context:    "Organized shelter, hearth, food, and\nwaste practices make camps safer.",
		gameEffect: "Capacity rises; exposure, disease,\nand some acute incident risks fall.",
		hint:       "Shelter workers compound these gains\nwithout making any risk vanish.",
	},
	gameapi.MedicinalKnowledge: {
		context:    "Caregiving and accumulated plant\nknowledge can ease illness burdens.",
		gameEffect: "Disease risk, health damage, and the\nseverity of outbreaks are reduced.",
		hint:       "It mitigates disease; it does not\nprevent every outbreak.",
	},
	gameapi.Trapping: {
		context:    "Snares and traps exchange direct\npursuit for planning and patience.",
		gameEffect: "Small and medium game yields improve;\nsome predation risk also falls.",
		hint:       "It is most valuable where suitable\nterrestrial fauna are present.",
	},
	gameapi.CoastalNavigation: {
		context:    "Open-water travel needs craft, route\nknowledge, and group coordination.",
		gameEffect: "Pelagic hunting improves and named\nWallacea passages become usable.",
		hint:       "Move to a passage endpoint; arbitrary\nwater tiles remain impassable.",
	},
}

var traitFieldNotes = [gameapi.HeritableTraitCount]struct {
	context, gameEffect, hint string
}{
	gameapi.ColdAdaptation: {
		context:    "Heritable cold responses can shift over\nmany generations under local selection.",
		gameEffect: "Higher values reduce cold exposure and\nsupport life in glacial environments.",
		hint:       "The value is inherited and exchanged by\ngene flow; it is not a technology.",
	},
	gameapi.HighAltitudeAdaptation: {
		context:    "High-altitude populations can accumulate\nphysiological responses to low oxygen.",
		gameEffect: "Higher values reduce hypoxia pressure in\nmountainous highlands.",
		hint:       "Selection is strongest where elevation and\noccupancy keep the pressure active.",
	},
	gameapi.InnateImmuneReactivity: {
		context:    "Immune responses trade pathogen defence\nagainst damaging overreaction.",
		gameEffect: "Local disease pressure selects the trait;\nextreme values carry their own burden.",
		hint:       "Medicine and camp hygiene remain separate,\nlearned layers of protection.",
	},
	gameapi.AridClimateAdaptation: {
		context:    "Heat and water scarcity create persistent\nselection in arid regions.",
		gameEffect: "Higher values reduce heat and water stress\nwithout creating water or food.",
		hint:       "Compare water stocks and demand before a\ndesert migration.",
	},
	gameapi.PigmentationLevel: {
		context:    "Pigmentation balances ultraviolet skin\nprotection against vitamin-D synthesis.",
		gameEffect: "Latitude-dependent UV pressure favours\ndifferent values in different regions.",
		hint:       "Neither end is universally best; movement\ncan reverse the local pressure.",
	},
	gameapi.FattyAcidMetabolism: {
		context:    "Dietary fat use varies heritably and can be\nselected where animal foods dominate.",
		gameEffect: "The value changes usable yield from animal\nand aquatic food sources.",
		hint:       "It changes conversion, not the number of\nanimals represented by fauna stock.",
	},
}

var regionFieldNotes = [gameapi.RegionCount]struct {
	context, hint string
}{
	gameapi.EastAfrica:       {context: "The campaign's sapiens bands begin here\namid riverine and savanna habitats.", hint: "Build resilient bands before committing to\nlonger dispersal routes."},
	gameapi.RestOfAfrica:     {context: "Africa contains several viable corridors,\nnot a single departure route.", hint: "Regional establishment rewards breadth, not\none prescribed historical path."},
	gameapi.Arabia:           {context: "Genetic reconstructions put the effective\nfounding dispersal in the low thousands.", hint: "A viable founder band must retain enough\npeople after movement and establishment."},
	gameapi.Levant:           {context: "The Levant repeatedly connected African\nand Eurasian populations.", hint: "This corridor can be useful without being the\nonly route out of Africa."},
	gameapi.Frangistan:       {context: "This broad western-Eurasian region is a\ndestination, not a privileged win route.", hint: "Establishment is one regional achievement\namong several route-neutral goals."},
	gameapi.CentralAsia:      {context: "Interior moisture and temperature shifts can\nopen and close steppe-like opportunities.", hint: "Watch water, exposure, and seasonal food\nrather than relying on colour alone."},
	gameapi.SouthAsia:        {context: "South Asia links western and eastern routes\nacross diverse monsoon habitats.", hint: "Its flora, fauna, and disease mix differs\nsubstantially by biome."},
	gameapi.SoutheastAsia:    {context: "Island and coastal routes add aquatic food\nand explicit water-passage constraints.", hint: "Cordage and navigation matter at named\nWallacea crossings."},
	gameapi.EastAsia:         {context: "East Asian dispersal spans tropical coasts,\ninteriors, and colder northern routes.", hint: "Keep adaptations and clothing aligned with\nthe route's changing pressures."},
	gameapi.YellowRiverBasin: {context: "The Yellow River basin is a distinct northern\nEast Asian destination.", hint: "Regional establishment remains independent\nof the route used to reach it."},
	gameapi.Sahul:            {context: "Reaching Sahul requires movement through the\nisland geography of Wallacea.", hint: "Only named passages cross open water; ordinary\nwater tiles remain impassable."},
	gameapi.Siberia:          {context: "Cold, low-flora habitats make animal foods,\nshelter, and clothing especially important.", hint: "Foraging potential can be low even when\nhunting opportunity remains useful."},
	gameapi.Beringia:         {context: "The Beringian gate responds to the full\nclimate function and may open repeatedly.", hint: "Inspect the current passage state rather\nthan assuming one fixed opening date."},
}

func CampaignOverviewFieldNote() render.FieldNote {
	return render.FieldNote{
		Topic:        "WELCOME",
		Introduction: "The campaign begins in East Africa.",
		Context:      "It is 80,000 years before present;\nthe map reveals as sapiens expand.",
		GameEffect:   "Outlined tiles are reachable; arrows\nchoose and Enter queues migration.",
		Hint:         "Archaic hominins—including a Tibetan\nDenisovan band—are computer-controlled.",
	}
}

func TechnologyFieldNote(technology gameapi.Tech, bandID gameapi.BandID, discoveries int) (render.FieldNote, bool) {
	if technology >= gameapi.TechCount {
		return render.FieldNote{}, false
	}
	entry := technologyFieldNotes[technology]
	introduction := fmt.Sprintf("Band %d learned %s!", bandID, technology)
	if discoveries > 1 {
		introduction = fmt.Sprintf("%d breakthroughs this turn.\nBand %d learned %s!", discoveries, bandID, technology)
	}
	return render.FieldNote{
		Topic:        technology.String(),
		Introduction: introduction,
		Context:      entry.context,
		GameEffect:   entry.gameEffect,
		Hint:         entry.hint,
	}, true
}

func TraitFieldNote(trait gameapi.HeritableTrait, value float64) (render.FieldNote, bool) {
	if trait >= gameapi.HeritableTraitCount {
		return render.FieldNote{}, false
	}
	entry := traitFieldNotes[trait]
	return render.FieldNote{
		Topic:        trait.String(),
		Introduction: fmt.Sprintf("Selected-band value: %.3f", value),
		Context:      entry.context,
		GameEffect:   entry.gameEffect,
		Hint:         entry.hint,
	}, true
}

func RegionEstablishedFieldNote(region gameapi.Region) (render.FieldNote, bool) {
	if region >= gameapi.RegionCount {
		return render.FieldNote{}, false
	}
	entry := regionFieldNotes[region]
	introduction := "Homo sapiens established " + region.String() + "."
	if region == gameapi.Arabia || region == gameapi.Levant {
		introduction += "\nA new founder population endures."
	}
	return render.FieldNote{
		Topic:        "REGION · " + region.String(),
		Introduction: introduction,
		Context:      entry.context,
		GameEffect:   "This route-neutral regional achievement\nremains latched for the campaign.",
		Hint:         entry.hint,
	}, true
}

func MacroEpisodeFieldNote(episode gameapi.MacroEpisodeSummary) (render.FieldNote, bool) {
	if episode.Episode != gameapi.CampanianIgnimbrite {
		return render.FieldNote{}, false
	}
	state := "elapsed"
	if episode.Warned {
		state = "warning"
	}
	if episode.Current {
		state = "active"
	}
	return render.FieldNote{
		Topic:        episode.Episode.String(),
		Introduction: "Regional volcanic episode: " + state + ".",
		Context:      "The Campanian Ignimbrite occurred about\n39,850 years before present.",
		GameEffect:   "The game uses a bounded regional impact\nenvelope, not literal demographic counts.",
		Hint:         "Warnings annotate explored destinations;\nthey never move a band automatically.",
	}, true
}

func TobaFieldNote() render.FieldNote {
	return render.FieldNote{
		Topic:        "TOBA · TIMELINE CONTEXT",
		Introduction: "The campaign has passed ~73,880 BP.",
		Context:      "Storey et al. (2012) date Toba; Lake\nMalawi shows no catastrophic winter.",
		GameEffect:   "Toba is a context marker only and has\nno effect on people, climate, or stock.",
		Hint:         "Lane et al. (2013) and Kappelman et\nal. (2024) argue against a simple collapse.",
	}
}

func ClimateEpochFieldNote(epoch gameapi.ClimateEpoch) (render.FieldNote, bool) {
	if epoch >= gameapi.ClimateEpochCount {
		return render.FieldNote{}, false
	}
	entry := [...]struct{ introduction, gameEffect, hint string }{
		gameapi.HumidOptimum: {
			introduction: "The moisture index is in its humid range.",
			gameEffect:   "The palette shifts greener; biome and\nresource rules still use continuous climate.",
			hint:         "Epoch names summarize the index and do\nnot impose a separate simulation phase.",
		},
		gameapi.AridTransition: {
			introduction: "The long drying trend is now visible.",
			gameEffect:   "The palette warms as regional moisture\nand biome boundaries continue to change.",
			hint:         "Abrupt pulses remain regional overlays,\nnot replacements for the long trend.",
		},
		gameapi.GlacialMaximum: {
			introduction: "The campaign has entered its driest epoch.",
			gameEffect:   "The palette cools; low vegetation and\ncold can independently constrain habitat.",
			hint:         "This compressed trend is a game model,\nnot a claim of uniform global aridity.",
		},
	}[epoch]
	return render.FieldNote{
		Topic:        "CLIMATE · " + epoch.String(),
		Introduction: entry.introduction,
		Context:      "MIS framework: Lisiecki & Raymo (2005);\nLGM definition: Clark et al. (2009).",
		GameEffect:   entry.gameEffect,
		Hint:         entry.hint,
	}, true
}

func BandContextFieldNote(frame *gameapi.Frame, band *gameapi.Band) render.FieldNote {
	if frame == nil || band == nil || int(band.TileID) >= len(frame.Tiles) {
		return CampaignOverviewFieldNote()
	}
	tile := frame.Tiles[band.TileID]
	return render.FieldNote{
		Topic:        fmt.Sprintf("BAND %d · %s", band.ID, tile.Region),
		Introduction: fmt.Sprintf("%d people · %.1f%% health · %.1f FU stored", band.Population, band.Health*100, band.StoredFood),
		Context:      fmt.Sprintf("The band occupies %s in the\n%s region.", tile.Biome, tile.Region),
		GameEffect:   fmt.Sprintf("Food %.0f/%.0f · water %.0f/%.0f\ncapacity %.0f · shelter %.0f%%", tile.FloraStock+tile.FaunaStock, tile.FloraCap+tile.FaunaCap, tile.WaterStock, tile.WaterCap, tile.EcologicalK, tile.NaturalShelter*100),
		Hint:         "Compare the cyan target inspector before\ncommitting a migration.",
	}
}

func EventFieldNote(event gameapi.Event) render.FieldNote {
	return render.FieldNote{
		Topic:        event.Kind.String(),
		Introduction: event.Summary,
		Context:      fmt.Sprintf("Recorded on turn %d for band %d.", event.Turn, event.BandID),
		GameEffect:   "The accepted frame already includes this\nevent's simulation consequences.",
		Hint:         "Review population, health, and mortality\nchanges in the selected-band panel.",
	}
}
