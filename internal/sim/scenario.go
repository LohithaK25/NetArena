package sim

import (
	"context"
	"fmt"
	"strings"
)

const (
	// MaxScenarioArrivals is the maximum number of requests in one scenario.
	MaxScenarioArrivals = 10_000

	// MaxScenarioEvents is the maximum number of explicitly scheduled events
	// across worker failures, recoveries, timeouts, and cancellations.
	MaxScenarioEvents = 30_000

	// MaxScenarioTopologyNodes is the maximum number of nodes in a topology.
	MaxScenarioTopologyNodes = 1_000

	// MaxScenarioTopologyConnections is the maximum number of connections in a topology.
	MaxScenarioTopologyConnections = 5_000

	// MaxScenarioServiceCapacity is the maximum service capacity.
	MaxScenarioServiceCapacity = 10_000

	// MaxScenarioTick is the maximum permitted service duration or scheduled tick.
	MaxScenarioTick = 1_000_000_000
)

// ScenarioArrival defines a request arrival in a simulation scenario.
type ScenarioArrival struct {
	RequestID  string
	SourceNode string
	At         Tick
	Priority   int
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
	Name             string
	Seed             int64
	ServiceCapacity  int
	ServiceDuration  Tick
	SchedulingPolicy SchedulingPolicy
	RoutingAlgorithm RoutingAlgorithm
	Topology         Topology

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
	Name        string
	Duration    Tick
	Summary     RunSummary
	Metrics     Metrics
	RouteTraces map[string][]RouteTraceHop
}

// Validate checks the scenario configuration before execution.
func (s Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("scenario name must not be empty")
	}
	if s.SchedulingPolicy != "" {
		if err := s.SchedulingPolicy.Validate(); err != nil {
			return fmt.Errorf("invalid scheduling policy: %w", err)
		}
	}
	if s.RoutingAlgorithm != "" {
		if err := s.RoutingAlgorithm.Validate(); err != nil {
			return fmt.Errorf("invalid routing algorithm: %w", err)
		}
	}
	if s.ServiceCapacity > MaxScenarioServiceCapacity {
		return fmt.Errorf(
			"service capacity %d exceeds maximum %d",
			s.ServiceCapacity,
			MaxScenarioServiceCapacity,
		)
	}
	if s.ServiceDuration > MaxScenarioTick {
		return fmt.Errorf(
			"service duration %d exceeds maximum tick %d",
			s.ServiceDuration,
			MaxScenarioTick,
		)
	}
	if len(s.Arrivals) > MaxScenarioArrivals {
		return fmt.Errorf(
			"arrival count %d exceeds maximum %d",
			len(s.Arrivals),
			MaxScenarioArrivals,
		)
	}
	if len(s.WorkerFailures)+len(s.WorkerRecoveries)+
		len(s.Timeouts)+len(s.Cancellations) > MaxScenarioEvents {
		return fmt.Errorf(
			"scheduled event count exceeds maximum %d",
			MaxScenarioEvents,
		)
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
		if arrival.At > MaxScenarioTick {
			return fmt.Errorf(
				"arrival %q tick %d exceeds maximum %d",
				arrival.RequestID,
				arrival.At,
				MaxScenarioTick,
			)
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
	if s.RoutingAlgorithm == RoutingShortestPath && !topologyConfigured {
		return fmt.Errorf("shortest-path routing requires a topology")
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
	if event.At > MaxScenarioTick {
		return fmt.Errorf("worker event tick exceeds maximum %d", MaxScenarioTick)
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
	if at > MaxScenarioTick {
		return fmt.Errorf("%s tick exceeds maximum %d", kind, MaxScenarioTick)
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
	return RunScenarioContext(context.Background(), s)
}

// RunScenarioContext validates and executes a scenario until completion or context cancellation.
func RunScenarioContext(ctx context.Context, s Scenario) (ScenarioResult, error) {
	if ctx == nil {
		return ScenarioResult{}, fmt.Errorf("simulation context must not be nil")
	}
	if err := ctx.Err(); err != nil {
		return ScenarioResult{}, err
	}
	if err := s.Validate(); err != nil {
		return ScenarioResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ScenarioResult{}, err
	}

	return runValidatedScenario(ctx, s)
}

// runValidatedScenario executes a scenario that has already passed validation.
func runValidatedScenario(ctx context.Context, s Scenario) (ScenarioResult, error) {
	var (
		engine *Engine
		err    error
	)

	if len(s.Topology.Nodes) > 0 || len(s.Topology.Connections) > 0 {
		engine, err = NewEngineWithTopologyAndSeed(
			s.ServiceCapacity,
			s.ServiceDuration,
			s.Topology,
			s.Seed,
		)
	} else {
		engine, err = NewEngineWithSeed(
			s.ServiceCapacity,
			s.ServiceDuration,
			s.Seed,
		)
	}
	if err != nil {
		return ScenarioResult{}, err
	}

	initialEventCapacity := len(s.Arrivals) +
		len(s.WorkerFailures) +
		len(s.WorkerRecoveries) +
		len(s.Timeouts) +
		len(s.Cancellations)

	engine.queue = newEventQueueWithCapacity(initialEventCapacity)

	if err := engine.SetRetryPolicy(s.RetryPolicy); err != nil {
		return ScenarioResult{}, err
	}
	schedulingPolicy := s.SchedulingPolicy
	if schedulingPolicy == "" {
		schedulingPolicy = SchedulingFIFO
	}
	if err := engine.SetSchedulingPolicy(schedulingPolicy); err != nil {
		return ScenarioResult{}, err
	}
	if len(s.Topology.Nodes) > 0 || len(s.Topology.Connections) > 0 {
		routingAlgorithm := s.RoutingAlgorithm
		if routingAlgorithm == "" {
			routingAlgorithm = RoutingRoundRobin
		}

		if err := engine.SetRoutingAlgorithm(routingAlgorithm); err != nil {
			return ScenarioResult{}, err
		}
	}
	if s.CircuitBreaker != nil {
		if err := engine.SetCircuitBreakerPolicy(*s.CircuitBreaker); err != nil {
			return ScenarioResult{}, err
		}
	}

	for _, arrival := range s.Arrivals {
		var err error

		if arrival.SourceNode != "" {
			err = engine.ScheduleRoutedArrivalWithPriority(
				arrival.RequestID,
				arrival.SourceNode,
				arrival.At,
				arrival.Priority,
			)
		} else {
			err = engine.ScheduleArrivalWithPriority(
				arrival.RequestID,
				arrival.At,
				arrival.Priority,
			)
		}

		if err != nil {
			return ScenarioResult{}, fmt.Errorf(
				"schedule arrival %q: %w",
				arrival.RequestID,
				err,
			)
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

	if err := engine.RunContext(ctx); err != nil {
		return ScenarioResult{}, fmt.Errorf("run scenario %q: %w", s.Name, err)
	}

	return ScenarioResult{
		Name:        s.Name,
		Duration:    engine.CurrentTick(),
		Summary:     engine.Summary(),
		Metrics:     engine.Metrics(),
		RouteTraces: engine.RouteTraces(),
	}, nil
}
