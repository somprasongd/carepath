package navigation_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"carepath/apps/api/internal/navigation"
	navigationpostgres "carepath/apps/api/internal/navigation/postgres"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

// Integration test of the routing service against a real Postgres. Requires
// the schema and seed data from infra/postgres/migrations; run `make
// migrate-up` first.
func newRoutingService(t *testing.T) navigation.Service {
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
	return navigation.NewService(navigationpostgres.New(database))
}

// The Happy Path demo journey's cross-floor leg — main entrance (ground) to
// blood collection (upper floor) — must come back as a walkable ordered path
// on the seeded graph.
func TestRouteAcrossFloorsOnSeed(t *testing.T) {
	svc := newRoutingService(t)
	ctx := context.Background()

	route, err := svc.Route(ctx, "I-1301/node-main-entrance", "I-1302/node-blood-collection", navigation.RouteOptions{})
	if err != nil {
		t.Fatalf("Route: %v", err)
	}

	if len(route.Nodes) == 0 || route.Nodes[0].ID != "I-1301/node-main-entrance" {
		t.Fatalf("route starts at %+v, want the main entrance", route.Nodes)
	}
	if route.Nodes[len(route.Nodes)-1].ID != "I-1302/node-blood-collection" {
		t.Fatalf("route ends at %+v, want blood collection", route.Nodes[len(route.Nodes)-1])
	}

	sum := 0.0
	for i, segment := range route.Segments {
		if segment.FromNodeID != route.Nodes[i].ID || segment.ToNodeID != route.Nodes[i+1].ID {
			t.Fatalf("segment %d = %s→%s, want %s→%s",
				i, segment.FromNodeID, segment.ToNodeID, route.Nodes[i].ID, route.Nodes[i+1].ID)
		}
		sum += segment.Distance
	}
	if len(route.Segments) != len(route.Nodes)-1 {
		t.Fatalf("%d segments for %d nodes, want len(nodes)-1", len(route.Segments), len(route.Nodes))
	}
	if route.TotalDistance != sum {
		t.Fatalf("total distance %v != sum of segments %v", route.TotalDistance, sum)
	}

	sawElevator := false
	for _, segment := range route.Segments {
		if segment.EdgeType == "ELEVATOR" {
			sawElevator = true
		}
	}
	if !sawElevator {
		t.Fatalf("cross-floor route never takes the elevator: %v", route)
	}
}

// A wheelchair route to the upper floor must exist and must not use the
// stairs transition the graph authors flagged not accessible.
func TestRouteAccessibleOnlyOnSeedAvoidsStairs(t *testing.T) {
	svc := newRoutingService(t)

	route, err := svc.Route(context.Background(), "I-1301/node-main-entrance", "I-1302/node-blood-collection", navigation.RouteOptions{AccessibleOnly: true})
	if err != nil {
		t.Fatalf("Route accessible-only: %v", err)
	}
	for _, segment := range route.Segments {
		if segment.EdgeType == "STAIRS" || !segment.Accessible {
			t.Fatalf("accessible route walks %v (%s), want accessible segments only", segment.ID, segment.EdgeType)
		}
	}
}

func TestRouteUnknownNodeOnSeed(t *testing.T) {
	svc := newRoutingService(t)

	if _, err := svc.Route(context.Background(), "I-1301/node-main-entrance", "I-1301/node-nope", navigation.RouteOptions{}); !errors.Is(err, navigation.ErrNodeNotFound) {
		t.Fatalf("error = %v (%s), want navigation.ErrNodeNotFound", err, apperr.KindOf(err))
	}
}
