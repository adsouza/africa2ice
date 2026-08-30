package logging

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
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

var _ io.Closer = (*bufferCloser)(nil)
