package navigation

import (
	"context"

	"carepath/apps/api/internal/servicepoint"
)

// Service is the only entry point other modules may call. Route computes
// the shortest path on top of the graph reads; RouteToServicePoint resolves
// a destination service point to its place's entry node first (#28) — the
// service point is the designed care↔place bridge, so this hop does not
// pull care flow into the graph (ADR-0002). Its methods take a plain ctx:
// when the caller opened a transaction, repository calls made here join it
// through that ctx.
type Service interface {
	GetNode(ctx context.Context, nodeID string) (NavNode, error)
	ListNodes(ctx context.Context) ([]NavNode, error)
	ListEdges(ctx context.Context) ([]NavEdge, error)
	Route(ctx context.Context, fromNodeID, toNodeID string, opts RouteOptions) (Route, error)
	RouteToServicePoint(ctx context.Context, fromNodeID, servicePointCode string, opts RouteOptions) (Route, error)
}

type service struct {
	repo          Repo
	servicePoints servicepoint.Service
}

func NewService(repo Repo, servicePoints servicepoint.Service) Service {
	return &service{repo: repo, servicePoints: servicePoints}
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

// RouteToServicePoint resolves the destination through the servicepoint
// module (code → place → entry node) and routes there. Resolution errors
// keep their classification: an unknown code stays servicepoint.ErrNotFound,
// and a service point with no mapped place or entry node is
// ErrDestinationUnmapped — both surface as 404 to the client.
func (s *service) RouteToServicePoint(ctx context.Context, fromNodeID, servicePointCode string, opts RouteOptions) (Route, error) {
	sp, err := s.servicePoints.GetByCode(ctx, servicePointCode)
	if err != nil {
		return Route{}, err
	}
	if sp.Place == nil || sp.Place.EntryNodeID == nil {
		return Route{}, ErrDestinationUnmapped
	}
	return s.Route(ctx, fromNodeID, *sp.Place.EntryNodeID, opts)
}
