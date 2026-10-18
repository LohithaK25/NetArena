package sim

import (
	"fmt"
	"math"
	"sort"
)

type routeLink struct {
	to                    string
	latency               Tick
	dropPackets           bool
	packetLossProbability float64
}

// Router selects destination nodes using deterministic round-robin routing.
type Router struct {
	nodes     map[string]Node
	outgoing  map[string][]routeLink
	cursors   map[string]int
	algorithm RoutingAlgorithm
}

// NewRouter validates the topology and builds a router using round-robin routing.
func NewRouter(topology Topology) (*Router, error) {
	return NewRouterWithAlgorithm(topology, RoutingRoundRobin)
}

// NewRouterWithAlgorithm validates the topology and builds a router using
// the specified routing algorithm.
func NewRouterWithAlgorithm(
	topology Topology,
	algorithm RoutingAlgorithm,
) (*Router, error) {
	if err := algorithm.Validate(); err != nil {
		return nil, fmt.Errorf("invalid routing algorithm: %w", err)
	}
	if err := topology.Validate(); err != nil {
		return nil, fmt.Errorf("invalid topology: %w", err)
	}

	router := &Router{
		nodes:     make(map[string]Node, len(topology.Nodes)),
		outgoing:  make(map[string][]routeLink, len(topology.Nodes)),
		cursors:   make(map[string]int),
		algorithm: algorithm,
	}

	for _, node := range topology.Nodes {
		router.nodes[node.ID] = node
	}

	// Preserve connection order so routing remains deterministic.
	for _, connection := range topology.Connections {
		router.outgoing[connection.From] = append(
			router.outgoing[connection.From],
			routeLink{
				to:                    connection.To,
				latency:               connection.Latency,
				dropPackets:           connection.DropPackets,
				packetLossProbability: connection.PacketLossProbability,
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
	return r.NextHopWithLatencyFiltered(from, func(Node) bool {
		return true
	})
}

// NextHopWithLatencyFiltered selects the next eligible destination using
// deterministic round-robin routing. The availability callback determines
// whether a destination can receive a request.

func (r *Router) NextHopWithLatencyFiltered(
	from string,
	available func(Node) bool,
) (Node, Tick, error) {
	node, latency, _, _, err := r.nextHopWithLinkDetails(from, available)
	return node, latency, err
}

func (r *Router) NextHopWithLinkStatus(
	from string,
	available func(Node) bool,
) (Node, Tick, bool, error) {
	node, latency, dropped, _, err := r.nextHopWithLinkDetails(from, available)
	return node, latency, dropped, err
}

func (r *Router) NextHopWithLinkStatusAndLossProbability(
	from string,
	available func(Node) bool,
) (Node, Tick, bool, float64, error) {
	return r.nextHopWithLinkDetails(from, available)
}

func (r *Router) nextHopWithLinkDetails(
	from string,
	available func(Node) bool,
) (Node, Tick, bool, float64, error) {
	if _, exists := r.nodes[from]; !exists {
		return Node{}, 0, false, 0, fmt.Errorf("unknown source node %q", from)
	}
	if available == nil {
		return Node{}, 0, false, 0, fmt.Errorf("availability callback must not be nil")
	}

	links := r.outgoing[from]
	if len(links) == 0 {
		return Node{}, 0, false, 0, fmt.Errorf("node %q has no outgoing connections", from)
	}

	if r.algorithm == RoutingShortestPath {
		return r.shortestPathNextHop(from, available)
	}

	start := r.cursors[from] % len(links)

	for offset := 0; offset < len(links); offset++ {
		index := (start + offset) % len(links)
		link := links[index]

		node, exists := r.nodes[link.to]
		if !exists {
			return Node{}, 0, false, 0, fmt.Errorf("unknown destination node %q", link.to)
		}
		if !available(node) {
			continue
		}

		r.cursors[from] = (index + 1) % len(links)
		return node, link.latency, link.dropPackets, link.packetLossProbability, nil
	}

	return Node{}, 0, false, 0, fmt.Errorf(
		"node %q has no available outgoing destinations", from,
	)
}

// shortestPathNextHop selects the first connection on the lowest-latency
// path to an available worker. Node IDs break equal-distance ties
// deterministically; connection order breaks equal-cost predecessor ties.
func (r *Router) shortestPathNextHop(
	from string,
	available func(Node) bool,
) (Node, Tick, bool, float64, error) {
	distances := map[string]Tick{from: 0}
	previousNode := make(map[string]string, len(r.nodes))
	previousLink := make(map[string]routeLink, len(r.nodes))
	visited := make(map[string]bool, len(r.nodes))

	nodeIDs := make([]string, 0, len(r.nodes))
	for id := range r.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	for range nodeIDs {
		current := ""
		var currentDistance Tick

		for _, id := range nodeIDs {
			distance, found := distances[id]
			if !found || visited[id] {
				continue
			}
			if current == "" || distance < currentDistance {
				current = id
				currentDistance = distance
			}
		}

		if current == "" {
			break
		}

		currentNode := r.nodes[current]
		if current != from && currentNode.Type == NodeWorker {
			firstHop := current

			for previousNode[firstHop] != from {
				parent, found := previousNode[firstHop]
				if !found {
					return Node{}, 0, false, 0, fmt.Errorf(
						"cannot reconstruct route from %q to %q",
						from, current,
					)
				}
				firstHop = parent
			}

			link, found := previousLink[firstHop]
			if !found {
				return Node{}, 0, false, 0, fmt.Errorf(
					"route from %q has no first connection",
					from,
				)
			}

			return r.nodes[firstHop], link.latency,
				link.dropPackets, link.packetLossProbability, nil
		}

		visited[current] = true

		for _, link := range r.outgoing[current] {
			destination, found := r.nodes[link.to]
			if !found {
				return Node{}, 0, false, 0, fmt.Errorf(
					"unknown destination node %q", link.to,
				)
			}
			if !available(destination) || visited[link.to] {
				continue
			}

			if link.latency > 0 &&
				currentDistance > Tick(math.MaxInt64)-link.latency {
				continue
			}

			candidate := currentDistance + link.latency
			existing, found := distances[link.to]
			if !found || candidate < existing {
				distances[link.to] = candidate
				previousNode[link.to] = current
				previousLink[link.to] = link
			}
		}
	}

	return Node{}, 0, false, 0, fmt.Errorf(
		"node %q has no available route to a worker", from,
	)
}
