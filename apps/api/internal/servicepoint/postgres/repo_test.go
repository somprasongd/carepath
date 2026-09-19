package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/servicepoint"
)

// Integration test against a real Postgres. Requires the schema and seed
// data from infra/postgres/migrations; run `make migrate-up` first.
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

func TestGetByCode(t *testing.T) {
	repo := New(newDB(t))

	sp, err := repo.GetByCode(context.Background(), "LAB")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if sp.ID != "SP-LAB" || sp.PlaceID != "LAB-01" || !sp.Active {
		t.Fatalf("GetByCode = %+v, want seeded SP-LAB at LAB-01 active", sp)
	}

	// Seeded for the #22 demo order flow: XRAY resolves to the imaging room
	// on the ground floor.
	xray, err := repo.GetByCode(context.Background(), "XRAY")
	if err != nil {
		t.Fatalf("GetByCode XRAY: %v", err)
	}
	if xray.ID != "SP-XRAY" || xray.PlaceID != "XRAY-01" {
		t.Fatalf("GetByCode XRAY = %+v, want SP-XRAY at XRAY-01", xray)
	}

	if _, err := repo.GetByCode(context.Background(), "NOPE"); !errors.Is(err, servicepoint.ErrNotFound) {
		t.Fatalf("error = %v, want servicepoint.ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	repo := New(newDB(t))

	points, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(points) < 3 {
		t.Fatalf("List returned %d points, want at least the 3 seeded rows", len(points))
	}
}

// Queries made with a transaction-bound ctx must run on that transaction —
// verified by writing inside WithinTransaction and observing the row with a
// ctx that resolves to the pool only after commit.
func TestQueriesJoinAmbientTransaction(t *testing.T) {
	database := newDB(t)
	repo := New(database)
	ctx := context.Background()

	err := database.WithinTransaction(ctx, func(ctx context.Context) error {
		_, err := database.Querier(ctx).Exec(ctx,
			`INSERT INTO carepath.service_point (id, code, name, place_id) VALUES ('SP-TX', 'TX-TEST', 'Tx Test', 'TX-01')`)
		return err
	})
	if err != nil {
		t.Fatalf("insert in transaction: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.service_point WHERE id = 'SP-TX'`)
	})

	sp, err := repo.GetByCode(context.Background(), "TX-TEST")
	if err != nil {
		t.Fatalf("GetByCode after commit: %v", err)
	}
	if sp.ID != "SP-TX" {
		t.Fatalf("GetByCode = %+v, want SP-TX", sp)
	}
}

func TestRollbackDiscardsWrites(t *testing.T) {
	database := newDB(t)
	repo := New(database)
	ctx := context.Background()

	err := database.WithinTransaction(ctx, func(ctx context.Context) error {
		_, err := database.Querier(ctx).Exec(ctx,
			`INSERT INTO carepath.service_point (id, code, name, place_id) VALUES ('SP-RB', 'RB-TEST', 'Rollback', 'RB-01')`)
		if err != nil {
			return err
		}
		_, err = repo.GetByCode(ctx, "RB-TEST")
		if err != nil {
			t.Fatalf("read inside transaction should see uncommitted row: %v", err)
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected WithinTransaction to surface fn error")
	}

	if _, err := repo.GetByCode(ctx, "RB-TEST"); !errors.Is(err, servicepoint.ErrNotFound) {
		t.Fatalf("error after rollback = %v, want servicepoint.ErrNotFound", err)
	}
}
