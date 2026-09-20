package visitlink_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/session"
	"carepath/apps/api/internal/visitlink"
)

// The HTTP surface of #136: the mint route answers to the HIS key alone
// (401 otherwise, in both url and qr formats), and redeem answers to the
// patient session alone — the token never rides a URL.

type fakeSessionService struct {
	tokens map[string]identity.Identity
}

func (f *fakeSessionService) Create(context.Context, string, string) (session.Session, error) {
	return session.Session{}, apperr.New(apperr.KindInvalid, "not used in these tests")
}

func (f *fakeSessionService) Get(_ context.Context, token string) (identity.Identity, error) {
	id, ok := f.tokens[token]
	if !ok {
		return identity.Identity{}, session.ErrNotFound
	}
	return id, nil
}

func newHandlerApp(repo *fakeRepo) *fiber.App {
	svc := visitlink.NewService(repo, &fakeClaims{}, []byte("secret-1"), "https://x.test",
		30*time.Minute, slog.Default())
	sessions := &fakeSessionService{tokens: map[string]identity.Identity{
		"token-patient": {Source: "line", ExternalID: "U-1"},
	}}
	app := fiber.New()
	visitlink.NewHandler(svc).Register(app.Group("/api/v1"), "his-key",
		session.RequireSession(sessions))
	return app
}

func req(t *testing.T, app *fiber.App, method, path, hisKey, bearer, body string) (*http.Response, string) {
	t.Helper()
	var rd *strings.Reader
	if body == "" {
		rd = strings.NewReader("")
	} else {
		rd = strings.NewReader(body)
	}
	r := httptest.NewRequest(method, path, rd)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if hisKey != "" {
		r.Header.Set("X-HIS-API-Key", hisKey)
	}
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

func TestMintRequiresTheHISKey(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-1"] = his.VisitActive
	app := newHandlerApp(repo)

	for _, key := range []string{"", "wrong-key"} {
		resp, body := req(t, app, http.MethodPost, "/api/v1/his/visits/VISIT-1/patient-link", key, "", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("mint with key %q: status %d: %s, want 401", key, resp.StatusCode, body)
		}
	}
}

func TestMintFormats(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-1"] = his.VisitActive
	app := newHandlerApp(repo)

	resp, body := req(t, app, http.MethodPost, "/api/v1/his/visits/VISIT-1/patient-link", "his-key", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mint default: status %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"visitUrl":"https://x.test/patient/journey#vt=`) {
		t.Fatalf("mint default body %s, want a fragment-form visitUrl", body)
	}

	resp, body = req(t, app, http.MethodPost,
		"/api/v1/his/visits/VISIT-1/patient-link?format=qr", "his-key", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mint qr: status %d: %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("mint qr content-type %s, want image/png", ct)
	}
	if !strings.HasPrefix(body, "\x89PNG") {
		t.Fatal("mint qr body is not a PNG")
	}

	resp, body = req(t, app, http.MethodPost,
		"/api/v1/his/visits/VISIT-1/patient-link?format=svg", "his-key", "", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("mint bad format: status %d: %s, want 400", resp.StatusCode, body)
	}
}

func TestRedeemOverHTTP(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-1"] = his.VisitActive
	app := newHandlerApp(repo)

	_, body := req(t, app, http.MethodPost, "/api/v1/his/visits/VISIT-1/patient-link", "his-key", "", "")
	var link struct {
		VisitURL string `json:"visitUrl"`
	}
	if err := json.Unmarshal([]byte(body), &link); err != nil {
		t.Fatalf("decode mint: %v", err)
	}
	raw, _ := strings.CutPrefix(link.VisitURL, "https://x.test/patient/journey#vt=")

	resp, body := req(t, app, http.MethodPost, "/api/v1/journeys/claim", "", "", `{"token":"`+raw+`"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("redeem without session: status %d: %s, want 401", resp.StatusCode, body)
	}

	resp, body = req(t, app, http.MethodPost, "/api/v1/journeys/claim", "", "token-patient", `{"token":"`+raw+`"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"visitId":"VISIT-1"`) {
		t.Fatalf("redeem: status %d: %s, want 200 with VISIT-1", resp.StatusCode, body)
	}

	// The token is accepted in the body only — a GET with it as a query
	// parameter is nothing to this route.
	resp, _ = req(t, app, http.MethodPost, "/api/v1/journeys/claim?token="+raw, "", "token-patient", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("redeem without body: status %d, want 400", resp.StatusCode)
	}

	// A garbled token answers the journey 404, not a distinct error.
	resp, body = req(t, app, http.MethodPost, "/api/v1/journeys/claim", "", "token-patient", `{"token":"garbage"}`)
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "journey not found") {
		t.Fatalf("redeem garbage: status %d: %s, want the journey 404", resp.StatusCode, body)
	}
}
