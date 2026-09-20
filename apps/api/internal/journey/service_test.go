package journey

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

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
	closed     map[string]map[string]bool
	upserts    int
	marks      int
	audits     []CommandAudit
	events     []StepStatusEvent
	inTxMarker bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{visits: map[string]Visit{}, applied: map[string]string{}, closed: map[string]map[string]bool{}}
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

// ListVisits mirrors the port's ordering contract: freshest sync first.
func (f *fakeRepo) ListVisits(ctx context.Context) ([]Visit, error) {
	_, f.inTxMarker = ctx.Value(txMarker{}).(bool)
	out := make([]Visit, 0, len(f.visits))
	for _, v := range f.visits {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SyncedAt.After(out[j].SyncedAt) })
	return out, nil
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

func (f *fakeRepo) CloseRound(_ context.Context, visitID, stepKey string) error {
	if f.closed[visitID] == nil {
		f.closed[visitID] = map[string]bool{}
	}
	f.closed[visitID][stepKey] = true
	return nil
}

func (f *fakeRepo) ClosedRounds(_ context.Context, visitID string) (map[string]bool, error) {
	out := map[string]bool{}
	for k, v := range f.closed[visitID] {
		out[k] = v
	}
	return out, nil
}

func (f *fakeRepo) AppendStatusEvents(ctx context.Context, events []StepStatusEvent) error {
	_, f.inTxMarker = ctx.Value(txMarker{}).(bool)
	f.events = append(f.events, events...)
	return nil
}

func snapshot() his.Visit {
	return his.Visit{
		VisitID: "VISIT-001", PatientRef: "PAT-001", PatientName: "สมชาย",
		VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics:  []his.Clinic{{Code: "MED"}},
		OpenedAt: openedAt,
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
	sp := &fakeServicepoint{known: map[string]bool{"REGISTRATION": true, "ORDERTYPE:LAB": true, "PHARMACY": true, "CLINIC:MED": true}}
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
	if got.Status != his.VisitActive {
		t.Fatalf("projected visit status = %s, want ACTIVE", got.Status)
	}
	clinic, ok := stepByKey(got.Steps, "CLINIC:MED:1")
	if !ok || clinic.ServicePointID == nil || *clinic.ServicePointID != "SP-CLINIC:MED" {
		t.Fatalf("clinic step = %+v, want resolved to SP-CLINIC:MED", clinic)
	}
	if !repo.inTxMarker {
		t.Fatal("plan writes did not run inside a transaction")
	}
}

// AC2 of #21 (still holds under ADR-0009): a duplicate event never re-applies.
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
}

// A binding with no configured service point is projected with an explicit
// nil binding, not an error (#21 AC3, still holds under ADR-0009).
func TestUnmappedBindingProjectsExplicitly(t *testing.T) {
	repo := newFakeRepo()
	sp := &fakeServicepoint{known: map[string]bool{}} // nothing mapped
	svc := NewService(&fakeHIS{visit: snapshot()}, sp, repo, &fakeTransactor{})

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("ApplyHISEvent: %v", err)
	}

	got := repo.visits["VISIT-001"]
	clinic, ok := stepByKey(got.Steps, "CLINIC:MED:1")
	if !ok || clinic.ServicePointID != nil {
		t.Fatalf("unmapped clinic step = %+v, want service point nil", clinic)
	}
}

// A later event re-reads the snapshot, so an HIS-side fact shows up in the
// plan on replan.
func TestApplyHISEventReplansOnNewFacts(t *testing.T) {
	repo := newFakeRepo()
	latest := snapshot()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": latest}}
	svc := newTestService(hisClient, repo)

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	latest.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(-time.Hour), Status: his.OrderPlaced},
	}
	hisClient.getVisits["VISIT-001"] = latest
	next := his.Event{EventID: "EVT-000002", VisitID: "VISIT-001", PatientRef: "PAT-001", Type: his.EventOrderPlaced}
	if err := svc.ApplyHISEvent(context.Background(), next); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	got := repo.visits["VISIT-001"]
	if _, ok := stepByKey(got.Steps, "LAB:1"); !ok {
		t.Fatalf("expected a LAB step after the order.placed fact, got %+v", got.Steps)
	}
}

// encounter.completed closes the visit's current round at that clinic.
func TestApplyHISEventEncounterCompletedClosesRound(t *testing.T) {
	repo := newFakeRepo()
	visit := snapshot()
	visit.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(10 * time.Minute), Status: his.OrderPlaced},
	}
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": visit}}
	svc := newTestService(hisClient, repo)

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Staff starts the clinic round, then the doctor orders a mid-visit lab
	// (round 2 gets tentatively inferred) before confirming no return is
	// needed.
	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "C1", To: CommandToStarted}, "staff-web",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("start clinic: %v", err)
	}
	view, err := svc.GetJourney(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("GetJourney: %v", err)
	}
	if _, ok := stepByViewKey(view.Steps, "CLINIC:MED:2"); !ok {
		t.Fatalf("expected an inferred round 2 before closing, steps=%+v", view.Steps)
	}

	closeEvent := his.Event{
		EventID: "EVT-000003", VisitID: "VISIT-001", PatientRef: "PAT-001", Type: his.EventEncounterCompleted,
		Payload: map[string]any{"clinicCode": "MED"},
	}
	if err := svc.ApplyHISEvent(context.Background(), closeEvent); err != nil {
		t.Fatalf("encounter.completed: %v", err)
	}
	view, _ = svc.GetJourney(context.Background(), "VISIT-001")
	if _, ok := stepByViewKey(view.Steps, "CLINIC:MED:2"); ok {
		t.Fatalf("round 2 should be dropped after encounter.completed, steps=%+v", view.Steps)
	}
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:1"); s.Status != StepCompleted {
		t.Fatalf("round 1 after close = %s, want COMPLETED", s.Status)
	}
}

// encounter.started (the HIS's call-in fact) starts the clinic's actionable
// round; a repeat call while it is already in progress changes nothing.
func TestApplyHISEventEncounterStartedOpensRound(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	startEvent := func(id string) his.Event {
		return his.Event{
			EventID: id, VisitID: "VISIT-001", PatientRef: "PAT-001", Type: his.EventEncounterStarted,
			Payload: map[string]any{"clinicCode": "MED"},
		}
	}
	if err := svc.ApplyHISEvent(context.Background(), startEvent("EVT-000002")); err != nil {
		t.Fatalf("encounter.started: %v", err)
	}
	view, _ := svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:1"); s.Status != StepStarted {
		t.Fatalf("round 1 after call-in = %s, want STARTED", s.Status)
	}

	// The clinic calls again while the round is already in progress (no
	// actionable round exists): a no-op, not an error.
	if err := svc.ApplyHISEvent(context.Background(), startEvent("EVT-000003")); err != nil {
		t.Fatalf("repeat encounter.started: %v", err)
	}
	view, _ = svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:1"); s.Status != StepStarted {
		t.Fatalf("round 1 after repeat call-in = %s, want STARTED (no-op)", s.Status)
	}
	if _, marked := repo.applied["EVT-000003"]; !marked {
		t.Fatal("the no-op fact must still be marked applied")
	}
}

// The HIS-driven return-to-doctor story: call in (round 1 starts), order
// mid-visit (round 2 inferred WAITING), results arrive (round 2 READY), call
// in again (round 1 implicitly finished, round 2 STARTED), complete the
// encounter (round 2 COMPLETED, cashier actionable). No staff command anywhere.
func TestApplyHISEventEncounterStartedReturnRound(t *testing.T) {
	repo := newFakeRepo()
	visit := snapshot()
	visit.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(10 * time.Minute), Status: his.OrderPlaced},
	}
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": visit}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	view, _ := svc.GetJourney(context.Background(), "VISIT-001")
	if _, ok := stepByViewKey(view.Steps, "CLINIC:MED:2"); ok {
		t.Fatal("no return may be inferred before the round is in progress")
	}

	apply := func(id, typ string) {
		t.Helper()
		ev := his.Event{EventID: id, VisitID: "VISIT-001", PatientRef: "PAT-001", Type: typ}
		if typ == his.EventEncounterStarted || typ == his.EventEncounterCompleted {
			ev.Payload = map[string]any{"clinicCode": "MED"}
		}
		if err := svc.ApplyHISEvent(context.Background(), ev); err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
	}

	// Call in: round 1 starts, the mid-visit lab infers the return.
	apply("EVT-000002", his.EventEncounterStarted)
	view, _ = svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:1"); s.Status != StepStarted {
		t.Fatalf("round 1 after call-in = %s, want STARTED", s.Status)
	}
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:2"); s.Status != StepWaiting {
		t.Fatalf("inferred round 2 = %s, want WAITING", s.Status)
	}

	// The lab happens; results are not back yet, so the return stays WAITING.
	visit.Orders[0].Status = his.OrderPerformed
	visit.Orders[0].PerformedAt = timePtr(openedAt.Add(20 * time.Minute))
	hisClient.getVisits["VISIT-001"] = visit
	apply("EVT-000003", his.EventOrderPerformed)
	view, _ = svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "LAB:1"); s.Status != StepCompleted {
		t.Fatalf("lab after performed = %s, want COMPLETED", s.Status)
	}
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:2"); s.Status != StepWaiting {
		t.Fatalf("round 2 before results = %s, want WAITING", s.Status)
	}

	// Results arrive: the return becomes actionable.
	visit.Orders[0].Status = his.OrderResulted
	visit.Orders[0].ResultedAt = timePtr(openedAt.Add(30 * time.Minute))
	hisClient.getVisits["VISIT-001"] = visit
	apply("EVT-000004", his.EventOrderResulted)
	view, _ = svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:2"); s.Status != StepReady {
		t.Fatalf("round 2 after results = %s, want READY", s.Status)
	}

	// Call in again: the patient is returning, so round 1 — open since the
	// first call — is implicitly finished and round 2 starts.
	apply("EVT-000005", his.EventEncounterStarted)
	view, _ = svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:1"); s.Status != StepCompleted {
		t.Fatalf("round 1 after the return call-in = %s, want COMPLETED (implicitly finished)", s.Status)
	}
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:2"); s.Status != StepStarted {
		t.Fatalf("round 2 after the return call-in = %s, want STARTED", s.Status)
	}

	// The doctor wraps up: the encounter completes the round in progress,
	// and with everything else terminal the cashier opens.
	apply("EVT-000006", his.EventEncounterCompleted)
	view, _ = svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:2"); s.Status != StepCompleted {
		t.Fatalf("round 2 after encounter.completed = %s, want COMPLETED", s.Status)
	}
	if s, _ := stepByViewKey(view.Steps, "CASHIER"); s.Status != StepReady {
		t.Fatalf("cashier at the end = %s, want READY", s.Status)
	}
}

// A call-in with no actionable round to open (results not back yet) is a
// no-op: the patient cannot be called into a return that is still WAITING.
func TestApplyHISEventEncounterStartedBeforeResultsIsNoOp(t *testing.T) {
	repo := newFakeRepo()
	visit := snapshot()
	visit.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(10 * time.Minute), Status: his.OrderPlaced},
	}
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": visit}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := svc.ApplyHISEvent(context.Background(), his.Event{
		EventID: "EVT-000002", VisitID: "VISIT-001", PatientRef: "PAT-001", Type: his.EventEncounterStarted,
		Payload: map[string]any{"clinicCode": "MED"},
	}); err != nil {
		t.Fatalf("first call-in: %v", err)
	}
	before := len(repo.events)
	if err := svc.ApplyHISEvent(context.Background(), his.Event{
		EventID: "EVT-000003", VisitID: "VISIT-001", PatientRef: "PAT-001", Type: his.EventEncounterStarted,
		Payload: map[string]any{"clinicCode": "MED"},
	}); err != nil {
		t.Fatalf("premature call-in: %v", err)
	}
	view, _ := svc.GetJourney(context.Background(), "VISIT-001")
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:1"); s.Status != StepStarted {
		t.Fatalf("round 1 = %s, want still STARTED", s.Status)
	}
	if s, _ := stepByViewKey(view.Steps, "CLINIC:MED:2"); s.Status != StepWaiting {
		t.Fatalf("round 2 = %s, want still WAITING", s.Status)
	}
	if got := len(repo.events); got != before {
		t.Fatalf("timeline grew by %d rows on a no-op call-in, want 0: %+v", got-before, repo.events[before:])
	}
}

func stepByViewKey(steps []StepView, key string) (StepView, bool) {
	for _, s := range steps {
		if s.StepKey == key {
			return s, true
		}
	}
	return StepView{}, false
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

// The view orders steps by sequence, lists every READY step as actionable,
// and recommends the first of them.
func TestGetJourneyResolvesActionableAndRecommended(t *testing.T) {
	repo := newFakeRepo()
	sp := "SP-CLINIC:MED"
	repo.visits["VISIT-001"] = Visit{
		VisitID: "VISIT-001", PatientRef: "PAT-001", Status: his.VisitActive,
		Steps: []Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: KindRegistration, Status: StepCompleted},
			{StepKey: "CLINIC:MED:1", Sequence: 2, Kind: KindClinic, ClinicCode: strPtr("MED"), Round: intPtr(1), Status: StepReady, ServicePointID: &sp},
			{StepKey: "CASHIER", Sequence: 3, Kind: KindCashier, Status: StepPending},
		},
	}
	svc := newTestService(&fakeHIS{}, repo)

	got, err := svc.GetJourney(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("GetJourney: %v", err)
	}
	if len(got.Steps) != 3 || got.Steps[0].Sequence != 1 || got.Steps[2].Sequence != 3 {
		t.Fatalf("steps = %+v, want 3 ordered by sequence", got.Steps)
	}
	if got.Completed {
		t.Fatal("completed = true, want false for an ACTIVE visit")
	}
	if len(got.Actionable) != 1 || got.Actionable[0].StepKey != "CLINIC:MED:1" {
		t.Fatalf("actionable = %+v, want just CLINIC:MED:1", got.Actionable)
	}
	if got.Recommended == nil || got.Recommended.StepKey != "CLINIC:MED:1" || got.Recommended.ServicePoint == nil {
		t.Fatalf("recommended = %+v, want CLINIC:MED:1 resolved to its service point", got.Recommended)
	}
}

func strPtr(v string) *string { return &v }

// A finished visit reports completed=true with no actionable step.
func TestGetJourneyCompletedVisitIsExplicit(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-001"] = Visit{
		VisitID: "VISIT-001", PatientRef: "PAT-001", Status: his.VisitCompleted,
		Steps: []Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: KindRegistration, Status: StepCompleted},
			{StepKey: "CASHIER", Sequence: 2, Kind: KindCashier, Status: StepCompleted},
		},
	}
	svc := newTestService(&fakeHIS{}, repo)

	got, err := svc.GetJourney(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("GetJourney: %v", err)
	}
	if got.Status != his.VisitCompleted || !got.Completed {
		t.Fatalf("status/completed = %s/%v, want COMPLETED/true", got.Status, got.Completed)
	}
	if len(got.Actionable) != 0 || got.Recommended != nil {
		t.Fatalf("actionable/recommended = %+v/%+v, want empty/nil on a completed visit", got.Actionable, got.Recommended)
	}
}

// A visit the poller has not projected yet has no journey to serve.
func TestGetJourneyNotFoundWhenNotProjected(t *testing.T) {
	svc := newTestService(&fakeHIS{}, newFakeRepo())
	if _, err := svc.GetJourney(context.Background(), "NOPE"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

// #37: the staff monitor lists every projected visit, freshest sync first.
func TestListJourneysResolvesEveryVisit(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-A"] = Visit{
		VisitID: "VISIT-A", PatientRef: "PAT-A", Status: his.VisitActive,
		SyncedAt: time.Now().Add(-time.Minute),
		Steps: []Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: KindRegistration, Status: StepCompleted},
			{StepKey: "CLINIC:MED:1", Sequence: 2, Kind: KindClinic, Status: StepStarted},
		},
	}
	repo.visits["VISIT-B"] = Visit{
		VisitID: "VISIT-B", PatientRef: "PAT-B", Status: his.VisitCompleted,
		SyncedAt: time.Now(),
		Steps: []Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: KindRegistration, Status: StepCompleted},
		},
	}
	svc := newTestService(&fakeHIS{}, repo)

	got, err := svc.ListJourneys(context.Background())
	if err != nil {
		t.Fatalf("ListJourneys: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("journeys = %d, want 2", len(got))
	}
	if got[0].VisitID != "VISIT-B" || got[1].VisitID != "VISIT-A" {
		t.Fatalf("order = %s then %s, want freshest (VISIT-B) first", got[0].VisitID, got[1].VisitID)
	}
	if !repo.inTxMarker {
		t.Fatal("list reads did not run inside a transaction")
	}
}

// No projected visits is an empty list, not null — the contract promises an
// array.
func TestListJourneysEmptyIsArray(t *testing.T) {
	svc := newTestService(&fakeHIS{}, newFakeRepo())
	got, err := svc.ListJourneys(context.Background())
	if err != nil {
		t.Fatalf("ListJourneys: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("got = %#v, want an empty non-nil slice", got)
	}
}

// A step transition is applied locally (never forwarded to the HIS) and the
// plan is recomputed so downstream gates react immediately.
func TestTransitionStepAppliesLocallyAndReplans(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	view, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "CMD-1", To: CommandToStarted}, "staff-web", Actor{UserID: "user-1", Username: "tester"})
	if err != nil {
		t.Fatalf("start clinic: %v", err)
	}
	if s, ok := stepByViewKey(view.Steps, "CLINIC:MED:1"); !ok || s.Status != StepStarted {
		t.Fatalf("clinic after start = %+v, want STARTED", s)
	}

	view, err = svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "CMD-2", To: CommandToCompleted}, "staff-web", Actor{UserID: "user-1", Username: "tester"})
	if err != nil {
		t.Fatalf("complete clinic: %v", err)
	}
	if s, ok := stepByViewKey(view.Steps, "CASHIER"); !ok || s.Status != StepReady {
		t.Fatalf("cashier after clinic completed = %+v, want READY", s)
	}
	if len(hisClient.getVisits) == 0 {
		t.Fatal("sanity: fake HIS visits missing")
	}
}

// A transition from a terminal status is rejected.
func TestTransitionStepRejectsFromTerminal(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := svc.TransitionStep(context.Background(), "VISIT-001", "REGISTRATION",
		TransitionCommand{CommandID: "CMD-X", To: CommandToStarted}, "staff-web", Actor{UserID: "user-1", Username: "tester"})
	if apperr.KindOf(err) != apperr.KindConflict {
		t.Fatalf("error = %v, want KindConflict (REGISTRATION is already COMPLETED)", err)
	}
}

// STARTED is rejected on a step that is not yet READY.
func TestTransitionStepRejectsStartBeforeReady(t *testing.T) {
	repo := newFakeRepo()
	visit := snapshot()
	visit.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(-time.Hour), Status: his.OrderPlaced},
	}
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": visit}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "CMD-Y", To: CommandToStarted}, "staff-web", Actor{UserID: "user-1", Username: "tester"})
	if apperr.KindOf(err) != apperr.KindConflict {
		t.Fatalf("error = %v, want KindConflict (clinic still PENDING behind the lab)", err)
	}
}

// An unknown target status is rejected before touching the repo.
func TestTransitionStepUnknownTargetRejected(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)

	_, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{To: "PAUSED"}, "staff-web", Actor{UserID: "user-1", Username: "tester"})
	if apperr.KindOf(err) != apperr.KindInvalid {
		t.Fatalf("error = %v, want KindInvalid", err)
	}
}

func TestTransitionStepUnknownStepIsNotFound(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := svc.TransitionStep(context.Background(), "VISIT-001", "NOPE",
		TransitionCommand{CommandID: "CMD-V", To: CommandToStarted}, "staff-web", Actor{UserID: "user-1", Username: "tester"})
	if apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("error = %v, want KindNotFound", err)
	}
}

// Every transition lands in the audit trail with its idempotency key and
// source; the key is generated when the client omits it.
func TestTransitionStepAuditsCommand(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{To: CommandToStarted}, "patient-web",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("transition without commandId: %v", err)
	}
	if len(repo.audits) != 1 {
		t.Fatalf("audits = %d, want 1", len(repo.audits))
	}
	audit := repo.audits[0]
	if audit.CommandID == "" || audit.VisitID != "VISIT-001" || audit.StepKey != "CLINIC:MED:1" ||
		audit.ToStatus != CommandToStarted || audit.Source != "patient-web" {
		t.Fatalf("audit = %+v, want generated key, VISIT-001 CLINIC:MED:1 STARTED from patient-web", audit)
	}
}

// The staff override drops a not-yet-started inferred return even without an
// encounter.completed fact.
func TestCloseRoundOverride(t *testing.T) {
	repo := newFakeRepo()
	visit := snapshot()
	visit.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(10 * time.Minute), Status: his.OrderPlaced},
	}
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": visit}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "C1", To: CommandToStarted}, "staff-web",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("start clinic: %v", err)
	}
	view, _ := svc.GetJourney(context.Background(), "VISIT-001")
	if _, ok := stepByViewKey(view.Steps, "CLINIC:MED:2"); !ok {
		t.Fatal("expected round 2 to be inferred before the override")
	}

	view, err := svc.CloseRound(context.Background(), "VISIT-001", "MED", "test", Actor{UserID: "user-1", Username: "tester"})
	if err != nil {
		t.Fatalf("CloseRound: %v", err)
	}
	if _, ok := stepByViewKey(view.Steps, "CLINIC:MED:2"); ok {
		t.Fatal("round 2 should be dropped after the close-round override")
	}
}

func TestCloseRoundNoOpenRoundIsNotFound(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := svc.CloseRound(context.Background(), "VISIT-001", "SURG", "test", Actor{UserID: "user-1", Username: "tester"})
	if !errors.Is(err, ErrNoOpenRound) {
		t.Fatalf("error = %v, want ErrNoOpenRound", err)
	}
}

// --- #85: the append-only step status timeline -----------------------------

func eventsOf(events []StepStatusEvent, stepKey string) []StepStatusEvent {
	var out []StepStatusEvent
	for _, ev := range events {
		if ev.StepKey == stepKey {
			out = append(out, ev)
		}
	}
	return out
}

// #85 AC: a visit's first projection appends one row per step, each with
// from_status NULL (the step just entered the plan).
func TestTimelineFirstProjectionRecordsEveryStep(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(&fakeHIS{visit: snapshot()}, repo)

	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("ApplyHISEvent: %v", err)
	}

	if len(repo.events) != 3 {
		t.Fatalf("events = %d, want 3 (REGISTRATION, CLINIC:MED:1, CASHIER): %+v", len(repo.events), repo.events)
	}
	byKey := map[string]StepStatusEvent{}
	for _, ev := range repo.events {
		if ev.FromStatus != nil {
			t.Fatalf("event %s from_status = %v, want nil (step just entered the plan)", ev.StepKey, *ev.FromStatus)
		}
		if ev.Source != EventSourcePlanner || ev.ActorUserID != "" {
			t.Fatalf("event %s source/actor = %s/%s, want planner with no actor", ev.StepKey, ev.Source, ev.ActorUserID)
		}
		if ev.VisitID != "VISIT-001" || ev.Kind == "" {
			t.Fatalf("event = %+v, want visit id and a copied kind", ev)
		}
		byKey[ev.StepKey] = ev
	}
	if byKey["REGISTRATION"].ToStatus != StepCompleted ||
		byKey["CLINIC:MED:1"].ToStatus != StepReady ||
		byKey["CASHIER"].ToStatus != StepPending {
		t.Fatalf("initial statuses = %+v, want COMPLETED/READY/PENDING", byKey)
	}
}

// #85 AC: a replan that changes nothing appends not a single row — the
// ingest poller replans regularly and the table must not grow on no-ops.
func TestTimelineNoOpReplanAppendsNothing(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	before := len(repo.events)

	// A fresh event id for the same unchanged snapshot still replans…
	next := his.Event{EventID: "EVT-000002", VisitID: "VISIT-001", PatientRef: "PAT-001", Type: his.EventVisitUpdated}
	if err := svc.ApplyHISEvent(context.Background(), next); err != nil {
		t.Fatalf("re-apply: %v", err)
	}
	if repo.upserts != 2 {
		t.Fatalf("upserts = %d, want 2 (the replan did run)", repo.upserts)
	}
	// …but the timeline must not have grown.
	if got := len(repo.events); got != before {
		t.Fatalf("events = %d, want %d after a no-op replan: %+v", got, before, repo.events[before:])
	}
}

// #85 AC: a staff transition is recorded with the pre-command status and the
// command's source and actor; planner-driven changes of the same replan
// round (the next step's gate opening) are recorded separately as planner
// events without an actor.
func TestTimelineTransitionRecordsCommandAndPlannerCascade(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "CMD-1", To: CommandToStarted}, "staff-web",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("start clinic: %v", err)
	}
	started := eventsOf(repo.events, "CLINIC:MED:1")
	if len(started) != 2 { // nil→READY from the first projection, READY→STARTED from the command
		t.Fatalf("clinic events = %+v, want the initial projection plus the command", started)
	}
	cmd := started[1]
	if cmd.FromStatus == nil || *cmd.FromStatus != StepReady || cmd.ToStatus != StepStarted {
		t.Fatalf("command event = %+v, want READY→STARTED", cmd)
	}
	if cmd.Source != "staff-web" || cmd.ActorUserID != "user-1" || cmd.ActorUsername != "tester" {
		t.Fatalf("command event source/actor = %s/%s,%s, want staff-web/user-1,tester", cmd.Source, cmd.ActorUserID, cmd.ActorUsername)
	}

	before := len(repo.events)
	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "CMD-2", To: CommandToCompleted}, "staff-web",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("complete clinic: %v", err)
	}
	round := repo.events[before:]
	if len(round) != 2 {
		t.Fatalf("events of the completing round = %+v, want the command plus the cashier gate opening", round)
	}
	completed := eventsOf(round, "CLINIC:MED:1")
	if len(completed) != 1 || completed[0].FromStatus == nil || *completed[0].FromStatus != StepStarted ||
		completed[0].ToStatus != StepCompleted || completed[0].Source != "staff-web" || completed[0].ActorUserID != "user-1" {
		t.Fatalf("completing command event = %+v, want STARTED→COMPLETED attributed to the command", completed)
	}
	cashier := eventsOf(round, "CASHIER")
	if len(cashier) != 1 || cashier[0].FromStatus == nil || *cashier[0].FromStatus != StepPending ||
		cashier[0].ToStatus != StepReady || cashier[0].Source != EventSourcePlanner || cashier[0].ActorUserID != "" {
		t.Fatalf("cashier event = %+v, want PENDING→READY by the planner with no actor", cashier)
	}
}

// #85 AC: replaying the same command id re-audits but never grows the
// timeline (nothing changed).
func TestTimelineDuplicateCommandDoesNotGrow(t *testing.T) {
	repo := newFakeRepo()
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": snapshot()}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	for i := 0; i < 2; i++ {
		if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
			TransitionCommand{CommandID: "CMD-1", To: CommandToStarted}, "staff-web",
			Actor{UserID: "user-1", Username: "tester"}); err != nil {
			t.Fatalf("start clinic %d: %v", i, err)
		}
	}
	started := eventsOf(repo.events, "CLINIC:MED:1")
	if len(started) != 2 {
		t.Fatalf("clinic events = %+v, want exactly the projection and one command row", started)
	}
	if len(repo.audits) != 2 {
		t.Fatalf("audits = %d, want 2 (a replayed command is still re-audited)", len(repo.audits))
	}
}

// #85 AC: a step withdrawn by a replan is recorded as CANCELLED. The HIS's
// encounter.completed fact drives the close, so the promotion to COMPLETED is
// attributed to the planner with no actor (the audit table has no row here
// either — no staff commanded anything).
func TestTimelineWithdrawnStepRecordedAsCancelled(t *testing.T) {
	repo := newFakeRepo()
	visit := snapshot()
	visit.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(10 * time.Minute), Status: his.OrderPlaced},
	}
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": visit}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "C1", To: CommandToStarted}, "staff-web",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("start clinic: %v", err)
	}
	before := len(repo.events)

	closeEvent := his.Event{
		EventID: "EVT-000003", VisitID: "VISIT-001", PatientRef: "PAT-001", Type: his.EventEncounterCompleted,
		Payload: map[string]any{"clinicCode": "MED"},
	}
	if err := svc.ApplyHISEvent(context.Background(), closeEvent); err != nil {
		t.Fatalf("encounter.completed: %v", err)
	}

	round := repo.events[before:]
	if len(round) != 2 {
		t.Fatalf("events of the closing round = %+v, want the round completing and the withdrawal", round)
	}
	completed := eventsOf(round, "CLINIC:MED:1")
	if len(completed) != 1 || completed[0].FromStatus == nil || *completed[0].FromStatus != StepStarted ||
		completed[0].ToStatus != StepCompleted || completed[0].Source != EventSourcePlanner || completed[0].ActorUserID != "" {
		t.Fatalf("round-1 completing event = %+v, want STARTED→COMPLETED by the planner with no actor", completed)
	}
	withdrawn := eventsOf(round, "CLINIC:MED:2")
	if len(withdrawn) != 1 || withdrawn[0].FromStatus == nil || *withdrawn[0].FromStatus != StepWaiting ||
		withdrawn[0].ToStatus != StepCancelled || withdrawn[0].Source != EventSourcePlanner {
		t.Fatalf("withdrawn round-2 event = %+v, want WAITING→CANCELLED", withdrawn)
	}
	if withdrawn[0].Kind != KindClinic {
		t.Fatalf("withdrawn event kind = %s, want the kind copied from the withdrawn step", withdrawn[0].Kind)
	}
}

// The staff close-round override takes effect through the replan, so its
// timeline row carries the command's source and actor (NFR-09), while the
// inferred return it drops is a planner withdrawal.
func TestTimelineCloseRoundAttributedToCommand(t *testing.T) {
	repo := newFakeRepo()
	visit := snapshot()
	visit.Orders = []his.Order{
		{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: openedAt.Add(10 * time.Minute), Status: his.OrderPlaced},
	}
	hisClient := &fakeHIS{getVisits: map[string]his.Visit{"VISIT-001": visit}}
	svc := newTestService(hisClient, repo)
	if err := svc.ApplyHISEvent(context.Background(), openedEvent()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := svc.TransitionStep(context.Background(), "VISIT-001", "CLINIC:MED:1",
		TransitionCommand{CommandID: "C1", To: CommandToStarted}, "staff-web",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("start clinic: %v", err)
	}
	before := len(repo.events)

	if _, err := svc.CloseRound(context.Background(), "VISIT-001", "MED", "test",
		Actor{UserID: "user-1", Username: "tester"}); err != nil {
		t.Fatalf("CloseRound: %v", err)
	}

	round := repo.events[before:]
	completed := eventsOf(round, "CLINIC:MED:1")
	if len(completed) != 1 || completed[0].FromStatus == nil || *completed[0].FromStatus != StepStarted ||
		completed[0].ToStatus != StepCompleted || completed[0].Source != "test" || completed[0].ActorUserID != "user-1" {
		t.Fatalf("close-round event = %+v, want STARTED→COMPLETED attributed to the command", completed)
	}
	if withdrawn := eventsOf(round, "CLINIC:MED:2"); len(withdrawn) != 1 || withdrawn[0].ToStatus != StepCancelled {
		t.Fatalf("withdrawn round-2 event = %+v, want CANCELLED", withdrawn)
	}
}
