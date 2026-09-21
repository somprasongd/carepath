package hospitalmap

import "context"

// Service is the only entry point other modules may call. Its methods take a
// plain ctx: when the caller opened a transaction, repository calls made here
// join it through that ctx.
type Service interface {
	GetPlace(ctx context.Context, placeID string) (Place, error)
	ListPlaces(ctx context.Context) ([]Place, error)
	// ListFloors returns every floor in display order. apps/web used to
	// hardcode this list (FLOOR_CODES) next to its bundled plans; since
	// ADR-0015 the plans are served, so the floor list has to be too.
	ListFloors(ctx context.Context) ([]Floor, error)
	GetFloor(ctx context.Context, floorID string) (Floor, error)
	// SetPlan points the floor at a plan and fixes its coordinate space.
	// The floorplan module calls this inside its upload transaction rather
	// than writing carepath.floor itself.
	SetPlan(ctx context.Context, floorID, viewBox, planID string) error
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

func (s *service) ListFloors(ctx context.Context) ([]Floor, error) {
	return s.repo.ListFloors(ctx)
}

func (s *service) GetFloor(ctx context.Context, floorID string) (Floor, error) {
	return s.repo.GetFloor(ctx, floorID)
}

func (s *service) SetPlan(ctx context.Context, floorID, viewBox, planID string) error {
	return s.repo.SetPlan(ctx, floorID, viewBox, planID)
}
