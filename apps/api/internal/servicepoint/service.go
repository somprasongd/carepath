package servicepoint

import "context"

// Service is the only entry point other modules may call. Its methods take a
// plain ctx: when the caller opened a transaction, repository calls made here
// join it through that ctx.
type Service interface {
	GetByCode(ctx context.Context, code string) (ServicePoint, error)
	List(ctx context.Context) ([]ServicePoint, error)
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) GetByCode(ctx context.Context, code string) (ServicePoint, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *service) List(ctx context.Context) ([]ServicePoint, error) {
	return s.repo.List(ctx)
}
