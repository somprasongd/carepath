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
