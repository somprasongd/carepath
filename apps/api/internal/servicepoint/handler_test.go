package servicepoint

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/auth"
)

// stubAuthService stands in for the composition root's auth service: it
// resolves any Bearer token to a fixed principal. The embedded nil interface
// keeps the stub compiling as auth.Service grows — only ParseAccessToken is
// ever exercised behind these handler tests.
type stubAuthService struct {
	auth.Service
	principal auth.Principal
}

func (s stubAuthService) ParseAccessToken(string) (auth.Principal, error) {
	return s.principal, nil
}

// newTestApp mounts the handler over the real service backed by the same
// fakes the service tests use. The staff guard is the real RequireRole over
// the stub auth service, so the /staff/my/service-points tests run the same
// principal resolution as production.
func newTestApp(myPointsPrincipal *auth.Principal) *fiber.App {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{places: seedPlaces()})
	app := fiber.New()
	guard := auth.RequireRole(stubAuthService{principal: deref(myPointsPrincipal)}, auth.RoleStaff, auth.RoleAdmin)
	NewHandler(svc).Register(app.Group("/api/v1"), guard)
	return app
}

func deref(p *auth.Principal) auth.Principal {
	if p == nil {
		return auth.Principal{}
	}
	return *p
}

func TestListEndpointReturnsPointsWithPlaceAndFloor(t *testing.T) {
	app := newTestApp(nil)

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
	app := newTestApp(nil)

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
	app := newTestApp(nil)

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

// decodePoints reads a 200 body into the id list — the assignment behavior
// under test is which points appear, not their shape (covered above).
func decodePoints(t *testing.T, resp *http.Response) []string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var points []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &points); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
	ids := make([]string, len(points))
	for i, p := range points {
		ids[i] = p.ID
	}
	return ids
}

func TestMyServicePointsStaffSeesOnlyAssigned(t *testing.T) {
	staff := auth.Principal{UserID: "user-staff", Username: "staff", Roles: []string{auth.RoleStaff}}
	app := newTestApp(&staff)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/staff/my/service-points", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ids := decodePoints(t, resp); len(ids) != 1 || ids[0] != "SP-LAB" {
		t.Fatalf("ids = %v, want [SP-LAB] — staff sees only the assigned point", ids)
	}
}

func TestMyServicePointsAdminSeesAll(t *testing.T) {
	admin := auth.Principal{UserID: "user-admin", Username: "admin", Roles: []string{auth.RoleAdmin}}
	app := newTestApp(&admin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/staff/my/service-points", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	// fakeRepo.List seeds SP-LAB and SP-REG; the assignment table has only
	// SP-LAB, so seeing both proves the admin bypass took the List path.
	if ids := decodePoints(t, resp); len(ids) != 2 {
		t.Fatalf("ids = %v, want both seeded points for ADMIN", ids)
	}
}

func TestMyServicePointsRejectsMissingToken(t *testing.T) {
	app := newTestApp(nil)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/staff/my/service-points", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
