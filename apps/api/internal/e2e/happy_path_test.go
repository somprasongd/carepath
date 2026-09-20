// Package e2e hosts the happy-path end-to-end check (#40): one scenario
// walked across every real module on the CarePath side — the canonical HIS
// boundary (httpclient against a contract-faithful scripted HIS), the
// journey projection and planner, servicepoint/hospitalmap, the location
// fix, and navigation routing — all backed by postgres. Everything is real
// but the process boundary.
//
// The scripted HIS restates the #39 demo scenario (walk-in MED patient,
// chest X-ray ordered mid-visit) with its own stable identifiers, because
// apps/api cannot import the mock-his module without breaking its
// module-only docker build. Events are applied through
// journey.Service.ApplyHISEvent — the exact path the ingest poller drives —
// rather than through the poller's shared checkpoint, which CI's parallel
// test packages also contend on (the checkpoint itself is covered by the
// journey package's ingest e2e test).
package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/his/httpclient"
	"carepath/apps/api/internal/hospitalmap"
	hospitalmappostgres "carepath/apps/api/internal/hospitalmap/postgres"
	"carepath/apps/api/internal/journey"
	journeypostgres "carepath/apps/api/internal/journey/postgres"
	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/location/manual"
	locationpostgres "carepath/apps/api/internal/location/postgres"
	"carepath/apps/api/internal/navigation"
	navigationpostgres "carepath/apps/api/internal/navigation/postgres"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/servicepoint"
	servicepointpostgres "carepath/apps/api/internal/servicepoint/postgres"
)

const (
	happyVisit   = "VISIT-HAPPY"
	happyPatient = "PAT-HAPPY"
	happyOrder   = "ORD-HAPPY-XRAY"
	happyDrug    = "ORD-HAPPY-DRUG"
)

var happyOpenedAt = time.Date(2026, 9, 19, 9, 3, 0, 0, time.FixedZone("ICT", 7*60*60))

// scriptedHIS is the #39 demo scenario restated: it serves the two canonical
// read surfaces over HTTP and exposes the same demo mutations the mock HIS
// console drives (order lifecycle, drug order). The test mutates state and
// appends the matching canonical events, exactly what Mock HIS does.
type scriptedHIS struct {
	mu     sync.Mutex
	visit  his.Visit
	events []his.Event
	seq    int
}

func newScriptedHIS() *scriptedHIS {
	s := &scriptedHIS{
		visit: his.Visit{
			VisitID: happyVisit, PatientRef: happyPatient, PatientName: "สมหญิง รักษ์ดี",
			VisitType: his.VisitTypeWalkin, Status: his.VisitActive,
			Clinics:  []his.Clinic{{Code: "MED", Name: "อายุรกรรม"}},
			OpenedAt: happyOpenedAt,
			Orders: []his.Order{
				{OrderRef: happyOrder, OrderType: his.OrderTypeXray, OrderName: "Chest X-ray",
					OrderedByClinic: "MED", OrderedAt: happyOpenedAt.Add(time.Minute), Status: his.OrderPlaced},
			},
		},
	}
	s.append(his.EventVisitOpened, map[string]any{
		"patientName": s.visit.PatientName, "visitType": s.visit.VisitType, "clinics": s.visit.Clinics,
	})
	s.append(his.EventOrderPlaced, s.orderPayload(0))
	return s
}

func (s *scriptedHIS) orderPayload(i int) map[string]any {
	o := s.visit.Orders[i]
	return map[string]any{
		"orderRef": o.OrderRef, "orderType": o.OrderType, "orderName": o.OrderName,
		"orderedByClinic": o.OrderedByClinic, "orderedAt": o.OrderedAt,
	}
}

func (s *scriptedHIS) append(eventType string, payload map[string]any) {
	s.seq++
	s.events = append(s.events, his.Event{
		EventID:    happyEventID(s.seq),
		OccurredAt: happyOpenedAt.Add(time.Duration(s.seq) * time.Minute),
		VisitID:    happyVisit, PatientRef: happyPatient,
		Type: eventType, Payload: payload,
	})
}

func happyEventID(seq int) string {
	return fmt.Sprintf("EVT-HAPPY-%06d", seq)
}

// resultOrder walks the X-ray through performed → resulted, the facts that
// complete a diagnostic step and make the return clinic round actionable.
func (s *scriptedHIS) resultOrder() {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := &s.visit.Orders[0]
	now := time.Now().UTC()
	o.Status = his.OrderPerformed
	o.PerformedAt = &now
	s.append(his.EventOrderPerformed, map[string]any{"orderRef": o.OrderRef})
	o.Status = his.OrderResulted
	o.ResultedAt = &now
	s.append(his.EventOrderResulted, map[string]any{"orderRef": o.OrderRef})
}

// startEncounter is the clinic calling the patient in — the console's "Call
// patient in" button. CarePath opens the clinic's next actionable round with
// it, implicitly finishing a round still in progress.
func (s *scriptedHIS) startEncounter() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.append(his.EventEncounterStarted, map[string]any{"clinicCode": "MED", "startedAt": time.Now().UTC()})
}

// placeDrug is the doctor prescribing at the return visit — the fact that
// adds the pharmacy step to the plan.
func (s *scriptedHIS) placeDrug() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.visit.Orders = append(s.visit.Orders, his.Order{
		OrderRef: happyDrug, OrderType: his.OrderTypeDrug, OrderName: "Paracetamol",
		OrderedByClinic: "MED", OrderedAt: time.Now().UTC(), Status: his.OrderPlaced,
	})
	s.append(his.EventOrderPlaced, s.orderPayload(len(s.visit.Orders)-1))
}

func (s *scriptedHIS) server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/visits/"+happyVisit:
			_ = json.NewEncoder(w).Encode(s.visit)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/events":
			after := r.URL.Query().Get("after")
			page := his.EventPage{}
			for _, e := range s.events {
				if after == "" || e.EventID > after {
					page.Events = append(page.Events, e)
				}
			}
			if n := len(page.Events); n > 0 {
				page.NextAfter = page.Events[n-1].EventID
			}
			_ = json.NewEncoder(w).Encode(page)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestHappyPath(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (needs migrations applied — see CI)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	ctx := context.Background()

	// Re-runnable against the deterministic scenario: clear this visit's
	// rows so a previous run (or a retry) cannot leak state (#39 AC5).
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := database.Querier(ctx).Exec(ctx, sql, args...); err != nil {
			t.Fatalf("setup exec: %v", err)
		}
	}
	exec(`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, happyVisit)
	exec(`DELETE FROM carepath.his_applied_event WHERE visit_id = $1`, happyVisit)
	exec(`DELETE FROM carepath.journey_command_audit WHERE visit_id = $1`, happyVisit)
	exec(`DELETE FROM carepath.location_observation WHERE visit_id = $1`, happyVisit)

	upstream := newScriptedHIS()
	server := upstream.server(t)
	defer server.Close()

	hisClient := httpclient.New(server.URL, server.Client())
	hospitalMap := hospitalmap.NewService(hospitalmappostgres.New(database))
	servicePoints := servicepoint.NewService(servicepointpostgres.New(database), hospitalMap)
	journeys := journey.NewService(hisClient, servicePoints, journeypostgres.New(database), database)
	navigationGraph := navigation.NewService(navigationpostgres.New(database), servicePoints)
	locations, err := location.NewService(locationpostgres.New(database), navigationGraph, manual.New())
	if err != nil {
		t.Fatalf("location service: %v", err)
	}

	// Patient entry: the two seed facts project the opening plan —
	// registration done, the clinic visit recommended, X-ray next.
	applyAll(t, journeys, upstream)

	// AC: verify current/next step.
	view := getJourney(t, journeys)
	assertStep(t, view, "REGISTRATION", "COMPLETED")
	assertStep(t, view, "CLINIC:MED:1", "READY")
	assertStep(t, view, "XRAY:1", "READY")
	assertStep(t, view, "CASHIER", "PENDING")
	if view.Recommended == nil || view.Recommended.StepKey != "CLINIC:MED:1" {
		t.Fatalf("recommended = %v, want CLINIC:MED:1", stepKeyOf(view.Recommended))
	}
	if len(view.Actionable) != 2 {
		t.Fatalf("actionable = %d steps, want clinic (current) + x-ray (next)", len(view.Actionable))
	}

	// Current location: a manual fix at the reception counter — the same
	// canonical observation the patient screen polls.
	fix, err := locations.Report(ctx, happyVisit, location.SourceManual, "I-1301/node-reception")
	if err != nil {
		t.Fatalf("report location: %v", err)
	}
	if fix.NodeID != "I-1301/node-reception" {
		t.Fatalf("fix node = %s, want I-1301/node-reception", fix.NodeID)
	}

	// AC: verify route response — from the current location to the
	// recommended step's service point, exactly what the navigate screen
	// requests.
	route := routeToRecommended(t, navigationGraph, "I-1301/node-reception", view)
	if route.Nodes[0].ID != "I-1301/node-reception" {
		t.Fatalf("route origin = %s, want the location's node", route.Nodes[0].ID)
	}
	if last := route.Nodes[len(route.Nodes)-1]; last.ID != "I-1301/node-opd-ns" {
		t.Fatalf("route destination = %s, want the OPD entry node I-1301/node-opd-ns", last.ID)
	}
	if route.TotalDistance <= 0 {
		t.Fatalf("route distance = %v, want a walkable path", route.TotalDistance)
	}

	// The clinic calls the patient in (HIS fact, the console's "Call patient
	// in") — this is what spawns the return round while diagnostics are
	// pending (ADR-0009 §3/§4). It also changes the patient's next
	// destination: the in-progress consult is no longer actionable, so the
	// X-ray becomes the recommendation.
	upstream.startEncounter()
	applyAll(t, journeys, upstream)
	view = getJourney(t, journeys)
	assertStep(t, view, "CLINIC:MED:1", "STARTED")
	assertStep(t, view, "CLINIC:MED:2", "WAITING")
	if view.Recommended == nil || view.Recommended.StepKey != "XRAY:1" {
		t.Fatalf("recommended after round opens = %v, want XRAY:1", stepKeyOf(view.Recommended))
	}
	route = routeToRecommended(t, navigationGraph, "I-1301/node-reception", view)
	if last := route.Nodes[len(route.Nodes)-1]; last.ID != "I-1301/node-xray" {
		t.Fatalf("destination after round opens = %s, want the X-ray entry node I-1301/node-xray", last.ID)
	}

	// AC: verify service completion changes the destination. The X-ray is
	// performed and resulted (HIS facts) → the return round becomes the
	// recommendation and the route's destination flips back to the clinic.
	upstream.resultOrder()
	applyAll(t, journeys, upstream)
	view = getJourney(t, journeys)
	assertStep(t, view, "XRAY:1", "COMPLETED")
	assertStep(t, view, "CLINIC:MED:2", "READY")
	if view.Recommended == nil || view.Recommended.StepKey != "CLINIC:MED:2" {
		t.Fatalf("recommended after x-ray resulted = %v, want CLINIC:MED:2", stepKeyOf(view.Recommended))
	}
	route = routeToRecommended(t, navigationGraph, "I-1301/node-reception", view)
	if last := route.Nodes[len(route.Nodes)-1]; last.ID != "I-1301/node-opd-ns" {
		t.Fatalf("destination after x-ray resulted = %s, want the OPD entry node again", last.ID)
	}

	// The clinic calls the patient back in (HIS fact): round 1 — open since
	// the first call — is implicitly finished and the return round starts.
	// The doctor prescribes at it → the pharmacy tail appears.
	upstream.startEncounter()
	applyAll(t, journeys, upstream)
	view = getJourney(t, journeys)
	assertStep(t, view, "CLINIC:MED:1", "COMPLETED")
	assertStep(t, view, "CLINIC:MED:2", "STARTED")
	upstream.placeDrug()
	applyAll(t, journeys, upstream)
	assertStep(t, getJourney(t, journeys), "PHARMACY", "PENDING")

	// The staff console stays a working override surface: a staff command
	// (not an HIS fact) wraps up the return round, which opens the cashier.
	transition(t, journeys, "CLINIC:MED:2", journey.CommandToCompleted)
	view = getJourney(t, journeys)
	assertStep(t, view, "CLINIC:MED:2", "COMPLETED")
	assertStep(t, view, "CASHIER", "READY")
	assertStep(t, view, "PHARMACY", "PENDING")
}

// applyAll feeds the scripted HIS's events through the same application
// path the ingest poller drives. Already-applied events are skipped, so it
// is safe to call again after each mutation.
func applyAll(t *testing.T, journeys journey.Service, upstream *scriptedHIS) {
	t.Helper()
	upstream.mu.Lock()
	events := append([]his.Event(nil), upstream.events...)
	upstream.mu.Unlock()
	for _, e := range events {
		if err := journeys.ApplyHISEvent(context.Background(), e); err != nil {
			t.Fatalf("apply %s: %v", e.EventID, err)
		}
	}
}

func getJourney(t *testing.T, journeys journey.Service) journey.View {
	t.Helper()
	view, err := journeys.GetJourney(context.Background(), happyVisit)
	if err != nil {
		t.Fatalf("get journey: %v", err)
	}
	return view
}

func assertStep(t *testing.T, view journey.View, stepKey, status string) {
	t.Helper()
	for _, s := range view.Steps {
		if s.StepKey == stepKey {
			if s.Status != status {
				t.Fatalf("%s status = %s, want %s", stepKey, s.Status, status)
			}
			return
		}
	}
	t.Fatalf("step %s not in plan: %v", stepKey, view.Steps)
}

func stepKeyOf(s *journey.StepView) string {
	if s == nil {
		return "<none>"
	}
	return s.StepKey
}

// routeToRecommended resolves the route the navigate screen would show:
// from the current location's node to the recommended step's service
// point code.
func routeToRecommended(t *testing.T, nav navigation.Service, fromNodeID string, view journey.View) navigation.Route {
	t.Helper()
	if view.Recommended == nil || view.Recommended.ServicePoint == nil {
		t.Fatalf("recommended step has no service point: %+v", view.Recommended)
	}
	route, err := nav.RouteToServicePoint(context.Background(), fromNodeID,
		view.Recommended.ServicePoint.Code, navigation.RouteOptions{})
	if err != nil {
		t.Fatalf("route to %s: %v", view.Recommended.ServicePoint.Code, err)
	}
	if len(route.Nodes) < 2 {
		t.Fatalf("route to %s has %d nodes, want a real path", view.Recommended.ServicePoint.Code, len(route.Nodes))
	}
	return route
}

func transition(t *testing.T, journeys journey.Service, stepKey, to string) {
	t.Helper()
	_, err := journeys.TransitionStep(context.Background(), happyVisit, stepKey,
		journey.TransitionCommand{CommandID: "e2e-" + stepKey + "-" + to, To: to},
		"e2e-test", journey.Actor{UserID: "e2e-user", Username: "e2e"})
	if err != nil {
		t.Fatalf("transition %s -> %s: %v", stepKey, to, err)
	}
}
