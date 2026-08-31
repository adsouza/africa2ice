package ui

import (
	"errors"
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// ActionKind identifies the closed set of intents that may cross from the UI
// into the host composition adapter. Pointer movement, drawing, and local
// workforce-draft edits deliberately have no ActionKind.
type ActionKind uint8

const (
	ActionInvalid ActionKind = iota
	ActionSimulationCommand
	ActionEndTurn
	ActionSave
	ActionLoad
	ActionDelete
	ActionListSlots
	ActionNavigate
)

func (kind ActionKind) String() string {
	switch kind {
	case ActionSimulationCommand:
		return "simulation_command"
	case ActionEndTurn:
		return "end_turn"
	case ActionSave:
		return "save"
	case ActionLoad:
		return "load"
	case ActionDelete:
		return "delete"
	case ActionListSlots:
		return "list_slots"
	case ActionNavigate:
		return "navigate"
	default:
		return "invalid"
	}
}

type ActionClass uint8

const (
	ActionClassInvalid ActionClass = iota
	ActionClassCampaign
	ActionClassStorage
	ActionClassNavigation
)

type NavigationKind uint8

const (
	NavigationInvalid NavigationKind = iota
	NavigationPush
	NavigationPop
	NavigationReset
)

// Action is a tagged value. Its fields are private so production callers can
// construct only payload shapes accepted by ValidateActionBatch.
type Action struct {
	kind       ActionKind
	command    gameapi.Command
	slot       int
	navigation NavigationKind
	scene      SceneID
}

func SimulationAction(command gameapi.Command) Action {
	return Action{kind: ActionSimulationCommand, command: command}
}

func EndTurnAction() Action { return Action{kind: ActionEndTurn} }

func SaveAction(slot int) Action   { return Action{kind: ActionSave, slot: slot} }
func LoadAction(slot int) Action   { return Action{kind: ActionLoad, slot: slot} }
func DeleteAction(slot int) Action { return Action{kind: ActionDelete, slot: slot} }
func ListSlotsAction() Action      { return Action{kind: ActionListSlots} }

func PushSceneAction(scene SceneID) Action {
	return Action{kind: ActionNavigate, navigation: NavigationPush, scene: scene}
}

func PopSceneAction() Action {
	return Action{kind: ActionNavigate, navigation: NavigationPop}
}

func ResetScenesAction() Action {
	return Action{kind: ActionNavigate, navigation: NavigationReset}
}

func (action Action) Kind() ActionKind                      { return action.kind }
func (action Action) Command() gameapi.Command              { return action.command }
func (action Action) Slot() int                             { return action.slot }
func (action Action) Navigation() (NavigationKind, SceneID) { return action.navigation, action.scene }

var ErrInvalidActionBatch = errors.New("invalid UI action batch")

// ValidateActionBatch performs atomic structural preflight. No host use case
// may be invoked unless this function accepts the complete batch.
func ValidateActionBatch(actions []Action) (ActionClass, error) {
	if len(actions) == 0 {
		return ActionClassInvalid, fmt.Errorf("%w: empty batch", ErrInvalidActionBatch)
	}
	class := classForAction(actions[0])
	if class == ActionClassInvalid {
		return class, fmt.Errorf("%w: action 0 has an invalid tag or payload", ErrInvalidActionBatch)
	}
	for index, action := range actions {
		if !validActionPayload(action) {
			return ActionClassInvalid, fmt.Errorf("%w: action %d has an invalid payload", ErrInvalidActionBatch, index)
		}
		if classForAction(action) != class {
			return ActionClassInvalid, fmt.Errorf("%w: action %d mixes action classes", ErrInvalidActionBatch, index)
		}
	}
	switch class {
	case ActionClassCampaign:
		seenEndTurn := false
		for index, action := range actions {
			if action.kind == ActionEndTurn {
				if seenEndTurn || index != len(actions)-1 {
					return ActionClassInvalid, fmt.Errorf("%w: EndTurn must appear at most once and last", ErrInvalidActionBatch)
				}
				seenEndTurn = true
			}
		}
	case ActionClassStorage, ActionClassNavigation:
		if len(actions) != 1 {
			return ActionClassInvalid, fmt.Errorf("%w: %s batch must contain exactly one action", ErrInvalidActionBatch, class)
		}
	}
	return class, nil
}

func (class ActionClass) String() string {
	switch class {
	case ActionClassCampaign:
		return "campaign"
	case ActionClassStorage:
		return "storage"
	case ActionClassNavigation:
		return "navigation"
	default:
		return "invalid"
	}
}

func classForAction(action Action) ActionClass {
	switch action.kind {
	case ActionSimulationCommand, ActionEndTurn:
		return ActionClassCampaign
	case ActionSave, ActionLoad, ActionDelete, ActionListSlots:
		return ActionClassStorage
	case ActionNavigate:
		return ActionClassNavigation
	default:
		return ActionClassInvalid
	}
}

func validActionPayload(action Action) bool {
	switch action.kind {
	case ActionSimulationCommand:
		return action.command != nil && action.slot == 0 && action.navigation == NavigationInvalid && action.scene == SceneGameplay
	case ActionEndTurn:
		return action.command == nil && action.slot == 0 && action.navigation == NavigationInvalid && action.scene == SceneGameplay
	case ActionSave, ActionLoad, ActionDelete:
		return action.command == nil && action.slot > 0 && action.navigation == NavigationInvalid && action.scene == SceneGameplay
	case ActionListSlots:
		return action.command == nil && action.slot == 0 && action.navigation == NavigationInvalid && action.scene == SceneGameplay
	case ActionNavigate:
		if action.command != nil || action.slot != 0 {
			return false
		}
		switch action.navigation {
		case NavigationPush:
			return action.scene > SceneGameplay && action.scene <= SceneSettings
		case NavigationPop, NavigationReset:
			return action.scene == SceneGameplay
		default:
			return false
		}
	default:
		return false
	}
}
