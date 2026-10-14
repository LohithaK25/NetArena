package sim

import "testing"

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
