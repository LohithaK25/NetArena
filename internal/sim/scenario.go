package sim

import (
	"fmt"
	"strings"
)

// ScenarioArrival defines a request arrival in a simulation scenario.
type ScenarioArrival struct {
	RequestID  string
	SourceNode string
	At         Tick
}

// ScenarioWorkerEvent defines a scheduled worker failure or recovery.
type ScenarioWorkerEvent struct {
	WorkerID string
	At       Tick
}

// ScenarioTimeout defines a request deadline.
type ScenarioTimeout struct {
	RequestID string
	At        Tick
}

// ScenarioCancellation defines a scheduled request cancellation.
type ScenarioCancellation struct {
	RequestID string
	At        Tick
}

// Scenario describes a deterministic simulation run.
type Scenario struct {
	Name            string
	ServiceCapacity int
	ServiceDuration Tick
	Topology        Topology

	Arrivals         []ScenarioArrival
	WorkerFailures   []ScenarioWorkerEvent
	WorkerRecoveries []ScenarioWorkerEvent
	Timeouts         []ScenarioTimeout
	Cancellations    []ScenarioCancellation

	RetryPolicy    RetryPolicy
	CircuitBreaker *CircuitBreakerPolicy
}

// ScenarioResult contains the outputs of one scenario execution.
type ScenarioResult struct {
	Name     string
	Duration Tick
	Summary  RunSummary
	Metrics  Metrics
}

// Validate checks the scenario configuration before execution.
func (s Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("scenario name must not be empty")
	}
	if s.ServiceCapacity <= 0 {
		return fmt.Errorf("service capacity must be positive")
	}
	if s.ServiceDuration <= 0 {
		return fmt.Errorf("service duration must be positive")
	}
	if len(s.Arrivals) == 0 {
		return fmt.Errorf("scenario must define at least one arrival")
	}

	seen := make(map[string]struct{}, len(s.Arrivals))
	for _, arrival := range s.Arrivals {
		if strings.TrimSpace(arrival.RequestID) == "" {
			return fmt.Errorf("arrival request ID must not be empty")
		}
		if arrival.At < 0 {
			return fmt.Errorf("arrival %q has a negative tick", arrival.RequestID)
		}
		if _, exists := seen[arrival.RequestID]; exists {
			return fmt.Errorf("duplicate arrival request ID %q", arrival.RequestID)
		}
		seen[arrival.RequestID] = struct{}{}
	}

	topologyConfigured := len(s.Topology.Nodes) > 0 || len(s.Topology.Connections) > 0
	if topologyConfigured {
		if err := s.Topology.Validate(); err != nil {
			return fmt.Errorf("invalid topology: %w", err)
		}
		for _, arrival := range s.Arrivals {
			if strings.TrimSpace(arrival.SourceNode) == "" {
				return fmt.Errorf("arrival %q requires a source node for topology scenarios", arrival.RequestID)
			}
			if !containsNode(s.Topology, arrival.SourceNode) {
				return fmt.Errorf("arrival %q references unknown source node %q", arrival.RequestID, arrival.SourceNode)
			}
		}
	} else {
		for _, arrival := range s.Arrivals {
			if arrival.SourceNode != "" {
				return fmt.Errorf("arrival %q specifies a source node without a topology", arrival.RequestID)
			}
		}
	}

	if err := s.RetryPolicy.Validate(); err != nil {
		return fmt.Errorf("invalid retry policy: %w", err)
	}
	if s.CircuitBreaker != nil {
		if err := s.CircuitBreaker.Validate(); err != nil {
			return fmt.Errorf("invalid circuit-breaker policy: %w", err)
		}
		if !topologyConfigured {
			return fmt.Errorf("circuit breakers require a topology")
		}
	}

	for _, event := range s.WorkerFailures {
		if err := s.validateWorkerEvent(event); err != nil {
			return err
		}
	}
	for _, event := range s.WorkerRecoveries {
		if err := s.validateWorkerEvent(event); err != nil {
			return err
		}
	}

	for _, timeout := range s.Timeouts {
		if err := s.validateRequestEvent(timeout.RequestID, timeout.At, "timeout"); err != nil {
			return err
		}
	}
	for _, cancellation := range s.Cancellations {
		if err := s.validateRequestEvent(cancellation.RequestID, cancellation.At, "cancellation"); err != nil {
			return err
		}
	}

	return nil
}

func (s Scenario) validateWorkerEvent(event ScenarioWorkerEvent) error {
	if event.At < 0 {
		return fmt.Errorf("worker event tick must not be negative")
	}
	if strings.TrimSpace(event.WorkerID) == "" {
		return fmt.Errorf("worker event ID must not be empty")
	}
	if len(s.Topology.Nodes) == 0 {
		return fmt.Errorf("worker events require a topology")
	}
	for _, node := range s.Topology.Nodes {
		if node.ID == event.WorkerID {
			if node.Type != NodeWorker {
				return fmt.Errorf("node %q is not a worker", event.WorkerID)
			}
			return nil
		}
	}
	return fmt.Errorf("unknown worker %q", event.WorkerID)
}

func (s Scenario) validateRequestEvent(requestID string, at Tick, kind string) error {
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("%s request ID must not be empty", kind)
	}
	if at < 0 {
		return fmt.Errorf("%s tick must not be negative", kind)
	}

	for _, arrival := range s.Arrivals {
		if arrival.RequestID != requestID {
			continue
		}

		if at < arrival.At {
			return fmt.Errorf(
				"%s for request %q occurs at tick %d before its arrival at tick %d",
				kind,
				requestID,
				at,
				arrival.At,
			)
		}

		return nil
	}

	return fmt.Errorf("%s references unknown request %q", kind, requestID)
}

func containsNode(topology Topology, nodeID string) bool {
	for _, node := range topology.Nodes {
		if node.ID == nodeID {
			return true
		}
	}
	return false
}

// RunScenario validates, configures, and executes a scenario deterministically.
func RunScenario(s Scenario) (ScenarioResult, error) {
	if err := s.Validate(); err != nil {
		return ScenarioResult{}, err
	}

	var (
		engine *Engine
		err    error
	)

	if len(s.Topology.Nodes) > 0 || len(s.Topology.Connections) > 0 {
		engine, err = NewEngineWithTopology(s.ServiceCapacity, s.ServiceDuration, s.Topology)
	} else {
		engine, err = NewEngine(s.ServiceCapacity, s.ServiceDuration)
	}
	if err != nil {
		return ScenarioResult{}, err
	}

	if err := engine.SetRetryPolicy(s.RetryPolicy); err != nil {
		return ScenarioResult{}, err
	}
	if s.CircuitBreaker != nil {
		if err := engine.SetCircuitBreakerPolicy(*s.CircuitBreaker); err != nil {
			return ScenarioResult{}, err
		}
	}

	for _, arrival := range s.Arrivals {
		if arrival.SourceNode != "" {
			err = engine.ScheduleRoutedArrival(arrival.RequestID, arrival.SourceNode, arrival.At)
		} else {
			err = engine.ScheduleArrival(arrival.RequestID, arrival.At)
		}
		if err != nil {
			return ScenarioResult{}, fmt.Errorf("schedule arrival %q: %w", arrival.RequestID, err)
		}
	}

	for _, event := range s.WorkerFailures {
		if err := engine.ScheduleWorkerFailure(event.WorkerID, event.At); err != nil {
			return ScenarioResult{}, fmt.Errorf("schedule worker failure: %w", err)
		}
	}
	for _, event := range s.WorkerRecoveries {
		if err := engine.ScheduleWorkerRecovery(event.WorkerID, event.At); err != nil {
			return ScenarioResult{}, fmt.Errorf("schedule worker recovery: %w", err)
		}
	}

	// Schedule request deadlines and cancellations after arrivals so request
	// records exist before the engine validates these events.
	for _, timeout := range s.Timeouts {
		if err := engine.ScheduleTimeout(timeout.RequestID, timeout.At); err != nil {
			return ScenarioResult{}, fmt.Errorf("schedule timeout: %w", err)
		}
	}
	for _, cancellation := range s.Cancellations {
		if err := engine.ScheduleCancel(cancellation.RequestID, cancellation.At); err != nil {
			return ScenarioResult{}, fmt.Errorf("schedule cancellation: %w", err)
		}
	}

	if err := engine.Run(); err != nil {
		return ScenarioResult{}, fmt.Errorf("run scenario %q: %w", s.Name, err)
	}

	return ScenarioResult{
		Name:     s.Name,
		Duration: engine.CurrentTick(),
		Summary:  engine.Summary(),
		Metrics:  engine.Metrics(),
	}, nil
}
