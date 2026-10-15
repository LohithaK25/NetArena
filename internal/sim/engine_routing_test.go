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

	err = engine.handleCompletion(event.RequestID, event.NodeID, event.AttemptID)

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

func TestRoutedQueuedRequestTimesOutAndPreservesWorkerQueue(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{From: "client", To: "worker"},
		},
	}

	engine, err := NewEngineWithTopology(1, 10, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	for _, id := range []string{"req-active", "req-timeout", "req-waiting"} {
		if err := engine.ScheduleRoutedArrival(id, "client", 0); err != nil {
			t.Fatalf("ScheduleRoutedArrival(%q): %v", id, err)
		}
	}

	if err := engine.ScheduleTimeout("req-timeout", 3); err != nil {
		t.Fatalf("ScheduleTimeout() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := engine.Requests()
	if got := requests["req-timeout"].State(); got != RequestTimedOut {
		t.Errorf("req-timeout state = %s, want timed_out", got)
	}
	if got := requests["req-active"].State(); got != RequestSucceeded {
		t.Errorf("req-active state = %s, want succeeded", got)
	}
	if got := requests["req-waiting"].State(); got != RequestSucceeded {
		t.Errorf("req-waiting state = %s, want succeeded", got)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
	if got := engine.workerBusy["worker"]; got != 0 {
		t.Errorf("worker busy count = %d, want 0", got)
	}
	if got := len(engine.workerWaiting["worker"]); got != 0 {
		t.Errorf("worker waiting queue = %v, want empty", engine.workerWaiting["worker"])
	}
}

func TestRoutedQueuedRequestCanBeCancelledAndPreservesWorkerQueue(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{From: "client", To: "worker"},
		},
	}

	engine, err := NewEngineWithTopology(1, 10, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	for _, id := range []string{"req-active", "req-cancel", "req-waiting"} {
		if err := engine.ScheduleRoutedArrival(id, "client", 0); err != nil {
			t.Fatalf("ScheduleRoutedArrival(%q) error = %v", id, err)
		}
	}

	if err := engine.ScheduleCancel("req-cancel", 3); err != nil {
		t.Fatalf("ScheduleCancel() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := engine.Requests()
	if got := requests["req-cancel"].State(); got != RequestCancelled {
		t.Errorf("cancelled request state = %s, want cancelled", got)
	}
	if got := requests["req-active"].State(); got != RequestSucceeded {
		t.Errorf("active request state = %s, want succeeded", got)
	}
	if got := requests["req-waiting"].State(); got != RequestSucceeded {
		t.Errorf("waiting request state = %s, want succeeded", got)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
	if got := engine.workerBusy["worker"]; got != 0 {
		t.Errorf("worker busy count = %d, want 0", got)
	}
	if got := len(engine.workerWaiting["worker"]); got != 0 {
		t.Errorf("worker waiting queue = %v, want empty", engine.workerWaiting["worker"])
	}
}

func TestMetricsReportEffectiveTopologyWorkerCapacity(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker-a", Type: NodeWorker, Capacity: 1},
			{ID: "worker-b", Type: NodeWorker, Capacity: 2},
			{ID: "worker-c", Type: NodeWorker},
		},
	}

	engine, err := NewEngineWithTopology(3, 2, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	got := engine.Metrics()

	if got.ServiceCapacity != 6 {
		t.Errorf("ServiceCapacity = %d, want 6", got.ServiceCapacity)
	}
	if got.WorkerUtilization != 0 {
		t.Errorf("WorkerUtilization = %v, want 0", got.WorkerUtilization)
	}
	if len(got.Workers) != 3 {
		t.Fatalf("Workers contains %d entries, want 3", len(got.Workers))
	}

	wantCapacities := map[string]int{
		"worker-a": 1,
		"worker-b": 2,
		"worker-c": 3,
	}
	for workerID, wantCapacity := range wantCapacities {
		worker, ok := got.Workers[workerID]
		if !ok {
			t.Errorf("Workers missing %q", workerID)
			continue
		}
		if worker.Capacity != wantCapacity {
			t.Errorf("%s capacity = %d, want %d",
				workerID, worker.Capacity, wantCapacity)
		}
		if worker.BusyWorkers != 0 || worker.WaitingRequests != 0 {
			t.Errorf("%s metrics = %+v, want no busy or waiting work",
				workerID, worker)
		}
	}
}

func TestScheduleWorkerEvents(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "schedule worker failure",
			call: func() error {
				return engine.ScheduleWorkerFailure("worker", 5)
			},
		},
		{
			name: "schedule worker recovery",
			call: func() error {
				return engine.ScheduleWorkerRecovery("worker", 10)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Fatalf("scheduling worker event failed: %v", err)
			}
		})
	}
}

func TestScheduleWorkerEventValidation(t *testing.T) {
	plainEngine, err := NewEngine(1, 2)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	topologyEngine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "failure without topology",
			call: func() error {
				return plainEngine.ScheduleWorkerFailure("worker", 1)
			},
		},
		{
			name: "recovery without topology",
			call: func() error {
				return plainEngine.ScheduleWorkerRecovery("worker", 1)
			},
		},
		{
			name: "empty worker ID",
			call: func() error {
				return topologyEngine.ScheduleWorkerFailure("", 1)
			},
		},
		{
			name: "unknown worker",
			call: func() error {
				return topologyEngine.ScheduleWorkerFailure("missing-worker", 1)
			},
		},
		{
			name: "non-worker node",
			call: func() error {
				return topologyEngine.ScheduleWorkerFailure("client", 1)
			},
		},
		{
			name: "failure scheduled in the past",
			call: func() error {
				if err := topologyEngine.ScheduleWorkerFailure("worker", 5); err != nil {
					return err
				}
				if err := topologyEngine.Run(); err != nil {
					return err
				}
				return topologyEngine.ScheduleWorkerFailure("worker", 4)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestWorkerFailureFailsInFlightRequestAndIgnoresStaleCompletion(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 10, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-fail", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 5); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-fail"]
	if request.State() != RequestFailed {
		t.Fatalf("request state = %s, want failed", request.State())
	}
	if request.FailureReason() != `worker "worker" failed` {
		t.Errorf("failure reason = %q", request.FailureReason())
	}
	if finishedAt, ok := request.FinishedAt(); !ok || finishedAt != 5 {
		t.Errorf("FinishedAt() = (%d, %v), want (5, true)", finishedAt, ok)
	}

	metrics := engine.Metrics()
	if metrics.InFlightRequests != 0 || metrics.FailedRequests != 1 {
		t.Errorf("unexpected lifecycle metrics: %+v", metrics)
	}
	if metrics.BusyWorkers != 0 || metrics.Workers["worker"].BusyWorkers != 0 {
		t.Errorf("worker capacity was not released: %+v", metrics)
	}
}

func TestWorkerRecoveryResumesQueuedRequestsInFIFOOrder(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 10, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	// The first request occupies the worker until tick 10.
	// The second request queues behind it.
	if err := engine.ScheduleRoutedArrival("req-1", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleRoutedArrival("req-2", "client", 1); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleRoutedArrival("req-3", "client", 2); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 5); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 8); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := engine.Requests()
	if requests["req-1"].State() != RequestFailed {
		t.Errorf("req-1 state = %s, want failed", requests["req-1"].State())
	}

	for _, id := range []string{"req-2", "req-3"} {
		if requests[id].State() != RequestSucceeded {
			t.Errorf("%s state = %s, want succeeded", id, requests[id].State())
		}
	}

	started2, ok2 := requests["req-2"].StartedAt()
	started3, ok3 := requests["req-3"].StartedAt()
	if !ok2 || !ok3 {
		t.Fatal("expected both queued requests to have start ticks")
	}
	if started2 != 8 {
		t.Errorf("req-2 started at %d, want 8", started2)
	}
	if started3 != 18 {
		t.Errorf("req-3 started at %d, want 18", started3)
	}

	metrics := engine.Metrics()
	if metrics.BusyWorkers != 0 || metrics.Workers["worker"].WaitingRequests != 0 {
		t.Errorf("worker did not drain after recovery: %+v", metrics)
	}
}

func TestWorkerMetricsReflectFailureAndRecovery(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleWorkerFailure("worker", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 6); err != nil {
		t.Fatal(err)
	}

	// Process the failure event first, without running later events.
	event, ok := engine.queue.Pop()
	if !ok {
		t.Fatal("expected a scheduled failure event")
	}
	engine.now = event.At
	if event.Type != EventWorkerFails {
		t.Fatalf("first event type = %s, want worker_fails", event.Type)
	}
	if err := engine.handleWorkerFailure(event.NodeID); err != nil {
		t.Fatalf("handleWorkerFailure() error = %v", err)
	}

	metrics := engine.Metrics()
	if metrics.Workers["worker"].Online {
		t.Fatal("worker should be offline after failure")
	}

	event, ok = engine.queue.Pop()
	if !ok {
		t.Fatal("expected a scheduled recovery event")
	}
	engine.now = event.At
	if event.Type != EventWorkerRecovers {
		t.Fatalf("next event type = %s, want worker_recovers", event.Type)
	}
	if err := engine.handleWorkerRecovery(event.NodeID); err != nil {
		t.Fatalf("handleWorkerRecovery() error = %v", err)
	}

	metrics = engine.Metrics()
	if !metrics.Workers["worker"].Online {
		t.Fatal("worker should be online after recovery")
	}
}

func TestRequestArrivingAtFailedWorkerWaitsForRecovery(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 5},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("req-1", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 8); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-1"]
	if request.State() != RequestSucceeded {
		t.Fatalf("request state = %s, want succeeded", request.State())
	}

	startedAt, ok := request.StartedAt()
	if !ok || startedAt != 8 {
		t.Errorf("StartedAt() = (%d, %v), want (8, true)", startedAt, ok)
	}
}

func TestWorkerFailureCanRetryRequest(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 1},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatal(err)
	}

	// This test will use the retry policy API introduced in this phase.
	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 1,
		Backoff:    3,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("req-retry", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 6); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-retry"]
	if request.State() != RequestSucceeded {
		t.Fatalf("request state = %s, want succeeded", request.State())
	}

	startedAt, ok := request.StartedAt()
	if !ok || startedAt != 7 {
		t.Errorf("StartedAt() = (%d, %v), want (7, true)", startedAt, ok)
	}
}

func TestWorkerFailureExhaustsRetryBudget(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 1},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 0,
		Backoff:    0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("req-exhausted", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 3); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-exhausted"]
	if request.State() != RequestFailed {
		t.Fatalf("request state = %s, want failed", request.State())
	}
	if got := request.FailureReason(); got != `worker "worker" failed` {
		t.Errorf("failure reason = %q, want worker failure reason", got)
	}
	if got := request.Attempts(); got != 1 {
		t.Errorf("Attempts() = %d, want 1", got)
	}
}

func TestWorkerFailureRetryWithZeroBackoff(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 1},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 1,
		Backoff:    0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("req-zero-backoff", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 4); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-zero-backoff"]
	if request.State() != RequestSucceeded {
		t.Fatalf("request state = %s, want succeeded", request.State())
	}
	if got := request.Attempts(); got != 2 {
		t.Errorf("Attempts() = %d, want 2", got)
	}
}

func TestWorkerFailureRetryWaitsForFailedWorker(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker-a", Type: NodeWorker},
			{ID: "worker-b", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker-a", Latency: 1},
			{From: "client", To: "worker-b", Latency: 2},
		},
	}

	engine, err := NewEngineWithTopology(2, 2, topology)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 1,
		Backoff:    0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("req-multi-worker", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker-a", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker-a", 6); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-multi-worker"]
	if request.State() != RequestSucceeded {
		t.Fatalf("request state = %s, want succeeded", request.State())
	}

	if got := request.Attempts(); got != 2 {
		t.Errorf("Attempts() = %d, want 2", got)
	}

	startedAt, ok := request.StartedAt()
	if !ok || startedAt != 6 {
		t.Errorf("StartedAt() = (%d, %v), want (6, true)", startedAt, ok)
	}
}

func TestCancelledRetryDoesNotBlockWorkerRecovery(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 1},
		},
	}

	engine, err := NewEngineWithTopology(1, 2, topology)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 1,
		Backoff:    0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("req-cancelled-retry", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleCancel("req-cancelled-retry", 4); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 6); err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-cancelled-retry"]
	if request.State() != RequestCancelled {
		t.Fatalf("request state = %s, want cancelled", request.State())
	}
}
