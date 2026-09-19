package postgres

import (
	"context"
	"os"
	"testing"

	"carepath/apps/api/internal/platform/db"
)

// Integration test against a real Postgres; run `make migrate-up` first.
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

func TestCheckpointRoundtrip(t *testing.T) {
	checkpoint := New(newDB(t))
	ctx := context.Background()

	if err := checkpoint.Save(ctx, "TEST-EVT-CURSOR"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Cleanup(func() {
		_ = checkpoint.Save(ctx, "")
	})

	got, err := checkpoint.LastEventID(ctx)
	if err != nil {
		t.Fatalf("LastEventID after save: %v", err)
	}
	if got != "TEST-EVT-CURSOR" {
		t.Fatalf("cursor = %q, want TEST-EVT-CURSOR", got)
	}
}
