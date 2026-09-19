package hospitalmap

import "context"

// Service is the only entry point other modules may call. Its methods take a
// plain ctx: when the caller opened a transaction, repository calls made here
// join it through that ctx.
type Service interface {
	GetPlace(ctx context.Context, placeID string) (Place, error)
	ListPlaces(ctx context.Context) ([]Place, error)
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) GetPlace(ctx context.Context, placeID string) (Place, error) {
	return s.repo.GetPlace(ctx, placeID)
}

func (s *service) ListPlaces(ctx context.Context) ([]Place, error) {
	return s.repo.ListPlaces(ctx)
}
