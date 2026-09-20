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

// #85: timeline rows round-trip through every column, and deleting the visit
// cascades them away with it.
func TestAppendStatusEventsRoundtripAndCascade(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-TL")
	repo := New(database)
	ctx := context.Background()

	if err := repo.UpsertVisit(ctx, journey.Visit{
		VisitID: "TEST-JOURNEY-TL", PatientRef: "PAT-TL", Status: "ACTIVE",
		Steps: []journey.Step{{StepKey: "CLINIC:MED:1", Sequence: 1, Kind: journey.KindClinic, Status: "READY"}},
	}); err != nil {
		t.Fatalf("seed visit: %v", err)
	}

	from := journey.StepReady
	events := []journey.StepStatusEvent{
		{
			VisitID: "TEST-JOURNEY-TL", StepKey: "CLINIC:MED:1", Kind: journey.KindClinic,
			ServicePointID: sp("SP-CLINIC-MED"), FromStatus: &from, ToStatus: journey.StepStarted,
			Source: "staff-web", ActorUserID: "user-1", ActorUsername: "tester",
		},
		{
			VisitID: "TEST-JOURNEY-TL", StepKey: "CASHIER", Kind: journey.KindCashier,
			ToStatus: journey.StepPending, Source: journey.EventSourcePlanner,
		},
	}
	if err := repo.AppendStatusEvents(ctx, events); err != nil {
		t.Fatalf("AppendStatusEvents: %v", err)
	}

	rows, err := database.Querier(ctx).Query(ctx,
		`SELECT step_key, kind, service_point_id, from_status, to_status, source, actor_user_id, actor_username, occurred_at
		 FROM carepath.journey_step_status_event WHERE visit_id = $1 ORDER BY event_id`, "TEST-JOURNEY-TL")
	if err != nil {
		t.Fatalf("query timeline: %v", err)
	}
	defer rows.Close()

	var got []journey.StepStatusEvent
	var occurred []time.Time
	for rows.Next() {
		var ev journey.StepStatusEvent
		var at time.Time
		if err := rows.Scan(&ev.StepKey, &ev.Kind, &ev.ServicePointID, &ev.FromStatus, &ev.ToStatus,
			&ev.Source, &ev.ActorUserID, &ev.ActorUsername, &at); err != nil {
			t.Fatalf("scan timeline row: %v", err)
		}
		ev.VisitID = "TEST-JOURNEY-TL"
		got = append(got, ev)
		occurred = append(occurred, at)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate timeline: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("timeline rows = %d, want 2", len(got))
	}
	first, second := got[0], got[1]
	if first.FromStatus == nil || *first.FromStatus != journey.StepReady || first.ToStatus != journey.StepStarted ||
		first.Source != "staff-web" || first.ActorUserID != "user-1" || first.ActorUsername != "tester" ||
		first.Kind != journey.KindClinic || first.ServicePointID == nil || *first.ServicePointID != "SP-CLINIC-MED" {
		t.Fatalf("first row = %+v, want every column of the command event back", first)
	}
	if second.FromStatus != nil || second.ToStatus != journey.StepPending ||
		second.Source != journey.EventSourcePlanner || second.ActorUserID != "" || second.ActorUsername != "" {
		t.Fatalf("second row = %+v, want a planner row with nil from_status and no actor", second)
	}
	for _, at := range occurred {
		if at.IsZero() {
			t.Fatal("occurred_at not defaulted")
		}
	}

	if _, err := database.Querier(ctx).Exec(ctx,
		`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, "TEST-JOURNEY-TL"); err != nil {
		t.Fatalf("delete visit: %v", err)
	}
	var count int
	if err := database.Querier(ctx).QueryRow(ctx,
		`SELECT count(*) FROM carepath.journey_step_status_event WHERE visit_id = $1`, "TEST-JOURNEY-TL",
	).Scan(&count); err != nil {
		t.Fatalf("count after delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("timeline rows after visit delete = %d, want 0 (cascade)", count)
	}
}

// #101: QueueStats must count only OTHER visits' READY steps per service
// point, average today's experienced waits (latest READY → latest STARTED
// per step, window resolved in the database), and keep "no samples" absent
// rather than zero.
func TestQueueStatsCountsAndWindow(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-QA")
	cleanupJourney(t, database, "TEST-JOURNEY-QB")
	cleanupJourney(t, database, "TEST-JOURNEY-QC")
	repo := New(database)
	ctx := context.Background()

	// The asking visit: its own READY step at SP-QA must never count as
	// someone ahead of itself.
	if err := repo.UpsertVisit(ctx, journey.Visit{
		VisitID: "TEST-JOURNEY-QA", PatientRef: "PAT-QA", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "LAB:1", Sequence: 1, Kind: journey.KindLab, Status: journey.StepReady, ServicePointID: sp("SP-QA")},
		},
	}); err != nil {
		t.Fatalf("seed asker: %v", err)
	}
	// Two other visits waiting at SP-QA, one at SP-QB (which has no samples).
	if err := repo.UpsertVisit(ctx, journey.Visit{
		VisitID: "TEST-JOURNEY-QB", PatientRef: "PAT-QB", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "LAB:1", Sequence: 1, Kind: journey.KindLab, Status: journey.StepReady, ServicePointID: sp("SP-QA")},
			{StepKey: "XRAY:1", Sequence: 2, Kind: journey.KindXray, Status: journey.StepReady, ServicePointID: sp("SP-QA")},
			{StepKey: "EKG:1", Sequence: 3, Kind: journey.KindEKG, Status: journey.StepReady, ServicePointID: sp("SP-QB")},
		},
	}); err != nil {
		t.Fatalf("seed waiting visit: %v", err)
	}
	// A finished visit whose LAB wait (20 minutes inside today's window)
	// becomes SP-QA's average; a second, yesterday-dated pair at SP-QB must
	// stay outside the window.
	if err := repo.UpsertVisit(ctx, journey.Visit{
		VisitID: "TEST-JOURNEY-QC", PatientRef: "PAT-QC", Status: "COMPLETED",
		Steps: []journey.Step{
			{StepKey: "LAB:1", Sequence: 1, Kind: journey.KindLab, Status: journey.StepCompleted, ServicePointID: sp("SP-QA")},
		},
	}); err != nil {
		t.Fatalf("seed finished visit: %v", err)
	}
	timeline := `
		INSERT INTO carepath.journey_step_status_event
			(visit_id, step_key, kind, service_point_id, to_status, source, occurred_at)
		VALUES
			('TEST-JOURNEY-QC', 'LAB:1', 'LAB', 'SP-QA', 'READY', 'planner',
			 (date_trunc('day', now() AT TIME ZONE $1) AT TIME ZONE $1) + interval '1 hour'),
			('TEST-JOURNEY-QC', 'LAB:1', 'LAB', 'SP-QA', 'STARTED', 'planner',
			 (date_trunc('day', now() AT TIME ZONE $1) AT TIME ZONE $1) + interval '1 hour 20 minutes'),
			('TEST-JOURNEY-QB', 'EKG:1', 'EKG', 'SP-QB', 'READY', 'planner',
			 (date_trunc('day', now() AT TIME ZONE $1) AT TIME ZONE $1) - interval '1 day'),
			('TEST-JOURNEY-QB', 'EKG:1', 'EKG', 'SP-QB', 'STARTED', 'planner',
			 (date_trunc('day', now() AT TIME ZONE $1) AT TIME ZONE $1) - interval '1 day' + interval '5 minutes')`
	if _, err := database.Querier(ctx).Exec(ctx, timeline, "Asia/Bangkok"); err != nil {
		t.Fatalf("seed timeline: %v", err)
	}

	stats, err := repo.QueueStats(ctx, "TEST-JOURNEY-QA", []string{"SP-QA", "SP-QB"}, "Asia/Bangkok")
	if err != nil {
		t.Fatalf("QueueStats: %v", err)
	}

	qa, ok := stats["SP-QA"]
	if !ok {
		t.Fatal("SP-QA absent from stats")
	}
	if qa.WaitingAhead != 2 {
		t.Fatalf("SP-QA waitingAhead = %d, want 2 (other visits only; the asker excluded)", qa.WaitingAhead)
	}
	if qa.AvgWaitMinutes == nil || absF(*qa.AvgWaitMinutes-20.0) > 0.001 {
		t.Fatalf("SP-QA avgWait = %v, want ~20 minutes from today's pair", qa.AvgWaitMinutes)
	}

	qb, ok := stats["SP-QB"]
	if !ok {
		t.Fatal("SP-QB absent from stats despite a waiting step")
	}
	if qb.WaitingAhead != 1 {
		t.Fatalf("SP-QB waitingAhead = %d, want 1", qb.WaitingAhead)
	}
	if qb.AvgWaitMinutes != nil {
		t.Fatalf("SP-QB avgWait = %v, want nil (only yesterday's pair — outside the window)", *qb.AvgWaitMinutes)
	}
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestStationQueueListsServingAndWaiting(t *testing.T) {
	database := newDB(t)
	cleanupJourney(t, database, "TEST-JOURNEY-SA")
	cleanupJourney(t, database, "TEST-JOURNEY-SB")
	cleanupJourney(t, database, "TEST-JOURNEY-SC")
	repo := New(database)
	ctx := context.Background()

	// One visit being served, one waiting (earlier arrival), one waiting
	// (later) whose second READY event is the arrival that counts — plus a
	// PENDING step at the same point that must not appear at all.
	if err := repo.UpsertVisit(ctx, journey.Visit{
		VisitID: "TEST-JOURNEY-SA", PatientRef: "PAT-SA", PatientName: "กำลังให้บริการ", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "LAB:1", Sequence: 1, Kind: journey.KindLab, Status: journey.StepStarted, ServicePointID: sp("SP-QA")},
			{StepKey: "CASHIER:1", Sequence: 2, Kind: journey.KindCashier, Status: journey.StepPending, ServicePointID: sp("SP-QA")},
		},
	}); err != nil {
		t.Fatalf("seed serving visit: %v", err)
	}
	if err := repo.UpsertVisit(ctx, journey.Visit{
		VisitID: "TEST-JOURNEY-SB", PatientRef: "PAT-SB", PatientName: "รอก่อน", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "LAB:1", Sequence: 1, Kind: journey.KindLab, Status: journey.StepReady, ServicePointID: sp("SP-QA")},
		},
	}); err != nil {
		t.Fatalf("seed early waiting visit: %v", err)
	}
	if err := repo.UpsertVisit(ctx, journey.Visit{
		VisitID: "TEST-JOURNEY-SC", PatientRef: "PAT-SC", PatientName: "รอหลัง", Status: "ACTIVE",
		Steps: []journey.Step{
			{StepKey: "XRAY:1", Sequence: 1, Kind: journey.KindXray, Status: journey.StepReady, ServicePointID: sp("SP-QA")},
		},
	}); err != nil {
		t.Fatalf("seed later waiting visit: %v", err)
	}
	timeline := `
		INSERT INTO carepath.journey_step_status_event
			(visit_id, step_key, kind, service_point_id, to_status, source, occurred_at)
		VALUES
			('TEST-JOURNEY-SA', 'LAB:1', 'LAB', 'SP-QA', 'READY', 'planner', now() - interval '40 minutes'),
			('TEST-JOURNEY-SA', 'LAB:1', 'LAB', 'SP-QA', 'STARTED', 'planner', now() - interval '10 minutes'),
			('TEST-JOURNEY-SB', 'LAB:1', 'LAB', 'SP-QA', 'READY', 'planner', now() - interval '25 minutes'),
			('TEST-JOURNEY-SC', 'XRAY:1', 'XRAY', 'SP-QA', 'READY', 'planner', now() - interval '5 minutes'),
			('TEST-JOURNEY-SC', 'XRAY:1', 'XRAY', 'SP-QA', 'STARTED', 'planner', now() - interval '4 minutes'),
			('TEST-JOURNEY-SC', 'XRAY:1', 'XRAY', 'SP-QA', 'READY', 'planner', now() - interval '3 minutes')`
	if _, err := database.Querier(ctx).Exec(ctx, timeline); err != nil {
		t.Fatalf("seed timeline: %v", err)
	}

	entries, err := repo.StationQueue(ctx, "SP-QA")
	if err != nil {
		t.Fatalf("StationQueue: %v", err)
	}
	queue := journey.StationQueue{} // shaping is domain-tested; assert raw feed facts
	for _, e := range entries {
		switch e.Status {
		case journey.StepStarted:
			queue.Serving = append(queue.Serving, e)
		case journey.StepReady:
			queue.Waiting = append(queue.Waiting, e)
		default:
			t.Fatalf("status %q leaked into the station queue (%+v)", e.Status, e)
		}
	}
	if len(queue.Serving) != 1 || queue.Serving[0].VisitID != "TEST-JOURNEY-SA" {
		t.Fatalf("serving = %+v, want only TEST-JOURNEY-SA", queue.Serving)
	}
	s := queue.Serving[0]
	if s.PatientName != "กำลังให้บริการ" || s.TotalSteps != 2 || s.Sequence != 1 {
		t.Fatalf("serving entry = %+v, want patient detail and 1-of-2 position", s)
	}
	if s.ReadyAt == nil || s.StartedAt == nil {
		t.Fatalf("serving entry = %+v, want both READY and STARTED times from the timeline", s)
	}
	if len(queue.Waiting) != 2 {
		t.Fatalf("waiting = %+v, want the two READY entries only", queue.Waiting)
	}
	byVisit := map[string]journey.StationQueueEntry{}
	for _, w := range queue.Waiting {
		byVisit[w.VisitID] = w
	}
	sb, sc := byVisit["TEST-JOURNEY-SB"], byVisit["TEST-JOURNEY-SC"]
	if sb.ReadyAt == nil || sc.ReadyAt == nil {
		t.Fatal("waiting entries carry no arrival time")
	}
	// SC's step re-entered READY after a brief STARTED: its arrival is the
	// LATEST READY event, not the first — same pairing rule as the wait stats.
	if !sc.ReadyAt.After(nowMinus(t, 4*time.Minute)) {
		t.Fatalf("SC readyAt = %v, want the latest READY event (~3 min ago)", sc.ReadyAt)
	}
	if sc.ReadyAt.Before(*sb.ReadyAt) {
		t.Fatalf("SC (%v) must have arrived after SB (%v)", sc.ReadyAt, sb.ReadyAt)
	}
	if sc.StartedAt != nil {
		t.Fatalf("SC startedAt = %v, want nil while waiting", sc.StartedAt)
	}
}

func nowMinus(t *testing.T, d time.Duration) time.Time {
	t.Helper()
	// The timeline rows above are anchored to now(); a 30-second slop keeps
	// this comparison about minutes, not test-runtime jitter.
	return time.Now().Add(-d).Add(-30 * time.Second)
}
