package journey

import (
	"testing"
	"time"

	"carepath/apps/api/internal/his"
)

// The planning-rules descriptor is the planner describing itself (#100), so
// these tests pin the derivation, not a snapshot: the phases must be the
// phase constants, the order-type map must be diagnosticKindByOrderType, and
// the examples must equal what the real Plan returns for the same facts —
// hardcoding the JSON away would fail here.
func TestPlanningRulesPhasesMatchPlannerConstants(t *testing.T) {
	phases := PlanningRulesDescriptor().Phases
	want := []struct {
		number int
		key    string
	}{
		{phaseRegistration, PhaseKeyRegistration},
		{phasePreVisit, PhaseKeyPreVisit},
		{phaseClinic, PhaseKeyClinic},
		{phaseMidVisit, PhaseKeyMidVisit},
		{phaseReturnClinic, PhaseKeyReturnClinic},
		{phaseCashier, PhaseKeyCashier},
		{phasePharmacy, PhaseKeyPharmacy},
	}
	if len(phases) != len(want) {
		t.Fatalf("phases: got %d, want %d — did the planner gain a phase?", len(phases), len(want))
	}
	for i, w := range want {
		if phases[i].Number != w.number || phases[i].Key != w.key {
			t.Fatalf("phase[%d] = {%d %s}, want {%d %s}", i, phases[i].Number, phases[i].Key, w.number, w.key)
		}
		if i > 0 && phases[i].Number <= phases[i-1].Number {
			t.Fatalf("phase[%d].number %d not ascending after %d", i, phases[i].Number, phases[i-1].Number)
		}
	}
}

func TestPlanningRulesOrderTypesMatchPlannerMap(t *testing.T) {
	got := PlanningRulesDescriptor().OrderTypes
	want := []PlanningOrderType{
		{OrderType: his.OrderTypeLab, StepKind: KindLab},
		{OrderType: his.OrderTypeXray, StepKind: KindXray},
		{OrderType: his.OrderTypeEKG, StepKind: KindEKG},
		{OrderType: his.OrderTypeUS, StepKind: KindUltrasound},
		// DRUG deliberately maps to no walked step: it only surfaces as the
		// visit-wide PHARMACY step, and the descriptor must say so (empty
		// kind) rather than invent one.
		{OrderType: his.OrderTypeDrug, StepKind: ""},
	}
	if len(got) != len(want) {
		t.Fatalf("orderTypes: got %d (%v), want %d — did a canonical order type change?", len(got), got, len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("orderTypes[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}

// exampleStatuses folds a worked example into stepKey→status for comparison.
func exampleStatuses(t *testing.T, ex PlanningExample) map[string]string {
	t.Helper()
	m := map[string]string{}
	for _, st := range ex.Steps {
		if st.Status == "" {
			t.Fatalf("example %s step %s: empty status", ex.ID, st.StepKey)
		}
		if _, dup := m[st.StepKey]; dup {
			t.Fatalf("example %s: duplicate stepKey %s", ex.ID, st.StepKey)
		}
		m[st.StepKey] = st.Status
	}
	return m
}

// The fresh-visit example must equal the real Plan over equivalent facts —
// the pre-visit diagnostic is the only actionable step and everything behind
// its gate is still PENDING, with the mid-visit DRUG surfacing as PHARMACY.
func TestPlanningRulesFreshVisitExampleMatchesPlan(t *testing.T) {
	rules := PlanningRulesDescriptor()
	ex := rules.Examples[0]
	if ex.ID != "fresh-visit" {
		t.Fatalf("examples[0].ID = %q, want fresh-visit", ex.ID)
	}

	statuses := exampleStatuses(t, ex)
	want := map[string]string{
		"REGISTRATION": StepCompleted,
		"LAB:1":        StepReady,
		"CLINIC:ENT:1": StepPending,
		"CASHIER":      StepPending,
		"PHARMACY":     StepPending,
	}
	if len(statuses) != len(want) {
		t.Fatalf("fresh-visit steps = %v, want exactly %v", statuses, want)
	}
	for key, status := range want {
		if statuses[key] != status {
			t.Fatalf("fresh-visit %s = %s, want %s", key, statuses[key], status)
		}
	}
}

// The round-return example shows the mid-visit loop (ADR-0009 §4): with
// round 1 STARTED and a mid-visit LAB ordered by the clinic, Plan spawns the
// round-2 return — WAITING until the result is back.
func TestPlanningRulesReturnExampleMatchesPlan(t *testing.T) {
	rules := PlanningRulesDescriptor()
	ex := rules.Examples[1]
	if ex.ID != "clinic-round-return" {
		t.Fatalf("examples[1].ID = %q, want clinic-round-return", ex.ID)
	}

	statuses := exampleStatuses(t, ex)
	want := map[string]string{
		"REGISTRATION": StepCompleted,
		"CLINIC:ENT:1": StepStarted, // prior preserved (§7)
		"LAB:1":        StepReady,   // mid-visit diagnostic
		"CLINIC:ENT:2": StepWaiting, // spawned return, waiting on results
		"CASHIER":      StepPending,
		"PHARMACY":     StepPending,
	}
	if len(statuses) != len(want) {
		t.Fatalf("return-example steps = %v, want exactly %v", statuses, want)
	}
	for key, status := range want {
		if statuses[key] != status {
			t.Fatalf("return-example %s = %s, want %s", key, statuses[key], status)
		}
	}

	for _, st := range ex.Steps {
		if st.StepKey == "CLINIC:ENT:2" && (st.Round == nil || *st.Round != 2) {
			t.Fatalf("CLINIC:ENT:2 round = %v, want 2", st.Round)
		}
	}
}

// The descriptor's examples cannot drift from the planner: rebuilding the
// same facts in a test and running the exported Plan must produce the same
// steps. If someone replaces the derivation with static JSON, this fails.
func TestPlanningRulesExamplesAreDerivedNotHandwritten(t *testing.T) {
	opened := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	visit := his.Visit{
		VisitID: "TEST-RETURN", VisitType: "APPOINTMENT", Status: "OPEN",
		Clinics: []his.Clinic{{Code: "ENT"}},
		Orders: []his.Order{
			{OrderRef: "ORD-3", OrderType: his.OrderTypeLab, OrderedByClinic: "ENT", OrderedAt: opened.Add(30 * time.Minute)},
			{OrderRef: "ORD-4", OrderType: his.OrderTypeDrug, OrderedByClinic: "ENT", OrderedAt: opened.Add(31 * time.Minute)},
		},
		OpenedAt: opened,
	}
	plan := Plan(visit, map[string]string{"CLINIC:ENT:1": StepStarted}, nil)

	ex := PlanningRulesDescriptor().Examples[1]
	if len(ex.Steps) != len(plan) {
		t.Fatalf("example has %d steps, Plan returns %d", len(ex.Steps), len(plan))
	}
	for i, st := range plan {
		got := ex.Steps[i]
		if got.StepKey != st.StepKey || got.Status != st.Status || got.Sequence != st.Sequence {
			t.Fatalf("step[%d] = {%s %s %d}, want {%s %s %d}",
				i, got.StepKey, got.Status, got.Sequence, st.StepKey, st.Status, st.Sequence)
		}
	}
}
