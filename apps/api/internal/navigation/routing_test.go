package navigation

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/platform/apperr"
)

// routeGraph is a small synthetic hospital: F1/a is the entrance, F1/c the
// destination, reachable either through F1/b (short) or F1/d (long).
func routeGraph() *fakeRepo {
	return &fakeRepo{
		nodes: map[string]NavNode{
			"F1/a": {ID: "F1/a", FloorID: "F1", NodeType: "ENTRANCE"},
			"F1/b": {ID: "F1/b", FloorID: "F1", NodeType: "CORRIDOR"},
			"F1/c": {ID: "F1/c", FloorID: "F1", NodeType: "PLACE_ENTRY"},
			"F1/d": {ID: "F1/d", FloorID: "F1", NodeType: "CORRIDOR"},
		},
		edges: bidirectional([][3]any{
			{"F1/a", "F1/b", 100.0},
			{"F1/b", "F1/c", 50.0},
			{"F1/a", "F1/d", 120.0},
			{"F1/d", "F1/c", 80.0},
		}, true),
	}
}

// bidirectional expands each connection into the two directed rows the seed
// stores, with one shared distance and accessible flag.
func bidirectional(connections [][3]any, accessible bool) []NavEdge {
	var edges []NavEdge
	for _, c := range connections {
		from, to, distance := c[0].(string), c[1].(string), c[2].(float64)
		edges = append(edges,
			NavEdge{ID: from + ">" + to, FromNodeID: from, ToNodeID: to, EdgeType: "CORRIDOR", Distance: distance, Accessible: accessible},
			NavEdge{ID: to + ">" + from, FromNodeID: to, ToNodeID: from, EdgeType: "CORRIDOR", Distance: distance, Accessible: accessible},
		)
	}
	return edges
}

func nodeIDs(route Route) []string {
	ids := make([]string, len(route.Nodes))
	for i, node := range route.Nodes {
		ids[i] = node.ID
	}
	return ids
}

func TestRouteReturnsOrderedShortestPath(t *testing.T) {
	svc := NewService(routeGraph(), nil)

	route, err := svc.Route(context.Background(), "F1/a", "F1/c", RouteOptions{})
	if err != nil {
		t.Fatalf("Route: %v", err)
	}

	want := []string{"F1/a", "F1/b", "F1/c"}
	got := nodeIDs(route)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("path = %v, want %v", got, want)
	}
	if len(route.Segments) != 2 ||
		route.Segments[0].FromNodeID != "F1/a" || route.Segments[0].ToNodeID != "F1/b" ||
		route.Segments[1].FromNodeID != "F1/b" || route.Segments[1].ToNodeID != "F1/c" {
		t.Fatalf("segments = %+v, want a→b→c", route.Segments)
	}
	if route.TotalDistance != 150 {
		t.Fatalf("total distance = %v, want 150", route.TotalDistance)
	}
}

func TestRouteUnknownEndpointIsNotFound(t *testing.T) {
	svc := NewService(routeGraph(), nil)
	ctx := context.Background()

	if _, err := svc.Route(ctx, "F1/a", "F1/missing", RouteOptions{}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("unknown destination: error = %v, want ErrNodeNotFound", err)
	}
	if _, err := svc.Route(ctx, "F1/missing", "F1/c", RouteOptions{}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("unknown origin: error = %v, want ErrNodeNotFound", err)
	}
}

func TestRouteUnreachableDestinationIsNoRoute(t *testing.T) {
	repo := routeGraph()
	repo.nodes["F2/z"] = NavNode{ID: "F2/z", FloorID: "F2", NodeType: "CORRIDOR"} // no edges at all
	svc := NewService(repo, nil)
	ctx := context.Background()

	if _, err := svc.Route(ctx, "F1/a", "F2/z", RouteOptions{}); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("unreachable destination: error = %v, want ErrNoRoute", err)
	}
	if _, err := svc.Route(ctx, "F2/z", "F1/a", RouteOptions{}); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("unreachable origin: error = %v, want ErrNoRoute", err)
	}
}

func TestRouteAccessibleOnlyAvoidsStairs(t *testing.T) {
	repo := &fakeRepo{
		nodes: map[string]NavNode{
			"F1/a": {ID: "F1/a", FloorID: "F1", NodeType: "ENTRANCE"},
			"F1/s": {ID: "F1/s", FloorID: "F1", NodeType: "STAIRS"},
			"F1/l": {ID: "F1/l", FloorID: "F1", NodeType: "ELEVATOR"},
			"F2/c": {ID: "F2/c", FloorID: "F2", NodeType: "PLACE_ENTRY"},
			"F2/s": {ID: "F2/s", FloorID: "F2", NodeType: "STAIRS"},
			"F2/l": {ID: "F2/l", FloorID: "F2", NodeType: "ELEVATOR"},
		},
		edges: append(
			bidirectional([][3]any{{"F1/a", "F1/s", 10.0}, {"F1/s", "F2/s", 10.0}, {"F2/s", "F2/c", 10.0}}, false),
			bidirectional([][3]any{{"F1/a", "F1/l", 40.0}, {"F1/l", "F2/l", 40.0}, {"F2/l", "F2/c", 40.0}}, true)...,
		),
	}
	svc := NewService(repo, nil)
	ctx := context.Background()

	plain, err := svc.Route(ctx, "F1/a", "F2/c", RouteOptions{})
	if err != nil {
		t.Fatalf("Route without options: %v", err)
	}
	if plain.TotalDistance != 30 || len(plain.Nodes) != 4 {
		t.Fatalf("plain route = %v (%v), want the short stairs path of 30", nodeIDs(plain), plain.TotalDistance)
	}

	accessible, err := svc.Route(ctx, "F1/a", "F2/c", RouteOptions{AccessibleOnly: true})
	if err != nil {
		t.Fatalf("Route accessible-only: %v", err)
	}
	if accessible.TotalDistance != 120 || len(accessible.Nodes) != 4 {
		t.Fatalf("accessible route = %v (%v), want the elevator detour of 120", nodeIDs(accessible), accessible.TotalDistance)
	}
	for _, segment := range accessible.Segments {
		if segment.EdgeType == "STAIRS" || !segment.Accessible {
			t.Fatalf("accessible route uses %v, want elevator segments only", segment.ID)
		}
	}

	if _, err := svc.Route(ctx, "F2/s", "F1/a", RouteOptions{AccessibleOnly: true}); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("stairs-only island with AccessibleOnly: error = %v, want ErrNoRoute", err)
	}
}

func TestRouteToSelfIsTrivial(t *testing.T) {
	svc := NewService(routeGraph(), nil)

	route, err := svc.Route(context.Background(), "F1/a", "F1/a", RouteOptions{})
	if err != nil {
		t.Fatalf("Route to self: %v", err)
	}
	if len(route.Nodes) != 1 || route.Nodes[0].ID != "F1/a" || len(route.Segments) != 0 || route.TotalDistance != 0 {
		t.Fatalf("route = %+v, want the single origin node with no segments", route)
	}
}

func TestRouteRejectsNegativeEdgeDistance(t *testing.T) {
	repo := routeGraph()
	repo.edges = append(repo.edges, NavEdge{ID: "F1/b>F1/d", FromNodeID: "F1/b", ToNodeID: "F1/d", EdgeType: "CORRIDOR", Distance: -1})
	svc := NewService(repo, nil)

	if _, err := svc.Route(context.Background(), "F1/a", "F1/c", RouteOptions{}); apperr.KindOf(err) != apperr.KindInternal {
		t.Fatalf("error = %v, want an internal-kind error for corrupt distance", err)
	}
}

func TestShortestDistancesSettlesWholeGraph(t *testing.T) {
	graph := routeGraph()
	nodes := make([]NavNode, 0, len(graph.nodes))
	for _, node := range graph.nodes {
		nodes = append(nodes, node)
	}

	dist, err := shortestDistances(nodes, graph.edges, "F1/a", RouteOptions{})
	if err != nil {
		t.Fatalf("shortestDistances: %v", err)
	}
	want := map[string]float64{"F1/a": 0, "F1/b": 100, "F1/c": 150, "F1/d": 120}
	if len(dist) != len(want) {
		t.Fatalf("distances = %v, want exactly %v", dist, want)
	}
	for id, d := range want {
		if dist[id] != d {
			t.Fatalf("dist[%s] = %v, want %v", id, dist[id], d)
		}
	}
}

func TestShortestDistancesOmitsUnreachableNodes(t *testing.T) {
	graph := routeGraph()
	graph.nodes["F1/x"] = NavNode{ID: "F1/x", FloorID: "F1", NodeType: "CORRIDOR"} // no edges
	nodes := make([]NavNode, 0, len(graph.nodes))
	for _, node := range graph.nodes {
		nodes = append(nodes, node)
	}

	dist, err := shortestDistances(nodes, graph.edges, "F1/a", RouteOptions{})
	if err != nil {
		t.Fatalf("shortestDistances: %v", err)
	}
	if _, ok := dist["F1/x"]; ok {
		t.Fatalf("unreachable F1/x present with %v", dist["F1/x"])
	}
	if _, ok := dist["F1/c"]; !ok {
		t.Fatal("reachable F1/c absent")
	}
}

func TestShortestDistancesUnknownOriginIsNotFound(t *testing.T) {
	graph := routeGraph()
	nodes := make([]NavNode, 0, len(graph.nodes))
	for _, node := range graph.nodes {
		nodes = append(nodes, node)
	}

	if _, err := shortestDistances(nodes, graph.edges, "F1/nope", RouteOptions{}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("error = %v, want ErrNodeNotFound", err)
	}
}

func TestShortestDistancesRejectsNegativeEdgeDistance(t *testing.T) {
	graph := routeGraph()
	graph.edges = append(graph.edges, NavEdge{ID: "F1/a>F1/b-neg", FromNodeID: "F1/a", ToNodeID: "F1/b", Distance: -1})
	nodes := make([]NavNode, 0, len(graph.nodes))
	for _, node := range graph.nodes {
		nodes = append(nodes, node)
	}

	if _, err := shortestDistances(nodes, graph.edges, "F1/a", RouteOptions{}); apperr.KindOf(err) != apperr.KindInternal {
		t.Fatalf("error = %v, want KindInternal for a negative edge", err)
	}
}
