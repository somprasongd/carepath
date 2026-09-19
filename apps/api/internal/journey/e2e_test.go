// Package journey_test hosts the end-to-end check of the inbound HIS
// boundary (#21 AC4): a fake HIS speaking the canonical HTTP contract, the
// real httpclient adapter, the ingest poller with its postgres checkpoint,
// and the journey projection backed by postgres — everything but the process
// boundary. Requires DATABASE_URL (see repo-root Makefile migrate-up).
package journey_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/his/httpclient"
	"carepath/apps/api/internal/his/ingest"
	ingestpostgres "carepath/apps/api/internal/his/ingest/postgres"
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
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/visits/VISIT-E2E/steps/") && strings.HasSuffix(r.URL.Path, "/transition"):
			var cmd his.TransitionCommand
			if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil || cmd.CommandID == "" || cmd.To == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid command"})
				return
			}
			sequence, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/visits/VISIT-E2E/steps/"), "/transition"))
			for i := range f.visit.Steps {
				if f.visit.Steps[i].Sequence != sequence {
					continue
				}
				switch {
				case f.visit.Steps[i].Status == cmd.To:
					// transition to current status: no-op success (contract)
				case f.visit.Steps[i].Status == "COMPLETED" || f.visit.Steps[i].Status == "CANCELLED":
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "step is terminal"})
					return
				default:
					f.visit.Steps[i].Status = cmd.To
					if cmd.To == "COMPLETED" {
						for j := range f.visit.Steps {
							if f.visit.Steps[j].Status == "PENDING" {
								f.visit.Steps[j].Status = "READY"
								break
							}
						}
					}
				}
				_ = json.NewEncoder(w).Encode(f.visit.Steps[i])
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "step not found"})
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

func e2eVisit() his.Visit {
	return his.Visit{
		VisitID:    "VISIT-E2E",
		PatientRef: "PAT-E2E",
		Status:     "ACTIVE",
		Steps: []his.VisitStep{
			{Sequence: 1, ServiceCode: "REGISTRATION", Status: "COMPLETED"},
			{Sequence: 2, ServiceCode: "LAB", Status: "READY"},
			{Sequence: 3, ServiceCode: "PHARMACY", Status: "PENDING"},
			{Sequence: 4, ServiceCode: "MYSTERY", Status: "PENDING"},
		},
	}
}

func TestIngestToProjectionEndToEnd(t *testing.T) {
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
	exec(`UPDATE carepath.his_ingest_state SET last_event_id = '' WHERE singleton`)

	upstream := &fakeHIS{visit: e2eVisit()}
	upstream.append(his.Event{
		EventID: "EVT-000001", VisitID: "VISIT-E2E", PatientRef: "PAT-E2E", Type: his.EventVisitOpened,
		Payload: map[string]any{"status": "ACTIVE"},
	})
	server := httptest.NewServer(upstream.handler(t))
	defer server.Close()

	hisClient := httpclient.New(server.URL, server.Client())
	servicePoints := servicepoint.NewService(servicepointpostgres.New(database))
	journeys := journey.NewService(hisClient, servicePoints, journeypostgres.New(database), database)
	poller := ingest.New(hisClient, journeys, ingestpostgres.New(database),
		slog.New(slog.NewTextHandler(&discard{}, &slog.HandlerOptions{Level: slog.LevelError})))

	// First poll: the seed event projects the whole journey.
	if n, err := poller.PollOnce(ctx); err != nil || n != 1 {
		t.Fatalf("first poll = (%d, %v), want (1, nil)", n, err)
	}
	got, err := journeys.GetVisit(ctx, "VISIT-E2E")
	if err != nil {
		t.Fatalf("GetVisit: %v", err)
	}
	if got.Status != "ACTIVE" || len(got.Steps) != 4 {
		t.Fatalf("projected = %+v, want ACTIVE with 4 steps", got)
	}
	if got.Steps[1].ServicePointID == nil || *got.Steps[1].ServicePointID != "SP-LAB" {
		t.Fatalf("LAB binding = %v, want seeded SP-LAB", got.Steps[1].ServicePointID)
	}
	if got.Steps[3].ServicePointID != nil {
		t.Fatalf("MYSTERY binding = %v, want nil (unmapped is explicit)", got.Steps[3].ServicePointID)
	}

	// The HIS completes LAB: a canonical event drives the projection forward.
	completed := e2eVisit()
	completed.Steps[1].Status = "COMPLETED"
	completed.Steps[2].Status = "READY"
	upstream.setVisit(completed)
	upstream.append(his.Event{
		EventID: "EVT-000002", VisitID: "VISIT-E2E", PatientRef: "PAT-E2E",
		Type:    his.EventServiceCompleted,
		Payload: map[string]any{"sequence": 2, "serviceCode": "LAB"},
	})
	if n, err := poller.PollOnce(ctx); err != nil || n != 1 {
		t.Fatalf("second poll = (%d, %v), want (1, nil)", n, err)
	}
	got, _ = journeys.GetVisit(ctx, "VISIT-E2E")
	if got.Steps[1].Status != "COMPLETED" || got.Steps[2].Status != "READY" {
		t.Fatalf("statuses = %s/%s, want COMPLETED/READY after HIS event",
			got.Steps[1].Status, got.Steps[2].Status)
	}

	// #18: the journey view reads the projection — ordered steps, resolved
	// service points, and the deterministic next step after the transition.
	view, err := journeys.GetJourney(ctx, "VISIT-E2E")
	if err != nil {
		t.Fatalf("GetJourney: %v", err)
	}
	if view.Completed || view.Current != nil {
		t.Fatalf("completed/current = %v/%+v, want false/nil mid-journey", view.Completed, view.Current)
	}
	if view.Next == nil || view.Next.Sequence != 3 || view.Next.ServiceCode != "PHARMACY" {
		t.Fatalf("next = %+v, want PHARMACY at sequence 3", view.Next)
	}
	if view.Next.ServicePoint == nil || view.Next.ServicePoint.ID != "SP-PHARMACY" {
		t.Fatalf("next service point = %+v, want seeded SP-PHARMACY", view.Next.ServicePoint)
	}
	if view.Steps[3].ServicePointID != nil || view.Steps[3].ServicePoint != nil {
		t.Fatalf("unmapped MYSTERY step = %+v, want nil binding", view.Steps[3])
	}

	// Duplicate delivery (cursor rewound): the eventId check answers it and
	// no step is duplicated (#21 AC2).
	exec(`UPDATE carepath.his_ingest_state SET last_event_id = 'EVT-000001' WHERE singleton`)
	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("duplicate-delivery poll: %v", err)
	}
	got, _ = journeys.GetVisit(ctx, "VISIT-E2E")
	if len(got.Steps) != 4 || got.Steps[1].Status != "COMPLETED" {
		t.Fatalf("after duplicate delivery = %d steps, status %s; want 4 steps, no change",
			len(got.Steps), got.Steps[1].Status)
	}

	// #19: a transition command goes through the real HTTP adapter to the
	// HIS, the projection refreshes synchronously, next is recomputed, and
	// the command lands in the audit table.
	view, err = journeys.TransitionStep(ctx, "VISIT-E2E", 3,
		his.TransitionCommand{CommandID: "E2E-CMD-1", To: "STARTED"}, "e2e-test")
	if err != nil {
		t.Fatalf("TransitionStep STARTED: %v", err)
	}
	if view.Current == nil || view.Current.Sequence != 3 || view.Current.Status != "STARTED" {
		t.Fatalf("current after STARTED = %+v, want PHARMACY STARTED", view.Current)
	}
	view, err = journeys.TransitionStep(ctx, "VISIT-E2E", 3,
		his.TransitionCommand{CommandID: "E2E-CMD-2", To: "COMPLETED"}, "e2e-test")
	if err != nil {
		t.Fatalf("TransitionStep COMPLETED: %v", err)
	}
	if view.Completed || view.Current != nil {
		t.Fatalf("completed/current after COMPLETED = %v/%+v, want false/nil mid-journey", view.Completed, view.Current)
	}
	if view.Next == nil || view.Next.Sequence != 4 || view.Next.Status != "READY" {
		t.Fatalf("next after COMPLETED = %+v, want MYSTERY READY at sequence 4", view.Next)
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
	exec(`DELETE FROM carepath.journey_command_audit WHERE visit_id = 'VISIT-E2E'`)
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
