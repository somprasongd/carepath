package servicepoint

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// newTestApp mounts the handler over the real service backed by the same
// fakes the service tests use.
func newTestApp() *fiber.App {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{places: seedPlaces()})
	app := fiber.New()
	NewHandler(svc).Register(app.Group("/api/v1"))
	return app
}

func TestListEndpointReturnsPointsWithPlaceAndFloor(t *testing.T) {
	app := newTestApp()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/service-points", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var points []map[string]any
	if err := json.Unmarshal(body, &points); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}
	for _, point := range points {
		place, ok := point["place"].(map[string]any)
		if !ok {
			t.Fatalf("point %v has no place", point)
		}
		if _, ok := place["floor"].(map[string]any); !ok {
			t.Fatalf("place %v has no floor", place)
		}
	}
}

func TestGetByCodeEndpointResolvesDestination(t *testing.T) {
	app := newTestApp()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/service-points/LAB", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var sp map[string]any
	if err := json.Unmarshal(body, &sp); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
	place, ok := sp["place"].(map[string]any)
	if !ok {
		t.Fatalf("service point %v has no place", sp)
	}
	floor, ok := place["floor"].(map[string]any)
	if !ok {
		t.Fatalf("place %v has no floor", place)
	}
	if floor["id"] != "I-1302" {
		t.Fatalf("floor id = %v, want I-1302", floor["id"])
	}
}

func TestGetByCodeEndpointNotFound(t *testing.T) {
	app := newTestApp()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/service-points/NOPE", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var envelope struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
	if envelope.Error != "service point not found" {
		t.Fatalf("error = %q, want service point not found", envelope.Error)
	}
}
