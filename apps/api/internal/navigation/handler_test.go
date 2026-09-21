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

// The queue-console assignment reads (#102) never run in this suite; the
// stubs exist only to satisfy the grown interface.
func (f *fakeServicePoints) ListForUser(context.Context, string) ([]servicepoint.ServicePoint, error) {
	panic("not implemented in fake")
}

func (f *fakeServicePoints) IsAssigned(context.Context, string, string) (bool, error) {
	panic("not implemented in fake")
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

// stairLiftGraph is the cross-floor miniature: reception on I-1301, the
// pharmacy on I-1302, a short stairs transition and a long lift one — the
// unrestricted route prefers the stairs, the accessible route must detour
// (mirrors migrations 000008's real seed shape).
func stairLiftGraph() *fakeRepo {
	return &fakeRepo{
		nodes: map[string]navigation.NavNode{
			"I-1301/node-reception": {ID: "I-1301/node-reception", FloorID: "I-1301", X: 150, Y: 190, NodeType: "PLACE_ENTRY"},
			"I-1301/node-stairs":    {ID: "I-1301/node-stairs", FloorID: "I-1301", X: 400, Y: 190, NodeType: "STAIRS"},
			"I-1301/node-lift":      {ID: "I-1301/node-lift", FloorID: "I-1301", X: 100, Y: 400, NodeType: "ELEVATOR"},
			"I-1302/node-stairs":    {ID: "I-1302/node-stairs", FloorID: "I-1302", X: 400, Y: 290, NodeType: "STAIRS"},
			"I-1302/node-lift":      {ID: "I-1302/node-lift", FloorID: "I-1302", X: 100, Y: 500, NodeType: "ELEVATOR"},
			"I-1302/node-pharmacy":  {ID: "I-1302/node-pharmacy", FloorID: "I-1302", X: 400, Y: 90, NodeType: "PLACE_ENTRY"},
		},
		edges: []navigation.NavEdge{
			{ID: "reception>stairs", FromNodeID: "I-1301/node-reception", ToNodeID: "I-1301/node-stairs", EdgeType: "CORRIDOR", Distance: 250, Accessible: true},
			{ID: "stairs>reception", FromNodeID: "I-1301/node-stairs", ToNodeID: "I-1301/node-reception", EdgeType: "CORRIDOR", Distance: 250, Accessible: true},
			{ID: "reception>lift", FromNodeID: "I-1301/node-reception", ToNodeID: "I-1301/node-lift", EdgeType: "CORRIDOR", Distance: 220, Accessible: true},
			{ID: "lift>reception", FromNodeID: "I-1301/node-lift", ToNodeID: "I-1301/node-reception", EdgeType: "CORRIDOR", Distance: 220, Accessible: true},
			{ID: "stairs>x", FromNodeID: "I-1301/node-stairs", ToNodeID: "I-1302/node-stairs", EdgeType: "STAIRS", Distance: 60, Accessible: false},
			{ID: "x>stairs", FromNodeID: "I-1302/node-stairs", ToNodeID: "I-1301/node-stairs", EdgeType: "STAIRS", Distance: 60, Accessible: false},
			{ID: "lift>x", FromNodeID: "I-1301/node-lift", ToNodeID: "I-1302/node-lift", EdgeType: "ELEVATOR", Distance: 200, Accessible: true},
			{ID: "x>lift", FromNodeID: "I-1302/node-lift", ToNodeID: "I-1301/node-lift", EdgeType: "ELEVATOR", Distance: 200, Accessible: true},
			{ID: "stairs2>pharmacy", FromNodeID: "I-1302/node-stairs", ToNodeID: "I-1302/node-pharmacy", EdgeType: "CORRIDOR", Distance: 200, Accessible: true},
			{ID: "pharmacy>stairs2", FromNodeID: "I-1302/node-pharmacy", ToNodeID: "I-1302/node-stairs", EdgeType: "CORRIDOR", Distance: 200, Accessible: true},
			{ID: "lift2>pharmacy", FromNodeID: "I-1302/node-lift", ToNodeID: "I-1302/node-pharmacy", EdgeType: "CORRIDOR", Distance: 400, Accessible: true},
			{ID: "pharmacy>lift2", FromNodeID: "I-1302/node-pharmacy", ToNodeID: "I-1302/node-lift", EdgeType: "CORRIDOR", Distance: 400, Accessible: true},
		},
	}
}

// #99 through the HTTP boundary: accessibleOnly=true must reach the
// router's RouteOptions — the returned cross-floor route takes the longer
// lift transition and carries no STAIRS segment, while the default query
// keeps the shorter stairs path.
func TestRouteAccessibleOnlyQueryAvoidsStairs(t *testing.T) {
	app := newTestApp(t, stairLiftGraph(), "I-1302/node-pharmacy")

	_, body := getRoute(t, app, "?from=I-1301/node-reception&to=PHARMACY")
	var defaultRoute navigation.Route
	if err := json.Unmarshal(body, &defaultRoute); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	if defaultRoute.TotalDistance != 510 { // 250 + 60 + 200
		t.Fatalf("default totalDistance = %v, want 510 (stairs)", defaultRoute.TotalDistance)
	}

	resp, body := getRoute(t, app, "?from=I-1301/node-reception&to=PHARMACY&accessibleOnly=true")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d (%s), want 200", resp.StatusCode, body)
	}
	var accessible navigation.Route
	if err := json.Unmarshal(body, &accessible); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	for _, segment := range accessible.Segments {
		if segment.EdgeType == "STAIRS" {
			t.Fatalf("accessible route still walks %q (%s)", segment.ID, segment.EdgeType)
		}
	}
	if accessible.TotalDistance != 820 { // 220 + 200 + 400
		t.Fatalf("accessible totalDistance = %v, want 820 (lift detour)", accessible.TotalDistance)
	}
}

// A present-but-unparseable flag is a typo, not a choice — it must be a 400
// rather than silently routing a wheelchair user over the stairs.
func TestRouteAccessibleOnlyRejectsNonBoolean(t *testing.T) {
	app := newTestApp(t, stairLiftGraph(), "I-1302/node-pharmacy")

	resp, body := getRoute(t, app, "?from=I-1301/node-reception&to=PHARMACY&accessibleOnly=yes")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d (%s), want 400", resp.StatusCode, body)
	}
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	if envelope.Code != "invalid" {
		t.Fatalf("code = %q, want invalid", envelope.Code)
	}
}

// #105/ADR-0015: the graph apps/web used to import from packages/floorplans
// at build time is read from here now. The shape matters as much as the
// content — the web app derives QR sticker payloads from the node list, and
// those stickers go on walls.
func TestGraphEndpointsExposeNodesAndEdges(t *testing.T) {
	app := newTestApp(t, routableGraph(), "I-1301/node-pharmacy")

	t.Run("nodes", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/navigation/nodes", nil))
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)

		var nodes []map[string]any
		if err := json.Unmarshal(body, &nodes); err != nil {
			t.Fatalf("decode: %v (%s)", err, body)
		}
		if len(nodes) != 3 {
			t.Fatalf("got %d nodes, want 3", len(nodes))
		}
		// Ids stay floor-prefixed and globally unique: node-lift exists on
		// both floors, so a floor-local id would collide in a QR payload.
		for _, node := range nodes {
			for _, key := range []string{"id", "floorId", "x", "y", "nodeType"} {
				if _, ok := node[key]; !ok {
					t.Errorf("node %v is missing %q", node["id"], key)
				}
			}
		}
	})

	t.Run("edges", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/navigation/edges", nil))
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)

		var edges []map[string]any
		if err := json.Unmarshal(body, &edges); err != nil {
			t.Fatalf("decode: %v (%s)", err, body)
		}
		// Both directions of both connections: a two-way corridor is two
		// rows, and a client assuming otherwise would draw one-way halls.
		if len(edges) != 4 {
			t.Fatalf("got %d edges, want 4 (two connections, both directions)", len(edges))
		}
		if _, ok := edges[0]["accessible"]; !ok {
			t.Error("edges omit accessible — the wheelchair route (FR-20) depends on it")
		}
	})
}
