// Package postgres is the journey module's persistence adapter over pgx.
// All queries resolve their connection from the ambient context, so they
// transparently run inside the caller's transaction when one is open.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/journey"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type Repo struct {
	database *db.DB
}

var _ journey.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

// UpsertVisit replaces the visit's plan: the visit row is upserted and its
// steps are deleted and re-inserted from the plan, so replanning never
// duplicates or keeps a step the current plan no longer has.
func (r *Repo) UpsertVisit(ctx context.Context, visit journey.Visit) error {
	q := r.database.Querier(ctx)
	_, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, patient_name, status, synced_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (visit_id) DO UPDATE
		 SET patient_ref = EXCLUDED.patient_ref,
		     patient_name = EXCLUDED.patient_name,
		     status = EXCLUDED.status,
		     synced_at = now()`,
		visit.VisitID, visit.PatientRef, visit.PatientName, visit.Status,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "journey: upsert visit %s", visit.VisitID)
	}
	if _, err := q.Exec(ctx,
		`DELETE FROM carepath.journey_step WHERE visit_id = $1`, visit.VisitID,
	); err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "journey: replace steps of %s", visit.VisitID)
	}
	for _, step := range visit.Steps {
		orderRefs := step.OrderRefs
		if orderRefs == nil {
			orderRefs = []string{}
		}
		if _, err := q.Exec(ctx,
			`INSERT INTO carepath.journey_step
			   (visit_id, step_key, sequence, kind, clinic_code, round, order_refs, status, service_point_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			visit.VisitID, step.StepKey, step.Sequence, step.Kind, step.ClinicCode, step.Round,
			orderRefs, step.Status, step.ServicePointID,
		); err != nil {
			return apperr.Wrapf(apperr.KindInternal, err,
				"journey: insert step %s of %s", step.StepKey, visit.VisitID)
		}
	}
	return nil
}

func (r *Repo) GetVisit(ctx context.Context, visitID string) (journey.Visit, error) {
	var visit journey.Visit
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT visit_id, patient_ref, coalesce(patient_name, ''), status, synced_at
		 FROM carepath.journey_visit WHERE visit_id = $1`, visitID,
	).Scan(&visit.VisitID, &visit.PatientRef, &visit.PatientName, &visit.Status, &visit.SyncedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return journey.Visit{}, journey.ErrNotFound
	}
	if err != nil {
		return journey.Visit{}, apperr.Wrapf(apperr.KindInternal, err, "journey: get visit %s", visitID)
	}

	steps, err := r.stepsOf(ctx, visitID)
	if err != nil {
		return journey.Visit{}, err
	}
	visit.Steps = steps
	return visit, nil
}

func (r *Repo) stepsOf(ctx context.Context, visitID string) ([]journey.Step, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		`SELECT step_key, sequence, kind, clinic_code, round, order_refs, status, service_point_id
		 FROM carepath.journey_step WHERE visit_id = $1 ORDER BY sequence`, visitID,
	)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: list steps of %s", visitID)
	}
	defer rows.Close()

	var steps []journey.Step
	for rows.Next() {
		var step journey.Step
		if err := rows.Scan(&step.StepKey, &step.Sequence, &step.Kind, &step.ClinicCode, &step.Round,
			&step.OrderRefs, &step.Status, &step.ServicePointID); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: scan step of %s", visitID)
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate steps of %s", visitID)
	}
	return steps, nil
}

// ListVisits returns every projected visit with its steps, freshest sync
// first (#37). Two queries and a Go-side group-join keep it simple at demo
// scale; ordering (synced_at DESC, visit_id as the tiebreaker) is part of
// this port's contract.
func (r *Repo) ListVisits(ctx context.Context) ([]journey.Visit, error) {
	visitRows, err := r.database.Querier(ctx).Query(ctx,
		`SELECT visit_id, patient_ref, coalesce(patient_name, ''), status, synced_at
		 FROM carepath.journey_visit ORDER BY synced_at DESC, visit_id`)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: list visits")
	}
	defer visitRows.Close()

	visits := make([]journey.Visit, 0)
	for visitRows.Next() {
		var visit journey.Visit
		if err := visitRows.Scan(&visit.VisitID, &visit.PatientRef, &visit.PatientName, &visit.Status, &visit.SyncedAt); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: scan visit row")
		}
		visits = append(visits, visit)
	}
	if err := visitRows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate visits")
	}

	stepRows, err := r.database.Querier(ctx).Query(ctx,
		`SELECT visit_id, step_key, sequence, kind, clinic_code, round, order_refs, status, service_point_id
		 FROM carepath.journey_step ORDER BY visit_id, sequence`)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: list steps")
	}
	defer stepRows.Close()

	stepsByVisit := make(map[string][]journey.Step, len(visits))
	for stepRows.Next() {
		var (
			visitID string
			step    journey.Step
		)
		if err := stepRows.Scan(&visitID, &step.StepKey, &step.Sequence, &step.Kind, &step.ClinicCode,
			&step.Round, &step.OrderRefs, &step.Status, &step.ServicePointID); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: scan step row")
		}
		stepsByVisit[visitID] = append(stepsByVisit[visitID], step)
	}
	if err := stepRows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate steps")
	}

	for i := range visits {
		visits[i].Steps = stepsByVisit[visits[i].VisitID]
	}
	return visits, nil
}

// MarkEventApplied records an eventId as consumed. Re-marking is a no-op, so
// concurrent applies of the same event both succeed.
func (r *Repo) MarkEventApplied(ctx context.Context, eventID, visitID string) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.his_applied_event (event_id, visit_id) VALUES ($1, $2)
		 ON CONFLICT (event_id) DO NOTHING`, eventID, visitID,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "journey: mark event %s applied", eventID)
	}
	return nil
}

func (r *Repo) EventApplied(ctx context.Context, eventID string) (bool, error) {
	var applied bool
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM carepath.his_applied_event WHERE event_id = $1)`, eventID,
	).Scan(&applied)
	if err != nil {
		return false, apperr.Wrapf(apperr.KindInternal, err, "journey: check event %s applied", eventID)
	}
	return applied, nil
}

// InsertCommandAudit records one staff transition command (#19 AC4, amended
// by ADR-0009). command_id is the primary key, so replaying the same id —
// e.g. a client retry after a failed local transaction — is a no-op, not a
// duplicate row.
func (r *Repo) InsertCommandAudit(ctx context.Context, audit journey.CommandAudit) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.journey_command_audit
		       (command_id, visit_id, step_key, to_status, source, actor_user_id, actor_username)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (command_id) DO NOTHING`,
		audit.CommandID, audit.VisitID, audit.StepKey, audit.ToStatus, audit.Source,
		audit.ActorUserID, audit.ActorUsername,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err,
			"journey: audit command %s of %s", audit.CommandID, audit.VisitID)
	}
	return nil
}

// CloseRound records a clinic round as explicitly confirmed finished
// (ADR-0009 §4). Idempotent: closing the same stepKey twice is a no-op.
func (r *Repo) CloseRound(ctx context.Context, visitID, stepKey string) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.journey_closed_round (visit_id, step_key)
		 VALUES ($1, $2)
		 ON CONFLICT (visit_id, step_key) DO NOTHING`,
		visitID, stepKey,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "journey: close round %s of %s", stepKey, visitID)
	}
	return nil
}

func (r *Repo) ClosedRounds(ctx context.Context, visitID string) (map[string]bool, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		`SELECT step_key FROM carepath.journey_closed_round WHERE visit_id = $1`, visitID)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: list closed rounds of %s", visitID)
	}
	defer rows.Close()

	closed := map[string]bool{}
	for rows.Next() {
		var stepKey string
		if err := rows.Scan(&stepKey); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: scan closed round of %s", visitID)
		}
		closed[stepKey] = true
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate closed rounds of %s", visitID)
	}
	return closed, nil
}

// AppendStatusEvents appends one batch of timeline rows for a single replan
// round (#85). There is deliberately no UPDATE, DELETE, or read-back path in
// code — the table is the append-only history of how step statuses traveled.
func (r *Repo) AppendStatusEvents(ctx context.Context, events []journey.StepStatusEvent) error {
	if len(events) == 0 {
		return nil
	}
	q := r.database.Querier(ctx)
	for _, ev := range events {
		if _, err := q.Exec(ctx,
			`INSERT INTO carepath.journey_step_status_event
			       (visit_id, step_key, kind, service_point_id, from_status, to_status, source, actor_user_id, actor_username)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			ev.VisitID, ev.StepKey, ev.Kind, ev.ServicePointID, ev.FromStatus, ev.ToStatus,
			ev.Source, ev.ActorUserID, ev.ActorUsername,
		); err != nil {
			return apperr.Wrapf(apperr.KindInternal, err,
				"journey: append status event %s of %s", ev.StepKey, ev.VisitID)
		}
	}
	return nil
}

// queueWaitingQuery counts, per service point, the READY steps of every
// visit except the one asking — the people ahead of that patient right now
// (#101). Current state lives in journey_step, so this is a plain grouped
// count; nothing here needs the timeline.
const queueWaitingQuery = `
SELECT js.service_point_id, count(*)
FROM carepath.journey_step js
WHERE js.status = 'READY' AND js.visit_id <> $1 AND js.service_point_id = ANY($2)
GROUP BY js.service_point_id
`

// queueAvgWaitQuery averages the waits patients actually experienced at the
// asked service points so far today (FR-17): per (visit, step), the latest
// STARTED within the window minus the latest READY strictly before it — the
// same pairing the analytics module uses for its overview, written again
// here on purpose (#101 chose module separation over a shared calculator;
// journey owns this table, analytics merely reads it). Midnight resolves in
// the database via AT TIME ZONE; zero samples surface as an absent row, and
// the service keeps that distinct from any number.
const queueAvgWaitQuery = `
WITH ws AS (
    SELECT (date_trunc('day', now() AT TIME ZONE $2) AT TIME ZONE $2) AS window_start
),
started AS (
    SELECT e.visit_id, e.step_key, max(e.occurred_at) AS started_at, e.service_point_id
    FROM carepath.journey_step_status_event e CROSS JOIN ws
    WHERE e.to_status = 'STARTED' AND e.occurred_at >= ws.window_start
      AND e.service_point_id = ANY($1)
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
)
SELECT service_point_id, avg(wait_minutes)
FROM wait_pairs
WHERE service_point_id IS NOT NULL
GROUP BY service_point_id
`

// QueueStats implements the patient queue read (#101). Two focused queries
// (current counts, windowed averages) joined in Go: each is meaningful on
// its own and the map merge is trivial, which one UNION'd query would not
// be. Points with neither waiting steps nor samples stay absent — callers
// treat the zero value as the honest "nobody ahead, no data yet".
func (r *Repo) QueueStats(ctx context.Context, excludeVisitID string, servicePointIDs []string, tz string) (map[string]journey.SPQueueStats, error) {
	q := r.database.Querier(ctx)
	stats := map[string]journey.SPQueueStats{}

	rows, err := q.Query(ctx, queueWaitingQuery, excludeVisitID, servicePointIDs)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: queue waiting counts")
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var waiting int
		if err := rows.Scan(&id, &waiting); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: scan queue waiting count")
		}
		stats[id] = journey.SPQueueStats{WaitingAhead: waiting}
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate queue waiting counts")
	}

	avgRows, err := q.Query(ctx, queueAvgWaitQuery, servicePointIDs, tz)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: queue average waits")
	}
	defer avgRows.Close()
	for avgRows.Next() {
		var id string
		var avg *float64
		if err := avgRows.Scan(&id, &avg); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: scan queue average wait")
		}
		s := stats[id] // waiting count may already be here; keep it
		s.AvgWaitMinutes = avg
		stats[id] = s
	}
	if err := avgRows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate queue average waits")
	}
	return stats, nil
}
