package sim

import (
	"math"
	"testing"
)

func TestEvaluateObjectives(t *testing.T) {
	tests := []struct {
		name          string
		objectives    *ScenarioObjectives
		result        ScenarioResult
		wantPassed    bool
		wantCount     int
		wantActual    []float64
		wantThreshold []float64
	}{
		{
			name: "success rate meets threshold",
			objectives: &ScenarioObjectives{
				MinSuccessRate: floatPtr(0.8),
			},
			result: ScenarioResult{
				Summary: RunSummary{
					TotalRequests:     10,
					SucceededRequests: 8,
				},
			},
			wantPassed: true,
			wantCount:  1,

			wantActual:    []float64{0.8},
			wantThreshold: []float64{0.8},
		},
		{
			name: "success rate below threshold",
			objectives: &ScenarioObjectives{
				MinSuccessRate: floatPtr(0.9),
			},
			result: ScenarioResult{
				Summary: RunSummary{
					TotalRequests:     10,
					SucceededRequests: 8,
				},
			},
			wantPassed: false,
			wantCount:  1,

			wantActual:    []float64{0.8},
			wantThreshold: []float64{0.9},
		},
		{
			name: "latency meets threshold",
			objectives: &ScenarioObjectives{
				MaxAverageLatency: floatPtr(10),
			},
			result: ScenarioResult{
				Summary: RunSummary{
					EndToEndLatency: LatencyStats{
						Count:   2,
						Average: 10,
					},
				},
			},
			wantPassed: true,
			wantCount:  1,

			wantActual:    []float64{10},
			wantThreshold: []float64{10},
		},
		{
			name: "latency exceeds threshold",
			objectives: &ScenarioObjectives{
				MaxAverageLatency: floatPtr(10),
			},
			result: ScenarioResult{
				Summary: RunSummary{
					EndToEndLatency: LatencyStats{
						Count:   2,
						Average: 11,
					},
				},
			},
			wantPassed: false,
			wantCount:  1,

			wantActual:    []float64{11},
			wantThreshold: []float64{10},
		},
		{
			name: "no successful requests cannot meet latency objective",
			objectives: &ScenarioObjectives{
				MaxAverageLatency: floatPtr(10),
			},
			result: ScenarioResult{
				Summary: RunSummary{
					TotalRequests: 2,
				},
			},
			wantPassed: false,
			wantCount:  1,

			wantActual:    []float64{0},
			wantThreshold: []float64{10},
		},
		{
			name: "all objectives must pass",
			objectives: &ScenarioObjectives{
				MinSuccessRate:    floatPtr(0.8),
				MaxAverageLatency: floatPtr(10),
			},
			result: ScenarioResult{
				Summary: RunSummary{
					TotalRequests:     10,
					SucceededRequests: 8,
					EndToEndLatency: LatencyStats{
						Count:   8,
						Average: 11,
					},
				},
			},
			wantPassed: false,
			wantCount:  2,

			wantActual:    []float64{0.8, 11},
			wantThreshold: []float64{0.8, 10},
		},
		{
			name:       "no objectives",
			objectives: nil,
			wantPassed: true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateObjectives(tt.objectives, tt.result)
			if err != nil {
				t.Fatalf("EvaluateObjectives() error = %v", err)
			}
			if got.Passed != tt.wantPassed {
				t.Errorf("Passed = %v, want %v", got.Passed, tt.wantPassed)
			}
			if len(got.Outcomes) != tt.wantCount {
				t.Fatalf(
					"outcome count = %d, want %d",
					len(got.Outcomes),
					tt.wantCount,
				)
			}
			if len(got.Outcomes) != len(tt.wantActual) ||
				len(got.Outcomes) != len(tt.wantThreshold) {
				t.Fatalf("outcome expectations have inconsistent lengths")
			}

			for i, outcome := range got.Outcomes {
				if outcome.Actual != tt.wantActual[i] {
					t.Errorf("outcome[%d] actual = %v, want %v",
						i, outcome.Actual, tt.wantActual[i])
				}
				if outcome.Threshold != tt.wantThreshold[i] {
					t.Errorf("outcome[%d] threshold = %v, want %v",
						i, outcome.Threshold, tt.wantThreshold[i])
				}
			}
		})
	}
}

func TestRunScenarioEvaluatesConfiguredObjectives(t *testing.T) {
	scenario := Scenario{
		Name:            "objective-integration",
		ServiceCapacity: 1,
		ServiceDuration: 1,
		Arrivals: []ScenarioArrival{
			{RequestID: "request-1", At: 0},
			{RequestID: "request-2", At: 1},
		},
		Objectives: &ScenarioObjectives{
			MinSuccessRate: floatPtr(1),
		},
	}

	result, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("RunScenario() error = %v", err)
	}

	if !result.Objectives.Configured {
		t.Fatal("Objectives.Configured = false, want true")
	}
	if !result.Objectives.Passed {
		t.Fatalf("Objectives.Passed = false, outcomes: %+v", result.Objectives.Outcomes)
	}
	if len(result.Objectives.Outcomes) != 1 {
		t.Fatalf("outcome count = %d, want 1", len(result.Objectives.Outcomes))
	}
}

// The exported evaluator must reject invalid thresholds even when callers
// bypass Scenario.Validate.
func TestEvaluateObjectivesRejectNonFiniteValues(t *testing.T) {
	nonFinite := []struct {
		name  string
		value float64
	}{
		{name: "NaN", value: math.NaN()},
		{name: "positive infinity", value: math.Inf(1)},
		{name: "negative infinity", value: math.Inf(-1)},
	}

	for _, tc := range nonFinite {
		t.Run("success rate/"+tc.name, func(t *testing.T) {
			_, err := EvaluateObjectives(
				&ScenarioObjectives{MinSuccessRate: floatPtr(tc.value)},
				ScenarioResult{},
			)
			if err == nil {
				t.Fatal("EvaluateObjectives() error = nil, want error for non-finite success rate")
			}
		})

		t.Run("average latency/"+tc.name, func(t *testing.T) {
			_, err := EvaluateObjectives(
				&ScenarioObjectives{MaxAverageLatency: floatPtr(tc.value)},
				ScenarioResult{},
			)
			if err == nil {
				t.Fatal("EvaluateObjectives() error = nil, want error for non-finite latency")
			}
		})
	}
}
