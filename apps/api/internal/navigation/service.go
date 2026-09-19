package navigation

import "context"

// Service is the only entry point other modules may call — the routing
// slice (#27) builds its shortest path on these reads. Its methods take a
// plain ctx: when the caller opened a transaction, repository calls made
// here join it through that ctx.
type Service interface {
	GetNode(ctx context.Context, nodeID string) (NavNode, error)
	ListNodes(ctx context.Context) ([]NavNode, error)
	ListEdges(ctx context.Context) ([]NavEdge, error)
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) GetNode(ctx context.Context, nodeID string) (NavNode, error) {
	return s.repo.GetNode(ctx, nodeID)
}

func (s *service) ListNodes(ctx context.Context) ([]NavNode, error) {
	return s.repo.ListNodes(ctx)
}

func (s *service) ListEdges(ctx context.Context) ([]NavEdge, error) {
	return s.repo.ListEdges(ctx)
}
