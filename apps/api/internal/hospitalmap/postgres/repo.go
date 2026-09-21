// Package postgres is the hospitalmap module's persistence adapter over
// pgx. All queries resolve their connection from the ambient context, so
// they transparently run inside the caller's transaction when one is open.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/hospitalmap"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

// x/y are numeric in the table but plain float64 in Go; cast in SQL so pgx
// scans them directly. f.viewbox is included so Place.Floor.ViewBox is
// actually populated — the contract has declared it since #105, but this
// query used to leave it off, so it always scanned as nil.
const selectColumns = `p.place_id, p.floor_id, p.name, p.place_type, p.x::float8, p.y::float8, p.entry_node_id,
	f.floor_id, f.building_id, f.code, f.name, f.level_order, f.viewbox`

const selectFrom = `
FROM carepath.place p
JOIN carepath.floor f ON f.floor_id = p.floor_id`

type Repo struct {
	database *db.DB
}

var _ hospitalmap.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

// scanner covers both pgx.Row (single) and pgx.Rows (many).
type scanner interface {
	Scan(dest ...any) error
}

func scanPlace(row scanner) (hospitalmap.Place, error) {
	var p hospitalmap.Place
	p.Floor = &hospitalmap.Floor{}
	if err := row.Scan(
		&p.ID, &p.FloorID, &p.Name, &p.PlaceType, &p.X, &p.Y, &p.EntryNodeID,
		&p.Floor.ID, &p.Floor.BuildingID, &p.Floor.Code, &p.Floor.Name, &p.Floor.LevelOrder, &p.Floor.ViewBox,
	); err != nil {
		return hospitalmap.Place{}, err
	}
	return p, nil
}

func (r *Repo) GetPlace(ctx context.Context, placeID string) (hospitalmap.Place, error) {
	place, err := scanPlace(r.database.Querier(ctx).QueryRow(ctx,
		"SELECT "+selectColumns+selectFrom+" WHERE p.place_id = $1", placeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return hospitalmap.Place{}, hospitalmap.ErrPlaceNotFound
	}
	if err != nil {
		return hospitalmap.Place{}, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: get place %q", placeID)
	}
	return place, nil
}

func (r *Repo) ListPlaces(ctx context.Context) ([]hospitalmap.Place, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		"SELECT "+selectColumns+selectFrom+" ORDER BY p.place_id")
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: list places")
	}
	defer rows.Close()

	var places []hospitalmap.Place
	for rows.Next() {
		place, err := scanPlace(rows)
		if err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: scan row")
		}
		places = append(places, place)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: iterate rows")
	}
	return places, nil
}

func (r *Repo) GetFloor(ctx context.Context, floorID string) (hospitalmap.Floor, error) {
	var f hospitalmap.Floor
	err := r.database.Querier(ctx).QueryRow(ctx, `
SELECT floor_id, building_id, code, name, level_order, viewbox, active_plan_id
FROM carepath.floor
WHERE floor_id = $1`, floorID).
		Scan(&f.ID, &f.BuildingID, &f.Code, &f.Name, &f.LevelOrder, &f.ViewBox, &f.ActivePlanID)
	if errors.Is(err, pgx.ErrNoRows) {
		return hospitalmap.Floor{}, hospitalmap.ErrFloorNotFound
	}
	if err != nil {
		return hospitalmap.Floor{}, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: get floor %q", floorID)
	}
	return f, nil
}

// SetPlan moves the floor's plan pointer and pins its coordinate space.
// viewbox is written every time but can only ever be written with the value
// it already holds — the floorplan module rejects an upload that disagrees
// with it, so this is how the first plan sets it and later ones confirm it.
func (r *Repo) SetPlan(ctx context.Context, floorID, viewBox, planID string) error {
	tag, err := r.database.Querier(ctx).Exec(ctx, `
UPDATE carepath.floor SET viewbox = $2, active_plan_id = $3 WHERE floor_id = $1`,
		floorID, viewBox, planID)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: set plan on floor %q", floorID)
	}
	if tag.RowsAffected() == 0 {
		return hospitalmap.ErrFloorNotFound
	}
	return nil
}

// ListFloors returns floors in the vertical order the UI shows them in.
// viewbox and active_plan_id are NULL until a floor has its first plan
// (migration 000023), which is the honest "no drawing yet" state.
func (r *Repo) ListFloors(ctx context.Context) ([]hospitalmap.Floor, error) {
	rows, err := r.database.Querier(ctx).Query(ctx, `
SELECT floor_id, building_id, code, name, level_order, viewbox, active_plan_id
FROM carepath.floor
ORDER BY level_order, floor_id`)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: list floors")
	}
	defer rows.Close()

	var floors []hospitalmap.Floor
	for rows.Next() {
		var f hospitalmap.Floor
		if err := rows.Scan(&f.ID, &f.BuildingID, &f.Code, &f.Name, &f.LevelOrder,
			&f.ViewBox, &f.ActivePlanID); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: scan floor")
		}
		floors = append(floors, f)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "hospitalmap: iterate floors")
	}
	return floors, nil
}
