package sim

import (
	"fmt"
	"strings"
)

// Tick represents one unit of virtual simulation time.
// Its conversion to real time is intentionally undefined for now.
type Tick int64

// RequestState describes the lifecycle state of a simulated request.
type RequestState uint8

const (
	RequestCreated RequestState = iota + 1
	RequestQueued
	RequestInFlight
	RequestRetryWaiting
	RequestSucceeded
	RequestFailed
	RequestCancelled
	RequestTimedOut
)

// String returns a readable name for a request state.
func (s RequestState) String() string {
	switch s {
	case RequestCreated:
		return "created"
	case RequestQueued:
		return "queued"
	case RequestInFlight:
		return "in_flight"
	case RequestRetryWaiting:
		return "retry_waiting"
	case RequestSucceeded:
		return "succeeded"
	case RequestFailed:
		return "failed"
	case RequestCancelled:
		return "cancelled"
	case RequestTimedOut:
		return "timed_out"
	default:
		return "unknown"
	}
}

// IsTerminal reports whether the request has reached a final state.
func (s RequestState) IsTerminal() bool {
	switch s {
	case RequestSucceeded, RequestFailed, RequestCancelled, RequestTimedOut:
		return true
	default:
		return false
	}
}

// Request is the authoritative lifecycle record for one simulated request.
// Fields are private so callers must use the validated transition method.
type Request struct {
	id                      string
	state                   RequestState
	createdAt               Tick
	queuedAt                Tick
	startedAt               Tick
	finishedAt              Tick
	lastTransitionAt        Tick
	hasQueuedAt             bool
	hasStartedAt            bool
	hasFinishedAt           bool
	failureReason           string
	attempts                int
	currentAttemptStartedAt Tick
	totalServiceTime        Tick
}

// NewRequest creates a request in the created state.
func NewRequest(id string, createdAt Tick) (*Request, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("request ID must not be empty")
	}
	if createdAt < 0 {
		return nil, fmt.Errorf("creation tick must not be negative")
	}

	return &Request{
		id:               id,
		state:            RequestCreated,
		createdAt:        createdAt,
		lastTransitionAt: createdAt,
	}, nil
}

func (r *Request) ID() string {
	return r.id
}

func (r *Request) State() RequestState {
	return r.state
}

func (r *Request) CreatedAt() Tick {
	return r.createdAt
}

func (r *Request) QueuedAt() (Tick, bool) {
	return r.queuedAt, r.hasQueuedAt
}

func (r *Request) StartedAt() (Tick, bool) {
	return r.startedAt, r.hasStartedAt
}

// ServiceTime returns the accumulated duration of completed service attempts.
// Time spent in a currently active attempt is not included until that attempt
// transitions out of RequestInFlight.
func (r *Request) ServiceTime() Tick {
	return r.totalServiceTime
}

func (r *Request) FinishedAt() (Tick, bool) {
	return r.finishedAt, r.hasFinishedAt
}

// Attempts returns the number of service attempts started for this request.
func (r *Request) Attempts() int {
	return r.attempts
}

func (r *Request) FailureReason() string {
	return r.failureReason
}

func (r *Request) IsTerminal() bool {
	return r.state.IsTerminal()
}

// Transition moves a request to a valid next state at virtual tick at.
// A non-empty reason is required only when transitioning to RequestFailed.
func (r *Request) Transition(next RequestState, at Tick, reason string) error {
	if at < r.lastTransitionAt {
		return fmt.Errorf(
			"request %q: transition tick %d precedes previous tick %d",
			r.id, at, r.lastTransitionAt,
		)
	}

	if !canTransition(r.state, next) {
		return fmt.Errorf(
			"request %q: invalid transition from %s to %s",
			r.id, r.state, next,
		)
	}

	if next == RequestFailed {
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("request %q: failed state requires a reason", r.id)
		}
	} else if reason != "" {
		return fmt.Errorf("request %q: reason is only valid for a failed state", r.id)
	}

	if r.state == RequestInFlight && next != RequestInFlight {
		r.totalServiceTime += at - r.currentAttemptStartedAt
	}

	switch next {
	case RequestQueued:
		r.queuedAt = at
		r.hasQueuedAt = true
	case RequestInFlight:
		r.startedAt = at
		r.hasStartedAt = true
		r.currentAttemptStartedAt = at
		r.attempts++
	case RequestSucceeded, RequestFailed, RequestCancelled, RequestTimedOut:
		r.finishedAt = at
		r.hasFinishedAt = true
	}

	r.state = next
	r.lastTransitionAt = at
	if next == RequestFailed {
		r.failureReason = strings.TrimSpace(reason)
	}

	return nil
}

func canTransition(current, next RequestState) bool {
	switch current {
	case RequestCreated:
		return next == RequestQueued ||
			next == RequestCancelled ||
			next == RequestTimedOut
	case RequestQueued:
		return next == RequestInFlight ||
			next == RequestFailed ||
			next == RequestCancelled ||
			next == RequestTimedOut
	case RequestInFlight:
		return next == RequestSucceeded ||
			next == RequestFailed ||
			next == RequestRetryWaiting ||
			next == RequestCancelled ||
			next == RequestTimedOut
	case RequestRetryWaiting:
		return next == RequestQueued ||
			next == RequestCancelled ||
			next == RequestTimedOut
	default:
		// Terminal and unknown states cannot transition.
		return false
	}
}

// clone returns an independent copy of the request lifecycle record.
func (r *Request) clone() *Request {
	snapshot := *r
	return &snapshot
}

// RetryPolicy defines how many times a request may be retried after
// an eligible failure and the delay before each retry.
type RetryPolicy struct {
	MaxRetries int
	Backoff    Tick
}

// Validate checks whether the retry policy is valid.
func (p RetryPolicy) Validate() error {
	if p.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}
	if p.Backoff < 0 {
		return fmt.Errorf("retry backoff must be non-negative")
	}
	return nil
}
