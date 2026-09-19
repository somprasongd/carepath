package servicepoint

import (
	"context"
	"errors"
	"testing"
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
	return nil, nil
}

func TestGetByCodeMapsServiceCodeToServicePoint(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"})

	sp, err := svc.GetByCode(context.Background(), "LAB")
	if err != nil {
		t.Fatalf("GetByCode(LAB): %v", err)
	}
	if sp.ID != "SP-LAB" || sp.Code != "LAB" || sp.PlaceID != "LAB-01" {
		t.Fatalf("service point = %+v, want SP-LAB / LAB / LAB-01", sp)
	}
}

func TestGetByCodeUnknownCodePropagatesNotFound(t *testing.T) {
	svc := NewService(&fakeRepo{code: "LAB"})

	if _, err := svc.GetByCode(context.Background(), "RADIOLOGY"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
