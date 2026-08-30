package logging

import "github.com/adsouza/africa2ice/pkg/gameapi"

type gameDecorator struct {
	session *Session
	next    gameapi.Game
}

func DecorateGame(session *Session, next gameapi.Game) gameapi.Game {
	if session == nil {
		return next
	}
	return &gameDecorator{session: session, next: next}
}

func (decorator *gameDecorator) Snapshot() (*gameapi.Frame, error) { return decorator.next.Snapshot() }

func (decorator *gameDecorator) NewCampaign() (*gameapi.Frame, error) {
	id, started := decorator.session.start("game", "new_campaign")
	frame, err := decorator.next.NewCampaign()
	attributes := []any{}
	if frame != nil {
		attributes = append(attributes, "turn", frame.Turn, "world_revision", frame.WorldRevision)
	}
	decorator.session.end(id, started, "game", "new_campaign", err, attributes...)
	return frame, err
}

func (decorator *gameDecorator) Apply(command gameapi.Command) (*gameapi.Frame, error) {
	id, started := decorator.session.start("game", "apply", "band_id", uint64(command.ActingBandID()))
	frame, err := decorator.next.Apply(command)
	attributes := []any{}
	if frame != nil {
		attributes = append(attributes, "turn", frame.Turn, "world_revision", frame.WorldRevision)
	}
	decorator.session.end(id, started, "game", "apply", err, attributes...)
	return frame, err
}

func (decorator *gameDecorator) EndTurn() (*gameapi.Frame, error) {
	id, started := decorator.session.start("game", "end_turn")
	frame, err := decorator.next.EndTurn()
	attributes := []any{}
	if frame != nil {
		attributes = append(attributes, "turn", frame.Turn, "world_revision", frame.WorldRevision)
	}
	decorator.session.end(id, started, "game", "end_turn", err, attributes...)
	return frame, err
}

func (decorator *gameDecorator) BeginSave(slot int) (gameapi.StorageOpID, error) {
	return decorator.beginStorage("save", slot, decorator.next.BeginSave)
}
func (decorator *gameDecorator) BeginLoad(slot int) (gameapi.StorageOpID, error) {
	return decorator.beginStorage("load", slot, decorator.next.BeginLoad)
}
func (decorator *gameDecorator) BeginDelete(slot int) (gameapi.StorageOpID, error) {
	return decorator.beginStorage("delete", slot, decorator.next.BeginDelete)
}

func (decorator *gameDecorator) BeginListSlots() (gameapi.StorageOpID, error) {
	id, started := decorator.session.start("game", "list_slots")
	operationID, err := decorator.next.BeginListSlots()
	decorator.session.end(id, started, "game", "list_slots", err, "storage_operation_id", uint64(operationID))
	return operationID, err
}

func (decorator *gameDecorator) beginStorage(name string, slot int, call func(int) (gameapi.StorageOpID, error)) (gameapi.StorageOpID, error) {
	id, started := decorator.session.start("game", name, "slot", slot)
	operationID, err := call(slot)
	decorator.session.end(id, started, "game", name, err, "storage_operation_id", uint64(operationID), "slot", slot)
	return operationID, err
}

func (decorator *gameDecorator) PollStorage() []gameapi.StorageResult {
	results := decorator.next.PollStorage()
	for _, result := range results {
		decorator.session.logger.Info("storage.completion", "storage_operation_id", uint64(result.OperationID), "operation", result.Operation, "slot", result.Slot, "error", result.Err)
	}
	return results
}

var _ gameapi.Game = (*gameDecorator)(nil)
