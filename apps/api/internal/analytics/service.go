package analytics

import (
	"context"
	"fmt"
	"time"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type service struct {
	repo Repo
	tx   db.Transactor
	tz   string
}

// NewService validates the timezone once at boot — a typo in
// ANALYTICS_TIMEZONE must fail loudly, not shift every midnight silently.
// All reads for one overview run in a single read-only transaction so the
// counts and averages describe one consistent snapshot.
func NewService(repo Repo, tx db.Transactor, tz string) (Service, error) {
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "analytics: unknown timezone %q", tz)
	}
	return &service{repo: repo, tx: tx, tz: tz}, nil
}

func (s *service) Overview(ctx context.Context, window string) (Overview, error) {
	if window != WindowToday {
		return Overview{}, apperr.New(apperr.KindInvalid,
			fmt.Sprintf("unsupported window %q (only %q exists)", window, WindowToday))
	}

	var data Data
	if err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		var err error
		data, err = s.repo.Fetch(ctx, s.tz)
		return err
	}); err != nil {
		return Overview{}, err
	}
	return shape(data, window), nil
}

// shape turns the fetched data into the wire overview: it passes the
// null-not-zero averages through untouched and picks the bottleneck —
// highest avgWaitMinutes, ties broken by waitingNow, then by the stable
// code order the repo returns.
func shape(data Data, window string) Overview {
	points := make([]ServicePointMetrics, len(data.Points))
	for i, p := range data.Points {
		points[i] = ServicePointMetrics{
			ServicePointID: p.ServicePointID, Code: p.Code, Name: p.Name,
			WaitingNow: p.WaitingNow, InProgressNow: p.InProgressNow,
			LongestWaitingMinutes: p.LongestWaitingMinutes,
			AvgWaitMinutes:        p.AvgWaitMinutes,
			AvgServiceMinutes:     p.AvgServiceMinutes,
			CompletedCount:        p.CompletedCount,
		}
	}
	return Overview{
		AsOf:   data.AsOf,
		Window: window,
		Summary: Summary{
			ActiveVisits:             data.ActiveVisits,
			AvgWaitMinutes:           data.AvgWaitMinutes,
			AvgVisitMinutes:          data.AvgVisitMinutes,
			BottleneckServicePointID: bottleneck(data.Points),
		},
		ServicePoints: points,
	}
}

// bottleneck picks the service point with the highest average wait. A
// service point with no wait data (nil average) can never be the
// bottleneck; equal averages break by who is still waiting now.
func bottleneck(points []PointData) *string {
	var best *PointData
	for i := range points {
		p := &points[i]
		if p.AvgWaitMinutes == nil {
			continue
		}
		if best == nil || *p.AvgWaitMinutes > *best.AvgWaitMinutes ||
			(*p.AvgWaitMinutes == *best.AvgWaitMinutes && p.WaitingNow > best.WaitingNow) {
			best = p
		}
	}
	if best == nil {
		return nil
	}
	id := best.ServicePointID
	return &id
}
