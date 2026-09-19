package navigation_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/servicepoint"
)

// fakeRepo is the seeded ground floor in miniature: reception → corridor →
// pharmacy, all two-way (mirrors migrations 000007/000008).
type fakeRepo struct {
	nodes map[string]navigation.NavNode
	edges []navigation.NavEdge
}

func (f *fakeRepo) GetNode(_ context.Context, nodeID string) (navigation.NavNode, error) {
	node, ok := f.nodes[nodeID]
	if !ok {
		return navigation.NavNode{}, navigation.ErrNodeNotFound
	}
	return node, nil
}

func (f *fakeRepo) ListNodes(context.Context) ([]navigation.NavNode, error) {
	var nodes []navigation.NavNode
	for _, node := range f.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (f *fakeRepo) ListEdges(context.Context) ([]navigation.NavEdge, error) {
	return f.edges, nil
}

// fakeServicePoints resolves code PHARMACY to its place's entry node; every
// other code is not found, the way the servicepoint module answers.
type fakeServicePoints struct {
	entry string
}

func (f *fakeServicePoints) GetByCode(_ context.Context, code string) (servicepoint.ServicePoint, error) {
	if code != "PHARMACY" {
		return servicepoint.ServicePoint{}, servicepoint.ErrNotFound
	}
	entry := f.entry
	return servicepoint.ServicePoint{
		ID: "SP-PHARMACY", Code: "PHARMACY", Name: "Pharmacy", PlaceID: "PHARMACY-01",
		Place: &hospitalmap.Place{ID: "PHARMACY-01", FloorID: "I-1301", Name: "Pharmacy", EntryNodeID: &entry},
	}, nil
}

func (f *fakeServicePoints) List(context.Context) ([]servicepoint.ServicePoint, error) {
	return nil, nil
}

func newTestApp(t *testing.T, repo navigation.Repo, pharmacyEntryNodeID string) *fiber.App {
	t.Helper()
	app := fiber.New()
	navigation.NewHandler(navigation.NewService(repo, &fakeServicePoints{entry: pharmacyEntryNodeID})).
		Register(app.Group("/api/v1"))
	return app
}

func routableGraph() *fakeRepo {
	return &fakeRepo{
		nodes: map[string]navigation.NavNode{
			"I-1301/node-reception": {ID: "I-1301/node-reception", FloorID: "I-1301", X: 150, Y: 190, NodeType: "PLACE_ENTRY"},
			"I-1301/node-corridor":  {ID: "I-1301/node-corridor", FloorID: "I-1301", X: 500, Y: 190, NodeType: "CORRIDOR"},
			"I-1301/node-pharmacy":  {ID: "I-1301/node-pharmacy", FloorID: "I-1301", X: 885, Y: 190, NodeType: "PLACE_ENTRY"},
		},
		edges: []navigation.NavEdge{
			{ID: "reception>corridor", FromNodeID: "I-1301/node-reception", ToNodeID: "I-1301/node-corridor", EdgeType: "CORRIDOR", Distance: 350, Accessible: true},
			{ID: "corridor>reception", FromNodeID: "I-1301/node-corridor", ToNodeID: "I-1301/node-reception", EdgeType: "CORRIDOR", Distance: 350, Accessible: true},
			{ID: "corridor>pharmacy", FromNodeID: "I-1301/node-corridor", ToNodeID: "I-1301/node-pharmacy", EdgeType: "CORRIDOR", Distance: 385, Accessible: true},
			{ID: "pharmacy>corridor", FromNodeID: "I-1301/node-pharmacy", ToNodeID: "I-1301/node-corridor", EdgeType: "CORRIDOR", Distance: 385, Accessible: true},
		},
	}
}

func getRoute(t *testing.T, app *fiber.App, query string) (*http.Response, []byte) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/navigation/route"+query, nil))
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, body
}

// AC #3 through the HTTP boundary: the response carries floor-local
// coordinates per node and segments that chain node-to-node — enough for the
// SVG overlay to draw a polyline per floor.
func TestRouteReturnsOverlayCoordinates(t *testing.T) {
	app := newTestApp(t, routableGraph(), "I-1301/node-pharmacy")

	resp, body := getRoute(t, app, "?from=I-1301/node-reception&to=PHARMACY")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d (%s), want 200", resp.StatusCode, body)
	}

	var raw struct {
		Nodes         []map[string]any `json:"nodes"`
		Segments      []map[string]any `json:"segments"`
		TotalDistance float64          `json:"totalDistance"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	if len(raw.Nodes) != 3 || len(raw.Segments) != 2 {
		t.Fatalf("got %d nodes / %d segments, want 3/2: %s", len(raw.Nodes), len(raw.Segments), body)
	}
	for _, key := range []string{"id", "floorId", "x", "y", "nodeType"} {
		if _, ok := raw.Nodes[0][key]; !ok {
			t.Fatalf("node payload %v lacks %q", raw.Nodes[0], key)
		}
	}
	for _, key := range []string{"fromNodeId", "toNodeId", "edgeType", "distance", "accessible"} {
		if _, ok := raw.Segments[0][key]; !ok {
			t.Fatalf("segment payload %v lacks %q", raw.Segments[0], key)
		}
	}
	if raw.Nodes[0]["id"] != "I-1301/node-reception" || raw.Nodes[2]["id"] != "I-1301/node-pharmacy" {
		t.Fatalf("route endpoints = %v → %v, want reception → pharmacy", raw.Nodes[0]["id"], raw.Nodes[2]["id"])
	}
	if raw.Segments[0]["fromNodeId"] != raw.Nodes[0]["id"] || raw.Segments[0]["toNodeId"] != raw.Nodes[1]["id"] {
		t.Fatalf("segment 0 = %v → %v, want node0 → node1", raw.Segments[0]["fromNodeId"], raw.Segments[0]["toNodeId"])
	}
	if raw.TotalDistance != 735 {
		t.Fatalf("totalDistance = %v, want 735", raw.TotalDistance)
	}
}

func TestRouteMissingParams(t *testing.T) {
	app := newTestApp(t, routableGraph(), "I-1301/node-pharmacy")

	for _, query := range []string{"", "?from=I-1301/node-reception", "?to=PHARMACY"} {
		resp, body := getRoute(t, app, query)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("query %q: status = %d (%s), want 400", query, resp.StatusCode, body)
		}
		var envelope struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Fatalf("decode %q: %v", body, err)
		}
		if envelope.Code != "invalid" {
			t.Fatalf("query %q: code = %q, want invalid", query, envelope.Code)
		}
	}
}

// AC #4: a destination that cannot be routed comes back as a 404 the
// frontend can branch on via the machine-readable code, not a 500.
func TestRouteUnroutableDestinationHasCode(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"unknown from node", "?from=I-1301/node-nope&to=PHARMACY"},
		{"unknown service point", "?from=I-1301/node-reception&to=LAB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t, routableGraph(), "I-1301/node-pharmacy")

			resp, body := getRoute(t, app, tt.query)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d (%s), want 404", resp.StatusCode, body)
			}
			var envelope struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatalf("decode %q: %v", body, err)
			}
			if envelope.Code != "not_found" {
				t.Fatalf("code = %q, want not_found", envelope.Code)
			}
		})
	}
}

// Two floors with no connecting edge: both nodes exist, but no walkable
// path connects them — ErrNoRoute must surface as 404 not_found too.
func TestRouteNoPathHasCode(t *testing.T) {
	app := newTestApp(t, &fakeRepo{
		nodes: map[string]navigation.NavNode{
			"I-1301/node-reception": {ID: "I-1301/node-reception", FloorID: "I-1301", X: 150, Y: 190, NodeType: "PLACE_ENTRY"},
			"I-1302/node-pharmacy":  {ID: "I-1302/node-pharmacy", FloorID: "I-1302", X: 885, Y: 190, NodeType: "PLACE_ENTRY"},
		},
	}, "I-1302/node-pharmacy")

	resp, body := getRoute(t, app, "?from=I-1301/node-reception&to=PHARMACY")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d (%s), want 404", resp.StatusCode, body)
	}
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	if envelope.Code != "not_found" {
		t.Fatalf("code = %q, want not_found", envelope.Code)
	}
}
