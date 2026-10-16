package sim

import (
	"math"
	"testing"
)

func TestCompareScenariosCalculatesMetricDeltas(t *testing.T) {
	baseline := ScenarioResult{
		Name:     "baseline",
		Duration: 10,
		Summary: RunSummary{
			TotalRequests:     10,
			SucceededRequests: 8,
			Retries:           2,
			EndToEndLatency:   LatencyStats{Count: 8, Average: 5},
		},
		Metrics: Metrics{WorkerUtilization: 0.5},
	}

	failure := ScenarioResult{
		Name:     "worker-failure",
		Duration: 20,
		Summary: RunSummary{
			TotalRequests:     10,
			SucceededRequests: 6,
			Retries:           5,
			EndToEndLatency:   LatencyStats{Count: 6, Average: 8},
		},
		Metrics: Metrics{WorkerUtilization: 0.75},
	}

	report, err := CompareScenarios(baseline, failure)
	if err != nil {
		t.Fatalf("CompareScenarios() error = %v", err)
	}

	if report.Baseline.Name != "baseline" {
		t.Errorf("Baseline.Name = %q, want baseline", report.Baseline.Name)
	}
	if report.Candidate.Name != "worker-failure" {
		t.Errorf("Candidate.Name = %q, want worker-failure", report.Candidate.Name)
	}

	assertNear(t, "baseline success rate", report.Baseline.SuccessRate, 0.8)
	assertNear(t, "candidate success rate", report.Candidate.SuccessRate, 0.6)
	assertNear(t, "success rate delta", report.Delta.SuccessRate, -0.2)

	assertNear(t, "baseline throughput", report.Baseline.Throughput, 0.8)
	assertNear(t, "candidate throughput", report.Candidate.Throughput, 0.3)
	assertNear(t, "throughput delta", report.Delta.Throughput, -0.5)

	assertNear(t, "latency delta", report.Delta.AverageEndToEndLatency, 3)
	if report.Delta.Retries != 3 {
		t.Errorf("Retries delta = %d, want 3", report.Delta.Retries)
	}
	assertNear(t, "worker utilization delta", report.Delta.WorkerUtilization, 0.25)
}

func TestCompareScenariosRejectsInvalidInputs(t *testing.T) {
	valid := ScenarioResult{
		Name:     "baseline",
		Duration: 10,
		Summary: RunSummary{
			TotalRequests:     1,
			SucceededRequests: 1,
			EndToEndLatency:   LatencyStats{Count: 1, Average: 1},
		},
	}

	tests := []struct {
		name      string
		baseline  ScenarioResult
		candidate ScenarioResult
	}{
		{
			name:      "empty baseline name",
			baseline:  ScenarioResult{Duration: 1},
			candidate: valid,
		},
		{
			name:      "empty candidate name",
			baseline:  valid,
			candidate: ScenarioResult{Duration: 1},
		},
		{
			name:      "duplicate names",
			baseline:  valid,
			candidate: valid,
		},
		{
			name: "zero baseline duration",
			baseline: ScenarioResult{
				Name: "zero-duration",
			},
			candidate: valid,
		},
		{
			name:      "zero candidate duration",
			baseline:  valid,
			candidate: ScenarioResult{Name: "zero-duration"},
		},
		{
			name: "no baseline requests",
			baseline: ScenarioResult{
				Name:     "empty",
				Duration: 1,
			},
			candidate: valid,
		},
		{
			name:      "no candidate requests",
			baseline:  valid,
			candidate: ScenarioResult{Name: "empty", Duration: 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CompareScenarios(test.baseline, test.candidate); err == nil {
				t.Fatal("CompareScenarios() error = nil, want error")
			}
		})
	}
}

func TestCompareScenariosHandlesZeroSuccesses(t *testing.T) {
	baseline := ScenarioResult{
		Name:     "baseline",
		Duration: 10,
		Summary:  RunSummary{TotalRequests: 2},
	}
	candidate := ScenarioResult{
		Name:     "all-failed",
		Duration: 10,
		Summary:  RunSummary{TotalRequests: 2},
	}

	report, err := CompareScenarios(baseline, candidate)
	if err != nil {
		t.Fatalf("CompareScenarios() error = %v", err)
	}

	assertNear(t, "baseline success rate", report.Baseline.SuccessRate, 0)
	assertNear(t, "candidate throughput", report.Candidate.Throughput, 0)
	assertNear(t, "latency with no successes", report.Candidate.AverageEndToEndLatency, 0)
}

func assertNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
