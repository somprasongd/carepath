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

// ListForUser returns the active service points assigned to the user
// through carepath.user_service_point (#102). Points assigned but since
// deactivated stay hidden — the queue console must not offer a dead point.
func (r *Repo) ListForUser(ctx context.Context, userID string) ([]servicepoint.ServicePoint, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		"SELECT "+selectColumns+" FROM carepath.service_point sp"+
			" JOIN carepath.user_service_point usp ON usp.service_point_id = sp.id"+
			" WHERE usp.user_id = $1 AND sp.active ORDER BY sp.code", userID)
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: list for user %q", userID)
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

// IsAssigned reports whether the user_service_point row exists. A point that
// does not exist answers false like a point the user simply lacks — callers
// ask "may this user work this queue", not "does the point exist".
func (r *Repo) IsAssigned(ctx context.Context, userID, servicePointID string) (bool, error) {
	var assigned bool
	err := r.database.Querier(ctx).QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM carepath.user_service_point WHERE user_id = $1 AND service_point_id = $2)",
		userID, servicePointID,
	).Scan(&assigned)
	if err != nil {
		return false, apperr.Wrapf(apperr.KindInternal, err, "servicepoint: is assigned")
	}
	return assigned, nil
}
