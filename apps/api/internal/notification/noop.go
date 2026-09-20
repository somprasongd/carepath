package notification

import (
	"context"
	"log/slog"
)

// Noop is the default Notifier adapter (ADR-0013 §1): it delivers nothing
// and says so, so dev/demo runs exercise the whole engine — criterion,
// dedupe, opt-out — with the send visibly suppressed instead of silently
// missing.
type Noop struct {
	log *slog.Logger
}

func NewNoop(log *slog.Logger) *Noop {
	if log == nil {
		log = slog.Default()
	}
	return &Noop{log: log}
}

func (n *Noop) Channel() string { return "noop" }

func (n *Noop) Notify(_ context.Context, recipient, text string) error {
	n.log.Info("queue notification suppressed (no channel configured)", "recipient", recipient, "text", text)
	return nil
}
