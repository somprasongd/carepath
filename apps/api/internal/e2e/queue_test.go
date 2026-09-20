// Patient queue read (#101, FR-17): the acceptance criteria are about who
// may ask and what "no data" looks like on the wire. Everything is real but
// the process boundary — the demo-bypass session service, the claim flow,
// RequirePatientVisit, and the journey service reading its own projection
// and timeline. The service point id is unique to this test, so the answer
// is deterministically the honest no-data one: zero ahead, explicit nulls.
package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"carepath/apps/api/internal/platform/db"
)

const queueVisit = "VISIT-E2E-QUEUE-1"

func seedQueueVisit(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	purge := func() {
		_, _ = database.Querier(ctx).Exec(ctx,
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, queueVisit)
	}
	purge()
	t.Cleanup(purge)
	q := database.Querier(ctx)
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, patient_name, status)
		 VALUES ($1, 'PATIENT-E2E-QUEUE', 'สมหญิง ทดสอบคิว', 'ACTIVE')`, queueVisit,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_step
		     (visit_id, step_key, sequence, kind, clinic_code, round, order_refs,
		      status, service_point_id)
		 VALUES ($1, 'ORDERTYPE:LAB:1', 2, 'LAB', NULL, NULL, '{}', 'READY', 'SP-E2E-QUEUE')`,
		queueVisit); err != nil {
		t.Fatalf("seed step: %v", err)
	}
}

func TestJourneyQueueGuardAndHonestNoData(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping e2e (needs migrated postgres)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)

	seedQueueVisit(t, database)
	app := newShareApp(t, database)
	queuePath := "/api/v1/journeys/" + queueVisit + "/queue"

	// No session: the module's one 401.
	resp, raw := get(t, app, queuePath, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: status %d: %s", resp.StatusCode, raw)
	}

	// A session that has not claimed the visit: the same 404 the journey
	// read gives — ownership must not leak existence (#96).
	token := patientSession(t, app)
	resp, raw = get(t, app, queuePath, token)
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(raw, "journey not found") {
		t.Fatalf("unclaimed: status %d: %s, want 404 journey not found", resp.StatusCode, raw)
	}

	// Claim, then read: the honest no-data answer for a point nobody has
	// waited at today — zero ahead, null averages, never zeros.
	claimResp, _ := doJSON(t, app, http.MethodPost,
		"/api/v1/journeys/"+queueVisit+"/claim", token, `{}`)
	if claimResp.StatusCode != http.StatusNoContent {
		t.Fatalf("claim: status %d", claimResp.StatusCode)
	}
	resp, raw = get(t, app, queuePath, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("queue: status %d: %s", resp.StatusCode, raw)
	}

	var view struct {
		VisitID string `json:"visitId"`
		Steps   []struct {
			StepKey              string   `json:"stepKey"`
			ServicePointID       string   `json:"servicePointId"`
			WaitingAhead         int      `json:"waitingAhead"`
			AvgWaitMinutes       *float64 `json:"avgWaitMinutes"`
			EstimatedWaitMinutes *float64 `json:"estimatedWaitMinutes"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(raw), &view); err != nil {
		t.Fatalf("decode queue: %v", err)
	}
	if view.VisitID != queueVisit || len(view.Steps) != 1 {
		t.Fatalf("view = %s, want the seeded visit with its one READY step", raw)
	}
	step := view.Steps[0]
	if step.StepKey != "ORDERTYPE:LAB:1" || step.ServicePointID != "SP-E2E-QUEUE" {
		t.Fatalf("step = %+v, want the seeded LAB step at SP-E2E-QUEUE", step)
	}
	if step.WaitingAhead != 0 {
		t.Fatalf("waitingAhead = %d, want 0 (nobody else waiting at a test-only point)", step.WaitingAhead)
	}
	if step.AvgWaitMinutes != nil || step.EstimatedWaitMinutes != nil {
		t.Fatalf("averages = %v/%v, want nil/nil — no samples must not become 0",
			step.AvgWaitMinutes, step.EstimatedWaitMinutes)
	}
	if !strings.Contains(raw, `"avgWaitMinutes":null`) || !strings.Contains(raw, `"estimatedWaitMinutes":null`) {
		t.Fatalf("wire format must carry explicit nulls, got %s", raw)
	}
}
