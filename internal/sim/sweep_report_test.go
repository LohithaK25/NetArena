package sim

import "testing"

func TestBuildSweepReportFromCapacitySweep(t *testing.T) {
	baseline := Scenario{
		Name:            "baseline",
		ServiceCapacity: 1,
		ServiceDuration: 5,
		Arrivals: []ScenarioArrival{
			{RequestID: "first", At: 0},
			{RequestID: "second", At: 1},
		},
	}

	suite, err := SweepServiceCapacity(baseline, []int{2, 4})
	if err != nil {
		t.Fatalf("SweepServiceCapacity() error = %v", err)
	}

	result, err := RunExperimentSuite(suite)
	if err != nil {
		t.Fatalf("RunExperimentSuite() error = %v", err)
	}

	report, err := BuildSweepReport(suite, result)
	if err != nil {
		t.Fatalf("BuildSweepReport() error = %v", err)
	}

	if report.Name != suite.Name {
		t.Errorf("report name = %q, want %q", report.Name, suite.Name)
	}

	if len(report.Scenarios) != 3 {
		t.Fatalf("report scenario count = %d, want 3", len(report.Scenarios))
	}

	wantNames := []string{
		"baseline",
		"baseline-capacity-2",
		"baseline-capacity-4",
	}
	wantCapacities := []int{1, 2, 4}

	for i, row := range report.Scenarios {
		if row.Name != wantNames[i] {
			t.Errorf("row %d name = %q, want %q", i, row.Name, wantNames[i])
		}
		if row.ServiceCapacity != wantCapacities[i] {
			t.Errorf("row %q capacity = %d, want %d",
				row.Name, row.ServiceCapacity, wantCapacities[i])
		}
		if row.TotalRequests != 2 {
			t.Errorf("row %q total requests = %d, want 2",
				row.Name, row.TotalRequests)
		}
		if row.SuccessRate < 0 || row.SuccessRate > 1 {
			t.Errorf("row %q has invalid success rate %v",
				row.Name, row.SuccessRate)
		}
	}

	if len(report.Comparisons) != 2 {
		t.Fatalf("comparison count = %d, want 2", len(report.Comparisons))
	}
}

func TestBuildSweepReportRejectsInvalidResults(t *testing.T) {
	validSuite := ExperimentSuite{
		Name: "valid",
		Scenarios: []Scenario{
			{
				Name:            "baseline",
				ServiceCapacity: 1,
				ServiceDuration: 1,
				Arrivals: []ScenarioArrival{
					{RequestID: "req-1", At: 0},
				},
			},
			{
				Name:            "candidate",
				ServiceCapacity: 2,
				ServiceDuration: 1,
				Arrivals: []ScenarioArrival{
					{RequestID: "req-1", At: 0},
				},
			},
		},
	}

	tests := []struct {
		name   string
		suite  ExperimentSuite
		result ExperimentSuiteResult
	}{
		{
			name:  "empty suite name",
			suite: ExperimentSuite{},
			result: ExperimentSuiteResult{
				Name: "valid",
			},
		},
		{
			name:  "no scenarios",
			suite: validSuite,
			result: ExperimentSuiteResult{
				Name: "valid",
			},
		},
		{
			name:  "only one scenario",
			suite: validSuite,
			result: ExperimentSuiteResult{
				Name: "valid",
				Scenarios: []ScenarioResult{
					{Name: "baseline"},
				},
			},
		},
		{
			name:  "missing comparison",
			suite: validSuite,
			result: ExperimentSuiteResult{
				Name: "valid",
				Scenarios: []ScenarioResult{
					{Name: "baseline"},
					{Name: "candidate"},
				},
			},
		},
		{
			name:  "suite and result names mismatch",
			suite: validSuite,
			result: ExperimentSuiteResult{
				Name: "different-suite",
			},
		},
		{
			name:  "scenario names mismatch",
			suite: validSuite,
			result: ExperimentSuiteResult{
				Name: "valid",
				Scenarios: []ScenarioResult{
					{Name: "wrong-baseline"},
					{Name: "candidate"},
				},
				Comparisons: []ComparisonReport{{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildSweepReport(tt.suite, tt.result)
			if err == nil {
				t.Fatal("BuildSweepReport() error = nil, want non-nil")
			}
		})
	}
}
