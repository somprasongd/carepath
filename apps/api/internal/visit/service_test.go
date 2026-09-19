package visit

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/servicepoint"
)

type txMarker struct{}

// fakeTransactor mimics db.DB: it tags ctx as transactional and joins an
// already-tagged ctx instead of opening a second transaction.
type fakeTransactor struct {
	called bool
	joined bool
}

func (f *fakeTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	f.called = true
	if _, ok := ctx.Value(txMarker{}).(bool); ok {
		f.joined = true
		return fn(ctx)
	}
	return fn(context.WithValue(ctx, txMarker{}, true))
}

type fakeHIS struct {
	visit his.Visit
	err   error
}

func (f *fakeHIS) GetVisit(_ context.Context, _ string) (his.Visit, error) {
	return f.visit, f.err
}

func (f *fakeHIS) Events(_ context.Context, _ string, _ int) (his.EventPage, error) {
	return his.EventPage{}, nil
}

type fakeServicepoint struct {
	gotCtx context.Context
	code   string
}

func (f *fakeServicepoint) GetByCode(ctx context.Context, code string) (servicepoint.ServicePoint, error) {
	f.gotCtx = ctx
	f.code = code
	if code == "UNKNOWN" {
		return servicepoint.ServicePoint{}, servicepoint.ErrNotFound
	}
	return servicepoint.ServicePoint{ID: "SP-" + code, Code: code, Name: code, PlaceID: "PLACE-1", Active: true}, nil
}

func (f *fakeServicepoint) List(context.Context) ([]servicepoint.ServicePoint, error) {
	return nil, nil
}

func readyVisit() his.Visit {
	return his.Visit{
		VisitID:    "VISIT-001",
		PatientRef: "PAT-001",
		Status:     "IN_PROGRESS",
		Steps: []his.VisitStep{
			{Sequence: 1, ServiceCode: "REGISTRATION", Status: "COMPLETED"},
			{Sequence: 2, ServiceCode: "LAB", Status: "READY"},
			{Sequence: 3, ServiceCode: "PHARMACY", Status: "PENDING"},
		},
	}
}

// The core architectural property: a cross-module call made inside
// WithinTransaction runs with the transaction-bound ctx, so the other module
// joins the same transaction without knowing about it.
func TestServicepointCallJoinsTransaction(t *testing.T) {
	sp := &fakeServicepoint{}
	svc := NewService(&fakeHIS{visit: readyVisit()}, sp, &fakeTransactor{})

	if _, err := svc.GetVisitView(context.Background(), "VISIT-001"); err != nil {
		t.Fatalf("GetVisitView: %v", err)
	}
	if sp.gotCtx == nil {
		t.Fatal("servicepoint service was not called")
	}
	if _, inTx := sp.gotCtx.Value(txMarker{}).(bool); !inTx {
		t.Fatal("servicepoint call did not run inside the visit transaction")
	}
	if sp.code != "LAB" {
		t.Fatalf("servicepoint called with code %q, want first READY step code %q", sp.code, "LAB")
	}
}

func TestNestedTransactionJoinsOuter(t *testing.T) {
	tx := &fakeTransactor{}
	svc := NewService(&fakeHIS{visit: readyVisit()}, &fakeServicepoint{}, tx)

	inner := &fakeTransactor{}
	if err := inner.WithinTransaction(context.Background(), func(ctx context.Context) error {
		_, err := svc.GetVisitView(ctx, "VISIT-001")
		return err
	}); err != nil {
		t.Fatalf("nested GetVisitView: %v", err)
	}
	if !tx.joined {
		t.Fatal("inner WithinTransaction did not join the outer transaction ctx")
	}
}

func TestGetVisitView(t *testing.T) {
	tests := []struct {
		name       string
		visit      his.Visit
		hisErr     error
		wantNext   bool
		wantSPCode string
	}{
		{name: "resolves first READY step with service point", visit: readyVisit(), wantNext: true, wantSPCode: "LAB"},
		{
			name:     "unknown service code still returns the step",
			visit:    his.Visit{VisitID: "V", Steps: []his.VisitStep{{Sequence: 1, ServiceCode: "UNKNOWN", Status: "READY"}}},
			wantNext: true,
		},
		{name: "no READY step yields no next", visit: his.Visit{VisitID: "V", Steps: []his.VisitStep{{Sequence: 1, ServiceCode: "LAB", Status: "PENDING"}}}},
		{name: "HIS not found propagates", hisErr: his.ErrVisitNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeHIS{visit: tt.visit, err: tt.hisErr}, &fakeServicepoint{}, &fakeTransactor{})

			view, err := svc.GetVisitView(context.Background(), "VISIT-001")
			switch {
			case tt.hisErr != nil:
				if !errors.Is(err, tt.hisErr) {
					t.Fatalf("error = %v, want %v", err, tt.hisErr)
				}
				return
			case err != nil:
				t.Fatalf("GetVisitView: %v", err)
			}
			if (view.Next != nil) != tt.wantNext {
				t.Fatalf("view.Next = %+v, want present: %v", view.Next, tt.wantNext)
			}
			if tt.wantSPCode != "" && (view.Next == nil || view.Next.ServicePoint == nil || view.Next.ServicePoint.Code != tt.wantSPCode) {
				t.Fatalf("view.Next.ServicePoint = %+v, want code %q", view.Next, tt.wantSPCode)
			}
		})
	}
}

func TestGetNextStep(t *testing.T) {
	svc := NewService(&fakeHIS{visit: readyVisit()}, &fakeServicepoint{}, &fakeTransactor{})

	next, err := svc.GetNextStep(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("GetNextStep: %v", err)
	}
	if next.Sequence != 2 || next.ServicePoint == nil {
		t.Fatalf("next = %+v, want sequence 2 with service point", next)
	}

	noReady := NewService(
		&fakeHIS{visit: his.Visit{VisitID: "V", Steps: []his.VisitStep{{Sequence: 1, ServiceCode: "LAB", Status: "PENDING"}}}},
		&fakeServicepoint{}, &fakeTransactor{})
	if _, err := noReady.GetNextStep(context.Background(), "V"); !errors.Is(err, ErrNoNextStep) {
		t.Fatalf("error = %v, want ErrNoNextStep", err)
	}
}
