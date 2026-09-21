package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
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

func TestGetPlace(t *testing.T) {
	repo := New(newDB(t))

	place, err := repo.GetPlace(context.Background(), "REG-01")
	if err != nil {
		t.Fatalf("GetPlace(REG-01): %v", err)
	}
	if place.FloorID != "I-1301" || place.Floor == nil || place.Floor.Name != "Ground Floor" {
		t.Fatalf("GetPlace(REG-01) = %+v, want I-1301 Ground Floor", place)
	}
	// The contract has declared Floor.viewBox since #105; this asserts the
	// query actually selects it rather than always scanning nil.
	if place.Floor.ViewBox == nil || *place.Floor.ViewBox != "0 0 1600 900" {
		t.Fatalf("GetPlace(REG-01).Floor.ViewBox = %v, want \"0 0 1600 900\"", place.Floor.ViewBox)
	}
	if place.X == nil || *place.X != 150 || place.Y == nil || *place.Y != 190 {
		t.Fatalf("GetPlace(REG-01) coordinates = (%v, %v), want (150, 190)", place.X, place.Y)
	}
	if place.EntryNodeID == nil || *place.EntryNodeID != "I-1301/node-reception" {
		t.Fatalf("GetPlace(REG-01) entry node = %v, want I-1301/node-reception", place.EntryNodeID)
	}

	// The lab is the blood-collection room on the upper floor (#23).
	lab, err := repo.GetPlace(context.Background(), "LAB-01")
	if err != nil {
		t.Fatalf("GetPlace(LAB-01): %v", err)
	}
	if lab.FloorID != "I-1302" || lab.Floor == nil || lab.Floor.LevelOrder != 2 {
		t.Fatalf("GetPlace(LAB-01) = %+v, want I-1302 (level 2)", lab)
	}

	if _, err := repo.GetPlace(context.Background(), "NOPE-01"); !errors.Is(err, hospitalmap.ErrPlaceNotFound) {
		t.Fatalf("error = %v, want hospitalmap.ErrPlaceNotFound", err)
	}
}

func TestListPlaces(t *testing.T) {
	repo := New(newDB(t))

	places, err := repo.ListPlaces(context.Background())
	if err != nil {
		t.Fatalf("ListPlaces: %v", err)
	}
	if len(places) < 5 {
		t.Fatalf("ListPlaces returned %d places, want at least the 5 seeded rows", len(places))
	}
	for _, place := range places {
		if place.Floor == nil {
			t.Fatalf("place %s has no floor resolved", place.ID)
		}
	}
}
