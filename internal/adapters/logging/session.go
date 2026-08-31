package logging

import (
	"encoding/hex"
	"io"
	"log/slog"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

type Session struct {
	logger    *slog.Logger
	closer    io.Closer
	sessionID string
	target    string
	started   time.Time
	operation atomic.Uint64
	closed    atomic.Bool
}

const maxPanicStackBytes = 16 * 1024

// GuardPanic records one bounded worker/entrypoint panic and re-panics with the
// original value. Install it after the session Close defer so the panic record
// is flushed before the sink closes during stack unwinding.
func GuardPanic(session *Session) {
	value := recover()
	if value == nil {
		return
	}
	if session != nil {
		stack := debug.Stack()
		clipped := false
		if len(stack) > maxPanicStackBytes {
			stack = stack[:maxPanicStackBytes]
			clipped = true
		}
		session.logger.Error("panic", "value", value, "stack", string(stack), "stack_clipped", clipped)
	}
	panic(value)
}

func newSession(entropy io.Reader, target string, writer io.WriteCloser) *Session {
	var identifier [16]byte
	_, entropyErr := io.ReadFull(entropy, identifier[:])
	if entropyErr != nil {
		fallback := uint64(time.Now().UnixNano())
		for index := range identifier {
			identifier[index] = byte(fallback >> (uint(index%8) * 8))
		}
	}
	session := &Session{closer: writer, sessionID: hex.EncodeToString(identifier[:]), target: target, started: time.Now()}
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})
	session.logger = slog.New(handler).With("session_id", session.sessionID, "target", target)
	if entropyErr != nil {
		session.logger.Warn("session.entropy_fallback", "error", entropyErr.Error())
	}
	session.logger.Info("session.start")
	return session
}

func (session *Session) Close() error {
	if session == nil || !session.closed.CompareAndSwap(false, true) {
		return nil
	}
	session.logger.Info("session.end", "duration_ms", float64(time.Since(session.started))/float64(time.Millisecond))
	return session.closer.Close()
}

func (session *Session) nextOperation() uint64 { return session.operation.Add(1) }

func (session *Session) start(layer, operation string, attributes ...any) (uint64, time.Time) {
	id := session.nextOperation()
	values := append([]any{"operation_id", id, "layer", layer, "operation", operation}, attributes...)
	session.logger.Info("operation.start", values...)
	return id, time.Now()
}

func (session *Session) end(id uint64, started time.Time, layer, operation string, err error, attributes ...any) {
	outcome := "ok"
	if err != nil {
		outcome = "error"
		attributes = append(attributes, "error", err.Error())
	}
	values := []any{"operation_id", id, "layer", layer, "operation", operation, "outcome", outcome, "duration_ms", float64(time.Since(started)) / float64(time.Millisecond)}
	values = append(values, attributes...)
	session.logger.Info("operation.end", values...)
}

// LogActionRejected records one batch-level rejection. Callers must invoke it
// only after preflight fails and before invoking any member of the batch.
func (session *Session) LogActionRejected(actionCount int, err error) {
	if session == nil {
		return
	}
	session.logger.Warn("action.rejected", "action_count", actionCount, "error", err)
}

// LogActionDispatch records the bounded scalar payload immediately before the
// host invokes one accepted typed action.
func (session *Session) LogActionDispatch(action ui.Action) {
	if session == nil {
		return
	}
	attributes := []any{"kind", action.Kind().String()}
	switch action.Kind() {
	case ui.ActionSimulationCommand:
		attributes = append(attributes, commandLogAttributes(action.Command())...)
	case ui.ActionSave, ui.ActionLoad, ui.ActionDelete:
		attributes = append(attributes, "slot", action.Slot())
	case ui.ActionNavigate:
		navigation, scene := action.Navigation()
		attributes = append(attributes, "navigation", uint8(navigation), "scene", uint8(scene))
	}
	session.logger.Info("action.dispatch", attributes...)
}

func commandLogAttributes(command gameapi.Command) []any {
	attributes := []any{"command", fmtCommandKind(command), "band_id", uint64(command.ActingBandID())}
	switch value := command.(type) {
	case gameapi.SetAssignment:
		attributes = append(attributes,
			"foraging_bp", value.AllocationBP[gameapi.Foraging],
			"hunting_fishing_bp", value.AllocationBP[gameapi.HuntingAndFishing],
			"toolcraft_bp", value.AllocationBP[gameapi.Toolcraft],
			"megafauna_bp", value.AllocationBP[gameapi.MegafaunaTracking],
			"shelter_bp", value.AllocationBP[gameapi.Shelter],
		)
	case gameapi.QueueMigration:
		attributes = append(attributes, "tile_id", uint64(value.TileID))
	case gameapi.SplitBand:
		attributes = append(attributes, "tile_id", uint64(value.Destination))
	case gameapi.ResearchTech:
		attributes = append(attributes, "technology", uint8(value.Tech))
	case gameapi.Interbreed:
		attributes = append(attributes, "target_band_id", uint64(value.TargetBandID))
	}
	return attributes
}

func fmtCommandKind(command gameapi.Command) string {
	switch command.(type) {
	case gameapi.SetAssignment:
		return "set_assignment"
	case gameapi.QueueMigration:
		return "queue_migration"
	case gameapi.SplitBand:
		return "split_band"
	case gameapi.ResearchTech:
		return "research_tech"
	case gameapi.Interbreed:
		return "interbreed"
	default:
		return "unknown"
	}
}
