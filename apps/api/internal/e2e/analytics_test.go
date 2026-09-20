// Executive analytics guard (#86): the AC is about who reaches the new
// surface — EXECUTIVE logins see the overview and nothing else. Everything
// here is real but the process boundary: the argon2id-seeded users from
// migration 000015, the actual JWT issuer and RequireRole middleware, and
// the analytics handler reading the timeline. The journey handler is
// mounted with its real guard too (its HIS client is never reached — a
// 403 never gets that far).
package e2e_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/analytics"
	analyticspostgres "carepath/apps/api/internal/analytics/postgres"
	"carepath/apps/api/internal/auth"
	authpostgres "carepath/apps/api/internal/auth/postgres"
	"carepath/apps/api/internal/his/httpclient"
	"carepath/apps/api/internal/journey"
	journeypostgres "carepath/apps/api/internal/journey/postgres"
	"carepath/apps/api/internal/platform/db"
)

type tokensBody struct {
	AccessToken string `json:"accessToken"`
}

type identityBody struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

func login(t *testing.T, app *fiber.App, username, password string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(
		`{"username":"`+username+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("login %s: %v", username, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login %s: status %d: %s", username, resp.StatusCode, body)
	}
	var body tokensBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("login %s: decode: %v", username, err)
	}
	return body.AccessToken
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
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

// newAnalyticsApp mounts the routes exactly as the composition root does.
func newAnalyticsApp(t *testing.T, database *db.DB) *fiber.App {
	t.Helper()
	authService := auth.NewService(authpostgres.New(database), database,
		auth.NewTokenIssuer([]byte("e2e-test-secret"), time.Minute), time.Hour)

	// The journey service needs an HIS client by construction; the guard
	// rejects every request this test sends it before the client is used.
	journeys := journey.NewService(httpclient.New("http://127.0.0.1:1", nil), nil,
		journeypostgres.New(database), database)

	analyticsService, err := analytics.NewService(analyticspostgres.New(database), database,
		"Asia/Bangkok")
	if err != nil {
		t.Fatalf("analytics service: %v", err)
	}

	app := fiber.New()
	authHandler := auth.NewHandler(authService)
	authHandler.Register(app.Group("/api/v1"))
	authHandler.RegisterMe(app.Group("/api/v1"), auth.RequireRole(authService))
	// The patient guard is a no-op here: this suite exercises the staff
	// surface only. The visit-ownership policy itself (#96) is pinned in
	// internal/session/guard_test.go.
	journey.NewHandler(journeys).Register(app.Group("/api/v1"),
		auth.RequireRole(authService, auth.RoleStaff, auth.RoleAdmin),
		func(c fiber.Ctx) error { return c.Next() })
	analytics.NewHandler(analyticsService).Register(app.Group("/api/v1"),
		auth.RequireRole(authService, auth.RoleStaff, auth.RoleAdmin, auth.RoleExecutive))
	return app
}

func TestAnalyticsOverviewRoleGuard(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (needs migrations applied — see CI)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	app := newAnalyticsApp(t, database)

	// No token → 401, not 403: the request is unauthenticated.
	resp, _ := get(t, app, "/api/v1/analytics/overview", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: status %d, want 401", resp.StatusCode)
	}

	// staff/demo (STAFF) reaches the overview.
	staff := login(t, app, "staff", "demo")
	resp, body := get(t, app, "/api/v1/analytics/overview?window=today", staff)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff: status %d: %s, want 200", resp.StatusCode, body)
	}
	var overview struct {
		Window string `json:"window"`
	}
	if err := json.Unmarshal([]byte(body), &overview); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if overview.Window != "today" {
		t.Fatalf("window = %q, want today", overview.Window)
	}

	// exec/demo (EXECUTIVE): same overview, and nothing else.
	exec := login(t, app, "exec", "demo")
	resp, _ = get(t, app, "/api/v1/analytics/overview", exec) // window omitted → today
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("exec analytics: status %d, want 200", resp.StatusCode)
	}
	resp, body = get(t, app, "/api/v1/auth/me", exec)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("exec auth/me: status %d", resp.StatusCode)
	}
	var identity identityBody
	if err := json.Unmarshal([]byte(body), &identity); err != nil {
		t.Fatalf("decode identity: %v", err)
	}
	if len(identity.Roles) != 1 || identity.Roles[0] != auth.RoleExecutive {
		t.Fatalf("exec roles = %v, want exactly [EXECUTIVE]", identity.Roles)
	}
	resp, body = get(t, app, "/api/v1/staff/visits", exec)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("exec staff/visits: status %d: %s, want 403", resp.StatusCode, body)
	}

	// The window is validated, not guessed.
	resp, _ = get(t, app, "/api/v1/analytics/overview?window=week", staff)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("window=week: status %d, want 400", resp.StatusCode)
	}
}
