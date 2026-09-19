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
