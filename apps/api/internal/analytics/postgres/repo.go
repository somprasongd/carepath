// Package postgres is the analytics module's read-only persistence adapter.
// Ownership: journey writes journey_step_status_event / journey_step /
// journey_visit, servicepoint owns service_point — analytics reads them for
// aggregation and never writes ("journey เขียน · analytics อ่าน", #86).
//
// All aggregation happens in SQL; midnight is resolved by the database with
// AT TIME ZONE so the numbers follow the configured analytics timezone, not
// the Go process's clock or locale. Averages over zero rows surface as SQL
// NULL and scan into nil pointers — "no data" must stay distinct from 0 all
// the way to the JSON.
package postgres

import (
	"context"

	"carepath/apps/api/internal/analytics"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type Repo struct {
	database *db.DB
}

var _ analytics.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

// summaryQuery lifts the visit-wide scalars: the snapshot time, currently
// ACTIVE visits, the average visit length over visits that finished within
// today (last timeline event inside the window), and the global average
// wait over steps that started today.
//
// started/ready pair per (visit, step): the latest STARTED within the
// window, minus the latest READY strictly before it — the wait the patient
// actually experienced for that step, however many replans happened around
// it.
const summaryQuery = `
WITH ws AS (
    SELECT (date_trunc('day', now() AT TIME ZONE $1) AT TIME ZONE $1) AS window_start
),
started AS (
    SELECT e.visit_id, e.step_key, max(e.occurred_at) AS started_at
    FROM carepath.journey_step_status_event e CROSS JOIN ws
    WHERE e.to_status = 'STARTED' AND e.occurred_at >= ws.window_start
    GROUP BY e.visit_id, e.step_key
),
wait_pairs AS (
    SELECT extract(epoch FROM (s.started_at - r.ready_at)) / 60.0 AS wait_minutes
    FROM started s
    JOIN LATERAL (
        SELECT max(e.occurred_at) AS ready_at
        FROM carepath.journey_step_status_event e
        WHERE e.visit_id = s.visit_id AND e.step_key = s.step_key
          AND e.to_status = 'READY' AND e.occurred_at < s.started_at
    ) r ON r.ready_at IS NOT NULL
),
visit_spans AS (
    SELECT e.visit_id, max(e.occurred_at) AS last_occ, min(e.occurred_at) AS first_occ
    FROM carepath.journey_step_status_event e
    GROUP BY e.visit_id
)
SELECT
    now() AS as_of,
    (SELECT count(*) FROM carepath.journey_visit v WHERE v.status = 'ACTIVE') AS active_visits,
    (SELECT avg(extract(epoch FROM (vs.last_occ - vs.first_occ)) / 60.0)
       FROM visit_spans vs
       JOIN carepath.journey_visit v ON v.visit_id = vs.visit_id
       WHERE v.status = 'COMPLETED' AND vs.last_occ >= (SELECT window_start FROM ws)) AS avg_visit_minutes,
    (SELECT avg(wait_minutes) FROM wait_pairs) AS avg_wait_minutes
`

// pointsQuery returns one row per active service point with every metric:
// the "now" counts and the longest current wait from current state, the
// windowed averages and completed count from the timeline.
const pointsQuery = `
WITH ws AS (
    SELECT (date_trunc('day', now() AT TIME ZONE $1) AT TIME ZONE $1) AS window_start
),
started AS (
    SELECT e.visit_id, e.step_key, max(e.occurred_at) AS started_at, e.service_point_id
    FROM carepath.journey_step_status_event e CROSS JOIN ws
    WHERE e.to_status = 'STARTED' AND e.occurred_at >= ws.window_start
    GROUP BY e.visit_id, e.step_key, e.service_point_id
),
wait_pairs AS (
    SELECT s.service_point_id,
           extract(epoch FROM (s.started_at - r.ready_at)) / 60.0 AS wait_minutes
    FROM started s
    JOIN LATERAL (
        SELECT max(e.occurred_at) AS ready_at
        FROM carepath.journey_step_status_event e
        WHERE e.visit_id = s.visit_id AND e.step_key = s.step_key
          AND e.to_status = 'READY' AND e.occurred_at < s.started_at
    ) r ON r.ready_at IS NOT NULL
),
completed AS (
    SELECT e.visit_id, e.step_key, max(e.occurred_at) AS completed_at, e.service_point_id
    FROM carepath.journey_step_status_event e CROSS JOIN ws
    WHERE e.to_status = 'COMPLETED' AND e.occurred_at >= ws.window_start
    GROUP BY e.visit_id, e.step_key, e.service_point_id
),
service_pairs AS (
    SELECT c.service_point_id,
           extract(epoch FROM (c.completed_at - st.started_at)) / 60.0 AS service_minutes
    FROM completed c
    JOIN LATERAL (
        SELECT max(e.occurred_at) AS started_at
        FROM carepath.journey_step_status_event e
        WHERE e.visit_id = c.visit_id AND e.step_key = c.step_key
          AND e.to_status = 'STARTED' AND e.occurred_at < c.completed_at
    ) st ON st.started_at IS NOT NULL
),
w AS (SELECT service_point_id, avg(wait_minutes) AS avg_wait
      FROM wait_pairs WHERE service_point_id IS NOT NULL GROUP BY service_point_id),
sv AS (SELECT service_point_id, avg(service_minutes) AS avg_service
       FROM service_pairs WHERE service_point_id IS NOT NULL GROUP BY service_point_id),
cc AS (SELECT service_point_id, count(*) AS completed_count
       FROM completed WHERE service_point_id IS NOT NULL GROUP BY service_point_id)
SELECT sp.id, sp.code, sp.name,
       (SELECT count(*) FROM carepath.journey_step js
         WHERE js.service_point_id = sp.id AND js.status = 'READY') AS waiting_now,
       (SELECT count(*) FROM carepath.journey_step js
         WHERE js.service_point_id = sp.id AND js.status = 'STARTED') AS in_progress_now,
       lw.longest_wait,
       w.avg_wait, sv.avg_service, COALESCE(cc.completed_count, 0) AS completed_count
FROM carepath.service_point sp
JOIN LATERAL (
    SELECT max(extract(epoch FROM (now() - r2.ready_at)) / 60.0) AS longest_wait
    FROM carepath.journey_step js
    JOIN LATERAL (
        SELECT max(e.occurred_at) AS ready_at
        FROM carepath.journey_step_status_event e
        WHERE e.visit_id = js.visit_id AND e.step_key = js.step_key
          AND e.to_status = 'READY'
    ) r2 ON r2.ready_at IS NOT NULL
    WHERE js.service_point_id = sp.id AND js.status = 'READY'
) lw ON true
LEFT JOIN w ON w.service_point_id = sp.id
LEFT JOIN sv ON sv.service_point_id = sp.id
LEFT JOIN cc ON cc.service_point_id = sp.id
WHERE sp.active
ORDER BY sp.code
`

func (r *Repo) Fetch(ctx context.Context, tz string) (analytics.Data, error) {
	q := r.database.Querier(ctx)

	var data analytics.Data
	if err := q.QueryRow(ctx, summaryQuery, tz).Scan(
		&data.AsOf, &data.ActiveVisits, &data.AvgVisitMinutes, &data.AvgWaitMinutes,
	); err != nil {
		return analytics.Data{}, apperr.Wrapf(apperr.KindInternal, err, "analytics: fetch summary")
	}

	rows, err := q.Query(ctx, pointsQuery, tz)
	if err != nil {
		return analytics.Data{}, apperr.Wrapf(apperr.KindInternal, err, "analytics: fetch service points")
	}
	defer rows.Close()
	for rows.Next() {
		var p analytics.PointData
		if err := rows.Scan(&p.ServicePointID, &p.Code, &p.Name,
			&p.WaitingNow, &p.InProgressNow, &p.LongestWaitingMinutes,
			&p.AvgWaitMinutes, &p.AvgServiceMinutes, &p.CompletedCount,
		); err != nil {
			return analytics.Data{}, apperr.Wrapf(apperr.KindInternal, err, "analytics: scan service point row")
		}
		data.Points = append(data.Points, p)
	}
	if err := rows.Err(); err != nil {
		return analytics.Data{}, apperr.Wrapf(apperr.KindInternal, err, "analytics: iterate service point rows")
	}
	return data, nil
}
