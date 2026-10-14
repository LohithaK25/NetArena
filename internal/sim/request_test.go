package sim

import "testing"

func TestNewRequestValidatesInput(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		createdAt Tick
		wantError bool
	}{
		{name: "valid request", id: "req-1", createdAt: 0},
		{name: "empty ID", id: "", createdAt: 0, wantError: true},
		{name: "whitespace ID", id: "   ", createdAt: 0, wantError: true},
		{name: "negative creation tick", id: "req-2", createdAt: -1, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := NewRequest(tt.id, tt.createdAt)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected an error")
				}
				if request != nil {
					t.Fatal("expected no request when construction fails")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}
			if request.State() != RequestCreated {
				t.Fatalf("initial state = %s, want created", request.State())
			}
			if request.CreatedAt() != tt.createdAt {
				t.Fatalf("creation tick = %d, want %d", request.CreatedAt(), tt.createdAt)
			}
		})
	}
}

func TestRequestSuccessfulLifecycle(t *testing.T) {
	request, err := NewRequest("req-1", 0)
	if err != nil {
		t.Fatal(err)
	}

	if got := request.ID(); got != "req-1" {
		t.Fatalf("ID() = %q, want %q", got, "req-1")
	}

	transitions := []struct {
		state RequestState
		at    Tick
	}{
		{state: RequestQueued, at: 2},
		{state: RequestInFlight, at: 4},
		{state: RequestSucceeded, at: 9},
	}

	for _, transition := range transitions {
		if err := request.Transition(transition.state, transition.at, ""); err != nil {
			t.Fatalf("transition to %s failed: %v", transition.state, err)
		}
	}

	if request.State() != RequestSucceeded {
		t.Fatalf("final state = %s, want succeeded", request.State())
	}
	if !request.IsTerminal() {
		t.Fatal("successful request should be terminal")
	}

	queuedAt, hasQueuedAt := request.QueuedAt()
	assertTick(t, queuedAt, hasQueuedAt, 2, true)
	startedAt, hasStartedAt := request.StartedAt()
	assertTick(t, startedAt, hasStartedAt, 4, true)
	finishedAt, hasFinishedAt := request.FinishedAt()
	assertTick(t, finishedAt, hasFinishedAt, 9, true)
}

func TestRequestFailureRequiresReason(t *testing.T) {
	request, err := NewRequest("req-fail", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Transition(RequestQueued, 2, ""); err != nil {
		t.Fatal(err)
	}

	if err := request.Transition(RequestFailed, 3, "  "); err == nil {
		t.Fatal("expected failure without a reason to be rejected")
	}
	if request.State() != RequestQueued {
		t.Fatalf("state changed after rejected transition: %s", request.State())
	}

	if err := request.Transition(RequestFailed, 3, "database unavailable"); err != nil {
		t.Fatal(err)
	}
	if got := request.FailureReason(); got != "database unavailable" {
		t.Fatalf("failure reason = %q, want database unavailable", got)
	}
}

func TestInvalidTransitionDoesNotMutateRequest(t *testing.T) {
	request, err := NewRequest("req-1", 5)
	if err != nil {
		t.Fatal(err)
	}

	if err := request.Transition(RequestSucceeded, 5, ""); err == nil {
		t.Fatal("expected direct transition from created to succeeded to fail")
	}
	if request.State() != RequestCreated {
		t.Fatalf("state changed after rejected transition: %s", request.State())
	}

	if err := request.Transition(RequestQueued, 4, ""); err == nil {
		t.Fatal("expected backwards virtual time to be rejected")
	}
	if request.State() != RequestCreated {
		t.Fatalf("state changed after backwards-time rejection: %s", request.State())
	}
}

func TestTerminalRequestCannotTransitionAgain(t *testing.T) {
	request, err := NewRequest("req-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Transition(RequestCancelled, 1, ""); err != nil {
		t.Fatal(err)
	}

	if err := request.Transition(RequestQueued, 2, ""); err == nil {
		t.Fatal("expected terminal request to reject another transition")
	}
	if request.State() != RequestCancelled {
		t.Fatalf("final state = %s, want cancelled", request.State())
	}
}

func TestQueuedRequestCanTimeout(t *testing.T) {
	request, err := NewRequest("req-timeout", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := request.Transition(RequestQueued, 1, ""); err != nil {
		t.Fatal(err)
	}
	if err := request.Transition(RequestTimedOut, 8, ""); err != nil {
		t.Fatal(err)
	}

	if request.State() != RequestTimedOut || !request.IsTerminal() {
		t.Fatalf("state = %s, terminal = %v; want timed_out, true",
			request.State(), request.IsTerminal())
	}
	finishedAt, hasFinishedAt := request.FinishedAt()
	assertTick(t, finishedAt, hasFinishedAt, 8, true)
}

func assertTick(t *testing.T, got Tick, present bool, want Tick, wantPresent bool) {
	t.Helper()
	if got != want || present != wantPresent {
		t.Fatalf("timestamp = (%d, %v), want (%d, %v)",
			got, present, want, wantPresent)
	}
}

func TestRequestStateNamesAndTerminalClassification(t *testing.T) {
	tests := []struct {
		state        RequestState
		wantName     string
		wantTerminal bool
	}{
		{RequestCreated, "created", false},
		{RequestQueued, "queued", false},
		{RequestInFlight, "in_flight", false},
		{RequestSucceeded, "succeeded", true},
		{RequestFailed, "failed", true},
		{RequestCancelled, "cancelled", true},
		{RequestTimedOut, "timed_out", true},
		{RequestState(255), "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			if got := tt.state.String(); got != tt.wantName {
				t.Fatalf("String() = %q, want %q", got, tt.wantName)
			}
			if got := tt.state.IsTerminal(); got != tt.wantTerminal {
				t.Fatalf("IsTerminal() = %v, want %v", got, tt.wantTerminal)
			}
		})
	}
}

func TestNewRequestHasNoTransitionTimestamps(t *testing.T) {
	request, err := NewRequest("req-new", 5)
	if err != nil {
		t.Fatal(err)
	}

	queuedAt, hasQueuedAt := request.QueuedAt()
	assertTick(t, queuedAt, hasQueuedAt, 0, false)

	startedAt, hasStartedAt := request.StartedAt()
	assertTick(t, startedAt, hasStartedAt, 0, false)

	finishedAt, hasFinishedAt := request.FinishedAt()
	assertTick(t, finishedAt, hasFinishedAt, 0, false)
}

func TestNonFailureTransitionRejectsReasonWithoutMutation(t *testing.T) {
	request, err := NewRequest("req-reason", 0)
	if err != nil {
		t.Fatal(err)
	}

	err = request.Transition(RequestQueued, 1, "unexpected reason")
	if err == nil {
		t.Fatal("expected a reason on a non-failure transition to be rejected")
	}
	if request.State() != RequestCreated {
		t.Fatalf("state changed after rejected transition: %s", request.State())
	}
}

func TestCreatedRequestCanBeCancelledOrTimedOut(t *testing.T) {
	for _, terminal := range []RequestState{
		RequestCancelled,
		RequestTimedOut,
	} {
		t.Run(terminal.String(), func(t *testing.T) {
			request, err := NewRequest("req-terminal", 3)
			if err != nil {
				t.Fatal(err)
			}

			if err := request.Transition(terminal, 3, ""); err != nil {
				t.Fatalf("transition to %s failed: %v", terminal, err)
			}
			if request.State() != terminal {
				t.Fatalf("state = %s, want %s", request.State(), terminal)
			}

			finishedAt, hasFinishedAt := request.FinishedAt()
			assertTick(t, finishedAt, hasFinishedAt, 3, true)
		})
	}
}
