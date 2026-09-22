// Nearby amenities (#109 / FR-26) against the real migrated database: the
// amenity places migration 000025 seeds, the actual nav graph from 000008,
// and the Dijkstra settle ranking them. No HTTP here — the handler layer is
// covered by the navigation handler tests; this suite proves the seeds and
// the graph produce a real, correctly-ranked answer.
package e2e_test

import (
	"context"
	"os"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
	hospitalmappostgres "carepath/apps/api/internal/hospitalmap/postgres"
	"carepath/apps/api/internal/navigation"
	navigationpostgres "carepath/apps/api/internal/navigation/postgres"
	"carepath/apps/api/internal/platform/db"
)

func newAmenityStack(t *testing.T) (navigation.Service, *db.DB) {
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
	places := hospitalmap.NewService(hospitalmappostgres.New(database))
	// No service points needed: amenity search resolves places directly.
	return navigation.NewService(navigationpostgres.New(database), nil, places), database
}

// AC: from a ground-floor origin every seeded amenity answers — the two
// ground-floor ones plus the upper-floor restroom through the lift — ranked
// by walking distance, nearest first, all floors reachable.
func TestNearestAmenitiesAgainstSeededGraph(t *testing.T) {
	nav, _ := newAmenityStack(t)

	result, err := nav.NearestAmenities(context.Background(), "I-1301/node-reception", navigation.RouteOptions{}, 0)
	if err != nil {
		t.Fatalf("NearestAmenities: %v", err)
	}

	byID := map[string]float64{}
	for _, amenity := range result.Amenities {
		byID[amenity.Place.ID] = amenity.Distance
	}
	for _, want := range []string{"RESTROOM-01", "FOOD-01", "WAITING-01", "RESTROOM-02"} {
		if _, ok := byID[want]; !ok {
			t.Fatalf("amenity %s missing from %v", want, byID)
		}
	}
	// Non-amenity places seeded by earlier migrations must not answer.
	for _, absent := range []string{"REG-01", "PHARMACY-01", "CASHIER-01"} {
		if _, ok := byID[absent]; ok {
			t.Fatalf("non-amenity place %s answered the search", absent)
		}
	}
	// Ascending by construction of the ranking…
	for i := 1; i < len(result.Amenities); i++ {
		if result.Amenities[i-1].Distance > result.Amenities[i].Distance {
			t.Fatalf("ranking not ascending: %v", result.Amenities)
		}
	}
	// …and the nearest amenity to the reception is on the reception's floor.
	if first := result.Amenities[0]; first.Place.FloorID != "I-1301" {
		t.Fatalf("nearest amenity = %s on %s, want a ground-floor one first", first.Place.ID, first.Place.FloorID)
	}
}

// AC: an amenity place routes exactly like a service point destination —
// the walk ends at the place's entry node, so the navigate screen can draw
// it with the same endpoint it already uses.
func TestRouteToAmenityPlaceAgainstSeededGraph(t *testing.T) {
	nav, _ := newAmenityStack(t)

	route, err := nav.RouteToPlace(context.Background(), "I-1301/node-reception", "RESTROOM-01", navigation.RouteOptions{})
	if err != nil {
		t.Fatalf("RouteToPlace: %v", err)
	}
	if last := route.Nodes[len(route.Nodes)-1]; last.ID != "I-1301/node-ramp" {
		t.Fatalf("route ends at %s, want I-1301/node-ramp (RESTROOM-01's entry)", last.ID)
	}
	if route.TotalDistance <= 0 {
		t.Fatalf("totalDistance = %v, want a real walk", route.TotalDistance)
	}
}
