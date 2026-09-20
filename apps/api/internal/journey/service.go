package journey

import (
	"context"
	"errors"
	"fmt"
	"sort"

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
	// immediately, and records the command in the audit log with its actor.
	TransitionStep(ctx context.Context, visitID, stepKey string, cmd TransitionCommand, source string, actor Actor) (View, error)
	// CloseRound is the staff override (ADR-0009 §4): confirms the visit's
	// latest round at clinicCode is finished, dropping any not-yet-started
	// inferred return, without waiting for (or in place of) the HIS's
	// encounter.completed fact. Audited like any other staff command.
	CloseRound(ctx context.Context, visitID, clinicCode, source string, actor Actor) (View, error)
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

		if event.Type == his.EventEncounterStarted {
			if clinicCode, _ := event.Payload["clinicCode"].(string); clinicCode != "" {
				if err := s.startNextRoundIn(ctx, event.VisitID, clinicCode, existing); err != nil {
					return err
				}
			}
		}
		if event.Type == his.EventEncounterCompleted {
			if clinicCode, _ := event.Payload["clinicCode"].(string); clinicCode != "" {
				if _, _, err := s.closeLatestRoundIn(ctx, event.VisitID, clinicCode, existing); err != nil {
					return err
				}
			}
		}

		visit, err := s.replanFromPrior(ctx, event.VisitID, snapshot, existing, nil)
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

func (s *service) TransitionStep(ctx context.Context, visitID, stepKey string, cmd TransitionCommand, source string, actor Actor) (View, error) {
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

		// The commanded write above is already in `existing`, so the diff in
		// replanFromPrior cannot see it — the attribution below records it in
		// the timeline with the pre-command status and the command's source
		// and actor (#85, NFR-09).
		if _, err := s.replanFromPrior(ctx, visitID, snapshot, existing, &commandAttribution{
			stepKey: stepKey, source: source, actor: actor, fromStatus: current,
		}); err != nil {
			return err
		}
		return s.repo.InsertCommandAudit(ctx, CommandAudit{
			CommandID: cmd.CommandID, VisitID: visitID, StepKey: stepKey, ToStatus: cmd.To, Source: source,
			ActorUserID: actor.UserID, ActorUsername: actor.Username,
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

func (s *service) CloseRound(ctx context.Context, visitID, clinicCode, source string, actor Actor) (View, error) {
	snapshot, err := s.his.GetVisit(ctx, visitID)
	if err != nil {
		return View{}, err
	}

	if source == "" {
		source = "unknown"
	}
	commandID := uuid.NewString()

	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.repo.GetVisit(ctx, visitID)
		if err != nil {
			return err
		}
		closedKey, found, err := s.closeLatestRoundIn(ctx, visitID, clinicCode, existing)
		if err != nil {
			return err
		}
		if !found {
			return ErrNoOpenRound
		}
		// The close takes effect through the replan (closed_round → the plan
		// promotes the round to COMPLETED), so its timeline row is attributed
		// to the command rather than the planner (#85, NFR-09).
		if _, err := s.replanFromPrior(ctx, visitID, snapshot, existing, &commandAttribution{
			stepKey: closedKey, source: source, actor: actor,
		}); err != nil {
			return err
		}
		// The override is a staff command like a transition (NFR-09): it is
		// audited with its own command id, so a repeated close is visible as
		// the (idempotent, side-effect-free) command it was.
		return s.repo.InsertCommandAudit(ctx, CommandAudit{
			CommandID: commandID, VisitID: visitID, StepKey: closedKey, ToStatus: CommandToCompleted,
			Source: source, ActorUserID: actor.UserID, ActorUsername: actor.Username,
		})
	})
	if err != nil {
		return View{}, err
	}
	logger.FromContext(ctx).Info("clinic round closed",
		"command_id", commandID, "visit_id", visitID, "clinic_code", clinicCode, "source", source,
		"actor_user_id", actor.UserID, "actor_username", actor.Username,
	)
	return s.GetJourney(ctx, visitID)
}

// startNextRoundIn applies the HIS's encounter.started fact (ADR-0009 §4):
// the clinic called the patient in, so the round that becomes actionable
// starts. It targets the highest round of clinicCode whose status is READY —
// round 1 on the first call, the inferred return once its diagnostics have
// resulted. Any lower round of the same clinic still STARTED is closed
// first: calling the patient in for the next round means the previous one
// ended when the patient was sent out for those diagnostics, and closing it
// there (rather than via encounter.completed) keeps exactly one round in
// progress. Calling in with no READY round (results not back yet, pre-visit
// gate still holding) is a no-op — the fact is still marked applied; the
// patient cannot be in a round that is not actionable yet.
func (s *service) startNextRoundIn(ctx context.Context, visitID, clinicCode string, existing Visit) error {
	readyIdx, readyRound := -1, 0
	startedKey, startedRound := "", 0
	for i, st := range existing.Steps {
		if st.Kind != KindClinic || st.ClinicCode == nil || *st.ClinicCode != clinicCode || st.Round == nil {
			continue
		}
		if st.Status == StepReady && *st.Round > readyRound {
			readyRound, readyIdx = *st.Round, i
		}
		if st.Status == StepStarted && *st.Round > startedRound {
			startedRound, startedKey = *st.Round, st.StepKey
		}
	}
	if readyIdx < 0 {
		return nil
	}
	if startedKey != "" && startedRound < readyRound {
		if err := s.repo.CloseRound(ctx, visitID, startedKey); err != nil {
			return err
		}
	}
	existing.Steps[readyIdx].Status = StepStarted
	return nil
}

// closeLatestRoundIn resolves the round CLINIC step "in encounter" for
// clinicCode in the given (already-loaded) visit and records it closed,
// returning the stepKey it closed. Preferring the highest STARTED round over
// a merely-inferred, not-yet-begun next round matters: a mid-encounter order
// tentatively creates that next round (WAITING/READY) before the patient
// ever returns, and closing must target the round the doctor is actually
// finishing, not that placeholder — dropping it is a side effect of closing
// the right one (ADR-0009 §4), handled by Plan. Falls back to the highest
// round overall when none is STARTED. Reports found=false when the visit has
// no round at that clinic yet.
func (s *service) closeLatestRoundIn(ctx context.Context, visitID, clinicCode string, existing Visit) (string, bool, error) {
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
		return "", false, nil
	}
	if err := s.repo.CloseRound(ctx, visitID, key); err != nil {
		return "", false, err
	}
	return key, true, nil
}

// commandAttribution attributes one step's status change in a replan to a
// command instead of the planner: the timeline row then carries the command's
// source and actor (NFR-09). fromStatus overrides the recorded pre-change
// status — TransitionStep needs it because it applies the commanded write to
// `existing` before replanning, leaving the prior map already holding the new
// status for the diff to see.
type commandAttribution struct {
	stepKey    string
	source     string
	actor      Actor
	fromStatus string
}

// replanFromPrior recomputes the plan against an already-loaded prior visit
// (letting a caller apply a direct status write to it first, e.g.
// TransitionStep), resolves service points, persists the result, and appends
// one timeline row per status change the round produced (#85). Must be called
// with the transaction-bound ctx.
func (s *service) replanFromPrior(ctx context.Context, visitID string, snapshot his.Visit, existing Visit, attribution *commandAttribution) (Visit, error) {
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
	// Timeline writes are silent by design (#85): the ingest poller replans
	// every few seconds and one log line per event would dwarf everything
	// else in the output.
	if events := statusEventsBetween(visitID, existing, visit, attribution); len(events) > 0 {
		if err := s.repo.AppendStatusEvents(ctx, events); err != nil {
			return Visit{}, err
		}
	}
	return visit, nil
}

// statusEventsBetween diffs the prior plan against the freshly computed one
// and returns one timeline row per change (#85): a step entering the plan
// (from_status NULL), a status change, and a step the replan withdrew
// (to CANCELLED). A round that changes nothing returns nothing — replans run
// constantly and the timeline must not grow on no-ops. Pure function, so the
// shape of the timeline is unit-testable without a database.
func statusEventsBetween(visitID string, prior Visit, next Visit, attribution *commandAttribution) []StepStatusEvent {
	priorStatus := make(map[string]string, len(prior.Steps))
	priorSteps := make(map[string]Step, len(prior.Steps))
	for _, st := range prior.Steps {
		priorStatus[st.StepKey] = st.Status
		priorSteps[st.StepKey] = st
	}

	nextKeys := make(map[string]bool, len(next.Steps))
	var events []StepStatusEvent
	for i := range next.Steps {
		st := next.Steps[i]
		nextKeys[st.StepKey] = true
		from, existed := priorStatus[st.StepKey]
		source, actor := EventSourcePlanner, Actor{}
		if attribution != nil && attribution.stepKey == st.StepKey {
			source, actor = attribution.source, attribution.actor
			if attribution.fromStatus != "" {
				from, existed = attribution.fromStatus, true
			}
		}
		if existed && from == st.Status {
			continue
		}
		ev := StepStatusEvent{
			VisitID: visitID, StepKey: st.StepKey, Kind: st.Kind, ServicePointID: st.ServicePointID,
			ToStatus: st.Status, Source: source, ActorUserID: actor.UserID, ActorUsername: actor.Username,
		}
		if existed {
			f := from
			ev.FromStatus = &f
		}
		events = append(events, ev)
	}

	// A replan may withdraw a not-yet-started step from the plan (ADR-0009
	// §7 — steps with history are never withdrawn). Its disappearance is a
	// status change like any other, recorded as CANCELLED with the kind and
	// service point copied from the withdrawn step: after the upsert no
	// journey_step row references it anymore.
	withdrawn := make([]string, 0)
	for _, st := range prior.Steps {
		if nextKeys[st.StepKey] {
			continue
		}
		// Terminal or in-progress steps are never withdrawn by contract; a
		// CANCELLED one already carries its terminal status. Guarding them
		// here keeps a planner bug from being recorded as history.
		if st.Status != StepPending && st.Status != StepWaiting && st.Status != StepReady {
			continue
		}
		withdrawn = append(withdrawn, st.StepKey)
	}
	sort.Strings(withdrawn)
	for _, key := range withdrawn {
		st := priorSteps[key]
		from := st.Status
		events = append(events, StepStatusEvent{
			VisitID: visitID, StepKey: st.StepKey, Kind: st.Kind, ServicePointID: st.ServicePointID,
			FromStatus: &from, ToStatus: StepCancelled, Source: EventSourcePlanner,
		})
	}
	return events
}

func indexOfStep(steps []Step, stepKey string) int {
	for i, s := range steps {
		if s.StepKey == stepKey {
			return i
		}
	}
	return -1
}
