package sim

import (
	"fmt"
	"testing"
)

func BenchmarkEventQueuePushPop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		queue := NewEventQueue()

		for j := 0; j < 1000; j++ {
			err := queue.Push(Event{
				At:        Tick(j),
				Sequence:  uint64(j + 1),
				Type:      EventRequestArrives,
				RequestID: "bench-request",
			})
			if err != nil {
				b.Fatal(err)
			}
		}

		for queue.Len() > 0 {
			if _, ok := queue.Pop(); !ok {
				b.Fatal("unexpected empty queue")
			}
		}
	}
}

func BenchmarkEngineRun1000Requests(b *testing.B) {
	for i := 0; i < b.N; i++ {
		engine, err := NewEngine(10, 3)
		if err != nil {
			b.Fatal(err)
		}

		for j := 0; j < 1000; j++ {
			if err := engine.ScheduleArrival(
				benchmarkRequestID(j),
				Tick(j%100),
			); err != nil {
				b.Fatal(err)
			}
		}

		if err := engine.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkRequestID(i int) string {
	// IDs are deterministic and unique within each benchmark iteration.
	const digits = "0123456789"
	if i == 0 {
		return "bench-0"
	}

	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}

	return "bench-" + string(buf[pos:])
}

func BenchmarkEventQueueCapacity(b *testing.B) {
	for _, capacity := range []int{0, 2000} {
		b.Run(fmt.Sprintf("Capacity%d", capacity), func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				queue := &EventQueue{
					events: make(eventHeap, 0, capacity),
				}

				for j := 0; j < 1000; j++ {
					if err := queue.Push(Event{
						At:        Tick(j),
						Type:      EventRequestArrives,
						RequestID: benchmarkRequestID(j),
						Sequence:  uint64(j + 1),
					}); err != nil {
						b.Fatalf("queue.Push() error = %v", err)
					}
				}

				for queue.Len() > 0 {
					if _, ok := queue.Pop(); !ok {
						b.Fatal("queue.Pop() unexpectedly returned no event")
					}
				}
			}
		})
	}
}

func BenchmarkEngineRequestsSnapshot(b *testing.B) {
	for _, requestCount := range []int{100, 500, 1000, 5000} {
		b.Run(benchmarkRequestCountName(requestCount), func(b *testing.B) {
			engine, err := NewEngine(10, 3)
			if err != nil {
				b.Fatal(err)
			}

			for i := 0; i < requestCount; i++ {
				if err := engine.ScheduleArrival(
					benchmarkRequestID(i),
					Tick(i%100),
				); err != nil {
					b.Fatal(err)
				}
			}

			if err := engine.Run(); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				requests := engine.Requests()
				if len(requests) != requestCount {
					b.Fatalf("Requests() returned %d requests, want %d", len(requests), requestCount)
				}
			}
		})
	}
}

func BenchmarkEngineSummary(b *testing.B) {
	for _, requestCount := range []int{100, 500, 1000, 5000} {
		b.Run(benchmarkRequestCountName(requestCount), func(b *testing.B) {
			engine, err := NewEngine(10, 3)
			if err != nil {
				b.Fatal(err)
			}

			for i := 0; i < requestCount; i++ {
				if err := engine.ScheduleArrival(
					benchmarkRequestID(i),
					Tick(i%100),
				); err != nil {
					b.Fatal(err)
				}
			}

			if err := engine.Run(); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = engine.Summary()
			}
		})
	}
}
