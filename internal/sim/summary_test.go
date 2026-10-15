package sim

import "testing"

func TestRunSummaryForCompletedRequests(t *testing.T) {
	engine, err := NewEngine(1, 5)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleArrival("first", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleArrival("second", 1); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	summary := engine.Summary()

	if summary.TotalRequests != 2 ||
		summary.SucceededRequests != 2 ||
		summary.FailedRequests != 0 {
		t.Fatalf("unexpected outcome counts: %+v", summary)
	}

	// First request: created at 0, started at 0, finished at 5.
	// Second request: created at 1, started at 5, finished at 10.
	if summary.CompletedRequests != 2 {
		t.Fatalf("CompletedRequests = %d, want 2", summary.CompletedRequests)
	}
	if summary.EndToEndLatency.Min != 5 ||
		summary.EndToEndLatency.Max != 9 ||
		summary.EndToEndLatency.Average != 7 {
		t.Errorf("unexpected end-to-end latency: %+v", summary.EndToEndLatency)
	}
	if summary.ServiceLatency.Min != 5 ||
		summary.ServiceLatency.Max != 5 ||
		summary.ServiceLatency.Average != 5 {
		t.Errorf("unexpected service latency: %+v", summary.ServiceLatency)
	}
	if summary.Retries != 0 {
		t.Errorf("Retries = %d, want 0", summary.Retries)
	}
}

func TestRunSummaryExcludesIncompleteRequestsFromLatency(t *testing.T) {
	engine, err := NewEngine(1, 10)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleArrival("timeout", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleTimeout("timeout", 2); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleArrival("cancelled", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleCancel("cancelled", 4); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	summary := engine.Summary()
	if summary.TimedOutRequests != 1 || summary.CancelledRequests != 1 {
		t.Fatalf("unexpected terminal counts: %+v", summary)
	}
	if summary.CompletedRequests != 0 {
		t.Errorf("CompletedRequests = %d, want 0", summary.CompletedRequests)
	}
	if summary.EndToEndLatency.Count != 0 || summary.ServiceLatency.Count != 0 {
		t.Errorf("incomplete requests contributed latency: %+v", summary)
	}
}

func TestRunSummaryIsDeterministic(t *testing.T) {
	engine, err := NewEngine(2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleArrival("a", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleArrival("b", 1); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	first := engine.Summary()
	second := engine.Summary()
	if first != second {
		t.Fatalf("Summary() changed between calls: first=%+v second=%+v", first, second)
	}
}

func TestRunSummaryExcludesTimedOutRequestsFromLatency(t *testing.T) {
	engine, err := NewEngine(1, 5)
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleArrival("failed", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleTimeout("failed", 2); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	summary := engine.Summary()
	if summary.FailedRequests != 0 || summary.TimedOutRequests != 1 {
		t.Fatalf("unexpected outcome counts: %+v", summary)
	}
	if summary.CompletedRequests != 0 {
		t.Errorf("CompletedRequests = %d, want 0", summary.CompletedRequests)
	}
	if summary.EndToEndLatency.Count != 0 || summary.ServiceLatency.Count != 0 {
		t.Errorf("unsuccessful request contributed latency: %+v", summary)
	}
}

func TestRunSummaryCountsRetries(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 1,
		Backoff:    0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("retry-summary", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 1); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	summary := engine.Summary()
	if summary.Retries != 1 {
		t.Errorf("Retries = %d, want 1", summary.Retries)
	}
	if summary.SucceededRequests != 1 {
		t.Errorf("SucceededRequests = %d, want 1", summary.SucceededRequests)
	}
}

func TestRunSummaryExcludesFailedRequestsFromLatency(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 0,
		Backoff:    0,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("failed-summary", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 1); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	request := engine.Requests()["failed-summary"]
	if request.State() != RequestFailed {
		t.Fatalf("request state = %s, want failed", request.State())
	}

	summary := engine.Summary()
	if summary.FailedRequests != 1 {
		t.Fatalf("FailedRequests = %d, want 1", summary.FailedRequests)
	}
	if summary.TimedOutRequests != 0 {
		t.Fatalf("TimedOutRequests = %d, want 0", summary.TimedOutRequests)
	}
	if summary.CompletedRequests != 0 {
		t.Errorf("CompletedRequests = %d, want 0", summary.CompletedRequests)
	}
	if summary.EndToEndLatency.Count != 0 || summary.ServiceLatency.Count != 0 {
		t.Errorf("failed request contributed to successful latency statistics: %+v", summary)
	}
}

func TestRunSummaryServiceLatencyIncludesAllAttempts(t *testing.T) {
	engine, err := NewEngineWithTopology(1, 2, routingTestTopology())
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.SetRetryPolicy(RetryPolicy{
		MaxRetries: 1,
		Backoff:    2,
	}); err != nil {
		t.Fatal(err)
	}

	if err := engine.ScheduleRoutedArrival("service-latency", "client", 0); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerFailure("worker", 1); err != nil {
		t.Fatal(err)
	}
	if err := engine.ScheduleWorkerRecovery("worker", 3); err != nil {
		t.Fatal(err)
	}
	if err := engine.Run(); err != nil {
		t.Fatal(err)
	}

	request := engine.Requests()["service-latency"]
	if request.State() != RequestSucceeded {
		t.Fatalf("request state = %s, want succeeded", request.State())
	}

	summary := engine.Summary()
	if summary.Retries != 1 {
		t.Fatalf("Retries = %d, want 1", summary.Retries)
	}

	// The first attempt runs for 1 tick and the retry runs for 2 ticks.
	// Service latency excludes the time spent waiting for worker recovery.
	if summary.ServiceLatency.Count != 1 {
		t.Fatalf("ServiceLatency.Count = %d, want 1", summary.ServiceLatency.Count)
	}
	if summary.ServiceLatency.Average != 3 {
		t.Errorf("ServiceLatency.Average = %v, want 3", summary.ServiceLatency.Average)
	}
}
