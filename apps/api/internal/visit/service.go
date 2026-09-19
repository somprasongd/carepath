package visit

import (
	"context"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/platform/logger"
	"carepath/apps/api/internal/servicepoint"
)

// Service builds the normalized visit view. The visit module depends on the
// servicepoint module only through its Service interface, never its Repo.
type Service interface {
	GetVisitView(ctx context.Context, visitID string) (VisitView, error)
	GetNextStep(ctx context.Context, visitID string) (*NextStep, error)
}

type service struct {
	his           his.Client
	servicePoints servicepoint.Service
	tx            db.Transactor
}

func NewService(hisClient his.Client, servicePoints servicepoint.Service, tx db.Transactor) Service {
	return &service{his: hisClient, servicePoints: servicePoints, tx: tx}
}

func (s *service) GetVisitView(ctx context.Context, visitID string) (VisitView, error) {
	visit, err := s.his.GetVisit(ctx, visitID)
	if err != nil {
		return VisitView{}, err
	}

	var view VisitView
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		view = VisitView{Visit: visit}
		next, err := s.resolveNextStep(ctx, visit)
		if err != nil {
			return err
		}
		view.Next = next
		return nil
	})
	if err != nil {
		return VisitView{}, err
	}
	return view, nil
}

func (s *service) GetNextStep(ctx context.Context, visitID string) (*NextStep, error) {
	visit, err := s.his.GetVisit(ctx, visitID)
	if err != nil {
		return nil, err
	}

	var next *NextStep
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		next, err = s.resolveNextStep(ctx, visit)
		return err
	})
	if err != nil {
		return nil, err
	}
	if next == nil {
		return nil, ErrNoNextStep
	}
	return next, nil
}

// resolveNextStep must be called with the transaction-bound ctx so that the
// servicepoint lookup joins the same transaction. It returns nil when no step
// is READY, which is a valid state for a full visit view.
func (s *service) resolveNextStep(ctx context.Context, visit his.Visit) (*NextStep, error) {
	for _, step := range visit.Steps {
		if step.Status != "READY" {
			continue
		}
		next := &NextStep{Sequence: step.Sequence, Status: step.Status}
		sp, err := s.servicePoints.GetByCode(ctx, step.ServiceCode)
		switch {
		case err == nil:
			next.ServicePoint = &sp
		case apperr.KindOf(err) == apperr.KindNotFound:
			// No configured service point for this code; the step itself is
			// still actionable.
		default:
			return nil, err
		}
		logger.FromContext(ctx).Debug("resolved next step",
			"visit_id", visit.VisitID,
			"sequence", step.Sequence,
			"service_code", step.ServiceCode,
		)
		return next, nil
	}
	return nil, nil
}
