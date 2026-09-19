package journey

import (
	"context"
	"errors"
	"testing"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/servicepoint"
)

type txMarker struct{}

// fakeTransactor mimics db.DB: it tags ctx as transactional and joins an
// already-tagged ctx instead of opening a second transaction.
type fakeTransactor struct{}

func (f *fakeTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txMarker{}).(bool); ok {
		return fn(ctx)
	}
	return fn(context.WithValue(ctx, txMarker{}, true))
}

type fakeHIS struct {
	visit     his.Visit
	err       error
	getCalls  int
	getVisits map[string]his.Visit
}

func (f *fakeHIS) GetVisit(_ context.Context, visitID string) (his.Visit, error) {
	f.getCalls++
	if f.getVisits != nil {
		if v, ok := f.getVisits[visitID]; ok {
			return v, nil
		}
	}
	return f.visit, f.err
}

func (f *fakeHIS) Events(_ context.Context, _ string, _ int) (his.EventPage, error) {
	return his.EventPage{}, nil
}

type fakeServicepoint struct {
	known map[string]bool
}

func (f *fakeServicepoint) GetByCode(_ context.Context, code string) (servicepoint.ServicePoint, error) {
	if !f.known[code] {
		return servicepoint.ServicePoint{}, servicepoint.ErrNotFound
	}
	return servicepoint.ServicePoint{ID: "SP-" + code, Code: code, Name: code, PlaceID: "PLACE-1", Active: true}, nil
}

func (f *fakeServicepoint) List(context.Context) ([]servicepoint.ServicePoint, error) {
	return nil, nil
}

type fakeRepo struct {
	visits     map[string]Visit
	applied    map[string]string
	upserts    int
	marks      int
	inTxMarker bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{visits: map[string]Visit{}, applied: map[string]string{}}
}

func (f *fakeRepo) UpsertVisit(ctx context.Context, visit Visit) error {
	f.upserts++
	_, f.inTxMarker = ctx.Value(txMarker{}).(bool)
	f.visits[visit.VisitID] = visit
	return nil
}

func (f *fakeRepo) GetVisit(_ context.Context, visitID string) (Visit, error) {
	if v, ok := f.visits[visitID]; ok {
		return v, nil
	}
	return Visit{}, ErrNotFound
}

func (f *fakeRepo) MarkEventApplied(ctx context.Context, eventID, visitID string) error {
	f.marks++
	_, f.inTxMarker = ctx.Value(txMarker{}).(bool)
	f.applied[eventID] = visitID
	return nil
}

func (f *fakeRepo) EventApplied(_ context.Context, eventID string) (bool, error) {
	_, ok := f.applied[eventID]
	return ok, nil
}

func snapshot() his.Visit {
	return his.Visit{
		VisitID:    "VISIT-001",
		PatientRef: "PAT-001",
		Status:     "ACTIVE",
		Steps: []his.VisitStep{
			{Sequence: 1, ServiceCode: "REGISTRATION", Status: "COMPLETED"},
			{Sequence: 2, ServiceCode: "LAB", Status: "READY"},
			{Sequence: 3, ServiceCode: "MYSTERY", Status: "PENDING"},
		},
	}
}

func openedEvent() his.Event {
	return his.Event{
		EventID:    "EVT-000001",
		VisitID:    "VISIT-001",
		PatientRef: "PAT-001",
		Type:       his.EventVisitOpened,
	}
}

func newTestService(hisClient his.Client, repo *fakeRepo) Service {
	sp := &fakeServicepoint{known: map[string]bool{"REGISTRATION": true, "LAB": true, "PHARMACY": true}}
	return NewService(hisClient, sp, repo, &fakeTransactor{})
}

func TestApplyHISEventProjectsJourney(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(&fakeHIS{visit: snapshot()}, repo)

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("ApplyHISEvent: %v", err)
	}

	got, err := svc.GetVisit(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("GetVisit: %v", err)
	}
	if got.Status != "ACTIVE" || len(got.Steps) != 3 {
		t.Fatalf("projected visit = %+v, want ACTIVE with 3 steps", got)
	}
	if got.Steps[1].ServicePointID == nil || *got.Steps[1].ServicePointID != "SP-LAB" {
		t.Fatalf("LAB step service point = %v, want SP-LAB", got.Steps[1].ServicePointID)
	}
	if !repo.inTxMarker {
		t.Fatal("projection writes did not run inside a transaction")
	}
}

// AC2 of #21: a duplicate event never re-projects and never duplicates steps.
func TestDuplicateEventIsNoOp(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{visit: snapshot()}
	svc := newTestService(hisClient, repo)

	event := openedEvent()
	for i := 0; i < 3; i++ {
		if err := svc.ApplyHISEvent(context.Background(), event); err != nil {
			t.Fatalf("apply %d: %v", i, err)
		}
	}
	if hisClient.getCalls != 1 {
		t.Fatalf("HIS snapshot reads = %d, want 1 (duplicates answered by the eventId check)", hisClient.getCalls)
	}
	if repo.upserts != 1 || repo.marks != 1 {
		t.Fatalf("upserts = %d, marks = %d, want 1 and 1", repo.upserts, repo.marks)
	}
	if got := repo.visits["VISIT-001"]; len(got.Steps) != 3 {
		t.Fatalf("projected steps = %d, want 3 (no duplicates)", len(got.Steps))
	}
}

// AC3 of #21: a service code without a configured service point is projected
// with an explicit nil binding, not an error.
func TestUnmappedServiceCodeProjectsExplicitly(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(&fakeHIS{visit: snapshot()}, repo)

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("ApplyHISEvent: %v", err)
	}

	got, _ := repo.visits["VISIT-001"]
	mystery := got.Steps[2]
	if mystery.ServiceCode != "MYSTERY" || mystery.ServicePointID != nil {
		t.Fatalf("unmapped step = %+v, want service point nil", mystery)
	}
	if len(got.Steps) != 3 {
		t.Fatalf("steps = %d, want the unmapped step kept alongside mapped ones", len(got.Steps))
	}
}

// A later event re-reads the snapshot, so an HIS-side transition shows up in
// the projection — the event "drives" the journey.
func TestApplyHISEventReconcilesStatus(t *testing.T) {
	repo := newFakeRepo()
	latest := snapshot()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": latest}}
	svc := newTestService(hisClient, repo)

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	latest.Steps[1].Status = "STARTED"
	latest.Steps[2].Status = "READY"
	next := his.Event{
		EventID: "EVT-000002", VisitID: "VISIT-001", PatientRef: "PAT-001",
		Type: his.EventServiceStarted,
	}
	if err := svc.ApplyHISEvent(context.Background(), next); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	got, _ := repo.visits["VISIT-001"]
	if got.Steps[1].Status != "STARTED" || got.Steps[2].Status != "READY" {
		t.Fatalf("statuses = %s/%s, want STARTED/READY after reconcile",
			got.Steps[1].Status, got.Steps[2].Status)
	}
}

// An event for a visit the HIS no longer knows is recorded as applied so the
// feed can advance; there is nothing to project.
func TestEventForUnknownVisitMarksApplied(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(&fakeHIS{err: his.ErrVisitNotFound}, repo)

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("ApplyHISEvent: %v (want applied-without-projection)", err)
	}
	if _, marked := repo.applied["EVT-000001"]; !marked {
		t.Fatal("event was not marked applied")
	}
	if repo.upserts != 0 {
		t.Fatalf("upserts = %d, want 0", repo.upserts)
	}
}

func TestInvalidEnvelopeRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(&fakeHIS{visit: snapshot()}, repo)

	noID := openedEvent()
	noID.EventID = ""
	if err := svc.ApplyHISEvent(context.Background(), noID); apperr.KindOf(err) != apperr.KindInvalid {
		t.Fatalf("error = %v, want KindInvalid", err)
	}
}

func TestUpstreamErrorPropagates(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(&fakeHIS{err: his.ErrUpstream}, repo)

	err := svc.ApplyHISEvent(context.Background(), openedEvent())
	if err == nil || apperr.KindOf(err) != apperr.KindUpstream {
		t.Fatalf("error = %v, want KindUpstream", err)
	}
	if len(repo.applied) != 0 {
		t.Fatal("event must not be marked applied on upstream failure")
	}
}

func TestGetVisitNotFound(t *testing.T) {
	svc := newTestService(&fakeHIS{}, newFakeRepo())
	if _, err := svc.GetVisit(context.Background(), "NOPE"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
