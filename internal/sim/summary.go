package sim

import "math"

type LatencyStats struct {
	Count   int
	Min     Tick
	Max     Tick
	Average float64
}

type RunSummary struct {
	TotalRequests        int
	CreatedRequests      int
	QueuedRequests       int
	RetryWaitingRequests int
	InFlightRequests     int
	SucceededRequests    int
	FailedRequests       int
	CancelledRequests    int
	TimedOutRequests     int

	CompletedRequests int
	Retries           int

	EndToEndLatency LatencyStats
	ServiceLatency  LatencyStats
}

// Summary returns deterministic outcome counts and latency statistics
// derived from the engine's request registry without creating snapshots.
func (e *Engine) Summary() RunSummary {
	summary := RunSummary{
		TotalRequests: len(e.requests),
	}

	var endToEndTotal int64
	var serviceTotal int64

	summary.EndToEndLatency.Min = Tick(math.MaxInt64)
	summary.ServiceLatency.Min = Tick(math.MaxInt64)

	for _, request := range e.requests {
		switch request.State() {
		case RequestCreated:
			summary.CreatedRequests++
		case RequestQueued:
			summary.QueuedRequests++
		case RequestInFlight:
			summary.InFlightRequests++
		case RequestRetryWaiting:
			summary.RetryWaitingRequests++
		case RequestSucceeded:
			summary.SucceededRequests++
		case RequestFailed:
			summary.FailedRequests++
		case RequestCancelled:
			summary.CancelledRequests++
		case RequestTimedOut:
			summary.TimedOutRequests++
		}

		if request.Attempts() > 1 {
			summary.Retries += request.Attempts() - 1
		}

		// Only successful requests contribute to completion latency statistics.
		// Other terminal states have a finish tick but did not complete successfully.
		if request.State() != RequestSucceeded {
			continue
		}

		finishedAt, finished := request.FinishedAt()
		if !finished {
			continue
		}

		createdAt := request.CreatedAt()
		endToEnd := finishedAt - createdAt

		summary.CompletedRequests++
		addLatency(&summary.EndToEndLatency, endToEnd)
		endToEndTotal += int64(endToEnd)

		service := request.ServiceTime()
		addLatency(&summary.ServiceLatency, service)
		serviceTotal += int64(service)
	}

	if summary.EndToEndLatency.Count > 0 {
		summary.EndToEndLatency.Average =
			float64(endToEndTotal) / float64(summary.EndToEndLatency.Count)
	} else {
		summary.EndToEndLatency.Min = 0
	}

	if summary.ServiceLatency.Count > 0 {
		summary.ServiceLatency.Average =
			float64(serviceTotal) / float64(summary.ServiceLatency.Count)
	} else {
		summary.ServiceLatency.Min = 0
	}

	return summary
}

func addLatency(stats *LatencyStats, latency Tick) {
	if stats.Count == 0 || latency < stats.Min {
		stats.Min = latency
	}
	if stats.Count == 0 || latency > stats.Max {
		stats.Max = latency
	}
	stats.Count++
}
