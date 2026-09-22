package navigation

import (
	"context"
	"slices"

	"carepath/apps/api/internal/hospitalmap"
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
	// RouteToPlace routes to a place's entry node directly (#109): amenity
	// places have no service point, so the code→place bridge
	// RouteToServicePoint walks does not exist for them — but the
	// place→entry-node hop is the same. Error contract matches the service
	// point variant: an unknown place stays hospitalmap.ErrPlaceNotFound, a
	// place without an entry node is ErrDestinationUnmapped.
	RouteToPlace(ctx context.Context, fromNodeID, placeID string, opts RouteOptions) (Route, error)
	// NearestAmenities ranks every amenity-typed place (hospitalmap's
	// AmenityPlaceTypes) by walking distance from one origin — one graph
	// settle answers the whole search (#109 / FR-26). An unknown origin is
	// the caller's error (ErrNodeNotFound); an amenity that cannot be
	// reached is simply absent from the list.
	NearestAmenities(ctx context.Context, fromNodeID string, opts RouteOptions, limit int) (AmenitySearch, error)
	// DistancesToServicePoints resolves each code (code → place → entry
	// node, the same bridge RouteToServicePoint uses) and returns its
	// walking distance from fromNodeID — one graph pass for many
	// destinations (#103's recommendation ranks a visit's actionable steps
	// from one patient position). Codes that cannot be resolved or reached
	// are simply absent from the map: ranking treats missing data as "no
	// distance", not as an error.
	DistancesToServicePoints(ctx context.Context, fromNodeID string, servicePointCodes []string, opts RouteOptions) (map[string]float64, error)
}

type service struct {
	repo          Repo
	servicePoints servicepoint.Service
	places        hospitalmap.Service
}

func NewService(repo Repo, servicePoints servicepoint.Service, places hospitalmap.Service) Service {
	return &service{repo: repo, servicePoints: servicePoints, places: places}
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

// RouteToPlace resolves the destination through the hospitalmap module
// (place → entry node) and routes there. Resolution errors keep their
// classification: an unknown place stays hospitalmap.ErrPlaceNotFound, and
// a place with no entry node is ErrDestinationUnmapped — both surface as
// 404 to the client, same as the service point variant.
func (s *service) RouteToPlace(ctx context.Context, fromNodeID, placeID string, opts RouteOptions) (Route, error) {
	place, err := s.places.GetPlace(ctx, placeID)
	if err != nil {
		return Route{}, err
	}
	if place.EntryNodeID == nil {
		return Route{}, ErrDestinationUnmapped
	}
	return s.Route(ctx, fromNodeID, *place.EntryNodeID, opts)
}

// NearestAmenities loads the graph once, settles it from the origin, and
// ranks the amenity-typed places by their entry nodes' settled distance.
// Places without an entry node or off the reachable component are dropped
// by rankAmenities — "no distance" is a ranking fact, not an error.
func (s *service) NearestAmenities(ctx context.Context, fromNodeID string, opts RouteOptions, limit int) (AmenitySearch, error) {
	nodes, err := s.repo.ListNodes(ctx)
	if err != nil {
		return AmenitySearch{}, err
	}
	edges, err := s.repo.ListEdges(ctx)
	if err != nil {
		return AmenitySearch{}, err
	}
	distances, err := shortestDistances(nodes, edges, fromNodeID, opts)
	if err != nil {
		return AmenitySearch{}, err
	}
	all, err := s.places.ListPlaces(ctx)
	if err != nil {
		return AmenitySearch{}, err
	}
	amenities := make([]hospitalmap.Place, 0, len(all))
	for _, place := range all {
		if slices.Contains(hospitalmap.AmenityPlaceTypes, place.PlaceType) {
			amenities = append(amenities, place)
		}
	}
	ranked := rankAmenities(amenities, distances)
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return AmenitySearch{From: fromNodeID, Amenities: ranked}, nil
}

// DistancesToServicePoints loads the graph once, settles it from the origin,
// and looks each code's entry node up in the result. An unknown origin is
// the caller's error (ErrNodeNotFound) and aborts the whole call; a code
// that resolves to nothing walkable is absent from the map by design.
func (s *service) DistancesToServicePoints(ctx context.Context, fromNodeID string, servicePointCodes []string, opts RouteOptions) (map[string]float64, error) {
	nodes, err := s.repo.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	edges, err := s.repo.ListEdges(ctx)
	if err != nil {
		return nil, err
	}
	distances, err := shortestDistances(nodes, edges, fromNodeID, opts)
	if err != nil {
		return nil, err
	}
	out := make(map[string]float64, len(servicePointCodes))
	for _, code := range servicePointCodes {
		sp, err := s.servicePoints.GetByCode(ctx, code)
		if err != nil {
			continue
		}
		if sp.Place == nil || sp.Place.EntryNodeID == nil {
			continue
		}
		if d, ok := distances[*sp.Place.EntryNodeID]; ok {
			out[code] = d
		}
	}
	return out, nil
}
