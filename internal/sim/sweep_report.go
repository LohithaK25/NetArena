package sim

import (
	"fmt"
	"strings"
)

// SweepReportRow contains the measurements for one scenario in a sweep.
type SweepReportRow struct {
	Name                   string
	ServiceCapacity        int
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

// SweepReport summarizes the scenarios and baseline comparisons in a sweep.
type SweepReport struct {
	Name        string
	Scenarios   []SweepReportRow
	Comparisons []ComparisonReport
}

// BuildSweepReport combines scenario configuration with completed execution results.
func BuildSweepReport(suite ExperimentSuite, result ExperimentSuiteResult) (SweepReport, error) {
	if err := suite.Validate(); err != nil {
		return SweepReport{}, fmt.Errorf("invalid experiment suite: %w", err)
	}

	if strings.TrimSpace(result.Name) == "" {
		return SweepReport{}, fmt.Errorf("experiment suite result name must not be empty")
	}
	if result.Name != suite.Name {
		return SweepReport{}, fmt.Errorf("suite name %q does not match result name %q", suite.Name, result.Name)
	}
	if len(result.Scenarios) != len(suite.Scenarios) {
		return SweepReport{}, fmt.Errorf(
			"scenario count mismatch: suite has %d scenarios, result has %d",
			len(suite.Scenarios),
			len(result.Scenarios),
		)
	}
	if len(result.Comparisons) != len(result.Scenarios)-1 {
		return SweepReport{}, fmt.Errorf(
			"comparison count must be one less than scenario count: got %d comparisons for %d scenarios",
			len(result.Comparisons),
			len(result.Scenarios),
		)
	}

	rows := make([]SweepReportRow, 0, len(result.Scenarios))
	for i, scenarioResult := range result.Scenarios {
		scenario := suite.Scenarios[i]
		if strings.TrimSpace(scenarioResult.Name) == "" {
			return SweepReport{}, fmt.Errorf("scenario result name must not be empty at index %d", i)
		}
		if scenarioResult.Name != scenario.Name {
			return SweepReport{}, fmt.Errorf(
				"scenario name mismatch at index %d: suite has %q, result has %q",
				i,
				scenario.Name,
				scenarioResult.Name,
			)
		}
		if scenarioResult.Duration <= 0 {
			return SweepReport{}, fmt.Errorf("scenario %q duration must be positive", scenarioResult.Name)
		}
		if scenarioResult.Summary.TotalRequests <= 0 {
			return SweepReport{}, fmt.Errorf("scenario %q must contain at least one request", scenarioResult.Name)
		}

		summary := scenarioResult.Summary
		rows = append(rows, SweepReportRow{
			Name:                   scenarioResult.Name,
			ServiceCapacity:        scenario.ServiceCapacity,
			Duration:               scenarioResult.Duration,
			TotalRequests:          summary.TotalRequests,
			SucceededRequests:      summary.SucceededRequests,
			FailedRequests:         summary.FailedRequests,
			CancelledRequests:      summary.CancelledRequests,
			TimedOutRequests:       summary.TimedOutRequests,
			SuccessRate:            float64(summary.SucceededRequests) / float64(summary.TotalRequests),
			Throughput:             float64(summary.SucceededRequests) / float64(scenarioResult.Duration),
			AverageEndToEndLatency: summary.EndToEndLatency.Average,
			Retries:                summary.Retries,
			WorkerUtilization:      scenarioResult.Metrics.WorkerUtilization,
		})
	}

	comparisons := append([]ComparisonReport(nil), result.Comparisons...)

	return SweepReport{
		Name:        result.Name,
		Scenarios:   rows,
		Comparisons: comparisons,
	}, nil
}
