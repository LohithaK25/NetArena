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
