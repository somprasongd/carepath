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
		`INSERT INTO carepath.journey_command_audit (command_id, visit_id, step_key, to_status, source)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (command_id) DO NOTHING`,
		audit.CommandID, audit.VisitID, audit.StepKey, audit.ToStatus, audit.Source,
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
