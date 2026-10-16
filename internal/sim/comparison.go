package sim

import (
	"fmt"
	"strings"
)

// ScenarioComparisonMetrics contains comparable measurements for one scenario.
type ScenarioComparisonMetrics struct {
	Name                   string
	Duration               Tick
	TotalRequests          int
	SucceededRequests      int
	FailedRequests         int
	CancelledRequests      int
	TimedOutRequests       int
	SuccessRate            float64
	Throughput             float64
	AverageEndToEndLatency float64
	Retries                int
	WorkerUtilization      float64
}

// ScenarioComparisonDelta contains candidate-minus-baseline metric changes.
type ScenarioComparisonDelta struct {
	SuccessRate            float64
	Throughput             float64
	AverageEndToEndLatency float64
	Retries                int
	WorkerUtilization      float64
}

// WorkerComparison contains one worker's metrics from each scenario.
type WorkerComparison struct {
	HasBaseline  bool
	HasCandidate bool

	Baseline  WorkerMetrics
	Candidate WorkerMetrics

	UtilizationDelta float64
}

// ComparisonReport compares a candidate scenario with a baseline.
type ComparisonReport struct {
	Baseline  ScenarioComparisonMetrics
	Candidate ScenarioComparisonMetrics
	Delta     ScenarioComparisonDelta
	Workers   map[string]WorkerComparison
}

// CompareScenarios compares two completed scenario results.
//
// Throughput is successful requests per virtual tick. Success rate is the
// fraction of successful requests among all requests. Worker utilization is
// the final snapshot, not a time-weighted average. Latency is zero when a
// scenario has no successful requests.
func CompareScenarios(baseline, candidate ScenarioResult) (ComparisonReport, error) {
	if strings.TrimSpace(baseline.Name) == "" {
		return ComparisonReport{}, fmt.Errorf("baseline scenario name must not be empty")
	}
	if strings.TrimSpace(candidate.Name) == "" {
		return ComparisonReport{}, fmt.Errorf("candidate scenario name must not be empty")
	}
	if baseline.Name == candidate.Name {
		return ComparisonReport{}, fmt.Errorf("scenario names must be different")
	}
	if baseline.Duration <= 0 {
		return ComparisonReport{}, fmt.Errorf("baseline duration must be positive")
	}
	if candidate.Duration <= 0 {
		return ComparisonReport{}, fmt.Errorf("candidate duration must be positive")
	}
	if baseline.Summary.TotalRequests <= 0 {
		return ComparisonReport{}, fmt.Errorf("baseline must contain at least one request")
	}
	if candidate.Summary.TotalRequests <= 0 {
		return ComparisonReport{}, fmt.Errorf("candidate must contain at least one request")
	}

	baseMetrics := comparisonMetrics(baseline)
	candidateMetrics := comparisonMetrics(candidate)
	workerComparisons := compareWorkers(baseline.Metrics.Workers, candidate.Metrics.Workers)

	return ComparisonReport{
		Baseline:  baseMetrics,
		Candidate: candidateMetrics,
		Delta: ScenarioComparisonDelta{
			SuccessRate:            candidateMetrics.SuccessRate - baseMetrics.SuccessRate,
			Throughput:             candidateMetrics.Throughput - baseMetrics.Throughput,
			AverageEndToEndLatency: candidateMetrics.AverageEndToEndLatency - baseMetrics.AverageEndToEndLatency,
			Retries:                candidateMetrics.Retries - baseMetrics.Retries,
			WorkerUtilization:      candidateMetrics.WorkerUtilization - baseMetrics.WorkerUtilization,
		},
		Workers: workerComparisons,
	}, nil
}

func comparisonMetrics(result ScenarioResult) ScenarioComparisonMetrics {
	summary := result.Summary

	successRate := float64(summary.SucceededRequests) / float64(summary.TotalRequests)
	throughput := float64(summary.SucceededRequests) / float64(result.Duration)

	return ScenarioComparisonMetrics{
		Name:                   result.Name,
		Duration:               result.Duration,
		TotalRequests:          summary.TotalRequests,
		SucceededRequests:      summary.SucceededRequests,
		FailedRequests:         summary.FailedRequests,
		CancelledRequests:      summary.CancelledRequests,
		TimedOutRequests:       summary.TimedOutRequests,
		SuccessRate:            successRate,
		Throughput:             throughput,
		AverageEndToEndLatency: summary.EndToEndLatency.Average,
		Retries:                summary.Retries,
		WorkerUtilization:      result.Metrics.WorkerUtilization,
	}
}

func compareWorkers(
	baseline, candidate map[string]WorkerMetrics,
) map[string]WorkerComparison {
	comparisons := make(map[string]WorkerComparison, len(baseline)+len(candidate))

	for workerID, baselineMetrics := range baseline {
		comparison := WorkerComparison{
			HasBaseline: true,
			Baseline:    baselineMetrics,
		}

		if candidateMetrics, exists := candidate[workerID]; exists {
			comparison.HasCandidate = true
			comparison.Candidate = candidateMetrics
			comparison.UtilizationDelta = candidateMetrics.Utilization - baselineMetrics.Utilization
		}

		comparisons[workerID] = comparison
	}

	for workerID, candidateMetrics := range candidate {
		if _, exists := comparisons[workerID]; exists {
			continue
		}

		comparisons[workerID] = WorkerComparison{
			HasCandidate: true,
			Candidate:    candidateMetrics,
		}
	}

	return comparisons
}
