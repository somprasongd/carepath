// Package db provides the PostgreSQL connection pool and transaction
// management shared by every module's postgres adapter.
//
// Transactions travel in context.Context: a service opens a transaction with
// WithinTransaction and passes the returned ctx to repositories and to other
// modules' services. Repositories call Querier(ctx) to obtain either the
// ambient transaction or the pool. A nested WithinTransaction call joins the
// ambient transaction instead of starting a new one, so cross-module service
// calls made inside a transaction share it without any module knowing how
// transactions are implemented.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the subset of pgx query methods satisfied by both *pgxpool.Pool
// and pgx.Tx, letting repositories run the same SQL inside or outside a
// transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Transactor is the port services depend on to control transaction
// boundaries. Implementations must run fn with a ctx whose Querier resolves
// to a single transaction, committing when fn succeeds and rolling back when
// it fails. If ctx already carries a transaction, fn joins it unchanged.
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type txKey struct{}

// DB owns the connection pool and implements Transactor.
type DB struct {
	pool *pgxpool.Pool
}

var (
	_ Querier    = (*pgxpool.Pool)(nil)
	_ Querier    = (pgx.Tx)(nil)
	_ Transactor = (*DB)(nil)
)

// New creates the pool and verifies connectivity with a ping.
func New(ctx context.Context, databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

func (d *DB) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// Querier returns the transaction carried by ctx, or the pool when ctx has none.
func (d *DB) Querier(ctx context.Context) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return d.pool
}
