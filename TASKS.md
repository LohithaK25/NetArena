# Current Tasks

## Phase 0 â€” Repository foundation
- [x] Create initial repository directory structure.
- [x] Copy the project handbook into `docs/PROJECT_PLAN.md`.
- [x] Add persistent AI-assisted development rules.
- [x] Add initial architecture and testing strategy documents.
- [x] Add verification log and repository README.
- [x] Inspect installed Git, Go, Node.js/npm, and Docker versions.
- [x] Review initial files and Git status; confirm no Git remote is configured.
- [x] Initialize the provisional Go module and add a minimal CLI with a focused test.
- [x] Establish Go 1.27.0 CI and reproducible local quality commands.
- [ ] Complete local race-detector verification with a compatible 64-bit C compiler.
- [x] Verify the workflow on GitHub and the instructions from a clean checkout after a remote and initial commit exist.

The module path `example.com/netarena` remains provisional. The repository remote is configured, hosted GitHub Actions passed, and clean-checkout verification is recorded in `docs/development/verification-log.md`.

## Phase 1 â€” Deterministic simulation foundation
- [x] Define domain types and request lifecycle.
- [x] Define virtual time and event ordering/tie-breaking.
- [x] Implement event priority queue.
- [x] Implement minimal request/service/resource simulation.
- [x] Add unit, invariant, deterministic replay, and benchmark tests.
- [x] Verify the first canonical scenario.
- [x] Add invariant and deterministic replay tests.
- [x] Add event queue and engine benchmarks.
- [x] Verify the canonical scenario.

## Phase 2 â€” Topology and Routing
- [x] Add topology model and validation.
- [x] Implement deterministic round-robin routing.
- [x] Integrate topology-aware request routing into the simulation engine.
- [x] Handle routing dead ends and cycles.
- [x] Test FIFO waiting under shared worker-capacity constraints.
- [x] Model independent capacity for each worker node.
- [x] Add configurable network-hop latency.
- [x] Add routing tests for multiple worker nodes and round-robin distribution.
- [x] Test worker-local FIFO queues and completion handoff failure handling.
- [x] Implement request deadlines and timeout events.
- [x] Ensure timeouts release active worker capacity and remove queued requests.
- [x] Define deterministic timeout priority when events share a tick.
- [x] Test stale events, same-tick completion races, and topology-aware waiting queues.
- [x] Implement explicit request cancellation scheduling and events.
- [x] Cancel queued and in-flight requests with correct queue and capacity cleanup.
- [x] Define deterministic cancellation priority for events at the same tick.
- [x] Test cancellation validation, routed queues, and same-tick event ordering.
- [x] Add read-only request lifecycle and service-capacity metrics.
- [x] Report effective per-worker capacity, queue depth, and utilization for routed simulations.
- [x] Return independent request snapshots to prevent external mutation of engine-owned request state.
- [x] Test lifecycle metrics for success, timeout, cancellation, and topology worker capacity.
## Working rules
- A task is not complete merely because code exists; verification evidence is required.
- Update this file and `docs/development/verification-log.md` when task status changes.
- Keep future work scoped to the current milestone unless a dependency requires otherwise.

## Phase 2.6 - Worker Failure Injection and Recovery
- [x] Add deterministic worker failure and recovery events.
- [x] Validate worker event scheduling and reject invalid worker targets and past timestamps.
- [x] Fail in-flight requests when their worker fails and release active capacity.
- [x] Preserve queued requests while a worker is offline.
- [x] Resume queued requests in FIFO order when a worker recovers.
- [x] Prevent new routing to offline workers and handle requests already in transit to failed workers.
- [x] Expose worker online/offline state through worker metrics.
- [x] Define deterministic event priority for worker failures and recovery.
- [x] Test worker event validation, failure handling, stale completion events, recovery, routing, and worker metrics.
- [x] Verify the implementation with tests, vet, build, and whitespace checks.

## Phase 2.11.4 - JSON and CSV Export
- [x] Add deterministic, indented JSON export for complete sweep reports.
- [x] Add CSV export with a stable 13-column header and one row per scenario.
- [x] Propagate encoding, CSV writing, and flushing errors.
- [x] Test JSON round-tripping, deterministic output, CSV escaping, empty reports, and writer failures.
- [x] Verify focused tests, the full test suite, static analysis, formatting, and whitespace checks.

## Phase 2.12 — Capacity Sweep CLI

- [x] Add the `netarena sweep` CLI command.
- [x] Support configurable candidate capacities, baseline capacity, service duration, request count, and arrival interval.
- [x] Export sweep reports to JSON or CSV at a user-specified path.
- [x] Validate command arguments and report file errors.
- [x] Add CLI tests for help, JSON/CSV exports, invalid arguments, and file errors.
- [x] Ignore generated root-level sweep report files.
- [x] Verify CLI integration with the full test suite and static analysis.
- [x] Verify the hosted CI workflow after configuring a Git remote.
- [x] Run race-detector verification in a supported environment (hosted GitHub Actions).

## Phase 2.13 — CLI Usability and Documentation

- [x] Replace the outdated foundation-only README with current implementation status.
- [x] Document prerequisites, quick start, build, tests, vet, and formatting checks.
- [x] Document capacity-sweep CLI options with JSON and CSV examples.
- [x] Explain report structure, metrics, and simulation-time interpretation.
- [x] Document project structure, architecture links, and known limitations.
- [x] Verify the documented CLI commands with real JSON and CSV exports.
- [x] Verify the full Go test suite, static analysis, and whitespace checks.
- [x] Verify the README instructions from a clean checkout after configuring a Git remote.

## Phase 2.14.1 — Experiment Suite Benchmarking

- [x] Add benchmarks for experiment-suite execution at 100, 500, and 1,000 requests per scenario.
- [x] Use three scenarios per suite with identical arrival schedules.
- [x] Measure execution time, bytes allocated, and allocation counts.
- [x] Run the benchmark three times per workload.
- [x] Verify the full Go test suite, static analysis, and whitespace checks.
- [x] Review the benchmark implementation and recorded results before committing.

## Phase 2.14.2 — Simulation Allocation Profiling and Queue Preallocation

- [x] Capture CPU and allocation profiles for experiment-suite execution.
- [x] Benchmark scenario validation separately.
- [x] Identify event-heap growth and request/snapshot allocations as profiling hotspots.
- [x] Add an internal event-queue constructor supporting initial capacity.
- [x] Reserve capacity for known initial scenario events.
- [x] Test initial queue capacity and preserve existing queue ordering behavior.
- [x] Verify the full test suite, static analysis, and whitespace checks.
- [x] Measure scenario performance before and after queue preallocation.
- [x] Re-run the cleaned scenario benchmark and record final measurements.

## Phase 2.14.3 — Reduce Redundant Scenario Validation

- [x] Add a regression test confirming `RunScenario` rejects invalid scenarios.
- [x] Extract scenario execution into a private helper for already-validated scenarios.
- [x] Preserve validation in the public `RunScenario` entry point.
- [x] Reuse suite validation in `RunExperimentSuite` to avoid validating each scenario twice.
- [x] Verify targeted tests, the full Go test suite, static analysis, and whitespace checks.
- [x] Benchmark experiment-suite execution and suite validation before and after the change.
- [x] Record allocation improvements and runtime trade-offs in the verification log.

## Phase 2.14.4 — Profile Request Snapshot Copying

- [x] Add benchmarks for `Engine.Requests()` snapshots and `Engine.Summary()` at 100, 500, 1,000, and 5,000 requests.
- [x] Profile snapshot allocation and CPU costs.
- [x] Optimize `Summary()` to read the internal request registry without creating snapshots.
- [x] Preserve independent snapshots returned by the public `Requests()` API.
- [x] Verify targeted tests, the full Go test suite, static analysis, and whitespace checks.
- [x] Benchmark both APIs before and after the summary optimization.
- [x] Review the final implementation and verification-log changes before committing.

## Phase 2.15 - HTTP API and Browser Workflow
- [x] Add the Go HTTP server entry point with configurable listen address.
- [x] Implement health and simulation API endpoints with request validation and JSON errors.
- [x] Add API handler tests for successful requests and invalid routes or methods.
- [x] Scaffold the React and TypeScript frontend under `web/`.
- [x] Add a scenario form for service capacity, service duration, and request arrivals.
- [x] Connect the frontend to the Go API and display simulation results and errors.
- [x] Add frontend tests for defaults, successful submissions, API errors, and arrival editing.
- [x] Re-run the complete Go test suite, `go vet`, and Go build.
- [x] Re-run frontend tests, lint, and production build.
- [x] Record final verification results in `docs/development/verification-log.md`.
- [x] Update architecture and testing documentation to reflect the implemented API and frontend.
- [x] Document API and frontend startup instructions in the README.
- [x] Add a README architecture diagram for the implemented API and browser workflow.
- [x] Capture and reference a real dashboard screenshot in the README.
- [x] Correct stale project-status and project-structure descriptions in the README.

## Phase 2.16 - Seeded Randomness and Probabilistic Packet Loss

- [x] Add engine-local random-number generation with seeded constructors while preserving legacy constructors.
- [x] Add scenario seed configuration and reproducible scenario execution.
- [x] Add configurable packet-loss probability to each directed connection.
- [x] Validate probability boundaries and reject invalid floating-point values.
- [x] Propagate connection loss settings through routing to the simulation engine.
- [x] Apply probabilistic loss while preserving deterministic packet-drop behavior.
- [x] Ensure zero-probability links do not consume random draws.
- [x] Test probability boundaries, certain loss, seeded reproducibility, and legacy constructor compatibility.
- [x] Verify the full Go test suite, static analysis, build, and whitespace checks.
- [x] Document implementation, regression coverage, and limitations in the verification log.

## Phase 3.1 - FIFO and Priority Scheduling

- [x] Add validated FIFO and priority scheduling policies, with FIFO as the default.
- [x] Add request priorities, with higher integer values representing higher priority.
- [x] Preserve existing arrival methods and default request priorities for compatibility.
- [x] Support explicit priorities on standard and topology-aware arrivals.
- [x] Add scenario-level scheduling policy configuration and arrival priorities.
- [x] Implement stable priority ordering in basic and worker waiting queues.
- [x] Preserve FIFO order among requests with equal priority.
- [x] Apply priority ordering to requests queued during worker outages and retries.
- [x] Reject scheduling-policy changes after event processing starts.
- [x] Test policy validation, default behavior, priority ordering, scenario execution, recovery, and retry behavior.
- [x] Verify the full Go test suite, static analysis, build, and whitespace checks.


## Phase 3.2 — Routing Algorithm and Route Visualization

- [x] Implement routing-algorithm selection in the browser workflow.
- [x] Add shortest-path routing configuration and route-trace visualization.
- [x] Validate topology node IDs and references, node types and capacities, connection latency, packet-loss probability, and duplicate directed connections.
- [x] Require network topology when shortest-path routing is selected.
- [x] Use stable React identities for topology editor rows and node options, independent of editable node IDs.
- [x] Add regression tests for duplicate node IDs, topology node editing/addition/removal, and shortest-path topology requirements.
- [x] Verify frontend tests and production build.
- [x] Complete final Go verification and review the overall working-tree status.
- [x] Finish the architecture/testing documentation review.

## Phase 3.3 — Scenario Objectives and Evaluation

- [x] Add configurable minimum success-rate and maximum average end-to-end latency objectives.
- [x] Evaluate configured objectives against simulation results and expose evaluation in API responses.
- [x] Display configured objectives and pass/fail evaluation in the browser workflow.
- [x] Add regression tests for objective validation, evaluation, API output, and frontend display.
- [x] Verify Go tests, static analysis, build, frontend tests, lint, and production build.
- [x] Record implementation and verification evidence in the development log.
