# Verification Log

## Phase 0 — Go foundation and CI

- **Date:** 2026-10-14
- **Repository:** `E:\NetArena`
- **Branch:** `main` (Phase 0 committed as `42dd958`; no remote configured)
- **Module:** `example.com/netarena` (provisional because no remote is configured)
- **Status:** Go foundation implemented; local checks recorded below. Hosted CI and clean-checkout verification remain pending.
- **Scope:** Minimal Go executable and unit test; GitHub Actions workflow; development documentation. No simulation engine or service features.

### Environment

These commands were run in PowerShell from the repository root. Go is installed at `C:\Program Files\Go\bin`, which was added to the command's process-local `PATH` because it is not on the default `PATH`.

| Tool | Command | Result |
|---|---|---|
| Git | `git --version` | `git version 2.54.0.windows.1` |
| Go | `go version` | `go version go1.27.0 windows/amd64` |
| Node.js | `node --version` | `v24.16.0` |
| npm | `npm --version` | `11.13.0` |
| Docker | `docker --version` | `Docker version 29.5.2, build 79eb04c` |
| Docker Compose | `docker compose version` | `Docker Compose version v5.1.4` |

### Local checks

- **Formatting — passed:** `gofmt -w cmd\netarena\main.go cmd\netarena\main_test.go`, then checked all repository Go files using `Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -l $_.FullName }`; no unformatted files were reported.
- **Vet — passed:** `go vet ./...`.
- **Standard tests — passed:** `go test ./...`; `ok example.com/netarena/cmd/netarena`.
- **Race tests — blocked by local compiler:** `go test -race ./...` failed before package tests because the only discovered C compiler is `C:\MinGW\bin\gcc.exe` (MinGW.org GCC 6.3.0), which reports `cc1.exe: sorry, unimplemented: 64-bit mode not compiled in`. The Go race detector requires a working 64-bit C compiler on this platform. Race behavior is not verified locally.
- **Build — passed:** `go build ./...`.
- **Executable smoke check — passed:** `go run ./cmd/netarena`; output: `NetArena Go foundation is ready.`
- **Hosted CI — not run:** No Git remote is configured, so GitHub Actions has not executed.

The documented exact local commands and formatting instructions are in the README. A passing ordinary test run does not replace the blocked race-enabled test.

### Decisions and remaining Phase 0 work

- Used `example.com/netarena` as a provisional module path in the absence of a remote. Replace it with the canonical import path before publishing or depending on imports from other packages.
- Kept the executable and its focused test on the Go standard library; no dependencies were added.
- The CLI test asserts the exact startup output. Simulation, API, persistence, frontend, and concurrency behavior are not implemented or verified.
- Configure a 64-bit C compiler and rerun `go test -race ./...`; verify Actions and setup instructions on GitHub after configuring a remote and initial commit.
- No benchmarks or deployment checks were run.

## Phase 1.1 — Request lifecycle
- Date: 2026-10-14
- Scope: Added the internal/sim request lifecycle model and unit tests.
- Implementation: Added virtual tick representation, request states, terminal-state classification, validated state transitions, lifecycle timestamps, and failure-reason validation. Request fields are private so state changes go through the transition API.
- Transition rules: Created requests can be queued, cancelled, or timed out; queued requests can start, fail, be cancelled, or time out; in-flight requests can succeed, fail, be cancelled, or time out. Terminal requests cannot transition again.
- Tests — passed: go test ./....
- Vet — passed: go vet ./....
- Build — passed: go build ./....
- Statement coverage — passed: go test -cover ./internal/sim reported 100.0%.
- Whitespace check — passed: git diff --check. Git emitted LF-to-CRLF line-ending warnings; these did not indicate a whitespace-check failure.
- Race detection: Still blocked by the previously documented local 32-bit GCC limitation. No race-enabled result is claimed.
- Hosted CI: Not verified; no remote is configured.
- Next milestone: Implement deterministic event ordering and a priority queue.
## Phase 1.2 — Event model and deterministic priority queue

- **Date:** 2026-10-14
- **Scope:** Added event types, event validation, and a priority queue in `internal/sim`.
- **Implementation:** Events use virtual ticks, sequence numbers, event types, and request IDs. The priority queue uses Go's standard-library `container/heap`.
- **Ordering:** Events are ordered by ascending tick, then sequence number, event type, and request ID. This defines deterministic ordering when earlier keys tie.
- **Validation:** Negative ticks, zero sequence numbers, unknown event types, and empty request IDs are rejected.
- **Queue tests:** Verified ordering, empty-queue behavior, reuse after draining, invalid-event rejection, and deterministic fallback ordering for sequence collisions.
- **Tests — passed:** `go test ./...`.
- **Vet — passed:** `go vet ./...`.
- **Build — passed:** `go build ./...`.
- **Statement coverage — passed:** `go test -cover ./internal/sim` reported 98.0%.
- **Whitespace check — passed:** `git diff --check`.
- **Race detection:** Still blocked by the documented local 32-bit GCC limitation. No race-enabled result is claimed.
- **Hosted CI:** Not verified; no remote is configured.
- **Remaining work:** Implement request/service/resource simulation, invariant and deterministic replay tests, benchmarks, and the first canonical scenario.
## Phase 1.3
- **Date:** 2026-10-14
- The discrete-event engine and its fixed service capacity and duration.
- FIFO waiting requests and service completion events.
- Deterministic event ordering and replay tests.
- Validation for invalid configuration, duplicate IDs, past arrivals, tick overflow, and event-sequence exhaustion.
- The waiting-queue regression test.
- **Verification results:** go test ./internal/sim, go test ./..., go vet ./..., and go build ./... passed.
-Any limitations still present, particularly that timeout handling is not implemented and race detection/hosted CI remain unverified.
## Phase 1.4 verification
- **Date:** 2026-10-14
**Scope**
- Added an engine invariant helper to validate worker capacity, in-flight request counts, request registry consistency, waiting-queue uniqueness, and timestamp ordering.
- Added a test that checks invariants during request arrival, queueing, service, and completion.
- Added benchmarks for event queue push/pop operations and a complete 1,000-request engine workload.
- Added a canonical deterministic scenario with two workers, service duration of three ticks, and four requests arriving at ticks 0, 0, 1, and 2.

**Canonical scenario results**
- All four requests reached `RequestSucceeded`.
- `req-a` and `req-b` started at tick 0 and finished at tick 3.
- `req-c` and `req-d` started at tick 3 and finished at tick 6.
- Final virtual tick: 6.
- Final busy worker count: 0.
- Final waiting queue: empty.

**Verification**
- `go test ./internal/sim -run '^TestCanonicalScenario$' -v` — passed.
- `go test ./...` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `go test ./internal/sim -run '^$' -bench . -benchmem` — passed.

**Benchmark baseline**
- Environment: Windows/amd64, Intel Core i7-1165G7 @ 2.80 GHz.
- Event queue, 1,000 pushes and pops: 266,585 ns/op in the first run; 396,420 ns/op in the second run.
- Engine, 1,000 requests: 855,053 ns/op in the first run; 870,424 ns/op in the second run.
- Benchmark values are initial measurements and have not been used to claim an optimization.

**Limitations**
- Request timeout handling remains unimplemented.
- Local race-detector verification remains blocked by the available C compiler.
- Hosted GitHub Actions CI remains unverified.
- Benchmark results may vary with system load.

## Phase 2 verification — 2026-10-15

- Topology validation and deterministic round-robin routing.
- Topology-aware request arrival and route-hop events.
- Failure handling for dead ends and routing cycles.
- FIFO waiting when the shared service capacity is full.
- Verification: `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` all passed.
- Limitation: all worker nodes currently share one global service-capacity pool; per-node capacity and network-hop latency are not implemented.
- Note: Git emitted LF-to-CRLF warnings, which did not cause the checks to fail.


## Phase 2.1 — Independent worker capacities

- **Date:** 2026-10-15
- **Scope:** Added configurable service capacity per worker node in the topology-aware simulation engine.
- **Capacity rules:** A positive `Node.Capacity` specifies the worker's capacity. A capacity of zero uses the engine's default service capacity. Negative capacities and nonzero capacities on non-worker nodes are rejected.
- **Scheduling:** Each worker maintains its own active-service count and FIFO waiting queue. Requests waiting for one worker do not consume another worker's capacity. The existing non-topology engine retains its shared service-capacity behavior.
- **Completion handling:** Completion events identify the worker that handled the request. Worker identity and active-work counts are validated. If the next waiting request cannot start, the completed request remains succeeded, worker counters are released, and the next request remains queued.
- **Tests:** Added independent-capacity coverage for workers with different capacities and a regression test for a failed completion handoff. The existing non-topology handoff regression test also remains passing.
- **Targeted regression tests — passed:** `go test ./internal/sim -run 'TestRoutedCompletionPreservesWaitingRequestWhenNextServiceCannotStart|TestCompletionPreservesWaitingRequestWhenNextServiceCannotStart' -count=1 -v`.
- **Standard tests — passed:** `go test ./...`.
- **Vet — passed:** `go vet ./...`.
- **Build — passed:** `go build ./...`.
- **Whitespace check — passed:** `git diff --check`. Git emitted LF-to-CRLF warnings; these did not cause the check to fail.
- **Remaining work:** Configurable network-hop latency. Race-detector verification remains blocked by the previously documented local compiler limitation; hosted CI remains unverified because no Git remote is configured.

## Phase 2.2 — Network-hop latency
- **Date:** 2026-10-15
- Connection latency is measured in simulation ticks; zero means instantaneous routing.
- Negative connection latency is rejected.
- The router preserves deterministic round-robin selection and returns the chosen connection's latency.
- Route-hop scheduling checks for tick overflow.
- Verification passed: focused latency/overflow tests, go test ./..., go vet ./..., go build ./..., and git diff --check.
- The race detector and hosted CI were not verified as part of this run.

## Phase 2.3 — Request timeouts
- **Date:** 2026-10-15
- **Scope:** Implemented request timeout scheduling and handling in the deterministic simulation engine.
- **Scheduling:** Added `ScheduleTimeout` validation for blank IDs, unknown requests, terminal requests, and deadlines earlier than the current simulation tick. Invalid negative deadlines are rejected by event validation.
- **Timeout behavior:** Queued requests are removed from their waiting queues. In-flight requests release worker capacity and allow the next eligible request to start. Timed-out requests transition to `RequestTimedOut`.
- **Stale events:** Arrival, route-hop, completion, and timeout handlers handle terminal requests without incorrectly processing them again.
- **Deterministic ordering:** Timeout events take priority over service-completion and route-hop events at the same tick. Arrival events are processed before timeout events at that tick.
- **Tests:** Covered timeout scheduling and validation, terminal requests, queued and in-flight timeouts, stale completion events, same-tick timeout/completion ordering, and topology-aware worker waiting queues.
- **Targeted tests — passed:** Timeout and event-queue regression tests, including `TestRoutedQueuedRequestTimesOutAndPreservesWorkerQueue`.
- **Standard tests — passed:** `go test ./...`.
- **Vet — passed:** `go vet ./...`.
- **Build — passed:** `go build ./...`.
- **Whitespace check — passed:** `git diff --check`. Git emitted LF-to-CRLF warnings, which did not cause the check to fail.
- **Limitations:** Race-detector verification remains blocked by the documented local compiler limitation. Hosted CI remains unverified because no Git remote is configured.

## Phase 2.4 — Request Cancellation

**Date:** 2026-10-15

### Implemented
- Added explicit request-cancellation events and scheduling validation.
- Added cancellation handling for queued and in-flight requests.
- Shared terminal-request cleanup logic between cancellation and timeout handling.
- Ensured cancelled requests leave waiting queues and release active capacity.
- Defined deterministic same-tick priority: arrival, timeout, cancellation, service completion, then route hop.
- Added tests for invalid cancellation schedules, terminal requests, queued and in-flight cancellation, routed worker queues, and cancellation versus completion at the same tick.

### Verification
- `go test ./internal/sim -count=1` — passed.
- `go test ./...` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — passed; Git emitted only Windows LF/CRLF conversion warnings.

### Limitations
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI has not been run because no Git remote is configured.

## Phase 2.5 — Simulation Metrics and Request Snapshots

**Date:** 2026-10-15

### Implemented
- Added read-only metrics for total, created, queued, in-flight, succeeded, failed, cancelled, and timed-out requests.
- Added aggregate service-capacity, busy-worker, waiting-request, and utilization metrics.
- Added per-worker metrics for topology-based simulations, including effective capacity, busy workers, waiting requests, and utilization.
- Made `Engine.Requests()` return independent request snapshots so callers cannot mutate engine-owned request lifecycle state through returned pointers.
- Added tests covering request lifecycle metrics, timeouts, cancellations, effective worker capacity, and snapshot independence.

### Verification
- `go test ./...` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — passed; Git emitted only Windows LF/CRLF conversion warnings.

### Limitations
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI has not been run because no Git remote is configured.

## Phase 2.6 - Worker Failure Injection and Recovery

**Date:** 2026-10-15

### Implemented
- Added worker failure and recovery events that identify workers independently of request IDs.
- Added scheduling validation for missing topology, invalid worker targets, non-worker nodes, and past timestamps.
- Defined deterministic event priority: arrival, timeout, cancellation, worker failure, worker recovery, service completion, then route hop.
- Worker failure marks the worker offline and fails its in-flight requests, releasing their active capacity.
- Requests already queued at a failed worker remain queued until recovery.
- Requests already in transit to a worker that fails before arrival wait in that worker's queue until recovery.
- Worker recovery resumes queued requests in FIFO order up to the worker's configured capacity.
- Filtered routing skips offline worker destinations when an available destination can be selected.
- Added worker online/offline state to per-worker metrics.
- Added regression tests for event validation, in-flight failure, stale completion events, FIFO recovery, routing around offline workers, requests arriving during an outage, and worker metrics.

### Verification
- `go test ./internal/sim -count=1` - passed.
- `go test ./...` - passed.
- `go vet ./...` - passed.
- `go build ./...` - passed.
- `git diff --check` - passed; Git emitted only Windows LF/CRLF conversion warnings.

### Limitations
- If all eligible outgoing destinations are unavailable, routing currently fails the request instead of waiting at the upstream node.
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.7 — Retry and Resilience Policies

**Date:** 2026-10-15

### Implemented
- Added configurable retry budgets and backoff delays for eligible request failures.
- Added retry-waiting lifecycle handling and deterministic retry events.
- Retried topology-routed requests through their original source after the backoff expires.
- Preserved retry behavior when the failed worker is offline, resuming retries when it becomes available.
- Added attempt tracking to distinguish service attempts and handle stale completion events safely.
- Covered retry success, exhausted retry budgets, zero backoff, offline-worker waiting, and cancellation during retry waiting.

### Verification
- Focused retry and worker-failure tests — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — passed.

### Limitations
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.8 — Circuit Breakers and Failure Isolation

**Date:** 2026-10-15

### Implemented
- Added configurable circuit-breaker policies with failure thresholds and cooldown durations.
- Added closed, open, and half-open circuit states for topology workers.
- Filtered routing to avoid offline workers and workers whose circuits are not accepting requests.
- Restricted half-open recovery to one probe request at a time.
- Reopened circuits when a probe is cancelled or times out, scheduling a new cooldown.
- Resumed queued work through the probe path after cooldown expiry, including when expiry occurs while the worker is busy.
- Added per-worker circuit state, consecutive-failure, and probe-in-flight metrics.
- Added tests for policy validation, default metrics, threshold-triggered opening, single-probe behavior, probe termination, queued-request resumption, and cooldown expiry during active service.

### Verification
- Focused circuit-breaker, worker-recovery, and worker-failure tests — passed.
- `go test ./internal/sim -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — passed; Git emitted only Windows LF-to-CRLF conversion warnings.

### Limitations
- If all eligible outgoing destinations are unavailable, routing currently fails the request instead of waiting at the upstream node.
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.9 — Simulation Observability and Scenario Reporting

**Date:** 2026-10-15

### Implemented
- Added deterministic run summaries with request lifecycle counts and terminal outcomes.
- Added a dedicated counter for requests waiting to retry, separate from queued requests.
- Added end-to-end latency statistics for successfully completed requests, including minimum, maximum, and average latency.
- Added service latency statistics that accumulate service time across attempts without counting retry backoff or downtime.
- Added retry counts derived from request attempt tracking.
- Excluded failed, cancelled, timed-out, and incomplete requests from successful-completion latency statistics.
- Added tests for summary determinism, latency calculations, retries, failure and timeout handling, cancellation, and service-time accumulation across attempts.

### Verification
- `go test ./internal/sim -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — passed; Git emitted only Windows LF-to-CRLF conversion warnings.

### Limitations
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.10 — Scenario Execution and Comparative Reports

**Date:** 2026-10-16

### Implemented
- Added reusable scenario definitions for request arrivals, worker failures and recoveries, timeouts, cancellations, retry policies, and circuit-breaker policies.
- Added scenario validation for configuration, duplicate request IDs, event timestamps, topology references, worker targets, and policy settings.
- Added deterministic scenario execution and structured results containing run summaries and metrics.
- Added comparative reports that calculate baseline and candidate metrics, including success rate, throughput, average end-to-end latency, retries, and worker utilization.
- Added comparison deltas to make changes between baseline and candidate scenarios explicit.
- Added per-worker comparisons containing baseline and candidate metrics, worker-presence indicators, and utilization deltas.
- Covered workers present in both scenarios, baseline-only workers, and candidate-only workers, including offline status and circuit-breaker state.

### Verification
- Focused scenario-comparison tests — passed.
- `go test ./internal/sim -run 'CompareScenariosIncludesPerWorkerMetrics' -count=1` — passed.
- `go test ./internal/sim -run 'CompareScenarios' -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed; Git emitted only Windows LF-to-CRLF conversion warnings.
- Per-worker comparison implementation committed as `97c86bd` (`feat(sim): compare per-worker metrics`).

### Limitations
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.11.1 — Experiment Suites

**Date:** 2026-10-16

### Implemented
- Added reusable experiment suites containing ordered scenario configurations.
- Added validation for suite names, minimum scenario count, duplicate scenario names, and invalid scenario configurations.
- Added deterministic execution of all scenarios in a suite.
- Compared every candidate scenario against the first scenario as the baseline.
- Added tests for valid and invalid suites, baseline comparisons, pre-execution validation, and deterministic repeated execution.

### Verification
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.
- Committed as `60eb04e` (`feat(sim): add experiment suites`).

### Limitations
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.11.2 — Service Capacity Sweeps

**Date:** 2026-10-16

### Implemented
- Added `SweepServiceCapacity` to generate an experiment suite from a baseline scenario and candidate service capacities.
- Preserved the baseline as the first scenario and generated uniquely named candidates for each requested capacity.
- Preserved the original scenario configuration while changing only the candidate name and service capacity.
- Added validation for invalid baselines, empty capacity lists, non-positive capacities, duplicate capacities, and capacities matching the baseline.
- Explicitly rejected topology-based scenarios because changing `ServiceCapacity` alone would not represent a topology worker-capacity sweep.
- Added tests for generated scenario names and capacities, configuration preservation, invalid inputs, and baseline immutability.

### Verification
- Focused service-capacity sweep tests — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.
- Committed as `46338a4` (`feat(sim): add service capacity sweeps`).

### Limitations
- Service capacity sweeps do not support topology-based scenarios.
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.11.3 — Parameter Sweep Reports

**Date:** 2026-10-16

### Implemented
- Added `SweepReport` and `SweepReportRow` for structured reporting of scenario sweep results.
- Included service capacity, duration, request outcomes, success rate, throughput, average end-to-end latency, retries, and worker utilization in each report row.
- Retained existing baseline comparison reports rather than duplicating comparison calculations.
- Combined scenario configuration with execution results through `BuildSweepReport`.
- Added validation for suite and result names, scenario counts and ordering, missing comparisons, non-positive durations, and scenarios without requests.
- Added tests for capacity sweep reporting, scenario ordering, report metrics, and invalid or mismatched results.

### Verification
- Focused sweep report tests — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.
- Committed as `45d3d2e` (`feat(sim): add parameter sweep reports`).

### Limitations
- Reports require the original experiment suite alongside its completed result so that configured service capacities can be reported accurately.
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.11.4 - JSON and CSV Export

**Date:** 2026-10-16

### Implemented
- Added `WriteSweepReportJSON` to serialize complete sweep reports as indented JSON.
- Preserved scenario ordering and deterministic string-map key ordering in JSON output.
- Added `WriteSweepReportCSV` with a stable 13-column header and one row per scenario.
- Used Go's CSV writer to correctly escape names containing commas, quotes, and newlines.
- Kept comparison and per-worker details in JSON; CSV contains scenario-level metrics.
- Added tests for JSON round-tripping, deterministic exports, CSV headers and escaping, empty reports, and writer-error propagation.

### Verification
- `go test ./internal/sim -run 'TestWriteSweepReport' -count=1` - passed.
- `go test ./... -count=1` - passed.
- `go vet ./...` - passed.
- `gofmt -l .\internal\sim\export.go .\internal\sim\export_test.go` - returned no unformatted files.
- `git diff --check` - passed.

### Limitations
- CSV exports scenario-level metrics only; full comparison and per-worker details are available in JSON.
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.12 — Capacity Sweep CLI

**Date:** 2026-10-16

### Implemented
- Added the `netarena sweep` command to run service-capacity experiments from the CLI.
- Added flags for candidate capacities, baseline capacity, service duration, request count, arrival interval, report format, and output path.
- Integrated the CLI with the existing experiment-suite, sweep-report, and export APIs.
- Added argument validation and error handling for report file creation, writing, and closing.
- Added CLI tests covering help output, JSON and CSV exports, invalid arguments, and output-file errors.
- Added root-level ignore rules for `sweep-report.json` and `sweep-report.csv`.

### Verification
- `go test ./cmd/netarena -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed with only Windows LF-to-CRLF warnings.
- Manual JSON and CSV CLI runs with candidate capacities `2,4,8` — passed; both reports contained the baseline and three candidate scenarios.
- `git check-ignore -v .\sweep-report.json .\sweep-report.csv` — confirmed both generated report files are ignored.

### Limitations
- Capacity sweeps use the existing non-topology scenario API.
- Race-detector verification remains blocked by the local 32-bit MinGW GCC toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.13 — CLI Usability and Documentation

**Date:** 2026-10-16

### Implemented
- Replaced the outdated foundation-only README with the current project status.
- Documented prerequisites, quick start, build, test, vet, and formatting commands.
- Documented the capacity-sweep CLI options and JSON/CSV export workflows.
- Added report structure examples, metric definitions, and simulation-time caveats.
- Documented project structure, source-of-truth documents, and known limitations.

### Verification
- `go run ./cmd/netarena` — passed; displayed CLI help.
- JSON sweep with candidate capacities `2,4` — passed; generated three scenarios including the baseline.
- CSV sweep with candidate capacities `2,4` — passed; generated three scenarios including the baseline.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed with only the Windows LF-to-CRLF warning.

### Limitations
- Clean-checkout verification and hosted CI remain pending because no Git remote is configured.
- Race-detector verification remains blocked by the local compiler toolchain.


## Phase 2.14.1 — Experiment Suite Benchmarking

**Date:** 2026-10-16

### Implemented

- Added `internal/sim/experiment_bench_test.go`.
- Added `BenchmarkRunExperimentSuite` for workloads of 100, 500, and 1,000 requests per scenario.
- Each suite contains a baseline and two capacity candidates, all using the same arrival schedule.
- Scenario construction and suite validation occur outside the timed benchmark section.
- Enabled allocation reporting for each benchmark.

### Benchmark environment

- OS/architecture: Windows/amd64.
- CPU: Intel Core i7-1165G7 @ 2.80 GHz.
- Command: `go test ./internal/sim -run '^$' -bench '^BenchmarkRunExperimentSuite$' -benchmem -count=3`.

### Results

Median of three runs:

| Workload | Time per suite | Bytes per suite | Allocations per suite |
|---|---:|---:|---:|
| 100 requests × 3 scenarios | 254,554 ns/op | 287,089 B/op | 1,982 allocs/op |
| 500 requests × 3 scenarios | 1,324,336 ns/op | 1,590,245 B/op | 9,230 allocs/op |
| 1,000 requests × 3 scenarios | 3,098,778 ns/op | 3,146,688 B/op | 18,284 allocs/op |

### Verification

- `go test ./internal/sim -run '^$' -bench '^BenchmarkRunExperimentSuite$' -benchmem -count=3` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.

### Limitations

- These measurements describe local synthetic workloads and are not production-performance claims.
- Benchmark timings vary with system load.
- No optimization was attempted or claimed in this milestone.
- Race-detector verification remains blocked by the local compiler toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.14.2 — Allocation Profiling and Queue Preallocation

**Date:** 2026-10-16

### Implemented

- Added an internal event-queue constructor that reserves initial storage.
- Updated `RunScenario` to reserve capacity for arrivals, worker failures,
  worker recoveries, timeouts, and cancellations.
- Kept the existing public `NewEventQueue` and engine constructors unchanged.
- Added tests covering zero, positive, and negative initial capacities.

### Verification

- `go test ./internal/sim -run 'TestEventQueue' -count=1` — passed.
- `go test ./internal/sim -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.

### Scenario benchmark

Environment: Windows/amd64, Intel Core i7-1165G7 @ 2.80 GHz.

Command:

`go test ./internal/sim -run '^$' -bench '^BenchmarkRunScenario$' -benchmem -count=3`

Pre-optimization and post-optimization measurements used the same `RunScenario` benchmark path. Post-optimization results below are medians of the latest three runs.

| Requests | Before (median) | After (median) | Runtime change | Allocation change |
|---:|---:|---:|---:|---:|
| 100 | 115.80 µs/op | 77.98 µs/op | 32.7% faster | 13.2% lower |
| 500 | 644.72 µs/op | 465.25 µs/op | 27.8% faster | 9.5% lower |
| 1,000 | 1,082.95 µs/op | 1,150.997 µs/op | 6.3% slower | 8.1% lower |

These are local synthetic benchmark measurements, not production performance claims. Runtime varies with system load, and the latest measurements do not show a consistent runtime improvement at every workload size. Allocation reductions were observed at all three sizes. The initial queue capacity reserves space for known initial events; dynamically generated events can still grow the heap.

### Limitations

- CPU and allocation profiles reflect synthetic workloads.
- Request snapshot copying and repeated scenario validation remain
  separate potential optimization areas.
- Race-detector verification remains blocked by the local compiler toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.14.3 — Reduce Redundant Scenario Validation

**Date:** 2026-10-16

### Implemented

- Added a regression test verifying that the public `RunScenario` function rejects invalid scenarios.
- Extracted the scenario execution body into the private `runValidatedScenario` helper.
- Preserved validation in `RunScenario`.
- Updated `RunExperimentSuite` to use the helper after the suite has passed validation, avoiding a second validation of every scenario.

### Verification

- `go test ./internal/sim -run 'TestRunScenarioRejectsInvalidScenario|TestRunExperimentSuite' -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.
- Experiment-suite and suite-validation benchmarks — passed.

### Benchmark environment

- OS/architecture: Windows/amd64.
- CPU: Intel Core i7-1165G7 @ 2.80 GHz.
- Baseline command: `go test ./internal/sim -run '^$' -bench '^(BenchmarkRunExperimentSuite|BenchmarkExperimentSuiteValidate)$' -benchmem -count=3`.
- Follow-up command: `go test ./internal/sim -run '^$' -bench '^(BenchmarkRunExperimentSuite|BenchmarkExperimentSuiteValidate)$' -benchmem -benchtime=2s -count=5`.

### Experiment-suite results

The baseline values are medians of three runs; follow-up values are medians of five runs.

| Requests per scenario | Baseline time | Follow-up time | Baseline bytes/op | Follow-up bytes/op | Baseline allocs/op | Follow-up allocs/op |
|---:|---:|---:|---:|---:|---:|---:|
| 100 | 265,644 ns | 269,587 ns | 249,721 | 239,233 | 1,964 | 1,955 |
| 500 | 1,247,571 ns | 1,463,810 ns | 1,447,276 | 1,365,364 | 9,206 | 9,197 |
| 1,000 | 2,674,208 ns | 3,029,532 ns | 2,905,407 | 2,741,582 | 18,257 | 18,242 |

### Interpretation and limitations

- Follow-up allocations decreased by approximately 4.2%–5.7% across the three workloads.
- Allocation counts decreased slightly at every workload size.
- The measured follow-up runtime was approximately 1.5% slower at 100 requests, 17.3% slower at 500 requests, and 13.3% slower at 1,000 requests compared with the recorded baseline.
- The baseline and follow-up runs used different benchmark durations, and the measurements showed runtime variability. These results do not establish a runtime improvement; the observed allocation reduction is the consistent benefit.
- Benchmarks use local synthetic workloads and are not production-performance claims.
- Race-detector verification remains blocked by the local compiler toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## Phase 2.14.4 — Profile Request Snapshot Copying

**Date:** 2026-10-16

### Implemented

- Added `BenchmarkEngineRequestsSnapshot` and `BenchmarkEngineSummary` for 100, 500, 1,000, and 5,000 requests.
- Captured CPU and allocation profiles for the request snapshot benchmark.
- The allocation profile attributed approximately 70.18% of sampled allocated volume to `Request.clone` and 29.71% to `Engine.Requests`.
- Updated `Engine.Summary()` to iterate over the engine-owned request registry directly instead of creating independent request snapshots.
- Preserved the public `Engine.Requests()` snapshot contract and its existing isolation test.

### Verification

- `go test ./internal/sim -run 'TestRunSummary|TestRequestsReturnsIndependentSnapshots' -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.
- Request snapshot and summary benchmarks — passed.

### Benchmark environment

- OS/architecture: Windows/amd64.
- CPU: Intel Core i7-1165G7 @ 2.80 GHz.
- Baseline snapshot command: `go test ./internal/sim -run '^$' -bench '^BenchmarkEngineRequestsSnapshot$' -benchmem -benchtime=1s -count=3`.
- Baseline summary command: `go test ./internal/sim -run '^$' -bench '^BenchmarkEngineSummary$' -benchmem -benchtime=1s -count=3`.
- Follow-up command: `go test ./internal/sim -run '^$' -bench '^BenchmarkEngine(RequestsSnapshot|Summary)$' -benchmem -benchtime=1s -count=3`.

### Results

Median of three runs. Times are per operation.

| Requests | `Requests()` time | `Requests()` bytes/op | `Requests()` allocs/op | `Summary()` time | `Summary()` bytes/op | `Summary()` allocs/op |
|---:|---:|---:|---:|---:|---:|---:|
| 100 | 7,226 ns | 14,696 B | 103 | 1,178 ns | 0 B | 0 |
| 500 | 38,232 ns | 83,304 B | 503 | 6,450 ns | 0 B | 0 |
| 1,000 | 107,069 ns | 166,608 B | 1,005 | 14,755 ns | 0 B | 0 |
| 5,000 | 696,788 ns | 778,434 B | 5,017 | 81,578 ns | 0 B | 0 |

### Interpretation and limitations

- Compared with the earlier summary benchmark medians, `Summary()` was approximately 86%–89% faster across the four workloads.
- `Summary()` reported zero bytes allocated and zero allocations per operation at every tested size.
- `Requests()` continues to allocate independent snapshots by design. Its behavior and measured allocation counts remained consistent with the baseline.
- Benchmark timings vary across runs, particularly at larger workloads. These results are local synthetic measurements, not production-performance claims.
- CPU and allocation profiles represent sampled costs accumulated over benchmark iterations, not memory retained at one time.
- Race-detector verification remains blocked by the local compiler toolchain.
- Hosted CI remains unverified because no Git remote is configured.

## HTTP API and Browser Workflow Milestone

**Date:** 2026-10-16

### Verification

- `go test ./... -count=1` — passed for the CLI, API, and simulation packages. Go package discovery also included `web/node_modules/flatted/golang/pkg/flatted`, which reported no test files.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `npm test` (from `web/`) — passed; 1 test file, 5 tests, including loading-state behavior.
- `npm run lint` (from `web/`) — passed.
- `npm run build` (from `web/`) — passed; TypeScript compilation and Vite production build completed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.
- Frontend tests cover defaults, successful submissions, API errors/retry, loading state, and arrival editing.

### Scope and limitations

- The Go API exposes health and simulation endpoints; the browser client submits scenarios and displays results or errors.
- Hosted CI, production deployment, persistence, authentication, full browser end-to-end coverage, and successful race-detector verification are not established by these checks.
- The local race-detector limitation and lack of configured Git remote remain as previously recorded.
- These checks establish a passing local build and test milestone, not production readiness or performance claims.

### Manual end-to-end smoke test

- Started the Go API server and confirmed `GET /api/v1/health` returned `status: ok`.
- Posted a `smoke-test` scenario directly to `POST /api/v1/simulations`; the API returned a duration of 10 ticks.
- Started the Vite development server and loaded the browser dashboard.
- Submitted `api-smoke-test` through the UI with service capacity 1, service duration 5, and two arrivals at ticks 0 and 1.
- Confirmed the UI displayed `COMPLETED`, 2 total requests, 2 succeeded, 0 failed, 0 cancelled, 7.00 average end-to-end latency, and 5.00 average service latency.
- Scope: one local happy-path scenario; this is manual smoke coverage, not automated browser E2E coverage.
- Metric clarification: the live API reports `workerUtilization: 0` after this scenario completes because the metric is a final-state snapshot of busy workers divided by service capacity. The earlier 75% value was not confirmed by the live API and should not be interpreted as average utilization over the simulation.

### Automated browser E2E verification

- Added Playwright with Chromium and a dedicated `npm run test:e2e` script.
- Configured Playwright to launch the Vite development server; the Go API must be running separately at `127.0.0.1:8080`.
- Added a browser test that submits the default scenario through the dashboard and verifies the successful API response, completion state, request counts, latency values, and final-state worker utilization.
- `npm run test:e2e` — passed, 1 test.
- `npm test` — passed, 5 unit tests; Vitest excludes the Playwright `e2e/` directory.
- `npm run lint` — passed.
- `npm run build` — passed.
- Playwright test results, reports, and traces are excluded by `web/.gitignore`.
- Scope: one automated Chromium happy-path test against the local Go API; this does not establish cross-browser coverage or production readiness.

### API Error-Response Contract Follow-up

**Date:** 2026-10-16

- Strengthened invalid simulation-request tests to verify the JSON content type and a non-empty `error` message, in addition to the HTTP status.
- Added a reusable `assertJSONError` test helper.
- Verified JSON error responses for oversized request bodies, unsupported endpoint methods, and unknown routes.
- `go test ./internal/api -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.

**Scope and limitations:** These checks verify the local API error-response contract. They do not establish hosted CI, production deployment, or successful race-detector verification.

### Reproducible Frontend Install and CI Readiness

**Date:** 2026-10-17

- `npm ci` (from `web/`) — passed using the committed `package-lock.json`; npm reported 0 vulnerabilities.
- `npm test` — passed, 5 tests.
- `npm run lint` — passed.
- `npm run build` — passed; TypeScript compilation and Vite production build completed.
- `npm run test:e2e` — passed, 1 Playwright Chromium test against the local Go API.
- The existing GitHub Actions workflow was extended with a frontend job covering dependency installation, unit tests, lint, production build, Chromium installation, and browser integration testing.
- The Go job retains formatting, vet, race-detector, and build checks.

**Scope and limitations:** Frontend checks passed locally after a clean dependency installation. The complete GitHub Actions workflow has not yet been verified on a hosted runner. Hosted race-detector results remain unverified, and local race-detector execution is still blocked by the available Windows compiler toolchain. No production deployment or production-readiness claim is made.


### Simulation Resource Limits and Event Budget

**Date:** 2026-10-17

- Added scenario limits for arrivals (10,000), aggregate scheduled events (30,000), topology nodes (1,000), topology connections (5,000), service capacity (10,000), and permitted service duration/scheduled ticks (1,000,000,000).
- Restricted retry policies to at most 100 retries.
- Added a runtime processed-event budget of 1,000,000 per engine, with a regression test confirming that the exhausted budget remains enforced across repeated `Run()` calls.
- Added tests for scenario resource limits, topology size limits, retry boundaries, aggregate scheduled events, and runtime event-budget exhaustion.

**Scope and limitations:** Resource limits are engineering guardrails, not measured capacity guarantees. The full local Go test suite, vet, and build passed after the latest event-budget regression test was added. Hosted CI and production readiness remain unverified.

### Frontend API Response Validation

**Date:** 2026-10-17

- Added runtime validation for successful simulation API responses, including the result, summary, latency metrics, simulation metrics, and per-worker metrics.
- Invalid successful responses now produce a clear `Invalid simulation response received from API` error instead of being returned as trusted simulation data.
- Preserved API-provided error messages for unsuccessful HTTP responses.
- Added direct API-client tests for valid responses, malformed response structures, non-object responses, and API error handling.
- `npm test` — passed, 9 tests across 2 files.
- `npm run lint` — passed.
- `npm run build` — passed, including TypeScript compilation and Vite production build.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.

**Scope and limitations:** Validation checks response structure and primitive types; it does not prove semantic consistency between metric values. No commit or push was performed.


### Frontend API Request Timeouts

**Date:** 2026-10-17

- Added a 30-second timeout to simulation API requests using `AbortController`.
- Requests are aborted when the timeout expires, and the timeout timer is cleared when the request finishes.
- Timeout failures return a clear error message while preserving the original error as the cause.
- Added a deterministic timeout test using Vitest fake timers.
- `npm test` — passed, 10 tests across 2 files.
- `npm run lint` — passed.
- `npm run build` — passed, including TypeScript compilation and Vite production build.

**Scope and limitations:** Timeout behavior is verified in frontend unit tests. A client-side timeout does not guarantee that the backend stops processing the simulation; server-side cancellation and resource cleanup must be verified separately. Hosted CI and production behavior remain unverified.

### Backend Simulation Cancellation and Execution Deadlines

**Date:** 2026-10-17

- Added `Engine.RunContext(ctx)` to allow simulation event processing to stop when its context is cancelled or its deadline expires. The existing `Run()` method remains available.
- Added `RunScenarioContext(ctx, scenario)` and preserved `RunScenario(scenario)` as a backwards-compatible wrapper.
- Updated the HTTP simulation handler to propagate request cancellation into the simulation engine.
- Added a configurable server-side simulation execution timeout, defaulting to 25 seconds. Execution deadline errors return HTTP `504 Gateway Timeout`; invalid scenarios continue to return HTTP `400 Bad Request`.
- Added tests for cancelled scenario contexts, cancelled HTTP requests, and HTTP execution timeouts.
- Restored the `EventRequestRetry` and unsupported-event handling cases during the `RunContext` refactor after regression tests exposed the missing retry-event processing.
- Targeted retry regression tests — passed, all 6 previously failing tests.
- `go test ./internal/api -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.

**Scope and limitations:** Context cancellation is cooperative and checked between simulation events; it does not interrupt an event handler midway through execution. The timeout and cancellation behavior has been verified locally. Hosted CI, production deployment, and production readiness remain unverified. No commit or push was performed.

### Mid-Execution Cancellation Regression Test

**Date:** 2026-10-17

- Added a deterministic test that cancels the simulation context after the engine processes its first event.
- Verified that `RunContext` returns an error matching `context.Canceled` and leaves the next scheduled event unprocessed.
- The test uses a custom context wrapper to trigger cancellation at a predictable check, avoiding sleep-based timing.
- Related cancellation and retry regression tests — passed.
- `go test ./internal/sim -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.

**Scope and limitations:** This verifies cooperative cancellation between events. It does not demonstrate interruption of an event handler while that handler is executing. No production-code changes, commits, or pushes were made for this test-only step.

### Concurrent API Simulation Requests

**Date:** 2026-10-17

- Added an in-process HTTP concurrency test for `POST /api/v1/simulations`.
- Tested 12 concurrent requests, each submitting a separate simulation with 20 arrivals.
- Verified successful HTTP responses, simulation-name isolation, and expected total and successful request counts.
- Fixed the test's start barrier to use a closed channel, ensuring all worker goroutines start together without a `WaitGroup` synchronization race.
- `go test ./internal/api -run TestRunSimulationHandlesConcurrentRequests -count=10 -timeout=30s` — passed, 10 consecutive runs.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.

**Scope and limitations:** This is a repeatable in-process concurrency correctness test, not a throughput benchmark or a guarantee of production capacity. It does not test network-level load, sustained traffic, or behavior under production deployment conditions. No production-code changes, commits, or pushes were made for this test-only step.

### Standalone HTTP API Load Benchmark

**Date:** 2026-10-17

- Added a standalone Go benchmark command under `cmd/netarena-bench/`, using only the Go standard library.
- Added configurable API URL, measured requests per concurrency level, arrivals per simulation, request timeout, and repetitions (`-runs`, default 3).
- Each concurrency level (1, 4, and 8) performs one warm-up request followed by the configured number of measured batches.
- Measures throughput, end-to-end HTTP latency through complete response-body reading, p50/p95 latency, HTTP errors, and invalid simulation responses.
- Aggregates throughput, p50, and p95 using medians across measured batches; error counts are cumulative across those batches.
- Added tests for percentile calculation, response validation, and batch-summary calculations, including HTTP failures, invalid responses, and zero elapsed time.
- Fixed response-body and JSON-decoding error classification.
- Invalid benchmark arguments now produce a non-zero process exit code; verified with `go run .\cmd\netarena-bench -requests 0` returning exit code 1.
- Failed requests print request-level diagnostics, including HTTP status or transport/body/JSON errors when available.
- `go test ./cmd/netarena-bench -count=1` — passed.
- `go vet ./cmd/netarena-bench` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Windows line-ending warnings only.

**Three-run local benchmark summary:** Each run used 20 measured requests per concurrency level and 20 arrivals per simulation against `127.0.0.1:18080`. Values below are medians across three measured batches per concurrency level.

| Concurrency | Median throughput (req/s) | Median p50 (ms) | Median p95 (ms) | HTTP errors across runs | Invalid results across runs |
|---:|---:|---:|---:|---:|---:|
| 1 | 1,061.93 | 0.59 | 1.35 | 0 | 0 |
| 4 | 2,639.92 | 1.16 | 2.51 | 0 | 0 |
| 8 | 4,521.72 | 1.15 | 3.09 | 0 | 0 |

**Scope and limitations:** All three measured batches at each concurrency level completed without HTTP errors or invalid simulation results. Concurrency 8 had the highest median throughput in this sample, but these are preliminary local measurements from a small workload, not production capacity figures. Results may vary with host load and runtime conditions. The benchmark does not test sustained load, remote network conditions, deployment infrastructure, or production traffic patterns. No commit or push was performed.

### Deterministic per-connection packet drops

**Date:** 2026-10-17

**Implementation**
- Added `Connection.DropPackets` to configure deterministic packet loss on an individual directed connection.
- The router preserves the setting and returns the selected connection's drop status to the simulation engine.
- When a request selects a connection with `DropPackets: true`, the request fails immediately with a connection-specific failure reason. This is a deterministic, all-packets-dropped fault model; probabilistic loss, retransmission, and packet-level TCP behavior are not implemented.
- The default `false` value preserves normal routing behavior.

**Regression coverage**
- Topology validation accepts a connection configured for packet drops.
- Router tests verify configuration preservation and returned drop status.
- Engine tests verify dropped-request failure, unchanged normal routing, and accumulated latency across a multi-hop route.

**Verification**
- `gofmt -w .\internal\sim\engine.go` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — passed; Git emitted Windows LF-to-CRLF warnings only.

**Limitations**
- Packet loss is currently a binary per-connection setting, not a configurable probability.
- A dropped request fails at the current simulation tick; connection latency is not incurred before the failure.
- Race-detector verification remains blocked by the local compiler limitation. Hosted CI remains unverified because no Git remote is configured.

### Seeded random-number generator foundation

**Date:** 2026-10-17

**Implementation**
- Added `NewEngineWithSeed` and `NewEngineWithTopologyAndSeed` constructors to initialize an engine-local pseudo-random number generator from an explicit `int64` seed.
- Added `Scenario.Seed` and wired scenario execution through the seeded constructors, for both topology and non-topology scenarios.
- Kept the existing engine constructors available for backward compatibility.

**Regression coverage**
- Engine tests verify that engines initialized with the same seed produce identical random-number sequences.
- A different-seed test verifies that the tested seeds produce different sequences.
- A topology-based constructor test verifies reproducibility with the same seed.
- A scenario test verifies identical results when the same scenario and seed are run twice.

**Verification**
- `gofmt -w .\internal\sim\engine_test.go` — passed.
- `go test ./internal/sim -run 'TestNewEngineWith(TopologyAnd)?Seed|TestNewEngineWithDifferentSeeds' -count=1` — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Git emitted Windows LF-to-CRLF warnings only.

**Limitations**
- The seeded generator is now used for probabilistic per-connection packet-loss decisions; see the subsequent "Probabilistic per-connection packet loss" section.
- Reproducibility depends on the same seed, inputs, and random-number call sequence. Changes to RNG consumption can change subsequent outcomes.
- Race-detector verification remains blocked by the local compiler limitation. Hosted CI remains unverified because no Git remote is configured.


### Probabilistic per-connection packet loss

**Date:** 2026-10-17

**Implementation**
- Added `Connection.PacketLossProbability` to configure probabilistic packet loss independently for each directed connection.
- Topology validation accepts probabilities from `0` through `1`, inclusive, and rejects negative values, values above `1`, `NaN`, and positive or negative infinity.
- The router carries the selected connection's loss probability and deterministic `DropPackets` setting to the engine through a shared link-selection helper.
- The engine uses its local RNG to evaluate probabilistic loss. Seeded constructors provide reproducible outcomes; legacy constructors initialize an RNG automatically.
- A zero probability consumes no random draw. Deterministically dropped packets and zero-probability links also avoid unnecessary RNG consumption.
- A dropped request fails immediately with a connection-specific failure reason. This models connection-level routing loss, not packet-level TCP retransmission or recovery.

**Regression coverage**
- Topology tests cover valid boundary and intermediate probabilities, negative values, values above `1`, `NaN`, and infinities.
- Engine tests verify zero-probability success, certain loss at probability `1`, seeded reproducibility, and probabilistic loss through the legacy constructor.
- Existing deterministic-drop and normal-routing tests continue to pass.
- Router selection logic is shared to avoid duplicating availability filtering and round-robin behavior.

**Verification**
- Focused packet-loss, seeded-RNG, and router tests — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Git emitted Windows LF-to-CRLF warnings only.

**Limitations**
- Probabilistic loss is applied when a request selects a connection; it does not model individual packets, TCP retransmission, or recovery.
- Reproducibility depends on the same seed, inputs, and random-number call sequence. Changes to RNG consumption can change subsequent outcomes.
- Race-detector verification remains blocked by the local compiler limitation. Hosted CI remains unverified because no Git remote is configured.

### FIFO and priority scheduling

**Date:** 2026-10-18

**Implementation**
- Added validated `SchedulingFIFO` and `SchedulingPriority` policies. FIFO remains the default for backward compatibility.
- Added request priorities. Higher integer values represent higher priority; existing arrival methods continue assigning priority `0`.
- Added priority-aware arrival methods for both standard and topology-aware engines.
- Added scenario-level scheduling policy configuration and per-arrival priority.
- Added stable priority insertion for basic-engine queues and topology worker queues. Higher-priority requests are selected first, and equal-priority requests retain FIFO ordering.
- Applied the same insertion behavior to normal queueing, offline-worker queues, and retry queueing.
- Prevented scheduling-policy changes after event processing begins, avoiding inconsistent ordering when a policy changes while requests are already queued.

**Regression coverage**
- Tests cover supported and unsupported policies, default FIFO behavior, priority-aware arrivals, stable ordering, and policy changes after processing begins.
- Scenario tests verify policy validation and observable behavior under contention.
- Routing tests verify priority ordering after worker recovery and preservation of priority across retries.
- Existing FIFO, recovery, retry, and simulation tests continue to pass.

**Verification**
- Focused scheduling, scenario, recovery, and retry tests — passed.
- `go test ./... -count=1` — passed.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — no whitespace errors; Git emitted Windows LF-to-CRLF warnings only.

**Limitations**
- Scheduling is non-preemptive: a running request is not interrupted when a higher-priority request arrives.
- Priority scheduling does not include aging or starvation prevention; sustained high-priority arrivals may delay low-priority requests.
- Race-detector verification remains blocked by the local compiler limitation. Hosted CI remains unverified because no Git remote is configured.

## Browser Workflow: Routing Algorithm and Route Visualization

**Date:** 2026-10-18

**Implemented**
- Added routing-algorithm selection and shortest-path configuration to the browser workflow.
- Added route-trace visualization for simulation results.
- Added client-side validation for topology node IDs, node types and capacities, connection references, latency, packet-loss probability, and duplicate directed connections.
- Required a network topology when shortest-path routing is selected.
- Added stable React keys for topology editor rows and node options, independent of editable node IDs.
- Added regression tests covering duplicate node IDs, topology node addition/removal and row identity, and shortest-path topology requirements.

**Frontend verification**
- `npm test -- --run` — passed; 2 test files and 14 tests passed.
- `npm run build` — passed; TypeScript compilation and Vite production build completed successfully.

**Verification scope and remaining limitations**
- The full Go test suite, `go vet ./...`, `go build ./...`, frontend tests, frontend production build, frontend lint, and `git diff --check` have passed.
- The architecture overview and testing strategy have been reviewed and updated for the implemented browser workflow.
- Full browser end-to-end verification, hosted CI, and successful race-detector verification remain unverified or blocked by the previously documented environment limitations.
- `go list ./...` discovers a third-party package under `web/node_modules/flatted/golang/pkg/flatted`. Its discovery has not been resolved; the Go checks passed despite it.

**Final verification update — 2026-10-18**

**Additional checks**
- `go test ./... -count=1` — passed across discovered Go packages.
- `go vet ./...` — passed.
- `go build ./...` — passed.
- `git diff --check` — passed with no whitespace errors. Git emitted LF-to-CRLF line-ending warnings on Windows.
- `npm run lint` — passed with no ESLint errors.

**Review notes**
- `go list ./...` also discovers `web/node_modules/flatted/golang/pkg/flatted`, a third-party package inside frontend dependencies. Its origin is confirmed; the reason for Go's discovery has not yet been resolved. The Go checks passed despite this.
- The working tree contains modified tracked files, untracked implementation files, and backup files. These have not been staged or committed.
- Hosted CI and race-detector verification remain unverified or blocked by the previously documented environment limitations.

**Browser end-to-end verification — 2026-10-18**

- Started the Go API locally with `go run ./cmd/netarena-server`.
- `Invoke-RestMethod http://127.0.0.1:8080/api/v1/health` — passed; returned `status: ok`.
- `npm run test:e2e` from `web/` — passed; 1 Playwright test verified scenario submission to the Go API and displayed simulation results.
- The initial E2E attempt failed because the API was not running. After starting the API, the test passed.

## Clean Checkout and Hosted CI Confirmation

**Date:** 2026-10-18
**Commit verified:** `e74beb0` (`Build NetArena simulation API and web dashboard`)

**Hosted verification**
- GitHub Actions workflow for the pushed `main` commit — passed, as confirmed in the GitHub Actions UI.
- Hosted CI includes Go formatting, `go vet`, race-enabled Go tests, Go build, frontend unit tests, lint, production build, and browser integration.

**Fresh-clone verification**
- Cloned `https://github.com/LohithaK25/NetArena.git` into `E:\NetArena-clean-checkout`; the clone checked out `e74beb0`.
- `go test ./cmd/... ./internal/... -count=1` — passed across the five application packages.
- `go vet ./cmd/... ./internal/...` — passed.
- `go build ./cmd/... ./internal/...` — passed.
- `npm ci` — passed; npm reported zero vulnerabilities.
- `npm test -- --run` — passed; 2 test files and 14 tests.
- `npm run lint` — passed.
- `npm run build` — passed.
- `npm run test:e2e` — passed; 1 Playwright test verified scenario submission to the Go API and display of simulation results.

**Race-detector limitation**
- Hosted GitHub Actions race-enabled tests passed as part of the successful workflow.
- Local Windows `go test -race ./cmd/... ./internal/...` remains blocked before tests execute because the installed MinGW compiler reports `cc1.exe: sorry, unimplemented: 64-bit mode not compiled in`.
- Local race-detector verification is therefore still outstanding; ordinary local test success is not treated as a substitute.

**Go package discovery**
- In the original working directory after frontend dependencies were installed, `go list ./...` discovered `web/node_modules/flatted/golang/pkg/flatted`.
- Explicit `./cmd/... ./internal/...` commands avoid that unrelated frontend dependency and passed locally.

## Scenario Objectives and Evaluation — Documentation reconciliation

**Date:** 2026-10-19
**Feature commit:** `b8c644e` (`feat: add scenario objectives and evaluation`)

**Implemented feature summary**

- Scenario configuration supports a minimum success-rate objective and a maximum average end-to-end latency objective.
- Objective evaluation is returned with simulation results and shown in the browser workflow.
- These are objective threshold checks, not a finalized overall game-scoring formula.

**Previously recorded feature verification**

- Go tests, `go vet`, and Go build — passed.
- Frontend tests — 18/18 passed across 2 test files.
- `npm run lint` and `npm run build` — passed.
- `git diff --check` — passed; Windows line-ending warnings were reported.
- These results are carried forward from the feature work; this documentation-only update does not rerun or independently re-establish them.

**Repository and CI status**

- The reviewed checkout was clean at `b8c644e`, tracking `origin/main`.
- Hosted GitHub Actions and clean-checkout verification are recorded above for commit `e74beb0`; hosted race-enabled tests passed in that workflow.
- Local Windows race-enabled tests remain blocked before test execution by the installed MinGW compiler's unsupported 64-bit mode.
- The Go module path remains `example.com/netarena`; migrating it is outside this documentation-only change.

**Documentation scope**

- Updated README, architecture overview, project handbook, testing strategy, and task tracker to reflect the implemented objective feature and recorded verification.
- Historical entries above are preserved rather than rewritten.
- No application code, dependencies, or API contracts were changed.
- Documentation diff reviewed; `git diff --check` reported no whitespace errors. Git reported LF-to-CRLF normalization warnings for affected files.
