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
		context:    "Twisted fibers make reusable lines,\nbindings, nets, and carrying gear.",
		gameEffect: "Inshore fishing improves and pelagic\nresources become accessible.",
		hint:       "It unlocks Trapping and is required\nfor Coastal Navigation.",
	},
	gameapi.Campcraft: {
		context:    "Organized shelter, hearth, food, and\nwaste practices make camps safer.",
		gameEffect: "Capacity rises; exposure, disease,\nand camp-predation risks fall.",
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
