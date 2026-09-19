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
func rnd(v int) *int       { return &v }

func TestUpsertAndGetRoundtrip(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-1")
	repo := New(database)
	ctx := context.Background()

	visit := journey.Visit{
		VisitID: "TEST-JOURNEY-1", PatientRef: "PAT-1", PatientName: "ทดสอบ", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: journey.KindRegistration, Status: "COMPLETED", ServicePointID: sp("SP-REG")},
			{StepKey: "CLINIC:MED:1", Sequence: 2, Kind: journey.KindClinic, ClinicCode: sp("MED"), Round: rnd(1), Status: "READY"}, // unmapped: nil binding
		},
	}
	if err := repo.UpsertVisit(ctx, visit); err != nil {
		t.Fatalf("UpsertVisit: %v", err)
	}

	got, err := repo.GetVisit(ctx, "TEST-JOURNEY-1")
	if err != nil {
		t.Fatalf("GetVisit: %v", err)
	}
	if got.Status != "ACTIVE" || got.PatientName != "ทดสอบ" || len(got.Steps) != 2 || got.SyncedAt.IsZero() {
		t.Fatalf("roundtrip = %+v, want ACTIVE 2 steps with patient name and synced_at set", got)
	}
	if got.Steps[0].ServicePointID == nil || *got.Steps[0].ServicePointID != "SP-REG" {
		t.Fatalf("step 1 service point = %v, want SP-REG", got.Steps[0].ServicePointID)
	}
	if got.Steps[1].ServicePointID != nil {
		t.Fatalf("unmapped step service point = %v, want nil", got.Steps[1].ServicePointID)
	}
	if got.Steps[1].ClinicCode == nil || *got.Steps[1].ClinicCode != "MED" || got.Steps[1].Round == nil || *got.Steps[1].Round != 1 {
		t.Fatalf("clinic step clinic/round = %v/%v, want MED/1", got.Steps[1].ClinicCode, got.Steps[1].Round)
	}
}

// Replaying the same plan — and later a shorter one — must neither
// duplicate steps nor keep stale ones (delete+insert replace semantics).
func TestUpsertIsIdempotentAndReplacesSteps(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-2")
	repo := New(database)
	ctx := context.Background()

	full := journey.Visit{
		VisitID: "TEST-JOURNEY-2", PatientRef: "PAT-2", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "LAB:1", Sequence: 1, Kind: journey.KindLab, Status: "READY", ServicePointID: sp("SP-LAB"), OrderRefs: []string{"ORD-1"}},
			{StepKey: "PHARMACY", Sequence: 2, Kind: journey.KindPharmacy, Status: "PENDING"},
			{StepKey: "EXTRA", Sequence: 3, Kind: journey.KindCashier, Status: "PENDING"},
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
	if len(got.Steps[0].OrderRefs) != 1 || got.Steps[0].OrderRefs[0] != "ORD-1" {
		t.Fatalf("order refs = %v, want [ORD-1]", got.Steps[0].OrderRefs)
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

// #37: the list read serves every projected visit with its steps grouped and
// ordered, freshest sync first. Extra visits from other tests may share the
// database — only the two seeded rows and their relative order are asserted.
func TestListVisitsGroupsStepsNewestFirst(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-L1")
	cleanupJourney(t, database, "TEST-JOURNEY-L2")
	repo := New(database)
	ctx := context.Background()

	older := journey.Visit{
		VisitID: "TEST-JOURNEY-L1", PatientRef: "PAT-L1", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: journey.KindRegistration, Status: "COMPLETED", ServicePointID: sp("SP-REG")},
			{StepKey: "CLINIC:MED:1", Sequence: 2, Kind: journey.KindClinic, Status: "READY"}, // unmapped: nil binding
		},
	}
	newer := journey.Visit{
		VisitID: "TEST-JOURNEY-L2", PatientRef: "PAT-L2", Status: "COMPLETED",
		Steps: []journey.Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: journey.KindRegistration, Status: "COMPLETED", ServicePointID: sp("SP-REG")},
		},
	}
	if err := repo.UpsertVisit(ctx, older); err != nil {
		t.Fatalf("upsert older: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // distinct synced_at for a deterministic order
	if err := repo.UpsertVisit(ctx, newer); err != nil {
		t.Fatalf("upsert newer: %v", err)
	}

	got, err := repo.ListVisits(ctx)
	if err != nil {
		t.Fatalf("ListVisits: %v", err)
	}
	posL1, posL2 := -1, -1
	for i := range got {
		switch got[i].VisitID {
		case "TEST-JOURNEY-L1":
			posL1 = i
		case "TEST-JOURNEY-L2":
			posL2 = i
		}
	}
	if posL1 < 0 || posL2 < 0 {
		t.Fatalf("list = %d visits, want both seeded rows present", len(got))
	}
	if posL2 > posL1 {
		t.Fatalf("positions = older %d, newer %d; want newer first (synced_at DESC)", posL1, posL2)
	}
	seeded := got[posL1]
	if len(seeded.Steps) != 2 || seeded.Steps[0].Sequence != 1 || seeded.Steps[1].Sequence != 2 {
		t.Fatalf("older visit steps = %+v, want 2 steps ordered by sequence", seeded.Steps)
	}
	if seeded.Steps[1].ServicePointID != nil {
		t.Fatalf("unmapped step = %+v, want nil binding", seeded.Steps[1])
	}
	if seeded.SyncedAt.IsZero() {
		t.Fatal("synced_at not loaded")
	}
}

// #19 AC4 (amended by ADR-0009): commands land in the audit table keyed by
// stepKey, and a replayed commandId never duplicates the row.
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
		StepKey: "CLINIC:MED:1", ToStatus: "STARTED", Source: "staff-web",
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

// The close-round table (ADR-0009 §4) is durable per visit/stepKey and
// idempotent.
func TestCloseRoundIsDurableAndIdempotent(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-6")
	repo := New(database)
	ctx := context.Background()

	if err := repo.UpsertVisit(ctx, journey.Visit{VisitID: "TEST-JOURNEY-6", PatientRef: "PAT-6", Status: "ACTIVE"}); err != nil {
		t.Fatalf("seed visit: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := repo.CloseRound(ctx, "TEST-JOURNEY-6", "CLINIC:MED:1"); err != nil {
			t.Fatalf("CloseRound %d: %v", i, err)
		}
	}
	closed, err := repo.ClosedRounds(ctx, "TEST-JOURNEY-6")
	if err != nil {
		t.Fatalf("ClosedRounds: %v", err)
	}
	if !closed["CLINIC:MED:1"] || len(closed) != 1 {
		t.Fatalf("closed = %v, want exactly {CLINIC:MED:1: true}", closed)
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
			Steps: []journey.Step{{StepKey: "LAB:1", Sequence: 1, Kind: journey.KindLab, Status: "READY"}},
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
