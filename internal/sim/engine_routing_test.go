package sim

import (
	"math"
	"testing"
)

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

func TestRoutedEngineUsesIndependentWorkerCapacities(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "lb", Type: NodeLoadBalancer},
			{ID: "worker-a", Type: NodeWorker, Capacity: 1},
			{ID: "worker-b", Type: NodeWorker, Capacity: 2},
		},
		Connections: []Connection{
			{From: "client", To: "lb"},
			{From: "lb", To: "worker-a"},
			{From: "lb", To: "worker-b"},
		},
	}

	engine, err := NewEngineWithTopology(4, 3, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	for _, id := range []string{"req-1", "req-2", "req-3", "req-4"} {
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
		"req-1": {started: 0, finished: 3},
		"req-2": {started: 0, finished: 3},
		"req-3": {started: 3, finished: 6},
		"req-4": {started: 0, finished: 3},
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

func TestRoutedCompletionPreservesWaitingRequestWhenNextServiceCannotStart(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{From: "client", To: "worker"},
		},
	}

	engine, err := NewEngineWithTopology(2, 3, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	for _, id := range []string{"req-1", "req-2"} {
		if err := engine.ScheduleRoutedArrival(id, "client", 0); err != nil {
			t.Fatalf("ScheduleRoutedArrival(%q): %v", id, err)
		}
	}

	// Process arrival and routing events until one request is active
	// and the other is waiting at the worker.
	for engine.workerBusy["worker"] == 0 ||
		len(engine.workerWaiting["worker"]) == 0 {
		event, ok := engine.queue.Pop()
		if !ok {
			t.Fatal("event queue emptied before both requests reached the worker")
		}
		engine.now = event.At

		switch event.Type {
		case EventRequestArrives:
			if err := engine.handleArrival(event.RequestID); err != nil {
				t.Fatalf("handleArrival(%q): %v", event.RequestID, err)
			}
		case EventRouteHop:
			if err := engine.handleRouteHop(event.RequestID, event.NodeID); err != nil {
				t.Fatalf("handleRouteHop(%q, %q): %v",
					event.RequestID, event.NodeID, err)
			}
		default:
			t.Fatalf("unexpected event type while preparing test: %s", event.Type)
		}
	}

	// Preserve the completion event, then exhaust the sequence counter.
	event, ok := engine.queue.Pop()
	if !ok {
		t.Fatal("expected a service completion event")
	}
	if event.Type != EventServiceCompletes {
		t.Fatalf("event type = %s, want service completion", event.Type)
	}

	engine.now = event.At
	engine.nextSequence = ^uint64(0)

	err = engine.handleCompletion(event.RequestID, event.NodeID)

	if got := engine.workerBusy["worker"]; got != 0 {
		t.Errorf("workerBusy = %d, want 0", got)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
	if got := engine.workerWaiting["worker"]; len(got) != 1 || got[0] != "req-2" {
		t.Errorf("worker waiting queue = %v, want [req-2]", got)
	}
	if got := engine.Requests()["req-1"].State(); got != RequestSucceeded {
		t.Errorf("req-1 state = %s, want succeeded", got)
	}
	if got := engine.Requests()["req-2"].State(); got != RequestQueued {
		t.Errorf("req-2 state = %s, want queued", got)
	}
}

func TestRoutedEngineAppliesNetworkHopLatency(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "lb", Type: NodeLoadBalancer},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "lb", Latency: 2},
			{From: "lb", To: "worker", Latency: 5},
		},
	}

	engine, err := NewEngineWithTopology(1, 3, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-latency", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-latency"]
	if request == nil {
		t.Fatal("request not found")
	}

	startedAt, started := request.StartedAt()
	if !started {
		t.Fatal("StartedAt() reports no start timestamp")
	}
	if want := Tick(7); startedAt != want {
		t.Errorf("StartedAt() = %d, want %d", startedAt, want)
	}

	finishedAt, finished := request.FinishedAt()
	if !finished {
		t.Fatal("FinishedAt() reports no finish timestamp")
	}
	if want := Tick(10); finishedAt != want {
		t.Errorf("FinishedAt() = %d, want %d", finishedAt, want)
	}
	if got, want := engine.CurrentTick(), Tick(10); got != want {
		t.Errorf("CurrentTick() = %d, want %d", got, want)
	}
	if got := request.State(); got != RequestSucceeded {
		t.Errorf("State() = %s, want %s", got, RequestSucceeded)
	}
}

func TestRoutedEngineRejectsRouteHopTickOverflow(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 1},
		},
	}

	engine, err := NewEngineWithTopology(1, 1, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	arrivalTick := Tick(math.MaxInt64)
	if err := engine.ScheduleRoutedArrival("req-overflow", "client", arrivalTick); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}

	err = engine.Run()
	if err == nil {
		t.Fatal("Run() accepted a route hop that overflows the tick")
	}
}
