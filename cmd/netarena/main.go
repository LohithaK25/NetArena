package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"example.com/netarena/internal/sim"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "netarena: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return writeHelp(out)
	}

	switch args[0] {
	case "sweep":
		return runSweep(args[1:], out)
	default:
		return fmt.Errorf("unknown command %q; run 'netarena help' for usage", args[0])
	}
}

func writeHelp(out io.Writer) error {
	_, err := fmt.Fprintln(out, `NetArena — network and service simulation

Usage:
  netarena sweep --capacities 2,4,8 --format json --output report.json

Commands:
  sweep    Run a service-capacity experiment sweep

Sweep options:
  --capacities         Comma-separated candidate capacities (required)
  --baseline-capacity  Baseline service capacity (default 1)
  --service-duration   Service duration in ticks (default 5)
  --requests           Number of requests to simulate (default 20)
  --arrival-interval   Ticks between request arrivals (default 1)
  --format             Report format: json or csv (default json)
  --output             Output report path (required)`)
	return err
}

func runSweep(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("sweep", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	capacitiesArg := flags.String("capacities", "", "comma-separated candidate capacities")
	format := flags.String("format", "json", "report format: json or csv")
	outputPath := flags.String("output", "", "report output path")
	baselineCapacity := flags.Int("baseline-capacity", 1, "baseline service capacity")
	serviceDuration := flags.Int("service-duration", 5, "service duration in ticks")
	requestCount := flags.Int("requests", 20, "number of requests")
	arrivalInterval := flags.Int("arrival-interval", 1, "ticks between request arrivals")

	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse sweep arguments: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if strings.TrimSpace(*capacitiesArg) == "" {
		return fmt.Errorf("--capacities is required")
	}
	if strings.TrimSpace(*outputPath) == "" {
		return fmt.Errorf("--output is required")
	}
	if *format != "json" && *format != "csv" {
		return fmt.Errorf("unsupported format %q; choose json or csv", *format)
	}
	if *baselineCapacity <= 0 {
		return fmt.Errorf("--baseline-capacity must be positive")
	}
	if *serviceDuration <= 0 {
		return fmt.Errorf("--service-duration must be positive")
	}
	if *requestCount <= 0 {
		return fmt.Errorf("--requests must be positive")
	}
	if *arrivalInterval <= 0 {
		return fmt.Errorf("--arrival-interval must be positive")
	}

	capacities, err := parseCapacities(*capacitiesArg)
	if err != nil {
		return err
	}

	scenario := sim.Scenario{
		Name:            "baseline",
		ServiceCapacity: *baselineCapacity,
		ServiceDuration: sim.Tick(*serviceDuration),
	}
	for i := 0; i < *requestCount; i++ {
		scenario.Arrivals = append(scenario.Arrivals, sim.ScenarioArrival{
			RequestID: fmt.Sprintf("request-%04d", i+1),
			At:        sim.Tick(i * *arrivalInterval),
		})
	}

	suite, err := sim.SweepServiceCapacity(scenario, capacities)
	if err != nil {
		return fmt.Errorf("build capacity sweep: %w", err)
	}
	result, err := sim.RunExperimentSuite(suite)
	if err != nil {
		return fmt.Errorf("run capacity sweep: %w", err)
	}
	report, err := sim.BuildSweepReport(suite, result)
	if err != nil {
		return fmt.Errorf("build sweep report: %w", err)
	}

	file, err := os.Create(*outputPath)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", *outputPath, err)
	}

	var writeErr error
	switch *format {
	case "json":
		writeErr = sim.WriteSweepReportJSON(file, report)
	case "csv":
		writeErr = sim.WriteSweepReportCSV(file, report)
	}
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("write %s report: %w", *format, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close output file %q: %w", *outputPath, closeErr)
	}

	_, err = fmt.Fprintf(out, "Sweep completed: %d scenarios, %d requests each\nReport: %s\n",
		len(report.Scenarios), *requestCount, *outputPath)
	return err
}

func parseCapacities(value string) ([]int, error) {
	parts := strings.Split(value, ",")
	capacities := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("invalid --capacities value %q: empty capacity", value)
		}
		capacity, err := strconv.Atoi(part)
		if err != nil || capacity <= 0 {
			return nil, fmt.Errorf("invalid capacity %q: capacities must be positive integers", part)
		}
		if _, exists := seen[capacity]; exists {
			return nil, fmt.Errorf("duplicate capacity %d", capacity)
		}
		seen[capacity] = struct{}{}
		capacities = append(capacities, capacity)
	}

	return capacities, nil
}
