package navigation

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/servicepoint"
)

// fakeRepo mirrors the seed rows from infra/postgres/migrations/000008.
type fakeRepo struct {
	nodes map[string]NavNode
	edges []NavEdge
}

func (f *fakeRepo) GetNode(_ context.Context, nodeID string) (NavNode, error) {
	node, ok := f.nodes[nodeID]
	if !ok {
		return NavNode{}, ErrNodeNotFound
	}
	return node, nil
}

func (f *fakeRepo) ListNodes(context.Context) ([]NavNode, error) {
	var nodes []NavNode
	for _, node := range f.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (f *fakeRepo) ListEdges(context.Context) ([]NavEdge, error) {
	return f.edges, nil
}

func TestGetNodeCarriesFloorAndCoordinate(t *testing.T) {
	svc := NewService(&fakeRepo{nodes: map[string]NavNode{
		"I-1302/node-blood-collection": {
			ID: "I-1302/node-blood-collection", FloorID: "I-1302", X: 165, Y: 405,
			NodeType: "PLACE_ENTRY",
		},
	}}, nil)

	node, err := svc.GetNode(context.Background(), "I-1302/node-blood-collection")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if node.FloorID != "I-1302" || node.X != 165 || node.Y != 405 || node.NodeType != "PLACE_ENTRY" {
		t.Fatalf("node = %+v, want I-1302 at (165,405) PLACE_ENTRY", node)
	}
}

func TestGetNodeUnknownIDPropagatesNotFound(t *testing.T) {
	svc := NewService(&fakeRepo{nodes: map[string]NavNode{}}, nil)

	if _, err := svc.GetNode(context.Background(), "I-1301/node-nope"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("error = %v, want ErrNodeNotFound", err)
	}
}

func TestListEdgesCarriesCost(t *testing.T) {
	svc := NewService(&fakeRepo{edges: []NavEdge{
		{ID: "a>b", FromNodeID: "a", ToNodeID: "b", EdgeType: "CORRIDOR", Distance: 200, Accessible: true},
		{ID: "b>a", FromNodeID: "b", ToNodeID: "a", EdgeType: "CORRIDOR", Distance: 200, Accessible: true},
	}}, nil)

	edges, err := svc.ListEdges(context.Background())
	if err != nil {
		t.Fatalf("ListEdges: %v", err)
	}
	if len(edges) != 2 || edges[0].Distance != 200 || !edges[0].Accessible {
		t.Fatalf("edges = %+v, want 2 directed rows with distance 200 accessible", edges)
	}
}

// fakeServicePoints answers service point lookups from a fixed map, the way
// the servicepoint module would: code → service point with its resolved
// place, or servicepoint.ErrNotFound.
type fakeServicePoints struct {
	byCode map[string]servicepoint.ServicePoint
}

// The queue-console assignment reads (#102) never run in this suite; the
// stubs exist only to satisfy the grown interface.
func (f *fakeServicePoints) ListForUser(context.Context, string) ([]servicepoint.ServicePoint, error) {
	panic("not implemented in fake")
}

func (f *fakeServicePoints) IsAssigned(context.Context, string, string) (bool, error) {
	panic("not implemented in fake")
}

func (f *fakeServicePoints) GetByCode(_ context.Context, code string) (servicepoint.ServicePoint, error) {
	sp, ok := f.byCode[code]
	if !ok {
		return servicepoint.ServicePoint{}, servicepoint.ErrNotFound
	}
	return sp, nil
}

func (f *fakeServicePoints) List(context.Context) ([]servicepoint.ServicePoint, error) {
	var points []servicepoint.ServicePoint
	for _, sp := range f.byCode {
		points = append(points, sp)
	}
	return points, nil
}

// routeTestGraph is the seeded ground floor in miniature: reception →
// corridor → pharmacy, all two-way, mirroring migrations 000007/000008.
func routeTestGraph() *fakeRepo {
	return &fakeRepo{
		nodes: map[string]NavNode{
			"I-1301/node-reception": {ID: "I-1301/node-reception", FloorID: "I-1301", X: 150, Y: 190, NodeType: "PLACE_ENTRY"},
			"I-1301/node-corridor":  {ID: "I-1301/node-corridor", FloorID: "I-1301", X: 500, Y: 190, NodeType: "CORRIDOR"},
			"I-1301/node-pharmacy":  {ID: "I-1301/node-pharmacy", FloorID: "I-1301", X: 885, Y: 190, NodeType: "PLACE_ENTRY"},
		},
		edges: []NavEdge{
			{ID: "reception>corridor", FromNodeID: "I-1301/node-reception", ToNodeID: "I-1301/node-corridor", EdgeType: "CORRIDOR", Distance: 350, Accessible: true},
			{ID: "corridor>reception", FromNodeID: "I-1301/node-corridor", ToNodeID: "I-1301/node-reception", EdgeType: "CORRIDOR", Distance: 350, Accessible: true},
			{ID: "corridor>pharmacy", FromNodeID: "I-1301/node-corridor", ToNodeID: "I-1301/node-pharmacy", EdgeType: "CORRIDOR", Distance: 385, Accessible: true},
			{ID: "pharmacy>corridor", FromNodeID: "I-1301/node-pharmacy", ToNodeID: "I-1301/node-corridor", EdgeType: "CORRIDOR", Distance: 385, Accessible: true},
		},
	}
}

func pharmacyServicePoints(place *hospitalmap.Place) *fakeServicePoints {
	return &fakeServicePoints{byCode: map[string]servicepoint.ServicePoint{
		"PHARMACY": {ID: "SP-PHARMACY", Code: "PHARMACY", Name: "Pharmacy", PlaceID: "PHARMACY-01", Place: place},
	}}
}

// AC #1/#2 of #28: `from` is a navigation node id, `to` a service point
// code, and the route ends at the place's entry node.
func TestRouteToServicePointEndsAtPlaceEntry(t *testing.T) {
	entry := "I-1301/node-pharmacy"
	svc := NewService(routeTestGraph(), pharmacyServicePoints(&hospitalmap.Place{
		ID: "PHARMACY-01", FloorID: "I-1301", Name: "Pharmacy", EntryNodeID: &entry,
	}))

	route, err := svc.RouteToServicePoint(context.Background(), "I-1301/node-reception", "PHARMACY", RouteOptions{})
	if err != nil {
		t.Fatalf("RouteToServicePoint: %v", err)
	}
	if route.Nodes[0].ID != "I-1301/node-reception" {
		t.Fatalf("route starts at %+v, want the reception node", route.Nodes[0])
	}
	if got := route.Nodes[len(route.Nodes)-1].ID; got != entry {
		t.Fatalf("route ends at %s, want the pharmacy entry node", got)
	}
	if len(route.Segments) != len(route.Nodes)-1 || route.TotalDistance != 735 {
		t.Fatalf("route = %d nodes / %d segments / total %v, want 3/2/735", len(route.Nodes), len(route.Segments), route.TotalDistance)
	}
}

func TestRouteToServicePointUnknownCodePropagatesNotFound(t *testing.T) {
	svc := NewService(routeTestGraph(), pharmacyServicePoints(nil))

	if _, err := svc.RouteToServicePoint(context.Background(), "I-1301/node-reception", "XRAY", RouteOptions{}); !errors.Is(err, servicepoint.ErrNotFound) {
		t.Fatalf("error = %v, want servicepoint.ErrNotFound", err)
	}
}

func TestRouteToServicePointUnknownFromNode(t *testing.T) {
	entry := "I-1301/node-pharmacy"
	svc := NewService(routeTestGraph(), pharmacyServicePoints(&hospitalmap.Place{
		ID: "PHARMACY-01", FloorID: "I-1301", Name: "Pharmacy", EntryNodeID: &entry,
	}))

	if _, err := svc.RouteToServicePoint(context.Background(), "I-1301/node-nope", "PHARMACY", RouteOptions{}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("error = %v, want ErrNodeNotFound", err)
	}
}

// A service point that resolves to no place, or to a place with no entry
// node, has nothing to route to — ErrDestinationUnmapped, not a panic.
func TestRouteToServicePointUnmappedDestination(t *testing.T) {
	tests := []struct {
		name  string
		place *hospitalmap.Place
	}{
		{"place missing", nil},
		{"entry node missing", &hospitalmap.Place{ID: "PHARMACY-01", FloorID: "I-1301", Name: "Pharmacy"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(routeTestGraph(), pharmacyServicePoints(tt.place))

			if _, err := svc.RouteToServicePoint(context.Background(), "I-1301/node-reception", "PHARMACY", RouteOptions{}); !errors.Is(err, ErrDestinationUnmapped) {
				t.Fatalf("error = %v, want ErrDestinationUnmapped", err)
			}
		})
	}
}
