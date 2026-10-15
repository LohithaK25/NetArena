package sim

import (
	"fmt"
)

// Router selects destination nodes using deterministic round-robin routing.
type Router struct {
	nodes    map[string]Node
	outgoing map[string][]string
	cursors  map[string]int
}

// NewRouter validates the topology and builds a router from a snapshot of it.
func NewRouter(topology Topology) (*Router, error) {
	if err := topology.Validate(); err != nil {
		return nil, fmt.Errorf("invalid topology: %w", err)
	}

	router := &Router{
		nodes:    make(map[string]Node, len(topology.Nodes)),
		outgoing: make(map[string][]string, len(topology.Nodes)),
		cursors:  make(map[string]int),
	}

	for _, node := range topology.Nodes {
		router.nodes[node.ID] = node
	}

	// Preserve connection order so routing remains deterministic.
	for _, connection := range topology.Connections {
		router.outgoing[connection.From] = append(
			router.outgoing[connection.From],
			connection.To,
		)
	}

	return router, nil
}

// NextHop selects the next destination for the given source node.
// Each source maintains an independent round-robin cursor.
func (r *Router) NextHop(from string) (Node, error) {
	if _, exists := r.nodes[from]; !exists {
		return Node{}, fmt.Errorf("unknown source node %q", from)
	}

	destinations := r.outgoing[from]
	if len(destinations) == 0 {
		return Node{}, fmt.Errorf("node %q has no outgoing connections", from)
	}

	cursor := r.cursors[from] % len(destinations)
	destinationID := destinations[cursor]
	r.cursors[from] = (cursor + 1) % len(destinations)

	return r.nodes[destinationID], nil
}
