package share

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/journey"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/servicepoint"
)

// ---- fakes -------------------------------------------------------------

type fakeRepo struct {
	links map[string]ShareLink // by token hash
}

func (f *fakeRepo) Create(_ context.Context, link ShareLink) error {
	if f.links == nil {
		f.links = map[string]ShareLink{}
	}
	f.links[link.TokenHash] = link
	return nil
}

func (f *fakeRepo) GetByTokenHash(_ context.Context, tokenHash string) (ShareLink, error) {
	link, ok := f.links[tokenHash]
	if !ok {
		return ShareLink{}, apperr.New(apperr.KindNotFound, "share link not found")
	}
	return link, nil
}

func (f *fakeRepo) CountActive(_ context.Context, visitID string) (int, error) {
	count := 0
	for _, link := range f.links {
		if link.VisitID == visitID && link.Active(time.Now()) {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepo) RevokeActive(_ context.Context, visitID string) (int, error) {
	now := time.Now()
	revoked := 0
	for hash, link := range f.links {
		if link.VisitID == visitID && link.Active(now) {
			link.RevokedAt = &now
			f.links[hash] = link
			revoked++
		}
	}
	return revoked, nil
}

// fakeJourneys answers from preset data; methods the tests never reach panic
// so an unexpected call fails loudly.
type fakeJourneys struct {
	visit journey.Visit
	view  journey.View
}

func (f *fakeJourneys) ApplyHISEvent(context.Context, his.Event) error {
	panic("not implemented in fake")
}

func (f *fakeJourneys) GetVisit(_ context.Context, _ string) (journey.Visit, error) {
	if f.visit.VisitID == "" {
		return journey.Visit{}, journey.ErrNotFound
	}
	return f.visit, nil
}

func (f *fakeJourneys) GetJourney(_ context.Context, _ string) (journey.View, error) {
	if f.view.VisitID == "" {
		return journey.View{}, journey.ErrNotFound
	}
	return f.view, nil
}

func (f *fakeJourneys) GetQueue(_ context.Context, _ string) (journey.QueueView, error) {
	panic("not implemented in fake")
}

func (f *fakeJourneys) ListJourneys(context.Context) ([]journey.View, error) {
	panic("not implemented in fake")
}

func (f *fakeJourneys) TransitionStep(context.Context, string, string, journey.TransitionCommand, string, journey.Actor) (journey.View, error) {
	panic("not implemented in fake")
}

func (f *fakeJourneys) CloseRound(context.Context, string, string, string, journey.Actor) (journey.View, error) {
	panic("not implemented in fake")
}

// fakeTx satisfies db.Transactor without a database.
type fakeTx struct{}

func (fakeTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func str(s string) *string { return &s }
func num(n int) *int       { return &n }

func newPlaceWithFloor(placeID, placeName, floorCode, floorName string) *hospitalmap.Place {
	return &hospitalmap.Place{
		ID: placeID, FloorID: "F-1", Name: placeName, PlaceType: "ROOM",
		Floor: &hospitalmap.Floor{ID: "F-1", BuildingID: "B-1", Code: floorCode, Name: floorName},
	}
}

// ---- service behaviour -------------------------------------------------

func newTestService(repo Repo, journeys journey.Service) Service {
	return NewService(repo, journeys, fakeTx{}, time.Hour)
}

// A created link resolves, carries the redacted step, and reports its expiry.
func TestCreateAndResolve(t *testing.T) {
	synced := time.Now().Add(-3 * time.Minute)
	journeys := &fakeJourneys{
		visit: journey.Visit{VisitID: "VISIT-SHARE-1", PatientRef: "PAT-1", Status: journey.VisitActive},
		view: journey.View{
			VisitID: "VISIT-SHARE-1", Status: journey.VisitActive, SyncedAt: synced,
			Recommended: &journey.StepView{
				StepKey: "LAB:1", Kind: journey.KindLab, Status: journey.StepReady,
				ServicePoint: &servicepoint.ServicePoint{ID: "SP-ORDERTYPE-LAB", Name: "Laboratory"},
			},
		},
	}
	svc := newTestService(&fakeRepo{}, journeys)

	secret, err := svc.CreateLink(context.Background(), "VISIT-SHARE-1")
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	if len(secret.Token) != 64 { // 32 bytes hex
		t.Fatalf("token length = %d, want 64 hex chars", len(secret.Token))
	}
	if !secret.ExpiresAt.After(time.Now()) {
		t.Fatalf("expiresAt %v not in the future", secret.ExpiresAt)
	}

	shared, err := svc.Resolve(context.Background(), secret.Token)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if shared.Status != StatusWaiting {
		t.Errorf("status = %q, want WAITING", shared.Status)
	}
	if shared.CurrentStep == nil || shared.CurrentStep.Title != "เจาะเลือด / ส่งตรวจแล็บ" {
		t.Fatalf("currentStep = %+v, want the LAB step in Thai", shared.CurrentStep)
	}
	if shared.CurrentStep.ServicePointName != "Laboratory" {
		t.Errorf("servicePointName = %q, want Laboratory", shared.CurrentStep.ServicePointName)
	}
	if !shared.ExpiresAt.Equal(secret.ExpiresAt) {
		t.Errorf("expiresAt = %v, want the link's expiry %v", shared.ExpiresAt, secret.ExpiresAt)
	}
}

// Every resolution failure must be the same error — unknown, expired, and
// revoked are indistinguishable to the caller (ADR-0011 §5).
func TestResolveFailuresAreIndistinguishable(t *testing.T) {
	repo := &fakeRepo{}
	journeys := &fakeJourneys{
		visit: journey.Visit{VisitID: "VISIT-SHARE-1", Status: journey.VisitActive},
		view:  journey.View{VisitID: "VISIT-SHARE-1", Status: journey.VisitActive},
	}
	svc := newTestService(repo, journeys)

	expired, err := svc.CreateLink(context.Background(), "VISIT-SHARE-1")
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	past := time.Now().Add(-time.Minute)
	expiredLink := repo.links[hashToken(expired.Token)]
	expiredLink.ExpiresAt = past
	repo.links[hashToken(expired.Token)] = expiredLink

	revoked, err := svc.CreateLink(context.Background(), "VISIT-SHARE-1")
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	revokedLink := repo.links[hashToken(revoked.Token)]
	revokedLink.RevokedAt = &past
	repo.links[hashToken(revoked.Token)] = revokedLink

	cases := map[string]string{
		"unknown token": "deadbeef",
		"expired token": expired.Token,
		"revoked token": revoked.Token,
	}
	want := ErrInvalidLink.Error()
	for name, token := range cases {
		_, err := svc.Resolve(context.Background(), token)
		if err == nil {
			t.Fatalf("%s: Resolve succeeded, want failure", name)
		}
		if apperr.KindOf(err) != apperr.KindUnauthorized {
			t.Errorf("%s: kind = %v, want unauthorized", name, apperr.KindOf(err))
		}
		if err.Error() != want {
			t.Errorf("%s: message = %q, want the single message %q", name, err.Error(), want)
		}
	}
}

// The 6th active link for one visit is refused with 409; links for other
// visits don't count against the cap, and revoking frees headroom.
func TestCreateLinkCap(t *testing.T) {
	repo := &fakeRepo{}
	journeys := &fakeJourneys{
		visit: journey.Visit{VisitID: "VISIT-SHARE-1", Status: journey.VisitActive},
		view:  journey.View{VisitID: "VISIT-SHARE-1", Status: journey.VisitActive},
	}
	svc := newTestService(repo, journeys)

	for i := 0; i < MaxActiveLinks; i++ {
		if _, err := svc.CreateLink(context.Background(), "VISIT-SHARE-1"); err != nil {
			t.Fatalf("link %d: %v", i+1, err)
		}
	}
	if _, err := svc.CreateLink(context.Background(), "VISIT-SHARE-1"); apperr.KindOf(err) != apperr.KindConflict {
		t.Fatalf("6th link: err = %v, want conflict", err)
	}

	// Another visit is unaffected…
	other := &fakeJourneys{
		visit: journey.Visit{VisitID: "VISIT-SHARE-2", Status: journey.VisitActive},
		view:  journey.View{VisitID: "VISIT-SHARE-2", Status: journey.VisitActive},
	}
	if _, err := newTestService(repo, other).CreateLink(context.Background(), "VISIT-SHARE-2"); err != nil {
		t.Fatalf("other visit: %v", err)
	}

	// …and revoking makes room again.
	if err := svc.RevokeAll(context.Background(), "VISIT-SHARE-1"); err != nil {
		t.Fatalf("RevokeAll: %v", err)
	}
	if _, err := svc.CreateLink(context.Background(), "VISIT-SHARE-1"); err != nil {
		t.Fatalf("after revoke: %v", err)
	}
}

// A missing visit is the journey module's 404, surfaced as-is.
func TestCreateLinkUnknownVisit(t *testing.T) {
	_, err := newTestService(&fakeRepo{}, &fakeJourneys{}).CreateLink(context.Background(), "VISIT-NOPE")
	if apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("err = %v, want not found", err)
	}
}

// ---- redaction (the regression test the AC asks for) --------------------

// BuildSharedJourney must not leak anything identifying. The assertions run
// against the marshalled JSON so a future field added to SharedJourney fails
// here the moment it carries sensitive text, not at a code review.
func TestSharedJourneyRedaction(t *testing.T) {
	view := journey.View{
		VisitID: "VISIT-002", PatientRef: "PATIENT-DEMO-002",
		PatientName: "สมหญิง รักษ์ดี", Status: journey.VisitActive, SyncedAt: time.Now(),
		Steps: []journey.StepView{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: journey.KindRegistration,
				Status: journey.StepCompleted, ServicePointID: str("SP-REG")},
			{StepKey: "CLINIC:MED:1", Sequence: 2, Kind: journey.KindClinic,
				ClinicCode: str("MED"), Round: num(1), OrderRefs: []string{"ORD-002"},
				Status: journey.StepStarted, ServicePointID: str("SP-CLINIC-MED"),
				ServicePoint: &servicepoint.ServicePoint{
					ID: "SP-CLINIC-MED", Code: "CLINIC:MED", Name: "อายุรกรรม (MED)",
					// Seed data exactly as migrated: an English floor name
					// with a numeric code (000007_hospital_map.up.sql).
					Place: newPlaceWithFloor("OPD-NS-01", "อาคารผู้ป่วยนอก", "2", "Upper Floor"),
				}},
			{StepKey: "CLINIC:MED:2", Sequence: 3, Kind: journey.KindClinic,
				ClinicCode: str("MED"), Round: num(2), Status: journey.StepPending},
		},
	}

	shared := BuildSharedJourney(view, time.Now().Add(time.Hour))
	if shared.Status != StatusInService {
		t.Fatalf("status = %q, want IN_SERVICE (a step is STARTED)", shared.Status)
	}
	if shared.CurrentStep == nil {
		t.Fatalf("currentStep missing")
	}
	if shared.CurrentStep.Title != "พบแพทย์" {
		t.Errorf("title = %q, want พบแพทย์", shared.CurrentStep.Title)
	}
	if shared.CurrentStep.ServicePointName != "อายุรกรรม" {
		t.Errorf("servicePointName = %q, want the code suffix stripped", shared.CurrentStep.ServicePointName)
	}
	if shared.CurrentStep.FloorName != "ชั้น 2" {
		t.Errorf("floorName = %q, want ชั้น 2 derived from the floor code, not the DB name", shared.CurrentStep.FloorName)
	}

	raw, err := json.Marshal(shared)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)
	for _, banned := range []string{
		"สมหญิง", "รักษ์ดี", // patient name
		"PATIENT-DEMO-002",         // patientRef
		"VISIT-002",                // visitId
		"CLINIC", "MED", "ORD-002", // stepKey, clinicCode, orderRef
		"SP-CLINIC-MED", "OPD-NS-01", // service point id, place id
	} {
		if strings.Contains(body, banned) {
			t.Errorf("shared JSON leaks %q: %s", banned, body)
		}
	}
}

// A completed visit answers DONE with no current step; a return-round clinic
// step titles as the follow-up visit, still without the clinic code.
func TestSharedJourneyCompletedAndFollowUp(t *testing.T) {
	done := journey.View{Completed: true, SyncedAt: time.Now()}
	if shared := BuildSharedJourney(done, time.Now()); shared.Status != StatusDone || shared.CurrentStep != nil {
		t.Errorf("completed visit = %s/%+v, want DONE with no step", shared.Status, shared.CurrentStep)
	}

	waiting := journey.View{
		SyncedAt:    time.Now(),
		Recommended: &journey.StepView{Kind: journey.KindClinic, Round: num(2), Status: journey.StepReady},
	}
	if shared := BuildSharedJourney(waiting, time.Now()); shared.CurrentStep == nil || shared.CurrentStep.Title != "กลับมาพบแพทย์" {
		t.Errorf("follow-up title = %+v, want กลับมาพบแพทย์", shared.CurrentStep)
	}
}

// The floor line comes from the code; an unmapped code falls back to the
// DB name rather than rendering a bare "ชั้น ".
func TestDisplayFloorName(t *testing.T) {
	if got := displayFloorName("1", "Ground Floor"); got != "ชั้น 1" {
		t.Errorf("displayFloorName(1, Ground Floor) = %q, want ชั้น 1", got)
	}
	if got := displayFloorName("", "Ground Floor"); got != "Ground Floor" {
		t.Errorf("displayFloorName(_, Ground Floor) = %q, want the name as fallback", got)
	}
}
