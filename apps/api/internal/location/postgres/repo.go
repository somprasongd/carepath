// Package postgres is the location module's persistence adapter over pgx.
// All queries resolve their connection from the ambient context, so they
// transparently run inside the caller's transaction when one is open.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/location"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type Repo struct {
	database *db.DB
}

var _ location.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

func (r *Repo) Record(ctx context.Context, obs location.Observation) error {
	_, err := r.database.Querier(ctx).Exec(ctx, `
		INSERT INTO carepath.location_observation
			(visit_id, source, node_id, floor_id, zone, confidence, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		obs.VisitID, string(obs.Source), obs.NodeID, obs.FloorID, obs.Zone, obs.Confidence, obs.ObservedAt)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "location: record observation for visit %q", obs.VisitID)
	}
	return nil
}

func (r *Repo) Latest(ctx context.Context, visitID string) (location.Observation, error) {
	var (
		obs    location.Observation
		source string
	)
	err := r.database.Querier(ctx).QueryRow(ctx, `
		SELECT o.visit_id, o.source, o.node_id, o.floor_id, o.zone, o.confidence, o.observed_at
		FROM carepath.location_observation o
		WHERE o.visit_id = $1
		ORDER BY o.id DESC
		LIMIT 1`, visitID).
		Scan(&obs.VisitID, &source, &obs.NodeID, &obs.FloorID, &obs.Zone, &obs.Confidence, &obs.ObservedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return location.Observation{}, location.ErrNoLocation
	}
	if err != nil {
		return location.Observation{}, apperr.Wrapf(apperr.KindInternal, err, "location: latest observation for visit %q", visitID)
	}
	obs.Source = location.Source(source)
	return obs, nil
}
