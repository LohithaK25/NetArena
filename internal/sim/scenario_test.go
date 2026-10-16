package sim

import (
	"reflect"
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
