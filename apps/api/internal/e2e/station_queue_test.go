// Staff station queue (#102, FR-15): the acceptance criteria are who may
// read a station's queue (assignment, reported as missing on the issue
// before the user_service_point migration), and that the console's call
// action really moves a step — the same staff transition endpoint the
// patients' screen reflects. The call leg replans from a scripted HIS
// snapshot, so the seeded step must match the planner exactly: step key
// LAB:1 bound to the seeded Laboratory point. Queue answers are
// contains-based because the Laboratory point also serves local demo data.
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

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/db"
)

const (
	stationVisit    = "VISIT-E2E-STATION-1"
	stationPoint    = "SP-ORDERTYPE-LAB" // the planner's binding for a LAB order
	stationStep     = "LAB:1"            // the planner's step key for that order
	unassignedPoint = "SP-REG"
)

func seedStationVisit(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	purge := func() {
		_, _ = database.Querier(ctx).Exec(ctx,
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, stationVisit)
	}
	purge()
	t.Cleanup(purge)
	q := database.Querier(ctx)
	// The Laboratory point and the demo staff account's assignment to it come
	// from migrations; the visit and its waiting step are ours alone.
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, patient_name, status)
		 VALUES ($1, 'PATIENT-E2E-STATION', 'สมชาย รอห้องแล็บ', 'ACTIVE')`, stationVisit,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_step
		     (visit_id, step_key, sequence, kind, clinic_code, round, order_refs,
		      status, service_point_id)
		 VALUES ($1, $2, 3, 'LAB', 'MED', 1, '{ORD-E2E-STATION}', 'READY', $3)`,
		stationVisit, stationStep, stationPoint); err != nil {
		t.Fatalf("seed step: %v", err)
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_step_status_event
		     (visit_id, step_key, kind, service_point_id, to_status, source, occurred_at)
		 VALUES ($1, $2, 'LAB', $3, 'READY', 'planner', now() - interval '12 minutes')`,
		stationVisit, stationStep, stationPoint); err != nil {
		t.Fatalf("seed timeline: %v", err)
	}
}

// stationHIS serves the snapshot the transition replans from: the same
// walkin + MED clinic + placed LAB order facts that produce a waiting LAB:1
// step bound to the Laboratory point.
func stationHIS(t *testing.T) *httptest.Server {
	t.Helper()
	visit := his.Visit{
		VisitID: stationVisit, PatientRef: "PATIENT-E2E-STATION", PatientName: "สมชาย รอห้องแล็บ",
		VisitType: his.VisitTypeWalkin, Status: his.VisitActive,
		Clinics:  []his.Clinic{{Code: "MED", Name: "อายุรกรรม"}},
		OpenedAt: time.Now().UTC().Add(-15 * time.Minute),
		Orders: []his.Order{{
			OrderRef: "ORD-E2E-STATION", OrderType: his.OrderTypeLab, OrderName: "CBC",
			OrderedByClinic: "MED", OrderedAt: time.Now().UTC().Add(-13 * time.Minute),
			Status: his.OrderPlaced,
		}},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/visits/"+stationVisit {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(visit)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func post(t *testing.T, app *fiber.App, path, token, body string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

type stationEntry struct {
	VisitID     string  `json:"visitId"`
	StepKey     string  `json:"stepKey"`
	PatientName string  `json:"patientName"`
	ReadyAt     *string `json:"readyAt"`
}

type stationQueue struct {
	ServicePointID string         `json:"servicePointId"`
	Serving        []stationEntry `json:"serving"`
	Waiting        []stationEntry `json:"waiting"`
}

func (q stationQueue) servingHas(visitID string) bool {
	for _, e := range q.Serving {
		if e.VisitID == visitID {
			return true
		}
	}
	return false
}

func (q stationQueue) waitingEntry(visitID string) *stationEntry {
	for i := range q.Waiting {
		if q.Waiting[i].VisitID == visitID {
			return &q.Waiting[i]
		}
	}
	return nil
}

func TestStationQueueAssignmentAndCallNext(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping e2e (needs migrated postgres)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)

	seedStationVisit(t, database)
	hisServer := stationHIS(t)
	t.Cleanup(hisServer.Close)
	app := newShareAppWithHIS(t, database, hisServer.URL, nil)

	// The picker feed: staff sees the seeded assignment, not the unassigned
	// points; admin sees the full list; and no token is rejected.
	resp, raw := get(t, app, "/api/v1/staff/my/service-points", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: status %d, want 401", resp.StatusCode)
	}
	staffToken := login(t, app, "staff", "demo")
	resp, raw = get(t, app, "/api/v1/staff/my/service-points", staffToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("staff my/service-points: status %d body %s", resp.StatusCode, raw)
	}
	if !strings.Contains(raw, `"id":"`+stationPoint+`"`) {
		t.Fatalf("staff points = %s, want the assigned %s", raw, stationPoint)
	}
	if strings.Contains(raw, `"id":"`+unassignedPoint+`"`) {
		t.Fatalf("staff points = %s, must not leak unassigned points", raw)
	}
	adminToken := login(t, app, "admin", "demo")
	resp, raw = get(t, app, "/api/v1/staff/my/service-points", adminToken)
	if resp.StatusCode != http.StatusOK || !strings.Contains(raw, `"id":"`+unassignedPoint+`"`) {
		t.Fatalf("admin my/service-points = %d %s, want the full list including %s",
			resp.StatusCode, raw, unassignedPoint)
	}

	// The queue read: unassigned staff 403, assigned staff 200 with the
	// waiting patient carrying patient detail and an arrival time.
	resp, raw = get(t, app, "/api/v1/staff/queue/"+unassignedPoint, staffToken)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("staff → unassigned point: status %d body %s, want 403", resp.StatusCode, raw)
	}
	var queue stationQueue
	readQueue := func() {
		t.Helper()
		resp, raw = get(t, app, "/api/v1/staff/queue/"+stationPoint, staffToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("queue read: status %d body %s", resp.StatusCode, raw)
		}
		if err := json.Unmarshal([]byte(raw), &queue); err != nil {
			t.Fatalf("decode queue %s: %v", raw, err)
		}
		if queue.ServicePointID != stationPoint {
			t.Fatalf("queue servicePointId = %s, want %s", queue.ServicePointID, stationPoint)
		}
	}
	readQueue()
	if queue.servingHas(stationVisit) {
		t.Fatalf("queue = %s, want %s still waiting, not serving", raw, stationVisit)
	}
	waiting := queue.waitingEntry(stationVisit)
	if waiting == nil {
		t.Fatalf("queue = %s, want %s waiting", raw, stationVisit)
	}
	if waiting.StepKey != stationStep || waiting.PatientName != "สมชาย รอห้องแล็บ" ||
		waiting.ReadyAt == nil {
		t.Fatalf("waiting entry = %+v, want patient detail and an arrival time", waiting)
	}

	// Call next — the console's action is the existing staff transition;
	// the queue must immediately show the step as being served.
	resp, raw = post(t, app, "/api/v1/journeys/"+stationVisit+"/steps/"+stationStep+"/transition",
		staffToken, `{"to":"STARTED","source":"staff-web"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("call next: status %d body %s", resp.StatusCode, raw)
	}
	readQueue()
	if !queue.servingHas(stationVisit) {
		t.Fatalf("queue after call = %s, want %s serving", raw, stationVisit)
	}
	if queue.waitingEntry(stationVisit) != nil {
		t.Fatalf("queue after call = %s, want %s out of the waiting list", raw, stationVisit)
	}
}
