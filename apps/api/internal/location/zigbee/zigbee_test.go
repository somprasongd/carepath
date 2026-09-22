package zigbee_test

import (
	"context"
	"testing"

	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/location/zigbee"
	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
)

// fakeGraph backs the provider with the zone layout of the seeded graph
// (packages/floorplans/graphs): PUBLIC holds an entrance, a corridor, and
// two place entries; PHARMACY a single place entry; ER an entrance and a
// corridor but no place entry.
type fakeGraph struct{}

var (
	zonePublic   = "PUBLIC"
	zonePharmacy = "PHARMACY"
	zoneER       = "ER"
)

var fakeNodes = []navigation.NavNode{
	{ID: "I-1301/node-main-entrance", FloorID: "I-1301", NodeType: "ENTRANCE", Zone: &zonePublic},
	{ID: "I-1301/node-ramp", FloorID: "I-1301", NodeType: "CORRIDOR", Zone: &zonePublic},
	{ID: "I-1301/node-cashier", FloorID: "I-1301", NodeType: "PLACE_ENTRY", Zone: &zonePublic},
	{ID: "I-1301/node-reception", FloorID: "I-1301", NodeType: "PLACE_ENTRY", Zone: &zonePublic},
	{ID: "I-1301/node-pharmacy", FloorID: "I-1301", NodeType: "PLACE_ENTRY", Zone: &zonePharmacy},
	{ID: "I-1301/node-er-entrance", FloorID: "I-1301", NodeType: "ENTRANCE", Zone: &zoneER},
	{ID: "I-1301/node-er-corridor", FloorID: "I-1301", NodeType: "CORRIDOR", Zone: &zoneER},
	{ID: "I-1302/node-lift", FloorID: "I-1302", NodeType: "ELEVATOR"},
}

func (f *fakeGraph) GetNode(_ context.Context, nodeID string) (navigation.NavNode, error) {
	for _, n := range fakeNodes {
		if n.ID == nodeID {
			return n, nil
		}
	}
	return navigation.NavNode{}, navigation.ErrNodeNotFound
}

func (f *fakeGraph) ListNodes(context.Context) ([]navigation.NavNode, error) { return fakeNodes, nil }
func (f *fakeGraph) ListEdges(context.Context) ([]navigation.NavEdge, error) { return nil, nil }

// Route is never called by the provider; it only satisfies the interface.
// Same for RouteToServicePoint (#28).
func (f *fakeGraph) Route(context.Context, string, string, navigation.RouteOptions) (navigation.Route, error) {
	return navigation.Route{}, nil
}

func (f *fakeGraph) RouteToServicePoint(context.Context, string, string, navigation.RouteOptions) (navigation.Route, error) {
	return navigation.Route{}, nil
}

func (f *fakeGraph) DistancesToServicePoints(context.Context, string, []string, navigation.RouteOptions) (map[string]float64, error) {
	return nil, navigation.ErrNoRoute
}

func (f *fakeGraph) RouteToPlace(context.Context, string, string, navigation.RouteOptions) (navigation.Route, error) {
	return navigation.Route{}, nil
}

func (f *fakeGraph) NearestAmenities(context.Context, string, navigation.RouteOptions, int) (navigation.AmenitySearch, error) {
	return navigation.AmenitySearch{}, nil
}

func newProvider() zigbee.Provider { return zigbee.New(&fakeGraph{}) }

// AC #2 of #33: a zone fix maps onto a node of the navigation graph,
// preferring the zone's destination point (first place entry by id) and
// carrying the zone and confidence metadata of a probabilistic fix.
func TestResolveZoneToRepresentativeNode(t *testing.T) {
	obs, err := newProvider().Resolve(context.Background(),
		`{"floorId":"I-1301","zone":"PUBLIC","confidence":0.8}`)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if obs.NodeID != "I-1301/node-cashier" {
		t.Fatalf("obs.NodeID = %q, want the first PLACE_ENTRY of PUBLIC by id", obs.NodeID)
	}
	if obs.Zone == nil || *obs.Zone != "PUBLIC" {
		t.Fatalf("obs.Zone = %v, want PUBLIC from the graph", obs.Zone)
	}
	if obs.Confidence == nil || *obs.Confidence != 0.8 {
		t.Fatalf("obs.Confidence = %v, want 0.8 from the fix", obs.Confidence)
	}
}

// A zone without place entries still resolves: entrance before corridor.
func TestResolveZoneWithoutPlaceEntry(t *testing.T) {
	obs, err := newProvider().Resolve(context.Background(), `{"floorId":"I-1301","zone":"ER"}`)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if obs.NodeID != "I-1301/node-er-entrance" {
		t.Fatalf("obs.NodeID = %q, want the ER entrance", obs.NodeID)
	}
	if obs.Confidence != nil {
		t.Fatalf("obs.Confidence = %v, want nil when the fix omits it", obs.Confidence)
	}
}

func TestResolveZoneCaseInsensitive(t *testing.T) {
	obs, err := newProvider().Resolve(context.Background(), `{"floorId":"I-1301","zone":" pharmacy "}`)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if obs.NodeID != "I-1301/node-pharmacy" {
		t.Fatalf("obs.NodeID = %q, want the pharmacy node", obs.NodeID)
	}
}

// The fix is the client's input, so everything wrong with it is an invalid
// fix (400-class), never an internal or not-found error.
func TestResolveInvalidFixes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"malformed json", `{"zone":`},
		{"not json at all", `tag-1234 in the lobby`},
		{"missing zone", `{"floorId":"I-1301"}`},
		{"missing floor", `{"zone":"PUBLIC"}`},
		{"empty fields", `{"floorId":" ","zone":"PUBLIC"}`},
		{"unknown zone", `{"floorId":"I-1301","zone":"ICU"}`},
		{"zone on the wrong floor", `{"floorId":"I-1302","zone":"PUBLIC"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newProvider().Resolve(context.Background(), tt.raw)
			if err != location.ErrInvalidFix {
				t.Fatalf("error = %v, want ErrInvalidFix", err)
			}
			if apperr.KindOf(err) != apperr.KindInvalid {
				t.Fatalf("kind = %v, want Invalid", apperr.KindOf(err))
			}
		})
	}
}
