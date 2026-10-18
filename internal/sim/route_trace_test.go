package sim

import "testing"

func TestEngineRouteTracesReturnsIndependentSnapshot(t *testing.T) {
	engine, err := NewEngine(1, 1)
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	engine.routeTraces["request-1"] = []RouteTraceHop{
		{
			From:    "client",
			To:      "worker",
			At:      2,
			Latency: 3,
			Status:  RouteHopForwarded,
		},
	}

	snapshot := engine.RouteTraces()
	snapshot["request-1"][0].To = "modified"
	snapshot["request-2"] = []RouteTraceHop{{From: "other"}}

	got := engine.RouteTraces()
	if got["request-1"][0].To != "worker" {
		t.Fatalf("engine trace was mutated through snapshot: %+v", got["request-1"][0])
	}
	if _, exists := got["request-2"]; exists {
		t.Fatal("adding an entry to the snapshot modified the engine traces")
	}
}
