// Visit share link guard (#89): the acceptance criteria are about who may
// mint, read, and revoke a share link — and that the shared answer leaks
// nothing identifying. Everything is real but the process boundary: the
// argon2id users from migration 000015, the actual JWT issuer, the session
// service with the demo bypass on, RequireSession, and the share service
// reading the journey projection. The HIS client is wired but never
// reached: every request here stops at a guard or a read.
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

	"carepath/apps/api/internal/auth"
	authpostgres "carepath/apps/api/internal/auth/postgres"
	"carepath/apps/api/internal/his/httpclient"
	"carepath/apps/api/internal/hospitalmap"
	hospitalmappostgres "carepath/apps/api/internal/hospitalmap/postgres"
	"carepath/apps/api/internal/journey"
	journeypostgres "carepath/apps/api/internal/journey/postgres"
	"carepath/apps/api/internal/location"
	locationpostgres "carepath/apps/api/internal/location/postgres"
	"carepath/apps/api/internal/navigation"
	navigationpostgres "carepath/apps/api/internal/navigation/postgres"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/servicepoint"
	servicepointpostgres "carepath/apps/api/internal/servicepoint/postgres"
	"carepath/apps/api/internal/session"
	sessionpostgres "carepath/apps/api/internal/session/postgres"
	"carepath/apps/api/internal/share"
	sharepostgres "carepath/apps/api/internal/share/postgres"
)

const shareVisit = "VISIT-E2E-SHARE-1"

func newShareApp(t *testing.T, database *db.DB) *fiber.App {
	return newShareAppWithHIS(t, database, "http://127.0.0.1:1")
}

// newShareAppWithHIS builds the same stack with the HIS client pointed at a
// live base URL. The station queue suite (#102) drives the real transition
// path, which replans from the HIS snapshot before writing.
func newShareAppWithHIS(t *testing.T, database *db.DB, hisBaseURL string) *fiber.App {
	t.Helper()
	authService := auth.NewService(authpostgres.New(database), database,
		auth.NewTokenIssuer([]byte("e2e-test-secret"), time.Minute), time.Hour)
	sessions := session.NewService(sessionpostgres.New(database), nil,
		true /* allowDemo */, time.Hour)
	claims := sessionpostgres.NewClaimRepo(database)
	// The journey service is fully wired (real service-point resolution), so
	// GetJourney runs for real — the shared answer must come from the actual
	// projection, not a stub. The HIS client is never reached: this test only
	// reads.
	hospitalMap := hospitalmap.NewService(hospitalmappostgres.New(database))
	servicePoints := servicepoint.NewService(servicepointpostgres.New(database), hospitalMap)
	navigationGraph := navigation.NewService(navigationpostgres.New(database), servicePoints)
	locations, err := location.NewService(locationpostgres.New(database), navigationGraph)
	if err != nil {
		t.Fatalf("location service: %v", err)
	}
	journeys := journey.NewService(httpclient.New(hisBaseURL, nil), servicePoints, navigationGraph, locations,
		journeypostgres.New(database), database, "Asia/Bangkok")
	shares := share.NewService(sharepostgres.New(database), journeys, database, time.Hour)

	app := fiber.New()
	session.NewHandler(sessions, claims).Register(app.Group("/api/v1"))
	authHandler := auth.NewHandler(authService)
	authHandler.Register(app.Group("/api/v1"))
	authHandler.RegisterMe(app.Group("/api/v1"), auth.RequireRole(authService))
	// The patient surfaces take the real visit-ownership guard (#96): the
	// session's identity must have claimed the visit before reading or
	// sharing it.
	patientVisitGuard := session.RequirePatientVisit(sessions, claims)
	staffGuard := auth.RequireRole(authService, auth.RoleStaff, auth.RoleAdmin)
	journey.NewHandler(journeys, servicePoints).Register(app.Group("/api/v1"),
		staffGuard,
		patientVisitGuard)
	share.NewHandler(shares).Register(app.Group("/api/v1"), patientVisitGuard)
	// The station queue suite (#102) shares this app: the picker feed reads
	// the same assignment table through the servicepoint module.
	servicepoint.NewHandler(servicePoints).Register(app.Group("/api/v1"), staffGuard)
	return app
}

// A visit with a READY lab step bound to the seeded Laboratory point, so the
// shared answer has a real step to describe without leaking its bindings.
func seedShareVisit(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	purge := func() {
		_, _ = database.Querier(ctx).Exec(ctx,
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, shareVisit)
	}
	purge()
	t.Cleanup(purge)
	q := database.Querier(ctx)
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, patient_name, status)
		 VALUES ($1, 'PATIENT-E2E-SHARE', 'สมชาย ทดสอบระบบ', 'ACTIVE')`, shareVisit,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_step
		     (visit_id, step_key, sequence, kind, clinic_code, round, order_refs,
		      status, service_point_id)
		 VALUES ($1, 'ORDERTYPE:LAB:1', 2, 'LAB', NULL, NULL, '{}', 'READY', 'SP-LAB')`,
		shareVisit); err != nil {
		t.Fatalf("seed step: %v", err)
	}
}

func patientSession(t *testing.T, app *fiber.App) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/session",
		strings.NewReader(`{"source":"demo"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("demo session: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("demo session: status %d: %s", resp.StatusCode, body)
	}
	var body struct {
		SessionToken string `json:"sessionToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	return body.SessionToken
}

func doJSON(t *testing.T, app *fiber.App, method, path, token, body string) (*http.Response, string) {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
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

func TestShareLinkLifecycle(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (needs migrations applied — see CI)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	app := newShareApp(t, database)
	seedShareVisit(t, database)

	// Creating a link takes a patient session.
	resp, body := doJSON(t, app, http.MethodPost, "/api/v1/journeys/"+shareVisit+"/share", "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("create without session: status %d: %s, want 401", resp.StatusCode, body)
	}
	patient := patientSession(t, app)
	// #96: a session that has not claimed the visit cannot share it — and the
	// 404 must not reveal whether the visit exists.
	resp, body = doJSON(t, app, http.MethodPost, "/api/v1/journeys/"+shareVisit+"/share", patient, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("create before claim: status %d: %s, want 404", resp.StatusCode, body)
	}
	resp, body = doJSON(t, app, http.MethodPost, "/api/v1/journeys/"+shareVisit+"/claim", patient, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("claim: status %d: %s, want 204", resp.StatusCode, body)
	}
	resp, body = doJSON(t, app, http.MethodPost, "/api/v1/journeys/"+shareVisit+"/share", patient, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status %d: %s, want 200", resp.StatusCode, body)
	}
	var link struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	if err := json.Unmarshal([]byte(body), &link); err != nil {
		t.Fatalf("decode link: %v", err)
	}
	if len(link.Token) != 64 {
		t.Fatalf("token length = %d, want 64 hex chars", len(link.Token))
	}

	// Reading the shared journey takes the share token, and answers with the
	// redacted step only.
	resp, body = get(t, app, "/api/v1/shared/journey", link.Token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shared read: status %d: %s, want 200", resp.StatusCode, body)
	}
	for _, banned := range []string{
		"สมชาย", "PATIENT-E2E-SHARE", shareVisit, // who
		"ORDERTYPE:LAB:1", "SP-LAB", // bindings
	} {
		if strings.Contains(body, banned) {
			t.Errorf("shared answer leaks %q: %s", banned, body)
		}
	}
	var shared struct {
		Status      string `json:"status"`
		CurrentStep *struct {
			Title            string `json:"title"`
			ServicePointName string `json:"servicePointName"`
		} `json:"currentStep"`
	}
	if err := json.Unmarshal([]byte(body), &shared); err != nil {
		t.Fatalf("decode shared: %v", err)
	}
	if shared.Status != "WAITING" || shared.CurrentStep == nil ||
		shared.CurrentStep.ServicePointName != "Laboratory" {
		t.Fatalf("shared = %s, want WAITING with the Laboratory step", body)
	}

	// Cross-token rule, every direction: neither the patient session nor a
	// staff JWT reads the shared surface, and the share token reaches
	// neither the staff commands nor the share commands themselves.
	resp, _ = get(t, app, "/api/v1/shared/journey", patient)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("session on shared: status %d, want 401", resp.StatusCode)
	}
	staff := login(t, app, "staff", "demo")
	resp, _ = get(t, app, "/api/v1/shared/journey", staff)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("staff JWT on shared: status %d, want 401", resp.StatusCode)
	}
	resp, _ = get(t, app, "/api/v1/staff/visits", link.Token)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("share token on staff: status %d, want 401", resp.StatusCode)
	}
	resp, _ = doJSON(t, app, http.MethodPost,
		"/api/v1/journeys/"+shareVisit+"/steps/ORDERTYPE:LAB:1/transition", link.Token,
		`{"to":"STARTED"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("share token on transition: status %d, want 401", resp.StatusCode)
	}
	resp, _ = doJSON(t, app, http.MethodPost, "/api/v1/journeys/"+shareVisit+"/share", link.Token, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("share token on create: status %d, want 401", resp.StatusCode)
	}

	// Since #96 the journey read is a patient surface behind the session +
	// visit claim: neither a share token nor an anonymous caller reaches it
	// — the risk ADR-0011 §7 once recorded as accepted is closed, and the
	// cross-token rule now holds in this direction too.
	resp, _ = get(t, app, "/api/v1/journeys/"+shareVisit, link.Token)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("share token on journey read: status %d, want 401", resp.StatusCode)
	}
	resp, _ = get(t, app, "/api/v1/journeys/"+shareVisit, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous journey read: status %d, want 401", resp.StatusCode)
	}

	// Revoking is idempotent and kills the link…
	resp, _ = doJSON(t, app, http.MethodDelete, "/api/v1/journeys/"+shareVisit+"/share", patient, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke: status %d, want 204", resp.StatusCode)
	}
	resp, _ = doJSON(t, app, http.MethodDelete, "/api/v1/journeys/"+shareVisit+"/share", patient, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke again: status %d, want 204 (idempotent)", resp.StatusCode)
	}
	resp, _ = get(t, app, "/api/v1/shared/journey", link.Token)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("shared read after revoke: status %d, want 401", resp.StatusCode)
	}

	// …an unknown visit answers the journey module's 404, and a share token
	// for it is equally dead.
	resp, _ = doJSON(t, app, http.MethodPost, "/api/v1/journeys/VISIT-E2E-NOPE/share", patient, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown visit: status %d, want 404", resp.StatusCode)
	}
}
