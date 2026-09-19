// Package journey_test hosts the end-to-end check of the inbound HIS
// boundary (#21 AC4, amended by ADR-0009): a fake HIS speaking the canonical
// HTTP contract, the real httpclient adapter, the ingest poller with its
// postgres checkpoint, and the journey plan backed by postgres — everything
// but the process boundary. Requires DATABASE_URL (see repo-root Makefile
// migrate-up).
package journey_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/his/httpclient"
	"carepath/apps/api/internal/his/ingest"
	ingestpostgres "carepath/apps/api/internal/his/ingest/postgres"
	"carepath/apps/api/internal/hospitalmap"
	hospitalmappostgres "carepath/apps/api/internal/hospitalmap/postgres"
	"carepath/apps/api/internal/journey"
	journeypostgres "carepath/apps/api/internal/journey/postgres"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/servicepoint"
	servicepointpostgres "carepath/apps/api/internal/servicepoint/postgres"
)

// fakeHIS serves the two canonical read surfaces over HTTP: the visit
// snapshot and the append-only event feed with an exclusive cursor.
type fakeHIS struct {
	mu     sync.Mutex
	visit  his.Visit
	events []his.Event
}

func (f *fakeHIS) setVisit(visit his.Visit) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.visit = visit
}

func (f *fakeHIS) append(event his.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeHIS) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/visits/VISIT-E2E":
			_ = json.NewEncoder(w).Encode(f.visit)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/events":
			after := r.URL.Query().Get("after")
			page := his.EventPage{}
			for _, e := range f.events {
				if after == "" || e.EventID > after {
					page.Events = append(page.Events, e)
				}
			}
			if n := len(page.Events); n > 0 {
				page.NextAfter = page.Events[n-1].EventID
			}
			_ = json.NewEncoder(w).Encode(page)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	})
}

var e2eOpenedAt = time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC)

func e2eVisit() his.Visit {
	return his.Visit{
		VisitID: "VISIT-E2E", PatientRef: "PAT-E2E", PatientName: "ทดสอบ E2E",
		VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: e2eOpenedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-E2E-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED",
				OrderedAt: e2eOpenedAt.Add(-time.Hour), Status: his.OrderPlaced},
		},
	}
}

func TestIngestToPlanEndToEnd(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (run make up / migrate-up first)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	ctx := context.Background()

	// Isolate this test's rows and reset the shared ingest cursor.
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := database.Querier(ctx).Exec(ctx, sql, args...); err != nil {
			t.Fatalf("setup exec: %v", err)
		}
	}
	exec(`DELETE FROM carepath.journey_visit WHERE visit_id = 'VISIT-E2E'`)
	exec(`DELETE FROM carepath.his_applied_event WHERE visit_id = 'VISIT-E2E'`)
	exec(`DELETE FROM carepath.journey_command_audit WHERE visit_id = 'VISIT-E2E'`)
	exec(`UPDATE carepath.his_ingest_state SET last_event_id = '' WHERE singleton`)

	upstream := &fakeHIS{visit: e2eVisit()}
	upstream.append(his.Event{
		EventID: "EVT-000001", VisitID: "VISIT-E2E", PatientRef: "PAT-E2E", Type: his.EventVisitOpened,
	})
	server := httptest.NewServer(upstream.handler(t))
	defer server.Close()

	hisClient := httpclient.New(server.URL, server.Client())
	hospitalMap := hospitalmap.NewService(hospitalmappostgres.New(database))
	servicePoints := servicepoint.NewService(servicepointpostgres.New(database), hospitalMap)
	journeys := journey.NewService(hisClient, servicePoints, journeypostgres.New(database), database)
	poller := ingest.New(hisClient, journeys, ingestpostgres.New(database),
		slog.New(slog.NewTextHandler(&discard{}, &slog.HandlerOptions{Level: slog.LevelError})))

	// First poll: the seed fact plans the whole visit — a pre-visit lab
	// before the clinic, per ADR-0009.
	if n, err := poller.PollOnce(ctx); err != nil || n != 1 {
		t.Fatalf("first poll = (%d, %v), want (1, nil)", n, err)
	}
	got, err := journeys.GetVisit(ctx, "VISIT-E2E")
	if err != nil {
		t.Fatalf("GetVisit: %v", err)
	}
	if got.Status != his.VisitActive {
		t.Fatalf("projected status = %s, want ACTIVE", got.Status)
	}
	lab := findStep(t, got.Steps, "LAB:1")
	if lab.ServicePointID == nil || *lab.ServicePointID != "SP-LAB" {
		t.Fatalf("lab binding = %v, want the seeded SP-LAB (ORDERTYPE:LAB)", lab.ServicePointID)
	}
	clinic := findStep(t, got.Steps, "CLINIC:MED:1")
	if clinic.Status != journey.StepPending {
		t.Fatalf("clinic round 1 before the lab is drawn = %s, want PENDING", clinic.Status)
	}

	// #37: the staff monitor's list serves the same plan.
	views, err := journeys.ListJourneys(ctx)
	if err != nil {
		t.Fatalf("ListJourneys: %v", err)
	}
	listed := false
	for _, v := range views {
		if v.VisitID != "VISIT-E2E" {
			continue
		}
		listed = true
		if len(v.Actionable) != 1 || v.Actionable[0].StepKey != "LAB:1" {
			t.Fatalf("listed VISIT-E2E actionable = %+v, want just LAB:1", v.Actionable)
		}
	}
	if !listed {
		t.Fatalf("ListJourneys = %d views, want VISIT-E2E present", len(views))
	}

	// The HIS reports the blood draw: a canonical fact drives the plan
	// forward and the clinic becomes actionable.
	drawn := e2eVisit()
	performedAt := e2eOpenedAt.Add(-30 * time.Minute)
	drawn.Orders[0].Status = his.OrderPerformed
	drawn.Orders[0].PerformedAt = &performedAt
	upstream.setVisit(drawn)
	upstream.append(his.Event{
		EventID: "EVT-000002", VisitID: "VISIT-E2E", PatientRef: "PAT-E2E", Type: his.EventOrderPerformed,
		Payload: map[string]any{"orderRef": "ORD-E2E-1"},
	})
	if n, err := poller.PollOnce(ctx); err != nil || n != 1 {
		t.Fatalf("second poll = (%d, %v), want (1, nil)", n, err)
	}
	view, err := journeys.GetJourney(ctx, "VISIT-E2E")
	if err != nil {
		t.Fatalf("GetJourney: %v", err)
	}
	if len(view.Actionable) != 1 || view.Actionable[0].StepKey != "CLINIC:MED:1" {
		t.Fatalf("actionable after lab performed = %+v, want just CLINIC:MED:1", view.Actionable)
	}

	// Duplicate delivery (cursor rewound): the eventId check answers it and
	// no step is duplicated (#21 AC2).
	exec(`UPDATE carepath.his_ingest_state SET last_event_id = 'EVT-000001' WHERE singleton`)
	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("duplicate-delivery poll: %v", err)
	}
	got, _ = journeys.GetVisit(ctx, "VISIT-E2E")
	if lab := findStep(t, got.Steps, "LAB:1"); lab.Status != journey.StepCompleted {
		t.Fatalf("after duplicate delivery lab status = %s, want COMPLETED unchanged", lab.Status)
	}

	// #19 (amended by ADR-0009): a staff transition is applied locally
	// through the real HTTP-backed journey service, the plan refreshes
	// synchronously, and the command lands in the audit table.
	view, err = journeys.TransitionStep(ctx, "VISIT-E2E", "CLINIC:MED:1",
		journey.TransitionCommand{CommandID: "E2E-CMD-1", To: journey.CommandToStarted}, "e2e-test")
	if err != nil {
		t.Fatalf("TransitionStep STARTED: %v", err)
	}
	if s := findStep(t, toSteps(view.Steps), "CLINIC:MED:1"); s.Status != journey.StepStarted {
		t.Fatalf("clinic after STARTED = %s, want STARTED", s.Status)
	}
	view, err = journeys.TransitionStep(ctx, "VISIT-E2E", "CLINIC:MED:1",
		journey.TransitionCommand{CommandID: "E2E-CMD-2", To: journey.CommandToCompleted}, "e2e-test")
	if err != nil {
		t.Fatalf("TransitionStep COMPLETED: %v", err)
	}
	if len(view.Actionable) != 1 || view.Actionable[0].StepKey != "CASHIER" {
		t.Fatalf("actionable after clinic completed = %+v, want just CASHIER", view.Actionable)
	}
	var audited int
	if err := database.Querier(ctx).QueryRow(ctx,
		`SELECT count(*) FROM carepath.journey_command_audit WHERE visit_id = 'VISIT-E2E' AND source = 'e2e-test'`,
	).Scan(&audited); err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if audited != 2 {
		t.Fatalf("audited commands = %d, want 2", audited)
	}
}

func findStep(t *testing.T, steps []journey.Step, key string) journey.Step {
	t.Helper()
	for _, s := range steps {
		if s.StepKey == key {
			return s
		}
	}
	t.Fatalf("step %q not found in %+v", key, steps)
	return journey.Step{}
}

func toSteps(views []journey.StepView) []journey.Step {
	out := make([]journey.Step, len(views))
	for i, v := range views {
		out[i] = journey.Step{StepKey: v.StepKey, Status: v.Status}
	}
	return out
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
