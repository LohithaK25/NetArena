package sim

import "fmt"

// SchedulingPolicy determines how queued requests are selected.
type SchedulingPolicy string

const (
	// SchedulingFIFO serves queued requests in arrival order.
	SchedulingFIFO SchedulingPolicy = "fifo"

	// SchedulingPriority serves higher-priority requests first.
	SchedulingPriority SchedulingPolicy = "priority"
)

// Validate checks whether the scheduling policy is supported.
func (p SchedulingPolicy) Validate() error {
	switch p {
	case SchedulingFIFO, SchedulingPriority:
		return nil
	default:
		return fmt.Errorf("unsupported scheduling policy %q", p)
	}
}
