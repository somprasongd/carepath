// Integration test against a real Postgres: the seeded hospital map joined
// to the seeded service points, exactly as the #24 acceptance criteria read
// the "mapping covers Registration, OPD, X-Ray, Pharmacy" checklist.
// Requires the schema and seed data from infra/postgres/migrations; run
// `make migrate-up` first.
package servicepoint_test

import (
	"context"
	"os"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
	hospitalmappostgres "carepath/apps/api/internal/hospitalmap/postgres"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/servicepoint"
	servicepointpostgres "carepath/apps/api/internal/servicepoint/postgres"
)

func newService(t *testing.T) servicepoint.Service {
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
	places := hospitalmap.NewService(hospitalmappostgres.New(database))
	return servicepoint.NewService(servicepointpostgres.New(database), places)
}

// The four demo mappings of #23/#24: every Happy Path service resolves to a
// place that knows its floor and SVG position.
func TestSeededMappingsResolveFloorAndPosition(t *testing.T) {
	svc := newService(t)

	tests := []struct {
		code          string
		wantPlaceID   string
		wantFloorID   string
		wantEntryNode string
	}{
		{code: "REGISTRATION", wantPlaceID: "REG-01", wantFloorID: "I-1301", wantEntryNode: "I-1301/node-reception"},
		{code: "DOCTOR", wantPlaceID: "OPD-NS-01", wantFloorID: "I-1301", wantEntryNode: "I-1301/node-opd-ns"},
		{code: "XRAY", wantPlaceID: "XRAY-01", wantFloorID: "I-1301", wantEntryNode: "I-1301/node-xray"},
		{code: "PHARMACY", wantPlaceID: "PHARMACY-01", wantFloorID: "I-1301", wantEntryNode: "I-1301/node-pharmacy"},
		{code: "LAB", wantPlaceID: "LAB-01", wantFloorID: "I-1302", wantEntryNode: "I-1302/node-blood-collection"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			sp, err := svc.GetByCode(context.Background(), tt.code)
			if err != nil {
				t.Fatalf("GetByCode(%s): %v", tt.code, err)
			}
			if sp.Place == nil {
				t.Fatalf("GetByCode(%s) place = nil, want %s", tt.code, tt.wantPlaceID)
			}
			if sp.Place.ID != tt.wantPlaceID || sp.Place.FloorID != tt.wantFloorID {
				t.Fatalf("place = %s on %s, want %s on %s", sp.Place.ID, sp.Place.FloorID, tt.wantPlaceID, tt.wantFloorID)
			}
			if sp.Place.Floor == nil || sp.Place.Floor.ID != tt.wantFloorID {
				t.Fatalf("floor = %+v, want %s", sp.Place.Floor, tt.wantFloorID)
			}
			if sp.Place.X == nil || sp.Place.Y == nil {
				t.Fatalf("place %s has no SVG position", sp.Place.ID)
			}
			if sp.Place.EntryNodeID == nil || *sp.Place.EntryNodeID != tt.wantEntryNode {
				t.Fatalf("entry node = %v, want %s", sp.Place.EntryNodeID, tt.wantEntryNode)
			}
		})
	}
}
