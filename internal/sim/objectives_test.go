package sim

import (
	"math"
	"testing"
)

func TestScenarioObjectivesValidation(t *testing.T) {
	valid := func() Scenario {
		return Scenario{
			Name:            "objective-test",
			ServiceCapacity: 1,
			ServiceDuration: 1,
			Arrivals: []ScenarioArrival{
				{RequestID: "request-1", At: 0},
			},
		}
	}

	tests := []struct {
		name       string
		objectives *ScenarioObjectives
		wantError  bool
	}{
		{
			name: "minimum success rate at zero",
			objectives: &ScenarioObjectives{
				MinSuccessRate: floatPtr(0),
			},
		},
		{
			name: "minimum success rate at one",
			objectives: &ScenarioObjectives{
				MinSuccessRate: floatPtr(1),
			},
		},
		{
			name: "maximum latency at zero",
			objectives: &ScenarioObjectives{
				MaxAverageLatency: floatPtr(0),
			},
		},
		{
			name: "both criteria",
			objectives: &ScenarioObjectives{
				MinSuccessRate:    floatPtr(0.95),
				MaxAverageLatency: floatPtr(20),
			},
		},
		{
			name: "no objectives configured",
		},
		{
			name: "success rate below zero",
			objectives: &ScenarioObjectives{
				MinSuccessRate: floatPtr(-0.01),
			},
			wantError: true,
		},
		{
			name: "success rate above one",
			objectives: &ScenarioObjectives{
				MinSuccessRate: floatPtr(1.01),
			},
			wantError: true,
		},
		{
			name: "negative latency",
			objectives: &ScenarioObjectives{
				MaxAverageLatency: floatPtr(-1),
			},
			wantError: true,
		},
		{
			name: "latency exceeds maximum tick",
			objectives: &ScenarioObjectives{
				MaxAverageLatency: floatPtr(MaxScenarioTick + 1),
			},
			wantError: true,
		},
		{
			name:       "empty objective configuration",
			objectives: &ScenarioObjectives{},
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := valid()
			scenario.Objectives = tt.objectives

			err := scenario.Validate()
			if tt.wantError && err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

// Non-finite thresholds must be rejected before a scenario can run.
func TestScenarioObjectivesRejectNonFiniteValues(t *testing.T) {
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
			scenario := Scenario{
				Name:            "invalid-objective",
				ServiceCapacity: 1,
				ServiceDuration: 1,
				Arrivals:        []ScenarioArrival{{RequestID: "request-1", At: 0}},
				Objectives:      &ScenarioObjectives{MinSuccessRate: floatPtr(tc.value)},
			}
			if err := scenario.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error for non-finite success rate")
			}
		})

		t.Run("average latency/"+tc.name, func(t *testing.T) {
			scenario := Scenario{
				Name:            "invalid-objective",
				ServiceCapacity: 1,
				ServiceDuration: 1,
				Arrivals:        []ScenarioArrival{{RequestID: "request-1", At: 0}},
				Objectives:      &ScenarioObjectives{MaxAverageLatency: floatPtr(tc.value)},
			}
			if err := scenario.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error for non-finite latency")
			}
		})
	}
}
