package ui

import (
	"fmt"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

type AcuteContext uint8

const (
	AcutePredationContext AcuteContext = iota
	AcuteDiseaseOutbreakContext
	AcuteFloodStormContext
	AcuteExposureFallContext
	AcuteCrossingMishapContext
	AcuteContextCount
)

var acuteEventClassifiers = [...]struct {
	fragment string
	context  AcuteContext
}{
	{fragment: "predation", context: AcutePredationContext},
	{fragment: "disease outbreak", context: AcuteDiseaseOutbreakContext},
	{fragment: "flood or storm", context: AcuteFloodStormContext},
	{fragment: "exposure or fall", context: AcuteExposureFallContext},
	{fragment: "crossing mishap", context: AcuteCrossingMishapContext},
}

func AcuteIncidentFieldNote(kind AcuteContext) (render.FieldNote, bool) {
	if kind >= AcuteContextCount {
		return render.FieldNote{}, false
	}
	entry := [...]struct{ topic, context, effect, hint, references string }{
		{"PREDATION", "Large predators and dangerous prey made subsistence work acutely hazardous.", "A bounded incident can remove people; shelter security, tools, trapping, and work choices modify risk multiplicatively.", "A low displayed chronic rate does not rule out an acute event on a later turn.", "Game hazard synthesis; sources in DESIGN §7."},
		{"DISEASE OUTBREAK", "Close contact, water, food, and camp conditions can amplify infectious disease.", "One shared draw can cause direct deaths and health loss; immunity, medicine, and hygiene affect different factors.", "Camp care reduces probability but never guarantees prevention.", "Game disease synthesis; sources in DESIGN §7."},
		{"FLOOD OR STORM", "River and coastal opportunity also carries episodic flood and storm exposure.", "A bounded incident can cause direct mortality according to biome, season, and uncovered protection.", "Natural caves are not a universal storm shield; inspect the modeled shelter and biome.", "Game hazard synthesis; sources in DESIGN §7."},
		{"EXPOSURE OR FALL", "Cold, heat, mountains, and travel create acute exposure and fall hazards.", "Clothing, firecraft, campcraft, shelter work, and adaptation reduce components without adding them together.", "Highland and glacial routes need both habitat capacity and risk preparation.", "Game hazard synthesis; sources in DESIGN §7."},
		{"CROSSING MISHAP", "Open-water crossings require craft, route knowledge, and coordinated movement.", "This risk activates only on a named passage and is resolved as a bounded incident.", "An open passage is usable, not safe; retain a viable founder population before crossing.", "O'Connor et al. (2011); game passage abstraction."},
	}[kind]
	return render.FieldNote{Topic: "ACUTE EVENT · " + entry.topic, Introduction: "This is a possible acute incident class.", Context: entry.context, GameEffect: entry.effect, Hint: entry.hint, References: entry.references}, true
}

var workforceFieldNotes = [gameapi.AssignmentCount]struct {
	gameEffect string
	hint       string
}{
	gameapi.Foraging: {
		gameEffect: "Foragers create potential plant-food yield; shared flora scarcity still limits harvest.",
		hint:       "Most useful where the tile's forageable plant-food stock and vegetation are high.",
	},
	gameapi.HuntingAndFishing: {
		gameEffect: "Hunters and fishers create potential animal yield from one shared fauna stock.",
		hint:       "Fauna weights are opportunities, not prey population counts or guaranteed catches.",
	},
	gameapi.Toolcraft: {
		gameEffect: "Toolcraft workers advance the selected research target with diminishing returns.",
		hint:       "Choose a target with keys 1–9; the DAG shows cost, progress, and prerequisites.",
	},
	gameapi.MegafaunaTracking: {
		gameEffect: "Trackers create high terrestrial hunting potential only where megafauna are supported.",
		hint:       "An unsupported allocation remains explicit; the game never silently reassigns it.",
	},
	gameapi.Shelter: {
		gameEffect: "Shelter and camp care reduce exposure, disease, predation, and incident pressure.",
		hint:       "Natural caves reduce exposure labor only; mitigations multiply and never erase all risk.",
	},
}

func WorkforceRoleFieldNote(role gameapi.WorkforceRole) (render.FieldNote, bool) {
	if role >= gameapi.AssignmentCount {
		return render.FieldNote{}, false
	}
	entry := workforceFieldNotes[role]
	return render.FieldNote{
		Topic:        "WORKFORCE · " + role.String(),
		Introduction: "A workforce share is a plan, not an outcome percentage.",
		Context:      "Bands combine learned skills with local resources and environmental limits.",
		GameEffect:   entry.gameEffect,
		Hint:         entry.hint,
		References:   "Game model; see DESIGN §7.",
	}, true
}

var technologyFieldNotes = [gameapi.TechCount]struct {
	context, gameEffect, hint string
}{
	gameapi.Firecraft: {
		context:    "Controlled fire supported warmth, cooking, light, and safer camps.",
		gameEffect: "Capacity rises; exposure and some predation risks fall.",
		hint:       "Campcraft and Medicinal Knowledge both build on Firecraft.",
	},
	gameapi.HaftedTools: {
		context:    "Binding worked stone to shafts made stronger, more versatile tools.",
		gameEffect: "Hunting improves; some predation risks and injuries become less severe.",
		hint:       "It unlocks clothing, cordage, and —with Firecraft—Campcraft.",
	},
	gameapi.PlantKnowledge: {
		context:    "Foraging relies on learned seasonal knowledge of edible plants.",
		gameEffect: "Plant-food collection and ecological capacity both improve.",
		hint:       "Combine it with Firecraft to unlock Medicinal Knowledge.",
	},
	gameapi.TailoredClothing: {
		context:    "Fitted hide garments protect more effectively than loosely draped skins.",
		gameEffect: "Capacity rises and exposure risk falls substantially.",
		hint:       "This is especially useful as bands enter colder regions.",
	},
	gameapi.CordageAndNets: {
		context:    "Katanda points/fish date to ~90 ka; Jerimalai pelagic catch to ~42 ka.",
		gameEffect: "Inshore fishing improves and pelagic resources become accessible.",
		hint:       "Shell hooks date to 23–16 ka; Lake Condah stone traps are post-campaign.",
	},
	gameapi.Campcraft: {
		context:    "Organized shelter, hearth, food, and waste practices make camps safer.",
		gameEffect: "Capacity rises; exposure, disease, and some acute incident risks fall.",
		hint:       "Shelter workers compound these gains without making any risk vanish.",
	},
	gameapi.MedicinalKnowledge: {
		context:    "Caregiving and accumulated plant knowledge can ease illness burdens.",
		gameEffect: "Disease risk, health damage, and the severity of outbreaks are reduced.",
		hint:       "It mitigates disease; it does not prevent every outbreak.",
	},
	gameapi.Trapping: {
		context:    "Snares and traps exchange direct pursuit for planning and patience.",
		gameEffect: "Small and medium game yields improve; some predation risk also falls.",
		hint:       "It is most valuable where suitable terrestrial fauna are present.",
	},
	gameapi.CoastalNavigation: {
		context:    "Open-water travel needs craft, route knowledge, and group coordination.",
		gameEffect: "Pelagic hunting improves and named Wallacea passages become usable.",
		hint:       "Move to a passage endpoint; arbitrary water tiles remain impassable.",
	},
}

var traitFieldNotes = [gameapi.HeritableTraitCount]struct {
	context, gameEffect, hint string
}{
	gameapi.ColdAdaptation: {
		context:    "Heritable cold responses can shift over many generations under local selection.",
		gameEffect: "Higher values reduce cold exposure and support life in glacial environments.",
		hint:       "The value is inherited and exchanged by gene flow; it is not a technology.",
	},
	gameapi.HighAltitudeAdaptation: {
		context:    "High-altitude populations can accumulate physiological responses to low oxygen.",
		gameEffect: "Higher values reduce hypoxia pressure in mountainous highlands.",
		hint:       "Selection is strongest where elevation and occupancy keep the pressure active.",
	},
	gameapi.InnateImmuneReactivity: {
		context:    "Immune responses trade pathogen defence against damaging overreaction.",
		gameEffect: "Local disease pressure selects the trait; extreme values carry their own burden.",
		hint:       "Medicine and camp hygiene remain separate, learned layers of protection.",
	},
	gameapi.AridClimateAdaptation: {
		context:    "Heat and water scarcity create persistent selection in arid regions.",
		gameEffect: "Higher values reduce heat and water stress without creating water or food.",
		hint:       "Compare water stocks and demand before a desert migration.",
	},
	gameapi.PigmentationLevel: {
		context:    "Pigmentation balances ultraviolet skin protection against vitamin-D synthesis.",
		gameEffect: "Latitude-dependent UV pressure favours different values in different regions.",
		hint:       "Neither end is universally best; movement can reverse the local pressure.",
	},
	gameapi.FattyAcidMetabolism: {
		context:    "Dietary fat use varies heritably and can be selected where animal foods dominate.",
		gameEffect: "The value changes usable yield from animal and aquatic food sources.",
		hint:       "It changes conversion, not the number of animals represented by fauna stock.",
	},
}

var regionFieldNotes = [gameapi.RegionCount]struct {
	context, hint string
}{
	gameapi.EastAfrica:       {context: "The campaign's sapiens bands begin here amid riverine and savanna habitats.", hint: "Build resilient bands before committing to longer dispersal routes."},
	gameapi.RestOfAfrica:     {context: "Africa contains several viable corridors, not a single departure route.", hint: "Regional establishment rewards breadth, not one prescribed historical path."},
	gameapi.Arabia:           {context: "Genetic reconstructions put the effective founding dispersal in the low thousands.", hint: "A viable founder band must retain enough people after movement and establishment."},
	gameapi.Levant:           {context: "The Levant repeatedly connected African and Eurasian populations.", hint: "This corridor can be useful without being the only route out of Africa."},
	gameapi.Frangistan:       {context: "This broad western-Eurasian region is a destination, not a privileged win route.", hint: "Establishment is one regional achievement among several route-neutral goals."},
	gameapi.CentralAsia:      {context: "Interior moisture and temperature shifts can open and close steppe-like opportunities.", hint: "Watch water, exposure, and seasonal food rather than relying on colour alone."},
	gameapi.SouthAsia:        {context: "South Asia links western and eastern routes across diverse monsoon habitats.", hint: "Its flora, fauna, and disease mix differs substantially by biome."},
	gameapi.SoutheastAsia:    {context: "Island and coastal routes add aquatic food and explicit water-passage constraints.", hint: "Cordage and navigation matter at named Wallacea crossings."},
	gameapi.EastAsia:         {context: "East Asian dispersal spans tropical coasts, interiors, and colder northern routes.", hint: "Keep adaptations and clothing aligned with the route's changing pressures."},
	gameapi.YellowRiverBasin: {context: "The Yellow River basin is a distinct northern East Asian destination.", hint: "Regional establishment remains independent of the route used to reach it."},
	gameapi.Sahul:            {context: "Reaching Sahul requires movement through the island geography of Wallacea.", hint: "Only named passages cross open water; ordinary water tiles remain impassable."},
	gameapi.Siberia:          {context: "Cold, low-flora habitats make animal foods, shelter, and clothing especially important.", hint: "Foraging potential can be low even when hunting opportunity remains useful."},
	gameapi.Beringia:         {context: "The Beringian gate responds to the full climate function and may open repeatedly.", hint: "Inspect the current passage state rather than assuming one fixed opening date."},
}

var biomeFieldNotes = [gameapi.BiomeCount]struct {
	context, gameEffect, hint, references string
}{
	gameapi.RiverineWoodland: {
		context:    "Rivers concentrate water, plant foods, and animals, while also concentrating pathogens.",
		gameEffect: "High forage opportunity and water can pair with elevated disease pressure.",
		hint:       "Camp care and medicine mitigate disease; they do not manufacture food or water.", references: "Game ecology synthesis; sources in DESIGN §7.",
	},
	gameapi.Savanna: {
		context:    "Open grassland mosaics support mixed plant and terrestrial-animal opportunities.",
		gameEffect: "A balanced biome whose usefulness shifts with moisture, season, and fauna mix.",
		hint:       "Compare current stocks rather than assuming the greenest-looking tile is best.", references: "Game ecology synthesis; sources in DESIGN §7.",
	},
	gameapi.CoastalShrubland: {
		context:    "Coastal settings can combine terrestrial and aquatic resources with storm exposure.",
		gameEffect: "Inshore opportunity is baseline; pelagic use and passages need later capabilities.",
		hint:       "Cordage and Nets improves aquatic use, but ordinary open water remains uninhabitable.", references: "O'Connor et al. (2011); Yellen et al. (1995).",
	},
	gameapi.MountainousHighlands: {
		context:    "Elevation changes temperature, oxygen, travel, and local resource capacity.",
		gameEffect: "Highlands carry hypoxia, fall, and exposure pressure plus slower movement.",
		hint:       "High-altitude adaptation and shelter help, but steep authored escarpments still block entry.", references: "Game highland abstraction; sources in DESIGN §6.",
	},
	gameapi.SemiAridDesert: {
		context:    "Low forageable vegetation and scarce water make arid routes sensitive to timing.",
		gameEffect: "Low capacity and heat/water stress can make stored food alone insufficient.",
		hint:       "Inspect both water and food before moving; arid adaptation reduces stress, not scarcity.", references: "Game climate synthesis; sources in DESIGN §7.",
	},
	gameapi.GlacialTundra: {
		context:    "Cold low-vegetation landscapes can offer more animal than plant-food opportunity.",
		gameEffect: "Foraging is weak while hunting, clothing, shelter, and cold adaptation gain importance.",
		hint:       "A low flora stock does not imply an empty fauna stock; read the tile's prey opportunity.", references: "Clark et al. (2009); game ecology abstraction.",
	},
}

func BiomeFieldNote(biome gameapi.Biome) (render.FieldNote, bool) {
	if biome >= gameapi.BiomeCount {
		return render.FieldNote{}, false
	}
	entry := biomeFieldNotes[biome]
	return render.FieldNote{Topic: "BIOME · " + biome.String(), Introduction: "Environmental opportunities and hazards are continuous beneath this label.", Context: entry.context, GameEffect: entry.gameEffect, Hint: entry.hint, References: entry.references}, true
}

func PassageFieldNote(passage gameapi.PassageID, status gameapi.PassageStatus) (render.FieldNote, bool) {
	if passage >= gameapi.PassageCount {
		return render.FieldNote{}, false
	}
	context := [...]string{
		"Northern Wallacea represents one modeled island-hopping corridor toward Sahul.",
		"Southern Wallacea represents a second modeled island-hopping corridor toward Sahul.",
		"Beringia responds to the full climate function and can open more than once.",
	}[passage]
	references := "O'Connor et al. (2011); Clarkson et al. (2017)."
	if passage == gameapi.BeringStrait {
		references = "Clark et al. (2009); game sea-level abstraction."
	}
	return render.FieldNote{
		Topic: "PASSAGE · " + passage.String(), Introduction: "Current status: " + status.String() + ".",
		Context: context, GameEffect: "Only named endpoints can cross open water; the destination cost and capability gates still apply.",
		Hint: "Select an endpoint and inspect the target status before committing the band's spatial action.", References: references,
	}, true
}

func SpeciesFieldNote(species gameapi.Species) (render.FieldNote, bool) {
	if species >= gameapi.SpeciesCount {
		return render.FieldNote{}, false
	}
	if species == gameapi.HomoSapiens {
		return render.FieldNote{Topic: species.String(), Introduction: "The player directs only Homo sapiens bands.", Context: "The campaign models multiple dispersing bands rather than a single species-wide population.", GameEffect: "Each band owns its food, health, workforce, technology, genetics, and spatial action.", Hint: "Tab and Shift+Tab cycle player bands; click visible markers to inspect any resident band.", References: "Game population abstraction; dispersal sources in DESIGN §1."}, true
	}
	return render.FieldNote{Topic: species.String(), Introduction: "Archaic bands are computer controlled and inspectable when explored.", Context: "Neanderthal- and Denisovan-related populations contributed ancestry to later human populations.", GameEffect: "Archaic bands survive, move, research, and adapt independently; co-location alone transfers no genes.", Hint: "A sapiens band sharing a tile may explicitly choose interbreeding if its spatial action remains.", References: "Reich et al. (2010); Chen et al. (2019)."}, true
}

func InterbreedingFieldNote(candidateCount int) render.FieldNote {
	return render.FieldNote{
		Topic: "INTERBREEDING", Introduction: fmt.Sprintf("%d eligible archaic band(s) share this tile.", candidateCount),
		Context:    "Archaic admixture contributed inherited variants to some later human populations.",
		GameEffect: "Interbreeding is an explicit spatial action that exchanges the whole modeled heritable vector reciprocally; ordinary co-location does nothing automatically.",
		Hint:       "Press J to cycle eligible targets and I to accept the highlighted one; migration and splitting then remain unavailable until next turn.",
		References: "Reich et al. (2010); Dannemann et al. (2016).",
	}
}

func EventKindFieldNote(kind gameapi.EventKind) (render.FieldNote, bool) {
	if kind >= gameapi.EventKindCount {
		return render.FieldNote{}, false
	}
	return render.FieldNote{Topic: "EVENT · " + kind.String(), Introduction: "Events summarize accepted changes in the bounded campaign feed.", Context: "Historical evidence rarely resolves a single band's turn-by-turn experience.", GameEffect: "The event text reports a modeled outcome; the accepted frame contains its actual mechanical consequences.", Hint: "Inspect the selected band's last-turn food, mortality, health, and population reports.", References: "Game abstraction; event-specific context in DESIGN §7."}, true
}

func AbruptClimateFieldNote(region gameapi.Region, magnitude float64) (render.FieldNote, bool) {
	if region >= gameapi.RegionCount {
		return render.FieldNote{}, false
	}
	return render.FieldNote{Topic: "REGIONAL CLIMATE PULSE · " + region.String(), Introduction: fmt.Sprintf("Current explored regional anomaly: %+.3f.", magnitude), Context: "Last-glacial abrupt warmings varied in timing, shape, and regional expression.", GameEffect: "A deterministic regional overlay temporarily changes moisture; it is not a separate campaign era or future-event forecast.", Hint: "Compare the same explored tile over later turns; the ecology can recover after the pulse passes.", References: "Rasmussen et al. (2014); Capron et al. (2021)."}, true
}

func CampaignOverviewFieldNote() render.FieldNote {
	return render.FieldNote{
		Topic:        "WELCOME",
		Introduction: "The campaign begins in East Africa.",
		Context:      "It is 80,000 years before present; the map reveals as sapiens expand.",
		GameEffect:   "Outlined tiles are reachable; arrows choose and Enter queues migration.",
		Hint:         "Archaic hominins—including a Tibetan Denisovan band—are computer-controlled.",
		References:   "Reich et al. (2010); Chen et al. (2019).",
	}
}

func TechnologyFieldNote(technology gameapi.Tech, bandID gameapi.BandID, discoveries int) (render.FieldNote, bool) {
	if technology >= gameapi.TechCount {
		return render.FieldNote{}, false
	}
	entry := technologyFieldNotes[technology]
	introduction := fmt.Sprintf("Band %d learned %s!", bandID, technology)
	if discoveries > 1 {
		introduction = fmt.Sprintf("%d breakthroughs this turn. Band %d learned %s!", discoveries, bandID, technology)
	}
	return render.FieldNote{
		Topic:        technology.String(),
		Introduction: introduction,
		Context:      entry.context,
		GameEffect:   entry.gameEffect,
		Hint:         entry.hint,
		References:   technologyReferences(technology),
	}, true
}

func TechnologyContextFieldNote(technology gameapi.Tech, band *gameapi.Band) (render.FieldNote, bool) {
	if band == nil {
		return render.FieldNote{}, false
	}
	note, ok := TechnologyFieldNote(technology, band.ID, 1)
	if !ok {
		return render.FieldNote{}, false
	}
	option := band.ResearchOptions[technology]
	state := fmt.Sprintf("Progress %.1f/%.0f.", band.ResearchProgress[technology], option.Cost)
	switch {
	case option.Acquired:
		state += " Learned by this band."
	case option.Current:
		state += " This band's active target."
	case option.Available:
		state += " Available to select."
	default:
		state += " Locked by direct prerequisites."
	}
	note.Introduction = state
	return note, true
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
		References:   traitReferences(trait),
	}, true
}

func RegionEstablishedFieldNote(region gameapi.Region) (render.FieldNote, bool) {
	if region >= gameapi.RegionCount {
		return render.FieldNote{}, false
	}
	entry := regionFieldNotes[region]
	introduction := "Homo sapiens established " + region.String() + "."
	if region == gameapi.Arabia || region == gameapi.Levant {
		introduction += " A new founder population endures."
	}
	return render.FieldNote{
		Topic:        "REGION · " + region.String(),
		Introduction: introduction,
		Context:      entry.context,
		GameEffect:   "This route-neutral regional achievement remains latched for the campaign.",
		Hint:         entry.hint,
		References:   regionReferences(region),
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
		Context:      "The Campanian Ignimbrite occurred about 39,850 years before present.",
		GameEffect:   "The game uses a bounded regional impact envelope, not literal demographic counts.",
		Hint:         "Warnings annotate explored destinations; they never move a band automatically.",
		References:   "Giaccio et al. (2017); Scarpati et al. (2020); USGS.",
	}, true
}

func TobaFieldNote() render.FieldNote {
	return render.FieldNote{
		Topic:        "TOBA · TIMELINE CONTEXT",
		Introduction: "The campaign has passed ~73,880 BP.",
		Context:      "Storey et al. (2012) date Toba; Lake Malawi shows no catastrophic winter.",
		GameEffect:   "Toba is a context marker only and has no effect on people, climate, or stock.",
		Hint:         "Lane et al. (2013) and Kappelman et al. (2024) argue against a simple collapse.",
		References:   "Storey et al. (2012); Lane et al. (2013); Kappelman et al. (2024).",
	}
}

func ClimateEpochFieldNote(epoch gameapi.ClimateEpoch) (render.FieldNote, bool) {
	if epoch >= gameapi.ClimateEpochCount {
		return render.FieldNote{}, false
	}
	entry := [...]struct{ introduction, gameEffect, hint string }{
		gameapi.HumidOptimum: {
			introduction: "The moisture index is in its humid range.",
			gameEffect:   "The palette shifts greener; biome and resource rules still use continuous climate.",
			hint:         "Epoch names summarize the index and do not impose a separate simulation phase.",
		},
		gameapi.AridTransition: {
			introduction: "The long drying trend is now visible.",
			gameEffect:   "The palette warms as regional moisture and biome boundaries continue to change.",
			hint:         "Abrupt pulses remain regional overlays, not replacements for the long trend.",
		},
		gameapi.GlacialMaximum: {
			introduction: "The campaign has entered its driest epoch.",
			gameEffect:   "The palette cools; low vegetation and cold can independently constrain habitat.",
			hint:         "This compressed trend is a game model, not a claim of uniform global aridity.",
		},
	}[epoch]
	return render.FieldNote{
		Topic:        "CLIMATE · " + epoch.String(),
		Introduction: entry.introduction,
		Context:      "MIS framework: Lisiecki & Raymo (2005); LGM definition: Clark et al. (2009).",
		GameEffect:   entry.gameEffect,
		Hint:         entry.hint,
		References:   "Lisiecki & Raymo (2005); Clark et al. (2009); Capron et al. (2021).",
	}, true
}

func BandContextFieldNote(frame *gameapi.Frame, band *gameapi.Band) render.FieldNote {
	if frame == nil || band == nil || int(band.TileID) >= len(frame.Tiles) {
		return CampaignOverviewFieldNote()
	}
	tile := frame.Tiles[band.TileID]
	control := "Player controlled"
	if band.Species == gameapi.ArchaicHominin {
		control = "Computer controlled · read only"
	}
	return render.FieldNote{
		Topic:        fmt.Sprintf("BAND %d · %s", band.ID, tile.Region),
		Introduction: fmt.Sprintf("%s. %d people · %.1f%% health · %.1f FU stored", control, band.Population, band.Health*100, band.StoredFood),
		Context:      fmt.Sprintf("The band occupies %s in the %s region.", tile.Biome, tile.Region),
		GameEffect:   fmt.Sprintf("Food %.0f/%.0f · water %.0f/%.0f capacity %.0f · shelter %.0f%%", tile.FloraStock+tile.FaunaStock, tile.FloraCap+tile.FaunaCap, tile.WaterStock, tile.WaterCap, tile.EcologicalK, tile.NaturalShelter*100),
		Hint:         "Compare the cyan target inspector before committing a migration.",
		References:   "Game abstraction; regional sources in DESIGN §6.",
	}
}

func EventFieldNote(event gameapi.Event) render.FieldNote {
	if event.Kind == gameapi.EventAcuteIncident {
		summary := strings.ToLower(event.Summary)
		for _, classifier := range acuteEventClassifiers {
			if strings.Contains(summary, classifier.fragment) {
				note, _ := AcuteIncidentFieldNote(classifier.context)
				note.Introduction = event.Summary
				return note
			}
		}
	}
	return render.FieldNote{
		Topic:        event.Kind.String(),
		Introduction: event.Summary,
		Context:      fmt.Sprintf("Recorded on turn %d for band %d.", event.Turn, event.BandID),
		GameEffect:   "The accepted frame already includes this event's simulation consequences.",
		Hint:         "Review population, health, and mortality changes in the selected-band panel.",
		References:   "Game event record; evidence notes vary by event.",
	}
}

func technologyReferences(technology gameapi.Tech) string {
	switch technology {
	case gameapi.CordageAndNets, gameapi.Trapping, gameapi.CoastalNavigation:
		return "Yellen et al. (1995); O'Connor et al. (2011); McNiven et al. (2012)."
	default:
		return "Game capability bundle; archaeological context in DESIGN §7."
	}
}

func traitReferences(trait gameapi.HeritableTrait) string {
	switch trait {
	case gameapi.PigmentationLevel:
		return "Jablonski & Chaplin (2010)."
	case gameapi.InnateImmuneReactivity:
		return "Dannemann et al. (2016)."
	default:
		return "Game-model synthesis; scientific context in DESIGN §7."
	}
}

func regionReferences(region gameapi.Region) string {
	switch region {
	case gameapi.Sahul, gameapi.SoutheastAsia:
		return "O'Connor et al. (2011); Clarkson et al. (2017)."
	case gameapi.Beringia, gameapi.Siberia:
		return "Clark et al. (2009); game geography abstraction."
	default:
		return "Regional dispersal synthesis; sources in DESIGN §6."
	}
}
