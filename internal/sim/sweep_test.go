package sim

import (
	"reflect"
	"testing"
)

func TestSweepServiceCapacityGeneratesSuite(t *testing.T) {
	baseline := Scenario{
		Name:            "baseline",
		ServiceCapacity: 1,
		ServiceDuration: 5,
		Arrivals: []ScenarioArrival{
			{RequestID: "first", At: 0},
			{RequestID: "second", At: 1},
		},
	}

	suite, err := SweepServiceCapacity(baseline, []int{2, 4, 8})
	if err != nil {
		t.Fatalf("SweepServiceCapacity() error = %v", err)
	}

	if suite.Name != "baseline-capacity-sweep" {
		t.Errorf("suite name = %q, want %q", suite.Name, "baseline-capacity-sweep")
	}

	if len(suite.Scenarios) != 4 {
		t.Fatalf("scenario count = %d, want 4", len(suite.Scenarios))
	}

	wantNames := []string{
		"baseline",
		"baseline-capacity-2",
		"baseline-capacity-4",
		"baseline-capacity-8",
	}
	wantCapacities := []int{1, 2, 4, 8}

	for i, scenario := range suite.Scenarios {
		if scenario.Name != wantNames[i] {
			t.Errorf("scenario %d name = %q, want %q", i, scenario.Name, wantNames[i])
		}
		if scenario.ServiceCapacity != wantCapacities[i] {
			t.Errorf("scenario %q capacity = %d, want %d",
				scenario.Name, scenario.ServiceCapacity, wantCapacities[i])
		}
		if err := scenario.Validate(); err != nil {
			t.Errorf("generated scenario %q is invalid: %v", scenario.Name, err)
		}
	}
}

func TestSweepServiceCapacityPreservesBaseline(t *testing.T) {
	baseline := Scenario{
		Name:            "baseline",
		ServiceCapacity: 1,
		ServiceDuration: 5,
		Arrivals: []ScenarioArrival{
			{RequestID: "first", At: 0},
			{RequestID: "second", At: 1},
		},
	}
	original := baseline

	_, err := SweepServiceCapacity(baseline, []int{2, 4})
	if err != nil {
		t.Fatalf("SweepServiceCapacity() error = %v", err)
	}

	if !reflect.DeepEqual(baseline, original) {
		t.Fatal("SweepServiceCapacity() mutated the baseline scenario")
	}
}

func TestSweepServiceCapacityRejectsInvalidInputs(t *testing.T) {
	validBaseline := Scenario{
		Name:            "baseline",
		ServiceCapacity: 1,
		ServiceDuration: 1,
		Arrivals: []ScenarioArrival{
			{RequestID: "req-1", At: 0},
		},
	}

	tests := []struct {
		name       string
		baseline   Scenario
		capacities []int
	}{
		{
			name:       "invalid baseline",
			baseline:   Scenario{},
			capacities: []int{2},
		},
		{
			name:       "empty capacities",
			baseline:   validBaseline,
			capacities: []int{},
		},
		{
			name:       "zero capacity",
			baseline:   validBaseline,
			capacities: []int{0},
		},
		{
			name:       "negative capacity",
			baseline:   validBaseline,
			capacities: []int{-2},
		},
		{
			name:       "duplicate capacities",
			baseline:   validBaseline,
			capacities: []int{2, 2},
		},
		{
			name:       "capacity duplicates baseline",
			baseline:   validBaseline,
			capacities: []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SweepServiceCapacity(tt.baseline, tt.capacities)
			if err == nil {
				t.Fatal("SweepServiceCapacity() error = nil, want non-nil")
			}
		})
	}
}

func TestSweepServiceCapacityPreservesScenarioConfiguration(t *testing.T) {
	baseline := Scenario{
		Name:            "baseline",
		ServiceCapacity: 1,
		ServiceDuration: 5,
		Arrivals: []ScenarioArrival{
			{RequestID: "first", At: 0},
			{RequestID: "second", At: 1},
		},
		RetryPolicy: RetryPolicy{MaxRetries: 2, Backoff: 3},
	}

	suite, err := SweepServiceCapacity(baseline, []int{2, 4})
	if err != nil {
		t.Fatalf("SweepServiceCapacity() error = %v", err)
	}

	for _, scenario := range suite.Scenarios {
		if scenario.ServiceDuration != baseline.ServiceDuration {
			t.Errorf("%q service duration = %d, want %d",
				scenario.Name, scenario.ServiceDuration, baseline.ServiceDuration)
		}
		if !reflect.DeepEqual(scenario.Arrivals, baseline.Arrivals) {
			t.Errorf("%q arrivals differ from baseline", scenario.Name)
		}
		if scenario.RetryPolicy != baseline.RetryPolicy {
			t.Errorf("%q retry policy differs from baseline", scenario.Name)
		}
	}
}

func TestSweepServiceCapacityRejectsTopology(t *testing.T) {
	baseline := Scenario{
		Name:            "topology-baseline",
		ServiceCapacity: 1,
		ServiceDuration: 1,
		Topology:        routingTestTopology(),
		Arrivals: []ScenarioArrival{
			{RequestID: "req-1", SourceNode: "client", At: 0},
		},
	}

	_, err := SweepServiceCapacity(baseline, []int{2, 4})
	if err == nil {
		t.Fatal("SweepServiceCapacity() error = nil, want topology to be rejected")
	}
}
