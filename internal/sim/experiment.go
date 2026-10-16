package sim

import (
	"fmt"
	"strings"
)

// ExperimentSuite defines an ordered set of scenarios to execute and compare.
// The first scenario is the baseline for every subsequent scenario.
type ExperimentSuite struct {
	Name      string
	Scenarios []Scenario
}

// ExperimentSuiteResult contains scenario results and baseline comparisons.
type ExperimentSuiteResult struct {
	Name        string
	Scenarios   []ScenarioResult
	Comparisons []ComparisonReport
}

// Validate checks the suite and all of its scenarios before execution.
func (s ExperimentSuite) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("experiment suite name must not be empty")
	}

	if len(s.Scenarios) < 2 {
		return fmt.Errorf("experiment suite must contain at least two scenarios")
	}

	seen := make(map[string]struct{}, len(s.Scenarios))
	for i, scenario := range s.Scenarios {
		if _, exists := seen[scenario.Name]; exists {
			return fmt.Errorf("duplicate scenario name %q", scenario.Name)
		}
		seen[scenario.Name] = struct{}{}

		if err := scenario.Validate(); err != nil {
			return fmt.Errorf("scenario %d (%q): %w", i, scenario.Name, err)
		}
	}

	return nil
}

// RunExperimentSuite validates and executes scenarios in order.
// Each candidate is compared with the first scenario as the baseline.
func RunExperimentSuite(s ExperimentSuite) (ExperimentSuiteResult, error) {
	if err := s.Validate(); err != nil {
		return ExperimentSuiteResult{}, err
	}

	result := ExperimentSuiteResult{
		Name:        s.Name,
		Scenarios:   make([]ScenarioResult, 0, len(s.Scenarios)),
		Comparisons: make([]ComparisonReport, 0, len(s.Scenarios)-1),
	}

	for _, scenario := range s.Scenarios {
		scenarioResult, err := RunScenario(scenario)
		if err != nil {
			return ExperimentSuiteResult{}, fmt.Errorf(
				"run scenario %q: %w",
				scenario.Name,
				err,
			)
		}

		result.Scenarios = append(result.Scenarios, scenarioResult)
	}

	baseline := result.Scenarios[0]
	for _, candidate := range result.Scenarios[1:] {
		comparison, err := CompareScenarios(baseline, candidate)
		if err != nil {
			return ExperimentSuiteResult{}, fmt.Errorf(
				"compare baseline %q with candidate %q: %w",
				baseline.Name,
				candidate.Name,
				err,
			)
		}

		result.Comparisons = append(result.Comparisons, comparison)
	}

	return result, nil
}
