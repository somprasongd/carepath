package servicepoint

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/platform/apperr"
)

// fakeRepo mirrors the seed rows from infra/postgres/migrations/000002:
// the external HIS service code is the lookup key, the result carries the
// CarePath place (AC3 of #20: external service code → service point).
type fakeRepo struct {
	code string
}

func (f *fakeRepo) GetByCode(_ context.Context, code string) (ServicePoint, error) {
	if code != f.code {
		return ServicePoint{}, ErrNotFound
	}
	return ServicePoint{ID: "SP-LAB", Code: code, Name: "Laboratory", PlaceID: "LAB-01", Active: true}, nil
}

func (f *fakeRepo) List(context.Context) ([]ServicePoint, error) {
	return []ServicePoint{
		{ID: "SP-LAB", Code: "LAB", Name: "Laboratory", PlaceID: "LAB-01", Active: true},
		{ID: "SP-REG", Code: "REGISTRATION", Name: "Registration", PlaceID: "REG-01", Active: true},
	}, nil
}

// fakePlaces stands in for the hospitalmap module's Service.
type fakePlaces struct {
	places map[string]hospitalmap.Place
	err    error
}

func (f *fakePlaces) GetPlace(_ context.Context, placeID string) (hospitalmap.Place, error) {
	if f.err != nil {
		return hospitalmap.Place{}, f.err
	}
	place, ok := f.places[placeID]
	if !ok {
		return hospitalmap.Place{}, hospitalmap.ErrPlaceNotFound
	}
	return place, nil
}

func (f *fakePlaces) ListPlaces(context.Context) ([]hospitalmap.Place, error) {
	if f.err != nil {
		return nil, f.err
	}
	var places []hospitalmap.Place
	for _, place := range f.places {
		places = append(places, place)
	}
	return places, nil
}

func seedPlaces() map[string]hospitalmap.Place {
	return map[string]hospitalmap.Place{
		"REG-01": {
			ID: "REG-01", FloorID: "I-1301", Name: "Reception", PlaceType: "COUNTER",
			Floor: &hospitalmap.Floor{ID: "I-1301", BuildingID: "BLD-I13", Code: "1", Name: "Ground Floor", LevelOrder: 1},
		},
		"LAB-01": {
			ID: "LAB-01", FloorID: "I-1302", Name: "Blood Collection", PlaceType: "ROOM",
			Floor: &hospitalmap.Floor{ID: "I-1302", BuildingID: "BLD-I13", Code: "2", Name: "Upper Floor", LevelOrder: 2},
		},
	}
}

func TestGetByCodeMapsServiceCodeToServicePoint(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{places: seedPlaces()})

	sp, err := svc.GetByCode(context.Background(), "LAB")
	if err != nil {
		t.Fatalf("GetByCode(LAB): %v", err)
	}
	if sp.ID != "SP-LAB" || sp.Code != "LAB" || sp.PlaceID != "LAB-01" {
		t.Fatalf("service point = %+v, want SP-LAB / LAB / LAB-01", sp)
	}
}

func TestGetByCodeUnknownCodePropagatesNotFound(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{places: seedPlaces()})

	if _, err := svc.GetByCode(context.Background(), "RADIOLOGY"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestGetByCodeResolvesPlaceAndFloor(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{places: seedPlaces()})

	sp, err := svc.GetByCode(context.Background(), "LAB")
	if err != nil {
		t.Fatalf("GetByCode(LAB): %v", err)
	}
	if sp.Place == nil {
		t.Fatalf("place = nil, want resolved LAB-01")
	}
	if sp.Place.Floor == nil || sp.Place.Floor.ID != "I-1302" {
		t.Fatalf("floor = %+v, want I-1302", sp.Place.Floor)
	}
}

func TestGetByCodeMissingPlaceStaysUnmapped(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{})

	sp, err := svc.GetByCode(context.Background(), "LAB")
	if err != nil {
		t.Fatalf("GetByCode(LAB) with unmapped place: %v", err)
	}
	if sp.Place != nil {
		t.Fatalf("place = %+v, want nil (explicit unmapped state)", sp.Place)
	}
}

func TestGetByCodePlaceLookupFailureSurfaces(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{err: apperr.New(apperr.KindInternal, "boom")})

	if _, err := svc.GetByCode(context.Background(), "LAB"); apperr.KindOf(err) != apperr.KindInternal {
		t.Fatalf("error = %v, want internal", err)
	}
}

func TestListResolvesPlacesInBatch(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{places: seedPlaces()})

	points, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}
	for _, sp := range points {
		if sp.Place == nil || sp.Place.ID != sp.PlaceID || sp.Place.Floor == nil {
			t.Fatalf("service point %s = %+v, want place %s with floor", sp.Code, sp.Place, sp.PlaceID)
		}
	}
}

func TestListPlaceLookupFailureSurfaces(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"}, &fakePlaces{err: apperr.New(apperr.KindInternal, "boom")})

	if _, err := svc.List(context.Background()); apperr.KindOf(err) != apperr.KindInternal {
		t.Fatalf("error = %v, want internal", err)
	}
}
