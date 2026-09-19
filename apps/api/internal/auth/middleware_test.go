package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

// The auth HTTP boundary: status codes, the uniform 401, and what RequireRole
// lets through. The service runs over the in-memory repo from service_test.go
// so these tests exercise the real handler and middleware, not stubs of them.
func newHTTPTestApp(t *testing.T) (*fiber.App, Service) {
	t.Helper()
	svc := newTestService(newFakeRepo())
	app := fiber.New()
	api := app.Group("/api/v1")
	NewHandler(svc).Register(api)
	NewHandler(svc).RegisterMe(app.Group("/api/v1"), RequireRole(svc))
	admin := app.Group("/api/v1/admin", RequireRole(svc, RoleAdmin))
	admin.Get("/probe", func(c fiber.Ctx) error { return c.SendString("admin-ok") })
	api.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })
	return app, svc
}

func doJSON(t *testing.T, app *fiber.App, method, path, token, body string) (*http.Response, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

func decodeTokens(t *testing.T, body string) TokenPair {
	t.Helper()
	var pair TokenPair
	if err := json.Unmarshal([]byte(body), &pair); err != nil {
		t.Fatalf("decode tokens response %q: %v", body, err)
	}
	return pair
}

func TestLoginEndpoint(t *testing.T) {
	app, svc := newHTTPTestApp(t)

	resp, body := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "",
		`{"username":"somchai.r","password":"demo"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d (%s), want 200", resp.StatusCode, body)
	}
	pair := decodeTokens(t, body)
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("tokens missing from %s", body)
	}
	if pair.Identity.Username != "somchai.r" || len(pair.Identity.Roles) != 1 {
		t.Fatalf("identity = %+v", pair.Identity)
	}
	if _, err := svc.ParseAccessToken(pair.AccessToken); err != nil {
		t.Fatalf("returned access token does not parse: %v", err)
	}

	// Wrong password and unknown user must be the same 401, byte for byte.
	respBad, bodyBad := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "",
		`{"username":"somchai.r","password":"nope"}`)
	respUnknown, bodyUnknown := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "",
		`{"username":"nobody","password":"demo"}`)
	if respBad.StatusCode != http.StatusUnauthorized || respUnknown.StatusCode != http.StatusUnauthorized {
		t.Fatalf("statuses = %d/%d, want 401/401", respBad.StatusCode, respUnknown.StatusCode)
	}
	if bodyBad != bodyUnknown {
		t.Fatalf("login failure bodies differ: %q vs %q", bodyBad, bodyUnknown)
	}

	if resp, body := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", `not json`); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed body status = %d (%s), want 400", resp.StatusCode, body)
	}
}

func TestRefreshAndLogoutEndpoints(t *testing.T) {
	app, _ := newHTTPTestApp(t)
	_, body := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "",
		`{"username":"somchai.r","password":"demo"}`)
	first := decodeTokens(t, body)

	resp, body := doJSON(t, app, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refreshToken":"`+first.RefreshToken+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d (%s), want 200", resp.StatusCode, body)
	}
	second := decodeTokens(t, body)
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	resp, body = doJSON(t, app, http.MethodPost, "/api/v1/auth/logout", "",
		`{"refreshToken":"`+second.RefreshToken+`"}`)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d (%s), want 204", resp.StatusCode, body)
	}
	if resp, body := doJSON(t, app, http.MethodPost, "/api/v1/auth/refresh", "",
		`{"refreshToken":"`+second.RefreshToken+`"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d (%s), want 401", resp.StatusCode, body)
	}
}

func TestMeEndpoint(t *testing.T) {
	app, _ := newHTTPTestApp(t)
	_, body := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "",
		`{"username":"admin","password":"demo"}`)
	admin := decodeTokens(t, body)

	resp, body := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", admin.AccessToken, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"username":"admin"`) {
		t.Fatalf("me = %d (%s), want 200 with the admin identity", resp.StatusCode, body)
	}
	if resp, _ := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", "", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me without token = %d, want 401", resp.StatusCode)
	}
}

func TestRequireRole(t *testing.T) {
	app, _ := newHTTPTestApp(t)
	_, staffBody := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "",
		`{"username":"somchai.r","password":"demo"}`)
	staff := decodeTokens(t, staffBody)

	// Authentication failures: absent, non-Bearer, and expired are all 401.
	expired := NewTokenIssuer([]byte("a-test-secret-32-bytes-long!!"), -time.Minute)
	expiredToken, _, _ := expired.IssueAccessToken(testPrincipal)
	for name, tc := range map[string]struct {
		token string
	}{
		"no header":  {},
		"not bearer": {token: "Basic abc"},
		"garbage":    {token: "garbage"},
		"expired":    {token: expiredToken},
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		if tc.token != "" {
			req.Header.Set("Authorization", tc.token)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want 401", name, resp.StatusCode)
		}
	}

	// STAFF passes the open-auth group and is refused the ADMIN-only group.
	if resp, body := doJSON(t, app, http.MethodGet, "/api/v1/auth/me", staff.AccessToken, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("staff on /auth/me = %d (%s), want 200", resp.StatusCode, body)
	}
	if resp, _ := doJSON(t, app, http.MethodGet, "/api/v1/admin/probe", staff.AccessToken, ""); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff on admin probe = %d, want 403", resp.StatusCode)
	}
	_, adminBody := doJSON(t, app, http.MethodPost, "/api/v1/auth/login", "",
		`{"username":"admin","password":"demo"}`)
	admin := decodeTokens(t, adminBody)
	if resp, body := doJSON(t, app, http.MethodGet, "/api/v1/admin/probe", admin.AccessToken, ""); resp.StatusCode != http.StatusOK || body != "admin-ok" {
		t.Fatalf("admin on admin probe = %d (%s), want 200 admin-ok", resp.StatusCode, body)
	}

	// Patient-style routes outside the guard stay open — no token needed.
	if resp, body := doJSON(t, app, http.MethodGet, "/api/v1/ping", "", ""); resp.StatusCode != http.StatusOK || body != "pong" {
		t.Fatalf("open route = %d (%s), want 200 pong", resp.StatusCode, body)
	}
}
