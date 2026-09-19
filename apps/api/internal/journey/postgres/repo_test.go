package postgres

import (
	"context"
	"os"
	"testing"
	"time"

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

// #19 AC4: commands land in the audit table with timestamp and source, and a
// replayed commandId (retry after a failed local transaction) never
// duplicates the row.
func TestInsertCommandAuditDedupesByCommandID(t *testing.T) {
	database := newDB(t)
	repo := New(database)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = database.Querier(ctx).Exec(ctx,
			`DELETE FROM carepath.journey_command_audit WHERE command_id = $1`, "TEST-CMD-1")
	})

	audit := journey.CommandAudit{
		CommandID: "TEST-CMD-1", VisitID: "TEST-JOURNEY-5",
		Sequence: 2, ToStatus: "STARTED", Source: "staff-web",
	}
	for i := 0; i < 2; i++ {
		if err := repo.InsertCommandAudit(ctx, audit); err != nil {
			t.Fatalf("InsertCommandAudit %d: %v", i, err)
		}
	}

	var count int
	var source string
	var createdAt *time.Time
	err := database.Querier(ctx).QueryRow(ctx,
		`SELECT count(*), min(source), min(created_at) FROM carepath.journey_command_audit WHERE command_id = $1`,
		"TEST-CMD-1",
	).Scan(&count, &source, &createdAt)
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if count != 1 || source != "staff-web" || createdAt == nil || createdAt.IsZero() {
		t.Fatalf("audit = count %d source %s created_at %v, want 1 row with source and timestamp", count, source, createdAt)
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
