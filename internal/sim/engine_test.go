package sim

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

type cancelAfterChecksContext struct {
	context.Context
	cancel      context.CancelFunc
	checks      int
	cancelAfter int
}

func (c *cancelAfterChecksContext) Err() error {
	c.checks++
	if c.checks >= c.cancelAfter {
		c.cancel()
	}
	return c.Context.Err()
}

func TestEngineProcessesTwoRequestsSequentially(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-1): %v", err)
	}
	if err := engine.ScheduleArrival("req-2", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-2): %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := engine.Requests()
	if len(requests) != 2 {
		t.Fatalf("Requests() returned %d requests, want 2", len(requests))
	}

	first := requests["req-1"]
	second := requests["req-2"]

	if first == nil || second == nil {
		t.Fatal("expected both requests to exist")
	}

	if first.State() != RequestSucceeded {
		t.Errorf("req-1 state = %s, want succeeded", first.State())
	}
	if second.State() != RequestSucceeded {
		t.Errorf("req-2 state = %s, want succeeded", second.State())
	}

	firstStarted, ok := first.StartedAt()
	if !ok || firstStarted != 0 {
		t.Errorf("req-1 start = (%d, %v), want (0, true)", firstStarted, ok)
	}

	firstFinished, ok := first.FinishedAt()
	if !ok || firstFinished != 3 {
		t.Errorf("req-1 finish = (%d, %v), want (3, true)", firstFinished, ok)
	}

	secondStarted, ok := second.StartedAt()
	if !ok || secondStarted != 3 {
		t.Errorf("req-2 start = (%d, %v), want (3, true)", secondStarted, ok)
	}

	secondFinished, ok := second.FinishedAt()
	if !ok || secondFinished != 6 {
		t.Errorf("req-2 finish = (%d, %v), want (6, true)", secondFinished, ok)
	}
}

func TestNewEngineValidatesConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		duration Tick
		wantErr  bool
	}{
		{"valid configuration", 1, 3, false},
		{"zero capacity", 0, 3, true},
		{"negative capacity", -1, 3, true},
		{"zero duration", 1, 0, true},
		{"negative duration", 1, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(tt.capacity, tt.duration)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected configuration error")
				}
				if engine != nil {
					t.Fatal("expected nil engine for invalid configuration")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewEngine() error = %v", err)
			}
			if engine == nil {
				t.Fatal("NewEngine() returned nil engine")
			}
		})
	}
}

func TestEngineRejectsDuplicateRequestID(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleArrival("req-1", 2); err == nil {
		t.Fatal("expected duplicate request ID to be rejected")
	}

	if got := len(engine.Requests()); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}
}

func TestEngineRejectsInvalidArrival(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		id   string
		at   Tick
	}{
		{"empty ID", "", 0},
		{"whitespace ID", "   ", 0},
		{"negative tick", "req-negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := engine.ScheduleArrival(tt.id, tt.at); err == nil {
				t.Fatal("expected invalid arrival to be rejected")
			}
		})
	}

	if got := len(engine.Requests()); got != 0 {
		t.Fatalf("request count after rejected arrivals = %d, want 0", got)
	}
}

func TestEngineRunCanBeCalledAgain(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	if got := engine.CurrentTick(); got != 3 {
		t.Fatalf("CurrentTick() = %d, want 3", got)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	if got := engine.CurrentTick(); got != 3 {
		t.Fatalf("CurrentTick() after second Run = %d, want 3", got)
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() = %d, want 0", got)
	}
}

func TestEngineRespectsServiceCapacity(t *testing.T) {
	engine, err := NewEngine(2, 3)
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"req-1", "req-2", "req-3", "req-4"} {
		if err := engine.ScheduleArrival(id, 0); err != nil {
			t.Fatalf("ScheduleArrival(%s): %v", id, err)
		}
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := engine.Requests()
	if len(requests) != 4 {
		t.Fatalf("request count = %d, want 4", len(requests))
	}

	for id, request := range requests {
		if request.State() != RequestSucceeded {
			t.Errorf("%s state = %s, want succeeded", id, request.State())
		}
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}

	if got := engine.CurrentTick(); got != 6 {
		t.Errorf("CurrentTick() = %d, want 6", got)
	}

	wantStart := map[string]Tick{
		"req-1": 0,
		"req-2": 0,
		"req-3": 3,
		"req-4": 3,
	}
	wantFinish := map[string]Tick{
		"req-1": 3,
		"req-2": 3,
		"req-3": 6,
		"req-4": 6,
	}

	for id, request := range requests {
		start, ok := request.StartedAt()
		if !ok || start != wantStart[id] {
			t.Errorf("%s start = (%d, %v), want (%d, true)",
				id, start, ok, wantStart[id])
		}

		finish, ok := request.FinishedAt()
		if !ok || finish != wantFinish[id] {
			t.Errorf("%s finish = (%d, %v), want (%d, true)",
				id, finish, ok, wantFinish[id])
		}
	}
}

func TestEngineProducesDeterministicResults(t *testing.T) {
	runScenario := func() map[string][3]int64 {
		t.Helper()

		engine, err := NewEngine(2, 4)
		if err != nil {
			t.Fatal(err)
		}

		arrivals := []struct {
			id string
			at Tick
		}{
			{"req-a", 0},
			{"req-b", 0},
			{"req-c", 1},
			{"req-d", 2},
		}

		for _, arrival := range arrivals {
			if err := engine.ScheduleArrival(arrival.id, arrival.at); err != nil {
				t.Fatalf("ScheduleArrival(%s): %v", arrival.id, err)
			}
		}

		if err := engine.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		results := make(map[string][3]int64)
		for id, request := range engine.Requests() {
			start, hasStart := request.StartedAt()
			finish, hasFinish := request.FinishedAt()
			if !hasStart || !hasFinish {
				t.Fatalf("%s is missing lifecycle timestamps", id)
			}

			results[id] = [3]int64{
				int64(request.State()),
				int64(start),
				int64(finish),
			}
		}

		return results
	}

	first := runScenario()
	second := runScenario()

	if len(first) != len(second) {
		t.Fatalf("result counts differ: %d and %d", len(first), len(second))
	}

	for id, want := range first {
		if got := second[id]; got != want {
			t.Errorf("%s result differs: first run %v, second run %v",
				id, want, got)
		}
	}
}

func TestEngineRejectsArrivalBeforeCurrentTick(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", 5); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := engine.CurrentTick(); got != 8 {
		t.Fatalf("CurrentTick() = %d, want 8", got)
	}

	if err := engine.ScheduleArrival("req-past", 7); err == nil {
		t.Fatal("ScheduleArrival() accepted an arrival before the current tick")
	}

	if _, exists := engine.Requests()["req-past"]; exists {
		t.Fatal("rejected request was added to the registry")
	}
}

func TestEngineRejectsDuplicateIDAfterCompletion(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatalf("first ScheduleArrival() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", engine.CurrentTick()); err == nil {
		t.Fatal("ScheduleArrival() accepted a previously used request ID")
	}

	if got := len(engine.Requests()); got != 1 {
		t.Fatalf("len(Requests()) = %d, want 1", got)
	}
}

func TestScheduleRejectsSequenceExhaustion(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	engine.nextSequence = math.MaxUint64

	err = engine.schedule(Event{
		At:        0,
		Type:      EventRequestArrives,
		RequestID: "req-1",
	})
	if err == nil {
		t.Fatal("schedule() succeeded with an exhausted event sequence")
	}

	if got := engine.queue.Len(); got != 0 {
		t.Fatalf("queue length = %d, want 0", got)
	}

	if engine.nextSequence != math.MaxUint64 {
		t.Fatal("event sequence changed after scheduling failed")
	}
}

func TestStartServiceLeavesStateUnchangedOnSequenceExhaustion(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	request, err := NewRequest("req-1", 0)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	if err := request.Transition(RequestQueued, 0, ""); err != nil {
		t.Fatalf("Transition(RequestQueued) error = %v", err)
	}

	engine.nextSequence = math.MaxUint64

	err = engine.startService(request)
	if err == nil {
		t.Fatal("startService() succeeded with an exhausted event sequence")
	}

	if got := request.State(); got != RequestQueued {
		t.Fatalf("request state = %v, want %v", got, RequestQueued)
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() = %d, want 0", got)
	}

	if got := engine.queue.Len(); got != 0 {
		t.Fatalf("queue length = %d, want 0", got)
	}
}

func TestCompletionPreservesWaitingRequestWhenNextServiceCannotStart(t *testing.T) {
	engine, err := NewEngine(1, 3)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-1) error = %v", err)
	}
	if err := engine.ScheduleArrival("req-2", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-2) error = %v", err)
	}

	// Process the first arrival and leave the second request waiting.
	event, ok := engine.queue.Pop()
	if !ok {
		t.Fatal("expected first arrival event")
	}
	engine.now = event.At
	if err := engine.handleArrival(event.RequestID); err != nil {
		t.Fatalf("handleArrival(req-1) error = %v", err)
	}

	event, ok = engine.queue.Pop()
	if !ok {
		t.Fatal("expected second arrival event")
	}
	engine.now = event.At
	if err := engine.handleArrival(event.RequestID); err != nil {
		t.Fatalf("handleArrival(req-2) error = %v", err)
	}

	if len(engine.waiting) != 1 || engine.waiting[0] != "req-2" {
		t.Fatalf("waiting queue = %v, want [req-2]", engine.waiting)
	}

	// Leave the completion event queued, but exhaust the sequence counter.
	engine.nextSequence = math.MaxUint64

	err = engine.handleCompletion("req-1", "", engine.attemptIDs["req-1"])
	if err == nil {
		t.Fatal("handleCompletion() succeeded despite sequence exhaustion")
	}

	if len(engine.waiting) != 1 || engine.waiting[0] != "req-2" {
		t.Fatalf("waiting queue after failure = %v, want [req-2]", engine.waiting)
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() = %d, want 0", got)
	}

	if got := engine.Requests()["req-2"].State(); got != RequestQueued {
		t.Fatalf("req-2 state = %v, want %v", got, RequestQueued)
	}
}

func assertEngineInvariants(t *testing.T, engine *Engine) {
	t.Helper()

	if engine.busyWorkers < 0 {
		t.Fatalf("busyWorkers = %d, want >= 0", engine.busyWorkers)
	}
	if engine.busyWorkers > engine.serviceCapacity {
		t.Fatalf(
			"busyWorkers = %d exceeds capacity %d",
			engine.busyWorkers,
			engine.serviceCapacity,
		)
	}

	inFlight := 0
	for id, request := range engine.requests {
		if request.ID() != id {
			t.Fatalf(
				"request registry key = %q, but request ID = %q",
				id,
				request.ID(),
			)
		}

		if request.State() == RequestInFlight {
			inFlight++
		}

		if request.CreatedAt() < 0 {
			t.Fatalf("request %q has negative creation tick", id)
		}

		queuedAt, hasQueuedAt := request.QueuedAt()
		startedAt, hasStartedAt := request.StartedAt()
		finishedAt, hasFinishedAt := request.FinishedAt()

		if hasQueuedAt && queuedAt < request.CreatedAt() {
			t.Fatalf("request %q was queued before creation", id)
		}

		if hasStartedAt {
			if !hasQueuedAt {
				t.Fatalf("request %q started without being queued", id)
			}
			if startedAt < queuedAt {
				t.Fatalf("request %q started before it was queued", id)
			}
		}

		if hasFinishedAt {
			if !hasStartedAt {
				t.Fatalf("request %q finished without starting service", id)
			}
			if finishedAt < startedAt {
				t.Fatalf("request %q finished before it started", id)
			}
		}

	}

	if inFlight != engine.busyWorkers {
		t.Fatalf(
			"in-flight requests = %d, busyWorkers = %d",
			inFlight,
			engine.busyWorkers,
		)
	}

	seen := make(map[string]bool, len(engine.waiting))
	for _, id := range engine.waiting {
		if seen[id] {
			t.Fatalf("request %q appears more than once in waiting queue", id)
		}
		seen[id] = true

		request, exists := engine.requests[id]
		if !exists {
			t.Fatalf("waiting queue references unknown request %q", id)
		}
		if request.State() != RequestQueued {
			t.Fatalf(
				"waiting request %q has state %v, want %v",
				id,
				request.State(),
				RequestQueued,
			)
		}
	}
}

func TestEngineMaintainsInvariantsDuringService(t *testing.T) {
	engine, err := NewEngine(1, 2)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	for _, id := range []string{"req-1", "req-2"} {
		if err := engine.ScheduleArrival(id, 0); err != nil {
			t.Fatalf("ScheduleArrival(%q) error = %v", id, err)
		}
	}

	assertEngineInvariants(t, engine)

	// Process the first arrival.
	event, ok := engine.queue.Pop()
	if !ok {
		t.Fatal("expected first arrival event")
	}
	engine.now = event.At
	if err := engine.handleArrival(event.RequestID); err != nil {
		t.Fatalf("handleArrival() error = %v", err)
	}
	assertEngineInvariants(t, engine)

	// Process the second arrival; it must wait for the busy worker.
	event, ok = engine.queue.Pop()
	if !ok {
		t.Fatal("expected second arrival event")
	}
	engine.now = event.At
	if err := engine.handleArrival(event.RequestID); err != nil {
		t.Fatalf("handleArrival() error = %v", err)
	}
	assertEngineInvariants(t, engine)

	if len(engine.waiting) != 1 || engine.waiting[0] != "req-2" {
		t.Fatalf("waiting = %v, want [req-2]", engine.waiting)
	}

	// Complete the first request; the waiting request should start.
	event, ok = engine.queue.Pop()
	if !ok {
		t.Fatal("expected service completion event")
	}
	engine.now = event.At
	if err := engine.handleCompletion(event.RequestID, "", event.AttemptID); err != nil {
		t.Fatalf("handleCompletion() error = %v", err)
	}
	assertEngineInvariants(t, engine)

	if got := engine.Requests()["req-2"].State(); got != RequestInFlight {
		t.Fatalf("req-2 state = %v, want %v", got, RequestInFlight)
	}

	// Complete the second request and verify the final state.
	event, ok = engine.queue.Pop()
	if !ok {
		t.Fatal("expected second service completion event")
	}
	engine.now = event.At
	if err := engine.handleCompletion(event.RequestID, "", event.AttemptID); err != nil {
		t.Fatalf("handleCompletion() error = %v", err)
	}
	assertEngineInvariants(t, engine)

	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() = %d, want 0", got)
	}
	if len(engine.waiting) != 0 {
		t.Fatalf("waiting queue = %v, want empty", engine.waiting)
	}
}

func TestCanonicalScenario(t *testing.T) {
	engine, err := NewEngine(2, 3)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	arrivals := []struct {
		id string
		at Tick
	}{
		{id: "req-a", at: 0},
		{id: "req-b", at: 0},
		{id: "req-c", at: 1},
		{id: "req-d", at: 2},
	}

	for _, arrival := range arrivals {
		if err := engine.ScheduleArrival(arrival.id, arrival.at); err != nil {
			t.Fatalf("ScheduleArrival(%q) error = %v", arrival.id, err)
		}
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertEngineInvariants(t, engine)

	expected := map[string]struct {
		started  Tick
		finished Tick
	}{
		"req-a": {started: 0, finished: 3},
		"req-b": {started: 0, finished: 3},
		"req-c": {started: 3, finished: 6},
		"req-d": {started: 3, finished: 6},
	}

	requests := engine.Requests()
	if len(requests) != len(expected) {
		t.Fatalf("request count = %d, want %d", len(requests), len(expected))
	}

	for id, want := range expected {
		request, exists := requests[id]
		if !exists {
			t.Fatalf("missing request %q", id)
		}

		if got := request.State(); got != RequestSucceeded {
			t.Errorf("%s state = %v, want %v", id, got, RequestSucceeded)
		}

		started, hasStarted := request.StartedAt()
		if !hasStarted || started != want.started {
			t.Errorf("%s started at (%d, %t), want (%d, true)",
				id, started, hasStarted, want.started)
		}

		finished, hasFinished := request.FinishedAt()
		if !hasFinished || finished != want.finished {
			t.Errorf("%s finished at (%d, %t), want (%d, true)",
				id, finished, hasFinished, want.finished)
		}
	}

	if got := engine.CurrentTick(); got != 6 {
		t.Errorf("CurrentTick() = %d, want 6", got)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
	if len(engine.waiting) != 0 {
		t.Errorf("waiting queue = %v, want empty", engine.waiting)
	}
}

func TestScheduleTimeoutSchedulesDeadlineWithoutChangingRequestState(t *testing.T) {
	engine, err := NewEngine(1, 5)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-timeout", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}

	if err := engine.ScheduleTimeout("req-timeout", 3); err != nil {
		t.Fatalf("ScheduleTimeout() error = %v", err)
	}

	request := engine.Requests()["req-timeout"]
	if request.State() != RequestCreated {
		t.Fatalf("state after scheduling timeout = %s, want created", request.State())
	}
}

func TestScheduleTimeoutRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		deadline  Tick
		setup     func(*Engine) error
		wantErr   bool
	}{
		{
			name:      "empty request ID",
			requestID: "",
			deadline:  5,
			wantErr:   true,
		},
		{
			name:      "unknown request",
			requestID: "missing",
			deadline:  5,
			wantErr:   true,
		},
		{
			name:      "negative deadline",
			requestID: "req-1",
			deadline:  -1,
			setup: func(e *Engine) error {
				return e.ScheduleArrival("req-1", 0)
			},
			wantErr: true,
		},
		{
			name:      "deadline before current tick",
			requestID: "req-1",
			deadline:  1,
			setup: func(e *Engine) error {
				if err := e.ScheduleArrival("req-1", 0); err != nil {
					return err
				}
				return e.Run()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(1, 2)
			if err != nil {
				t.Fatalf("NewEngine() error = %v", err)
			}

			if tt.setup != nil {
				if err := tt.setup(engine); err != nil {
					t.Fatalf("setup error = %v", err)
				}
			}

			err = engine.ScheduleTimeout(tt.requestID, tt.deadline)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ScheduleTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduleTimeoutRejectsTerminalRequest(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-terminal", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}

	// Run the request to completion before attempting to schedule a timeout.
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if err := engine.ScheduleTimeout("req-terminal", engine.CurrentTick()); err == nil {
		t.Fatal("ScheduleTimeout() accepted a terminal request")
	}
}

func TestQueuedRequestTimesOutAndLeavesWaitingQueue(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-active", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-active) error = %v", err)
	}
	if err := engine.ScheduleArrival("req-waiting", 1); err != nil {
		t.Fatalf("ScheduleArrival(req-waiting) error = %v", err)
	}
	if err := engine.ScheduleTimeout("req-waiting", 3); err != nil {
		t.Fatalf("ScheduleTimeout() error = %v", err)
	}

	err = engine.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	waiting := engine.Requests()["req-waiting"]
	if waiting.State() != RequestTimedOut {
		t.Fatalf("waiting request state = %s, want timed_out", waiting.State())
	}

	finishedAt, hasFinishedAt := waiting.FinishedAt()
	if !hasFinishedAt || finishedAt != 3 {
		t.Fatalf("FinishedAt() = (%d, %v), want (3, true)", finishedAt, hasFinishedAt)
	}

	active := engine.Requests()["req-active"]
	if active.State() != RequestSucceeded {
		t.Fatalf("active request state = %s, want succeeded", active.State())
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() = %d, want 0", got)
	}
}

func TestInFlightRequestTimesOutAndReleasesWorker(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-timeout", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.ScheduleTimeout("req-timeout", 3); err != nil {
		t.Fatalf("ScheduleTimeout() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-timeout"]
	if got := request.State(); got != RequestTimedOut {
		t.Fatalf("state = %s, want %s", got, RequestTimedOut)
	}

	finishedAt, ok := request.FinishedAt()
	if !ok || finishedAt != 3 {
		t.Fatalf("FinishedAt() = (%d, %v), want (3, true)", finishedAt, ok)
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() = %d, want 0", got)
	}
}

func TestTimedOutInFlightRequestDoesNotSucceedOnStaleCompletion(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-timeout", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.ScheduleTimeout("req-timeout", 3); err != nil {
		t.Fatalf("ScheduleTimeout() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-timeout"]
	if got := request.State(); got != RequestTimedOut {
		t.Fatalf("state after stale completion = %s, want %s", got, RequestTimedOut)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() after stale completion = %d, want 0", got)
	}
}

func TestTimeoutWinsWhenCompletionOccursAtSameTick(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-deadline", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.ScheduleTimeout("req-deadline", 10); err != nil {
		t.Fatalf("ScheduleTimeout() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-deadline"]
	if got := request.State(); got != RequestTimedOut {
		t.Fatalf("state = %s, want %s", got, RequestTimedOut)
	}

	finishedAt, ok := request.FinishedAt()
	if !ok || finishedAt != 10 {
		t.Fatalf("FinishedAt() = (%d, %v), want (10, true)", finishedAt, ok)
	}

	if got := engine.BusyWorkers(); got != 0 {
		t.Fatalf("BusyWorkers() = %d, want 0", got)
	}
}

func TestQueuedRequestCanBeCancelledAndLeavesWaitingQueue(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	for _, id := range []string{"req-active", "req-cancel", "req-waiting"} {
		if err := engine.ScheduleArrival(id, 0); err != nil {
			t.Fatalf("ScheduleArrival(%q) error = %v", id, err)
		}
	}
	if err := engine.ScheduleCancel("req-cancel", 3); err != nil {
		t.Fatalf("ScheduleCancel() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := engine.Requests()
	if got := requests["req-cancel"].State(); got != RequestCancelled {
		t.Errorf("cancelled request state = %s, want cancelled", got)
	}
	if got := requests["req-active"].State(); got != RequestSucceeded {
		t.Errorf("active request state = %s, want succeeded", got)
	}
	if got := requests["req-waiting"].State(); got != RequestSucceeded {
		t.Errorf("waiting request state = %s, want succeeded", got)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
	if len(engine.waiting) != 0 {
		t.Errorf("waiting queue = %v, want empty", engine.waiting)
	}
}

func TestInFlightRequestCanBeCancelledAndReleasesWorker(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-cancel", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.ScheduleArrival("req-next", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.ScheduleCancel("req-cancel", 3); err != nil {
		t.Fatalf("ScheduleCancel() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	requests := engine.Requests()
	if got := requests["req-cancel"].State(); got != RequestCancelled {
		t.Errorf("cancelled request state = %s, want cancelled", got)
	}
	if got := requests["req-next"].State(); got != RequestSucceeded {
		t.Errorf("next request state = %s, want succeeded", got)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
}

func TestCancellationWinsWhenCompletionOccursAtSameTick(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-cancel", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.ScheduleCancel("req-cancel", 10); err != nil {
		t.Fatalf("ScheduleCancel() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request := engine.Requests()["req-cancel"]
	if got := request.State(); got != RequestCancelled {
		t.Fatalf("request state = %s, want cancelled", got)
	}

	finishedAt, ok := request.FinishedAt()
	if !ok {
		t.Fatal("cancelled request has no finished timestamp")
	}
	if finishedAt != 10 {
		t.Errorf("finished tick = %d, want 10", finishedAt)
	}
	if got := engine.BusyWorkers(); got != 0 {
		t.Errorf("BusyWorkers() = %d, want 0", got)
	}
}

func TestScheduleCancelRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		at        Tick
		setup     func(*Engine) error
		wantErr   bool
	}{
		{
			name:      "empty request ID",
			requestID: " ",
			at:        1,
			wantErr:   true,
		},
		{
			name:      "unknown request",
			requestID: "missing",
			at:        1,
			wantErr:   true,
		},
		{
			name:      "negative tick",
			requestID: "req-cancel",
			at:        -1,
			setup: func(e *Engine) error {
				return e.ScheduleArrival("req-cancel", 0)
			},
			wantErr: true,
		},
		{
			name:      "tick before current time",
			requestID: "req-cancel",
			at:        0,
			setup: func(e *Engine) error {
				if err := e.ScheduleArrival("req-cancel", 0); err != nil {
					return err
				}
				if err := e.Run(); err != nil {
					return err
				}
				return nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(1, 1)
			if err != nil {
				t.Fatalf("NewEngine() error = %v", err)
			}

			if tt.setup != nil {
				if err := tt.setup(engine); err != nil {
					t.Fatalf("setup error = %v", err)
				}
			}

			err = engine.ScheduleCancel(tt.requestID, tt.at)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ScheduleCancel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduleCancelRejectsTerminalRequest(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-terminal", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if err := engine.ScheduleCancel("req-terminal", engine.CurrentTick()); err == nil {
		t.Fatal("ScheduleCancel() accepted a terminal request")
	}
}

func TestMetricsReflectRequestLifecycleAndCapacity(t *testing.T) {
	engine, err := NewEngine(1, 5)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-1): %v", err)
	}
	if err := engine.ScheduleArrival("req-2", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-2): %v", err)
	}

	// Before the simulation runs, both requests are created.
	got := engine.Metrics()
	if got.TotalRequests != 2 ||
		got.CreatedRequests != 2 ||
		got.QueuedRequests != 0 ||
		got.InFlightRequests != 0 ||
		got.BusyWorkers != 0 ||
		got.ServiceCapacity != 1 ||
		got.WaitingRequests != 0 ||
		got.WorkerUtilization != 0 {
		t.Fatalf("Metrics() before Run() = %+v, want 2 created requests and no active work", got)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run(): %v", err)
	}

	got = engine.Metrics()
	if got.TotalRequests != 2 ||
		got.CreatedRequests != 0 ||
		got.QueuedRequests != 0 ||
		got.InFlightRequests != 0 ||
		got.SucceededRequests != 2 ||
		got.FailedRequests != 0 ||
		got.CancelledRequests != 0 ||
		got.TimedOutRequests != 0 ||
		got.BusyWorkers != 0 ||
		got.WaitingRequests != 0 ||
		got.WorkerUtilization != 0 {
		t.Fatalf("Metrics() after Run() = %+v, want 2 successes and no active or waiting work", got)
	}
}

func TestMetricsCountTimeoutsAndCancellations(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-timeout", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-timeout): %v", err)
	}
	if err := engine.ScheduleTimeout("req-timeout", 2); err != nil {
		t.Fatalf("ScheduleTimeout(req-timeout): %v", err)
	}

	if err := engine.ScheduleArrival("req-cancel", 3); err != nil {
		t.Fatalf("ScheduleArrival(req-cancel): %v", err)
	}
	if err := engine.ScheduleCancel("req-cancel", 4); err != nil {
		t.Fatalf("ScheduleCancel(req-cancel): %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run(): %v", err)
	}

	got := engine.Metrics()
	if got.TotalRequests != 2 ||
		got.TimedOutRequests != 1 ||
		got.CancelledRequests != 1 ||
		got.SucceededRequests != 0 ||
		got.FailedRequests != 0 ||
		got.CreatedRequests != 0 ||
		got.QueuedRequests != 0 ||
		got.InFlightRequests != 0 ||
		got.BusyWorkers != 0 ||
		got.WaitingRequests != 0 ||
		got.WorkerUtilization != 0 {
		t.Fatalf("Metrics() = %+v, want one timeout, one cancellation, and no active work", got)
	}
}

func TestRequestsReturnsIndependentSnapshots(t *testing.T) {
	engine, err := NewEngine(1, 5)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-snapshot", 0); err != nil {
		t.Fatalf("ScheduleArrival(): %v", err)
	}

	snapshots := engine.Requests()
	snapshot := snapshots["req-snapshot"]
	if snapshot == nil {
		t.Fatal("Requests() did not return the scheduled request")
	}

	// Mutate the returned copy. The engine's request must remain unchanged.
	if err := snapshot.Transition(RequestCancelled, 0, ""); err != nil {
		t.Fatalf("Transition() on snapshot: %v", err)
	}

	if got := engine.Requests()["req-snapshot"].State(); got != RequestCreated {
		t.Fatalf("engine request state = %s, want created", got)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run(): %v", err)
	}

	if got := engine.Requests()["req-snapshot"].State(); got != RequestSucceeded {
		t.Fatalf("engine request state after Run() = %s, want succeeded", got)
	}
}

func TestEngineRunRejectsExceededEventBudget(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// A small budget makes the boundary test fast and deterministic.
	engine.maxProcessedEvents = 1

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleArrival("req-2", 0); err != nil {
		t.Fatal(err)
	}

	err = engine.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want event-budget error")
	}
	if !strings.Contains(err.Error(), "processed event limit") {
		t.Fatalf("Run() error = %q, want processed event limit error", err)
	}
}

func TestEngineRunKeepsBudgetExhaustedAcrossCalls(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	engine.maxProcessedEvents = 1

	for _, id := range []string{"req-1", "req-2"} {
		if err := engine.ScheduleArrival(id, 0); err != nil {
			t.Fatal(err)
		}
	}

	if err := engine.Run(); err == nil {
		t.Fatal("first Run() error = nil, want event-budget error")
	}

	if err := engine.Run(); err == nil {
		t.Fatal("second Run() error = nil, want event-budget error")
	}
}

func TestEngineRunContextStopsAfterCancellationDuringExecution(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("req-1", 0); err != nil {
		t.Fatalf("ScheduleArrival(req-1): %v", err)
	}
	if err := engine.ScheduleArrival("req-2", 10); err != nil {
		t.Fatalf("ScheduleArrival(req-2): %v", err)
	}

	baseCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx := &cancelAfterChecksContext{
		Context:     baseCtx,
		cancel:      cancel,
		cancelAfter: 2,
	}

	err = engine.RunContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunContext() error = %v, want %v", err, context.Canceled)
	}

	if engine.CurrentTick() != 0 {
		t.Errorf("CurrentTick() = %d, want 0 after processing only the first event", engine.CurrentTick())
	}

	if engine.processedEvents != 1 {
		t.Errorf("processedEvents = %d, want 1", engine.processedEvents)
	}
}

func TestEngineRunContextRejectsCancelledContext(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	if err := engine.ScheduleArrival("request-1", 0); err != nil {
		t.Fatalf("ScheduleArrival() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = engine.RunContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestEngineAccumulatesLatencyAcrossRoute(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "lb", Type: NodeLoadBalancer},
			{ID: "service", Type: NodeService},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{From: "client", To: "lb", Latency: 3},
			{From: "lb", To: "service", Latency: 5},
			{From: "service", To: "worker", Latency: 2},
		},
	}

	engine, err := NewEngineWithTopology(1, 4, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-1", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request, exists := engine.Requests()["req-1"]
	if !exists {
		t.Fatal("request req-1 not found")
	}
	if got := request.State(); got != RequestSucceeded {
		t.Fatalf("request state = %s, want %s", got, RequestSucceeded)
	}
	if got := engine.CurrentTick(); got != 14 {
		t.Fatalf("CurrentTick() = %d, want 14", got)
	}
}

func TestEngineFailsRequestWhenPacketIsDropped(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{
				From:        "client",
				To:          "worker",
				Latency:     3,
				DropPackets: true,
			},
		},
	}

	engine, err := NewEngineWithTopology(1, 4, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-drop", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request, exists := engine.Requests()["req-drop"]
	if !exists {
		t.Fatal("request req-drop not found")
	}
	if got := request.State(); got != RequestFailed {
		t.Fatalf("request state = %s, want %s", got, RequestFailed)
	}

	wantReason := `packet dropped on connection "client" -> "worker"`
	if got := request.FailureReason(); got != wantReason {
		t.Fatalf("failure reason = %q, want %q", got, wantReason)
	}
	if got := engine.CurrentTick(); got != 0 {
		t.Fatalf("current tick = %d, want 0", got)
	}
}

func TestEngineRoutesSuccessfullyWhenPacketsAreNotDropped(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{
				From:    "client",
				To:      "worker",
				Latency: 3,
			},
		},
	}

	engine, err := NewEngineWithTopology(1, 4, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-normal", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}
	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request, exists := engine.Requests()["req-normal"]
	if !exists {
		t.Fatal("request req-normal not found")
	}
	if got := request.State(); got != RequestSucceeded {
		t.Fatalf("request state = %s, want %s", got, RequestSucceeded)
	}
	if got := engine.CurrentTick(); got != 7 {
		t.Fatalf("current tick = %d, want 7 (3 latency + 4 service)", got)
	}
}

func TestNewEngineWithSeedProducesReproducibleRandomSequence(t *testing.T) {
	first, err := NewEngineWithSeed(1, 1, 42)
	if err != nil {
		t.Fatalf("first NewEngineWithSeed() error = %v", err)
	}

	second, err := NewEngineWithSeed(1, 1, 42)
	if err != nil {
		t.Fatalf("second NewEngineWithSeed() error = %v", err)
	}

	for i := 0; i < 10; i++ {
		gotFirst := first.rng.Int63()
		gotSecond := second.rng.Int63()

		if gotFirst != gotSecond {
			t.Fatalf(
				"random values differ at position %d: %d != %d",
				i,
				gotFirst,
				gotSecond,
			)
		}
	}
}

func TestNewEngineWithTopologyAndSeedProducesReproducibleRandomSequence(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{From: "client", To: "worker", Latency: 2},
		},
	}

	first, err := NewEngineWithTopologyAndSeed(1, 1, topology, 42)
	if err != nil {
		t.Fatalf("first NewEngineWithTopologyAndSeed() error = %v", err)
	}

	second, err := NewEngineWithTopologyAndSeed(1, 1, topology, 42)
	if err != nil {
		t.Fatalf("second NewEngineWithTopologyAndSeed() error = %v", err)
	}

	for i := 0; i < 10; i++ {
		gotFirst := first.rng.Int63()
		gotSecond := second.rng.Int63()

		if gotFirst != gotSecond {
			t.Fatalf(
				"random values differ at position %d: %d != %d",
				i,
				gotFirst,
				gotSecond,
			)
		}
	}
}

func TestNewEngineWithDifferentSeedsCanProduceDifferentSequences(t *testing.T) {
	first, err := NewEngineWithSeed(1, 1, 42)
	if err != nil {
		t.Fatalf("first NewEngineWithSeed() error = %v", err)
	}

	second, err := NewEngineWithSeed(1, 1, 43)
	if err != nil {
		t.Fatalf("second NewEngineWithSeed() error = %v", err)
	}

	different := false
	for i := 0; i < 10; i++ {
		if first.rng.Int63() != second.rng.Int63() {
			different = true
			break
		}
	}

	if !different {
		t.Fatal("different seeds produced identical sequences across 10 values")
	}
}

func TestEngineAppliesProbabilisticPacketLoss(t *testing.T) {
	tests := []struct {
		name        string
		probability float64
		wantState   RequestState
	}{
		{
			name:        "zero probability never drops",
			probability: 0,
			wantState:   RequestSucceeded,
		},
		{
			name:        "certain probability always drops",
			probability: 1,
			wantState:   RequestFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topology := Topology{
				Nodes: []Node{
					{ID: "client", Type: NodeClient},
					{ID: "worker", Type: NodeWorker, Capacity: 1},
				},
				Connections: []Connection{
					{
						From:                  "client",
						To:                    "worker",
						Latency:               2,
						PacketLossProbability: tt.probability,
					},
				},
			}

			engine, err := NewEngineWithTopologyAndSeed(1, 1, topology, 42)
			if err != nil {
				t.Fatalf("NewEngineWithTopologyAndSeed() error = %v", err)
			}
			if err := engine.ScheduleRoutedArrival("request-1", "client", 0); err != nil {
				t.Fatalf("ScheduleRoutedArrival() error = %v", err)
			}
			if err := engine.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			request := engine.Requests()["request-1"]
			if got := request.State(); got != tt.wantState {
				t.Fatalf("request state = %s, want %s", got, tt.wantState)
			}
		})
	}
}

func TestEngineProbabilisticPacketLossIsReproducible(t *testing.T) {
	run := func(seed int64) []RequestState {
		t.Helper()

		topology := Topology{
			Nodes: []Node{
				{ID: "client", Type: NodeClient},
				{ID: "worker", Type: NodeWorker, Capacity: 1},
			},
			Connections: []Connection{
				{
					From:                  "client",
					To:                    "worker",
					Latency:               1,
					PacketLossProbability: 0.5,
				},
			},
		}

		engine, err := NewEngineWithTopologyAndSeed(1, 1, topology, seed)
		if err != nil {
			t.Fatalf("NewEngineWithTopologyAndSeed() error = %v", err)
		}

		for i := 0; i < 20; i++ {
			if err := engine.ScheduleRoutedArrival(
				fmt.Sprintf("request-%d", i),
				"client",
				Tick(i*3),
			); err != nil {
				t.Fatalf("ScheduleRoutedArrival() error = %v", err)
			}
		}

		if err := engine.Run(); err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		states := make([]RequestState, 20)
		for i := range states {
			states[i] = engine.Requests()[fmt.Sprintf("request-%d", i)].State()
		}
		return states
	}

	first := run(42)
	second := run(42)

	for i := range first {
		if first[i] != second[i] {
			t.Fatalf(
				"request %d state differs for identical seeds: %s vs %s",
				i,
				first[i],
				second[i],
			)
		}
	}
}

func TestEngineAppliesProbabilisticPacketLossWithLegacyConstructor(t *testing.T) {
	topology := Topology{
		Nodes: []Node{
			{ID: "client", Type: NodeClient},
			{ID: "worker", Type: NodeWorker, Capacity: 1},
		},
		Connections: []Connection{
			{
				From:                  "client",
				To:                    "worker",
				Latency:               1,
				PacketLossProbability: 1,
			},
		},
	}

	engine, err := NewEngineWithTopology(1, 1, topology)
	if err != nil {
		t.Fatalf("NewEngineWithTopology() error = %v", err)
	}

	if err := engine.ScheduleRoutedArrival("req-legacy", "client", 0); err != nil {
		t.Fatalf("ScheduleRoutedArrival() error = %v", err)
	}

	if err := engine.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	request, exists := engine.Requests()["req-legacy"]
	if !exists {
		t.Fatal("request req-legacy not found")
	}
	if request.State() != RequestFailed {
		t.Errorf("request state = %v, want %v", request.State(), RequestFailed)
	}
}

func TestPriorityQueueRemovalPreservesRemainingOrder(t *testing.T) {
	tests := []struct {
		name          string
		terminalState RequestState
		scheduleEvent func(*Engine, string, Tick) error
	}{
		{
			name:          "timeout",
			terminalState: RequestTimedOut,
			scheduleEvent: func(e *Engine, id string, at Tick) error {
				return e.ScheduleTimeout(id, at)
			},
		},
		{
			name:          "cancellation",
			terminalState: RequestCancelled,
			scheduleEvent: func(e *Engine, id string, at Tick) error {
				return e.ScheduleCancel(id, at)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(1, 10)
			if err != nil {
				t.Fatalf("NewEngine() error = %v", err)
			}
			if err := engine.SetSchedulingPolicy(SchedulingPriority); err != nil {
				t.Fatalf("SetSchedulingPolicy() error = %v", err)
			}

			if err := engine.ScheduleArrival("blocker", 0); err != nil {
				t.Fatal(err)
			}

			arrivals := []struct {
				id       string
				priority int
			}{
				{id: "low", priority: 1},
				{id: "high", priority: 10},
				{id: "medium", priority: 5},
			}

			for _, arrival := range arrivals {
				if err := engine.ScheduleArrivalWithPriority(
					arrival.id, 1, arrival.priority,
				); err != nil {
					t.Fatalf("schedule %s: %v", arrival.id, err)
				}
			}

			// Remove the highest-priority request while it is waiting.
			if err := tt.scheduleEvent(engine, "high", 3); err != nil {
				t.Fatalf("schedule terminal event: %v", err)
			}

			if err := engine.Run(); err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			requests := engine.Requests()
			if got := requests["high"].State(); got != tt.terminalState {
				t.Errorf("high state = %s, want %s", got, tt.terminalState)
			}

			wantStart := map[string]Tick{
				"medium": 10,
				"low":    20,
			}
			for id, want := range wantStart {
				startedAt, ok := requests[id].StartedAt()
				if !ok || startedAt != want {
					t.Errorf(
						"%s started at (%d, %v), want (%d, true)",
						id, startedAt, ok, want,
					)
				}
			}
		})
	}
}
