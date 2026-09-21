package floorplan

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
)

// storeRepo is an in-memory stand-in for the append-only table: inserting
// content a floor already has is a no-op, exactly as ON CONFLICT DO NOTHING
// makes it in Postgres.
type storeRepo struct {
	plans map[string]Stored // planID -> row
	svgs  map[string]string // planID -> normalized
	raws  map[string]string // planID -> as uploaded
}

func newStoreRepo() *storeRepo {
	return &storeRepo{plans: map[string]Stored{}, svgs: map[string]string{}, raws: map[string]string{}}
}

func (r *storeRepo) Insert(_ context.Context, plan Stored, svg, svgRaw string) error {
	if _, exists := r.plans[plan.PlanID]; exists {
		return nil
	}
	r.plans[plan.PlanID] = plan
	r.svgs[plan.PlanID] = svg
	r.raws[plan.PlanID] = svgRaw
	return nil
}

func (r *storeRepo) Get(_ context.Context, floorID, planID string) (Stored, error) {
	plan, ok := r.plans[planID]
	if !ok || plan.FloorID != floorID {
		return Stored{}, ErrPlanNotFound
	}
	return plan, nil
}

func (r *storeRepo) ListByFloor(_ context.Context, floorID string) ([]Stored, error) {
	var out []Stored
	for _, plan := range r.plans {
		if plan.FloorID == floorID {
			out = append(out, plan)
		}
	}
	return out, nil
}

func (r *storeRepo) Raw(_ context.Context, floorID, planID string) (string, error) {
	if plan, ok := r.plans[planID]; !ok || plan.FloorID != floorID {
		return "", ErrPlanNotFound
	}
	return r.raws[planID], nil
}

func (r *storeRepo) SVG(_ context.Context, floorID, sha string) (string, error) {
	for id, plan := range r.plans {
		if plan.FloorID == floorID && plan.SHA256 == sha {
			return r.svgs[id], nil
		}
	}
	return "", ErrPlanNotFound
}

func (r *storeRepo) Digests(_ context.Context, planIDs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range planIDs {
		if plan, ok := r.plans[id]; ok {
			out[id] = plan.SHA256
		}
	}
	return out, nil
}

// mapStub is the hospitalmap side: one floor, its places, and the pointer
// SetPlan moves.
type mapStub struct {
	hospitalmap.Service
	floor  hospitalmap.Floor
	places []hospitalmap.Place
	set    []string // planIDs SetPlan was called with, in order
}

func (m *mapStub) GetFloor(_ context.Context, floorID string) (hospitalmap.Floor, error) {
	if floorID != m.floor.ID {
		return hospitalmap.Floor{}, hospitalmap.ErrFloorNotFound
	}
	return m.floor, nil
}

func (m *mapStub) ListPlaces(context.Context) ([]hospitalmap.Place, error) { return m.places, nil }

func (m *mapStub) SetPlan(_ context.Context, floorID, viewBox, planID string) error {
	m.floor.ViewBox = &viewBox
	m.floor.ActivePlanID = &planID
	m.set = append(m.set, planID)
	return nil
}

// graphStub carries globally-unique node ids, the form the graph actually
// stores them in.
type graphStub struct {
	navigation.Service
	nodes []navigation.NavNode
}

func (g *graphStub) ListNodes(context.Context) ([]navigation.NavNode, error) { return g.nodes, nil }

// directTx runs fn without a transaction — the service's boundary is what is
// under test here, not Postgres's.
type directTx struct{}

func (directTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func newUploadService(t *testing.T, floor hospitalmap.Floor, places []hospitalmap.Place, nodes []navigation.NavNode) (Service, *storeRepo, *mapStub) {
	t.Helper()
	repo := newStoreRepo()
	maps := &mapStub{floor: floor, places: places}
	return NewService(repo, maps, &graphStub{nodes: nodes}, directTx{}), repo, maps
}

func groundFloor() hospitalmap.Floor {
	return hospitalmap.Floor{ID: "I-1301", BuildingID: "BLD-I13", Code: "1", Name: "Ground Floor", LevelOrder: 1}
}

func shippedGroundPlan(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "packages", "floorplans", "floors", "i-1301-ground.svg")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return src
}

func TestUploadStoresAndPointsTheFloorAtThePlan(t *testing.T) {
	svc, repo, maps := newUploadService(t, groundFloor(), nil, nil)

	stored, err := svc.Upload(context.Background(), "I-1301", shippedGroundPlan(t), "USR-1")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if stored.ViewBox != "0 0 1600 900" {
		t.Errorf("ViewBox = %q, want the plan's own space", stored.ViewBox)
	}
	if stored.CreatedBy == nil || *stored.CreatedBy != "USR-1" {
		t.Errorf("CreatedBy = %v, want the acting admin", stored.CreatedBy)
	}
	if len(maps.set) != 1 || maps.set[0] != stored.PlanID {
		t.Errorf("SetPlan calls = %v, want one naming %s", maps.set, stored.PlanID)
	}

	// What is stored is the normalizer's output, not the uploaded bytes.
	svg, err := repo.SVG(context.Background(), "I-1301", stored.SHA256)
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	if svg == string(shippedGroundPlan(t)) {
		t.Error("the stored plan is the uploaded bytes verbatim")
	}
	if raw, _ := repo.Raw(context.Background(), "I-1301", stored.PlanID); raw != string(shippedGroundPlan(t)) {
		t.Error("the raw column is not the file as uploaded")
	}
}

// Plan ids are derived from content, so an admin who uploads the same file
// twice gets the row that is already there rather than a duplicate.
func TestUploadOfUnchangedContentIsIdempotent(t *testing.T) {
	svc, repo, _ := newUploadService(t, groundFloor(), nil, nil)
	src := shippedGroundPlan(t)

	first, err := svc.Upload(context.Background(), "I-1301", src, "USR-1")
	if err != nil {
		t.Fatalf("first Upload: %v", err)
	}
	second, err := svc.Upload(context.Background(), "I-1301", src, "USR-2")
	if err != nil {
		t.Fatalf("second Upload: %v", err)
	}
	if first.PlanID != second.PlanID {
		t.Errorf("plan ids %s and %s, want one row", first.PlanID, second.PlanID)
	}
	if len(repo.plans) != 1 {
		t.Errorf("stored %d plans, want 1", len(repo.plans))
	}
}

// The floor's coordinate space is fixed by its first plan. This is the
// failure with no symptom — nothing errors, every pin and route just moves —
// so the upload has to be the thing that refuses.
func TestUploadHoldsTheFloorsCoordinateSpace(t *testing.T) {
	space := "0 0 800 450"
	floor := groundFloor()
	floor.ViewBox = &space
	svc, _, maps := newUploadService(t, floor, nil, nil)

	_, err := svc.Upload(context.Background(), "I-1301", shippedGroundPlan(t), "USR-1")
	if err == nil {
		t.Fatal("Upload accepted a plan drawn in a different coordinate space")
	}
	if apperr.KindOf(err) != apperr.KindInvalid {
		t.Errorf("KindOf(err) = %v, want KindInvalid", apperr.KindOf(err))
	}
	if len(maps.set) != 0 {
		t.Error("the floor was re-pointed despite the upload failing")
	}
}

// A place or node the plan does not draw is reported, not refused — the
// patient app already degrades such a place to "not routable yet", and
// refusing a whole floor over one undrawn room would be worse.
func TestUploadReportsModelGapsWithoutRefusing(t *testing.T) {
	places := []hospitalmap.Place{
		{ID: "REG-01", FloorID: "I-1301"},
		{ID: "GHOST-01", FloorID: "I-1301"},
		{ID: "LAB-01", FloorID: "I-1302"}, // another floor: not this plan's business
	}
	nodes := []navigation.NavNode{
		{ID: "I-1301/node-reception", FloorID: "I-1301"},
		{ID: "I-1301/node-nowhere", FloorID: "I-1301"},
		{ID: "I-1302/node-lift", FloorID: "I-1302"},
	}
	svc, _, _ := newUploadService(t, groundFloor(), places, nodes)

	stored, err := svc.Upload(context.Background(), "I-1301", shippedGroundPlan(t), "USR-1")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	got := map[string]string{}
	for _, w := range stored.Warnings {
		got[w.Ref] = w.Code
	}
	if got["GHOST-01"] != WarnPlaceMissing {
		t.Errorf("GHOST-01 warning = %q, want %s", got["GHOST-01"], WarnPlaceMissing)
	}
	// Graph ids are floor-prefixed but SVG ids are floor-local; the service
	// compares them in the SVG's form, so node-reception must not warn.
	if _, warned := got["node-reception"]; warned {
		t.Error("node-reception warned, so the floor prefix was not stripped before comparing")
	}
	if got["node-nowhere"] != WarnNodeMissing {
		t.Errorf("node-nowhere warning = %q, want %s", got["node-nowhere"], WarnNodeMissing)
	}
	// The other floor's place and node are not this plan's business.
	for _, ref := range []string{"LAB-01", "node-lift"} {
		if _, warned := got[ref]; warned {
			t.Errorf("%s warned, but it is on another floor", ref)
		}
	}
}

// Health must reflect the map model as it stands now, not as it stood at
// upload time — the whole reason it exists rather than just reading
// Stored.Warnings.
func TestHealthReflectsPlacesAddedAfterUpload(t *testing.T) {
	places := []hospitalmap.Place{{ID: "REG-01", FloorID: "I-1301"}}
	svc, _, maps := newUploadService(t, groundFloor(), places, nil)
	ctx := context.Background()

	if _, err := svc.Upload(ctx, "I-1301", shippedGroundPlan(t), "USR-1"); err != nil {
		t.Fatalf("Upload: %v", err)
	}

	warnAboutNew := func(warnings []Warning) bool {
		for _, w := range warnings {
			if w.Code == WarnPlaceMissing && w.Ref == "NEW-01" {
				return true
			}
		}
		return false
	}

	before, err := svc.Health(ctx, "I-1301")
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if warnAboutNew(before) {
		t.Fatalf("Health warned about NEW-01 before it existed: %+v", before)
	}

	// A place is added to the floor after the plan was accepted — the plan
	// itself has not changed, but it no longer covers everything the model
	// knows about.
	maps.places = append(maps.places, hospitalmap.Place{ID: "NEW-01", FloorID: "I-1301"})

	after, err := svc.Health(ctx, "I-1301")
	if err != nil {
		t.Fatalf("Health after a place was added: %v", err)
	}
	if !warnAboutNew(after) {
		t.Errorf("Health = %+v, want a PLACE_MISSING warning for NEW-01", after)
	}
}

func TestHealthWithNoActivePlan(t *testing.T) {
	svc, _, _ := newUploadService(t, groundFloor(), nil, nil)

	warnings, err := svc.Health(context.Background(), "I-1301")
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Health with no active plan = %+v, want none", warnings)
	}
}

func TestHealthOnUnknownFloor(t *testing.T) {
	svc, _, _ := newUploadService(t, groundFloor(), nil, nil)

	if _, err := svc.Health(context.Background(), "I-9999"); apperr.KindOf(err) != apperr.KindNotFound {
		t.Errorf("KindOf(err) = %v, want KindNotFound", apperr.KindOf(err))
	}
}

func TestUploadRejectsHostileContentBeforeStoringAnything(t *testing.T) {
	svc, repo, maps := newUploadService(t, groundFloor(), nil, nil)
	hostile := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900">` +
		`<g data-floor="I-1301"><g id="route-layer"/><script>alert(1)</script></g></svg>`)

	if _, err := svc.Upload(context.Background(), "I-1301", hostile, "USR-1"); err == nil {
		t.Fatal("Upload accepted a plan carrying a script element")
	}
	if len(repo.plans) != 0 || len(maps.set) != 0 {
		t.Error("a rejected upload left state behind")
	}
}

func TestUploadToAnUnknownFloor(t *testing.T) {
	svc, _, _ := newUploadService(t, groundFloor(), nil, nil)

	_, err := svc.Upload(context.Background(), "I-9999", shippedGroundPlan(t), "USR-1")
	if apperr.KindOf(err) != apperr.KindNotFound {
		t.Errorf("KindOf(err) = %v, want KindNotFound", apperr.KindOf(err))
	}
}

// Plans are append-only, so rolling back is a pointer move — the earlier
// drawing was never anywhere else.
func TestActivateMovesThePointerBack(t *testing.T) {
	svc, _, maps := newUploadService(t, groundFloor(), nil, nil)
	ctx := context.Background()

	first, err := svc.Upload(ctx, "I-1301", shippedGroundPlan(t), "USR-1")
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	// A second, different drawing on the same floor and coordinate space.
	edited := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1600 900">` +
		`<g data-floor="I-1301"><g id="route-layer"/><rect data-place-id="REG-01"/></g></svg>`)
	second, err := svc.Upload(ctx, "I-1301", edited, "USR-1")
	if err != nil {
		t.Fatalf("second Upload: %v", err)
	}
	if first.PlanID == second.PlanID {
		t.Fatal("two different drawings landed on one plan id")
	}

	back, err := svc.Activate(ctx, "I-1301", first.PlanID)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if back.PlanID != first.PlanID {
		t.Errorf("Activate returned %s, want %s", back.PlanID, first.PlanID)
	}
	if last := maps.set[len(maps.set)-1]; last != first.PlanID {
		t.Errorf("floor points at %s, want %s", last, first.PlanID)
	}

	// Rolling back did not remove the newer plan: it is still there to go
	// forward to.
	if _, err := svc.Activate(ctx, "I-1301", second.PlanID); err != nil {
		t.Errorf("the rolled-back-from plan is gone: %v", err)
	}
}

func TestActivateUnknownPlan(t *testing.T) {
	svc, _, _ := newUploadService(t, groundFloor(), nil, nil)

	if _, err := svc.Activate(context.Background(), "I-1301", "FP-NOPE"); err != ErrPlanNotFound {
		t.Errorf("err = %v, want ErrPlanNotFound", err)
	}
}
