package servicepoint

import (
	"context"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/platform/apperr"
)

// Service is the only entry point other modules may call. Its methods take a
// plain ctx: when the caller opened a transaction, repository calls made here
// join it through that ctx.
type Service interface {
	GetByCode(ctx context.Context, code string) (ServicePoint, error)
	List(ctx context.Context) ([]ServicePoint, error)
	// ListForUser returns the staff user's assigned points (#102) — the
	// picker for the queue console.
	ListForUser(ctx context.Context, userID string) ([]ServicePoint, error)
	// IsAssigned reports whether the user may work this point's queue.
	IsAssigned(ctx context.Context, userID, servicePointID string) (bool, error)
}

type service struct {
	repo   Repo
	places hospitalmap.Service
}

func NewService(repo Repo, places hospitalmap.Service) Service {
	return &service{repo: repo, places: places}
}

func (s *service) GetByCode(ctx context.Context, code string) (ServicePoint, error) {
	sp, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return ServicePoint{}, err
	}
	place, err := s.places.GetPlace(ctx, sp.PlaceID)
	if err != nil {
		if apperr.KindOf(err) == apperr.KindNotFound {
			return sp, nil
		}
		return ServicePoint{}, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: resolve place %q", sp.PlaceID)
	}
	sp.Place = &place
	return sp, nil
}

func (s *service) List(ctx context.Context) ([]ServicePoint, error) {
	points, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return s.attachPlaces(ctx, points)
}

// ListForUser returns the service points the staff user is assigned to
// (#102) with places resolved — the points their queue console may show.
func (s *service) ListForUser(ctx context.Context, userID string) ([]ServicePoint, error) {
	points, err := s.repo.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.attachPlaces(ctx, points)
}

// IsAssigned reports whether the user may work the queue of this service
// point. Non-assignment and a nonexistent point read the same: the caller
// is not allowed either way, and that is the only question being asked.
func (s *service) IsAssigned(ctx context.Context, userID, servicePointID string) (bool, error) {
	return s.repo.IsAssigned(ctx, userID, servicePointID)
}

// attachPlaces resolves the place (and floor) of each point in one batch,
// leaving Place nil for points whose place is not in the map — the same
// explicit unmapped state as everywhere else in this module.
func (s *service) attachPlaces(ctx context.Context, points []ServicePoint) ([]ServicePoint, error) {
	places, err := s.places.ListPlaces(ctx)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: list places")
	}
	byID := make(map[string]hospitalmap.Place, len(places))
	for _, place := range places {
		byID[place.ID] = place
	}
	for i := range points {
		if place, ok := byID[points[i].PlaceID]; ok {
			place := place
			points[i].Place = &place
		}
	}
	return points, nil
}
