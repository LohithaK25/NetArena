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
	id               string
	state            RequestState
	createdAt        Tick
	queuedAt         Tick
	startedAt        Tick
	finishedAt       Tick
	lastTransitionAt Tick
	hasQueuedAt      bool
	hasStartedAt     bool
	hasFinishedAt    bool
	failureReason    string
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

func (r *Request) FinishedAt() (Tick, bool) {
	return r.finishedAt, r.hasFinishedAt
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

	switch next {
	case RequestQueued:
		r.queuedAt = at
		r.hasQueuedAt = true
	case RequestInFlight:
		r.startedAt = at
		r.hasStartedAt = true
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
