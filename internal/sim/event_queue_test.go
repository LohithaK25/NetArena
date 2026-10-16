package sim

import (
	"fmt"
	"testing"
)

func makeTestEvent(at Tick, sequence uint64) Event {
	return Event{
		At:        at,
		Sequence:  sequence,
		Type:      EventRequestArrives,
		RequestID: "req-test",
	}
}

func TestEventQueueOrdersByTickThenSequence(t *testing.T) {
	queue := NewEventQueue()

	events := []Event{
		makeTestEvent(8, 6),
		makeTestEvent(3, 5),
		makeTestEvent(3, 2),
		makeTestEvent(5, 1),
		makeTestEvent(3, 4),
	}

	for _, event := range events {
		if err := queue.Push(event); err != nil {
			t.Fatalf("Push(%+v): %v", event, err)
		}
	}

	if got, want := queue.Len(), len(events); got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}

	want := []Event{
		makeTestEvent(3, 2),
		makeTestEvent(3, 4),
		makeTestEvent(3, 5),
		makeTestEvent(5, 1),
		makeTestEvent(8, 6),
	}

	for i, expected := range want {
		got, ok := queue.Pop()
		if !ok {
			t.Fatalf("Pop() %d: expected an event", i)
		}
		if got != expected {
			t.Fatalf("Pop() %d = %+v, want %+v", i, got, expected)
		}
	}

	if got := queue.Len(); got != 0 {
		t.Fatalf("Len() after draining = %d, want 0", got)
	}
}

func TestEventQueuePopOnEmptyQueue(t *testing.T) {
	queue := NewEventQueue()

	event, ok := queue.Pop()
	if ok {
		t.Fatalf("Pop() on empty queue returned %+v, want no event", event)
	}
	if got := queue.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
}

func TestEventQueueRejectsInvalidEvent(t *testing.T) {
	queue := NewEventQueue()

	err := queue.Push(Event{
		At:        -1,
		Sequence:  1,
		Type:      EventRequestArrives,
		RequestID: "req-invalid",
	})
	if err == nil {
		t.Fatal("Push() should reject an event with a negative tick")
	}

	if got := queue.Len(); got != 0 {
		t.Fatalf("Len() after rejected event = %d, want 0", got)
	}
}

func TestEventQueueCanBeReusedAfterBecomingEmpty(t *testing.T) {
	queue := NewEventQueue()

	first := makeTestEvent(2, 1)
	if err := queue.Push(first); err != nil {
		t.Fatal(err)
	}

	if got, ok := queue.Pop(); !ok || got != first {
		t.Fatalf("first Pop() = (%+v, %v), want (%+v, true)", got, ok, first)
	}

	second := makeTestEvent(1, 2)
	if err := queue.Push(second); err != nil {
		t.Fatal(err)
	}

	if got, ok := queue.Pop(); !ok || got != second {
		t.Fatalf("second Pop() = (%+v, %v), want (%+v, true)", got, ok, second)
	}
}

func TestEventQueuePrioritizesTimeoutBeforeCompletion(t *testing.T) {
	queue := NewEventQueue()

	events := []Event{
		{
			At: 3, Sequence: 1,
			Type: EventServiceCompletes, RequestID: "req-a",
		},
		{
			At: 3, Sequence: 2,
			Type: EventRequestTimeout, RequestID: "req-a",
		},
		{
			At: 3, Sequence: 3,
			Type: EventRequestCancel, RequestID: "req-a",
		},
		{
			At: 3, Sequence: 4,
			Type: EventRequestArrives, RequestID: "req-b",
		},
		{
			At: 3, Sequence: 5,
			Type: EventRouteHop, RequestID: "req-c", NodeID: "node-a",
		},
		{
			At: 3, Sequence: 6,
			Type: EventWorkerFails, NodeID: "worker-a",
		},
		{
			At: 3, Sequence: 7,
			Type: EventWorkerRecovers, NodeID: "worker-a",
		},
	}

	for _, event := range events {
		if err := queue.Push(event); err != nil {
			t.Fatalf("Push(%+v): %v", event, err)
		}
	}

	// At the same tick, arrivals, timeouts, and cancellations precede
	// worker failure and recovery, service completion, and route hops.
	want := []Event{
		events[3], events[1], events[2],
		events[5], events[6], events[0], events[4],
	}

	for i, expected := range want {
		got, ok := queue.Pop()
		if !ok {
			t.Fatalf("Pop() %d: expected an event", i)
		}
		if got != expected {
			t.Fatalf("Pop() %d = %+v, want %+v", i, got, expected)
		}
	}
}

func TestEventQueueInitialCapacity(t *testing.T) {
	for _, capacity := range []int{0, 8, -1} {
		t.Run(fmt.Sprintf("capacity_%d", capacity), func(t *testing.T) {
			queue := newEventQueueWithCapacity(capacity)

			if got := queue.Len(); got != 0 {
				t.Fatalf("Len() = %d, want 0", got)
			}

			wantCapacity := capacity
			if wantCapacity < 0 {
				wantCapacity = 0
			}
			if got := cap(queue.events); got < wantCapacity {
				t.Fatalf("backing capacity = %d, want at least %d", got, wantCapacity)
			}
		})
	}
}
