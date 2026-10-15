package sim

import (
	"fmt"
	"math"
	"strings"
)

// Engine runs a deterministic, discrete-event service simulation.
type Engine struct {
	queue                 *EventQueue
	requests              map[string]*Request
	now                   Tick
	nextSequence          uint64
	serviceCapacity       int
	serviceDuration       Tick
	busyWorkers           int
	waiting               []string
	router                *Router
	requestNodes          map[string]string
	requestSources        map[string]string
	visitedNodes          map[string]map[string]bool
	workerBusy            map[string]int
	workerWaiting         map[string][]string
	workerOnline          map[string]bool
	retryPolicy           RetryPolicy
	retryCounts           map[string]int
	attemptIDs            map[string]uint64
	retryWorkers          map[string]string
	circuitBreakers       map[string]*circuitBreaker
	circuitBreakerPolicy  CircuitBreakerPolicy
	circuitBreakerEnabled bool
}

// NewEngine creates an engine with a fixed service capacity and duration.
func NewEngine(serviceCapacity int, serviceDuration Tick) (*Engine, error) {
	if serviceCapacity <= 0 {
		return nil, fmt.Errorf("service capacity must be positive")
	}
	if serviceDuration <= 0 {
		return nil, fmt.Errorf("service duration must be positive")
	}

	return &Engine{
		queue:           NewEventQueue(),
		requests:        make(map[string]*Request),
		serviceCapacity: serviceCapacity,
		serviceDuration: serviceDuration,
		requestNodes:    make(map[string]string),
		requestSources:  make(map[string]string),
		visitedNodes:    make(map[string]map[string]bool),
		workerBusy:      make(map[string]int),
		workerWaiting:   make(map[string][]string),
		workerOnline:    make(map[string]bool),
		retryCounts:     make(map[string]int),
		attemptIDs:      make(map[string]uint64),
		retryWorkers:    make(map[string]string),
		circuitBreakers: make(map[string]*circuitBreaker),
	}, nil
}

// NewEngineWithTopology creates an engine configured for topology-aware routing.
func NewEngineWithTopology(
	serviceCapacity int,
	serviceDuration Tick,
	topology Topology,
) (*Engine, error) {
	if err := topology.Validate(); err != nil {
		return nil, fmt.Errorf("invalid topology: %w", err)
	}

	engine, err := NewEngine(serviceCapacity, serviceDuration)
	if err != nil {
		return nil, err
	}

	router, err := NewRouter(topology)
	if err != nil {
		return nil, err
	}

	engine.router = router

	for _, node := range topology.Nodes {
		if node.Type == NodeWorker {
			engine.workerOnline[node.ID] = true
			engine.circuitBreakers[node.ID] = &circuitBreaker{
				state: CircuitClosed,
			}
		}
	}

	return engine, nil
}

// ScheduleArrival schedules a request to arrive at a virtual tick.
func (e *Engine) ScheduleArrival(requestID string, at Tick) error {
	return e.scheduleArrival(requestID, at, "")
}

func (e *Engine) scheduleArrival(requestID string, at Tick, sourceNode string) error {
	if e.router != nil && sourceNode == "" {
		return fmt.Errorf("topology-aware engine requires a source node")
	}
	if e.router == nil && sourceNode != "" {
		return fmt.Errorf("cannot specify a source node without a topology")
	}
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("request ID must not be empty")
	}
	if at < e.now {
		return fmt.Errorf("arrival tick %d precedes current tick %d", at, e.now)
	}
	if _, exists := e.requests[requestID]; exists {
		return fmt.Errorf("request %q already exists", requestID)
	}
	if e.nextSequence == math.MaxUint64 {
		return fmt.Errorf("event sequence exhausted")
	}

	request, err := NewRequest(requestID, at)
	if err != nil {
		return err
	}

	if err := e.schedule(Event{
		At:        at,
		Type:      EventRequestArrives,
		RequestID: requestID,
	}); err != nil {
		return err
	}

	e.requests[requestID] = request

	if sourceNode != "" {
		e.requestSources[requestID] = sourceNode
		e.requestNodes[requestID] = sourceNode
		e.visitedNodes[requestID] = map[string]bool{sourceNode: true}
	}

	return nil
}

// ScheduleRoutedArrival schedules a request that begins at a topology node.
func (e *Engine) ScheduleRoutedArrival(requestID, sourceNode string, at Tick) error {
	if e.router == nil {
		return fmt.Errorf("engine has no topology configured")
	}
	if _, exists := e.router.nodes[sourceNode]; !exists {
		return fmt.Errorf("unknown source node %q", sourceNode)
	}

	return e.scheduleArrival(requestID, at, sourceNode)
}

// ScheduleWorkerFailure schedules a worker to become unavailable at a virtual tick.
func (e *Engine) ScheduleWorkerFailure(workerID string, at Tick) error {
	return e.scheduleWorkerEvent(workerID, at, EventWorkerFails)
}

// ScheduleWorkerRecovery schedules a worker to become available at a virtual tick.
func (e *Engine) ScheduleWorkerRecovery(workerID string, at Tick) error {
	return e.scheduleWorkerEvent(workerID, at, EventWorkerRecovers)
}

func (e *Engine) scheduleWorkerEvent(workerID string, at Tick, eventType EventType) error {
	if e.router == nil {
		return fmt.Errorf("engine has no topology configured")
	}

	if strings.TrimSpace(workerID) == "" {
		return fmt.Errorf("worker ID must not be empty")
	}

	node, exists := e.router.nodes[workerID]
	if !exists {
		return fmt.Errorf("unknown worker %q", workerID)
	}
	if node.Type != NodeWorker {
		return fmt.Errorf("node %q is not a worker", workerID)
	}

	if at < e.now {
		return fmt.Errorf(
			"worker event tick %d precedes current tick %d",
			at,
			e.now,
		)
	}

	return e.schedule(Event{
		At:     at,
		Type:   eventType,
		NodeID: workerID,
	})
}

// ScheduleTimeout schedules an absolute deadline for an existing request.
func (e *Engine) ScheduleTimeout(requestID string, deadline Tick) error {
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("request ID must not be empty")
	}

	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("cannot schedule timeout for unknown request %q", requestID)
	}

	if request.IsTerminal() {
		return fmt.Errorf("cannot schedule timeout for terminal request %q", requestID)
	}

	if deadline < e.now {
		return fmt.Errorf(
			"timeout deadline %d precedes current tick %d",
			deadline,
			e.now,
		)
	}

	return e.schedule(Event{
		At:        deadline,
		Type:      EventRequestTimeout,
		RequestID: requestID,
	})
}

// ScheduleCancel schedules cancellation of a non-terminal request.
func (e *Engine) ScheduleCancel(requestID string, at Tick) error {
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("request ID must not be empty")
	}

	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("cannot schedule cancellation for unknown request %q", requestID)
	}
	if request.IsTerminal() {
		return fmt.Errorf("cannot schedule cancellation for terminal request %q", requestID)
	}
	if at < e.now {
		return fmt.Errorf(
			"cancellation tick %d precedes current tick %d",
			at,
			e.now,
		)
	}

	return e.schedule(Event{
		At:        at,
		Type:      EventRequestCancel,
		RequestID: requestID,
	})
}

// Run processes every currently scheduled event.
func (e *Engine) Run() error {
	for e.queue.Len() > 0 {
		event, ok := e.queue.Pop()
		if !ok {
			return fmt.Errorf("event queue unexpectedly empty")
		}
		if event.At < e.now {
			return fmt.Errorf("event tick %d precedes current tick %d", event.At, e.now)
		}

		e.now = event.At

		switch event.Type {
		case EventRouteHop:
			if err := e.handleRouteHop(event.RequestID, event.NodeID); err != nil {
				return err
			}
		case EventRequestArrives:
			if err := e.handleArrival(event.RequestID); err != nil {
				return err
			}
		case EventServiceCompletes:
			if err := e.handleCompletion(event.RequestID, event.NodeID, event.AttemptID); err != nil {
				return err
			}
		case EventRequestTimeout:
			if err := e.handleTimeout(event.RequestID); err != nil {
				return err
			}
		case EventRequestCancel:
			if err := e.handleCancel(event.RequestID); err != nil {
				return err
			}
		case EventWorkerFails:
			if err := e.handleWorkerFailure(event.NodeID); err != nil {
				return err
			}
		case EventWorkerRecovers:
			if err := e.handleWorkerRecovery(event.NodeID); err != nil {
				return err
			}

		case EventCircuitCooldownElapsed:
			if err := e.handleCircuitCooldownElapsed(event.NodeID); err != nil {
				return err
			}

		case EventRequestRetry:
			if err := e.handleRequestRetry(event.RequestID); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported event type: %s", event.Type)
		}
	}

	return nil
}

// Requests returns independent snapshots of the request registry.
// Mutating a returned Request does not change engine-owned state.
func (e *Engine) Requests() map[string]*Request {
	snapshot := make(map[string]*Request, len(e.requests))
	for id, request := range e.requests {
		snapshot[id] = request.clone()
	}
	return snapshot
}

// CurrentTick returns the current virtual time.
func (e *Engine) CurrentTick() Tick {
	return e.now
}

// BusyWorkers returns the number of workers currently processing requests.
func (e *Engine) BusyWorkers() int {
	return e.busyWorkers
}

func (e *Engine) schedule(event Event) error {
	if e.nextSequence == math.MaxUint64 {
		return fmt.Errorf("event sequence exhausted")
	}

	nextSequence := e.nextSequence + 1
	event.Sequence = nextSequence

	if err := e.queue.Push(event); err != nil {
		return err
	}

	e.nextSequence = nextSequence
	return nil
}

func (e *Engine) handleArrival(requestID string) error {
	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("arrival references unknown request %q", requestID)
	}

	if request.State() != RequestCreated {
		switch request.State() {
		case RequestSucceeded, RequestFailed, RequestCancelled, RequestTimedOut:
			return nil
		default:
			return fmt.Errorf(
				"arrival for request %q requires created state, got %s",
				requestID,
				request.State(),
			)
		}
	}

	if err := request.Transition(RequestQueued, e.now, ""); err != nil {
		return err
	}

	if e.router != nil {
		sourceNode, exists := e.requestNodes[requestID]
		if !exists {
			return fmt.Errorf("routed request %q has no source node", requestID)
		}
		return e.schedule(Event{
			At:        e.now,
			Type:      EventRouteHop,
			RequestID: requestID,
			NodeID:    sourceNode,
		})
	}

	if e.busyWorkers < e.serviceCapacity {
		return e.startService(request)
	}

	e.waiting = append(e.waiting, requestID)
	return nil
}

func (e *Engine) handleRouteHop(requestID, nodeID string) error {
	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("route hop references unknown request %q", requestID)
	}
	switch request.State() {
	case RequestSucceeded, RequestFailed, RequestCancelled, RequestTimedOut:
		return nil
	}
	if e.router == nil {
		return fmt.Errorf("route hop encountered without a configured router")
	}
	if request.State() != RequestQueued {
		return fmt.Errorf(
			"route hop for request %q requires queued state, got %s",
			requestID,
			request.State(),
		)
	}

	node, exists := e.router.nodes[nodeID]
	if !exists {
		return e.failQueuedRequest(request, fmt.Sprintf("unknown routing node %q", nodeID))
	}

	if node.Type == NodeWorker {
		if !e.workerOnline[nodeID] {
			e.workerWaiting[nodeID] = append(e.workerWaiting[nodeID], requestID)
			e.requestNodes[requestID] = nodeID
			return nil
		}

		capacity, err := e.workerCapacity(nodeID)
		if err != nil {
			return err
		}

		if e.workerBusy[nodeID] >= capacity {
			e.workerWaiting[nodeID] = append(e.workerWaiting[nodeID], requestID)
			e.requestNodes[requestID] = nodeID
			return nil
		}

		if !e.acquireCircuitProbe(nodeID, requestID) {
			return e.failQueuedRequest(
				request,
				fmt.Sprintf("worker %q circuit is not accepting requests", nodeID),
			)
		}

		return e.startWorkerService(request, nodeID)
	}

	next, latency, err := e.router.NextHopWithLatencyFiltered(
		nodeID,

		func(destination Node) bool {
			if destination.Type != NodeWorker {
				return true
			}
			return e.workerOnline[destination.ID] &&
				e.circuitAllowsRoute(destination.ID)
		},
	)
	if err != nil {
		return e.failQueuedRequest(request, err.Error())
	}

	visited := e.visitedNodes[requestID]
	if visited == nil {
		visited = make(map[string]bool)
		e.visitedNodes[requestID] = visited
	}
	if visited[next.ID] {
		return e.failQueuedRequest(
			request,
			fmt.Sprintf("routing cycle detected at node %q", next.ID),
		)
	}

	if latency > 0 && e.now > Tick(math.MaxInt64)-latency {
		return fmt.Errorf("route-hop tick overflow for request %q", requestID)
	}

	if err := e.schedule(Event{
		At:        e.now + latency,
		Type:      EventRouteHop,
		RequestID: requestID,
		NodeID:    next.ID,
	}); err != nil {
		return err
	}

	visited[next.ID] = true
	e.requestNodes[requestID] = next.ID

	return nil
}

func (e *Engine) failQueuedRequest(request *Request, reason string) error {
	if err := request.Transition(RequestFailed, e.now, reason); err != nil {
		return err
	}
	delete(e.requestNodes, request.ID())
	delete(e.visitedNodes, request.ID())
	return nil
}

func (e *Engine) startService(request *Request) error {
	if e.busyWorkers >= e.serviceCapacity {
		return fmt.Errorf("service capacity exceeded")
	}
	if e.now > Tick(math.MaxInt64)-e.serviceDuration {
		return fmt.Errorf("service completion tick overflow")
	}
	if e.nextSequence == math.MaxUint64 {
		return fmt.Errorf("event sequence exhausted")
	}
	if err := request.Transition(RequestInFlight, e.now, ""); err != nil {
		return err
	}
	e.attemptIDs[request.ID()]++
	attemptID := e.attemptIDs[request.ID()]

	e.busyWorkers++

	return e.schedule(Event{
		At:        e.now + e.serviceDuration,
		Type:      EventServiceCompletes,
		RequestID: request.ID(),
		AttemptID: attemptID,
	})
}

func (e *Engine) startWorkerService(request *Request, workerID string) error {
	capacity, err := e.workerCapacity(workerID)
	if err != nil {
		return err
	}
	if e.workerBusy[workerID] >= capacity {
		return fmt.Errorf("worker %q capacity exceeded", workerID)
	}
	if e.now > Tick(math.MaxInt64)-e.serviceDuration {
		return fmt.Errorf("service completion tick overflow")
	}
	if e.nextSequence == math.MaxUint64 {
		return fmt.Errorf("event sequence exhausted")
	}
	if err := request.Transition(RequestInFlight, e.now, ""); err != nil {
		return err
	}

	e.attemptIDs[request.ID()]++
	attemptID := e.attemptIDs[request.ID()]

	e.workerBusy[workerID]++
	e.busyWorkers++

	if err := e.schedule(Event{
		At:        e.now + e.serviceDuration,
		Type:      EventServiceCompletes,
		RequestID: request.ID(),
		NodeID:    workerID,
		AttemptID: attemptID,
	}); err != nil {
		e.workerBusy[workerID]--
		e.busyWorkers--
		return err
	}

	return nil
}

func (e *Engine) handleCompletion(requestID, workerID string, attemptID uint64) error {
	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("completion references unknown request %q", requestID)
	}
	if attemptID != e.attemptIDs[requestID] {
		return nil
	}
	switch request.State() {
	case RequestSucceeded, RequestFailed, RequestCancelled, RequestTimedOut:
		return nil
	}
	if e.busyWorkers <= 0 {
		return fmt.Errorf("completion encountered with no busy workers")
	}

	if e.router == nil {
		if err := request.Transition(RequestSucceeded, e.now, ""); err != nil {
			return err
		}

		e.busyWorkers--

		if len(e.waiting) > 0 {
			nextID := e.waiting[0]
			nextRequest, exists := e.requests[nextID]
			if !exists {
				return fmt.Errorf("waiting queue references unknown request %q", nextID)
			}
			if err := e.startService(nextRequest); err != nil {
				return err
			}
			e.waiting = e.waiting[1:]
		}
		return nil
	}

	recordedWorker := e.requestNodes[requestID]
	if workerID == "" || recordedWorker != workerID {
		return fmt.Errorf("completion worker mismatch for request %q", requestID)
	}
	if e.workerBusy[workerID] <= 0 {
		return fmt.Errorf("worker %q completed a request with no active work", workerID)
	}

	waiting := e.workerWaiting[workerID]
	var nextRequest *Request

	if len(waiting) > 0 {
		nextID := waiting[0]
		var found bool
		nextRequest, found = e.requests[nextID]
		if !found {
			return fmt.Errorf("worker %q waiting queue references unknown request %q",
				workerID, nextID)
		}
		if nextRequest.State() != RequestQueued {
			return fmt.Errorf("worker %q waiting request %q is not queued",
				workerID, nextID)
		}
	}

	// The current service has completed. Release its worker slot even
	// if the next request cannot be scheduled.
	if err := request.Transition(RequestSucceeded, e.now, ""); err != nil {
		return err
	}
	e.recordCircuitSuccess(workerID, requestID)
	e.workerBusy[workerID]--
	e.busyWorkers--

	delete(e.requestNodes, requestID)
	delete(e.visitedNodes, requestID)

	if nextRequest != nil {
		breaker := e.circuitBreakers[workerID]

		// Don't start queued work while the circuit is open or a
		// half-open recovery probe is already in flight.
		if e.circuitBreakerEnabled && breaker != nil {
			if breaker.state == CircuitOpen && e.now >= breaker.openUntil {
				return e.handleCircuitCooldownElapsed(workerID)
			}
			if breaker.state != CircuitClosed {
				return nil
			}
		}

		if err := e.startWorkerService(nextRequest, workerID); err != nil {
			return err
		}
		e.workerWaiting[workerID] = e.workerWaiting[workerID][1:]
	}

	return nil
}
func (e *Engine) handleTimeout(requestID string) error {
	return e.handleTerminalRequest(requestID, RequestTimedOut)
}

func (e *Engine) handleCancel(requestID string) error {
	return e.handleTerminalRequest(requestID, RequestCancelled)
}
func (e *Engine) handleTerminalRequest(requestID string, terminalState RequestState) error {
	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("terminal event references unknown request %q", requestID)
	}

	state := request.State()

	// Ignore stale timeout events for requests that are already terminal.
	switch state {
	case RequestSucceeded, RequestFailed, RequestCancelled, RequestTimedOut:
		return nil
	}

	workerID := ""
	if state == RequestInFlight {
		if e.router == nil {
			if e.busyWorkers <= 0 {
				return fmt.Errorf("terminal event encountered with no busy workers")
			}
		} else {
			workerID = e.requestNodes[requestID]
			if workerID == "" {
				return fmt.Errorf("in-flight request %q has no worker", requestID)
			}
			if e.workerBusy[workerID] <= 0 {
				return fmt.Errorf("worker %q timed out a request with no active work", workerID)
			}
		}
	}

	// Remove a queued request from whichever waiting queue contains it.
	if state == RequestQueued {
		if e.router == nil {
			e.waiting = removeRequestID(e.waiting, requestID)
		} else if workerID = e.requestNodes[requestID]; workerID != "" {
			e.workerWaiting[workerID] = removeRequestID(
				e.workerWaiting[workerID],
				requestID,
			)
		}
	}

	if err := request.Transition(terminalState, e.now, ""); err != nil {
		return err
	}

	// Release capacity if the request was actively being processed.
	if state == RequestInFlight {
		e.busyWorkers--

		if e.router == nil {
			if len(e.waiting) > 0 {
				nextID := e.waiting[0]
				nextRequest, found := e.requests[nextID]
				if !found {
					return fmt.Errorf("waiting queue references unknown request %q", nextID)
				}
				if err := e.startService(nextRequest); err != nil {
					return err
				}
				e.waiting = e.waiting[1:]
			}
		} else {
			e.workerBusy[workerID]--

			waiting := e.workerWaiting[workerID]
			if e.workerOnline[workerID] &&
				len(waiting) > 0 &&
				(!e.circuitBreakerEnabled ||
					e.circuitBreakers[workerID] == nil ||
					e.circuitBreakers[workerID].state == CircuitClosed) {
				nextID := waiting[0]
				nextRequest, found := e.requests[nextID]
				if !found {
					return fmt.Errorf(
						"worker %q waiting queue references unknown request %q",
						workerID,
						nextID,
					)
				}
				if nextRequest.State() != RequestQueued {
					return fmt.Errorf(
						"worker %q waiting request %q is not queued",
						workerID,
						nextID,
					)
				}
				if err := e.startWorkerService(nextRequest, workerID); err != nil {
					return err
				}
				e.workerWaiting[workerID] = waiting[1:]
			}
		}
	}

	if e.circuitBreakerEnabled {
		for workerID, breaker := range e.circuitBreakers {
			if breaker.probeRequestID != requestID {
				continue
			}

			if breaker.state == CircuitHalfOpen {
				if err := e.openCircuit(workerID, breaker); err != nil {
					return err
				}
			} else {
				breaker.probeRequestID = ""
			}
		}
	}

	delete(e.requestNodes, requestID)
	delete(e.visitedNodes, requestID)

	return nil
}

func removeRequestID(queue []string, requestID string) []string {
	for i, id := range queue {
		if id == requestID {
			return append(queue[:i], queue[i+1:]...)
		}
	}
	return queue
}

func (e *Engine) workerCapacity(workerID string) (int, error) {
	if e.router == nil {
		return e.serviceCapacity, nil
	}

	node, exists := e.router.nodes[workerID]
	if !exists {
		return 0, fmt.Errorf("unknown worker node %q", workerID)
	}
	if node.Type != NodeWorker {
		return 0, fmt.Errorf("node %q is not a worker", workerID)
	}
	if node.Capacity > 0 {
		return node.Capacity, nil
	}

	return e.serviceCapacity, nil
}

func (e *Engine) handleWorkerFailure(workerID string) error {
	if e.router == nil {
		return fmt.Errorf("worker failure encountered without a configured router")
	}

	node, exists := e.router.nodes[workerID]
	if !exists || node.Type != NodeWorker {
		return fmt.Errorf("worker failure references invalid worker %q", workerID)
	}

	// Repeated failure events are idempotent.
	if !e.workerOnline[workerID] {
		return nil
	}

	// Count one failure for this worker outage.
	if e.circuitBreakerEnabled {
		breaker := e.circuitBreakers[workerID]
		willOpen := breaker.state == CircuitHalfOpen ||
			(breaker.state == CircuitClosed &&
				breaker.consecutiveFailures+1 >= e.circuitBreakerPolicy.FailureThreshold)

		if willOpen &&
			e.now > Tick(math.MaxInt64)-e.circuitBreakerPolicy.OpenDuration {
			return fmt.Errorf("circuit-breaker cooldown tick overflow for worker %q", workerID)
		}

		if err := e.recordCircuitFailure(workerID, breaker.probeRequestID); err != nil {
			return err
		}
	}

	e.workerOnline[workerID] = false

	// Fail all requests currently in flight on this worker.
	// The eventual completion events will be ignored because these
	// requests are terminal.
	for requestID, request := range e.requests {
		if request.State() != RequestInFlight {
			continue
		}
		if e.requestNodes[requestID] != workerID {
			continue
		}

		e.workerBusy[workerID]--
		e.busyWorkers--
		e.attemptIDs[requestID]++
		delete(e.requestNodes, requestID)
		delete(e.visitedNodes, requestID)

		if e.retryCounts[requestID] < e.retryPolicy.MaxRetries {
			e.retryWorkers[requestID] = workerID
			if e.now > Tick(math.MaxInt64)-e.retryPolicy.Backoff {
				return fmt.Errorf("retry tick overflow for request %q", requestID)
			}
			if e.nextSequence == math.MaxUint64 {
				return fmt.Errorf("event sequence exhausted")
			}

			if err := request.Transition(
				RequestRetryWaiting,
				e.now,
				"",
			); err != nil {
				return err
			}

			e.retryCounts[requestID]++

			if err := e.schedule(Event{
				At:        e.now + e.retryPolicy.Backoff,
				Type:      EventRequestRetry,
				RequestID: requestID,
			}); err != nil {
				return err
			}
		} else {
			if err := request.Transition(
				RequestFailed,
				e.now,
				fmt.Sprintf("worker %q failed", workerID),
			); err != nil {
				return err
			}
		}

	}

	if e.workerBusy[workerID] != 0 {
		return fmt.Errorf(
			"worker %q has %d active requests after failure",
			workerID,
			e.workerBusy[workerID],
		)
	}

	// Keep the waiting queue intact. Requests resume in FIFO order
	// when this worker recovers.
	return nil
}

func (e *Engine) handleWorkerRecovery(workerID string) error {
	if e.router == nil {
		return fmt.Errorf("worker recovery encountered without a configured router")
	}

	node, exists := e.router.nodes[workerID]
	if !exists || node.Type != NodeWorker {
		return fmt.Errorf("worker recovery references invalid worker %q", workerID)
	}

	// Repeated recovery events are idempotent.
	if e.workerOnline[workerID] {
		return nil
	}
	e.workerOnline[workerID] = true

	if e.circuitBreakerEnabled {
		breaker := e.circuitBreakers[workerID]
		if breaker != nil && breaker.state == CircuitOpen {
			// Keep the worker idle until the cooldown expires.
			if e.now < breaker.openUntil {
				return nil
			}

			// If the cooldown has elapsed, let the cooldown handler
			// resume queued work through the recovery-probe path.
			return e.handleCircuitCooldownElapsed(workerID)
		}
	}

	capacity, err := e.workerCapacity(workerID)

	if err != nil {
		return err
	}

	// Restart queued requests in FIFO order, up to worker capacity.
	for e.workerBusy[workerID] < capacity &&
		len(e.workerWaiting[workerID]) > 0 {
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

		if request.State() == RequestSucceeded ||
			request.State() == RequestFailed ||
			request.State() == RequestCancelled ||
			request.State() == RequestTimedOut {
			e.workerWaiting[workerID] = waiting[1:]
			continue
		}

		if request.State() != RequestQueued &&
			request.State() != RequestRetryWaiting {
			return fmt.Errorf(
				"worker %q waiting request %q is not queued",
				workerID,
				requestID,
			)
		}

		if request.State() == RequestRetryWaiting {
			if err := request.Transition(RequestQueued, e.now, ""); err != nil {
				return err
			}
		}

		if err := e.startWorkerService(request, workerID); err != nil {
			return err
		}

		e.workerWaiting[workerID] = waiting[1:]
	}

	return nil
}

// SetRetryPolicy configures the retry behavior for eligible request failures.
func (e *Engine) SetRetryPolicy(policy RetryPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	e.retryPolicy = policy
	return nil
}

// SetCircuitBreakerPolicy configures circuit breakers for topology workers.
func (e *Engine) SetCircuitBreakerPolicy(policy CircuitBreakerPolicy) error {
	if e.router == nil {
		return fmt.Errorf("circuit breakers require a configured topology")
	}
	if err := policy.Validate(); err != nil {
		return err
	}

	e.circuitBreakerPolicy = policy
	e.circuitBreakerEnabled = true
	return nil
}

func (e *Engine) handleRequestRetry(requestID string) error {
	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("retry references unknown request %q", requestID)
	}

	if request.State() != RequestRetryWaiting {
		return nil
	}

	if e.router == nil {
		if err := request.Transition(RequestQueued, e.now, ""); err != nil {
			return err
		}
		if e.busyWorkers < e.serviceCapacity {
			return e.startService(request)
		}
		e.waiting = append(e.waiting, requestID)
		return nil
	}

	sourceNode, exists := e.requestSources[requestID]
	if !exists {
		return fmt.Errorf(
			"cannot retry request %q: original source not recorded",
			requestID,
		)
	}

	workerID, hasRetryWorker := e.retryWorkers[requestID]
	if !hasRetryWorker {
		return fmt.Errorf(
			"cannot retry request %q: failed worker not recorded",
			requestID,
		)
	}

	if !e.workerOnline[workerID] {
		e.workerWaiting[workerID] = append(e.workerWaiting[workerID], requestID)
		e.requestNodes[requestID] = workerID
		return nil
	}

	if err := request.Transition(RequestQueued, e.now, ""); err != nil {
		return err
	}

	delete(e.requestNodes, requestID)
	delete(e.visitedNodes, requestID)

	e.requestNodes[requestID] = sourceNode
	e.visitedNodes[requestID] = map[string]bool{
		sourceNode: true,
	}

	return e.handleRouteHop(requestID, sourceNode)
}
