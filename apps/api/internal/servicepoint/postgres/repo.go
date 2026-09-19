// Package postgres is the servicepoint module's persistence adapter over
// pgx. All queries resolve their connection from the ambient context, so
// they transparently run inside the caller's transaction when one is open.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/servicepoint"
)

const selectColumns = "id, code, name, place_id, active"

type Repo struct {
	database *db.DB
}

var _ servicepoint.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

func (r *Repo) GetByCode(ctx context.Context, code string) (servicepoint.ServicePoint, error) {
	var sp servicepoint.ServicePoint
	err := r.database.Querier(ctx).QueryRow(ctx,
		"SELECT "+selectColumns+" FROM carepath.service_point WHERE code = $1 AND active",
		code,
	).Scan(&sp.ID, &sp.Code, &sp.Name, &sp.PlaceID, &sp.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return servicepoint.ServicePoint{}, servicepoint.ErrNotFound
	}
	if err != nil {
		return servicepoint.ServicePoint{}, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: get by code %q", code)
	}
	return sp, nil
}

func (r *Repo) List(ctx context.Context) ([]servicepoint.ServicePoint, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		"SELECT "+selectColumns+" FROM carepath.service_point WHERE active ORDER BY code")
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: list")
	}
	defer rows.Close()

	var points []servicepoint.ServicePoint
	for rows.Next() {
		var sp servicepoint.ServicePoint
		if err := rows.Scan(&sp.ID, &sp.Code, &sp.Name, &sp.PlaceID, &sp.Active); err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: scan row")
		}
		points = append(points, sp)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: iterate rows")
	}
	return points, nil
}
