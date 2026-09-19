// Package ingest is the inbound HIS adapter: it polls the canonical event
// feed (MVP transport per ADR-0008 §4: REST pull) and drives the journey
// projection through the Applier port. Feed paging, the durable cursor, and
// envelope handling live here so the journey domain stays free of transport
// concerns (#21 AC1). A future push/webhook transport replaces this package
// without touching the canonical contract or the journey domain.
package ingest

import (
	"context"
	"log/slog"
	"time"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
)

// pageSize caps how many events one feed request asks for.
const pageSize = 100

// Applier is the journey-side consumer of canonical events, satisfied by
// journey.Service.
type Applier interface {
	ApplyHISEvent(ctx context.Context, event his.Event) error
}

// EventSource is the feed-reading surface of the HIS port.
type EventSource interface {
	Events(ctx context.Context, after string, limit int) (his.EventPage, error)
}

// Checkpoint durably stores the feed cursor (the last consumed eventId).
type Checkpoint interface {
	LastEventID(ctx context.Context) (string, error)
	Save(ctx context.Context, lastEventID string) error
}

// Poller drains the event feed into the journey projection.
type Poller struct {
	source     EventSource
	applier    Applier
	checkpoint Checkpoint
	log        *slog.Logger
}

func New(source EventSource, applier Applier, checkpoint Checkpoint, log *slog.Logger) *Poller {
	return &Poller{source: source, applier: applier, checkpoint: checkpoint, log: log}
}

// PollOnce drains every page currently available on the feed, starting from
// the stored cursor, and returns the number of events consumed. The cursor is
// saved after each page: if applying fails mid-page, the cursor still points
// before the failing event and the next poll retries it. Malformed envelopes
// are skipped with a warning — a poison message must not block the feed.
func (p *Poller) PollOnce(ctx context.Context) (int, error) {
	after, err := p.checkpoint.LastEventID(ctx)
	if err != nil {
		return 0, err
	}

	consumed := 0
	for {
		page, err := p.source.Events(ctx, after, pageSize)
		if err != nil {
			return consumed, err
		}
		if len(page.Events) == 0 {
			return consumed, nil
		}

		for _, event := range page.Events {
			if err := p.applier.ApplyHISEvent(ctx, event); err != nil {
				if apperr.KindOf(err) == apperr.KindInvalid {
					p.log.Warn("skipping invalid HIS event",
						"event_id", event.EventID, "type", event.Type, "error", err.Error())
					continue
				}
				return consumed, err
			}
			consumed++
		}

		next := page.NextAfter
		if next == "" {
			// Defensive: a non-empty page must carry a cursor; fall back to
			// its last event id so progress is never lost.
			next = page.Events[len(page.Events)-1].EventID
		}
		if err := p.checkpoint.Save(ctx, next); err != nil {
			return consumed, err
		}
		p.log.Debug("ingested event page", "next_after", next, "page_size", len(page.Events))
		after = next
	}
}

// Run polls the feed on every tick until ctx is cancelled. A failed poll is
// logged and retried on the next tick; the cursor keeps the position.
func (p *Poller) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := p.PollOnce(ctx); err != nil {
			if ctx.Err() == nil {
				p.log.Error("HIS ingest poll failed", "error", err.Error())
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
