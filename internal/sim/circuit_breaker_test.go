package sim

import "testing"

func TestCircuitBreakerPolicyValidation(t *testing.T) {
	tests := []struct {
		name    string
		policy  CircuitBreakerPolicy
		wantErr bool
	}{
		{
			name:   "valid policy",
			policy: CircuitBreakerPolicy{FailureThreshold: 2, OpenDuration: 5},
		},
		{
			name:    "zero failure threshold",
			policy:  CircuitBreakerPolicy{FailureThreshold: 0, OpenDuration: 5},
			wantErr: true,
		},
		{
			name:    "negative failure threshold",
			policy:  CircuitBreakerPolicy{FailureThreshold: -1, OpenDuration: 5},
			wantErr: true,
		},
		{
			name:    "zero open duration",
			policy:  CircuitBreakerPolicy{FailureThreshold: 2, OpenDuration: 0},
			wantErr: true,
		},
		{
			name:    "negative open duration",
			policy:  CircuitBreakerPolicy{FailureThreshold: 2, OpenDuration: -1},
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

func TestCircuitBreakerMetricsDefaultToClosed(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}

	worker, ok := engine.Metrics().Workers["worker"]
	if !ok {
		t.Fatal("worker metrics missing for worker")
	}

	if worker.CircuitState != CircuitClosed {
		t.Errorf("CircuitState = %v, want %v", worker.CircuitState, CircuitClosed)
	}
	if worker.ConsecutiveFailures != 0 {
		t.Errorf("ConsecutiveFailures = %d, want 0", worker.ConsecutiveFailures)
	}
	if worker.ProbeInFlight {
		t.Error("ProbeInFlight = true, want false")
	}
}

func TestCircuitBreakerOpensAfterFailureThreshold(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.SetCircuitBreakerPolicy(CircuitBreakerPolicy{
		FailureThreshold: 1,
		OpenDuration:     5,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleWorkerFailure("worker", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	worker := engine.Metrics().Workers["worker"]
	if worker.CircuitState != CircuitOpen {
		t.Errorf("CircuitState = %v, want %v", worker.CircuitState, CircuitOpen)
	}
	if worker.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1", worker.ConsecutiveFailures)
	}
}

func TestCircuitBreakerPolicyRequiresTopology(t *testing.T) {
	engine, err := NewEngine(1, 2)
	if err != nil {
		t.Fatal(err)
	}

	err = engine.SetCircuitBreakerPolicy(CircuitBreakerPolicy{
		FailureThreshold: 2,
		OpenDuration:     5,
	})
	if err == nil {
		t.Fatal("SetCircuitBreakerPolicy() should require a topology")
	}
}

func TestCircuitBreakerHalfOpenAllowsOnlyOneProbe(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.SetCircuitBreakerPolicy(CircuitBreakerPolicy{
		FailureThreshold: 2,
		OpenDuration:     5,
	}); err != nil {
		t.Fatal(err)
	}

	breaker := engine.circuitBreakers["worker"]
	breaker.state = CircuitOpen
	breaker.openUntil = engine.now

	if !engine.acquireCircuitProbe("worker", "request-1") {
		t.Fatal("first request should acquire the half-open probe")
	}
	if engine.acquireCircuitProbe("worker", "request-2") {
		t.Fatal("second request should not acquire an occupied probe")
	}
	if !engine.acquireCircuitProbe("worker", "request-1") {
		t.Fatal("probe owner should retain its reservation")
	}

	worker := engine.Metrics().Workers["worker"]
	if !worker.ProbeInFlight {
		t.Error("ProbeInFlight = false, want true")
	}
	if worker.CircuitState != CircuitHalfOpen {
		t.Errorf("CircuitState = %v, want %v", worker.CircuitState, CircuitHalfOpen)
	}
}

func TestCircuitBreakerProbeTerminationSchedulesCooldown(t *testing.T) {
	tests := []struct {
		name          string
		terminalState RequestState
	}{
		{name: "cancelled probe", terminalState: RequestCancelled},
		{name: "timed out probe", terminalState: RequestTimedOut},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
			if err != nil {
				t.Fatal(err)
			}
			if err := engine.SetCircuitBreakerPolicy(CircuitBreakerPolicy{
				FailureThreshold: 2,
				OpenDuration:     5,
			}); err != nil {
				t.Fatal(err)
			}

			request, err := NewRequest("probe", 0)
			if err != nil {
				t.Fatal(err)
			}
			if err := request.Transition(RequestQueued, 0, ""); err != nil {
				t.Fatal(err)
			}
			if err := request.Transition(RequestInFlight, 1, ""); err != nil {
				t.Fatal(err)
			}

			engine.requests["probe"] = request
			engine.requestNodes["probe"] = "worker"
			engine.workerBusy["worker"] = 1
			engine.now = 2

			breaker := engine.circuitBreakers["worker"]
			breaker.state = CircuitHalfOpen
			breaker.probeRequestID = "probe"

			if err := engine.handleTerminalRequest("probe", tt.terminalState); err != nil {
				t.Fatalf("handleTerminalRequest() error = %v", err)
			}

			if request.State() != tt.terminalState {
				t.Errorf("request state = %v, want %v", request.State(), tt.terminalState)
			}
			if breaker.state != CircuitOpen {
				t.Errorf("circuit state = %v, want %v", breaker.state, CircuitOpen)
			}
			if breaker.probeRequestID != "" {
				t.Errorf("probe reservation = %q, want empty", breaker.probeRequestID)
			}
			if breaker.openUntil != 7 {
				t.Errorf("openUntil = %d, want 7", breaker.openUntil)
			}

			event, ok := engine.queue.Pop()
			if !ok {
				t.Fatal("expected a scheduled cooldown event")
			}
			if event.Type != EventCircuitCooldownElapsed {
				t.Errorf("event type = %v, want %v", event.Type, EventCircuitCooldownElapsed)
			}
			if event.NodeID != "worker" {
				t.Errorf("event worker = %q, want worker", event.NodeID)
			}
			if event.At != 7 {
				t.Errorf("event tick = %d, want 7", event.At)
			}
		})
	}
}

func TestCircuitBreakerCooldownResumesQueuedRequestAsProbe(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.SetCircuitBreakerPolicy(CircuitBreakerPolicy{
		FailureThreshold: 2,
		OpenDuration:     5,
	}); err != nil {
		t.Fatal(err)
	}

	request, err := NewRequest("queued", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Transition(RequestQueued, 0, ""); err != nil {
		t.Fatal(err)
	}

	engine.requests["queued"] = request
	engine.requestNodes["queued"] = "worker"
	engine.workerWaiting["worker"] = []string{"queued"}
	engine.workerOnline["worker"] = true
	engine.now = 4

	breaker := engine.circuitBreakers["worker"]
	breaker.state = CircuitOpen
	breaker.openUntil = 5

	// The request must remain queued before cooldown expiry.
	if err := engine.handleCircuitCooldownElapsed("worker"); err != nil {
		t.Fatal(err)
	}
	if request.State() != RequestQueued {
		t.Fatalf("state before cooldown = %v, want %v", request.State(), RequestQueued)
	}
	if breaker.state != CircuitOpen {
		t.Fatalf("circuit before cooldown = %v, want %v", breaker.state, CircuitOpen)
	}

	// At expiry, the oldest waiting request should become the probe.
	engine.now = 5
	if err := engine.handleCircuitCooldownElapsed("worker"); err != nil {
		t.Fatal(err)
	}

	if request.State() != RequestInFlight {
		t.Errorf("state after cooldown = %v, want %v", request.State(), RequestInFlight)
	}
	if breaker.state != CircuitHalfOpen {
		t.Errorf("circuit after cooldown = %v, want %v", breaker.state, CircuitHalfOpen)
	}
	if breaker.probeRequestID != "queued" {
		t.Errorf("probe reservation = %q, want queued", breaker.probeRequestID)
	}
	if got := engine.workerWaiting["worker"]; len(got) != 0 {
		t.Errorf("waiting queue = %v, want empty", got)
	}
}

func TestCircuitBreakerCompletionResumesExpiredCooldown(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 10, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.SetCircuitBreakerPolicy(CircuitBreakerPolicy{
		FailureThreshold: 2,
		OpenDuration:     5,
	}); err != nil {
		t.Fatal(err)
	}

	// Prepare one active request and one queued request.
	for _, id := range []string{"active", "queued"} {
		if err := engine.ScheduleRoutedArrival(id, "client", 0); err != nil {
			t.Fatal(err)
		}
	}

	for engine.workerBusy["worker"] == 0 ||
		len(engine.workerWaiting["worker"]) == 0 {
		event, ok := engine.queue.Pop()
		if !ok {
			t.Fatal("event queue emptied before worker had active and queued requests")
		}
		engine.now = event.At

		switch event.Type {
		case EventRequestArrives:
			if err := engine.handleArrival(event.RequestID); err != nil {
				t.Fatal(err)
			}
		case EventRouteHop:
			if err := engine.handleRouteHop(event.RequestID, event.NodeID); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unexpected event while preparing test: %s", event.Type)
		}
	}

	breaker := engine.circuitBreakers["worker"]
	breaker.state = CircuitOpen
	breaker.openUntil = 5

	// The cooldown expires while the worker remains busy.
	engine.now = 5
	if err := engine.handleCircuitCooldownElapsed("worker"); err != nil {
		t.Fatal(err)
	}
	if engine.Requests()["queued"].State() != RequestQueued {
		t.Fatal("queued request should wait while the worker is at capacity")
	}

	// Completion frees capacity after the cooldown has elapsed.
	var completion Event
	for engine.queue.Len() > 0 {
		event, ok := engine.queue.Pop()
		if !ok {
			break
		}
		if event.Type == EventServiceCompletes && event.RequestID == "active" {
			completion = event
			break
		}
	}
	if completion.Type != EventServiceCompletes {
		t.Fatal("active request completion event not found")
	}

	engine.now = completion.At
	if err := engine.handleCompletion(
		completion.RequestID,
		completion.NodeID,
		completion.AttemptID,
	); err != nil {
		t.Fatalf("handleCompletion() error = %v", err)
	}

	if got := engine.Requests()["queued"].State(); got != RequestInFlight {
		t.Errorf("queued request state = %v, want %v", got, RequestInFlight)
	}
	if breaker.state != CircuitHalfOpen {
		t.Errorf("circuit state = %v, want %v", breaker.state, CircuitHalfOpen)
	}
	if breaker.probeRequestID != "queued" {
		t.Errorf("probe reservation = %q, want queued", breaker.probeRequestID)
	}
	if got := engine.workerBusy["worker"]; got != 1 {
		t.Errorf("worker busy count = %d, want 1", got)
	}
}
