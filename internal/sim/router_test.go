package sim

import "testing"

func TestRouterSelectsSingleDestination(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "service", Type: NodeService},
		},
		Connections: []Connection{
			{From: "client", To: "service"},
		},
	}

	router, err := NewRouter(topology)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	got, err := router.NextHop("client")
	if err != nil {
		t.Fatalf("NextHop() error = %v", err)
	}
	if got.ID != "service" {
		t.Fatalf("NextHop() ID = %q, want %q", got.ID, "service")
	}
}

func TestRouterRoundRobinIsDeterministic(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "service-a", Type: NodeService},
			{ID: "service-b", Type: NodeService},
			{ID: "service-c", Type: NodeService},
		},
		Connections: []Connection{
			{From: "client", To: "service-b"},
			{From: "client", To: "service-a"},
			{From: "client", To: "service-c"},
		},
	}

	router, err := NewRouter(topology)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	want := []string{"service-b", "service-a", "service-c", "service-b", "service-a"}
	for i, wantID := range want {
		got, err := router.NextHop("client")
		if err != nil {
			t.Fatalf("NextHop() call %d error = %v", i, err)
		}
		if got.ID != wantID {
			t.Errorf("NextHop() call %d ID = %q, want %q", i, got.ID, wantID)
		}
	}
}

func TestRouterCursorsAreIndependentPerSource(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client-a", Type: NodeClient},
			{ID: "client-b", Type: NodeClient},
			{ID: "service-a", Type: NodeService},
			{ID: "service-b", Type: NodeService},
		},
		Connections: []Connection{
			{From: "client-a", To: "service-a"},
			{From: "client-a", To: "service-b"},
			{From: "client-b", To: "service-b"},
			{From: "client-b", To: "service-a"},
		},
	}

	router, err := NewRouter(topology)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	firstA, err := router.NextHop("client-a")
	if err != nil {
		t.Fatal(err)
	}
	firstB, err := router.NextHop("client-b")
	if err != nil {
		t.Fatal(err)
	}
	secondA, err := router.NextHop("client-a")
	if err != nil {
		t.Fatal(err)
	}

	if firstA.ID != "service-a" || secondA.ID != "service-b" {
		t.Errorf("client-a routes = %q, %q; want service-a, service-b", firstA.ID, secondA.ID)
	}
	if firstB.ID != "service-b" {
		t.Errorf("client-b first route = %q, want service-b", firstB.ID)
	}
}

func TestRouterRejectsUnknownSourceAndDeadEnd(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "service", Type: NodeService},
		},
		Connections: []Connection{
			{From: "client", To: "service"},
		},
	}

	router, err := NewRouter(topology)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	if _, err := router.NextHop("missing"); err == nil {
		t.Error("NextHop() with unknown source should return an error")
	}
	if _, err := router.NextHop("service"); err == nil {
		t.Error("NextHop() with no outgoing connections should return an error")
	}
}

func TestNewRouterRejectsInvalidTopology(t *testing.T) {
	_, err := NewRouter(Topology{
		Nodes: []Node{{ID: "client", Type: NodeClient}},
		Connections: []Connection{
			{From: "client", To: "missing"},
		},
	})
	if err == nil {
		t.Fatal("NewRouter() should reject invalid topology")
	}
}

func TestRouterReturnsLatencyForSelectedConnection(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker-a", Type: NodeWorker},
			{ID: "worker-b", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker-a", Latency: 2},
			{From: "client", To: "worker-b", Latency: 5},
		},
	}

	router, err := NewRouter(topology)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	first, firstLatency, err := router.NextHopWithLatency("client")
	if err != nil {
		t.Fatalf("first NextHopWithLatency() error = %v", err)
	}
	if first.ID != "worker-a" || firstLatency != 2 {
		t.Fatalf(
			"first hop = (%q, %d), want (%q, 2)",
			first.ID, firstLatency, "worker-a",
		)
	}

	second, secondLatency, err := router.NextHopWithLatency("client")
	if err != nil {
		t.Fatalf("second NextHopWithLatency() error = %v", err)
	}
	if second.ID != "worker-b" || secondLatency != 5 {
		t.Fatalf(
			"second hop = (%q, %d), want (%q, 5)",
			second.ID, secondLatency, "worker-b",
		)
	}
}
