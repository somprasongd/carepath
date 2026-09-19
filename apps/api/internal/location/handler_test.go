package location_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/location/qr"
	"carepath/apps/api/internal/location/zigbee"
	"carepath/apps/api/internal/navigation"
)

// fakePlaces backs the real QR provider in these tests: REG-01 is routable,
// anything else is not on the map.
type fakePlaces struct{}

func (f *fakePlaces) GetPlace(_ context.Context, placeID string) (hospitalmap.Place, error) {
	if placeID == "REG-01" {
		node := "I-1301/node-reception"
		return hospitalmap.Place{ID: placeID, FloorID: "I-1301", Name: "Reception", EntryNodeID: &node}, nil
	}
	return hospitalmap.Place{}, hospitalmap.ErrPlaceNotFound
}

func (f *fakePlaces) ListPlaces(context.Context) ([]hospitalmap.Place, error) { return nil, nil }

// The handler is exercised with the real QR provider over a fake hospital
// map — the HTTP boundary is what matters here: status codes, error
// mapping, and payload shape.
func newTestApp(t *testing.T) (*fiber.App, *fakeRepo) {
	t.Helper()
	repo := &fakeRepo{}
	svc, err := location.NewService(repo, &fakeNavigation{}, qr.New(&fakePlaces{}))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	app := fiber.New()
	location.NewHandler(svc).Register(app.Group("/api/v1"))
	return app, repo
}

func postLocation(t *testing.T, app *fiber.App, visitID, body string) (*http.Response, location.Observation) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/journeys/"+visitID+"/location",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	var obs location.Observation
	if resp.StatusCode == http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(raw, &obs); err != nil {
			t.Fatalf("decode response %q: %v", raw, err)
		}
	}
	return resp, obs
}

func TestReportScan(t *testing.T) {
	app, repo := newTestApp(t)

	resp, obs := postLocation(t, app, "VISIT-001",
		`{"source":"QR","raw":"https://carepath.example/location/REG-01"}`)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (%s), want 200", resp.StatusCode, body)
	}
	if obs.VisitID != "VISIT-001" || obs.NodeID != "I-1301/node-reception" || obs.FloorID != "I-1301" {
		t.Fatalf("obs = %+v, want the canonical reception fix for VISIT-001", obs)
	}
	if obs.Source != location.SourceQR {
		t.Fatalf("obs.Source = %q, want QR", obs.Source)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("repo.recorded = %d entries, want 1", len(repo.recorded))
	}

	// GET returns what the scan recorded — the route start point.
	get, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/journeys/VISIT-001/location", nil))
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if get.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", get.StatusCode)
	}
	raw, _ := io.ReadAll(get.Body)
	var current location.Observation
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatalf("decode GET response %q: %v", raw, err)
	}
	if current.NodeID != obs.NodeID || current.Source != location.SourceQR || current.VisitID != "VISIT-001" {
		t.Fatalf("GET = %+v, want the reported observation", current)
	}
}

// AC #4 through the HTTP boundary: a scan that does not resolve must come
// back as a handled 400, not a 500.
func TestReportUnresolvableScan(t *testing.T) {
	app, _ := newTestApp(t)

	tests := []struct {
		name string
		body string
	}{
		{"malformed json", `{"source":`},
		{"unknown source", `{"source":"SMOKE_SIGNAL","raw":"x"}`},
		{"payload without a location marker", `{"source":"QR","raw":"not-a-location-payload"}`},
		{"stale QR for a removed place", `{"source":"QR","raw":"https://carepath.example/location/ROOM-404"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := postLocation(t, app, "VISIT-001", tt.body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestCurrentBeforeAnyScan(t *testing.T) {
	app, _ := newTestApp(t)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/journeys/VISIT-NEW/location", nil))
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 before any scan", resp.StatusCode)
	}
}

// demoGraph is the slice of the seeded navigation graph the Zigbee
// simulator resolves against: PUBLIC holds an entrance, a corridor, and two
// place entries; PHARMACY one place entry.
var (
	zonePublic   = "PUBLIC"
	zonePharmacy = "PHARMACY"
	demoGraph    = []navigation.NavNode{
		{ID: "I-1301/node-main-entrance", FloorID: "I-1301", NodeType: "ENTRANCE", Zone: &zonePublic},
		{ID: "I-1301/node-ramp", FloorID: "I-1301", NodeType: "CORRIDOR", Zone: &zonePublic},
		{ID: "I-1301/node-cashier", FloorID: "I-1301", NodeType: "PLACE_ENTRY", Zone: &zonePublic},
		{ID: "I-1301/node-reception", FloorID: "I-1301", NodeType: "PLACE_ENTRY", Zone: &zonePublic},
		{ID: "I-1301/node-pharmacy", FloorID: "I-1301", NodeType: "PLACE_ENTRY", Zone: &zonePharmacy},
		{ID: "I-1302/node-lift", FloorID: "I-1302", NodeType: "ELEVATOR"},
	}
)

// The demo app wires the real Zigbee simulator provider so the endpoint
// tests exercise the same path production does — handler-composed fix to
// provider to canonical observation.
func newDemoTestApp(t *testing.T) (*fiber.App, *fakeRepo) {
	t.Helper()
	repo := &fakeRepo{}
	nav := &fakeNavigation{nodes: demoGraph}
	svc, err := location.NewService(repo, nav, qr.New(&fakePlaces{}), zigbee.New(nav))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	app := fiber.New()
	handler := location.NewHandler(svc)
	handler.Register(app.Group("/api/v1"))
	handler.RegisterDemo(app.Group("/api/v1"))
	return app, repo
}

func postZigbeeFix(t *testing.T, app *fiber.App, body string) (*http.Response, location.Observation) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/demo/zigbee/location", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	var obs location.Observation
	if resp.StatusCode == http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(raw, &obs); err != nil {
			t.Fatalf("decode response %q: %v", raw, err)
		}
	}
	return resp, obs
}

// AC #1 of #33 through the HTTP boundary: a zone update posted to the demo
// endpoint becomes the visit's current Zigbee observation, resolved to the
// zone's representative node.
func TestSimulateZigbeeZoneFix(t *testing.T) {
	app, repo := newDemoTestApp(t)

	resp, obs := postZigbeeFix(t, app,
		`{"visitId":"VISIT-Z","floorId":"I-1301","zone":"PUBLIC","confidence":0.8}`)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (%s), want 200", resp.StatusCode, body)
	}
	if obs.VisitID != "VISIT-Z" || obs.NodeID != "I-1301/node-cashier" || obs.FloorID != "I-1301" {
		t.Fatalf("obs = %+v, want the PUBLIC representative node for VISIT-Z", obs)
	}
	if obs.Zone == nil || *obs.Zone != "PUBLIC" {
		t.Fatalf("obs.Zone = %v, want PUBLIC", obs.Zone)
	}
	if obs.Source != location.SourceZigbee {
		t.Fatalf("obs.Source = %q, want ZIGBEE", obs.Source)
	}
	if obs.Confidence == nil || *obs.Confidence != 0.8 {
		t.Fatalf("obs.Confidence = %v, want 0.8", obs.Confidence)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("repo.recorded = %d entries, want 1", len(repo.recorded))
	}
	if got := repo.recorded[0]; got.VisitID != obs.VisitID || got.NodeID != obs.NodeID || got.Source != obs.Source {
		t.Fatalf("repo.recorded[0] = %+v, want the returned observation", got)
	}

	// The simulator wrote through the canonical store: the visit's current
	// location is now the Zigbee fix.
	get, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/journeys/VISIT-Z/location", nil))
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	raw, _ := io.ReadAll(get.Body)
	var current location.Observation
	if err := json.Unmarshal(raw, &current); err != nil {
		t.Fatalf("decode GET response %q: %v", raw, err)
	}
	if current.NodeID != obs.NodeID || current.Source != location.SourceZigbee {
		t.Fatalf("GET = %+v, want the simulated observation", current)
	}
}

func TestSimulateZigbeeInvalidFixes(t *testing.T) {
	app, _ := newDemoTestApp(t)

	tests := []struct {
		name string
		body string
	}{
		{"malformed json", `{"zone":`},
		{"missing visit", `{"floorId":"I-1301","zone":"PUBLIC"}`},
		{"missing zone", `{"visitId":"VISIT-Z","floorId":"I-1301"}`},
		{"unknown zone", `{"visitId":"VISIT-Z","floorId":"I-1301","zone":"ICU"}`},
		{"zone on wrong floor", `{"visitId":"VISIT-Z","floorId":"I-1302","zone":"PUBLIC"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := postZigbeeFix(t, app, tt.body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}
