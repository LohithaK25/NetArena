package sim

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// WriteSweepReportJSON writes a complete sweep report as JSON.
// Scenario order is preserved; encoding/json sorts string map keys.
func WriteSweepReportJSON(w io.Writer, report SweepReport) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("encode sweep report as JSON: %w", err)
	}
	return nil
}

// WriteSweepReportCSV writes one row per scenario, with a stable column order.
// Comparison details are available in the JSON export.
func WriteSweepReportCSV(w io.Writer, report SweepReport) error {
	writer := csv.NewWriter(w)

	header := []string{
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
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write sweep report CSV header: %w", err)
	}

	for _, row := range report.Scenarios {
		record := []string{
			row.Name,
			strconv.Itoa(row.ServiceCapacity),
			strconv.FormatInt(int64(row.Duration), 10),
			strconv.Itoa(row.TotalRequests),
			strconv.Itoa(row.SucceededRequests),
			strconv.Itoa(row.FailedRequests),
			strconv.Itoa(row.CancelledRequests),
			strconv.Itoa(row.TimedOutRequests),
			strconv.FormatFloat(row.SuccessRate, 'g', -1, 64),
			strconv.FormatFloat(row.Throughput, 'g', -1, 64),
			strconv.FormatFloat(row.AverageEndToEndLatency, 'g', -1, 64),
			strconv.Itoa(row.Retries),
			strconv.FormatFloat(row.WorkerUtilization, 'g', -1, 64),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write sweep report CSV scenario %q: %w", row.Name, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush sweep report CSV: %w", err)
	}
	return nil
}
