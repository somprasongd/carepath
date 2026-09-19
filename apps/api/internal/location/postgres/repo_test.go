package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/platform/db"
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

func TestRecordAndLatest(t *testing.T) {
	repo := New(newDB(t))
	ctx := context.Background()
	visitID := "VISIT-LOC-TEST"
	t.Cleanup(func() {
		_, _ = repo.database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.location_observation WHERE visit_id = $1`, visitID)
	})

	first := location.Observation{
		VisitID:    visitID,
		NodeID:     "I-1301/node-reception",
		FloorID:    "I-1301",
		Zone:       nil, // the service fills this from the node; repo stores whatever it gets
		Source:     location.SourceManual,
		ObservedAt: time.Now(),
	}
	if err := repo.Record(ctx, first); err != nil {
		t.Fatalf("Record first: %v", err)
	}

	second := location.Observation{
		VisitID:    visitID,
		NodeID:     "I-1301/node-pharmacy",
		FloorID:    "I-1301",
		Zone:       strPtr("PHARMACY"),
		Source:     location.SourceZigbee,
		Confidence: floatPtr(0.87),
		ObservedAt: time.Now(),
	}
	if err := repo.Record(ctx, second); err != nil {
		t.Fatalf("Record second: %v", err)
	}

	got, err := repo.Latest(ctx, visitID)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got.NodeID != second.NodeID || got.Source != location.SourceZigbee {
		t.Fatalf("Latest = %+v, want the second observation (%s, ZIGBEE)", got, second.NodeID)
	}
	if got.Zone == nil || *got.Zone != "PHARMACY" {
		t.Fatalf("Latest.Zone = %v, want PHARMACY", got.Zone)
	}
	if got.Confidence == nil || *got.Confidence != 0.87 {
		t.Fatalf("Latest.Confidence = %v, want 0.87", got.Confidence)
	}
	if got.VisitID != visitID {
		t.Fatalf("Latest.VisitID = %q, want %q", got.VisitID, visitID)
	}
}

func TestLatestWithoutObservations(t *testing.T) {
	repo := New(newDB(t))

	if _, err := repo.Latest(context.Background(), "VISIT-LOC-NONE"); !errors.Is(err, location.ErrNoLocation) {
		t.Fatalf("error = %v, want location.ErrNoLocation", err)
	}
}

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
