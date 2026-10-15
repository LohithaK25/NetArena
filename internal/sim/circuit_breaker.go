package sim

import (
	"fmt"
	"math"
)

// CircuitState describes a worker's circuit-breaker state.
type CircuitState uint8

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// String returns the readable circuit-breaker state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// CircuitBreakerPolicy configures when a worker circuit opens and
// how long it remains open before allowing a recovery probe.
type CircuitBreakerPolicy struct {
	FailureThreshold int
	OpenDuration     Tick
}

// Validate checks whether the circuit-breaker policy is valid.
func (p CircuitBreakerPolicy) Validate() error {
	if p.FailureThreshold <= 0 {
		return fmt.Errorf("failure threshold must be positive")
	}
	if p.OpenDuration <= 0 {
		return fmt.Errorf("open duration must be positive")
	}
	return nil
}

type circuitBreaker struct {
	state               CircuitState
	consecutiveFailures int
	openUntil           Tick
	probeRequestID      string
}

func (e *Engine) circuitAllowsRoute(workerID string) bool {
	if !e.circuitBreakerEnabled {
		return true
	}

	breaker, exists := e.circuitBreakers[workerID]
	if !exists {
		return true
	}

	switch breaker.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		return e.workerOnline[workerID] && e.now >= breaker.openUntil
	case CircuitHalfOpen:
		return breaker.probeRequestID == ""
	default:
		return false
	}
}

// acquireCircuitProbe reserves the half-open probe for a request that
// has reached the worker and is ready to start service.
func (e *Engine) acquireCircuitProbe(workerID, requestID string) bool {
	if !e.circuitBreakerEnabled {
		return true
	}

	breaker, exists := e.circuitBreakers[workerID]
	if !exists {
		return true
	}

	switch breaker.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if !e.workerOnline[workerID] || e.now < breaker.openUntil {
			return false
		}
		breaker.state = CircuitHalfOpen
		breaker.probeRequestID = requestID
		return true
	case CircuitHalfOpen:
		return breaker.probeRequestID == requestID
	default:
		return false
	}
}

func (e *Engine) recordCircuitSuccess(workerID, requestID string) {
	if !e.circuitBreakerEnabled {
		return
	}

	breaker, exists := e.circuitBreakers[workerID]
	if !exists {
		return
	}

	if breaker.state == CircuitHalfOpen && breaker.probeRequestID == requestID {
		breaker.state = CircuitClosed
		breaker.consecutiveFailures = 0
		breaker.openUntil = 0
		breaker.probeRequestID = ""
		return
	}

	if breaker.state == CircuitClosed {
		breaker.consecutiveFailures = 0
	}
}

func (e *Engine) recordCircuitFailure(workerID, requestID string) error {
	if !e.circuitBreakerEnabled {
		return nil
	}

	breaker, exists := e.circuitBreakers[workerID]
	if !exists {
		return nil
	}

	if breaker.state == CircuitHalfOpen && breaker.probeRequestID == requestID {
		return e.openCircuit(workerID, breaker)
	}

	if breaker.state == CircuitClosed {
		breaker.consecutiveFailures++
		if breaker.consecutiveFailures >= e.circuitBreakerPolicy.FailureThreshold {
			return e.openCircuit(workerID, breaker)
		}
	}

	return nil
}

func (e *Engine) openCircuit(workerID string, breaker *circuitBreaker) error {
	if e.now > Tick(math.MaxInt64)-e.circuitBreakerPolicy.OpenDuration {
		return fmt.Errorf("circuit-breaker cooldown tick overflow for worker %q", workerID)
	}

	openUntil := e.now + e.circuitBreakerPolicy.OpenDuration

	if err := e.schedule(Event{
		At:     openUntil,
		Type:   EventCircuitCooldownElapsed,
		NodeID: workerID,
	}); err != nil {
		return err
	}

	breaker.state = CircuitOpen
	breaker.probeRequestID = ""
	breaker.openUntil = openUntil
	return nil
}

func (e *Engine) handleCircuitCooldownElapsed(workerID string) error {
	if !e.circuitBreakerEnabled {
		return nil
	}

	breaker, exists := e.circuitBreakers[workerID]
	if !exists {
		return fmt.Errorf("circuit cooldown references unknown worker %q", workerID)
	}

	// Ignore stale cooldown events.
	if breaker.state != CircuitOpen || e.now < breaker.openUntil {
		return nil
	}

	if !e.workerOnline[workerID] {
		return nil
	}

	// A new request may claim the probe through normal routing.
	// If work is already queued, resume the oldest request as the probe.
	if len(e.workerWaiting[workerID]) == 0 {
		return nil
	}

	capacity, err := e.workerCapacity(workerID)
	if err != nil {
		return err
	}
	if e.workerBusy[workerID] >= capacity {
		return nil
	}

	waiting := e.workerWaiting[workerID]
	requestID := waiting[0]
	request, found := e.requests[requestID]
	if !found {
		return fmt.Errorf(
			"worker %q waiting queue references unknown request %q",
			workerID,
			requestID,
		)
	}

	switch request.State() {
	case RequestSucceeded, RequestFailed, RequestCancelled, RequestTimedOut:
		e.workerWaiting[workerID] = waiting[1:]
		return e.handleCircuitCooldownElapsed(workerID)
	case RequestRetryWaiting:
		if err := request.Transition(RequestQueued, e.now, ""); err != nil {
			return err
		}
	case RequestQueued:
		// Ready to start the probe.
	default:
		return fmt.Errorf(
			"worker %q waiting request %q is not queued",
			workerID,
			requestID,
		)
	}

	if !e.acquireCircuitProbe(workerID, requestID) {
		return nil
	}

	if err := e.startWorkerService(request, workerID); err != nil {
		breaker.state = CircuitOpen
		breaker.probeRequestID = ""
		return err
	}

	e.workerWaiting[workerID] = waiting[1:]
	return nil
}
