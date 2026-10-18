package sim

// RouteHopStatus describes the outcome of a selected routing hop.
type RouteHopStatus string

const (
	RouteHopForwarded     RouteHopStatus = "forwarded"
	RouteHopDropped       RouteHopStatus = "dropped"
	RouteHopCycleDetected RouteHopStatus = "cycle_detected"
)

// RouteTraceHop records one connection selected for a request.
type RouteTraceHop struct {
	From    string         `json:"from"`
	To      string         `json:"to"`
	At      Tick           `json:"at"`
	Latency Tick           `json:"latency"`
	Status  RouteHopStatus `json:"status"`
}
