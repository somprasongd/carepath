package navigation

import (
	"context"
	"errors"
	"testing"
)

// fakeRepo mirrors the seed rows from infra/postgres/migrations/000008.
type fakeRepo struct {
	nodes map[string]NavNode
	edges []NavEdge
}

func (f *fakeRepo) GetNode(_ context.Context, nodeID string) (NavNode, error) {
	node, ok := f.nodes[nodeID]
	if !ok {
		return NavNode{}, ErrNodeNotFound
	}
	return node, nil
}

func (f *fakeRepo) ListNodes(context.Context) ([]NavNode, error) {
	var nodes []NavNode
	for _, node := range f.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (f *fakeRepo) ListEdges(context.Context) ([]NavEdge, error) {
	return f.edges, nil
}

func TestGetNodeCarriesFloorAndCoordinate(t *testing.T) {
	svc := NewService(&fakeRepo{nodes: map[string]NavNode{
		"I-1302/node-blood-collection": {
			ID: "I-1302/node-blood-collection", FloorID: "I-1302", X: 165, Y: 405,
			NodeType: "PLACE_ENTRY",
		},
	}})

	node, err := svc.GetNode(context.Background(), "I-1302/node-blood-collection")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if node.FloorID != "I-1302" || node.X != 165 || node.Y != 405 || node.NodeType != "PLACE_ENTRY" {
		t.Fatalf("node = %+v, want I-1302 at (165,405) PLACE_ENTRY", node)
	}
}

func TestGetNodeUnknownIDPropagatesNotFound(t *testing.T) {
	svc := NewService(&fakeRepo{nodes: map[string]NavNode{}})

	if _, err := svc.GetNode(context.Background(), "I-1301/node-nope"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("error = %v, want ErrNodeNotFound", err)
	}
}

func TestListEdgesCarriesCost(t *testing.T) {
	svc := NewService(&fakeRepo{edges: []NavEdge{
		{ID: "a>b", FromNodeID: "a", ToNodeID: "b", EdgeType: "CORRIDOR", Distance: 200, Accessible: true},
		{ID: "b>a", FromNodeID: "b", ToNodeID: "a", EdgeType: "CORRIDOR", Distance: 200, Accessible: true},
	}})

	edges, err := svc.ListEdges(context.Background())
	if err != nil {
		t.Fatalf("ListEdges: %v", err)
	}
	if len(edges) != 2 || edges[0].Distance != 200 || !edges[0].Accessible {
		t.Fatalf("edges = %+v, want 2 directed rows with distance 200 accessible", edges)
	}
}
