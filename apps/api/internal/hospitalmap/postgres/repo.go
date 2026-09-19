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
// scans them directly.
const selectColumns = `p.place_id, p.floor_id, p.name, p.place_type, p.x::float8, p.y::float8, p.entry_node_id,
	f.floor_id, f.building_id, f.code, f.name, f.level_order`

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
		&p.Floor.ID, &p.Floor.BuildingID, &p.Floor.Code, &p.Floor.Name, &p.Floor.LevelOrder,
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
