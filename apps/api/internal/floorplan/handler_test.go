package floorplan

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/hospitalmap"
)

// fakeRepo stands in for the postgres adapter. Plans are keyed the way the
// real table is — by floor and digest together — so a test that addresses
// one floor's drawing through another floor's URL misses, exactly as it
// would against Postgres.
type fakeRepo struct {
	svgs    map[string]string // "<floorID>/<sha>" -> svg
	digests map[string]string // planID -> sha
}

func (f *fakeRepo) SVG(_ context.Context, floorID, sha256 string) (string, error) {
	svg, ok := f.svgs[floorID+"/"+sha256]
	if !ok {
		return "", ErrPlanNotFound
	}
	return svg, nil
}

func (f *fakeRepo) Digests(_ context.Context, planIDs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range planIDs {
		if sha, ok := f.digests[id]; ok {
			out[id] = sha
		}
	}
	return out, nil
}

// The write path is covered by the service tests; these handler cases only
// need the read side to answer.
func (f *fakeRepo) Insert(context.Context, Stored, string, string) error { return nil }

func (f *fakeRepo) Get(context.Context, string, string) (Stored, error) {
	return Stored{}, ErrPlanNotFound
}

func (f *fakeRepo) ListByFloor(context.Context, string) ([]Stored, error) { return nil, nil }

func (f *fakeRepo) Raw(context.Context, string, string) (string, error) {
	return "", ErrPlanNotFound
}

// fakeFloors is the hospitalmap side: two floors with a plan and one still
// waiting for its first upload.
type fakeFloors struct {
	hospitalmap.Service
	floors []hospitalmap.Floor
}

func (f *fakeFloors) ListFloors(context.Context) ([]hospitalmap.Floor, error) {
	return f.floors, nil
}

func (f *fakeFloors) GetFloor(_ context.Context, floorID string) (hospitalmap.Floor, error) {
	for _, floor := range f.floors {
		if floor.ID == floorID {
			return floor, nil
		}
	}
	return hospitalmap.Floor{}, hospitalmap.ErrFloorNotFound
}

func ptr(s string) *string { return &s }

const groundSHA = "b6c8f46935f1fef6beb61a098d615dcbbf9c3ef5514cb45e8683a6525b979324"

func newTestApp() *fiber.App {
	repo := &fakeRepo{
		svgs:    map[string]string{"I-1301/" + groundSHA: `<svg xmlns="http://www.w3.org/2000/svg"/>`},
		digests: map[string]string{"FP-I1301": groundSHA},
	}
	floors := &fakeFloors{floors: []hospitalmap.Floor{
		{ID: "I-1301", BuildingID: "BLD-I13", Code: "1", Name: "Ground Floor", LevelOrder: 1,
			ViewBox: ptr("0 0 1600 900"), ActivePlanID: ptr("FP-I1301")},
		{ID: "I-1399", BuildingID: "BLD-I13", Code: "9", Name: "Undrawn Floor", LevelOrder: 9},
	}}

	app := fiber.New()
	// The admin guard is a plain 403 here: these cases are about the public
	// read routes, and the guard itself is auth's to test.
	denyAdmin := func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) }
	NewHandler(NewService(repo, floors, nil, nil), floors).Register(app.Group("/api/v1"), denyAdmin)
	return app
}

func get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := newTestApp().Test(httptest.NewRequest(http.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	return resp
}

func TestListFloorsCarriesThePlanURL(t *testing.T) {
	resp := get(t, "/api/v1/floors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var views []FloorView
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &views); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if len(views) != 2 {
		t.Fatalf("got %d floors, want 2", len(views))
	}

	// The URL the web app fetches is assembled here, not by the client:
	// the digest in it is what makes the response cacheable forever.
	want := "/api/v1/floors/I-1301/plan/" + groundSHA + ".svg"
	if views[0].PlanURL == nil || *views[0].PlanURL != want {
		t.Errorf("PlanURL = %v, want %q", views[0].PlanURL, want)
	}
	if views[0].ViewBox == nil || *views[0].ViewBox != "0 0 1600 900" {
		t.Errorf("ViewBox = %v, want the floor's coordinate space", views[0].ViewBox)
	}
	// A client that already holds a plan id (e.g. from the admin history
	// listing) can tell it is the active one directly from this field,
	// rather than checking whether its digest appears inside PlanURL.
	if views[0].ActivePlanID == nil || *views[0].ActivePlanID != "FP-I1301" {
		t.Errorf("ActivePlanID = %v, want FP-I1301", views[0].ActivePlanID)
	}

	// A floor with no plan yet says so by omission rather than by handing
	// out a URL that would 404.
	if views[1].PlanURL != nil {
		t.Errorf("PlanURL = %v on a floor with no plan, want absent", *views[1].PlanURL)
	}
	if views[1].ActivePlanID != nil {
		t.Errorf("ActivePlanID = %v on a floor with no plan, want absent", *views[1].ActivePlanID)
	}

	// The pointer must be revalidated — it is the only thing between a new
	// plan and the patients who need it.
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
}

func TestPlanIsServedImmutableAndHardened(t *testing.T) {
	resp := get(t, "/api/v1/floors/I-1301/plan/"+groundSHA+".svg")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	for header, want := range map[string]string{
		"Content-Type":  "image/svg+xml; charset=utf-8",
		"Cache-Control": "public, max-age=31536000, immutable",
		// This URL is directly navigable, and a directly-navigated SVG would
		// otherwise run in the API's own origin.
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'",
	} {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestPlanLookupIsScopedToItsFloor(t *testing.T) {
	cases := map[string]string{
		"unknown digest":            "/api/v1/floors/I-1301/plan/" + groundSHA[:63] + "0.svg",
		"right digest, wrong floor": "/api/v1/floors/I-1302/plan/" + groundSHA + ".svg",
		"missing .svg suffix":       "/api/v1/floors/I-1301/plan/" + groundSHA,
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			if resp := get(t, path); resp.StatusCode != http.StatusNotFound {
				t.Errorf("status = %d, want 404", resp.StatusCode)
			}
		})
	}
}

// Admin routes need a guard that lets the call through, unlike newTestApp's
// denyAdmin — used only for the admin-route cases below.
func newAdminTestApp() *fiber.App {
	repo := &fakeRepo{
		svgs:    map[string]string{"I-1301/" + groundSHA: `<svg xmlns="http://www.w3.org/2000/svg"/>`},
		digests: map[string]string{"FP-I1301": groundSHA},
	}
	floors := &fakeFloors{floors: []hospitalmap.Floor{
		{ID: "I-1301", BuildingID: "BLD-I13", Code: "1", Name: "Ground Floor", LevelOrder: 1,
			ViewBox: ptr("0 0 1600 900"), ActivePlanID: ptr("FP-I1301")},
		{ID: "I-1399", BuildingID: "BLD-I13", Code: "9", Name: "Undrawn Floor", LevelOrder: 9},
	}}

	app := fiber.New()
	allowAdmin := func(c fiber.Ctx) error { return c.Next() }
	NewHandler(NewService(repo, floors, nil, nil), floors).Register(app.Group("/api/v1"), allowAdmin)
	return app
}

// Health with no active plan never touches the navigation graph, so it is
// safe to exercise here despite this app wiring a nil graph service.
func TestHealthRouteOnFloorWithNoActivePlan(t *testing.T) {
	resp, err := newAdminTestApp().Test(httptest.NewRequest(http.MethodGet, "/api/v1/admin/floors/I-1399/health", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var warnings []Warning
	if err := json.Unmarshal(body, &warnings); err != nil {
		t.Fatalf("decode: %v (%s)", err, body)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %+v, want none for a floor with no active plan", warnings)
	}
}

func TestHealthRouteOnUnknownFloorIs404(t *testing.T) {
	resp, err := newAdminTestApp().Test(httptest.NewRequest(http.MethodGet, "/api/v1/admin/floors/I-9999/health", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// A typo'd or stale floor id must 404, not answer an empty list — the two
// are indistinguishable from a floor that genuinely has no plans yet.
func TestListPlansOnUnknownFloorIs404(t *testing.T) {
	resp, err := newAdminTestApp().Test(httptest.NewRequest(http.MethodGet, "/api/v1/admin/floors/I-9999/plans", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestListPlansOnKnownFloorSucceeds(t *testing.T) {
	resp, err := newAdminTestApp().Test(httptest.NewRequest(http.MethodGet, "/api/v1/admin/floors/I-1301/plans", nil))
	if err != nil {
		t.Fatalf("test request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
