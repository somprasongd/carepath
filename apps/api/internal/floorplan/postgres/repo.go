// Package postgres is the floorplan module's persistence adapter over pgx.
// All queries resolve their connection from the ambient context, so they
// transparently run inside the caller's transaction when one is open.
package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/floorplan"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type Repo struct {
	database *db.DB
}

var _ floorplan.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

// SVG reads a plan by floor and digest. Rows are append-only, so a hit here
// is immutable — which is what the serving handler's year-long cache header
// rests on.
func (r *Repo) SVG(ctx context.Context, floorID, sha256 string) (string, error) {
	var svg string
	err := r.database.Querier(ctx).QueryRow(ctx,
		"SELECT svg FROM carepath.floor_plan WHERE floor_id = $1 AND sha256 = $2",
		floorID, sha256).Scan(&svg)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", floorplan.ErrPlanNotFound
	}
	if err != nil {
		return "", apperr.Wrapf(apperr.KindInternal, err, "floorplan: read plan for floor %q", floorID)
	}
	return svg, nil
}

// Insert stores a plan. ON CONFLICT DO NOTHING makes re-uploading an
// unchanged file idempotent rather than an error: the plan id is derived
// from the content, so the row that is already there *is* the upload.
func (r *Repo) Insert(ctx context.Context, plan floorplan.Stored, svg, svgRaw string) error {
	warnings, err := json.Marshal(plan.Warnings)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "floorplan: encode warnings")
	}
	_, err = r.database.Querier(ctx).Exec(ctx, `
INSERT INTO carepath.floor_plan (plan_id, floor_id, sha256, viewbox, svg, svg_raw, warnings, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (plan_id) DO NOTHING`,
		plan.PlanID, plan.FloorID, plan.SHA256, plan.ViewBox, svg, svgRaw, warnings, plan.CreatedBy)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "floorplan: insert plan for floor %q", plan.FloorID)
	}
	return nil
}

const storedColumns = `plan_id, floor_id, sha256, viewbox, warnings, created_at, created_by`

func scanStored(row interface{ Scan(...any) error }) (floorplan.Stored, error) {
	var (
		plan     floorplan.Stored
		warnings []byte
	)
	if err := row.Scan(&plan.PlanID, &plan.FloorID, &plan.SHA256, &plan.ViewBox,
		&warnings, &plan.CreatedAt, &plan.CreatedBy); err != nil {
		return floorplan.Stored{}, err
	}
	if err := json.Unmarshal(warnings, &plan.Warnings); err != nil {
		return floorplan.Stored{}, err
	}
	return plan, nil
}

func (r *Repo) Get(ctx context.Context, floorID, planID string) (floorplan.Stored, error) {
	plan, err := scanStored(r.database.Querier(ctx).QueryRow(ctx,
		"SELECT "+storedColumns+" FROM carepath.floor_plan WHERE floor_id = $1 AND plan_id = $2",
		floorID, planID))
	if errors.Is(err, pgx.ErrNoRows) {
		return floorplan.Stored{}, floorplan.ErrPlanNotFound
	}
	if err != nil {
		return floorplan.Stored{}, apperr.Wrapf(apperr.KindInternal, err, "floorplan: get plan %q", planID)
	}
	return plan, nil
}

// ListByFloor returns the floor's plans newest first — the order a rollback
// reads them in.
func (r *Repo) ListByFloor(ctx context.Context, floorID string) ([]floorplan.Stored, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		"SELECT "+storedColumns+" FROM carepath.floor_plan WHERE floor_id = $1 ORDER BY created_at DESC, plan_id",
		floorID)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "floorplan: list plans for floor %q", floorID)
	}
	defer rows.Close()

	plans := make([]floorplan.Stored, 0)
	for rows.Next() {
		plan, err := scanStored(rows)
		if err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "floorplan: scan plan")
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "floorplan: iterate plans")
	}
	return plans, nil
}

// Raw returns the bytes as uploaded. This column never reaches a patient —
// only the normalized svg does.
func (r *Repo) Raw(ctx context.Context, floorID, planID string) (string, error) {
	var raw string
	err := r.database.Querier(ctx).QueryRow(ctx,
		"SELECT svg_raw FROM carepath.floor_plan WHERE floor_id = $1 AND plan_id = $2",
		floorID, planID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", floorplan.ErrPlanNotFound
	}
	if err != nil {
		return "", apperr.Wrapf(apperr.KindInternal, err, "floorplan: read raw plan %q", planID)
	}
	return raw, nil
}

func (r *Repo) Digests(ctx context.Context, planIDs []string) (map[string]string, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		"SELECT plan_id, sha256 FROM carepath.floor_plan WHERE plan_id = ANY($1)", planIDs)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "floorplan: read plan digests")
	}
	defer rows.Close()

	digests := make(map[string]string, len(planIDs))
	for rows.Next() {
		var planID, sha string
		if err := rows.Scan(&planID, &sha); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "floorplan: scan digest")
		}
		digests[planID] = sha
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "floorplan: iterate digests")
	}
	return digests, nil
}
