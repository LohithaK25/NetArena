package sim

import "fmt"

// EventType identifies the kind of work an event represents.
type EventType uint8

const (
	EventRequestArrives EventType = iota + 1
	EventRequestTimeout
	EventServiceCompletes
	EventRouteHop
)

// String returns a readable event type name.
func (t EventType) String() string {
	switch t {
	case EventRequestArrives:
		return "request_arrives"
	case EventRequestTimeout:
		return "request_timeout"
	case EventServiceCompletes:
		return "service_completes"
	case EventRouteHop:
		return "route_hop"
	default:
		return "unknown"
	}
}

// Event represents a scheduled action in virtual simulation time.
// Sequence breaks ties between events scheduled for the same tick.
// NodeID identifies the node being processed by a route-hop event.
type Event struct {
	At        Tick
	Sequence  uint64
	Type      EventType
	RequestID string
	NodeID    string
}

// Validate checks that an event contains valid scheduling data.
func (e Event) Validate() error {
	if e.At < 0 {
		return fmt.Errorf("event tick must not be negative")
	}
	if e.Sequence == 0 {
		return fmt.Errorf("event sequence must be positive")
	}
	if e.Type < EventRequestArrives || e.Type > EventRouteHop {
		return fmt.Errorf("invalid event type: %d", e.Type)
	}
	if e.RequestID == "" {
		return fmt.Errorf("event request ID must not be empty")
	}
	if e.Type == EventRouteHop && e.NodeID == "" {
		return fmt.Errorf("route-hop event node ID must not be empty")
	}
	return nil
}
