package sim

import "fmt"

// RoutingAlgorithm determines how the router selects a path.
type RoutingAlgorithm string

const (
	// RoutingRoundRobin selects eligible outgoing connections in rotation.
	RoutingRoundRobin RoutingAlgorithm = "round_robin"

	// RoutingShortestPath selects a path with minimum total connection latency.
	RoutingShortestPath RoutingAlgorithm = "shortest_path"
)

// Validate checks whether the routing algorithm is supported.
func (a RoutingAlgorithm) Validate() error {
	switch a {
	case RoutingRoundRobin, RoutingShortestPath:
		return nil
	default:
		return fmt.Errorf("unsupported routing algorithm %q", a)
	}
}
