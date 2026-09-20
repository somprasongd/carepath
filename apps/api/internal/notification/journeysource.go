package notification

import (
	"context"

	"carepath/apps/api/internal/journey"
)

// JourneySource adapts journey.Service to the sweep's QueueSource port: the
// visit list filtered to what a notification could ever be about, and the
// queue picture thinned to the criterion's single number. It keeps the
// engine testable against fakes and keeps the notification module from
// reaching into plans or transitions.
type JourneySource struct {
	journeys journey.Service
}

var _ QueueSource = (*JourneySource)(nil)

func NewJourneySource(journeys journey.Service) *JourneySource {
	return &JourneySource{journeys: journeys}
}

// ActiveVisits projects ListJourneys onto the sweep candidates: ACTIVE
// visits whose recommended step is bound to a service point. Everyone else
// — finished visits, unmapped recommendations — has no queue question.
func (s *JourneySource) ActiveVisits(ctx context.Context) ([]ActiveVisit, error) {
	views, err := s.journeys.ListJourneys(ctx)
	if err != nil {
		return nil, err
	}
	var out []ActiveVisit
	for _, view := range views {
		if view.Status != journey.VisitActive || view.Recommended == nil || view.Recommended.ServicePoint == nil {
			continue
		}
		out = append(out, ActiveVisit{
			VisitID:      view.VisitID,
			StepKey:      view.Recommended.StepKey,
			ServicePoint: view.Recommended.ServicePoint.Code,
		})
	}
	return out, nil
}

// Queue thins the FR-17 picture to waiting-ahead per step.
func (s *JourneySource) Queue(ctx context.Context, visitID string) ([]QueueStep, error) {
	queue, err := s.journeys.GetQueue(ctx, visitID)
	if err != nil {
		return nil, err
	}
	out := make([]QueueStep, 0, len(queue.Steps))
	for _, step := range queue.Steps {
		out = append(out, QueueStep{StepKey: step.StepKey, WaitingAhead: step.WaitingAhead})
	}
	return out, nil
}
