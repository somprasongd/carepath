package journey

import (
	"fmt"
	"sort"

	"carepath/apps/api/internal/his"
)

// Planning phases (ADR-0009 §3). A step becomes actionable only once every
// step of a strictly lower phase is COMPLETED or CANCELLED; steps sharing a
// phase carry no order between them (§6) — Plan may return more than one
// READY step at once.
const (
	phaseRegistration = 0
	phasePreVisit     = 10
	phaseClinic       = 20
	phaseMidVisit     = 30
	phaseReturnClinic = 40
	phaseCashier      = 90
	phasePharmacy     = 95
)

// diagnosticKindByOrderType maps a canonical HIS order type to the step kind
// it produces. DRUG is intentionally absent: a prescription is not, by
// itself, a step the patient walks to (only the resulting PHARMACY step is).
var diagnosticKindByOrderType = map[string]string{
	his.OrderTypeLab:  KindLab,
	his.OrderTypeXray: KindXray,
	his.OrderTypeEKG:  KindEKG,
	his.OrderTypeUS:   KindUltrasound,
}

// planStep is the planner's working representation before status is resolved.
type planStep struct {
	stepKey    string
	kind       string
	phase      int
	clinicCode *string
	round      *int
	orderRefs  []string
}

// Plan derives the ordered journey plan from one HIS visit snapshot
// (ADR-0009). It is a pure function: the same (visit, prior, closedRounds)
// always yields the same plan.
//
//   - prior is the status of every stepKey in the visit's current stored
//     plan. It lets Plan preserve staff/fact-driven history (§7 — a STARTED,
//     COMPLETED, or CANCELLED step is never reordered or removed) and tells
//     Plan whether a clinic round is currently open (its step is STARTED).
//   - closedRounds is the set of clinic-round stepKeys explicitly confirmed
//     finished — by the HIS's encounter.completed fact or the staff
//     close-round override (§4) — regardless of what prior says.
//
// Known limitation: a clinic round returns the patient to the doctor at most
// once (round 1 -> round 2). Modelling deeper loops is deferred; see
// docs/adr/0009-carepath-owns-journey-plan.md §4.
func Plan(visit his.Visit, prior map[string]string, closedRounds map[string]bool) []Step {
	if prior == nil {
		prior = map[string]string{}
	}
	if closedRounds == nil {
		closedRounds = map[string]bool{}
	}

	var plan []planStep
	plan = append(plan, planStep{stepKey: "REGISTRATION", kind: KindRegistration, phase: phaseRegistration})

	preVisit, midByClinic := partitionOrders(visit)

	typeSeq := map[string]int{}
	for _, ot := range sortedOrderTypes(preVisit) {
		plan = append(plan, planStep{
			stepKey:   fmt.Sprintf("%s:1", ot),
			kind:      diagnosticKindByOrderType[ot],
			phase:     phasePreVisit,
			orderRefs: orderRefsOf(preVisit[ot]),
		})
		typeSeq[ot] = 2 // #1 just used by the pre-visit step
	}

	for _, clinic := range visit.Clinics {
		clinic := clinic
		round1Key := fmt.Sprintf("CLINIC:%s:1", clinic.Code)
		plan = append(plan, planStep{
			stepKey: round1Key, kind: KindClinic, phase: phaseClinic,
			clinicCode: &clinic.Code, round: intPtr(1),
		})

		mid := midByClinic[clinic.Code]
		if len(mid) == 0 {
			continue
		}
		var midRefs []string
		for _, ot := range sortedOrderTypes(mid) {
			seq := typeSeq[ot]
			if seq == 0 {
				seq = 1
			}
			typeSeq[ot] = seq + 1
			refs := orderRefsOf(mid[ot])
			midRefs = append(midRefs, refs...)
			plan = append(plan, planStep{
				stepKey: fmt.Sprintf("%s:%d", ot, seq), kind: diagnosticKindByOrderType[ot], phase: phaseMidVisit,
				clinicCode: &clinic.Code, orderRefs: refs,
			})
		}

		// A follow-up round is spawned while the round that ordered the
		// diagnostics is still open — closing it (§4) means "no need to
		// return", even though the diagnostics themselves still happen. Once
		// the follow-up has history of its own (STARTED/COMPLETED/CANCELLED)
		// it must keep existing regardless of the round's open-ness (§7): a
		// later close must never erase a return that already happened.
		round2Key := fmt.Sprintf("CLINIC:%s:2", clinic.Code)
		isOpen := prior[round1Key] == StepStarted && !closedRounds[round1Key]
		if isOpen || isHistory(prior[round2Key]) {
			plan = append(plan, planStep{
				stepKey: round2Key, kind: KindClinic, phase: phaseReturnClinic,
				clinicCode: &clinic.Code, round: intPtr(2), orderRefs: midRefs,
			})
		}
	}

	plan = append(plan, planStep{stepKey: "CASHIER", kind: KindCashier, phase: phaseCashier})
	if hasActiveDrugOrder(visit) {
		plan = append(plan, planStep{stepKey: "PHARMACY", kind: KindPharmacy, phase: phasePharmacy})
	}

	return resolveStatuses(plan, visit, prior, closedRounds)
}

func resolveStatuses(plan []planStep, visit his.Visit, prior map[string]string, closed map[string]bool) []Step {
	orderByRef := make(map[string]his.Order, len(visit.Orders))
	for _, o := range visit.Orders {
		orderByRef[o.OrderRef] = o
	}

	steps := make([]Step, len(plan))
	for i, p := range plan {
		steps[i] = Step{
			StepKey: p.stepKey, Sequence: i + 1, Kind: p.kind,
			ClinicCode: p.clinicCode, Round: p.round, OrderRefs: p.orderRefs,
		}
	}

	// Phase 10 gate for phase 20 (clinic round 1): every pre-visit diagnostic
	// must be COMPLETED or CANCELLED first.
	phase10Terminal := true
	for i, p := range plan {
		if p.phase != phasePreVisit {
			continue
		}
		steps[i].Status = diagnosticStatus(p, orderByRef, prior)
		if steps[i].Status != StepCompleted && steps[i].Status != StepCancelled {
			phase10Terminal = false
		}
	}

	for i, p := range plan {
		switch p.phase {
		case phaseRegistration:
			steps[i].Status = StepCompleted
		case phaseClinic:
			steps[i].Status = fixedGateStatus(prior[p.stepKey], closed[p.stepKey], phase10Terminal)
		case phaseMidVisit:
			steps[i].Status = diagnosticStatus(p, orderByRef, prior)
		case phaseReturnClinic:
			steps[i].Status = returnClinicStatus(p, orderByRef, prior, closed)
		}
	}

	cashierIdx, pharmacyIdx := -1, -1
	for i, p := range plan {
		switch p.phase {
		case phaseCashier:
			cashierIdx = i
		case phasePharmacy:
			pharmacyIdx = i
		}
	}
	if cashierIdx >= 0 {
		everythingElseDone := true
		for i := range steps {
			if i == cashierIdx || i == pharmacyIdx {
				continue
			}
			if steps[i].Status != StepCompleted && steps[i].Status != StepCancelled {
				everythingElseDone = false
				break
			}
		}
		steps[cashierIdx].Status = fixedGateStatus(prior[steps[cashierIdx].StepKey], false, everythingElseDone)
	}
	if pharmacyIdx >= 0 {
		cashierDone := cashierIdx >= 0 && steps[cashierIdx].Status == StepCompleted
		steps[pharmacyIdx].Status = fixedGateStatus(prior[steps[pharmacyIdx].StepKey], false, cashierDone)
	}

	return steps
}

// isHistory reports whether status is one a fresh step must never be
// re-created over (§7): once true, the step already exists in the plan.
func isHistory(status string) bool {
	return status == StepStarted || status == StepCompleted || status == StepCancelled
}

// isTerminal reports whether status is truly locked: even an explicit round
// close (which promotes STARTED to COMPLETED) must never revert it.
func isTerminal(status string) bool {
	return status == StepCompleted || status == StepCancelled
}

// diagnosticStatus resolves a LAB/XRAY/EKG/ULTRASOUND step. It completes at
// order.performed regardless of prior STARTED (§5 — the patient's part is
// done once the procedure happened); a prior COMPLETED/CANCELLED is
// preserved; otherwise it is READY (both phase 10 and phase 30 diagnostic
// steps are only ever created once actionable — trivially so for phase 10,
// contingent on the round being open for phase 30).
func diagnosticStatus(p planStep, orderByRef map[string]his.Order, prior map[string]string) string {
	if v := prior[p.stepKey]; v == StepCompleted || v == StepCancelled {
		return v
	}
	if len(p.orderRefs) > 0 {
		allCancelled, allPerformed := true, true
		for _, ref := range p.orderRefs {
			o := orderByRef[ref]
			if o.Status != his.OrderCancelled {
				allCancelled = false
			}
			if o.PerformedAt == nil && o.Status != his.OrderCancelled {
				allPerformed = false
			}
		}
		switch {
		case allCancelled:
			return StepCancelled
		case allPerformed:
			return StepCompleted
		}
	}
	if prior[p.stepKey] == StepStarted {
		return StepStarted
	}
	return StepReady
}

// returnClinicStatus resolves a round-2 CLINIC step: WAITING until every
// diagnostic order of the round it followed is resulted, then READY (§5). An
// explicit close of this round (§4) promotes it straight to COMPLETED, even
// over a STARTED in progress — but never over an already-terminal status.
func returnClinicStatus(p planStep, orderByRef map[string]his.Order, prior map[string]string, closed map[string]bool) string {
	if v := prior[p.stepKey]; isTerminal(v) {
		return v
	}
	if closed[p.stepKey] {
		return StepCompleted
	}
	if prior[p.stepKey] == StepStarted {
		return StepStarted
	}
	for _, ref := range p.orderRefs {
		o := orderByRef[ref]
		if o.Status == his.OrderCancelled {
			continue
		}
		if o.ResultedAt == nil {
			return StepWaiting
		}
	}
	return StepReady
}

// fixedGateStatus resolves a step with no HIS-fact-driven status of its own
// (CLINIC round 1, CASHIER, PHARMACY): a terminal status is preserved, an
// explicit close promotes even a STARTED round to COMPLETED, an in-progress
// STARTED otherwise holds, and absent either, READY iff the gate is open.
func fixedGateStatus(priorStatus string, closed bool, gateOpen bool) string {
	if isTerminal(priorStatus) {
		return priorStatus
	}
	if closed {
		return StepCompleted
	}
	if priorStatus == StepStarted {
		return StepStarted
	}
	if gateOpen {
		return StepReady
	}
	return StepPending
}

// partitionOrders splits non-drug orders into the pre-visit bucket (ordered
// at or before the visit opened, grouped by order type) and the per-clinic
// mid-visit bucket (everything else, grouped by ordering clinic then type).
func partitionOrders(visit his.Visit) (map[string][]his.Order, map[string]map[string][]his.Order) {
	preVisit := map[string][]his.Order{}
	mid := map[string]map[string][]his.Order{}
	for _, o := range visit.Orders {
		if o.OrderType == his.OrderTypeDrug {
			continue
		}
		if !o.OrderedAt.After(visit.OpenedAt) {
			preVisit[o.OrderType] = append(preVisit[o.OrderType], o)
			continue
		}
		byType, ok := mid[o.OrderedByClinic]
		if !ok {
			byType = map[string][]his.Order{}
			mid[o.OrderedByClinic] = byType
		}
		byType[o.OrderType] = append(byType[o.OrderType], o)
	}
	return preVisit, mid
}

func hasActiveDrugOrder(visit his.Visit) bool {
	for _, o := range visit.Orders {
		if o.OrderType == his.OrderTypeDrug && o.Status != his.OrderCancelled {
			return true
		}
	}
	return false
}

func sortedOrderTypes(byType map[string][]his.Order) []string {
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

func orderRefsOf(orders []his.Order) []string {
	refs := make([]string, len(orders))
	for i, o := range orders {
		refs[i] = o.OrderRef
	}
	return refs
}

func intPtr(v int) *int { return &v }
