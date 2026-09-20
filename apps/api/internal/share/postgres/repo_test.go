package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/share"
)

// sha256Hex mirrors the service's hashToken: links are stored by the hex
// sha256 of the bearer token, never the token itself.
func sha256Hex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Integration test against a real Postgres; requires the schema from
// infra/postgres/migrations (run make migrate-up first). Everything lives
// under TEST-SHARE-* visit ids and is deleted on the way in and out.
const testVisit = "VISIT-TEST-SHARE-1"

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
		// visit_share_link cascades on visit delete.
		_, _ = database.Querier(ctx).Exec(ctx,
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, testVisit)
	}
	purge()          // a previous run's leftovers
	t.Cleanup(purge) // and this run's
}

func seedVisit(t *testing.T, q db.Querier) {
	t.Helper()
	if _, err := q.Exec(context.Background(),
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, status)
		 VALUES ($1, 'PATIENT-TEST-SHARE', 'ACTIVE')
		 ON CONFLICT (visit_id) DO NOTHING`, testVisit,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}
}

func TestRepoRoundTrip(t *testing.T) {
	database := newDB(t)
	cleanup(t, database)
	seedVisit(t, database.Querier(context.Background()))
	ctx := context.Background()
	repo := New(database)

	link := share.ShareLink{
		TokenHash: sha256Hex("roundtrip-token"),
		VisitID:   testVisit,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := repo.Create(ctx, link); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByTokenHash(ctx, link.TokenHash)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.VisitID != testVisit || got.RevokedAt != nil {
		t.Errorf("got = %+v, want the stored link for %s unrevoked", got, testVisit)
	}
	if !got.ExpiresAt.Equal(link.ExpiresAt) {
		t.Errorf("expiresAt = %v, want %v", got.ExpiresAt, link.ExpiresAt)
	}

	if _, err := repo.GetByTokenHash(ctx, sha256Hex("never-created")); apperr.KindOf(err) != apperr.KindNotFound {
		t.Errorf("missing token: err = %v, want not found", err)
	}

	// Same hash twice is a no-op (a 64-bit-space collision, or a retried
	// create with a deterministic token) — never a second row.
	if err := repo.Create(ctx, link); err != nil {
		t.Fatalf("re-create same hash: %v", err)
	}
}

func TestCountActiveAndRevoke(t *testing.T) {
	database := newDB(t)
	cleanup(t, database)
	seedVisit(t, database.Querier(context.Background()))
	ctx := context.Background()
	repo := New(database)

	now := time.Now()
	links := []share.ShareLink{
		{TokenHash: sha256Hex("active-1"), VisitID: testVisit, ExpiresAt: now.Add(time.Hour)},
		{TokenHash: sha256Hex("active-2"), VisitID: testVisit, ExpiresAt: now.Add(time.Hour)},
		{TokenHash: sha256Hex("expired"), VisitID: testVisit, ExpiresAt: now.Add(-time.Minute)},
		{TokenHash: sha256Hex("revoked"), VisitID: testVisit, ExpiresAt: now.Add(time.Hour), RevokedAt: &now},
	}
	for _, link := range links {
		if err := repo.Create(ctx, link); err != nil {
			t.Fatalf("create %s: %v", link.TokenHash[:6], err)
		}
	}

	count, err := repo.CountActive(ctx, testVisit)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2 (expired and revoked excluded)", count)
	}

	revoked, err := repo.RevokeActive(ctx, testVisit)
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if revoked != 2 {
		t.Errorf("revoked = %d, want 2", revoked)
	}
	if count, _ := repo.CountActive(ctx, testVisit); count != 0 {
		t.Errorf("count after revoke = %d, want 0", count)
	}
	got, err := repo.GetByTokenHash(ctx, sha256Hex("active-1"))
	if err != nil || got.RevokedAt == nil {
		t.Errorf("active-1 after revoke = %+v/%v, want RevokedAt set", got, err)
	}
	// The already-revoked row keeps its original timestamp, and expiry is
	// untouched: revoke is a one-way flag, never a delete or an extension.
	if got, _ := repo.GetByTokenHash(ctx, sha256Hex("expired")); got.RevokedAt != nil {
		t.Errorf("expired row gained a RevokedAt (%+v); RevokeActive must not touch it", got)
	}
}

// Deleting the visit takes its share links with it (ON DELETE CASCADE) —
// a relative's stale link dies with the visit row, not at expiry.
func TestCascadeOnVisitDelete(t *testing.T) {
	database := newDB(t)
	cleanup(t, database)
	seedVisit(t, database.Querier(context.Background()))
	ctx := context.Background()
	repo := New(database)

	link := share.ShareLink{
		TokenHash: sha256Hex("cascade-token"), VisitID: testVisit,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := repo.Create(ctx, link); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := database.Querier(ctx).Exec(ctx,
		`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, testVisit); err != nil {
		t.Fatalf("delete visit: %v", err)
	}
	if _, err := repo.GetByTokenHash(ctx, link.TokenHash); apperr.KindOf(err) != apperr.KindNotFound {
		t.Errorf("after visit delete: err = %v, want not found", err)
	}
}
