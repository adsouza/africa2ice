package logging

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

type bufferCloser struct{ bytes.Buffer }

func (buffer *bufferCloser) Close() error { return nil }

type countingGame struct {
	apply, end, snapshots int
	newCampaign           int
	frame                 gameapi.Frame
	err                   error
}

func (game *countingGame) Snapshot() (*gameapi.Frame, error) {
	game.snapshots++
	frame := game.frame
	return &frame, game.err
}

func (*countingGame) StateHash() (string, error) { return "test-hash", nil }

func (game *countingGame) NewCampaign() (*gameapi.Frame, error) {
	game.newCampaign++
	frame := game.frame
	return &frame, game.err
}
func (game *countingGame) Apply(gameapi.Command) (*gameapi.Frame, error) {
	game.apply++
	frame := game.frame
	return &frame, game.err
}
func (game *countingGame) EndTurn() (*gameapi.Frame, error) {
	game.end++
	frame := game.frame
	return &frame, game.err
}
func (*countingGame) BeginSave(int) (gameapi.StorageOpID, error)   { return 1, nil }
func (*countingGame) BeginLoad(int) (gameapi.StorageOpID, error)   { return 2, nil }
func (*countingGame) BeginDelete(int) (gameapi.StorageOpID, error) { return 3, nil }
func (*countingGame) BeginListSlots() (gameapi.StorageOpID, error) { return 4, nil }
func (*countingGame) PollStorage() []gameapi.StorageResult         { return nil }

func TestGameDecoratorCallsExactlyOnceAndDoesNotSnapshot(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("x", 16)), "test", output)
	next := &countingGame{frame: gameapi.Frame{Turn: 2, WorldRevision: 3}}
	decorated := DecorateGame(session, next)
	frame, err := decorated.Apply(gameapi.SetAssignment{BandID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if frame.Turn != 2 || next.apply != 1 || next.snapshots != 0 {
		t.Fatalf("decorator changed call behavior: %#v %#v", frame, next)
	}
	if !strings.Contains(output.String(), "operation.start") || !strings.Contains(output.String(), "operation.end") {
		t.Fatal("operation correlation not logged")
	}
	_ = session.Close()
}

func TestDecoratorPreservesError(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("y", 16)), "test", output)
	want := errors.New("boom")
	next := &countingGame{err: want}
	_, got := DecorateGame(session, next).EndTurn()
	if !errors.Is(got, want) || next.end != 1 {
		t.Fatalf("error/call changed: %v %#v", got, next)
	}
	_ = session.Close()
}

type countingSettingsStore struct {
	reads, writes, polls int
	completions          []ui.UISettingsCompletion
}

func (store *countingSettingsStore) BeginRead(uint64) error {
	store.reads++
	return nil
}

func (store *countingSettingsStore) BeginWrite(uint64, ui.UISettings) error {
	store.writes++
	return nil
}

func (store *countingSettingsStore) Poll() []ui.UISettingsCompletion {
	store.polls++
	return store.completions
}

func TestSettingsDecoratorCallsOnceAndDoesNotLogPreferenceValues(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("z", 16)), "test", output)
	settings := ui.UISettings{SchemaVersion: 1, FieldNotesVisible: false, MasterVolume: 0.731, Muted: true}
	next := &countingSettingsStore{completions: []ui.UISettingsCompletion{{Operation: ui.UISettingsWrite, Revision: 4, Settings: settings}}}
	decorated := DecorateUISettingsStore(session, next)
	if err := decorated.BeginRead(1); err != nil {
		t.Fatal(err)
	}
	if err := decorated.BeginWrite(4, settings); err != nil {
		t.Fatal(err)
	}
	if got := decorated.Poll(); len(got) != 1 || got[0].Settings != settings {
		t.Fatalf("decorated completions = %#v", got)
	}
	if next.reads != 1 || next.writes != 1 || next.polls != 1 {
		t.Fatalf("wrapped calls = reads %d writes %d polls %d", next.reads, next.writes, next.polls)
	}
	logged := output.String()
	if strings.Contains(logged, "0.731") || strings.Contains(logged, "FieldNotesVisible") || strings.Contains(logged, "Muted") {
		t.Fatalf("settings values leaked into log: %s", logged)
	}
	if !strings.Contains(logged, "settings.completion") || !strings.Contains(logged, "settings_revision") {
		t.Fatalf("settings lifecycle missing from log: %s", logged)
	}
	_ = session.Close()
}

type countingRepository struct {
	writes, reads, deletes, lists, polls, closes int
	completions                                  []application.RepositoryCompletion
}

func (repository *countingRepository) BeginWrite(application.RepositoryOpID, application.SlotID, application.SaveState) error {
	repository.writes++
	return nil
}
func (repository *countingRepository) BeginRead(application.RepositoryOpID, application.SlotID) error {
	repository.reads++
	return nil
}
func (repository *countingRepository) BeginDelete(application.RepositoryOpID, application.SlotID) error {
	repository.deletes++
	return nil
}
func (repository *countingRepository) BeginList(application.RepositoryOpID) error {
	repository.lists++
	return nil
}
func (repository *countingRepository) Poll() []application.RepositoryCompletion {
	repository.polls++
	return repository.completions
}
func (*countingRepository) Writable() bool { return true }
func (repository *countingRepository) Close() error {
	repository.closes++
	return nil
}

func TestRepositoryDecoratorCallsOnceWithoutLoggingSavePayload(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("r", 16)), "test", output)
	next := &countingRepository{completions: []application.RepositoryCompletion{{OperationID: 7, Operation: application.RepositoryWrite, SlotID: application.QuickSave}}}
	decorated := DecorateCampaignRepository(session, next)
	state := application.SaveState{SchemaVersion: 1, WorldSeed: 9}
	if err := decorated.BeginWrite(7, application.QuickSave, state); err != nil {
		t.Fatal(err)
	}
	if got := decorated.Poll(); len(got) != 1 || got[0].OperationID != 7 {
		t.Fatalf("repository completions = %#v", got)
	}
	if err := decorated.Close(); err != nil {
		t.Fatal(err)
	}
	if next.writes != 1 || next.polls != 1 || next.closes != 1 {
		t.Fatalf("repository calls = %#v", next)
	}
	logged := output.String()
	if strings.Contains(logged, "WorldSeed") || strings.Contains(logged, "world_seed") {
		t.Fatalf("save payload leaked into repository log: %s", logged)
	}
	if !strings.Contains(logged, "repository.completion") {
		t.Fatalf("repository completion missing: %s", logged)
	}
	_ = session.Close()
}

func TestPanicGuardLogsAndRepanicsWithOriginalValue(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("p", 16)), "test", output)
	want := &struct{ message string }{message: "original"}
	func() {
		defer func() {
			if got := recover(); got != want {
				t.Fatalf("recovered panic = %#v, want original %#v", got, want)
			}
		}()
		defer GuardPanic(session)
		panic(want)
	}()
	if logged := output.String(); !strings.Contains(logged, `"msg":"panic"`) || !strings.Contains(logged, "stack_clipped") {
		t.Fatalf("panic was not logged with bounded-stack metadata: %s", logged)
	}
	_ = session.Close()
}

var _ io.Closer = (*bufferCloser)(nil)
