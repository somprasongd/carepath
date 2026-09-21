// Visit link end to end (#136): the HIS mints a slip link over the inbound
// key-authenticated surface, the patient web redeems it behind a session,
// and the redeemed claim is what the #96 journey guard accepts. The
// lifetime policy (completed grace, cancel) is driven through the
// projection's completed_at, exactly as the real event path would set it.
// The QR-only suite drives the other half of the front door: a patient with
// no LINE and no prior session bootstraps straight from the token.
package e2e_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/session"
	sessionpostgres "carepath/apps/api/internal/session/postgres"
	"carepath/apps/api/internal/visitlink"
	visitlinkpostgres "carepath/apps/api/internal/visitlink/postgres"
)

const linkVisit = "VISIT-E2E-LINK-1"
const linkVisit2 = "VISIT-E2E-LINK-2"

const (
	e2eHISKey     = "e2e-his-key"
	e2eLinkSecret = "e2e-link-secret"
)

// fakeVerifier trusts an id token as a distinct LINE user, so the suite can
// hold sessions for genuinely different identities — the demo source mints
// one shared identity, which would make the stranger assertions vacuous.
type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, idToken string) (identity.Identity, error) {
	return identity.Identity{Source: "line", ExternalID: idToken}, nil
}

func newVisitLinkApp(t *testing.T, database *db.DB) (*fiber.App, session.Service) {
	t.Helper()
	// The full share-suite stack: a real journey service (service points,
	// navigation, locations all wired) behind the real #96 patient guard.
	// The session handler still mounts the demo claim route — this app
	// models a demo deployment, where main.go retires that route only when
	// the mint is live; the redeem route (/journeys/claim) is always on.
	// The visitlink service rides in as the visit-token resolver exactly as
	// main.go wires it, so the QR-only bootstrap runs through the same HTTP
	// route the browser hits.
	claims := sessionpostgres.NewClaimRepo(database)
	links := visitlink.NewService(visitlinkpostgres.New(database), claims,
		[]byte(e2eLinkSecret), "https://carepath.test", 30*time.Minute, slog.Default())
	app := newShareAppWithHIS(t, database, "http://127.0.0.1:1", links)
	sessions := session.NewService(sessionpostgres.New(database), fakeVerifier{},
		links, claims, true /* allowDemo */, time.Hour)
	visitlink.NewHandler(links).Register(app.Group("/api/v1"), e2eHISKey,
		session.RequireSession(sessions))
	return app, sessions
}

// lineSession mints a bearer for a distinct synthetic LINE user straight
// through the session service — session minting itself is the session
// module's own test surface; here only the redeem and guard behavior is
// under test.
func lineSession(t *testing.T, sessions session.Service, userID string) string {
	t.Helper()
	sess, err := sessions.Create(context.Background(), "line", userID)
	if err != nil {
		t.Fatalf("line session %s: %v", userID, err)
	}
	return sess.Token
}

func seedLinkVisit(t *testing.T, database *db.DB) {
	seedVisitFor(t, database, linkVisit)
}

func seedVisitFor(t *testing.T, database *db.DB, visitID string) {
	t.Helper()
	ctx := context.Background()
	purge := func() {
		_, _ = database.Querier(ctx).Exec(ctx,
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, visitID)
	}
	purge()
	t.Cleanup(purge)
	q := database.Querier(ctx)
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, patient_name, status)
		 VALUES ($1, 'PATIENT-E2E-LINK', 'สมชาย ทดสอบลิงก์', 'ACTIVE')`, visitID,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_step
		     (visit_id, step_key, sequence, kind, clinic_code, round, order_refs,
		      status, service_point_id)
		 VALUES ($1, 'ORDERTYPE:LAB:1', 2, 'LAB', NULL, NULL, '{}', 'READY', 'SP-LAB')`,
		visitID); err != nil {
		t.Fatalf("seed step: %v", err)
	}
}

func mint(t *testing.T, app *fiber.App, visitID, key, query string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/his/visits/"+visitID+"/patient-link"+query, nil)
	if key != "" {
		req.Header.Set("X-HIS-API-Key", key)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

func linkToken(t *testing.T, body string) string {
	t.Helper()
	var link struct {
		VisitURL string `json:"visitUrl"`
	}
	if err := json.Unmarshal([]byte(body), &link); err != nil {
		t.Fatalf("decode mint response %s: %v", body, err)
	}
	raw, ok := strings.CutPrefix(link.VisitURL, "https://carepath.test/patient/journey#vt=")
	if !ok || raw == "" {
		t.Fatalf("visitUrl %q is not the expected fragment form", link.VisitURL)
	}
	return raw
}

func redeem(t *testing.T, app *fiber.App, token, bearer string) (*http.Response, string) {
	t.Helper()
	return doJSON(t, app, http.MethodPost, "/api/v1/journeys/claim", bearer,
		`{"token":"`+token+`"}`)
}

func setVisitState(t *testing.T, database *db.DB, status string, completedAgo *time.Duration) {
	t.Helper()
	ctx := context.Background()
	var completedAt any
	if completedAgo != nil {
		completedAt = time.Now().Add(-*completedAgo)
	}
	if _, err := database.Querier(ctx).Exec(ctx,
		`UPDATE carepath.journey_visit SET status = $1, completed_at = $2 WHERE visit_id = $3`,
		status, completedAt, linkVisit,
	); err != nil {
		t.Fatalf("set visit state: %v", err)
	}
}

func TestVisitLinkLifecycle(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (needs migrations applied — see CI)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	app, sessions := newVisitLinkApp(t, database)
	seedLinkVisit(t, database)

	// The mint surface answers to the HIS key alone, and an unprojected
	// visit gets the journey 404.
	resp, body := mint(t, app, linkVisit, "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("mint without key: status %d: %s, want 401", resp.StatusCode, body)
	}
	resp, body = mint(t, app, "VISIT-E2E-NOPE", e2eHISKey, "")
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "journey not found") {
		t.Fatalf("mint unknown visit: status %d: %s, want the journey 404", resp.StatusCode, body)
	}

	// Idempotent mint; rotation changes the URL.
	resp, body = mint(t, app, linkVisit, e2eHISKey, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mint: status %d: %s", resp.StatusCode, body)
	}
	token := linkToken(t, body)
	_, body2 := mint(t, app, linkVisit, e2eHISKey, "")
	if linkToken(t, body2) != token {
		t.Fatal("re-mint returned a different token — a re-print would kill the printed slip")
	}
	_, bodyRotated := mint(t, app, linkVisit, e2eHISKey, "?rotate=true")
	rotated := linkToken(t, bodyRotated)
	if rotated == token {
		t.Fatal("rotate returned the same token")
	}

	// Redeem behind a session; the minted claim is what the journey guard
	// accepts (#96 unchanged).
	resp, body = redeem(t, app, rotated, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("redeem without session: status %d: %s, want 401", resp.StatusCode, body)
	}
	patient := patientSession(t, app)
	resp, body = redeem(t, app, rotated, patient)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"visitId":"`+linkVisit+`"`) {
		t.Fatalf("redeem: status %d: %s, want 200 with the visit id", resp.StatusCode, body)
	}
	resp, body = get(t, app, "/api/v1/journeys/"+linkVisit, patient)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("journey read after redeem: status %d: %s, want 200", resp.StatusCode, body)
	}

	// A session that never redeemed sees nothing — editing ?visit= to
	// someone else's visit is the attack this whole link exists to kill.
	stranger := lineSession(t, sessions, "U-STRANGER")
	resp, body = get(t, app, "/api/v1/journeys/"+linkVisit, stranger)
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "journey not found") {
		t.Fatalf("stranger journey read: status %d: %s, want the journey 404", resp.StatusCode, body)
	}

	// Rotation killed the first token even for a fresh session.
	resp, _ = redeem(t, app, token, stranger)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("redeem rotated-out token: status %d, want 404", resp.StatusCode)
	}

	// The link's lifetime follows the visit: cancelled dies at once.
	setVisitState(t, database, "CANCELLED", nil)
	resp, _ = redeem(t, app, rotated, stranger)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("redeem after cancel: status %d, want 404", resp.StatusCode)
	}

	// Completed lives out its 30-minute grace: a fresh session holding the
	// slip can still get in and read the summary…
	setVisitState(t, database, "COMPLETED", &[]time.Duration{10 * time.Minute}[0])
	resp, body = redeem(t, app, rotated, stranger)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("redeem in grace: status %d: %s, want 200", resp.StatusCode, body)
	}
	resp, body = get(t, app, "/api/v1/journeys/"+linkVisit, stranger)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stranger read in grace: status %d: %s, want 200", resp.StatusCode, body)
	}
	// …and stops after it. The claim the first patient minted survives: the
	// token gates the front door, not the sessions already inside.
	setVisitState(t, database, "COMPLETED", &[]time.Duration{45 * time.Minute}[0])
	latecomer := lineSession(t, sessions, "U-LATE")
	resp, _ = redeem(t, app, rotated, latecomer)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("redeem past grace: status %d, want 404", resp.StatusCode)
	}
	resp, _ = get(t, app, "/api/v1/journeys/"+linkVisit, patient)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("earlier claim after grace: status %d, want 200 — claims outlive the token", resp.StatusCode)
	}
}

// The QR-only front door (#136 layer 2): a hospital without a LINE OA hands
// the patient a printed slip and nothing else. The scanner arrives with no
// bearer at all, and the slip token alone must bootstrap a claimed,
// visit-scoped session over the ordinary POST /auth/session route.
func TestVisitLinkQRBootstrap(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (needs migrations applied — see CI)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	app, _ := newVisitLinkApp(t, database)
	seedVisitFor(t, database, linkVisit)
	seedVisitFor(t, database, linkVisit2)

	// The HIS mints both slips at print time.
	_, body := mint(t, app, linkVisit, e2eHISKey, "")
	token := linkToken(t, body)
	_, body2 := mint(t, app, linkVisit2, e2eHISKey, "")
	token2 := linkToken(t, body2)

	// The anonymous exchange: no Authorization header, no LINE, no demo.
	resp, body := doJSON(t, app, http.MethodPost, "/api/v1/auth/session", "",
		`{"source":"visit-token","idToken":"`+token+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("visit-token session: status %d: %s, want 200", resp.StatusCode, body)
	}
	var created struct {
		SessionToken string `json:"sessionToken"`
		VisitID      string `json:"visitId"`
		Identity     struct {
			Source     string `json:"source"`
			ExternalID string `json:"externalId"`
		} `json:"identity"`
	}
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("decode session response %s: %v", body, err)
	}
	if created.SessionToken == "" || created.VisitID != linkVisit {
		t.Fatalf("session response %s: want a token for %s", body, linkVisit)
	}
	if created.Identity.Source != "visit" || created.Identity.ExternalID != "visit:"+linkVisit {
		t.Errorf("identity = %+v, want the visit-scoped identity", created.Identity)
	}

	// The bootstrapped bearer reads its own journey…
	resp, body = get(t, app, "/api/v1/journeys/"+linkVisit, created.SessionToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("journey read with bootstrapped session: status %d: %s, want 200", resp.StatusCode, body)
	}
	// …and nothing else: swapping ?visit= for a neighbor's id must stay the
	// journey 404 even though the mint is live — the hole this door closes.
	resp, _ = get(t, app, "/api/v1/journeys/"+linkVisit2, created.SessionToken)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("neighbor journey read: status %d, want the journey 404", resp.StatusCode)
	}

	// Garbage answers the same 404 as redeem — shape-checked, so exactly 22
	// base64url characters that decode to nothing minted.
	resp, _ = doJSON(t, app, http.MethodPost, "/api/v1/auth/session", "",
		`{"source":"visit-token","idToken":"AAAAAAAAAAAAAAAAAAAAAA"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("garbage token: status %d, want 404", resp.StatusCode)
	}

	// Rotation kills the bootstrap path too: a re-printed slip must stop the
	// old token from minting fresh sessions.
	_, _ = mint(t, app, linkVisit2, e2eHISKey, "?rotate=true")
	resp, _ = doJSON(t, app, http.MethodPost, "/api/v1/auth/session", "",
		`{"source":"visit-token","idToken":"`+token2+`"}`)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("rotated-out token bootstrap: status %d, want 404", resp.StatusCode)
	}
}
