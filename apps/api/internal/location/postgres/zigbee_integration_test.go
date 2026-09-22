package postgres

import (
	"context"
	"testing"

	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/location/zigbee"
	"carepath/apps/api/internal/navigation"
	navigationpostgres "carepath/apps/api/internal/navigation/postgres"
)

// AC #2/#3 of #33 over the seeded schema: a simulated Zigbee zone fix maps
// onto the zone's node of the navigation graph, becomes the visit's current
// location, and routing consumes it unchanged — here cross-floor, from
// WELLNESS on I-1302 down to the pharmacy on I-1301 through the lift edge
// of migrations 000007/000008.
func TestZigbeeZoneFixFeedsRouting(t *testing.T) {
	database := newDB(t)
	ctx := context.Background()

	// The provider only validates fixes against graph nodes, so the
	// servicepoint side of the routing service (#28) is not wired here.
	graph := navigation.NewService(navigationpostgres.New(database), nil, nil)
	svc, err := location.NewService(New(database), graph, zigbee.New(graph))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	visitID := "VISIT-ZIGBEE-TEST"
	t.Cleanup(func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.location_observation WHERE visit_id = $1`, visitID)
	})

	obs, err := svc.Report(ctx, visitID, location.SourceZigbee,
		`{"floorId":"I-1302","zone":"WELLNESS","confidence":0.85}`)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if obs.NodeID != "I-1302/node-blood-collection" || obs.FloorID != "I-1302" {
		t.Fatalf("obs = %+v, want the WELLNESS representative node on I-1302", obs)
	}
	if obs.Zone == nil || *obs.Zone != "WELLNESS" {
		t.Fatalf("obs.Zone = %v, want WELLNESS", obs.Zone)
	}
	if obs.Confidence == nil || *obs.Confidence != 0.85 {
		t.Fatalf("obs.Confidence = %v, want 0.85", obs.Confidence)
	}

	current, err := svc.Current(ctx, visitID)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if current.NodeID != obs.NodeID || current.Source != location.SourceZigbee {
		t.Fatalf("Current = %+v, want the reported Zigbee observation", current)
	}

	route, err := graph.Route(ctx, current.NodeID, "I-1301/node-pharmacy", navigation.RouteOptions{})
	if err != nil {
		t.Fatalf("Route from the zone fix: %v", err)
	}
	if len(route.Nodes) < 2 || route.Nodes[0].ID != "I-1302/node-blood-collection" ||
		route.Nodes[len(route.Nodes)-1].ID != "I-1301/node-pharmacy" {
		t.Fatalf("route nodes = %+v, want blood-collection → pharmacy", route.Nodes)
	}
	if route.Nodes[0].FloorID == route.Nodes[len(route.Nodes)-1].FloorID {
		t.Fatalf("route stays on %s; want a cross-floor path", route.Nodes[0].FloorID)
	}
}
