// Package postgres is the navigation module's persistence adapter over pgx.
// All queries resolve their connection from the ambient context, so they
// transparently run inside the caller's transaction when one is open.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

// x/y and distance are numeric in the tables but plain float64 in Go; cast
// in SQL so pgx scans them directly.
const selectNode = `n.node_id, n.floor_id, n.x::float8, n.y::float8, n.node_type, n.zone`
const selectEdge = `e.edge_id, e.from_node_id, e.to_node_id, e.edge_type, e.distance::float8, e.accessible`

type Repo struct {
	database *db.DB
}

var _ navigation.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

// scanner covers both pgx.Row (single) and pgx.Rows (many).
type scanner interface {
	Scan(dest ...any) error
}

func scanNode(row scanner) (navigation.NavNode, error) {
	var n navigation.NavNode
	if err := row.Scan(&n.ID, &n.FloorID, &n.X, &n.Y, &n.NodeType, &n.Zone); err != nil {
		return navigation.NavNode{}, err
	}
	return n, nil
}

func scanEdge(row scanner) (navigation.NavEdge, error) {
	var e navigation.NavEdge
	if err := row.Scan(&e.ID, &e.FromNodeID, &e.ToNodeID, &e.EdgeType, &e.Distance, &e.Accessible); err != nil {
		return navigation.NavEdge{}, err
	}
	return e, nil
}

func (r *Repo) GetNode(ctx context.Context, nodeID string) (navigation.NavNode, error) {
	node, err := scanNode(r.database.Querier(ctx).QueryRow(ctx,
		"SELECT "+selectNode+" FROM carepath.nav_node n WHERE n.node_id = $1", nodeID))
	if errors.Is(err, pgx.ErrNoRows) {
		return navigation.NavNode{}, navigation.ErrNodeNotFound
	}
	if err != nil {
		return navigation.NavNode{}, apperr.Wrapf(apperr.KindInternal, err, "navigation: get node %q", nodeID)
	}
	return node, nil
}

func (r *Repo) ListNodes(ctx context.Context) ([]navigation.NavNode, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		"SELECT "+selectNode+" FROM carepath.nav_node n ORDER BY n.node_id")
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "navigation: list nodes")
	}
	defer rows.Close()

	var nodes []navigation.NavNode
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "navigation: scan node row")
		}
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "navigation: iterate node rows")
	}
	return nodes, nil
}

func (r *Repo) ListEdges(ctx context.Context) ([]navigation.NavEdge, error) {
	rows, err := r.database.Querier(ctx).Query(ctx,
		"SELECT "+selectEdge+" FROM carepath.nav_edge e ORDER BY e.edge_id")
	if err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "navigation: list edges")
	}
	defer rows.Close()

	var edges []navigation.NavEdge
	for rows.Next() {
		edge, err := scanEdge(rows)
		if err != nil {
			return nil, apperr.Wrapf(apperr.KindInternal, err, "navigation: scan edge row")
		}
		edges = append(edges, edge)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrapf(apperr.KindInternal, err, "navigation: iterate edge rows")
	}
	return edges, nil
}
