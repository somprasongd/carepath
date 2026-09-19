package navigation

import (
	"container/heap"
	"math"

	"carepath/apps/api/internal/platform/apperr"
)

// shortestRoute returns the cheapest walk from → to over the given directed,
// costed edges using Dijkstra — the graph is two floors and a few dozen
// nodes, so an A* heuristic is not worth its coupling yet (see
// docs/architecture/technical-blueprint.md). It is a pure function over the
// in-memory graph so it stays unit-testable without a database and callable
// from any transport (#28's HTTP API included). Unknown endpoints yield
// ErrNodeNotFound; a missing connection yields ErrNoRoute.
func shortestRoute(nodes []NavNode, edges []NavEdge, from, to string, opts RouteOptions) (Route, error) {
	byID := make(map[string]NavNode, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}
	if _, ok := byID[from]; !ok {
		return Route{}, ErrNodeNotFound
	}
	if _, ok := byID[to]; !ok {
		return Route{}, ErrNodeNotFound
	}

	adjacency := make(map[string][]NavEdge)
	for _, edge := range edges {
		// Dijkstra silently returns wrong answers on negative costs, and
		// the authored distances are never negative — treat it as corrupt
		// graph data rather than navigating it.
		if edge.Distance < 0 {
			return Route{}, apperr.New(apperr.KindInternal, "navigation: negative distance on edge "+edge.ID)
		}
		if opts.AccessibleOnly && !edge.Accessible {
			continue
		}
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], edge)
	}

	// dist/enteredBy/done follow the lazy-deletion Dijkstra variant: the
	// heap may hold stale entries for an already-settled node, which `done`
	// skips on pop.
	dist := make(map[string]float64, len(nodes))
	for id := range byID {
		dist[id] = math.Inf(1)
	}
	dist[from] = 0
	enteredBy := make(map[string]NavEdge, len(nodes))
	done := make(map[string]bool, len(nodes))

	pq := &priorityQueue{{nodeID: from}}
	heap.Init(pq)
	for pq.Len() > 0 {
		current := heap.Pop(pq).(pqItem)
		if done[current.nodeID] {
			continue
		}
		done[current.nodeID] = true
		if current.nodeID == to {
			break
		}
		for _, edge := range adjacency[current.nodeID] {
			if done[edge.ToNodeID] {
				continue
			}
			if next := dist[current.nodeID] + edge.Distance; next < dist[edge.ToNodeID] {
				dist[edge.ToNodeID] = next
				enteredBy[edge.ToNodeID] = edge
				heap.Push(pq, pqItem{nodeID: edge.ToNodeID, dist: next})
			}
		}
	}

	if !done[to] {
		return Route{}, ErrNoRoute
	}

	segments := make([]NavEdge, 0, len(done))
	for id := to; id != from; {
		edge, ok := enteredBy[id]
		if !ok {
			// Unreachable with the invariants above: every settled node
			// other than the origin was relaxed through some edge. Guard
			// anyway so a future edit fails loudly instead of hanging.
			return Route{}, apperr.New(apperr.KindInternal, "navigation: lost path bookkeeping at "+id)
		}
		segments = append(segments, edge)
		id = edge.FromNodeID
	}
	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}

	route := Route{
		Nodes:         make([]NavNode, 0, len(segments)+1),
		Segments:      segments,
		TotalDistance: dist[to],
	}
	route.Nodes = append(route.Nodes, byID[from])
	for _, edge := range segments {
		route.Nodes = append(route.Nodes, byID[edge.ToNodeID])
	}
	return route, nil
}

// pqItem is one frontier entry: a node and the cheapest known cost to it.
type pqItem struct {
	nodeID string
	dist   float64
}

type priorityQueue []pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }
func (pq *priorityQueue) Push(x any)        { *pq = append(*pq, x.(pqItem)) }
func (pq *priorityQueue) Pop() any {
	old := *pq
	item := old[len(old)-1]
	*pq = old[:len(old)-1]
	return item
}
