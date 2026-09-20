package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"carepath/apps/api/internal/analytics"
	"carepath/apps/api/internal/platform/db"
)

// Integration test against a real Postgres. Requires the schema (and the
// service-point/place seeds) from infra/postgres/migrations; run
// `make migrate-up` first. Everything the assertions target lives under
// TEST-ANALYTICS-* ids — the query aggregates the whole database, so
// visit-wide scalars are asserted tolerantly (another test package may run
// in parallel) while the dedicated service point's row is exact.
const (
	testSP    = "SP-TEST-ANALYTICS"
	visitA    = "VISIT-TEST-ANALYTICS-A"
	visitB    = "VISIT-TEST-ANALYTICS-B"
	visitC    = "VISIT-TEST-ANALYTICS-C"
	visitD    = "VISIT-TEST-ANALYTICS-D"
	testTimez = "Asia/Bangkok"
)

func newDB(t *testing.T) *db.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (run make up / migrate-up first)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	return database
}

func cleanup(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	purge := func() {
		for _, visit := range []string{visitA, visitB, visitC, visitD} {
			// journey_step_status_event and journey_step cascade on visit delete.
			_, _ = database.Querier(ctx).Exec(ctx,
				`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, visit)
		}
		_, _ = database.Querier(ctx).Exec(ctx,
			`DELETE FROM carepath.service_point WHERE id = $1`, testSP)
	}
	purge()          // a previous run's leftovers
	t.Cleanup(purge) // and this run's
}

// windowStart mirrors the repo's own midnight computation, so "yesterday"
// and "today" placements are relative to exactly the boundary the query
// uses — the test stays deterministic whenever it runs.
func windowStart(t *testing.T, database *db.DB) time.Time {
	t.Helper()
	var ws time.Time
	if err := database.Querier(context.Background()).QueryRow(context.Background(),
		`SELECT date_trunc('day', now() AT TIME ZONE $1) AT TIME ZONE $1`, testTimez,
	).Scan(&ws); err != nil {
		t.Fatalf("compute window start: %v", err)
	}
	return ws
}

type event struct {
	visit, step, kind string
	to                string
	at                time.Time
	unbound           bool // leave service_point_id NULL
}

func ev(visit, step, kind, to string, at time.Time, unbound ...bool) event {
	e := event{visit: visit, step: step, kind: kind, to: to, at: at}
	if len(unbound) > 0 {
		e.unbound = unbound[0]
	}
	return e
}

func seedEvents(t *testing.T, q db.Querier, events []event) {
	t.Helper()
	for _, e := range events {
		sp := any(testSP)
		if e.unbound {
			sp = nil
		}
		if _, err := q.Exec(context.Background(),
			`INSERT INTO carepath.journey_step_status_event
			     (visit_id, step_key, kind, service_point_id, from_status, to_status, source, occurred_at)
			 VALUES ($1, $2, $3, $4, NULL, $5, 'planner', $6)`,
			e.visit, e.step, e.kind, sp, e.to, e.at,
		); err != nil {
			t.Fatalf("seed event %s/%s->%s: %v", e.visit, e.step, e.to, err)
		}
	}
}

func TestFetchAggregatesWindowAndCurrentState(t *testing.T) {
	database := newDB(t)
	cleanup(t, database)
	ctx := context.Background()
	q := database.Querier(ctx)
	ws := windowStart(t, database)

	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.service_point (id, code, name, place_id) VALUES ($1, 'TEST-ANALYTICS', 'Analytics Test Point', 'LAB-01')`,
		testSP); err != nil {
		t.Fatalf("seed service point: %v", err)
	}
	for _, visit := range []struct{ id, status string }{
		{visitA, "ACTIVE"}, {visitB, "ACTIVE"}, {visitC, "COMPLETED"}, {visitD, "COMPLETED"},
	} {
		if _, err := q.Exec(ctx,
			`INSERT INTO carepath.journey_visit (visit_id, patient_ref, status) VALUES ($1, 'PAT-TEST-ANALYTICS', $2)`,
			visit.id, visit.status); err != nil {
			t.Fatalf("seed visit %s: %v", visit.id, err)
		}
	}

	// Visit A carries the "now" state: one step currently READY (its READY
	// event is inserted at now(), so the longest current wait is ~0) and one
	// currently STARTED. The STARTED step has no prior READY event — a staff
	// force-start — so it can never enter a wait pair and the exact wait
	// average below stays exact.
	seedEvents(t, q, []event{
		ev(visitA, "WAIT", "CLINIC", "READY", time.Now()),
		ev(visitA, "IP", "CLINIC", "STARTED", ws.Add(3*time.Minute)),
	})
	for i, step := range []struct{ key, status string }{
		{"WAIT", "READY"}, {"IP", "STARTED"},
	} {
		if _, err := q.Exec(ctx,
			`INSERT INTO carepath.journey_step (visit_id, step_key, sequence, kind, status, service_point_id)
			 VALUES ($1, $2, $3, 'CLINIC', $4, $5)`,
			visitA, step.key, i+1, step.status, testSP,
		); err != nil {
			t.Fatalf("seed journey_step %s: %v", step.key, err)
		}
	}

	// Visit B carries the windowed numbers, anchored to the window start so
	// the arithmetic is exact whenever the test runs:
	//   waits:    (20−10) + (45−30) = 25 min over 2 samples → 12.5
	//   services: (30−20) + (40−25) = 25 min over 2 samples → 12.5
	//   completed today: W1 and W3 → 2
	seedEvents(t, q, []event{
		ev(visitB, "W1", "CLINIC", "READY", ws.Add(10*time.Minute)),
		ev(visitB, "W1", "CLINIC", "STARTED", ws.Add(20*time.Minute)),
		ev(visitB, "W1", "CLINIC", "COMPLETED", ws.Add(30*time.Minute)),
		ev(visitB, "W2", "CLINIC", "READY", ws.Add(30*time.Minute)),
		ev(visitB, "W2", "CLINIC", "STARTED", ws.Add(45*time.Minute)),
		ev(visitB, "W3", "CLINIC", "STARTED", ws.Add(25*time.Minute)),
		ev(visitB, "W3", "CLINIC", "COMPLETED", ws.Add(40*time.Minute)),
	})

	// Visit C completed entirely before the window opened — every number
	// must exclude it (the midnight boundary the AC asks about).
	seedEvents(t, q, []event{
		ev(visitC, "Y1", "CLINIC", "READY", ws.Add(-3*time.Hour)),
		ev(visitC, "Y1", "CLINIC", "STARTED", ws.Add(-150*time.Minute)),
		ev(visitC, "Y1", "CLINIC", "COMPLETED", ws.Add(-2*time.Hour)),
	})

	// Visit D finished today with a one-hour span; it guarantees the
	// visit-wide average has at least one sample. Its events are unbound —
	// they must not touch any service point's row.
	seedEvents(t, q, []event{
		ev(visitD, "D1", "CLINIC", "READY", ws.Add(5*time.Minute), true),
		ev(visitD, "D1", "CLINIC", "COMPLETED", ws.Add(65*time.Minute), true),
	})

	repo := New(database)
	data, err := repo.Fetch(ctx, testTimez)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	var got *analytics.PointData
	for i := range data.Points {
		if data.Points[i].ServicePointID == testSP {
			got = &data.Points[i]
		}
	}
	if got == nil {
		t.Fatalf("service point %s missing from %d points", testSP, len(data.Points))
	}

	if got.WaitingNow != 1 || got.InProgressNow != 1 {
		t.Errorf("now counts = %d waiting / %d in progress, want 1/1", got.WaitingNow, got.InProgressNow)
	}
	if got.AvgWaitMinutes == nil || *got.AvgWaitMinutes != 12.5 {
		t.Errorf("avgWait = %v, want 12.5", got.AvgWaitMinutes)
	}
	if got.AvgServiceMinutes == nil || *got.AvgServiceMinutes != 12.5 {
		t.Errorf("avgService = %v, want 12.5", got.AvgServiceMinutes)
	}
	if got.CompletedCount != 2 {
		t.Errorf("completedCount = %d, want 2 (yesterday's completion excluded)", got.CompletedCount)
	}
	// The still-waiting step's READY event landed at insertion time, so the
	// longest current wait is small but non-negative — its exact value
	// depends on when the test runs.
	if got.LongestWaitingMinutes == nil || *got.LongestWaitingMinutes < 0 || *got.LongestWaitingMinutes > 5 {
		t.Errorf("longestWaiting = %v, want a small non-negative value", got.LongestWaitingMinutes)
	}

	if data.ActiveVisits < 2 {
		t.Errorf("activeVisits = %d, want at least this test's two ACTIVE visits", data.ActiveVisits)
	}
	if data.AvgVisitMinutes == nil {
		t.Error("avgVisitMinutes = nil, want visit D's completion in the sample")
	}
	if data.AvgWaitMinutes == nil {
		t.Error("global avgWaitMinutes = nil, want the windowed samples")
	}
	if data.AsOf.IsZero() {
		t.Error("asOf = zero, want the database clock")
	}
}

// With nothing but yesterday's data, every average must come back NULL —
// SQL's avg over zero rows, not a Go zero.
func TestFetchReturnsNullAveragesWhenNoSamplesToday(t *testing.T) {
	database := newDB(t)
	cleanup(t, database)
	ctx := context.Background()
	q := database.Querier(ctx)
	ws := windowStart(t, database)

	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.service_point (id, code, name, place_id) VALUES ($1, 'TEST-ANALYTICS', 'Analytics Test Point', 'LAB-01')`,
		testSP); err != nil {
		t.Fatalf("seed service point: %v", err)
	}
	if _, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, status) VALUES ($1, 'PAT-TEST-ANALYTICS', 'COMPLETED')`,
		visitC); err != nil {
		t.Fatalf("seed visit: %v", err)
	}
	seedEvents(t, q, []event{
		ev(visitC, "Y1", "CLINIC", "READY", ws.Add(-3*time.Hour)),
		ev(visitC, "Y1", "CLINIC", "STARTED", ws.Add(-150*time.Minute)),
		ev(visitC, "Y1", "CLINIC", "COMPLETED", ws.Add(-2*time.Hour)),
	})

	repo := New(database)
	data, err := repo.Fetch(ctx, testTimez)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	var got *analytics.PointData
	for i := range data.Points {
		if data.Points[i].ServicePointID == testSP {
			got = &data.Points[i]
		}
	}
	if got == nil {
		t.Fatalf("service point %s missing", testSP)
	}
	if got.AvgWaitMinutes != nil || got.AvgServiceMinutes != nil || got.LongestWaitingMinutes != nil {
		t.Errorf("averages = %v/%v/%v, want all nil for yesterday-only data",
			got.AvgWaitMinutes, got.AvgServiceMinutes, got.LongestWaitingMinutes)
	}
	if got.CompletedCount != 0 {
		t.Errorf("completedCount = %d, want 0", got.CompletedCount)
	}
	if got.WaitingNow != 0 || got.InProgressNow != 0 {
		t.Errorf("now counts = %d/%d, want 0/0", got.WaitingNow, got.InProgressNow)
	}
}
