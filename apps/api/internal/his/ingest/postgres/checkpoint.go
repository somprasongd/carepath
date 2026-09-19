// Package postgres is the ingest adapter's checkpoint storage over pgx: the
// durable feed cursor kept in the carepath.his_ingest_state singleton row.
package postgres

import (
	"context"

	"carepath/apps/api/internal/his/ingest"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type Checkpoint struct {
	database *db.DB
}

var _ ingest.Checkpoint = (*Checkpoint)(nil)

func New(database *db.DB) *Checkpoint {
	return &Checkpoint{database: database}
}

func (c *Checkpoint) LastEventID(ctx context.Context) (string, error) {
	var last string
	err := c.database.Querier(ctx).QueryRow(ctx,
		`SELECT last_event_id FROM carepath.his_ingest_state WHERE singleton`,
	).Scan(&last)
	if err != nil {
		return "", apperr.Wrapf(apperr.KindInternal, err, "ingest: load cursor")
	}
	return last, nil
}

func (c *Checkpoint) Save(ctx context.Context, lastEventID string) error {
	_, err := c.database.Querier(ctx).Exec(ctx,
		`UPDATE carepath.his_ingest_state
		 SET last_event_id = $1, updated_at = now() WHERE singleton`, lastEventID,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "ingest: save cursor %s", lastEventID)
	}
	return nil
}
