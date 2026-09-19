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

// UpsertVisit replaces the visit's projection: the visit row is upserted and
// its steps are deleted and re-inserted from the snapshot, so replaying the
// same snapshot never duplicates or keeps stale steps.
func (r *Repo) UpsertVisit(ctx context.Context, visit journey.Visit) error {
	q := r.database.Querier(ctx)
	_, err := q.Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, status, synced_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (visit_id) DO UPDATE
		 SET patient_ref = EXCLUDED.patient_ref,
		     status = EXCLUDED.status,
		     synced_at = now()`,
		visit.VisitID, visit.PatientRef, visit.Status,
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
		if _, err := q.Exec(ctx,
			`INSERT INTO carepath.journey_step (visit_id, sequence, service_code, status, service_point_id)
			 VALUES ($1, $2, $3, $4, $5)`,
			visit.VisitID, step.Sequence, step.ServiceCode, step.Status, step.ServicePointID,
		); err != nil {
			return apperr.Wrapf(apperr.KindInternal, err,
				"journey: insert step %d of %s", step.Sequence, visit.VisitID)
		}
	}
	return nil
}

func (r *Repo) GetVisit(ctx context.Context, visitID string) (journey.Visit, error) {
	var visit journey.Visit
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT visit_id, patient_ref, status, synced_at
		 FROM carepath.journey_visit WHERE visit_id = $1`, visitID,
	).Scan(&visit.VisitID, &visit.PatientRef, &visit.Status, &visit.SyncedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return journey.Visit{}, journey.ErrNotFound
	}
	if err != nil {
		return journey.Visit{}, apperr.Wrapf(apperr.KindInternal, err, "journey: get visit %s", visitID)
	}

	rows, err := r.database.Querier(ctx).Query(ctx,
		`SELECT sequence, service_code, status, service_point_id
		 FROM carepath.journey_step WHERE visit_id = $1 ORDER BY sequence`, visitID,
	)
	if err != nil {
		return journey.Visit{}, apperr.Wrapf(apperr.KindInternal, err, "journey: list steps of %s", visitID)
	}
	defer rows.Close()

	for rows.Next() {
		var step journey.Step
		if err := rows.Scan(&step.Sequence, &step.ServiceCode, &step.Status, &step.ServicePointID); err != nil {
			return journey.Visit{}, apperr.Wrapf(apperr.KindInternal, err, "journey: scan step of %s", visitID)
		}
		visit.Steps = append(visit.Steps, step)
	}
	if err := rows.Err(); err != nil {
		return journey.Visit{}, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate steps of %s", visitID)
	}
	return visit, nil
}

// ListVisits returns every projected visit with its steps, freshest sync
// first (#37). Two queries and a Go-side group-join keep it simple at demo
// scale; ordering (synced_at DESC, visit_id as the tiebreaker) is part of
// this port's contract.
func (r *Repo) ListVisits(ctx context.Context) ([]journey.Visit, error) {
	visitRows, err := r.database.Querier(ctx).Query(ctx,
		`SELECT visit_id, patient_ref, status, synced_at
		 FROM carepath.journey_visit ORDER BY synced_at DESC, visit_id`)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: list visits")
	}
	defer visitRows.Close()

	visits := make([]journey.Visit, 0)
	for visitRows.Next() {
		var visit journey.Visit
		if err := visitRows.Scan(&visit.VisitID, &visit.PatientRef, &visit.Status, &visit.SyncedAt); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: scan visit row")
		}
		visits = append(visits, visit)
	}
	if err := visitRows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "journey: iterate visits")
	}

	stepRows, err := r.database.Querier(ctx).Query(ctx,
		`SELECT visit_id, sequence, service_code, status, service_point_id
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
		if err := stepRows.Scan(&visitID, &step.Sequence, &step.ServiceCode, &step.Status, &step.ServicePointID); err != nil {
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

// InsertCommandAudit records one forwarded transition command (#19 AC4).
// command_id is the primary key, so replaying the same id — e.g. a client
// retry after a failed local transaction — is a no-op, not a duplicate row.
func (r *Repo) InsertCommandAudit(ctx context.Context, audit journey.CommandAudit) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.journey_command_audit (command_id, visit_id, sequence, to_status, source)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (command_id) DO NOTHING`,
		audit.CommandID, audit.VisitID, audit.Sequence, audit.ToStatus, audit.Source,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err,
			"journey: audit command %s of %s", audit.CommandID, audit.VisitID)
	}
	return nil
}
