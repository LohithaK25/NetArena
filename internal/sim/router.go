package sim

import (
	"fmt"
)

type routeLink struct {
	to      string
	latency Tick
}

// Router selects destination nodes using deterministic round-robin routing.
type Router struct {
	nodes    map[string]Node
	outgoing map[string][]routeLink
	cursors  map[string]int
}

// NewRouter validates the topology and builds a router from a snapshot of it.
func NewRouter(topology Topology) (*Router, error) {
	if err := topology.Validate(); err != nil {
		return nil, fmt.Errorf("invalid topology: %w", err)
	}

	router := &Router{
		nodes:    make(map[string]Node, len(topology.Nodes)),
		outgoing: make(map[string][]routeLink, len(topology.Nodes)),
		cursors:  make(map[string]int),
	}

	for _, node := range topology.Nodes {
		router.nodes[node.ID] = node
	}

	// Preserve connection order so routing remains deterministic.
	for _, connection := range topology.Connections {
		router.outgoing[connection.From] = append(
			router.outgoing[connection.From],
			routeLink{
				to:      connection.To,
				latency: connection.Latency,
			},
		)
	}

	return router, nil
}

// NextHop selects the next destination for the given source node.
// Each source maintains an independent round-robin cursor.
// NextHop returns the next node selected by the router.
func (r *Router) NextHop(from string) (Node, error) {
	node, _, err := r.NextHopWithLatency(from)
	return node, err
}

// NextHopWithLatency returns the next node and the latency of the
// selected connection. Selection remains deterministic round-robin.
func (r *Router) NextHopWithLatency(from string) (Node, Tick, error) {
	if _, exists := r.nodes[from]; !exists {
		return Node{}, 0, fmt.Errorf("unknown source node %q", from)
	}

	links := r.outgoing[from]
	if len(links) == 0 {
		return Node{}, 0, fmt.Errorf("node %q has no outgoing connections", from)
	}

	cursor := r.cursors[from] % len(links)
	link := links[cursor]
	r.cursors[from] = (cursor + 1) % len(links)

	node, exists := r.nodes[link.to]
	if !exists {
		return Node{}, 0, fmt.Errorf("unknown destination node %q", link.to)
	}

	return node, link.latency, nil
}
