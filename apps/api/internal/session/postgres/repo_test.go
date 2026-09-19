package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/session"
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

func cleanupToken(t *testing.T, database *db.DB, token string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.patient_session WHERE token = $1`, token)
	})
}

func TestCreateAndGet(t *testing.T) {
	database := newDB(t)
	repo := New(database)
	ctx := context.Background()

	sess := session.Session{
		Token:     "test-token-create-get",
		Identity:  identity.Identity{Source: "line", ExternalID: "U123", DisplayName: "Somchai"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	cleanupToken(t, database, sess.Token)

	if err := repo.Create(ctx, sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, sess.Token)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Identity != sess.Identity {
		t.Errorf("Identity = %+v, want %+v", got.Identity, sess.Identity)
	}
}

func TestGetNotFound(t *testing.T) {
	repo := New(newDB(t))

	if _, err := repo.Get(context.Background(), "does-not-exist"); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("error = %v, want session.ErrNotFound", err)
	}
}

func TestGetExpiredExcluded(t *testing.T) {
	database := newDB(t)
	repo := New(database)
	ctx := context.Background()

	sess := session.Session{
		Token:     "test-token-expired",
		Identity:  identity.Identity{Source: "demo", ExternalID: "demo-user"},
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	cleanupToken(t, database, sess.Token)

	if err := repo.Create(ctx, sess); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := repo.Get(ctx, sess.Token); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("error = %v, want session.ErrNotFound for expired row", err)
	}
}
