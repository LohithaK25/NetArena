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
	}, nil
}

// ScheduleArrival schedules a request to arrive at a virtual tick.
func (e *Engine) ScheduleArrival(requestID string, at Tick) error {
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

	// Reserve the ID before scheduling so duplicate arrivals are rejected.
	e.requests[requestID] = request

	if err := e.schedule(Event{
		At:        at,
		Type:      EventRequestArrives,
		RequestID: requestID,
	}); err != nil {
		delete(e.requests, requestID)
		return err
	}

	return nil
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

	if e.busyWorkers < e.serviceCapacity {
		return e.startService(request)
	}

	e.waiting = append(e.waiting, requestID)
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
