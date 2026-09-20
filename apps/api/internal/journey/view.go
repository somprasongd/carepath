package journey

import (
	"time"

	"carepath/apps/api/internal/servicepoint"
)

// StepView is one projected step resolved to its service point for display.
// A nil ServicePoint is the explicit unmapped state.
type StepView struct {
	StepKey        string                     `json:"stepKey"`
	Sequence       int                        `json:"sequence"`
	Kind           string                     `json:"kind"`
	ClinicCode     *string                    `json:"clinicCode"`
	Round          *int                       `json:"round"`
	OrderRefs      []string                   `json:"orderRefs"`
	Status         string                     `json:"status"`
	ServicePointID *string                    `json:"servicePointId"`
	ServicePoint   *servicepoint.ServicePoint `json:"servicePoint,omitempty"`
}

// Recommendation criteria (#103, FR-04), sent as stable codes — the
// patient-facing text lives in the web catalogs (ADR-0012).
const (
	// RecommendNearest: the actionable step whose service point is the
	// shortest walk from the visit's last known location.
	RecommendNearest = "NEAREST"
	// RecommendPlanOrder: no usable location, so the plan's own sequence
	// decides — the documented fallback, not a silent first-element grab.
	RecommendPlanOrder = "PLAN_ORDER"
)

// View is the patient/staff-facing journey: the plan with steps ordered by
// sequence, every currently-READY step in Actionable, and CarePath's pick
// among them as Recommended. Completed makes a finished visit unambiguous
// without parsing status.
type View struct {
	VisitID     string     `json:"visitId"`
	PatientRef  string     `json:"patientRef"`
	PatientName string     `json:"patientName,omitempty"`
	Status      string     `json:"status"`
	Completed   bool       `json:"completed"`
	Steps       []StepView `json:"steps"`
	Actionable  []StepView `json:"actionable"`
	Recommended *StepView  `json:"recommended,omitempty"`
	// RecommendationReason names the criterion behind Recommended, so the
	// pick is explainable rather than a black box. Empty exactly when
	// Recommended is nil.
	RecommendationReason string    `json:"recommendationReason,omitempty"`
	SyncedAt             time.Time `json:"syncedAt"`
}

// assembleView resolves service points for the plan's stored bindings and
// derives actionable/recommended. Recommendation starts at the plan-order
// fallback — the first actionable step by sequence; GetJourney then upgrades
// it to the distance criterion when the visit has a known location
// (recommendFromLocation, ADR-0009 §6).
func assembleView(visit Visit, points []servicepoint.ServicePoint) View {
	byID := make(map[string]servicepoint.ServicePoint, len(points))
	for _, sp := range points {
		byID[sp.ID] = sp
	}

	steps := make([]StepView, len(visit.Steps))
	for i, step := range visit.Steps {
		orderRefs := step.OrderRefs
		if orderRefs == nil {
			orderRefs = []string{}
		}
		steps[i] = StepView{
			StepKey: step.StepKey, Sequence: step.Sequence, Kind: step.Kind,
			ClinicCode: step.ClinicCode, Round: step.Round, OrderRefs: orderRefs,
			Status: step.Status, ServicePointID: step.ServicePointID,
		}
		if step.ServicePointID != nil {
			if sp, ok := byID[*step.ServicePointID]; ok {
				steps[i].ServicePoint = &sp
			}
		}
	}

	view := View{
		VisitID: visit.VisitID, PatientRef: visit.PatientRef, PatientName: visit.PatientName,
		Status: visit.Status, Completed: visit.Status == VisitCompleted,
		Steps: steps, Actionable: []StepView{}, SyncedAt: visit.SyncedAt,
	}
	if !view.Completed {
		for i := range steps {
			if steps[i].Status == StepReady {
				view.Actionable = append(view.Actionable, steps[i])
			}
		}
		if len(view.Actionable) > 0 {
			recommended := view.Actionable[0]
			view.Recommended = &recommended
			view.RecommendationReason = RecommendPlanOrder
		}
	}
	return view
}

// recommendByDistance re-picks Recommended as the actionable step bound
// closest to the patient's last known position (#103, FR-04). distances maps
// service point code → walking distance; actionable steps without a distance
// (unmapped point, unreachable place) keep their plan order after every
// ranked step, and equal distances stay with the earlier sequence — so the
// pick only moves when the patient's position or plan does, never because a
// refetch raced a queue tick. Returns whether the distance criterion took
// effect; when it did not, the plan-order fallback assembleView set stands.
func recommendByDistance(view *View, distances map[string]float64) bool {
	if len(view.Actionable) == 0 || len(distances) == 0 {
		return false
	}
	best := -1
	var bestDist float64
	for i := range view.Actionable {
		step := view.Actionable[i]
		if step.ServicePoint == nil {
			continue
		}
		d, ok := distances[step.ServicePoint.Code]
		if !ok {
			continue
		}
		// Actionable is sequence-ordered, so a strict < keeps the earlier
		// step on ties.
		if best < 0 || d < bestDist {
			best, bestDist = i, d
		}
	}
	if best < 0 {
		return false
	}
	recommended := view.Actionable[best]
	view.Recommended = &recommended
	view.RecommendationReason = RecommendNearest
	return true
}

// bindingKey derives the servicepoint lookup code for a step (ADR-0009):
// a clinic step binds by "CLINIC:<code>", a diagnostic step by
// "ORDERTYPE:<type>", and a fixed step (REGISTRATION/CASHIER/PHARMACY) by
// its own kind.
func bindingKey(step Step) string {
	switch step.Kind {
	case KindClinic:
		if step.ClinicCode != nil {
			return "CLINIC:" + *step.ClinicCode
		}
		return KindClinic
	case KindLab:
		return "ORDERTYPE:LAB"
	case KindXray:
		return "ORDERTYPE:XRAY"
	case KindEKG:
		return "ORDERTYPE:EKG"
	case KindUltrasound:
		return "ORDERTYPE:US"
	default:
		return step.Kind
	}
}
