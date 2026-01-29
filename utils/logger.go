package utils

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type Logger = slog.Logger

func NewLogger() *Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(h)
}

type AuditEvent struct {
	Action string
	UserID int
	At     time.Time
}

type AuditLogger struct {
	log *Logger
	ch  chan AuditEvent
}

func NewAuditLogger(log *Logger, buffer int) *AuditLogger {
	return &AuditLogger{
		log: log,
		ch:  make(chan AuditEvent, buffer),
	}
}

// Log — неблокирующий вызов: если канал переполнен, событие дропается.
func (a *AuditLogger) Log(action string, userID int) {
	select {
	case a.ch <- AuditEvent{Action: action, UserID: userID, At: time.Now()}:
	default:
	}
}

func (a *AuditLogger) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				a.log.Info("audit logger stopped")
				return
			case ev := <-a.ch:
				a.log.Info("audit", "action", ev.Action, "user_id", ev.UserID, "at", ev.At.UTC().Format(time.RFC3339Nano))
			}
		}
	}()
}

type ErrorReporter struct {
	log *Logger
	ch  chan error
}

func NewErrorReporter(log *Logger, buffer int) *ErrorReporter {
	return &ErrorReporter{
		log: log,
		ch:  make(chan error, buffer),
	}
}

func (e *ErrorReporter) Channel() chan<- error {
	return e.ch
}

func (e *ErrorReporter) Report(err error) {
	select {
	case e.ch <- err:
	default:
	}
}

func (e *ErrorReporter) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				e.log.Info("error reporter stopped")
				return
			case err := <-e.ch:
				e.log.Error("async error", "err", err.Error())
			}
		}
	}()
}

