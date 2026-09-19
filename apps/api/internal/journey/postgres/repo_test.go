package postgres

import (
	"context"
	"os"
	"testing"

	"carepath/apps/api/internal/journey"
	"carepath/apps/api/internal/platform/db"
)

// Integration test against a real Postgres. Requires the schema from
// infra/postgres/migrations; run `make migrate-up` first.
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

func cleanupJourney(t *testing.T, database *db.DB, visitID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, visitID)
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.his_applied_event WHERE visit_id = $1`, visitID)
	})
}

func sp(id string) *string { return &id }

func TestUpsertAndGetRoundtrip(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-1")
	repo := New(database)
	ctx := context.Background()

	visit := journey.Visit{
		VisitID:    "TEST-JOURNEY-1",
		PatientRef: "PAT-1",
		Status:     "ACTIVE",
		Steps: []journey.Step{
			{Sequence: 1, ServiceCode: "REGISTRATION", Status: "COMPLETED", ServicePointID: sp("SP-REG")},
			{Sequence: 2, ServiceCode: "MYSTERY", Status: "READY"}, // unmapped: nil binding
		},
	}
	if err := repo.UpsertVisit(ctx, visit); err != nil {
		t.Fatalf("UpsertVisit: %v", err)
	}

	got, err := repo.GetVisit(ctx, "TEST-JOURNEY-1")
	if err != nil {
		t.Fatalf("GetVisit: %v", err)
	}
	if got.Status != "ACTIVE" || len(got.Steps) != 2 || got.SyncedAt.IsZero() {
		t.Fatalf("roundtrip = %+v, want ACTIVE 2 steps with synced_at set", got)
	}
	if got.Steps[0].ServicePointID == nil || *got.Steps[0].ServicePointID != "SP-REG" {
		t.Fatalf("step 1 service point = %v, want SP-REG", got.Steps[0].ServicePointID)
	}
	if got.Steps[1].ServicePointID != nil {
		t.Fatalf("unmapped step service point = %v, want nil", got.Steps[1].ServicePointID)
	}
}

// Replaying the same snapshot — and later a shorter one — must neither
// duplicate steps nor keep stale ones (delete+insert replace semantics).
func TestUpsertIsIdempotentAndReplacesSteps(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-2")
	repo := New(database)
	ctx := context.Background()

	full := journey.Visit{
		VisitID: "TEST-JOURNEY-2", PatientRef: "PAT-2", Status: "ACTIVE",
		Steps: []journey.Step{
			{Sequence: 1, ServiceCode: "LAB", Status: "READY", ServicePointID: sp("SP-LAB")},
			{Sequence: 2, ServiceCode: "PHARMACY", Status: "PENDING"},
			{Sequence: 3, ServiceCode: "EXTRA", Status: "PENDING"},
		},
	}
	for i := 0; i < 2; i++ {
		if err := repo.UpsertVisit(ctx, full); err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}
	got, _ := repo.GetVisit(ctx, "TEST-JOURNEY-2")
	if len(got.Steps) != 3 {
		t.Fatalf("steps after replay = %d, want 3 (no duplicates)", len(got.Steps))
	}

	shorter := full
	shorter.Status = "COMPLETED"
	shorter.Steps = full.Steps[:2]
	if err := repo.UpsertVisit(ctx, shorter); err != nil {
		t.Fatalf("upsert shorter: %v", err)
	}
	got, _ = repo.GetVisit(ctx, "TEST-JOURNEY-2")
	if got.Status != "COMPLETED" || len(got.Steps) != 2 {
		t.Fatalf("after replace = %+v, want COMPLETED with stale step 3 removed", got)
	}
}

func TestAppliedEventDedup(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-3")
	repo := New(database)
	ctx := context.Background()

	applied, err := repo.EventApplied(ctx, "TEST-EVT-1")
	if err != nil {
		t.Fatalf("EventApplied: %v", err)
	}
	if applied {
		t.Fatal("unseen event reported as applied")
	}
	for i := 0; i < 2; i++ {
		if err := repo.MarkEventApplied(ctx, "TEST-EVT-1", "TEST-JOURNEY-3"); err != nil {
			t.Fatalf("MarkEventApplied %d: %v", i, err)
		}
	}
	if applied, _ := repo.EventApplied(ctx, "TEST-EVT-1"); !applied {
		t.Fatal("marked event reported as not applied")
	}
}

func TestGetVisitNotFound(t *testing.T) {
	repo := New(newDB(t))
	if _, err := repo.GetVisit(context.Background(), "NOPE"); err != journey.ErrNotFound {
		t.Fatalf("error = %v, want journey.ErrNotFound", err)
	}
}

// Projection writes and the applied-event marker must share one transaction:
// a failure inside the transaction leaves neither behind.
func TestProjectionRollsBackAtomically(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-4")
	repo := New(database)
	ctx := context.Background()

	err := database.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := repo.UpsertVisit(ctx, journey.Visit{
			VisitID: "TEST-JOURNEY-4", PatientRef: "PAT-4", Status: "ACTIVE",
			Steps: []journey.Step{{Sequence: 1, ServiceCode: "LAB", Status: "READY"}},
		}); err != nil {
			return err
		}
		if err := repo.MarkEventApplied(ctx, "TEST-EVT-4", "TEST-JOURNEY-4"); err != nil {
			return err
		}
		return context.DeadlineExceeded // force rollback
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}

	if _, err := repo.GetVisit(ctx, "TEST-JOURNEY-4"); err != journey.ErrNotFound {
		t.Fatalf("visit error = %v, want ErrNotFound after rollback", err)
	}
	if applied, _ := repo.EventApplied(ctx, "TEST-EVT-4"); applied {
		t.Fatal("event marked applied despite rollback")
	}
}
