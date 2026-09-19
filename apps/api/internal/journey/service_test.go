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
	visit          his.Visit
	err            error
	getCalls       int
	getVisits      map[string]his.Visit
	transitionErr  error
	transitionCmds []his.TransitionCommand
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

// TransitionStep applies the canonical cascades on the stored snapshot so the
// service under test sees realistic post-command state: completing a step
// readies the next PENDING one, and closing the last open step completes the
// visit. Legality itself is the HIS's job and faked via transitionErr.
func (f *fakeHIS) TransitionStep(_ context.Context, visitID string, sequence int, cmd his.TransitionCommand) (his.VisitStep, error) {
	f.transitionCmds = append(f.transitionCmds, cmd)
	if f.transitionErr != nil {
		return his.VisitStep{}, f.transitionErr
	}
	v, ok := f.getVisits[visitID]
	if !ok {
		return his.VisitStep{}, his.ErrVisitNotFound
	}
	for i := range v.Steps {
		if v.Steps[i].Sequence != sequence {
			continue
		}
		v.Steps[i].Status = cmd.To
		if cmd.To == his.CommandToCompleted {
			for j := range v.Steps {
				if v.Steps[j].Status == "PENDING" {
					v.Steps[j].Status = "READY"
					break
				}
			}
			open := 0
			for _, st := range v.Steps {
				if st.Status != "COMPLETED" && st.Status != "CANCELLED" {
					open++
				}
			}
			if open == 0 {
				v.Status = "COMPLETED"
			}
		}
		f.getVisits[visitID] = v
		return v.Steps[i], nil
	}
	return his.VisitStep{}, apperr.New(apperr.KindNotFound, "step not found")
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
	out := make([]servicepoint.ServicePoint, 0, len(f.known))
	for code := range f.known {
		out = append(out, servicepoint.ServicePoint{
			ID: "SP-" + code, Code: code, Name: code, PlaceID: "PLACE-1", Active: true,
		})
	}
	return out, nil
}

type fakeRepo struct {
	visits     map[string]Visit
	applied    map[string]string
	upserts    int
	marks      int
	audits     []CommandAudit
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

func (f *fakeRepo) InsertCommandAudit(ctx context.Context, audit CommandAudit) error {
	_, f.inTxMarker = ctx.Value(txMarker{}).(bool)
	f.audits = append(f.audits, audit)
	return nil
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

// #18 AC2/AC3: the view orders steps by sequence and resolves current (first
// STARTED) and next (first READY) deterministically, with the bound service
// point resolved for display and unmapped steps kept explicitly unbound.
func TestGetJourneyOrdersStepsAndResolvesCurrentNext(t *testing.T) {
	repo := newFakeRepo()
	lab := "SP-LAB"
	// Stored out of order: the view must sort by sequence, not trust insertion.
	repo.visits["VISIT-001"] = Visit{
		VisitID: "VISIT-001", PatientRef: "PAT-001", Status: "ACTIVE",
		Steps: []Step{
			{Sequence: 3, ServiceCode: "MYSTERY", Status: "READY"},
			{Sequence: 1, ServiceCode: "REGISTRATION", Status: "COMPLETED", ServicePointID: &lab},
			{Sequence: 2, ServiceCode: "LAB", Status: "STARTED", ServicePointID: &lab},
		},
	}
	svc := newTestService(&fakeHIS{}, repo)

	got, err := svc.GetJourney(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("GetJourney: %v", err)
	}
	if got.Steps[0].Sequence != 1 || got.Steps[1].Sequence != 2 || got.Steps[2].Sequence != 3 {
		t.Fatalf("step order = %d,%d,%d, want 1,2,3",
			got.Steps[0].Sequence, got.Steps[1].Sequence, got.Steps[2].Sequence)
	}
	if got.Completed {
		t.Fatal("completed = true, want false for an ACTIVE visit")
	}
	if got.Current == nil || got.Current.Sequence != 2 || got.Current.ServicePoint == nil || got.Current.ServicePoint.ID != "SP-LAB" {
		t.Fatalf("current = %+v, want LAB (seq 2) resolved to SP-LAB", got.Current)
	}
	if got.Next == nil || got.Next.Sequence != 3 || got.Next.ServiceCode != "MYSTERY" {
		t.Fatalf("next = %+v, want the unmapped READY step at seq 3", got.Next)
	}
	if got.Next.ServicePointID != nil || got.Next.ServicePoint != nil {
		t.Fatalf("unmapped next = %+v, want explicit nil binding", got.Next)
	}
}

// #18 AC4: a finished visit reports completed=true with no actionable step.
func TestGetJourneyCompletedVisitIsExplicit(t *testing.T) {
	repo := newFakeRepo()
	lab := "SP-LAB"
	repo.visits["VISIT-001"] = Visit{
		VisitID: "VISIT-001", PatientRef: "PAT-001", Status: "COMPLETED",
		Steps: []Step{
			{Sequence: 1, ServiceCode: "REGISTRATION", Status: "COMPLETED", ServicePointID: &lab},
			{Sequence: 2, ServiceCode: "LAB", Status: "COMPLETED", ServicePointID: &lab},
		},
	}
	svc := newTestService(&fakeHIS{}, repo)

	got, err := svc.GetJourney(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("GetJourney: %v", err)
	}
	if got.Status != "COMPLETED" || !got.Completed {
		t.Fatalf("status/completed = %s/%v, want COMPLETED/true", got.Status, got.Completed)
	}
	if got.Current != nil || got.Next != nil {
		t.Fatalf("current/next = %+v/%+v, want nil/nil on a completed visit", got.Current, got.Next)
	}
}

// A visit the poller has not projected yet has no journey to serve (#18
// design: no read-through fallback — 404 until the projection exists).
func TestGetJourneyNotFoundWhenNotProjected(t *testing.T) {
	svc := newTestService(&fakeHIS{}, newFakeRepo())
	if _, err := svc.GetJourney(context.Background(), "NOPE"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

// #19 AC1/AC3: an allowed transition is forwarded, the projection is
// refreshed synchronously, and completing a step recomputes next — the
// following PENDING step becomes the next READY one in the same response.
func TestTransitionStepAllowedAndRecomputesNext(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed projection: %v", err)
	}
	repo.audits = nil

	view, err := svc.TransitionStep(context.Background(), "VISIT-001", 2,
		his.TransitionCommand{CommandID: "CMD-1", To: his.CommandToStarted}, "staff-web")
	if err != nil {
		t.Fatalf("start LAB: %v", err)
	}
	if view.Current == nil || view.Current.Sequence != 2 || view.Current.Status != "STARTED" {
		t.Fatalf("current after start = %+v, want LAB STARTED", view.Current)
	}
	if view.Next != nil {
		t.Fatalf("next after start = %+v, want nil (nothing READY yet)", view.Next)
	}

	view, err = svc.TransitionStep(context.Background(), "VISIT-001", 2,
		his.TransitionCommand{CommandID: "CMD-2", To: his.CommandToCompleted}, "staff-web")
	if err != nil {
		t.Fatalf("complete LAB: %v", err)
	}
	if view.Current != nil {
		t.Fatalf("current after complete = %+v, want nil", view.Current)
	}
	if view.Next == nil || view.Next.Sequence != 3 || view.Next.Status != "READY" {
		t.Fatalf("next after complete = %+v, want MYSTERY READY at sequence 3", view.Next)
	}
	if view.Completed {
		t.Fatal("completed = true, want false while steps remain open")
	}
}

// #19 AC2: a transition the HIS rejects (illegal/out-of-order) surfaces as a
// Conflict and leaves neither audit nor projection writes behind.
func TestTransitionStepRejectsIllegalFromHIS(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{
		getVisits:     map[string]his.Visit{"VISIT-001": snapshot()},
		transitionErr: apperr.New(apperr.KindConflict, "step 1 is COMPLETED and cannot transition to STARTED"),
	}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed projection: %v", err)
	}
	repo.audits = nil
	upserts := repo.upserts

	_, err := svc.TransitionStep(context.Background(), "VISIT-001", 1,
		his.TransitionCommand{CommandID: "CMD-X", To: his.CommandToStarted}, "staff-web")
	if apperr.KindOf(err) != apperr.KindConflict {
		t.Fatalf("error = %v, want KindConflict", err)
	}
	if len(repo.audits) != 0 || repo.upserts != upserts {
		t.Fatalf("audits = %d, upserts = %d (want 0 additional) after a rejected command", len(repo.audits), repo.upserts-upserts)
	}
}

// An unknown target status never reaches the HIS.
func TestTransitionStepUnknownTargetRejectedLocally(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)

	_, err := svc.TransitionStep(context.Background(), "VISIT-001", 2,
		his.TransitionCommand{To: "PAUSED"}, "staff-web")
	if apperr.KindOf(err) != apperr.KindInvalid {
		t.Fatalf("error = %v, want KindInvalid", err)
	}
	if len(hisClient.transitionCmds) != 0 {
		t.Fatalf("HIS calls = %d, want 0", len(hisClient.transitionCmds))
	}
}

func TestTransitionStepUnknownVisitIsNotFound(t *testing.T) {
	svc := newTestService(&fakeHIS{getVisits: map[string]his.Visit{}}, newFakeRepo())
	_, err := svc.TransitionStep(context.Background(), "NOPE", 1,
		his.TransitionCommand{CommandID: "CMD-V", To: his.CommandToStarted}, "staff-web")
	if apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("error = %v, want KindNotFound", err)
	}
}

// #19 AC4: every forwarded command lands in the audit trail with its
// idempotency key and source; the key is generated when the client omits it.
func TestTransitionStepAuditsCommand(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed projection: %v", err)
	}
	repo.audits = nil

	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", 2,
		his.TransitionCommand{To: his.CommandToStarted}, "patient-web"); err != nil {
		t.Fatalf("transition without commandId: %v", err)
	}
	if len(repo.audits) != 1 {
		t.Fatalf("audits = %d, want 1", len(repo.audits))
	}
	audit := repo.audits[0]
	if audit.CommandID == "" || audit.VisitID != "VISIT-001" || audit.Sequence != 2 ||
		audit.ToStatus != his.CommandToStarted || audit.Source != "patient-web" {
		t.Fatalf("audit = %+v, want generated key, VISIT-001 seq 2 STARTED from patient-web", audit)
	}
	if len(hisClient.transitionCmds) != 1 || hisClient.transitionCmds[0].CommandID != audit.CommandID {
		t.Fatalf("HIS command = %+v, want it keyed by the generated id", hisClient.transitionCmds)
	}
}
