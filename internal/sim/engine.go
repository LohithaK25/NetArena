package sim

import (
	"fmt"
	"math"
	"strings"
)

// Engine runs a deterministic, discrete-event service simulation.
type Engine struct {
	queue           *EventQueue
	requests        map[string]*Request
	now             Tick
	nextSequence    uint64
	serviceCapacity int
	serviceDuration Tick
	busyWorkers     int
	waiting         []string
	router          *Router
	requestNodes    map[string]string
	visitedNodes    map[string]map[string]bool
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
		visitedNodes:    make(map[string]map[string]bool),
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
			if err := e.handleCompletion(event.RequestID); err != nil {
				return err
			}
		case EventRequestTimeout:
			return fmt.Errorf("request timeout handling is not implemented yet")
		default:
			return fmt.Errorf("unsupported event type: %s", event.Type)
		}
	}

	return nil
}

// Requests returns a snapshot of the request registry.
// Request objects remain owned by the engine.
func (e *Engine) Requests() map[string]*Request {
	snapshot := make(map[string]*Request, len(e.requests))
	for id, request := range e.requests {
		snapshot[id] = request
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
		if e.busyWorkers >= e.serviceCapacity {
			e.waiting = append(e.waiting, requestID)
			e.requestNodes[requestID] = nodeID
			return nil
		}
		return e.startService(request)
	}

	next, err := e.router.NextHop(nodeID)
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

	if err := e.schedule(Event{
		At:        e.now,
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

	e.busyWorkers++

	return e.schedule(Event{
		At:        e.now + e.serviceDuration,
		Type:      EventServiceCompletes,
		RequestID: request.ID(),
	})
}

func (e *Engine) handleCompletion(requestID string) error {
	request, exists := e.requests[requestID]
	if !exists {
		return fmt.Errorf("completion references unknown request %q", requestID)
	}
	if e.busyWorkers <= 0 {
		return fmt.Errorf("completion encountered with no busy workers")
	}

	if err := request.Transition(RequestSucceeded, e.now, ""); err != nil {
		return err
	}

	e.busyWorkers--
	delete(e.requestNodes, requestID)
	delete(e.visitedNodes, requestID)

	if len(e.waiting) > 0 {
		nextID := e.waiting[0]

		nextRequest, exists := e.requests[nextID]
		if !exists {
			return fmt.Errorf("waiting queue references unknown request %q", nextID)
		}

		if e.router != nil {
			if _, exists := e.requestNodes[nextID]; !exists {
				return fmt.Errorf("routed waiting request %q has no worker node", nextID)
			}
			if err := e.startService(nextRequest); err != nil {
				return err
			}
		} else {
			if err := e.startService(nextRequest); err != nil {
				return err
			}
		}

		e.waiting = e.waiting[1:]
	}

	return nil
}
