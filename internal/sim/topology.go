package sim

import "fmt"

// NodeType identifies the role of a component in the simulated topology.
type NodeType uint8

const (
	NodeClient NodeType = iota + 1
	NodeLoadBalancer
	NodeService
	NodeWorker
)

// String returns the stable textual name of a node type.
func (t NodeType) String() string {
	switch t {
	case NodeClient:
		return "client"
	case NodeLoadBalancer:
		return "load_balancer"
	case NodeService:
		return "service"
	case NodeWorker:
		return "worker"
	default:
		return "unknown"
	}
}

// Node describes one component in a simulated infrastructure topology.
type Node struct {
	ID   string
	Type NodeType
	Name string
}

// Connection is a directed link between two topology nodes.
type Connection struct {
	From string
	To   string
}

// Topology describes the nodes and directed connections in a simulation.
type Topology struct {
	Nodes       []Node
	Connections []Connection
}

// Validate checks topology identifiers, node types, and connection endpoints.
func (t Topology) Validate() error {
	nodes := make(map[string]Node, len(t.Nodes))

	for _, node := range t.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID must not be empty")
		}
		if _, exists := nodes[node.ID]; exists {
			return fmt.Errorf("duplicate node ID %q", node.ID)
		}
		if node.Type < NodeClient || node.Type > NodeWorker {
			return fmt.Errorf("node %q has invalid type %d", node.ID, node.Type)
		}
		nodes[node.ID] = node
	}

	seenConnections := make(map[Connection]bool, len(t.Connections))
	for _, connection := range t.Connections {
		if connection.From == "" || connection.To == "" {
			return fmt.Errorf("connection endpoints must not be empty")
		}
		if connection.From == connection.To {
			return fmt.Errorf("connection %q cannot point to itself", connection.From)
		}
		if _, exists := nodes[connection.From]; !exists {
			return fmt.Errorf("connection references unknown source node %q", connection.From)
		}
		if _, exists := nodes[connection.To]; !exists {
			return fmt.Errorf("connection references unknown destination node %q", connection.To)
		}
		if seenConnections[connection] {
			return fmt.Errorf(
				"duplicate connection from %q to %q",
				connection.From,
				connection.To,
			)
		}
		seenConnections[connection] = true
	}

	return nil
}
