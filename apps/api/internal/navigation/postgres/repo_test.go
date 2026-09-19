package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"carepath/apps/api/internal/navigation"
	"carepath/apps/api/internal/platform/db"
)

// Integration test against a real Postgres. Requires the schema and seed
// data from infra/postgres/migrations; run `make migrate-up` first.
func newDB(t *testing.T) *db.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (run make up / migrate-up first)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	return database
}

// floorGraph mirrors the hand-authored shape of
// packages/floorplans/graphs/*.json — the source of truth the migration
// seeds from. Parsing it here makes this test a drift guard: if the DB and
// the authored graph disagree, routing (#27) would silently navigate a
// hospital that no longer matches the plans.
type floorGraph struct {
	FloorID string `json:"floorId"`
	Nodes   []struct {
		ID   string  `json:"id"`
		X    float64 `json:"x"`
		Y    float64 `json:"y"`
		Type string  `json:"type"`
		Zone *string `json:"zone"`
	} `json:"nodes"`
	Edges []struct {
		From       string  `json:"from"`
		To         string  `json:"to"`
		Distance   float64 `json:"distance"`
		Accessible bool    `json:"accessible"`
	} `json:"edges"`
	Transitions []struct {
		From       string  `json:"from"`
		ToFloorID  string  `json:"toFloorId"`
		To         string  `json:"to"`
		Type       string  `json:"type"`
		Distance   float64 `json:"distance"`
		Accessible bool    `json:"accessible"`
	} `json:"transitions"`
}

func gid(floorID, localID string) string { return floorID + "/" + localID }

func loadFloorGraphs(t *testing.T) []floorGraph {
	t.Helper()
	dir := filepath.Join("..", "..", "..", "..", "..", "packages", "floorplans", "graphs")
	var graphs []floorGraph
	for _, name := range []string{"i-1301.json", "i-1302.json"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v (the graph JSON is the seed source of truth)", name, err)
		}
		var g floorGraph
		if err := json.Unmarshal(raw, &g); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		graphs = append(graphs, g)
	}
	return graphs
}

// The seeded graph must be exactly the hand-authored packages/floorplans
// graph: every node with its floor, coordinate, type and zone; every
// connection as two directed rows with the authored cost — never derived
// from SVG geometry at runtime.
func TestSeedMatchesFloorplanGraphs(t *testing.T) {
	repo := New(newDB(t))
	graphs := loadFloorGraphs(t)
	ctx := context.Background()

	for _, g := range graphs {
		for _, n := range g.Nodes {
			node, err := repo.GetNode(ctx, gid(g.FloorID, n.ID))
			if err != nil {
				t.Fatalf("GetNode(%s): %v", gid(g.FloorID, n.ID), err)
			}
			if node.FloorID != g.FloorID || node.X != n.X || node.Y != n.Y ||
				node.NodeType != n.Type || !zoneEqual(node.Zone, n.Zone) {
				t.Errorf("node %s = %+v, want floor %s at (%v,%v) type %s zone %v",
					n.ID, node, g.FloorID, n.X, n.Y, n.Type, n.Zone)
			}
		}
	}

	nodes, err := repo.ListNodes(ctx)
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	wantNodes := 0
	for _, g := range graphs {
		wantNodes += len(g.Nodes)
	}
	if len(nodes) != wantNodes {
		t.Errorf("ListNodes returned %d nodes, want exactly the %d authored ones", len(nodes), wantNodes)
	}

	edges, err := repo.ListEdges(ctx)
	if err != nil {
		t.Fatalf("ListEdges: %v", err)
	}
	byKey := make(map[string]navigation.NavEdge, len(edges))
	for _, e := range edges {
		byKey[e.FromNodeID+"|"+e.ToNodeID] = e
	}
	wantEdges := make(map[string]navigation.NavEdge)
	for _, g := range graphs {
		for _, e := range g.Edges {
			addDirection(t, wantEdges, g.FloorID, e.From, g.FloorID, e.To, "CORRIDOR", e.Distance, e.Accessible)
			addDirection(t, wantEdges, g.FloorID, e.To, g.FloorID, e.From, "CORRIDOR", e.Distance, e.Accessible)
		}
		for _, tr := range g.Transitions {
			addDirection(t, wantEdges, g.FloorID, tr.From, tr.ToFloorID, tr.To, tr.Type, tr.Distance, tr.Accessible)
			addDirection(t, wantEdges, tr.ToFloorID, tr.To, g.FloorID, tr.From, tr.Type, tr.Distance, tr.Accessible)
		}
	}
	if len(edges) != len(wantEdges) {
		t.Errorf("ListEdges returned %d rows, want %d directed rows", len(edges), len(wantEdges))
	}
	for key, want := range wantEdges {
		got, ok := byKey[key]
		if !ok {
			t.Errorf("edge %s missing from seed", key)
			continue
		}
		if got.EdgeType != want.EdgeType || got.Distance != want.Distance || got.Accessible != want.Accessible {
			t.Errorf("edge %s = %+v, want type %s distance %v accessible %v",
				key, got, want.EdgeType, want.Distance, want.Accessible)
		}
	}
}

func addDirection(t *testing.T, dst map[string]navigation.NavEdge, fromFloor, from, toFloor, to, edgeType string, distance float64, accessible bool) {
	t.Helper()
	fromID, toID := gid(fromFloor, from), gid(toFloor, to)
	dst[fromID+"|"+toID] = navigation.NavEdge{
		FromNodeID: fromID, ToNodeID: toID,
		EdgeType: edgeType, Distance: distance, Accessible: accessible,
	}
}

func zoneEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// The service-point → destination → entry-node chain must end in a real
// graph node: this is the join patients navigate by (#24 AC, completed by
// #26's FK).
func TestServicePointsResolveToGraphNodes(t *testing.T) {
	database := newDB(t)
	repo := New(database)
	ctx := context.Background()

	rows, err := database.Querier(ctx).Query(ctx, `
		SELECT sp.code, p.entry_node_id
		FROM carepath.service_point sp
		JOIN carepath.place p ON p.place_id = sp.place_id
		WHERE sp.active`)
	if err != nil {
		t.Fatalf("query service points: %v", err)
	}
	defer rows.Close()

	resolved := make(map[string]string)
	for rows.Next() {
		var code, entryNode string
		if err := rows.Scan(&code, &entryNode); err != nil {
			t.Fatalf("scan: %v", err)
		}
		resolved[code] = entryNode
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate: %v", err)
	}

	for _, code := range []string{"REGISTRATION", "DOCTOR", "XRAY", "LAB", "PHARMACY"} {
		entryNode, ok := resolved[code]
		if !ok {
			t.Fatalf("active service point %s missing from seed", code)
		}
		if _, err := repo.GetNode(ctx, entryNode); err != nil {
			t.Errorf("service point %s entry node %s: %v", code, entryNode, err)
		}
	}
}

// The Happy Path demo journey — register, see the doctor, lab (upstairs via
// the elevator), X-ray, pharmacy — must be walkable on the seeded graph
// from the main entrance.
func TestHappyPathReachableFromMainEntrance(t *testing.T) {
	repo := New(newDB(t))
	ctx := context.Background()

	edges, err := repo.ListEdges(ctx)
	if err != nil {
		t.Fatalf("ListEdges: %v", err)
	}
	adjacency := make(map[string][]string)
	for _, e := range edges {
		adjacency[e.FromNodeID] = append(adjacency[e.FromNodeID], e.ToNodeID)
	}

	visited := map[string]bool{"I-1301/node-main-entrance": true}
	queue := []string{"I-1301/node-main-entrance"}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}

	for _, entry := range []string{
		"I-1301/node-reception",        // REGISTRATION
		"I-1301/node-opd-ns",           // DOCTOR
		"I-1301/node-xray",             // XRAY
		"I-1301/node-pharmacy",         // PHARMACY
		"I-1302/node-blood-collection", // LAB — cross-floor via the elevator transition
	} {
		if !visited[entry] {
			t.Errorf("happy-path entry node %s not reachable from the main entrance", entry)
		}
	}

	// The stairs transition stays flagged not accessible — routing (#27)
	// must be able to avoid it for wheelchair routes.
	for _, e := range edges {
		if e.EdgeType == "STAIRS" && e.Accessible {
			t.Errorf("stairs edge %s is accessible, want the authored false", e.ID)
		}
	}
}

func TestGetNodeNotFound(t *testing.T) {
	repo := New(newDB(t))

	if _, err := repo.GetNode(context.Background(), "I-1301/node-nope"); !errors.Is(err, navigation.ErrNodeNotFound) {
		t.Fatalf("error = %v, want navigation.ErrNodeNotFound", err)
	}
}
