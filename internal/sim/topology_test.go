package sim

import "testing"

func TestTopologyValidatesDirectedServicePath(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client-1", Type: NodeClient, Name: "Client"},
			{ID: "lb-1", Type: NodeLoadBalancer, Name: "Load Balancer"},
			{ID: "service-1", Type: NodeService, Name: "API Service"},
			{ID: "worker-1", Type: NodeWorker, Name: "Worker 1"},
		},
		Connections: []Connection{
			{From: "client-1", To: "lb-1"},
			{From: "lb-1", To: "service-1"},
			{From: "service-1", To: "worker-1"},
		},
	}

	if err := topology.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestTopologyRejectsInvalidStructures(t *testing.T) {
	tests := []struct {
		name     string
		topology Topology
	}{
		{
			name: "empty node ID",
			topology: Topology{
				Nodes: []Node{{ID: "", Type: NodeClient}},
			},
		},
		{
			name: "duplicate node ID",
			topology: Topology{
				Nodes: []Node{
					{ID: "node-1", Type: NodeClient},
					{ID: "node-1", Type: NodeWorker},
				},
			},
		},
		{
			name: "invalid node type",
			topology: Topology{
				Nodes: []Node{{ID: "node-1", Type: 99}},
			},
		},
		{
			name: "unknown source",
			topology: Topology{
				Nodes: []Node{{ID: "node-1", Type: NodeClient}},
				Connections: []Connection{
					{From: "missing", To: "node-1"},
				},
			},
		},
		{
			name: "unknown destination",
			topology: Topology{
				Nodes: []Node{{ID: "node-1", Type: NodeClient}},
				Connections: []Connection{
					{From: "node-1", To: "missing"},
				},
			},
		},
		{
			name: "self connection",
			topology: Topology{
				Nodes: []Node{{ID: "node-1", Type: NodeClient}},
				Connections: []Connection{
					{From: "node-1", To: "node-1"},
				},
			},
		},
		{
			name: "duplicate connection",
			topology: Topology{
				Nodes: []Node{
					{ID: "client-1", Type: NodeClient},
					{ID: "service-1", Type: NodeService},
				},
				Connections: []Connection{
					{From: "client-1", To: "service-1"},
					{From: "client-1", To: "service-1"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.topology.Validate(); err == nil {
				t.Fatal("Validate() succeeded for invalid topology")
			}
		})
	}
}

func TestNodeTypeString(t *testing.T) {
	tests := []struct {
		nodeType NodeType
		want     string
	}{
		{NodeClient, "client"},
		{NodeLoadBalancer, "load_balancer"},
		{NodeService, "service"},
		{NodeWorker, "worker"},
		{NodeType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.nodeType.String(); got != tt.want {
			t.Errorf("NodeType(%d).String() = %q, want %q", tt.nodeType, got, tt.want)
		}
	}
}

func TestTopologyRejectsNegativeWorkerCapacity(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "worker", Type: NodeWorker, Capacity: -1},
		},
	}

	if err := topology.Validate(); err == nil {
		t.Fatal("Validate() accepted a negative worker capacity")
	}
}
