package logging

import (
	"encoding/hex"
	"io"
	"log/slog"
	"sync/atomic"
	"time"
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
