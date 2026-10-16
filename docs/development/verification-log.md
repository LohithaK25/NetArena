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
