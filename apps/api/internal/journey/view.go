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
	SyncedAt    time.Time  `json:"syncedAt"`
}

// assembleView resolves service points for the plan's stored bindings and
// derives actionable/recommended. Recommendation today is simply the first
// actionable step by sequence — a placeholder until the navigation module
// can rank by distance/queue length (ADR-0009 §6).
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
		}
	}
	return view
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
