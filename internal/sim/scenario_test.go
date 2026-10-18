package sim

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestScenarioRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		scenario Scenario
	}{
		{
			name: "empty name",
			scenario: Scenario{
				ServiceCapacity: 1,
				ServiceDuration: 1,
			},
		},
		{
			name: "zero capacity",
			scenario: Scenario{
				Name:            "invalid",
				ServiceCapacity: 0,
				ServiceDuration: 1,
			},
		},
		{
			name: "zero duration",
			scenario: Scenario{
				Name:            "invalid",
				ServiceCapacity: 1,
				ServiceDuration: 0,
			},
		},
		{
			name: "duplicate arrival ID",
			scenario: Scenario{
				Name:            "invalid",
				ServiceCapacity: 1,
				ServiceDuration: 1,
				Arrivals: []ScenarioArrival{
					{RequestID: "req-1", At: 0},
					{RequestID: "req-1", At: 1},
				},
			},
		},
		{
			name: "negative arrival tick",
			scenario: Scenario{
				Name:            "invalid",
				ServiceCapacity: 1,
				ServiceDuration: 1,
				Arrivals: []ScenarioArrival{
					{RequestID: "req-1", At: -1},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.scenario.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

func TestRunScenarioContextRejectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scenario := Scenario{
		Name:            "cancelled",
		ServiceCapacity: 1,
		ServiceDuration: 1,
		Arrivals: []ScenarioArrival{
			{RequestID: "req-1", At: 0},
		},
	}

	_, err := RunScenarioContext(ctx, scenario)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunScenarioContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestScenarioRunnerProducesDeterministicResult(t *testing.T) {
	scenario := Scenario{
		Name:            "baseline",
		ServiceCapacity: 1,
		ServiceDuration: 5,
		Arrivals: []ScenarioArrival{
			{RequestID: "first", At: 0},
			{RequestID: "second", At: 1},
		},
	}

	first, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("first RunScenario() error = %v", err)
	}

	second, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("second RunScenario() error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("scenario results differ:\nfirst:  %+v\nsecond: %+v", first, second)
	}

	if first.Name != "baseline" {
		t.Errorf("Name = %q, want baseline", first.Name)
	}
	if first.Summary.SucceededRequests != 2 {
		t.Errorf("SucceededRequests = %d, want 2", first.Summary.SucceededRequests)
	}
	if first.Summary.EndToEndLatency.Average != 7 {
		t.Errorf("EndToEndLatency.Average = %v, want 7", first.Summary.EndToEndLatency.Average)
	}
	if first.Duration != 10 {
		t.Errorf("Duration = %d, want 10", first.Duration)
	}
}

func TestScenarioRunnerSupportsTopologyAndWorkerFailure(t *testing.T) {
	scenario := Scenario{
		Name:            "worker-recovery",
		ServiceCapacity: 1,
		ServiceDuration: 2,
		Topology:        routingTestTopology(),
		RetryPolicy:     RetryPolicy{MaxRetries: 1, Backoff: 3},
		Arrivals: []ScenarioArrival{
			{RequestID: "req-1", SourceNode: "client", At: 0},
		},
		WorkerFailures: []ScenarioWorkerEvent{
			{WorkerID: "worker", At: 1},
		},
		WorkerRecoveries: []ScenarioWorkerEvent{
			{WorkerID: "worker", At: 3},
		},
	}

	result, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("RunScenario() error = %v", err)
	}

	if result.Summary.SucceededRequests != 1 {
		t.Errorf("SucceededRequests = %d, want 1", result.Summary.SucceededRequests)
	}
	if result.Metrics.Workers["worker"].Online != true {
		t.Errorf("worker Online = false, want true after recovery")
	}
}

func TestRunScenarioRejectsInvalidScenario(t *testing.T) {
	scenario := Scenario{
		Name:            "invalid",
		ServiceCapacity: 0,
		ServiceDuration: 1,
		Arrivals: []ScenarioArrival{
			{RequestID: "req-1", At: 0},
		},
	}

	if _, err := RunScenario(scenario); err == nil {
		t.Fatal("RunScenario() error = nil, want validation error")
	}
}

func TestScenarioRejectsRequestEventsBeforeArrival(t *testing.T) {
	tests := []struct {
		name     string
		scenario Scenario
	}{
		{
			name: "timeout before arrival",
			scenario: Scenario{
				Name:            "invalid-timeout",
				ServiceCapacity: 1,
				ServiceDuration: 1,
				Arrivals: []ScenarioArrival{
					{RequestID: "req-1", At: 5},
				},
				Timeouts: []ScenarioTimeout{
					{RequestID: "req-1", At: 4},
				},
			},
		},
		{
			name: "cancellation before arrival",
			scenario: Scenario{
				Name:            "invalid-cancellation",
				ServiceCapacity: 1,
				ServiceDuration: 1,
				Arrivals: []ScenarioArrival{
					{RequestID: "req-1", At: 5},
				},
				Cancellations: []ScenarioCancellation{
					{RequestID: "req-1", At: 4},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.scenario.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

func TestScenarioRejectsResourceLimitViolations(t *testing.T) {
	validScenario := func() Scenario {
		return Scenario{
			Name:            "resource-limit-test",
			ServiceCapacity: 1,
			ServiceDuration: 1,
			Arrivals: []ScenarioArrival{
				{RequestID: "request-1", At: 0},
			},
		}
	}

	tests := []struct {
		name      string
		scenario  func() Scenario
		wantError bool
	}{
		{
			name: "arrival count at limit",
			scenario: func() Scenario {
				s := validScenario()
				s.Arrivals = make([]ScenarioArrival, MaxScenarioArrivals)
				for i := range s.Arrivals {
					s.Arrivals[i] = ScenarioArrival{
						RequestID: fmt.Sprintf("request-%d", i),
						At:        0,
					}
				}
				return s
			},
		},
		{
			name: "arrival count above limit",
			scenario: func() Scenario {
				s := validScenario()
				s.Arrivals = make([]ScenarioArrival, MaxScenarioArrivals+1)
				for i := range s.Arrivals {
					s.Arrivals[i] = ScenarioArrival{
						RequestID: fmt.Sprintf("request-%d", i),
						At:        0,
					}
				}
				return s
			},
			wantError: true,
		},
		{
			name: "service capacity above limit",
			scenario: func() Scenario {
				s := validScenario()
				s.ServiceCapacity = MaxScenarioServiceCapacity + 1
				return s
			},
			wantError: true,
		},
		{
			name: "service duration above limit",
			scenario: func() Scenario {
				s := validScenario()
				s.ServiceDuration = MaxScenarioTick + 1
				return s
			},
			wantError: true,
		},
		{
			name: "arrival tick above limit",
			scenario: func() Scenario {
				s := validScenario()
				s.Arrivals[0].At = MaxScenarioTick + 1
				return s
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scenario().Validate()
			if tt.wantError && err == nil {
				t.Fatal("Validate() error = nil, want an error")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestScenarioRejectsTooManyScheduledEvents(t *testing.T) {
	scenario := Scenario{
		Name:            "event-limit-test",
		ServiceCapacity: 1,
		ServiceDuration: 1,
		Arrivals: []ScenarioArrival{
			{RequestID: "request-1", At: 0},
		},
		WorkerFailures: make([]ScenarioWorkerEvent, MaxScenarioEvents+1),
	}

	err := scenario.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want scheduled-event limit error")
	}
	if !strings.Contains(err.Error(), "scheduled event count") {
		t.Fatalf("Validate() error = %q, want scheduled-event limit error", err)
	}
}

func TestRunScenarioWithSameSeedProducesSameResult(t *testing.T) {
	scenario := Scenario{
		Name:            "seeded-replay",
		Seed:            42,
		ServiceCapacity: 1,
		ServiceDuration: 3,
		Arrivals: []ScenarioArrival{
			{RequestID: "req-1", At: 0},
			{RequestID: "req-2", At: 1},
			{RequestID: "req-3", At: 2},
		},
	}

	first, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("first RunScenario() error = %v", err)
	}

	second, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("second RunScenario() error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf(
			"results differ for identical seeds:\nfirst:  %+v\nsecond: %+v",
			first,
			second,
		)
	}
}

func TestScenarioValidatesSchedulingPolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  SchedulingPolicy
		wantErr bool
	}{
		{
			name:   "default policy",
			policy: "",
		},
		{
			name:   "FIFO policy",
			policy: SchedulingFIFO,
		},
		{
			name:   "priority policy",
			policy: SchedulingPriority,
		},
		{
			name:    "unsupported policy",
			policy:  SchedulingPolicy("round-robin"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := Scenario{
				Name:             "scheduling-policy-validation",
				ServiceCapacity:  1,
				ServiceDuration:  1,
				SchedulingPolicy: tt.policy,
				Arrivals: []ScenarioArrival{
					{RequestID: "req-1", At: 0},
				},
			}

			err := scenario.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf(
					"Validate() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestRunScenarioAppliesSchedulingPolicy(t *testing.T) {
	tests := []struct {
		name              string
		policy            SchedulingPolicy
		wantTimedOutCount int
	}{
		{
			name:              "omitted policy defaults to FIFO",
			policy:            "",
			wantTimedOutCount: 1,
		},
		{
			name:              "explicit FIFO",
			policy:            SchedulingFIFO,
			wantTimedOutCount: 1,
		},
		{
			name:              "priority scheduling",
			policy:            SchedulingPriority,
			wantTimedOutCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := Scenario{
				Name:             "scheduling-policy-integration",
				ServiceCapacity:  1,
				ServiceDuration:  5,
				SchedulingPolicy: tt.policy,
				Arrivals: []ScenarioArrival{
					{RequestID: "blocker", At: 0},
					{RequestID: "low", At: 1, Priority: 1},
					{RequestID: "high", At: 1, Priority: 10},
					{RequestID: "normal", At: 1, Priority: 0},
				},
				Timeouts: []ScenarioTimeout{
					{RequestID: "high", At: 7},
					{RequestID: "low", At: 11},
				},
			}

			result, err := RunScenario(scenario)
			if err != nil {
				t.Fatalf("RunScenario() error = %v", err)
			}

			if got := result.Summary.TimedOutRequests; got != tt.wantTimedOutCount {
				t.Errorf(
					"timed-out requests = %d, want %d",
					got,
					tt.wantTimedOutCount,
				)
			}
		})
	}
}

func TestRunScenarioAppliesRoutingAlgorithm(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "relay", Type: NodeService},
			{ID: "worker-slow", Type: NodeWorker},
			{ID: "worker-fast", Type: NodeWorker},
		},
		Connections: []Connection{
			// Round-robin selects this direct connection first.
			{From: "client", To: "worker-slow", Latency: 8},
			{From: "client", To: "relay", Latency: 2},
			{From: "relay", To: "worker-fast", Latency: 2},
		},
	}

	run := func(algorithm RoutingAlgorithm) ScenarioResult {
		t.Helper()

		result, err := RunScenario(Scenario{
			Name:             "routing-algorithm-test",
			ServiceCapacity:  1,
			ServiceDuration:  1,
			RoutingAlgorithm: algorithm,
			Topology:         topology,
			Arrivals: []ScenarioArrival{
				{RequestID: "request-1", SourceNode: "client", At: 0},
			},
		})
		if err != nil {
			t.Fatalf("RunScenario(%q) error = %v", algorithm, err)
		}
		if result.Summary.SucceededRequests != 1 {
			t.Fatalf(
				"RunScenario(%q) succeeded requests = %d, want 1",
				algorithm,
				result.Summary.SucceededRequests,
			)
		}

		return result
	}

	roundRobin := run(RoutingRoundRobin)
	shortestPath := run(RoutingShortestPath)

	if shortestPath.Summary.EndToEndLatency.Average >=
		roundRobin.Summary.EndToEndLatency.Average {
		t.Errorf(
			"shortest-path average latency = %v, round-robin = %v; want shortest-path to be lower",
			shortestPath.Summary.EndToEndLatency.Average,
			roundRobin.Summary.EndToEndLatency.Average,
		)
	}
}

func TestRunScenarioReturnsActualRouteTraces(t *testing.T) {
	scenario := Scenario{
		Name:             "route-trace-result",
		ServiceCapacity:  1,
		ServiceDuration:  1,
		RoutingAlgorithm: RoutingShortestPath,
		Topology: Topology{
			Nodes: []Node{
				{ID: "client", Type: NodeClient},
				{ID: "relay", Type: NodeService},
				{ID: "worker-slow", Type: NodeWorker},
				{ID: "worker-fast", Type: NodeWorker},
			},
			Connections: []Connection{
				{From: "client", To: "worker-slow", Latency: 8},
				{From: "client", To: "relay", Latency: 2},
				{From: "relay", To: "worker-fast", Latency: 2},
			},
		},
		Arrivals: []ScenarioArrival{
			{RequestID: "request-1", SourceNode: "client", At: 0},
		},
	}

	result, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("RunScenario() error = %v", err)
	}

	hops := result.RouteTraces["request-1"]
	want := []RouteTraceHop{
		{From: "client", To: "relay", At: 0, Latency: 2, Status: RouteHopForwarded},
		{From: "relay", To: "worker-fast", At: 2, Latency: 2, Status: RouteHopForwarded},
	}

	if len(hops) != len(want) {
		t.Fatalf("route traces = %+v, want %+v", hops, want)
	}
	for i := range want {
		if hops[i] != want[i] {
			t.Errorf("hop %d = %+v, want %+v", i, hops[i], want[i])
		}
	}
}
