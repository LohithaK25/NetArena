package sim

import "testing"

func experimentTestScenario(name string, capacity int) Scenario {
	return Scenario{
		Name:            name,
		ServiceCapacity: capacity,
		ServiceDuration: 2,
		Arrivals: []ScenarioArrival{
			{RequestID: "request-1", At: 0},
		},
	}
}

func TestExperimentSuiteValidate(t *testing.T) {
	validBaseline := experimentTestScenario("baseline", 1)
	validCandidate := experimentTestScenario("candidate", 2)

	tests := []struct {
		name  string
		suite ExperimentSuite
	}{
		{
			name: "empty suite name",
			suite: ExperimentSuite{
				Scenarios: []Scenario{validBaseline, validCandidate},
			},
		},
		{
			name: "no scenarios",
			suite: ExperimentSuite{
				Name: "empty",
			},
		},
		{
			name: "only one scenario",
			suite: ExperimentSuite{
				Name:      "single",
				Scenarios: []Scenario{validBaseline},
			},
		},
		{
			name: "duplicate scenario names",
			suite: ExperimentSuite{
				Name: "duplicates",
				Scenarios: []Scenario{
					validBaseline,
					experimentTestScenario("baseline", 2),
				},
			},
		},
		{
			name: "invalid scenario",
			suite: ExperimentSuite{
				Name: "invalid",
				Scenarios: []Scenario{
					validBaseline,
					experimentTestScenario("invalid", 0),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.suite.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want an error")
			}
		})
	}

	validSuite := ExperimentSuite{
		Name: "valid experiment",
		Scenarios: []Scenario{
			validBaseline,
			validCandidate,
		},
	}
	if err := validSuite.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestRunExperimentSuiteComparesScenariosToBaseline(t *testing.T) {
	suite := ExperimentSuite{
		Name: "capacity experiment",
		Scenarios: []Scenario{
			experimentTestScenario("baseline", 1),
			experimentTestScenario("candidate-a", 2),
			experimentTestScenario("candidate-b", 3),
		},
	}

	result, err := RunExperimentSuite(suite)
	if err != nil {
		t.Fatalf("RunExperimentSuite() error = %v", err)
	}

	if result.Name != suite.Name {
		t.Errorf("result name = %q, want %q", result.Name, suite.Name)
	}

	if len(result.Scenarios) != 3 {
		t.Fatalf("scenario results = %d, want 3", len(result.Scenarios))
	}

	if len(result.Comparisons) != 2 {
		t.Fatalf("comparisons = %d, want 2", len(result.Comparisons))
	}

	if result.Scenarios[0].Name != "baseline" {
		t.Errorf(
			"first scenario = %q, want baseline",
			result.Scenarios[0].Name,
		)
	}

	wantCandidates := []string{"candidate-a", "candidate-b"}
	for i, comparison := range result.Comparisons {
		if comparison.Baseline.Name != "baseline" {
			t.Errorf(
				"comparison %d baseline = %q, want baseline",
				i,
				comparison.Baseline.Name,
			)
		}

		if comparison.Candidate.Name != wantCandidates[i] {
			t.Errorf(
				"comparison %d candidate = %q, want %q",
				i,
				comparison.Candidate.Name,
				wantCandidates[i],
			)
		}
	}
}

func TestRunExperimentSuiteIsDeterministic(t *testing.T) {
	suite := ExperimentSuite{
		Name: "deterministic experiment",
		Scenarios: []Scenario{
			experimentTestScenario("baseline", 1),
			experimentTestScenario("candidate", 2),
		},
	}

	first, err := RunExperimentSuite(suite)
	if err != nil {
		t.Fatalf("first RunExperimentSuite() error = %v", err)
	}

	second, err := RunExperimentSuite(suite)
	if err != nil {
		t.Fatalf("second RunExperimentSuite() error = %v", err)
	}

	if len(first.Scenarios) != len(second.Scenarios) {
		t.Fatalf(
			"scenario counts differ: first=%d second=%d",
			len(first.Scenarios),
			len(second.Scenarios),
		)
	}

	for i := range first.Scenarios {
		firstResult := first.Scenarios[i]
		secondResult := second.Scenarios[i]

		if firstResult.Name != secondResult.Name {
			t.Errorf("scenario %d name differs between runs", i)
		}
		if firstResult.Duration != secondResult.Duration {
			t.Errorf("scenario %d duration differs between runs", i)
		}
		if firstResult.Summary != secondResult.Summary {
			t.Errorf("scenario %d summary differs between runs", i)
		}
		if firstResult.Metrics.WorkerUtilization != secondResult.Metrics.WorkerUtilization {
			t.Errorf("scenario %d worker utilization differs between runs", i)
		}
		if len(firstResult.Metrics.Workers) != len(secondResult.Metrics.Workers) {
			t.Errorf("scenario %d worker counts differ between runs", i)
		}

		for workerID, firstWorker := range firstResult.Metrics.Workers {
			secondWorker, ok := secondResult.Metrics.Workers[workerID]
			if !ok || firstWorker != secondWorker {
				t.Errorf("scenario %d worker %q differs between runs", i, workerID)
			}
		}
	}

	if len(first.Comparisons) != len(second.Comparisons) {
		t.Fatalf(
			"comparison counts differ: first=%d second=%d",
			len(first.Comparisons),
			len(second.Comparisons),
		)
	}

	for i := range first.Comparisons {
		if first.Comparisons[i].Baseline != second.Comparisons[i].Baseline {
			t.Errorf("comparison %d baseline differs between runs", i)
		}
		if first.Comparisons[i].Candidate != second.Comparisons[i].Candidate {
			t.Errorf("comparison %d candidate differs between runs", i)
		}
		if first.Comparisons[i].Delta != second.Comparisons[i].Delta {
			t.Errorf("comparison %d delta differs between runs", i)
		}
		if len(first.Comparisons[i].Workers) != len(second.Comparisons[i].Workers) {
			t.Errorf("comparison %d worker counts differ between runs", i)
		}
		for workerID, firstWorker := range first.Comparisons[i].Workers {
			secondWorker, ok := second.Comparisons[i].Workers[workerID]
			if !ok || firstWorker != secondWorker {
				t.Errorf("comparison %d worker %q differs between runs", i, workerID)
			}
		}
	}
}

func TestRunExperimentSuiteRejectsInvalidScenarioBeforeExecution(t *testing.T) {
	suite := ExperimentSuite{
		Name: "invalid experiment",
		Scenarios: []Scenario{
			experimentTestScenario("baseline", 1),
			experimentTestScenario("invalid", 0),
		},
	}

	result, err := RunExperimentSuite(suite)
	if err == nil {
		t.Fatal("RunExperimentSuite() error = nil, want validation error")
	}

	if result.Name != "" || result.Scenarios != nil || result.Comparisons != nil {
		t.Errorf("result = %+v, want zero-value result on validation failure", result)
	}
}
