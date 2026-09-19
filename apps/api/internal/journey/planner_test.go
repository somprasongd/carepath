package journey

import (
	"testing"
	"time"

	"carepath/apps/api/internal/his"
)

var openedAt = time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC)

func before(mins int) time.Time { return openedAt.Add(-time.Duration(mins) * time.Minute) }
func after(mins int) time.Time  { return openedAt.Add(time.Duration(mins) * time.Minute) }

func timePtr(t time.Time) *time.Time { return &t }

func stepByKey(steps []Step, key string) (Step, bool) {
	for _, s := range steps {
		if s.StepKey == key {
			return s, true
		}
	}
	return Step{}, false
}

func mustStep(t *testing.T, steps []Step, key string) Step {
	t.Helper()
	s, ok := stepByKey(steps, key)
	if !ok {
		t.Fatalf("step %q not found in plan %+v", key, steps)
	}
	return s
}

// Case 1: an appointment patient with no orders at all goes straight from
// registration to the assigned clinic, then cashier — no diagnostics, no
// pharmacy.
func TestPlan_AppointmentNoOrders(t *testing.T) {
	visit := his.Visit{
		VisitID: "V1", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
	}
	steps := Plan(visit, nil, nil)

	reg := mustStep(t, steps, "REGISTRATION")
	if reg.Status != StepCompleted {
		t.Fatalf("registration = %s, want COMPLETED", reg.Status)
	}
	clinic := mustStep(t, steps, "CLINIC:MED:1")
	if clinic.Status != StepReady {
		t.Fatalf("clinic round 1 = %s, want READY (nothing gates it)", clinic.Status)
	}
	cashier := mustStep(t, steps, "CASHIER")
	if cashier.Status != StepPending {
		t.Fatalf("cashier = %s, want PENDING (clinic not done)", cashier.Status)
	}
	if _, ok := stepByKey(steps, "PHARMACY"); ok {
		t.Fatal("no PHARMACY step expected without a drug order")
	}
}

// Case 2: an appointment patient with a pre-visit lab order must draw blood
// before the clinic becomes actionable.
func TestPlan_AppointmentWithPreVisitLab(t *testing.T) {
	visit := his.Visit{
		VisitID: "V2", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-1", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: before(30), Status: his.OrderPlaced},
		},
	}
	steps := Plan(visit, nil, nil)

	lab := mustStep(t, steps, "LAB:1")
	if lab.Status != StepReady {
		t.Fatalf("pre-visit lab = %s, want READY", lab.Status)
	}
	clinic := mustStep(t, steps, "CLINIC:MED:1")
	if clinic.Status != StepPending {
		t.Fatalf("clinic round 1 = %s, want PENDING (lab not drawn yet)", clinic.Status)
	}

	// Once the blood is drawn, the clinic becomes actionable — resulted is
	// not required for a first clinic visit (ADR-0009 §3/§5).
	visit.Orders[0].Status = his.OrderPerformed
	visit.Orders[0].PerformedAt = timePtr(after(5))
	steps = Plan(visit, nil, nil)
	if s := mustStep(t, steps, "LAB:1"); s.Status != StepCompleted {
		t.Fatalf("performed lab = %s, want COMPLETED", s.Status)
	}
	if s := mustStep(t, steps, "CLINIC:MED:1"); s.Status != StepReady {
		t.Fatalf("clinic round 1 after lab performed = %s, want READY", s.Status)
	}
}

// Case 3: a walk-in patient assigned straight to a clinic, no orders yet.
func TestPlan_WalkinNoOrders(t *testing.T) {
	visit := his.Visit{
		VisitID: "V3", VisitType: his.VisitTypeWalkin, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "SURG"}}, OpenedAt: openedAt,
	}
	steps := Plan(visit, nil, nil)
	if s := mustStep(t, steps, "CLINIC:SURG:1"); s.Status != StepReady {
		t.Fatalf("clinic round 1 = %s, want READY", s.Status)
	}
}

// Case 4: mid-consultation the doctor orders more tests. The planner infers
// a return to the same clinic while the encounter is open (round 1 STARTED).
func TestPlan_MidVisitOrderInfersReturn(t *testing.T) {
	visit := his.Visit{
		VisitID: "V4", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-2", OrderType: his.OrderTypeXray, OrderedByClinic: "MED", OrderedAt: after(10), Status: his.OrderPlaced},
		},
	}
	prior := map[string]string{"CLINIC:MED:1": StepStarted}
	steps := Plan(visit, prior, nil)

	xray := mustStep(t, steps, "XRAY:1")
	if xray.Status != StepReady {
		t.Fatalf("mid-visit xray = %s, want READY", xray.Status)
	}
	round2, ok := stepByKey(steps, "CLINIC:MED:2")
	if !ok {
		t.Fatal("expected an inferred return-to-clinic step (CLINIC:MED:2)")
	}
	if round2.Status != StepWaiting {
		t.Fatalf("round 2 before result = %s, want WAITING", round2.Status)
	}

	// The x-ray is taken but not yet resulted: the patient can leave the
	// x-ray room, but the doctor visit stays WAITING (§5).
	visit.Orders[0].Status = his.OrderPerformed
	visit.Orders[0].PerformedAt = timePtr(after(15))
	steps = Plan(visit, prior, nil)
	if s := mustStep(t, steps, "XRAY:1"); s.Status != StepCompleted {
		t.Fatalf("performed xray = %s, want COMPLETED", s.Status)
	}
	if s := mustStep(t, steps, "CLINIC:MED:2"); s.Status != StepWaiting {
		t.Fatalf("round 2 after performed-not-resulted = %s, want WAITING", s.Status)
	}

	// Once resulted, the return visit becomes actionable.
	visit.Orders[0].ResultedAt = timePtr(after(20))
	steps = Plan(visit, prior, nil)
	if s := mustStep(t, steps, "CLINIC:MED:2"); s.Status != StepReady {
		t.Fatalf("round 2 after resulted = %s, want READY", s.Status)
	}
}

// Case 5: a drug order implies a pharmacy step and never a return to the
// clinic on its own.
func TestPlan_DrugOrderNoReturnButPharmacy(t *testing.T) {
	visit := his.Visit{
		VisitID: "V5", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-3", OrderType: his.OrderTypeDrug, OrderedByClinic: "MED", OrderedAt: after(5), Status: his.OrderPlaced},
		},
	}
	prior := map[string]string{"CLINIC:MED:1": StepStarted, "CASHIER": StepCompleted}
	steps := Plan(visit, prior, nil)

	if _, ok := stepByKey(steps, "CLINIC:MED:2"); ok {
		t.Fatal("a drug order must never infer a return to the clinic")
	}
	pharmacy, ok := stepByKey(steps, "PHARMACY")
	if !ok {
		t.Fatal("expected a PHARMACY step when a drug was ordered")
	}
	if pharmacy.Status != StepReady {
		t.Fatalf("pharmacy with cashier already done = %s, want READY", pharmacy.Status)
	}
}

// Case 6: cashier is exactly one step regardless of how many clinics the
// visit touches, and only becomes actionable once everything else is done.
func TestPlan_CashierOnceAcrossMultipleClinics(t *testing.T) {
	visit := his.Visit{
		VisitID: "V6", VisitType: his.VisitTypeWalkin, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}, {Code: "SURG"}}, OpenedAt: openedAt,
	}
	prior := map[string]string{"CLINIC:MED:1": StepCompleted}
	steps := Plan(visit, prior, nil)

	count := 0
	for _, s := range steps {
		if s.StepKey == "CASHIER" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("cashier steps = %d, want exactly 1", count)
	}
	if s := mustStep(t, steps, "CASHIER"); s.Status != StepPending {
		t.Fatalf("cashier while SURG still open = %s, want PENDING", s.Status)
	}

	prior["CLINIC:SURG:1"] = StepCompleted
	steps = Plan(visit, prior, nil)
	if s := mustStep(t, steps, "CASHIER"); s.Status != StepReady {
		t.Fatalf("cashier once both clinics done = %s, want READY", s.Status)
	}
}

// Case 7: confirming the encounter is finished (§4) drops a not-yet-started
// inferred return, even though the diagnostic step itself stays.
func TestPlan_ClosedRoundDropsUnstartedFollowUp(t *testing.T) {
	visit := his.Visit{
		VisitID: "V7", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-4", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: after(10), Status: his.OrderPlaced},
		},
	}
	prior := map[string]string{"CLINIC:MED:1": StepStarted}
	closed := map[string]bool{"CLINIC:MED:1": true}

	steps := Plan(visit, prior, closed)
	if _, ok := stepByKey(steps, "CLINIC:MED:2"); ok {
		t.Fatal("a not-yet-started follow-up must be dropped once the round is closed")
	}
	if _, ok := stepByKey(steps, "LAB:1"); !ok {
		t.Fatal("the diagnostic itself must still exist — the test still has to happen")
	}
	if s := mustStep(t, steps, "CLINIC:MED:1"); s.Status != StepCompleted {
		t.Fatalf("closed round 1 = %s, want COMPLETED", s.Status)
	}
}

// A follow-up that already started is history (§7) and survives a close.
func TestPlan_ClosedRoundKeepsStartedFollowUp(t *testing.T) {
	visit := his.Visit{
		VisitID: "V8", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-5", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: after(10),
				Status: his.OrderResulted, PerformedAt: timePtr(after(15)), ResultedAt: timePtr(after(20))},
		},
	}
	prior := map[string]string{"CLINIC:MED:1": StepStarted, "CLINIC:MED:2": StepStarted}
	closed := map[string]bool{"CLINIC:MED:1": true}

	steps := Plan(visit, prior, closed)
	round2 := mustStep(t, steps, "CLINIC:MED:2")
	if round2.Status != StepStarted {
		t.Fatalf("already-started follow-up = %s, want STARTED preserved", round2.Status)
	}
}

// History is never reordered or reverted by a replan (§7): a COMPLETED
// diagnostic step stays COMPLETED even if its order is later reported
// cancelled by a stale/duplicate fact.
func TestPlan_CompletedStepNeverReverts(t *testing.T) {
	visit := his.Visit{
		VisitID: "V9", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-6", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: before(10), Status: his.OrderCancelled},
		},
	}
	prior := map[string]string{"LAB:1": StepCompleted}
	steps := Plan(visit, prior, nil)
	if s := mustStep(t, steps, "LAB:1"); s.Status != StepCompleted {
		t.Fatalf("previously-completed step = %s, want COMPLETED preserved", s.Status)
	}
}

// A cancelled pre-visit order still counts as terminal for the clinic gate.
func TestPlan_CancelledPreVisitOrderClearsGate(t *testing.T) {
	visit := his.Visit{
		VisitID: "V10", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
		Orders: []his.Order{
			{OrderRef: "ORD-7", OrderType: his.OrderTypeLab, OrderedByClinic: "MED", OrderedAt: before(10), Status: his.OrderCancelled},
		},
	}
	steps := Plan(visit, nil, nil)
	if s := mustStep(t, steps, "LAB:1"); s.Status != StepCancelled {
		t.Fatalf("cancelled pre-visit lab = %s, want CANCELLED", s.Status)
	}
	if s := mustStep(t, steps, "CLINIC:MED:1"); s.Status != StepReady {
		t.Fatalf("clinic round 1 with only a cancelled prerequisite = %s, want READY", s.Status)
	}
}

// Sequence is assigned in plan order and is purely display; stepKey is the
// stable address.
func TestPlan_SequenceIsDisplayOrderOnly(t *testing.T) {
	visit := his.Visit{
		VisitID: "V11", VisitType: his.VisitTypeAppointment, Status: his.VisitActive,
		Clinics: []his.Clinic{{Code: "MED"}}, OpenedAt: openedAt,
	}
	steps := Plan(visit, nil, nil)
	for i, s := range steps {
		if s.Sequence != i+1 {
			t.Fatalf("step %d (%s) sequence = %d, want %d", i, s.StepKey, s.Sequence, i+1)
		}
	}
}
