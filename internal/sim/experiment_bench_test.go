package sim

import "testing"

func BenchmarkRunExperimentSuite(b *testing.B) {
	for _, requestCount := range []int{100, 500, 1000} {
		b.Run(benchmarkRequestCountName(requestCount), func(b *testing.B) {
			suite := benchmarkExperimentSuite(requestCount)

			if err := suite.Validate(); err != nil {
				b.Fatalf("suite.Validate() error = %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if _, err := RunExperimentSuite(suite); err != nil {
					b.Fatalf("RunExperimentSuite() error = %v", err)
				}
			}
		})
	}
}

func benchmarkExperimentSuite(requestCount int) ExperimentSuite {
	makeScenario := func(name string, capacity int) Scenario {
		arrivals := make([]ScenarioArrival, requestCount)
		for i := range arrivals {
			arrivals[i] = ScenarioArrival{
				RequestID: benchmarkRequestID(i),
				At:        Tick(i % 100),
			}
		}

		return Scenario{
			Name:            name,
			ServiceCapacity: capacity,
			ServiceDuration: 3,
			Arrivals:        arrivals,
		}
	}

	return ExperimentSuite{
		Name: "benchmark-experiment",
		Scenarios: []Scenario{
			makeScenario("baseline", 1),
			makeScenario("capacity-2", 2),
			makeScenario("capacity-4", 4),
		},
	}
}

func benchmarkRequestCountName(requestCount int) string {
	switch requestCount {
	case 100:
		return "Requests100"
	case 500:
		return "Requests500"
	case 1000:
		return "Requests1000"
	default:
		return "Requests"
	}
}

func BenchmarkExperimentSuiteValidate(b *testing.B) {
	for _, requestCount := range []int{100, 500, 1000} {
		b.Run(benchmarkRequestCountName(requestCount), func(b *testing.B) {
			suite := benchmarkExperimentSuite(requestCount)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := suite.Validate(); err != nil {
					b.Fatalf("suite.Validate() error = %v", err)
				}
			}
		})
	}
}

func BenchmarkRunScenario(b *testing.B) {
	for _, requestCount := range []int{100, 500, 1000} {
		b.Run(benchmarkRequestCountName(requestCount), func(b *testing.B) {
			scenario := benchmarkExperimentSuite(requestCount).Scenarios[0]

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if _, err := RunScenario(scenario); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
