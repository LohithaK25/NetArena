package sim

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func exportTestReport() SweepReport {
	return SweepReport{
		Name: "capacity sweep",
		Scenarios: []SweepReportRow{
			{
				Name:                   "baseline",
				ServiceCapacity:        1,
				Duration:               10,
				TotalRequests:          5,
				SucceededRequests:      4,
				FailedRequests:         1,
				SuccessRate:            0.8,
				Throughput:             0.4,
				AverageEndToEndLatency: 2.5,
				Retries:                2,
				WorkerUtilization:      0.75,
			},
			{
				Name:                   "candidate, \"large\"\ncapacity",
				ServiceCapacity:        4,
				Duration:               10,
				TotalRequests:          5,
				SucceededRequests:      5,
				SuccessRate:            1,
				Throughput:             0.5,
				AverageEndToEndLatency: 2,
				WorkerUtilization:      0.5,
			},
		},
		Comparisons: []ComparisonReport{
			{
				Baseline:  ScenarioComparisonMetrics{Name: "baseline"},
				Candidate: ScenarioComparisonMetrics{Name: "candidate"},
				Delta:     ScenarioComparisonDelta{SuccessRate: 0.2},
				Workers: map[string]WorkerComparison{
					"worker-1": {
						HasBaseline:      true,
						HasCandidate:     true,
						Baseline:         WorkerMetrics{Capacity: 1, Utilization: 0.75},
						Candidate:        WorkerMetrics{Capacity: 4, Utilization: 0.5},
						UtilizationDelta: -0.25,
					},
				},
			},
		},
	}
}

type exportFailingWriter struct{}

func (exportFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestWriteSweepReportJSONRoundTrip(t *testing.T) {
	report := exportTestReport()

	var output bytes.Buffer
	if err := WriteSweepReportJSON(&output, report); err != nil {
		t.Fatalf("WriteSweepReportJSON() error = %v", err)
	}

	var got SweepReport
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	if !reflect.DeepEqual(got, report) {
		t.Errorf("JSON round trip differs:\n got: %#v\nwant: %#v", got, report)
	}
}

func TestWriteSweepReportJSONIsDeterministic(t *testing.T) {
	report := exportTestReport()

	var first, second bytes.Buffer
	if err := WriteSweepReportJSON(&first, report); err != nil {
		t.Fatalf("first export error = %v", err)
	}
	if err := WriteSweepReportJSON(&second, report); err != nil {
		t.Fatalf("second export error = %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("JSON exports differ for the same report")
	}
}

func TestWriteSweepReportJSONPropagatesWriterError(t *testing.T) {
	if err := WriteSweepReportJSON(exportFailingWriter{}, exportTestReport()); err == nil {
		t.Fatal("WriteSweepReportJSON() error = nil, want writer error")
	}
}

func TestWriteSweepReportCSVWritesScenarioRowsAndEscapesFields(t *testing.T) {
	report := exportTestReport()

	var output bytes.Buffer
	if err := WriteSweepReportCSV(&output, report); err != nil {
		t.Fatalf("WriteSweepReportCSV() error = %v", err)
	}

	records, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatalf("CSV output is invalid: %v", err)
	}

	wantHeader := []string{
		"name",
		"service_capacity",
		"duration",
		"total_requests",
		"succeeded_requests",
		"failed_requests",
		"cancelled_requests",
		"timed_out_requests",
		"success_rate",
		"throughput",
		"average_end_to_end_latency",
		"retries",
		"worker_utilization",
	}
	if !reflect.DeepEqual(records[0], wantHeader) {
		t.Fatalf("CSV header = %#v, want %#v", records[0], wantHeader)
	}
	if len(records) != len(report.Scenarios)+1 {
		t.Fatalf("CSV record count = %d, want %d", len(records), len(report.Scenarios)+1)
	}

	if records[1][0] != "baseline" {
		t.Errorf("first scenario name = %q, want baseline", records[1][0])
	}
	if records[1][1] != "1" || records[1][2] != "10" {
		t.Errorf("first scenario capacity/duration = %q/%q, want 1/10", records[1][1], records[1][2])
	}
	if records[1][8] != "0.8" {
		t.Errorf("success rate = %q, want 0.8", records[1][8])
	}
	if records[2][0] != report.Scenarios[1].Name {
		t.Errorf("escaped scenario name = %q, want original %q", records[2][0], report.Scenarios[1].Name)
	}
}

func TestWriteSweepReportCSVIsDeterministic(t *testing.T) {
	report := exportTestReport()

	var first, second bytes.Buffer
	if err := WriteSweepReportCSV(&first, report); err != nil {
		t.Fatalf("first export error = %v", err)
	}
	if err := WriteSweepReportCSV(&second, report); err != nil {
		t.Fatalf("second export error = %v", err)
	}
	if first.String() != second.String() {
		t.Error("CSV exports differ for the same report")
	}
}

func TestWriteSweepReportCSVEmptyReport(t *testing.T) {
	var output bytes.Buffer
	if err := WriteSweepReportCSV(&output, SweepReport{Name: "empty"}); err != nil {
		t.Fatalf("WriteSweepReportCSV() error = %v", err)
	}

	records, err := csv.NewReader(strings.NewReader(output.String())).ReadAll()
	if err != nil {
		t.Fatalf("CSV output is invalid: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("CSV record count = %d, want header only", len(records))
	}
}

func TestWriteSweepReportCSVPropagatesWriterError(t *testing.T) {
	if err := WriteSweepReportCSV(exportFailingWriter{}, exportTestReport()); err == nil {
		t.Fatal("WriteSweepReportCSV() error = nil, want writer error")
	}
}

var _ io.Writer = exportFailingWriter{}
