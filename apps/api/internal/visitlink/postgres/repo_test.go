package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

func newDB(t *testing.T) *db.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (needs migrations applied — see CI)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	return database
}

// The token row's contract (#136): an unprojected visit has no link, a
// stored hash resolves back with the projection state the lifetime policy
// reads, and Store replaces (rotation) rather than accumulating versions.
func TestTokenRepo(t *testing.T) {
	database := newDB(t)
	repo := New(database)
	ctx := context.Background()

	const visitID = "VISIT-TEST-LINK-1"
	cleanup := func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, visitID)
	}
	cleanup()
	t.Cleanup(cleanup)

	// Current on an unprojected visit: exists=false — mint answers the
	// journey 404 from this, not from a foreign-key error.
	_, _, exists, err := repo.Current(ctx, visitID)
	if err != nil || exists {
		t.Fatalf("Current unprojected: %v %v, want false nil", exists, err)
	}

	seeded := time.Now().Add(-time.Hour)
	if _, err := database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, status, completed_at)
		 VALUES ($1, 'PATIENT-LINK-TEST', 'COMPLETED', $2)`, visitID, seeded,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}

	// No token minted yet: the visit exists, the row does not.
	hash, version, exists, err := repo.Current(ctx, visitID)
	if err != nil || !exists || hash != "" || version != 0 {
		t.Fatalf("Current before mint: %q %d %v %v", hash, version, exists, err)
	}

	if err := repo.Store(ctx, visitID, "hash-v1", 1); err != nil {
		t.Fatalf("Store: %v", err)
	}
	hash, version, exists, err = repo.Current(ctx, visitID)
	if err != nil || !exists || hash != "hash-v1" || version != 1 {
		t.Fatalf("Current after mint: %q %d %v %v", hash, version, exists, err)
	}

	// Resolve returns the projection state alongside the row.
	gotVisit, gotStatus, gotVersion, completedAt, err := repo.Resolve(ctx, "hash-v1")
	if err != nil || gotVisit != visitID || gotStatus != "COMPLETED" || gotVersion != 1 || completedAt == nil {
		t.Fatalf("Resolve: %q %q %d %v %v", gotVisit, gotStatus, gotVersion, completedAt, err)
	}

	// Rotation replaces the row — the previous hash no longer resolves.
	if err := repo.Store(ctx, visitID, "hash-v2", 2); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, _, _, _, err := repo.Resolve(ctx, "hash-v1"); apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("Resolve old hash: %v, want KindNotFound", err)
	}
	if _, _, _, _, err := repo.Resolve(ctx, "hash-unknown"); apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("Resolve unknown hash: %v, want KindNotFound", err)
	}
}
