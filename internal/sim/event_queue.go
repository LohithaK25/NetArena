package sim

import (
	"container/heap"
	"fmt"
)

type eventHeap []Event

func (h eventHeap) Len() int {
	return len(h)
}

func (h eventHeap) Less(i, j int) bool {
	a, b := h[i], h[j]

	if a.At != b.At {
		return a.At < b.At
	}

	priorityA := eventPriority(a.Type)
	priorityB := eventPriority(b.Type)
	if priorityA != priorityB {
		return priorityA < priorityB
	}

	if a.Sequence != b.Sequence {
		return a.Sequence < b.Sequence
	}
	if a.Type != b.Type {
		return a.Type < b.Type
	}
	return a.RequestID < b.RequestID
}

func eventPriority(eventType EventType) int {
	switch eventType {
	case EventRequestArrives:
		return 0
	case EventRequestTimeout:
		return 1
	case EventRequestCancel:
		return 2
	case EventServiceCompletes:
		return 3
	case EventRouteHop:
		return 4
	default:
		return 5
	}
}

func (h eventHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *eventHeap) Push(value any) {
	event, ok := value.(Event)
	if !ok {
		panic("eventHeap.Push: value must be an Event")
	}
	*h = append(*h, event)
}

func (h *eventHeap) Pop() any {
	old := *h
	last := len(old) - 1
	event := old[last]
	old[last] = Event{}
	*h = old[:last]
	return event
}

// EventQueue schedules events in deterministic virtual-time order.
type EventQueue struct {
	events eventHeap
}

// NewEventQueue creates an empty event queue.
func NewEventQueue() *EventQueue {
	queue := &EventQueue{}
	heap.Init(&queue.events)
	return queue
}

// Len returns the number of scheduled events.
func (q *EventQueue) Len() int {
	return q.events.Len()
}

// Push validates and schedules an event.
func (q *EventQueue) Push(event Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("schedule event: %w", err)
	}

	heap.Push(&q.events, event)
	return nil
}

// Pop removes and returns the next event.
// The boolean is false when the queue is empty.
func (q *EventQueue) Pop() (Event, bool) {
	if q.Len() == 0 {
		return Event{}, false
	}

	value := heap.Pop(&q.events)
	event, ok := value.(Event)
	if !ok {
		return Event{}, false
	}

	return event, true
}
