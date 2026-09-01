package ui

import (
	"errors"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestValidateActionBatchAcceptsEveryClosedClass(t *testing.T) {
	tests := []struct {
		name    string
		actions []Action
		want    ActionClass
	}{
		{name: "commands", actions: []Action{SimulationAction(gameapi.ResearchTech{BandID: 1, Tech: gameapi.Firecraft}), SimulationAction(gameapi.QueueMigration{BandID: 1, TileID: 2})}, want: ActionClassCampaign},
		{name: "commands and turn", actions: []Action{SimulationAction(gameapi.ResearchTech{BandID: 1, Tech: gameapi.Firecraft}), EndTurnAction()}, want: ActionClassCampaign},
		{name: "turn only", actions: []Action{EndTurnAction()}, want: ActionClassCampaign},
		{name: "save", actions: []Action{SaveAction(1)}, want: ActionClassStorage},
		{name: "load", actions: []Action{LoadAction(99)}, want: ActionClassStorage},
		{name: "delete", actions: []Action{DeleteAction(101)}, want: ActionClassStorage},
		{name: "list", actions: []Action{ListSlotsAction()}, want: ActionClassStorage},
		{name: "navigation", actions: []Action{PushSceneAction(SceneMenu)}, want: ActionClassNavigation},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ValidateActionBatch(test.actions)
			if err != nil || got != test.want {
				t.Fatalf("ValidateActionBatch() = (%v, %v), want (%v, nil)", got, err, test.want)
			}
		})
	}
}

func TestValidateActionBatchRejectsEveryForbiddenMixtureAndOrdering(t *testing.T) {
	tests := map[string][]Action{
		"empty":                    nil,
		"invalid tag":              {{}},
		"campaign plus storage":    {SimulationAction(gameapi.ResearchTech{BandID: 1}), SaveAction(1)},
		"campaign plus navigation": {EndTurnAction(), PushSceneAction(SceneMenu)},
		"storage plus navigation":  {LoadAction(1), PopSceneAction()},
		"two storage":              {SaveAction(1), LoadAction(1)},
		"two navigation":           {PushSceneAction(SceneMenu), PopSceneAction()},
		"end turn not last":        {EndTurnAction(), SimulationAction(gameapi.ResearchTech{BandID: 1})},
		"two end turns":            {EndTurnAction(), EndTurnAction()},
		"bad storage slot":         {SaveAction(0)},
		"bad navigation target":    {PushSceneAction(SceneGameplay)},
	}
	for name, actions := range tests {
		t.Run(name, func(t *testing.T) {
			if class, err := ValidateActionBatch(actions); class != ActionClassInvalid || !errors.Is(err, ErrInvalidActionBatch) {
				t.Fatalf("ValidateActionBatch() = (%v, %v), want invalid batch", class, err)
			}
		})
	}
}
