package session_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/session"
)

// The visit-ownership policy of #96, pinned at the middleware level with
// fakes: the acceptance cases are 401 without/with a bad token, 404 for a
// visit the identity never claimed (with the journey read's exact error
// text), and 200 once claimed — plus the identity-keying rule that a fresh
// token for the same LINE user keeps its claims.

// fakeService mints nothing; it resolves pre-seeded tokens to identities.
type fakeService struct {
	tokens map[string]identity.Identity
}

func (f *fakeService) Create(context.Context, string, string) (session.Session, error) {
	return session.Session{}, apperr.New(apperr.KindInvalid, "not used in these tests")
}

func (f *fakeService) Get(_ context.Context, token string) (identity.Identity, error) {
	id, ok := f.tokens[token]
	if !ok {
		return identity.Identity{}, session.ErrNotFound
	}
	return id, nil
}

// fakeClaims owns a fixed world of visits; Claim fails with the journey-
// shaped 404 for anything outside it.
type fakeClaims struct {
	visits map[string]bool    // visits that exist at all
	claims map[[3]string]bool // (source, externalID, visitID) claimed
}

func (f *fakeClaims) Claim(_ context.Context, id identity.Identity, visitID string) error {
	if !f.visits[visitID] {
		return session.ErrVisitNotFound
	}
	f.claims[[3]string{id.Source, id.ExternalID, visitID}] = true
	return nil
}

func (f *fakeClaims) HasClaimed(_ context.Context, id identity.Identity, visitID string) (bool, error) {
	return f.claims[[3]string{id.Source, id.ExternalID, visitID}], nil
}

func newGuardApp() (*fiber.App, *fakeService, *fakeClaims) {
	svc := &fakeService{
		tokens: map[string]identity.Identity{
			"token-patient-a": {Source: "line", ExternalID: "U-A"},
			"token-patient-b": {Source: "line", ExternalID: "U-B"},
			// Same LINE user as token-patient-a, a later login.
			"token-patient-a-relogin": {Source: "line", ExternalID: "U-A"},
		},
	}
	claims := &fakeClaims{
		visits: map[string]bool{"VISIT-1": true, "VISIT-2": true},
		claims: map[[3]string]bool{},
	}
	app := fiber.New()
	session.NewHandler(svc, claims).Register(app.Group("/api/v1"))
	app.Get("/api/v1/journeys/:visitId", session.RequirePatientVisit(svc, claims),
		func(c fiber.Ctx) error { return c.SendString("journey-body") })
	return app, svc, claims
}

func get(t *testing.T, app *fiber.App, path, token string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func post(t *testing.T, app *fiber.App, path, token string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func TestVisitGuardRejectsUnauthenticated(t *testing.T) {
	app, _, _ := newGuardApp()

	resp, body := get(t, app, "/api/v1/journeys/VISIT-1", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: status %d: %s, want 401", resp.StatusCode, body)
	}

	resp, body = get(t, app, "/api/v1/journeys/VISIT-1", "not-a-session-token")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token: status %d: %s, want 401", resp.StatusCode, body)
	}

	// A staff JWT is just another token the session module does not know.
	resp, body = get(t, app, "/api/v1/journeys/VISIT-1", "some-staff-jwt")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("staff jwt: status %d: %s, want 401", resp.StatusCode, body)
	}
}

func TestVisitGuardHidesUnclaimedVisit(t *testing.T) {
	app, _, _ := newGuardApp()

	// Another patient's session, no claim on VISIT-1.
	resp, body := get(t, app, "/api/v1/journeys/VISIT-1", "token-patient-b")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unclaimed: status %d: %s, want 404", resp.StatusCode, body)
	}
	// The 404 body must match the journey read's unknown-visit text exactly,
	// so ownership never leaks whether the visit exists.
	if !strings.Contains(body, "journey not found") {
		t.Fatalf("unclaimed 404 body = %q, want the journey 404 text", body)
	}

	// A visit nobody projected answers with the same shape.
	resp, body = get(t, app, "/api/v1/journeys/VISIT-404", "token-patient-b")
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "journey not found") {
		t.Fatalf("unknown visit: status %d: %s, want the same 404", resp.StatusCode, body)
	}
}

func TestClaimThenRead(t *testing.T) {
	app, _, _ := newGuardApp()

	resp, body := post(t, app, "/api/v1/journeys/VISIT-1/claim", "token-patient-a")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("claim: status %d: %s, want 204", resp.StatusCode, body)
	}

	resp, body = get(t, app, "/api/v1/journeys/VISIT-1", "token-patient-a")
	if resp.StatusCode != http.StatusOK || body != "journey-body" {
		t.Fatalf("read after claim: status %d: %s, want 200 with body", resp.StatusCode, body)
	}

	// The claim is idempotent, and the other patient still gets the 404.
	resp, body = post(t, app, "/api/v1/journeys/VISIT-1/claim", "token-patient-a")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("re-claim: status %d: %s, want 204", resp.StatusCode, body)
	}
	resp, body = get(t, app, "/api/v1/journeys/VISIT-1", "token-patient-b")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("other patient after claim: status %d: %s, want 404", resp.StatusCode, body)
	}
}

func TestClaimUnknownVisitIsJourneyNotFound(t *testing.T) {
	app, _, _ := newGuardApp()

	resp, body := post(t, app, "/api/v1/journeys/VISIT-404/claim", "token-patient-a")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("claim unknown: status %d: %s, want 404", resp.StatusCode, body)
	}
	if !strings.Contains(body, "journey not found") {
		t.Fatalf("claim unknown body = %q, want the journey 404 text", body)
	}
}

func TestClaimsAreKeyedByIdentityNotToken(t *testing.T) {
	app, _, _ := newGuardApp()

	if _, body := post(t, app, "/api/v1/journeys/VISIT-1/claim", "token-patient-a"); len(body) > 0 {
		t.Fatalf("claim body should be empty, got %q", body)
	}
	// A re-logged-in token for the same identity keeps the claim: ownership
	// belongs to the person, not the 24h session row.
	resp, body := get(t, app, "/api/v1/journeys/VISIT-1", "token-patient-a-relogin")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("relogin read: status %d: %s, want 200", resp.StatusCode, body)
	}
}
