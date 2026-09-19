package journey

import (
	"context"
	"sort"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/platform/logger"
	"carepath/apps/api/internal/servicepoint"
)

// Service is the only entry point other modules may call.
type Service interface {
	// ApplyHISEvent projects one canonical HIS event onto the journey. The
	// visit snapshot is re-read from the HIS and upserted, so the projection
	// never guesses state the event payload does not carry. Duplicate
	// delivery is a no-op: an eventId already applied never re-projects and
	// never duplicates a step (#21 AC2).
	ApplyHISEvent(ctx context.Context, event his.Event) error
	// GetVisit returns the projected journey for reading.
	GetVisit(ctx context.Context, visitID string) (Visit, error)
	// GetJourney returns the patient-facing view of the projection: steps
	// ordered by sequence, each resolved to its service point, with the
	// deterministic current (first STARTED) and next (first READY) step.
	GetJourney(ctx context.Context, visitID string) (View, error)
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
			// nothing to project; record the event as applied so the feed
			// can advance instead of retrying forever.
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
		visit, err := s.project(ctx, snapshot)
		if err != nil {
			return err
		}
		if err := s.repo.UpsertVisit(ctx, visit); err != nil {
			return err
		}
		if err := s.repo.MarkEventApplied(ctx, event.EventID, event.VisitID); err != nil {
			return err
		}
		logger.FromContext(ctx).Debug("journey projected",
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
		view, err = s.buildView(ctx, visit)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}

// buildView resolves service points for the projection's stored bindings and
// derives current/next. Resolution is deterministic: steps are ordered by
// sequence, current is the first STARTED step, next the first READY one. It
// must be called with the transaction-bound ctx so the servicepoint read
// joins the projection read.
func (s *service) buildView(ctx context.Context, visit Visit) (View, error) {
	points, err := s.servicePoints.List(ctx)
	if err != nil {
		return View{}, err
	}
	byID := make(map[string]servicepoint.ServicePoint, len(points))
	for _, sp := range points {
		byID[sp.ID] = sp
	}

	steps := make([]StepView, len(visit.Steps))
	for i, step := range visit.Steps {
		steps[i] = StepView{
			Sequence:       step.Sequence,
			ServiceCode:    step.ServiceCode,
			Status:         step.Status,
			ServicePointID: step.ServicePointID,
		}
		if step.ServicePointID != nil {
			if sp, ok := byID[*step.ServicePointID]; ok {
				steps[i].ServicePoint = &sp
			}
		}
	}
	sort.Slice(steps, func(a, b int) bool { return steps[a].Sequence < steps[b].Sequence })

	view := View{
		VisitID:    visit.VisitID,
		PatientRef: visit.PatientRef,
		Status:     visit.Status,
		Completed:  visit.Status == statusVisitCompleted,
		Steps:      steps,
		SyncedAt:   visit.SyncedAt,
	}
	for i := range steps {
		switch {
		case view.Current == nil && steps[i].Status == statusStepStarted:
			view.Current = &steps[i]
		case view.Next == nil && steps[i].Status == statusStepReady:
			view.Next = &steps[i]
		}
	}
	return view, nil
}

// project resolves each step's service-point binding. A step whose service
// code has no configured service point is kept with a nil binding and a warn
// log — the journey stays usable and the gap is explicit (#21 AC3). It must
// be called with the transaction-bound ctx so lookups join the projection
// write.
func (s *service) project(ctx context.Context, snapshot his.Visit) (Visit, error) {
	log := logger.FromContext(ctx)
	visit := Visit{
		VisitID:    snapshot.VisitID,
		PatientRef: snapshot.PatientRef,
		Status:     snapshot.Status,
	}
	for _, step := range snapshot.Steps {
		projected := Step{
			Sequence:    step.Sequence,
			ServiceCode: step.ServiceCode,
			Status:      step.Status,
		}
		sp, err := s.servicePoints.GetByCode(ctx, step.ServiceCode)
		switch {
		case err == nil:
			projected.ServicePointID = &sp.ID
		case apperr.KindOf(err) == apperr.KindNotFound:
			log.Warn("unmapped service code in projected step",
				"visit_id", snapshot.VisitID,
				"sequence", step.Sequence,
				"service_code", step.ServiceCode,
			)
		default:
			return Visit{}, err
		}
		visit.Steps = append(visit.Steps, projected)
	}
	return visit, nil
}
