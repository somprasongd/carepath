package postgres

import (
	"context"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
	hospitalmappostgres "carepath/apps/api/internal/hospitalmap/postgres"
	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/location/qr"
	"carepath/apps/api/internal/navigation"
	navigationpostgres "carepath/apps/api/internal/navigation/postgres"
)

// AC #3 of #32, end to end over the seeded schema: a scanned QR fix becomes
// the visit's current location, and that location's node is a valid start
// for the navigation module's routing — the whole point of the canonical
// fix. REG-01 (reception) and the pharmacy node come from migrations
// 000007/000008.
func TestCurrentLocationFeedsRouting(t *testing.T) {
	database := newDB(t)
	ctx := context.Background()

	places := hospitalmap.NewService(hospitalmappostgres.New(database))
	// The location module only uses the graph to validate fixes against
	// nodes, so the servicepoint side of the routing service (#28) is not
	// wired here.
	graph := navigation.NewService(navigationpostgres.New(database), nil, nil)
	svc, err := location.NewService(New(database), graph, qr.New(places, graph))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	visitID := "VISIT-ROUTE-TEST"
	t.Cleanup(func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.location_observation WHERE visit_id = $1`, visitID)
	})

	obs, err := svc.Report(ctx, visitID, location.SourceQR,
		"https://carepath.example/location/REG-01")
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if obs.NodeID != "I-1301/node-reception" || obs.FloorID != "I-1301" {
		t.Fatalf("obs = %+v, want the REG-01 entry node on I-1301", obs)
	}

	current, err := svc.Current(ctx, visitID)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}

	route, err := graph.Route(ctx, current.NodeID, "I-1301/node-pharmacy", navigation.RouteOptions{})
	if err != nil {
		t.Fatalf("Route from the current location: %v", err)
	}
	if len(route.Nodes) < 2 || route.Nodes[0].ID != "I-1301/node-reception" ||
		route.Nodes[len(route.Nodes)-1].ID != "I-1301/node-pharmacy" {
		t.Fatalf("route = %+v, want reception → pharmacy", route.Nodes)
	}
}
