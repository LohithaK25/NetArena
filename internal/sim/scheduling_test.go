package sim

import "testing"

func TestSchedulingPolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  SchedulingPolicy
		wantErr bool
	}{
		{
			name:   "FIFO",
			policy: SchedulingFIFO,
		},
		{
			name:   "priority",
			policy: SchedulingPriority,
		},
		{
			name:    "empty policy",
			policy:  "",
			wantErr: true,
		},
		{
			name:    "unsupported policy",
			policy:  "round-robin",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngineSchedulingPolicyDefaultsToFIFO(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if got := engine.schedulingPolicy; got != SchedulingFIFO {
		t.Fatalf("default scheduling policy = %q, want %q", got, SchedulingFIFO)
	}
}

func TestEngineSetSchedulingPolicy(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.SetSchedulingPolicy(SchedulingPriority); err != nil {
		t.Fatalf("SetSchedulingPolicy(priority) error = %v", err)
	}
	if got := engine.schedulingPolicy; got != SchedulingPriority {
		t.Fatalf("scheduling policy = %q, want %q", got, SchedulingPriority)
	}

	if err := engine.SetSchedulingPolicy(SchedulingPolicy("invalid")); err == nil {
		t.Fatal("SetSchedulingPolicy(invalid) error = nil, want an error")
	}
	if got := engine.schedulingPolicy; got != SchedulingPriority {
		t.Fatalf("policy after invalid update = %q, want %q", got, SchedulingPriority)
	}
}

func TestScheduleArrivalWithPriority(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("default", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.ScheduleArrivalWithPriority("urgent", 0, 7); err != nil {
		t.Fatalf("ScheduleArrivalWithPriority() error = %v", err)
	}

	requests := engine.Requests()
	if got := requests["default"].Priority(); got != 0 {
		t.Errorf("default request priority = %d, want 0", got)
	}
	if got := requests["urgent"].Priority(); got != 7 {
		t.Errorf("urgent request priority = %d, want 7", got)
	}
}

func TestScheduleRoutedArrivalWithPriority(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 1},
		},
	}

	engine, err := NewEngineWithTopology(1, 1, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("default", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}
	if err := engine.ScheduleRoutedArrivalWithPriority("urgent", "client", 0, 9); err != nil {
		t.Fatalf("ScheduleRoutedArrivalWithPriority() error = %v", err)
	}

	requests := engine.Requests()
	if got := requests["default"].Priority(); got != 0 {
		t.Errorf("default request priority = %d, want 0", got)
	}
	if got := requests["urgent"].Priority(); got != 9 {
		t.Errorf("urgent request priority = %d, want 9", got)
	}
}

func TestEnqueueWaitingRequestOrdersByPriorityStably(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.SetSchedulingPolicy(SchedulingPriority); err != nil {
		t.Fatalf("SetSchedulingPolicy() error = %v", err)
	}

	inputs := []struct {
		id       string
		priority int
	}{
		{id: "low", priority: 1},
		{id: "high-1", priority: 3},
		{id: "middle", priority: 2},
		{id: "high-2", priority: 3},
	}

	var queue []string
	for _, input := range inputs {
		request, err := NewRequestWithPriority(input.id, 0, input.priority)
		if err != nil {
			t.Fatalf("NewRequestWithPriority() error = %v", err)
		}
		engine.requests[input.id] = request
		queue = engine.enqueueWaitingRequest(queue, input.id)
	}

	want := []string{"high-1", "high-2", "middle", "low"}
	if len(queue) != len(want) {
		t.Fatalf("queue = %v, want %v", queue, want)
	}
	for i := range want {
		if queue[i] != want[i] {
			t.Fatalf("queue = %v, want %v", queue, want)
		}
	}
}

func TestEnqueueWaitingRequestPreservesFIFOByDefault(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	queue := engine.enqueueWaitingRequest(nil, "first")
	queue = engine.enqueueWaitingRequest(queue, "second")
	queue = engine.enqueueWaitingRequest(queue, "third")

	want := []string{"first", "second", "third"}
	for i := range want {
		if queue[i] != want[i] {
			t.Fatalf("queue = %v, want %v", queue, want)
		}
	}
}

func TestPrioritySchedulingOrdersQueuedRequestsStably(t *testing.T) {
	engine, err := NewEngine(1, 5)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.SetSchedulingPolicy(SchedulingPriority); err != nil {
		t.Fatalf("SetSchedulingPolicy() error = %v", err)
	}

	// Occupy the only worker so the remaining requests must wait.
	if err := engine.ScheduleArrival("blocker", 0); err != nil {
		t.Fatalf("ScheduleArrival(blocker) error = %v", err)
	}

	// All three arrive while the blocker is being processed.
	// The two priority-3 requests must retain their relative FIFO order.
	arrivals := []struct {
		id       string
		priority int
	}{
		{id: "low", priority: 1},
		{id: "high-1", priority: 3},
		{id: "high-2", priority: 3},
	}

	for _, arrival := range arrivals {
		if err := engine.ScheduleArrivalWithPriority(
			arrival.id, 1, arrival.priority,
		); err != nil {
			t.Fatalf("ScheduleArrivalWithPriority(%s) error = %v", arrival.id, err)
		}
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantStart := map[string]Tick{
		"blocker": 0,
		"high-1":  5,
		"high-2":  10,
		"low":     15,
	}

	requests := engine.Requests()
	for id, want := range wantStart {
		request, exists := requests[id]
		if !exists {
			t.Fatalf("request %q not found", id)
		}

		got, ok := request.StartedAt()
		if !ok {
			t.Fatalf("request %q never started", id)
		}
		if got != want {
			t.Errorf("request %q started at tick %d, want %d", id, got, want)
		}
	}
}

func TestEngineSetSchedulingPolicyRejectsChangeAfterProcessingStarts(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.SetSchedulingPolicy(SchedulingPriority); err != nil {
		t.Fatalf("SetSchedulingPolicy(priority) error = %v", err)
	}
	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if err := engine.SetSchedulingPolicy(SchedulingFIFO); err == nil {
		t.Fatal("SetSchedulingPolicy(fifo) after processing error = nil, want an error")
	}

	if got := engine.schedulingPolicy; got != SchedulingPriority {
		t.Fatalf("scheduling policy = %q, want %q", got, SchedulingPriority)
	}
}
