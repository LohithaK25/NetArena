package sim

import "testing"

func routingTestTopology() Topology {
	return Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "lb", Type: NodeLoadBalancer},
			{ID: "service", Type: NodeService},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "lb"},
			{From: "lb", To: "service"},
			{From: "service", To: "worker"},
		},
	}
}

func TestRoutedEngineProcessesRequestThroughTopology(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-1", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-1"]
	if request == nil {
		t.Fatal("request not found")
	}
	if request.State() != RequestSucceeded {
		t.Fatalf("request state = %s, want succeeded", request.State())
	}

	started, ok := request.StartedAt()
	if !ok || started != 0 {
		t.Errorf("StartedAt() = (%d, %v), want (0, true)", started, ok)
	}
	finished, ok := request.FinishedAt()
	if !ok || finished != 2 {
		t.Errorf("FinishedAt() = (%d, %v), want (2, true)", finished, ok)
	}
}

func TestRoutedEngineFailsRequestAtDeadEnd(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}
	if err := engine.ScheduleRoutedArrival("req-dead-end", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-dead-end"]
	if request.State() != RequestFailed {
		t.Fatalf("request state = %s, want failed", request.State())
	}
	if request.FailureReason() == "" {
		t.Error("failed request should have a failure reason")
	}
	if _, ok := request.FinishedAt(); !ok {
		t.Error("failed request should have a finish tick")
	}
}

func TestRoutedEngineRejectsUnknownSource(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-1", "missing", 0); err == nil {
		t.Fatal("ScheduleRoutedArrival() accepted an unknown source node")
	}
	if len(engine.Requests()) != 0 {
		t.Fatal("rejected request should not be registered")
	}
}

func TestRoutedEngineRejectsInvalidTopology(t *testing.T) {
	_, err := NewEngineWithTopology(1, 2, Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
		},
		Connections: []Connection{
			{From: "client", To: "missing"},
		},
	})
	if err == nil {
		t.Fatal("NewEngineWithTopology() accepted an invalid topology")
	}
}

func TestRoutedEngineDetectsRoutingCycles(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "lb", Type: NodeLoadBalancer},
		},
		Connections: []Connection{
			{From: "client", To: "lb"},
			{From: "lb", To: "client"},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}
	if err := engine.ScheduleRoutedArrival("req-cycle", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-cycle"]
	if request.State() != RequestFailed {
		t.Fatalf("request state = %s, want failed", request.State())
	}
	if request.FailureReason() == "" {
		t.Error("cycle failure should include a reason")
	}
}

func TestRoutedEngineRespectsWorkerCapacityAndFIFO(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker"},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	for _, id := range []string{"req-1", "req-2", "req-3"} {
		if err := engine.ScheduleRoutedArrival(id, "client", 0); err != nil {
			t.Fatalf("ScheduleRoutedArrival(%q): %v", id, err)
		}
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]struct {
		started  Tick
		finished Tick
	}{
		"req-1": {started: 0, finished: 2},
		"req-2": {started: 2, finished: 4},
		"req-3": {started: 4, finished: 6},
	}

	for id, expected := range want {
		request := engine.Requests()[id]
		if request == nil {
			t.Fatalf("request %q not found", id)
		}
		if request.State() != RequestSucceeded {
			t.Errorf("%s state = %s, want succeeded", id, request.State())
		}

		started, hasStarted := request.StartedAt()
		if !hasStarted || started != expected.started {
			t.Errorf("%s start = (%d, %v), want (%d, true)",
				id, started, hasStarted, expected.started)
		}

		finished, hasFinished := request.FinishedAt()
		if !hasFinished || finished != expected.finished {
			t.Errorf("%s finish = (%d, %v), want (%d, true)",
				id, finished, hasFinished, expected.finished)
		}
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
}

func TestRoutedEngineRejectsArrivalWithoutSource(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err == nil {
		t.Fatal("ScheduleArrival() should require a source on a topology-aware engine")
	}
	if len(engine.Requests()) != 0 {
		t.Fatal("rejected request should not be registered")
	}
	if engine.queue.Len() != 0 {
		t.Fatal("rejected request should not schedule an event")
	}
}
