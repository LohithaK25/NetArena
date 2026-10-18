package sim

import (
	"fmt"
	"math"
)

// ObjectiveOutcome reports whether one configured criterion was achieved.
type ObjectiveOutcome struct {
	Name      string  `json:"name"`
	Passed    bool    `json:"passed"`
	Actual    float64 `json:"actual"`
	Threshold float64 `json:"threshold"`
	Message   string  `json:"message"`
}

// ObjectiveEvaluation summarizes the result of evaluating scenario objectives.
type ObjectiveEvaluation struct {
	Configured bool               `json:"configured"`
	Passed     bool               `json:"passed"`
	Outcomes   []ObjectiveOutcome `json:"outcomes"`
}

// EvaluateObjectives evaluates configured objectives against actual simulation results.
func EvaluateObjectives(
	objectives *ScenarioObjectives,
	result ScenarioResult,
) (ObjectiveEvaluation, error) {
	if objectives == nil {
		return ObjectiveEvaluation{
			Configured: false,
			Passed:     true,
			Outcomes:   []ObjectiveOutcome{},
		}, nil
	}

	if objectives.MinSuccessRate != nil {
		rate := *objectives.MinSuccessRate
		if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 || rate > 1 {
			return ObjectiveEvaluation{}, fmt.Errorf(
				"minimum success rate must be finite and between 0 and 1",
			)
		}
	}

	if objectives.MaxAverageLatency != nil {
		latency := *objectives.MaxAverageLatency
		if math.IsNaN(latency) || math.IsInf(latency, 0) || latency < 0 || latency > MaxScenarioTick {
			return ObjectiveEvaluation{}, fmt.Errorf(
				"maximum average latency must be finite and between 0 and %d",
				MaxScenarioTick,
			)
		}
	}

	if objectives.MinSuccessRate == nil &&
		objectives.MaxAverageLatency == nil {
		return ObjectiveEvaluation{}, fmt.Errorf(
			"scenario objectives must define at least one criterion",
		)
	}

	evaluation := ObjectiveEvaluation{
		Configured: true,
		Passed:     true,
		Outcomes:   make([]ObjectiveOutcome, 0, 2),
	}

	if objectives.MinSuccessRate != nil {
		actual := 0.0
		if result.Summary.TotalRequests > 0 {
			actual = float64(result.Summary.SucceededRequests) /
				float64(result.Summary.TotalRequests)
		}

		threshold := *objectives.MinSuccessRate
		passed := actual >= threshold

		evaluation.Outcomes = append(evaluation.Outcomes, ObjectiveOutcome{
			Name:      "minimum_success_rate",
			Passed:    passed,
			Actual:    actual,
			Threshold: threshold,
			Message: fmt.Sprintf(
				"Success rate %.1f%%; minimum required %.1f%%",
				actual*100,
				threshold*100,
			),
		})

		evaluation.Passed = evaluation.Passed && passed
	}

	if objectives.MaxAverageLatency != nil {
		threshold := *objectives.MaxAverageLatency
		latency := result.Summary.EndToEndLatency
		passed := latency.Count > 0 && latency.Average <= threshold

		message := fmt.Sprintf(
			"Average end-to-end latency %.2f ticks; maximum allowed %.2f ticks",
			latency.Average,
			threshold,
		)
		if latency.Count == 0 {
			message = "No successful requests; average latency cannot be evaluated"
		}

		evaluation.Outcomes = append(evaluation.Outcomes, ObjectiveOutcome{
			Name:      "maximum_average_latency",
			Passed:    passed,
			Actual:    latency.Average,
			Threshold: threshold,
			Message:   message,
		})

		evaluation.Passed = evaluation.Passed && passed
	}

	return evaluation, nil
}
