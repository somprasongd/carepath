package hospitalmap

import (
	"context"
	"errors"
	"testing"
)

// fakeRepo mirrors the seed rows from infra/postgres/migrations/000007.
type fakeRepo struct {
	places map[string]Place
}

func (f *fakeRepo) GetPlace(_ context.Context, placeID string) (Place, error) {
	place, ok := f.places[placeID]
	if !ok {
		return Place{}, ErrPlaceNotFound
	}
	return place, nil
}

// ListFloors is unused by these cases; the service passes it straight
// through to the repo.
func (f *fakeRepo) ListFloors(context.Context) ([]Floor, error) { return nil, nil }

// The floor write path (#105) belongs to the floorplan module; nothing
// here exercises it.
func (f *fakeRepo) GetFloor(context.Context, string) (Floor, error) {
	return Floor{}, nil
}

func (f *fakeRepo) SetPlan(context.Context, string, string, string) error { return nil }

func (f *fakeRepo) ListPlaces(context.Context) ([]Place, error) {
	var places []Place
	for _, place := range f.places {
		places = append(places, place)
	}
	return places, nil
}

func TestGetPlaceResolvesFloor(t *testing.T) {
	svc := NewService(&fakeRepo{places: map[string]Place{
		"LAB-01": {
			ID: "LAB-01", FloorID: "I-1302", Name: "Blood Collection", PlaceType: "ROOM",
			Floor: &Floor{ID: "I-1302", BuildingID: "BLD-I13", Code: "2", Name: "Upper Floor", LevelOrder: 2},
		},
	}})

	place, err := svc.GetPlace(context.Background(), "LAB-01")
	if err != nil {
		t.Fatalf("GetPlace(LAB-01): %v", err)
	}
	if place.Floor == nil || place.Floor.ID != "I-1302" || place.Floor.LevelOrder != 2 {
		t.Fatalf("floor = %+v, want I-1302 (level 2)", place.Floor)
	}
}

func TestGetPlaceUnknownIDPropagatesNotFound(t *testing.T) {
	svc := NewService(&fakeRepo{places: map[string]Place{}})

	if _, err := svc.GetPlace(context.Background(), "NOPE-01"); !errors.Is(err, ErrPlaceNotFound) {
		t.Fatalf("error = %v, want ErrPlaceNotFound", err)
	}
}

func TestListPlaces(t *testing.T) {
	svc := NewService(&fakeRepo{places: map[string]Place{
		"REG-01": {ID: "REG-01", FloorID: "I-1301"},
		"LAB-01": {ID: "LAB-01", FloorID: "I-1302"},
	}})

	places, err := svc.ListPlaces(context.Background())
	if err != nil {
		t.Fatalf("ListPlaces: %v", err)
	}
	if len(places) != 2 {
		t.Fatalf("len(places) = %d, want 2", len(places))
	}
}
