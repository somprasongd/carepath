package navigation

import "context"

// Service is the only entry point other modules may call. Route computes
// the shortest path on top of the graph reads. Its methods take a plain
// ctx: when the caller opened a transaction, repository calls made here
// join it through that ctx.
type Service interface {
	GetNode(ctx context.Context, nodeID string) (NavNode, error)
	ListNodes(ctx context.Context) ([]NavNode, error)
	ListEdges(ctx context.Context) ([]NavEdge, error)
	Route(ctx context.Context, fromNodeID, toNodeID string, opts RouteOptions) (Route, error)
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

// Route loads the whole graph (a few dozen rows) and runs the shortest-path
// search over it in memory; errors from the reads pass through unchanged so
// their classification survives.
func (s *service) Route(ctx context.Context, fromNodeID, toNodeID string, opts RouteOptions) (Route, error) {
	nodes, err := s.repo.ListNodes(ctx)
	if err != nil {
		return Route{}, err
	}
	edges, err := s.repo.ListEdges(ctx)
	if err != nil {
		return Route{}, err
	}
	return shortestRoute(nodes, edges, fromNodeID, toNodeID, opts)
}
