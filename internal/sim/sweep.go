package sim

import (
	"fmt"
)

// SweepServiceCapacity creates an experiment suite that compares a baseline
// scenario against scenarios using the requested service capacities.
//
// The baseline is included as the first scenario. Candidate capacities must
// be positive, unique, and different from the baseline capacity.
//
// Topology-based scenarios are rejected because their worker capacities are
// configured on topology nodes rather than solely by ServiceCapacity.
func SweepServiceCapacity(baseline Scenario, capacities []int) (ExperimentSuite, error) {
	if err := baseline.Validate(); err != nil {
		return ExperimentSuite{}, fmt.Errorf("invalid baseline scenario: %w", err)
	}

	if len(baseline.Topology.Nodes) > 0 || len(baseline.Topology.Connections) > 0 {
		return ExperimentSuite{}, fmt.Errorf("service capacity sweeps do not support topology-based scenarios")
	}

	if len(capacities) == 0 {
		return ExperimentSuite{}, fmt.Errorf("at least one candidate capacity is required")
	}

	seen := map[int]struct{}{
		baseline.ServiceCapacity: {},
	}

	scenarios := make([]Scenario, 0, len(capacities)+1)
	scenarios = append(scenarios, baseline)

	for _, capacity := range capacities {
		if capacity <= 0 {
			return ExperimentSuite{}, fmt.Errorf("capacity must be positive: %d", capacity)
		}

		if _, exists := seen[capacity]; exists {
			return ExperimentSuite{}, fmt.Errorf("duplicate capacity: %d", capacity)
		}
		seen[capacity] = struct{}{}

		candidate := baseline
		candidate.Name = fmt.Sprintf("%s-capacity-%d", baseline.Name, capacity)
		candidate.ServiceCapacity = capacity
		scenarios = append(scenarios, candidate)
	}

	suite := ExperimentSuite{
		Name:      baseline.Name + "-capacity-sweep",
		Scenarios: scenarios,
	}

	if err := suite.Validate(); err != nil {
		return ExperimentSuite{}, fmt.Errorf("invalid generated experiment suite: %w", err)
	}

	return suite, nil
}
