package notification

import (
	"context"
	"log/slog"
	"time"
)

// nearQueueThreshold is the waiting-ahead count at or below which the visit
// counts as "approaching" (ADR-0013 §2). Kept a constant, not an env knob:
// it is part of the product promise, not a deployment detail.
const nearQueueThreshold = 3

// nearQueueText is the one message the module ever sends (ADR-0013 §5): a
// fixed, generic Thai sentence — no step, clinic, doctor, or order, because
// it lands on the phone's lock screen. A declared exception to ADR-0012's
// catalog mechanism: a push payload has no web client to map a code.
const nearQueueText = "ใกล้ถึงคิวของคุณแล้ว กรุณากลับมาที่จุดบริการที่แอปแนะนำไว้"

// Service is the notification engine: the background sweep that decides
// who is near, the dedupe that guarantees exactly once per step, and the
// opt-out surface. Everything except the actual send.
type Service struct {
	queues     QueueSource
	store      Store
	recipients Recipients
	notifier   Notifier
	log        *slog.Logger
}

func NewService(queues QueueSource, store Store, recipients Recipients, notifier Notifier, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{queues: queues, store: store, recipients: recipients, notifier: notifier, log: log}
}

// Run drives the background sweep (ADR-0013 §2) until the context ends —
// the same ticker shape the HIS ingest poller uses. The first sweep runs
// immediately so a restart does not wait a full interval to catch up.
func (s *Service) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.Sweep(ctx); err != nil {
			if ctx.Err() == nil {
				s.log.Error("queue notification sweep failed", "error", err.Error())
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Sweep evaluates every active visit once. One visit's failure must not
// stop the rest — the sweep is best-effort by design, so per-visit errors
// are logged and the sweep continues.
func (s *Service) Sweep(ctx context.Context) error {
	visits, err := s.queues.ActiveVisits(ctx)
	if err != nil {
		return err
	}
	for _, visit := range visits {
		s.evaluate(ctx, visit)
	}
	return nil
}

// evaluate runs the full decision chain for one visit, in the cheapest
// first order: queue criterion → opt-out → recipient → exactly-once claim →
// send. The claim is the second-to-last step on purpose:
//
//   - opt-out and no-recipient skip without claiming — turning
//     notifications back on, or a later real LINE login, can still deliver
//     the step's message (ADR-0013 §3);
//   - a failed send leaves the claim in place — a transient channel error
//     must not spam retries any more than a success would re-send.
func (s *Service) evaluate(ctx context.Context, visit ActiveVisit) {
	steps, err := s.queues.Queue(ctx, visit.VisitID)
	if err != nil {
		s.log.Warn("queue notification: queue read failed", "visit_id", visit.VisitID, "error", err.Error())
		return
	}
	near := false
	for _, step := range steps {
		if step.StepKey == visit.StepKey && step.WaitingAhead <= nearQueueThreshold {
			near = true
			break
		}
	}
	if !near {
		return
	}

	enabled, err := s.store.Enabled(ctx, visit.VisitID)
	if err != nil {
		s.log.Warn("queue notification: pref read failed", "visit_id", visit.VisitID, "error", err.Error())
		return
	}
	if !enabled {
		return
	}

	recipient, ok, err := s.recipients.LineUser(ctx, visit.VisitID)
	if err != nil {
		s.log.Warn("queue notification: recipient resolve failed", "visit_id", visit.VisitID, "error", err.Error())
		return
	}
	if !ok {
		// Demo-mode visits have no LINE identity; nothing is claimed, so a
		// later real login can still be notified. Logged at info so a demo
		// sees the engine fire without implying a message went out.
		s.log.Info("queue notification near, but no LINE recipient for visit", "visit_id", visit.VisitID, "step_key", visit.StepKey)
		return
	}

	won, err := s.store.SendOnce(ctx, visit.VisitID, visit.StepKey, s.notifier.Channel())
	if err != nil {
		s.log.Warn("queue notification: claim failed", "visit_id", visit.VisitID, "step_key", visit.StepKey, "error", err.Error())
		return
	}
	if !won {
		return // already sent for this step — the exactly-once guarantee
	}

	if err := s.notifier.Notify(ctx, recipient, nearQueueText); err != nil {
		s.log.Error("queue notification send failed", "visit_id", visit.VisitID, "step_key", visit.StepKey, "channel", s.notifier.Channel(), "error", err.Error())
		return
	}
	s.log.Info("queue notification sent", "visit_id", visit.VisitID, "step_key", visit.StepKey, "channel", s.notifier.Channel())
}

// GetPref answers the visit's opt-out state for the patient surface.
func (s *Service) GetPref(ctx context.Context, visitID string) (Pref, error) {
	enabled, err := s.store.Enabled(ctx, visitID)
	if err != nil {
		return Pref{}, err
	}
	return Pref{VisitID: visitID, Enabled: enabled}, nil
}

// SetPref stores the patient's opt-out choice.
func (s *Service) SetPref(ctx context.Context, visitID string, enabled bool) (Pref, error) {
	if err := s.store.SetEnabled(ctx, visitID, enabled); err != nil {
		return Pref{}, err
	}
	return Pref{VisitID: visitID, Enabled: enabled}, nil
}
