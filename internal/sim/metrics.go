package sim

// WorkerMetrics is a read-only snapshot of one worker's capacity and queue.
type WorkerMetrics struct {
	Capacity        int
	BusyWorkers     int
	WaitingRequests int
	Utilization     float64
	Online          bool
}

// Metrics is a read-only snapshot of the simulation's current state.
// Lifecycle counts are derived from the authoritative request registry.
type Metrics struct {
	TotalRequests     int
	CreatedRequests   int
	QueuedRequests    int
	InFlightRequests  int
	SucceededRequests int
	FailedRequests    int
	CancelledRequests int
	TimedOutRequests  int

	BusyWorkers       int
	ServiceCapacity   int
	WaitingRequests   int
	WorkerUtilization float64

	Workers map[string]WorkerMetrics
}

// Metrics returns a snapshot without mutating the engine or advancing time.
// For topology-aware engines, ServiceCapacity is the sum of effective
// capacities across worker nodes.
func (e *Engine) Metrics() Metrics {
	metrics := Metrics{
		TotalRequests: len(e.requests),
		BusyWorkers:   e.busyWorkers,
		Workers:       make(map[string]WorkerMetrics),
	}

	for _, request := range e.requests {
		switch request.State() {
		case RequestCreated:
			metrics.CreatedRequests++
		case RequestQueued:
			metrics.QueuedRequests++
		case RequestInFlight:
			metrics.InFlightRequests++
		case RequestSucceeded:
			metrics.SucceededRequests++
		case RequestFailed:
			metrics.FailedRequests++
		case RequestCancelled:
			metrics.CancelledRequests++
		case RequestTimedOut:
			metrics.TimedOutRequests++
		}
	}

	if e.router == nil {
		metrics.ServiceCapacity = e.serviceCapacity
		metrics.WaitingRequests = len(e.waiting)
	} else {
		for workerID, node := range e.router.nodes {
			if node.Type != NodeWorker {
				continue
			}

			capacity := node.Capacity
			if capacity == 0 {
				capacity = e.serviceCapacity
			}

			busy := e.workerBusy[workerID]
			waiting := len(e.workerWaiting[workerID])

			worker := WorkerMetrics{
				Capacity:        capacity,
				BusyWorkers:     busy,
				WaitingRequests: waiting,
				Online:          e.workerOnline[workerID],
			}
			if capacity > 0 {
				worker.Utilization = float64(busy) / float64(capacity)
			}

			metrics.Workers[workerID] = worker
			metrics.ServiceCapacity += capacity
			metrics.WaitingRequests += waiting
		}
	}

	if metrics.ServiceCapacity > 0 {
		metrics.WorkerUtilization = float64(metrics.BusyWorkers) /
			float64(metrics.ServiceCapacity)
	}

	return metrics
}
