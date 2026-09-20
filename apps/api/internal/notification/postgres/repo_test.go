package postgres

import (
	"context"
	"os"
	"testing"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

// Integration test against a real Postgres, following the other modules'
// repo-test harness: requires the schema from infra/postgres/migrations and
// skips without DATABASE_URL.
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

// The store's contract is the exactly-once claim and the absent-means-
// enabled preference (ADR-0013 §3/§4); the recipients port maps a visit to
// the line-source identity that claimed it, ignoring demo claims.
func TestStoreAndRecipients(t *testing.T) {
	database := newDB(t)
	store := New(database)
	recipients := NewRecipients(database)
	ctx := context.Background()

	const visitID = "VISIT-TEST-NOTIF-1"
	cleanup := func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, visitID)
	}
	cleanup()
	t.Cleanup(cleanup)

	if _, err := database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, status)
		 VALUES ($1, 'PATIENT-NOTIF-TEST', 'ACTIVE')`, visitID,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}

	// Absent preference reads as enabled.
	if enabled, err := store.Enabled(ctx, visitID); err != nil || !enabled {
		t.Fatalf("Enabled before any write: %v %v, want true nil", enabled, err)
	}

	// The claim is atomic: first caller wins, second does not.
	won, err := store.SendOnce(ctx, visitID, "LAB:1", "line")
	if err != nil || !won {
		t.Fatalf("first SendOnce: %v %v, want true nil", won, err)
	}
	won, err = store.SendOnce(ctx, visitID, "LAB:1", "line")
	if err != nil || won {
		t.Fatalf("second SendOnce: %v %v, want false nil — exactly once per step", won, err)
	}
	// A different step is a different slot.
	won, err = store.SendOnce(ctx, visitID, "CLINIC:MED:2", "line")
	if err != nil || !won {
		t.Fatalf("SendOnce other step: %v %v, want true nil", won, err)
	}

	// Opt-out round trip, and the FK's journey-shaped 404.
	if err := store.SetEnabled(ctx, visitID, false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if enabled, err := store.Enabled(ctx, visitID); err != nil || enabled {
		t.Fatalf("Enabled after opt-out: %v %v, want false nil", enabled, err)
	}
	if err := store.SetEnabled(ctx, "VISIT-NEVER-PROJECTED", true); apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("SetEnabled unknown visit: err = %v, want KindNotFound", err)
	}

	// No claims at all → no recipient.
	if _, ok, err := recipients.LineUser(ctx, visitID); err != nil || ok {
		t.Fatalf("LineUser before claim: %v %v, want false nil", ok, err)
	}
	// A demo claim is not a LINE recipient.
	if _, err := database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.patient_visit_claim (source, external_id, visit_id)
		 VALUES ('demo', 'demo-user', $1)`, visitID,
	); err != nil {
		t.Fatalf("seed demo claim: %v", err)
	}
	if _, ok, err := recipients.LineUser(ctx, visitID); err != nil || ok {
		t.Fatalf("LineUser with only a demo claim: %v %v, want false nil", ok, err)
	}
	// The latest line-source claim wins.
	for i, external := range []string{"U-OLD", "U-NEW"} {
		if _, err := database.Querier(ctx).Exec(ctx,
			`INSERT INTO carepath.patient_visit_claim (source, external_id, visit_id, claimed_at)
			 VALUES ('line', $2, $1, now() + make_interval(secs => $3))`, visitID, external, i*60,
		); err != nil {
			t.Fatalf("seed line claim %s: %v", external, err)
		}
	}
	user, ok, err := recipients.LineUser(ctx, visitID)
	if err != nil || !ok || user != "U-NEW" {
		t.Fatalf("LineUser: %v %v %v, want U-NEW true nil", user, ok, err)
	}
}
