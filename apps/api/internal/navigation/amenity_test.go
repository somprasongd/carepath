package navigation

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
)

func TestRankAmenitiesSortsNearestFirstAndDropsUnroutable(t *testing.T) {
	near, mid, far, unreachable := "n1", "n2", "n3", "n5"
	places := []hospitalmap.Place{
		{ID: "FAR", PlaceType: "RESTROOM", EntryNodeID: &far},
		{ID: "NEAR", PlaceType: "WAITING_AREA", EntryNodeID: &near},
		{ID: "MID", PlaceType: "FOOD_STALL", EntryNodeID: &mid},
		{ID: "NO-ENTRY", PlaceType: "RESTROOM"},                              // place without a graph anchor
		{ID: "UNREACHABLE", PlaceType: "AMENITY", EntryNodeID: &unreachable}, // off the reachable component
	}
	distances := map[string]float64{far: 900, near: 100, mid: 500}

	ranked := rankAmenities(places, distances)

	if len(ranked) != 3 {
		t.Fatalf("ranked = %+v, want 3 entries (entry-less and unreachable dropped)", ranked)
	}
	wantOrder := []string{"NEAR", "MID", "FAR"}
	for i, want := range wantOrder {
		if ranked[i].Place.ID != want {
			t.Fatalf("ranked[%d] = %s, want %s", i, ranked[i].Place.ID, want)
		}
	}
	if ranked[0].Distance != 100 {
		t.Fatalf("ranked[0].Distance = %v, want 100", ranked[0].Distance)
	}
}

func TestRankAmenitiesTiesFallBackToPlaceID(t *testing.T) {
	na, nb := "na", "nb"
	places := []hospitalmap.Place{
		{ID: "B", PlaceType: "RESTROOM", EntryNodeID: &nb},
		{ID: "A", PlaceType: "RESTROOM", EntryNodeID: &na},
	}
	distances := map[string]float64{na: 100, nb: 100}

	ranked := rankAmenities(places, distances)

	if !(ranked[0].Place.ID == "A" && ranked[1].Place.ID == "B") {
		t.Fatalf("ranked = [%s, %s], want stable place-id order [A, B]", ranked[0].Place.ID, ranked[1].Place.ID)
	}
}

// fakePlaces answers hospitalmap lookups from a fixed list, the way the
// hospitalmap service answers from the place table.
type fakePlaces struct {
	places []hospitalmap.Place
}

func (f *fakePlaces) GetPlace(_ context.Context, placeID string) (hospitalmap.Place, error) {
	for _, place := range f.places {
		if place.ID == placeID {
			return place, nil
		}
	}
	return hospitalmap.Place{}, hospitalmap.ErrPlaceNotFound
}

func (f *fakePlaces) ListPlaces(context.Context) ([]hospitalmap.Place, error) { return f.places, nil }

func (f *fakePlaces) ListFloors(context.Context) ([]hospitalmap.Floor, error) { return nil, nil }

func (f *fakePlaces) GetFloor(context.Context, string) (hospitalmap.Floor, error) {
	return hospitalmap.Floor{}, hospitalmap.ErrFloorNotFound
}

func (f *fakePlaces) SetPlan(context.Context, string, string, string) error { return nil }

func amenityTestPlaces() []hospitalmap.Place {
	corridor, pharmacy := "I-1301/node-corridor", "I-1301/node-pharmacy"
	return []hospitalmap.Place{
		{ID: "REG-01", FloorID: "I-1301", Name: "Reception", PlaceType: "COUNTER", EntryNodeID: &corridor},
		{ID: "RESTROOM-01", FloorID: "I-1301", Name: "Restroom", PlaceType: "RESTROOM", EntryNodeID: &corridor},
		{ID: "WAITING-01", FloorID: "I-1301", Name: "Waiting", PlaceType: "WAITING_AREA", EntryNodeID: &pharmacy},
		{ID: "FOOD-NO-ENTRY", FloorID: "I-1301", Name: "Food", PlaceType: "FOOD_STALL"},
	}
}

// #109 AC: the search ranks amenity-typed places by walking distance from
// one origin and silently drops what it cannot reach — non-amenity places,
// entry-less places and unreachable nodes never appear, and none of them is
// an error.
func TestNearestAmenitiesRanksAmenityPlacesOnly(t *testing.T) {
	svc := NewService(routeTestGraph(), nil, &fakePlaces{places: amenityTestPlaces()})

	result, err := svc.NearestAmenities(context.Background(), "I-1301/node-reception", RouteOptions{}, 0)
	if err != nil {
		t.Fatalf("NearestAmenities: %v", err)
	}
	if result.From != "I-1301/node-reception" {
		t.Fatalf("From = %s", result.From)
	}
	if len(result.Amenities) != 2 {
		t.Fatalf("amenities = %+v, want RESTROOM-01 (350) then WAITING-01 (735)", result.Amenities)
	}
	first, second := result.Amenities[0], result.Amenities[1]
	if first.Place.ID != "RESTROOM-01" || first.Distance != 350 {
		t.Fatalf("nearest = %+v, want RESTROOM-01 at 350", first)
	}
	if second.Place.ID != "WAITING-01" || second.Distance != 735 {
		t.Fatalf("second = %+v, want WAITING-01 at 735", second)
	}
}

func TestNearestAmenitiesUnknownOriginIsNotFound(t *testing.T) {
	svc := NewService(routeTestGraph(), nil, &fakePlaces{places: amenityTestPlaces()})

	if _, err := svc.NearestAmenities(context.Background(), "I-1301/node-nope", RouteOptions{}, 0); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("error = %v, want ErrNodeNotFound", err)
	}
}

func TestNearestAmenitiesLimitTruncates(t *testing.T) {
	svc := NewService(routeTestGraph(), nil, &fakePlaces{places: amenityTestPlaces()})

	result, err := svc.NearestAmenities(context.Background(), "I-1301/node-reception", RouteOptions{}, 1)
	if err != nil {
		t.Fatalf("NearestAmenities: %v", err)
	}
	if len(result.Amenities) != 1 || result.Amenities[0].Place.ID != "RESTROOM-01" {
		t.Fatalf("amenities = %+v, want only RESTROOM-01", result.Amenities)
	}
}

// #109 AC: a place destination routes exactly like a service point — the
// route ends at the place's entry node.
func TestRouteToPlaceEndsAtEntryNode(t *testing.T) {
	places := amenityTestPlaces()
	svc := NewService(routeTestGraph(), nil, &fakePlaces{places: places})

	route, err := svc.RouteToPlace(context.Background(), "I-1301/node-reception", "RESTROOM-01", RouteOptions{})
	if err != nil {
		t.Fatalf("RouteToPlace: %v", err)
	}
	if last := route.Nodes[len(route.Nodes)-1]; last.ID != "I-1301/node-corridor" {
		t.Fatalf("route ends at %s, want I-1301/node-corridor", last.ID)
	}
}

func TestRouteToPlaceUnknownPlaceKeepsNotFound(t *testing.T) {
	svc := NewService(routeTestGraph(), nil, &fakePlaces{places: amenityTestPlaces()})

	if _, err := svc.RouteToPlace(context.Background(), "I-1301/node-reception", "NOPE-01", RouteOptions{}); !errors.Is(err, hospitalmap.ErrPlaceNotFound) {
		t.Fatalf("error = %v, want hospitalmap.ErrPlaceNotFound", err)
	}
}

func TestRouteToPlaceWithoutEntryNodeIsUnmapped(t *testing.T) {
	svc := NewService(routeTestGraph(), nil, &fakePlaces{places: amenityTestPlaces()})

	if _, err := svc.RouteToPlace(context.Background(), "I-1301/node-reception", "FOOD-NO-ENTRY", RouteOptions{}); !errors.Is(err, ErrDestinationUnmapped) {
		t.Fatalf("error = %v, want ErrDestinationUnmapped", err)
	}
}
