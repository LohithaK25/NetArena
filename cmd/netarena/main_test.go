package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunShowsHelpWithoutArguments(t *testing.T) {
	var output bytes.Buffer

	if err := run(nil, &output); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(output.String(), "sweep") {
		t.Fatalf("help output does not mention sweep: %q", output.String())
	}
}

func TestRunExportsJSONSweep(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "report.json")
	var output bytes.Buffer

	err := run([]string{
		"sweep",
		"--capacities", "2,4",
		"--format", "json",
		"--output", outputPath,
	}, &output)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read JSON report: %v", err)
	}

	var report struct {
		Name      string `json:"Name"`
		Scenarios []struct {
			Name            string `json:"Name"`
			ServiceCapacity int    `json:"ServiceCapacity"`
		} `json:"Scenarios"`
		Comparisons []json.RawMessage `json:"Comparisons"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("invalid JSON report: %v", err)
	}
	if report.Name != "baseline-capacity-sweep" {
		t.Errorf("suite name = %q, want %q", report.Name, "baseline-capacity-sweep")
	}
	if len(report.Scenarios) != 3 {
		t.Fatalf("scenario count = %d, want baseline plus two candidates", len(report.Scenarios))
	}
	if report.Scenarios[0].ServiceCapacity != 1 {
		t.Errorf("baseline capacity = %d, want 1", report.Scenarios[0].ServiceCapacity)
	}
	if report.Scenarios[1].ServiceCapacity != 2 || report.Scenarios[2].ServiceCapacity != 4 {
		t.Errorf("candidate capacities = %d, %d; want 2, 4",
			report.Scenarios[1].ServiceCapacity, report.Scenarios[2].ServiceCapacity)
	}
	if len(report.Comparisons) != 2 {
		t.Errorf("comparison count = %d, want 2", len(report.Comparisons))
	}
}

func TestRunExportsCSVSweep(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "report.csv")
	var output bytes.Buffer

	err := run([]string{
		"sweep",
		"--capacities", "2,4",
		"--format", "csv",
		"--output", outputPath,
	}, &output)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("open CSV report: %v", err)
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatalf("invalid CSV report: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("CSV record count = %d, want header plus three scenarios", len(records))
	}
	if len(records[0]) != 13 {
		t.Fatalf("CSV column count = %d, want 13", len(records[0]))
	}
}

func TestRunRejectsInvalidSweepArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing subcommand",
			args: []string{"unknown"},
		},
		{
			name: "missing capacities",
			args: []string{"sweep", "--output", "report.json"},
		},
		{
			name: "missing output",
			args: []string{"sweep", "--capacities", "2"},
		},
		{
			name: "invalid format",
			args: []string{
				"sweep", "--capacities", "2",
				"--format", "xml", "--output", "report.xml",
			},
		},
		{
			name: "invalid capacity",
			args: []string{
				"sweep", "--capacities", "2,nope",
				"--output", "report.json",
			},
		},
		{
			name: "duplicate capacity",
			args: []string{
				"sweep", "--capacities", "2,2",
				"--output", "report.json",
			},
		},
		{
			name: "capacity matches baseline",
			args: []string{
				"sweep", "--capacities", "1",
				"--output", "report.json",
			},
		},
		{
			name: "zero requests",
			args: []string{
				"sweep", "--capacities", "2",
				"--requests", "0", "--output", "report.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := run(tt.args, &output); err == nil {
				t.Fatal("run() error = nil, want validation error")
			}
		})
	}
}

func TestRunReportsOutputFileErrors(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "missing", "report.json")
	var output bytes.Buffer

	err := run([]string{
		"sweep",
		"--capacities", "2",
		"--output", outputPath,
	}, &output)
	if err == nil {
		t.Fatal("run() error = nil, want output file error")
	}
}
