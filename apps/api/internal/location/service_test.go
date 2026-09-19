package location_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"carepath/apps/api/internal/location"
	locationmock "carepath/apps/api/internal/location/mock"
	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
)

// fakeRepo is an in-memory location.Repo: Latest returns the last recorded
// observation per visit.
type fakeRepo struct {
	recorded []location.Observation
}

func (f *fakeRepo) Record(_ context.Context, obs location.Observation) error {
	f.recorded = append(f.recorded, obs)
	return nil
}

func (f *fakeRepo) Latest(_ context.Context, visitID string) (location.Observation, error) {
	for i := len(f.recorded) - 1; i >= 0; i-- {
		if f.recorded[i].VisitID == visitID {
			return f.recorded[i], nil
		}
	}
	return location.Observation{}, location.ErrNoLocation
}

// fakeNavigation implements just GetNode: every id exists on I-1301 with the
// zone from a lookup table, except NotFoundNode.
type fakeNavigation struct {
	gotNodeID string
}

const notFoundNode = "I-1301/node-missing"

var nodeZones = map[string]*string{
	"I-1301/node-reception": ptr("PUBLIC"),
	"I-1301/node-lift":      nil,
}

func ptr(s string) *string        { return &s }
func floatPtr(f float64) *float64 { return &f }

func (f *fakeNavigation) GetNode(_ context.Context, nodeID string) (navigation.NavNode, error) {
	f.gotNodeID = nodeID
	if nodeID == notFoundNode {
		return navigation.NavNode{}, navigation.ErrNodeNotFound
	}
	if _, ok := nodeZones[nodeID]; !ok {
		return navigation.NavNode{}, navigation.ErrNodeNotFound
	}
	return navigation.NavNode{ID: nodeID, FloorID: "I-1301", Zone: nodeZones[nodeID]}, nil
}

func (f *fakeNavigation) ListNodes(context.Context) ([]navigation.NavNode, error) { return nil, nil }
func (f *fakeNavigation) ListEdges(context.Context) ([]navigation.NavEdge, error) { return nil, nil }

// Route is unused by these tests — location.Service never calls it — but is
// required to satisfy navigation.Service (#27 added it after this fake was
// written).
func (f *fakeNavigation) Route(context.Context, string, string, navigation.RouteOptions) (navigation.Route, error) {
	return navigation.Route{}, nil
}

func newService(t *testing.T, providers ...location.Provider) (location.Service, *fakeRepo, *fakeNavigation) {
	t.Helper()
	repo := &fakeRepo{}
	nav := &fakeNavigation{}
	svc, err := location.NewService(repo, nav, providers...)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc, repo, nav
}

// The core ADR-0004 property: resolving a fix goes through the provider for
// translation only — the service canonicalizes the result against the
// navigation graph (floor and zone from the node, source from registration,
// observed-at stamped when the provider had none).
func TestResolveCanonicalizes(t *testing.T) {
	observedAt := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	provider := locationmock.New(location.SourceZigbee, "I-1301/node-reception")
	provider.Fix.ObservedAt = observedAt
	confidence := 0.9
	provider.Fix.Confidence = &confidence
	svc, _, nav := newService(t, provider)

	obs, err := svc.Resolve(context.Background(), location.SourceZigbee, `{"zone":"OPD","rssi":-62}`)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if obs.NodeID != "I-1301/node-reception" || obs.FloorID != "I-1301" {
		t.Fatalf("obs = %+v, want node I-1301/node-reception on floor I-1301", obs)
	}
	if obs.Zone == nil || *obs.Zone != "PUBLIC" {
		t.Fatalf("obs.Zone = %v, want PUBLIC from the graph node", obs.Zone)
	}
	if obs.Source != location.SourceZigbee {
		t.Fatalf("obs.Source = %q, want ZIGBEE", obs.Source)
	}
	if !obs.ObservedAt.Equal(observedAt) {
		t.Fatalf("obs.ObservedAt = %v, want the provider timestamp %v", obs.ObservedAt, observedAt)
	}
	if obs.Confidence == nil || *obs.Confidence != 0.9 {
		t.Fatalf("obs.Confidence = %v, want 0.9", obs.Confidence)
	}
	if nav.gotNodeID != "I-1301/node-reception" {
		t.Fatalf("canonicalization did not look up the node (got %q)", nav.gotNodeID)
	}
	// The raw fix is opaque to the service — it must reach the provider
	// untouched, protocol and all.
	if len(provider.Raws) != 1 || provider.Raws[0] != `{"zone":"OPD","rssi":-62}` {
		t.Fatalf("provider.Raws = %v, want the raw fix passed through", provider.Raws)
	}
}

func TestResolveDefaults(t *testing.T) {
	svc, _, _ := newService(t, locationmock.New(location.SourceManual, "I-1301/node-lift"))

	before := time.Now()
	obs, err := svc.Resolve(context.Background(), location.SourceManual, "I-1301/node-lift")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if obs.Zone != nil {
		t.Fatalf("obs.Zone = %v, want nil for a zoneless node", obs.Zone)
	}
	if obs.Confidence != nil {
		t.Fatalf("obs.Confidence = %v, want nil for an exact source", obs.Confidence)
	}
	if obs.ObservedAt.Before(before) {
		t.Fatalf("obs.ObservedAt = %v, want the service stamp (>= %v)", obs.ObservedAt, before)
	}
}

func TestResolveUnknownSource(t *testing.T) {
	svc, _, _ := newService(t, locationmock.New(location.SourceManual, "I-1301/node-lift"))

	if _, err := svc.Resolve(context.Background(), location.SourceQR, "anything"); !errors.Is(err, location.ErrUnknownSource) {
		t.Fatalf("error = %v, want ErrUnknownSource", err)
	}
}

func TestResolveProviderFailure(t *testing.T) {
	boom := apperr.New(apperr.KindUpstream, "positioning service unreachable")
	provider := locationmock.New(location.SourceZigbee, "I-1301/node-reception")
	provider.Err = boom
	svc, _, _ := newService(t, provider)

	if _, err := svc.Resolve(context.Background(), location.SourceZigbee, "fix"); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want the provider error passed through", err)
	}
}

func TestResolveUnknownNode(t *testing.T) {
	svc, _, _ := newService(t, locationmock.New(location.SourceManual, notFoundNode))

	_, err := svc.Resolve(context.Background(), location.SourceManual, notFoundNode)
	if !errors.Is(err, location.ErrUnknownNode) {
		t.Fatalf("error = %v, want ErrUnknownNode", err)
	}
	if apperr.KindOf(err) != apperr.KindInvalid {
		t.Fatalf("kind = %v, want Invalid", apperr.KindOf(err))
	}
}

func TestResolveInvalidConfidence(t *testing.T) {
	provider := locationmock.New(location.SourceZigbee, "I-1301/node-reception")
	provider.Fix.Confidence = floatPtr(1.5)
	svc, _, _ := newService(t, provider)

	if _, err := svc.Resolve(context.Background(), location.SourceZigbee, "fix"); apperr.KindOf(err) != apperr.KindInvalid {
		t.Fatalf("error = %v, want Invalid for out-of-range confidence", err)
	}
}

func TestReportRecordsCurrentLocation(t *testing.T) {
	svc, repo, _ := newService(t, locationmock.New(location.SourceQR, "I-1301/node-reception"))

	obs, err := svc.Report(context.Background(), "VISIT-001", location.SourceQR, "https://carepath.example/location/reception")
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if obs.VisitID != "VISIT-001" {
		t.Fatalf("obs.VisitID = %q, want VISIT-001", obs.VisitID)
	}
	if len(repo.recorded) != 1 || repo.recorded[0] != obs {
		t.Fatalf("repo.recorded = %+v, want exactly the returned observation", repo.recorded)
	}

	current, err := svc.Current(context.Background(), "VISIT-001")
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if current.NodeID != obs.NodeID || current.Source != location.SourceQR {
		t.Fatalf("Current = %+v, want the reported observation", current)
	}
}

func TestReportRequiresVisit(t *testing.T) {
	svc, _, _ := newService(t, locationmock.New(location.SourceQR, "I-1301/node-reception"))

	if _, err := svc.Report(context.Background(), "", location.SourceQR, "fix"); apperr.KindOf(err) != apperr.KindInvalid {
		t.Fatalf("error = %v, want Invalid for empty visitId", err)
	}
}

func TestCurrentWithoutLocation(t *testing.T) {
	svc, _, _ := newService(t, locationmock.New(location.SourceQR, "I-1301/node-reception"))

	_, err := svc.Current(context.Background(), "VISIT-NONE")
	if !errors.Is(err, location.ErrNoLocation) {
		t.Fatalf("error = %v, want ErrNoLocation", err)
	}
	if apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("kind = %v, want NotFound", apperr.KindOf(err))
	}
}

func TestSourcesListsRegistration(t *testing.T) {
	svc, _, _ := newService(t,
		locationmock.New(location.SourceZigbee, "I-1301/node-reception"),
		locationmock.New(location.SourceManual, "I-1301/node-lift"),
	)

	sources := svc.Sources()
	if len(sources) != 2 || sources[0] != location.SourceManual || sources[1] != location.SourceZigbee {
		t.Fatalf("Sources() = %v, want [MANUAL ZIGBEE]", sources)
	}
}

func TestNewServiceRejectsDuplicateSource(t *testing.T) {
	_, err := location.NewService(&fakeRepo{}, &fakeNavigation{},
		locationmock.New(location.SourceQR, "I-1301/node-reception"),
		locationmock.New(location.SourceQR, "I-1301/node-lift"),
	)
	if apperr.KindOf(err) != apperr.KindInternal {
		t.Fatalf("error = %v, want Internal for duplicate provider source", err)
	}
}
