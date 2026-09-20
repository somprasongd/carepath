package journey

import (
	"time"

	"carepath/apps/api/internal/his"
)

// PlanningRules is the machine-readable description of the rules the planner
// actually runs on (FR-13/US-16, #100): the phase ladder, the order-type →
// step-kind mapping, and worked examples computed by calling Plan over
// canonical scenarios. Everything is derived from planner.go itself — an edit
// to the planner changes this response — so the staff screen can never drift
// from the rules in force. Per ADR-0012 it carries codes only; every display
// string lives in the client.
type PlanningRules struct {
	Phases     []PlanningPhase     `json:"phases"`
	OrderTypes []PlanningOrderType `json:"orderTypes"`
	Examples   []PlanningExample   `json:"examples"`
}

// PlanningPhase is one rung of the phase ladder (ADR-0009 §3): a step becomes
// actionable only once every strictly-lower phase is COMPLETED or CANCELLED;
// steps sharing a phase carry no order between them (§6).
type PlanningPhase struct {
	Number int    `json:"number"`
	Key    string `json:"key"`
}

// PlanningOrderType maps one canonical HIS order type to the step kind the
// planner derives from it. StepKind is empty when the order type deliberately
// produces no walked step: DRUG only surfaces as the visit-wide PHARMACY step,
// so no planStep of its own exists.
type PlanningOrderType struct {
	OrderType string `json:"orderType"`
	StepKind  string `json:"stepKind"`
}

// PlanningExample is one worked scenario: the HIS facts fed in and the plan
// the real Plan function returned. The statuses are the live resolution, not
// illustrative strings — the same inputs a visit would carry produce the same
// steps a patient would see.
type PlanningExample struct {
	ID       string                  `json:"id"`
	Scenario PlanningExampleScenario `json:"scenario"`
	Steps    []PlanningExampleStep   `json:"steps"`
}

// PlanningExampleScenario summarizes the HIS facts a scenario feeds Plan:
// which clinics the visit belongs to and which orders were placed. An order
// whose OrderedByClinic is empty was placed at or before the visit opened
// (pre-visit); one placed later belongs to the named clinic (mid-visit).
type PlanningExampleScenario struct {
	VisitType string                 `json:"visitType"`
	Clinics   []string               `json:"clinics"`
	Orders    []PlanningExampleOrder `json:"orders"`
}

// PlanningExampleOrder is one order of a worked scenario's input facts.
type PlanningExampleOrder struct {
	OrderRef        string `json:"orderRef"`
	OrderType       string `json:"orderType"`
	OrderedByClinic string `json:"orderedByClinic"`
}

// PlanningExampleStep is one step of a worked example: the plan Step shape
// minus projection concerns — there is no service-point binding and no stored
// row behind a synthetic scenario.
type PlanningExampleStep struct {
	StepKey    string   `json:"stepKey"`
	Sequence   int      `json:"sequence"`
	Kind       string   `json:"kind"`
	ClinicCode *string  `json:"clinicCode"`
	Round      *int     `json:"round"`
	OrderRefs  []string `json:"orderRefs"`
	Status     string   `json:"status"`
}

// Phase keys exposed to the client (ADR-0009 §3), one per phase constant.
const (
	PhaseKeyRegistration = "REGISTRATION"
	PhaseKeyPreVisit     = "PRE_VISIT"
	PhaseKeyClinic       = "CLINIC"
	PhaseKeyMidVisit     = "MID_VISIT"
	PhaseKeyReturnClinic = "RETURN_CLINIC"
	PhaseKeyCashier      = "CASHIER"
	PhaseKeyPharmacy     = "PHARMACY"
)

// PlanningRulesDescriptor derives the planning-rule description from the
// planner's own constants, map, and Plan function (#100). Pure and
// side-effect-free: no database, no HIS call — the endpoint can be served
// straight from it.
func PlanningRulesDescriptor() PlanningRules {
	phases := []PlanningPhase{
		{Number: phaseRegistration, Key: PhaseKeyRegistration},
		{Number: phasePreVisit, Key: PhaseKeyPreVisit},
		{Number: phaseClinic, Key: PhaseKeyClinic},
		{Number: phaseMidVisit, Key: PhaseKeyMidVisit},
		{Number: phaseReturnClinic, Key: PhaseKeyReturnClinic},
		{Number: phaseCashier, Key: PhaseKeyCashier},
		{Number: phasePharmacy, Key: PhaseKeyPharmacy},
	}

	// The canonical order types are the his package's; the kind each produces
	// (if any) is whatever the planner's map says today.
	orderTypes := []PlanningOrderType{
		{OrderType: his.OrderTypeLab, StepKind: diagnosticKindByOrderType[his.OrderTypeLab]},
		{OrderType: his.OrderTypeXray, StepKind: diagnosticKindByOrderType[his.OrderTypeXray]},
		{OrderType: his.OrderTypeEKG, StepKind: diagnosticKindByOrderType[his.OrderTypeEKG]},
		{OrderType: his.OrderTypeUS, StepKind: diagnosticKindByOrderType[his.OrderTypeUS]},
		{OrderType: his.OrderTypeDrug, StepKind: diagnosticKindByOrderType[his.OrderTypeDrug]},
	}

	return PlanningRules{
		Phases:     phases,
		OrderTypes: orderTypes,
		Examples: []PlanningExample{
			freshVisitExample(),
			clinicReturnExample(),
		},
	}
}

// freshVisitExample is the plain walk-in: a pre-visit LAB ordered at
// registration, one clinic, a prescription ordered by that clinic. Nothing has
// happened yet, so the pre-visit diagnostic is the only actionable step and
// everything behind its gate is still PENDING.
func freshVisitExample() PlanningExample {
	opened := mockTime(2026, 9, 20, 9, 0)
	visit := his.Visit{
		VisitID: "EXAMPLE-FRESH", VisitType: "WALKIN", Status: "OPEN",
		Clinics: []his.Clinic{{Code: "ENT"}},
		Orders: []his.Order{
			// OrderedAt at the visit's open → pre-visit (§2).
			{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedAt: opened},
			// DRUG ordered by the clinic mid-visit: no walked step, but the
			// visit gains a PHARMACY step.
			{OrderRef: "ORD-2", OrderType: his.OrderTypeDrug, OrderedByClinic: "ENT", OrderedAt: opened.Add(30 * time.Minute)},
		},
		OpenedAt: opened,
	}
	return exampleFrom("fresh-visit", visit, nil, nil)
}

// clinicReturnExample is the mid-visit loop: the ENT encounter started, then
// the doctor sent the patient out for a LAB. While the round that ordered it
// is still STARTED, Plan spawns the round-2 return — WAITING until the result
// is back — and the prescription turns into the visit's PHARMACY step.
func clinicReturnExample() PlanningExample {
	opened := mockTime(2026, 9, 20, 9, 0)
	visit := his.Visit{
		VisitID: "EXAMPLE-RETURN", VisitType: "APPOINTMENT", Status: "OPEN",
		Clinics: []his.Clinic{{Code: "ENT"}},
		Orders: []his.Order{
			// Ordered after the visit opened, by the clinic → mid-visit.
			{OrderRef: "ORD-3", OrderType: his.OrderTypeLab, OrderedByClinic: "ENT", OrderedAt: opened.Add(30 * time.Minute)},
			{OrderRef: "ORD-4", OrderType: his.OrderTypeDrug, OrderedByClinic: "ENT", OrderedAt: opened.Add(31 * time.Minute)},
		},
		OpenedAt: opened,
	}
	prior := map[string]string{"CLINIC:ENT:1": StepStarted}
	return exampleFrom("clinic-round-return", visit, prior, nil)
}

func exampleFrom(id string, visit his.Visit, prior map[string]string, closedRounds map[string]bool) PlanningExample {
	steps := Plan(visit, prior, closedRounds)
	out := PlanningExample{
		ID: id,
		Scenario: PlanningExampleScenario{
			VisitType: visit.VisitType,
			Clinics:   clinicCodesOf(visit),
			Orders:    make([]PlanningExampleOrder, len(visit.Orders)),
		},
		Steps: make([]PlanningExampleStep, len(steps)),
	}
	for i, o := range visit.Orders {
		out.Scenario.Orders[i] = PlanningExampleOrder{
			OrderRef: o.OrderRef, OrderType: o.OrderType, OrderedByClinic: o.OrderedByClinic,
		}
	}
	for i, st := range steps {
		out.Steps[i] = PlanningExampleStep{
			StepKey: st.StepKey, Sequence: st.Sequence, Kind: st.Kind,
			ClinicCode: st.ClinicCode, Round: st.Round, OrderRefs: st.OrderRefs,
			Status: st.Status,
		}
	}
	return out
}

func clinicCodesOf(visit his.Visit) []string {
	codes := make([]string, len(visit.Clinics))
	for i, c := range visit.Clinics {
		codes[i] = c.Code
	}
	return codes
}

// mockTime builds a fixed timestamp for scenario fixtures. The date is
// arbitrary — Plan only compares timestamps against each other.
func mockTime(year, month, day, hour, minute int) time.Time {
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
}
