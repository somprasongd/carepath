package journey

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/platform/logger"
	"carepath/apps/api/internal/servicepoint"
)

// Service is the only entry point other modules may call.
type Service interface {
	// ApplyHISEvent projects one canonical HIS fact onto the journey plan.
	// The visit snapshot is re-read from the HIS and the plan recomputed
	// (ADR-0009), so the projection never guesses state the fact does not
	// carry. Duplicate delivery is a no-op: an eventId already applied never
	// re-projects.
	ApplyHISEvent(ctx context.Context, event his.Event) error
	// GetVisit returns the projected journey for reading.
	GetVisit(ctx context.Context, visitID string) (Visit, error)
	// GetJourney returns the patient/staff-facing view of the plan: steps in
	// display order, each resolved to its service point, with every
	// currently-actionable step and CarePath's recommendation among them.
	GetJourney(ctx context.Context, visitID string) (View, error)
	// ListJourneys returns the projected journey of every visit for the
	// staff visit monitor (#37), freshest sync first.
	ListJourneys(ctx context.Context) ([]View, error)
	// TransitionStep applies a staff command to one step's status directly
	// (ADR-0009 §1 — CarePath owns step status; this is never forwarded to
	// the HIS), then recomputes the plan so downstream steps' gates react
	// immediately, and records the command in the audit log.
	TransitionStep(ctx context.Context, visitID, stepKey string, cmd TransitionCommand, source string) (View, error)
	// CloseRound is the staff override (ADR-0009 §4): confirms the visit's
	// latest round at clinicCode is finished, dropping any not-yet-started
	// inferred return, without waiting for (or in place of) the HIS's
	// encounter.completed fact.
	CloseRound(ctx context.Context, visitID, clinicCode string) (View, error)
}

type service struct {
	his           his.Client
	servicePoints servicepoint.Service
	repo          Repo
	tx            db.Transactor
}

func NewService(hisClient his.Client, servicePoints servicepoint.Service, repo Repo, tx db.Transactor) Service {
	return &service{his: hisClient, servicePoints: servicePoints, repo: repo, tx: tx}
}

func (s *service) ApplyHISEvent(ctx context.Context, event his.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}

	// Fast path: a duplicate is answered by the eventId check alone, without
	// a snapshot round-trip. The check is repeated inside the transaction,
	// which is the one that guarantees it.
	applied, err := s.repo.EventApplied(ctx, event.EventID)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	// The HIS is an external system and never participates in database
	// transactions (ADR-0007), so the snapshot is read before one is opened.
	snapshot, err := s.his.GetVisit(ctx, event.VisitID)
	if err != nil {
		if apperr.KindOf(err) == apperr.KindNotFound {
			// The HIS no longer knows this visit (or not yet). There is
			// nothing to project; record the fact as applied so the feed can
			// advance instead of retrying forever.
			log := logger.FromContext(ctx)
			log.Warn("event for unknown visit; marking applied without projection",
				"event_id", event.EventID, "visit_id", event.VisitID, "type", event.Type)
			return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
				return s.repo.MarkEventApplied(ctx, event.EventID, event.VisitID)
			})
		}
		return err
	}

	return s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		applied, err := s.repo.EventApplied(ctx, event.EventID)
		if err != nil {
			return err
		}
		if applied {
			return nil
		}

		existing, err := s.repo.GetVisit(ctx, event.VisitID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		if event.Type == his.EventEncounterCompleted {
			if clinicCode, _ := event.Payload["clinicCode"].(string); clinicCode != "" {
				if _, err := s.closeLatestRoundIn(ctx, event.VisitID, clinicCode, existing); err != nil {
					return err
				}
			}
		}

		visit, err := s.replanFromPrior(ctx, event.VisitID, snapshot, existing)
		if err != nil {
			return err
		}
		if err := s.repo.MarkEventApplied(ctx, event.EventID, event.VisitID); err != nil {
			return err
		}
		logger.FromContext(ctx).Debug("journey replanned",
			"event_id", event.EventID,
			"visit_id", visit.VisitID,
			"type", event.Type,
			"status", visit.Status,
			"steps", len(visit.Steps),
		)
		return nil
	})
}

func (s *service) GetVisit(ctx context.Context, visitID string) (Visit, error) {
	return s.repo.GetVisit(ctx, visitID)
}

func (s *service) GetJourney(ctx context.Context, visitID string) (View, error) {
	var view View
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		visit, err := s.repo.GetVisit(ctx, visitID)
		if err != nil {
			return err
		}
		points, err := s.servicePoints.List(ctx)
		if err != nil {
			return err
		}
		view = assembleView(visit, points)
		return nil
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

func (s *service) ListJourneys(ctx context.Context) ([]View, error) {
	var views []View
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		visits, err := s.repo.ListVisits(ctx)
		if err != nil {
			return err
		}
		points, err := s.servicePoints.List(ctx)
		if err != nil {
			return err
		}
		views = make([]View, len(visits))
		for i, visit := range visits {
			views[i] = assembleView(visit, points)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return views, nil
}

func (s *service) TransitionStep(ctx context.Context, visitID, stepKey string, cmd TransitionCommand, source string) (View, error) {
	switch cmd.To {
	case CommandToStarted, CommandToCompleted, CommandToCancelled:
	default:
		return View{}, apperr.New(apperr.KindInvalid, fmt.Sprintf("unknown target status %q", cmd.To))
	}
	if cmd.CommandID == "" {
		cmd.CommandID = uuid.NewString()
	}
	if source == "" {
		source = "unknown"
	}

	// The HIS is external and never joins a database transaction (ADR-0007);
	// the fresh snapshot the replan uses is read up front.
	snapshot, err := s.his.GetVisit(ctx, visitID)
	if err != nil {
		return View{}, err
	}

	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.repo.GetVisit(ctx, visitID)
		if err != nil {
			return err
		}
		idx := indexOfStep(existing.Steps, stepKey)
		if idx < 0 {
			return apperr.New(apperr.KindNotFound, fmt.Sprintf("step %q not found", stepKey))
		}
		current := existing.Steps[idx].Status
		if current != cmd.To {
			switch {
			case current == StepCompleted || current == StepCancelled:
				return apperr.New(apperr.KindConflict,
					fmt.Sprintf("step %s is %s and cannot transition to %s", stepKey, current, cmd.To))
			case cmd.To == CommandToStarted && current != StepReady:
				return apperr.New(apperr.KindConflict,
					fmt.Sprintf("step %s is %s and cannot transition to STARTED", stepKey, current))
			}
			existing.Steps[idx].Status = cmd.To
		}

		if _, err := s.replanFromPrior(ctx, visitID, snapshot, existing); err != nil {
			return err
		}
		return s.repo.InsertCommandAudit(ctx, CommandAudit{
			CommandID: cmd.CommandID, VisitID: visitID, StepKey: stepKey, ToStatus: cmd.To, Source: source,
		})
	})
	if err != nil {
		return View{}, err
	}
	logger.FromContext(ctx).Info("step transitioned",
		"command_id", cmd.CommandID, "visit_id", visitID, "step_key", stepKey, "to", cmd.To, "source", source,
	)
	return s.GetJourney(ctx, visitID)
}

func (s *service) CloseRound(ctx context.Context, visitID, clinicCode string) (View, error) {
	snapshot, err := s.his.GetVisit(ctx, visitID)
	if err != nil {
		return View{}, err
	}

	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.repo.GetVisit(ctx, visitID)
		if err != nil {
			return err
		}
		found, err := s.closeLatestRoundIn(ctx, visitID, clinicCode, existing)
		if err != nil {
			return err
		}
		if !found {
			return ErrNoOpenRound
		}
		_, err = s.replanFromPrior(ctx, visitID, snapshot, existing)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.GetJourney(ctx, visitID)
}

// closeLatestRoundIn resolves the round CLINIC step "in encounter" for
// clinicCode in the given (already-loaded) visit and records it closed.
// Preferring the highest STARTED round over a merely-inferred, not-yet-begun
// next round matters: a mid-encounter order tentatively creates that next
// round (WAITING/READY) before the patient ever returns, and closing must
// target the round the doctor is actually finishing, not that placeholder —
// dropping it is a side effect of closing the right one (ADR-0009 §4),
// handled by Plan. Falls back to the highest round overall when none is
// STARTED. Reports false when the visit has no round at that clinic yet.
func (s *service) closeLatestRoundIn(ctx context.Context, visitID, clinicCode string, existing Visit) (bool, error) {
	latestKey, latestRound := "", 0
	startedKey, startedRound := "", 0
	for _, st := range existing.Steps {
		if st.Kind != KindClinic || st.ClinicCode == nil || *st.ClinicCode != clinicCode || st.Round == nil {
			continue
		}
		if *st.Round > latestRound {
			latestRound, latestKey = *st.Round, st.StepKey
		}
		if st.Status == StepStarted && *st.Round > startedRound {
			startedRound, startedKey = *st.Round, st.StepKey
		}
	}
	key := startedKey
	if key == "" {
		key = latestKey
	}
	if key == "" {
		return false, nil
	}
	if err := s.repo.CloseRound(ctx, visitID, key); err != nil {
		return false, err
	}
	return true, nil
}

// replan recomputes the plan from the visit's currently-stored steps.
func (s *service) replan(ctx context.Context, visitID string, snapshot his.Visit) (Visit, error) {
	existing, err := s.repo.GetVisit(ctx, visitID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Visit{}, err
	}
	return s.replanFromPrior(ctx, visitID, snapshot, existing)
}

// replanFromPrior recomputes the plan against an already-loaded prior visit
// (letting a caller apply a direct status write to it first, e.g.
// TransitionStep), resolves service points, and persists the result. Must be
// called with the transaction-bound ctx.
func (s *service) replanFromPrior(ctx context.Context, visitID string, snapshot his.Visit, existing Visit) (Visit, error) {
	prior := make(map[string]string, len(existing.Steps))
	for _, st := range existing.Steps {
		prior[st.StepKey] = st.Status
	}
	closed, err := s.repo.ClosedRounds(ctx, visitID)
	if err != nil {
		return Visit{}, err
	}

	steps := Plan(snapshot, prior, closed)
	visit := Visit{
		VisitID: snapshot.VisitID, PatientRef: snapshot.PatientRef, PatientName: snapshot.PatientName,
		Status: snapshot.Status, Steps: steps,
	}

	log := logger.FromContext(ctx)
	for i := range visit.Steps {
		code := bindingKey(visit.Steps[i])
		sp, err := s.servicePoints.GetByCode(ctx, code)
		switch {
		case err == nil:
			visit.Steps[i].ServicePointID = &sp.ID
		case apperr.KindOf(err) == apperr.KindNotFound:
			log.Warn("unmapped step binding",
				"visit_id", visitID, "step_key", visit.Steps[i].StepKey, "binding", code)
		default:
			return Visit{}, err
		}
	}

	if err := s.repo.UpsertVisit(ctx, visit); err != nil {
		return Visit{}, err
	}
	return visit, nil
}

func indexOfStep(steps []Step, stepKey string) int {
	for i, s := range steps {
		if s.StepKey == stepKey {
			return i
		}
	}
	return -1
}
