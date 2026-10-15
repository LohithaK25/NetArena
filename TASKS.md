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
- [ ] Verify the workflow on GitHub and the instructions from a clean checkout after a remote and initial commit exist.

The module path `example.com/netarena` is provisional until a canonical repository remote is configured. Local Go checks and exact tool versions are recorded in `docs/development/verification-log.md`; no hosted CI run is claimed.

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
## Working rules
- A task is not complete merely because code exists; verification evidence is required.
- Update this file and `docs/development/verification-log.md` when task status changes.
- Keep future work scoped to the current milestone unless a dependency requires otherwise.
