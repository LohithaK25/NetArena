# NetArena — Project Handbook & Engineering Journal

> **Document status:** Active engineering handbook; implementation and verification records updated through the HTTP API and browser workflow milestone
> **Project status:** Deterministic simulation engine, Go CLI, HTTP API, and initial React/TypeScript browser workflow implemented; broader game and production capabilities remain planned
> **Purpose:** Single source of truth for the project vision, architecture, implementation decisions, testing, deployment, and the reasoning needed to explain the project in interviews.

---

## 1. How to use this document

This file is the durable project record. Keep it in the NetArena repository and update it as work progresses.

### Rules for keeping it trustworthy

1. **Separate planned, implemented, and verified work.** A feature is not complete because code was generated or written; it is complete only after its acceptance criteria are met and verification evidence is recorded.
2. **Never invent measurements.** Performance, capacity, coverage, and reliability claims must come from recorded tests.
3. **Record the reasoning.** For important choices, explain the problem, alternatives, decision, trade-offs, and evidence.
4. **Record failures as well as successes.** Preserve the symptoms, root cause, fix, and regression test.
5. **Update this handbook with each milestone.** Update the plan, task state, architecture decisions, testing record, and verification log as appropriate.
6. **Do not silently change the vision or stack.** Propose meaningful changes with a rationale and record the decision.
7. **Use the repository as the source of truth.** At the start of a new AI-assisted work session, read this file, `AGENTS.md`, `TASKS.md`, the architecture documents, and the latest verification log before changing code.

### Status vocabulary

- **Planned:** Agreed direction; not implemented.
- **In progress:** Work has begun but acceptance criteria are not all met.
- **Implemented:** Code exists, but verification may be incomplete.
- **Verified:** Acceptance criteria passed and reproducible evidence is recorded.
- **Blocked:** Progress depends on an unresolved issue.
- **Deferred:** Intentionally postponed, with a reason.

---

## 2. Project vision

### Working title

**NetArena — A Distributed-Systems Simulation and Strategy Game**

### One-sentence description

NetArena is a browser-based systems strategy game in which players build, operate, and defend simulated distributed infrastructure against traffic spikes, resource contention, network problems, service failures, and security incidents.

### The core idea

A player is responsible for keeping a simulated online service healthy. Requests arrive; services consume resources; queues grow; links introduce latency and packet loss; dependencies fail; and infrastructure costs money. The player must choose how to route traffic, allocate resources, scale services, configure caches, and respond to incidents while meeting service-level objectives.

The game should be fun to interact with, but the simulation must be technically meaningful. Its mechanics should demonstrate real computer science concepts rather than merely display attractive charts.

### Why build it?

The project is intended to complement TraceForge, which already explores backend workflows, persistence, idempotency, recovery, and reliability. NetArena should add a distinct systems-engineering angle: simulation, operating-system concepts, networking, algorithms, resource scheduling, concurrency, and eventually multiplayer correctness.

The project should demonstrate the ability to:
- Design and implement a coherent software system.
- Explain data structures, algorithms, state transitions, and trade-offs.
- Test correctness and failure behavior systematically.
- Measure performance instead of relying on guesses.
- Build an interactive frontend connected to a robust backend.
- Design for security, observability, and eventual multiplayer deployment.
- Explain every major design decision during technical interviews.

### Target audience

- Software engineering interviewers interested in backend, systems, platform, infrastructure, or full-stack engineering.
- Developers and students learning how distributed services behave under load and failure.
- Portfolio visitors who can understand the system through a playable demonstration.

### Product principles

1. **Correctness before complexity.**
2. **Simulation fidelity before visual polish.**
3. **Tests and documentation from the first milestone.**
4. **A playable single-player product before multiplayer expansion.**
5. **Measure before optimizing.**
6. **Security boundaries are explicit; clients are never trusted as authoritative.**
7. **Optional AI must add measurable value and must not be required for the core game to work.**
8. **Prefer a small number of justified technologies over an inflated stack.**

### Non-goals for the initial release

- A full production cloud provider or general-purpose orchestration platform.
- A packet-level emulator of every real-world networking protocol.
- Real attacks against real infrastructure.
- Real-money transactions or paid competitive mechanics.
- An LLM controlling the simulation without authorization or validation.
- Microservices, Kubernetes, message brokers, or other infrastructure added solely to look impressive.
- Claims of production readiness without security, load, recovery, and operational evidence.

---

## 3. Product concept and core gameplay

### Main player loop

1. **Build:** Place simulated components such as clients, load balancers, application services, caches, databases, and network links.
2. **Configure:** Set resource limits, routing policies, replica counts, scheduling policies, cache behavior, and defensive controls.
3. **Run:** Generate traffic and advance the simulation.
4. **Observe:** Inspect throughput, latency, queues, resource usage, errors, dropped requests, costs, and service objectives.
5. **Respond:** React to traffic spikes, failures, congestion, and security events.
6. **Evaluate:** Review the outcome, costs, bottlenecks, and replay of the scenario.
7. **Improve:** Change the architecture or configuration and try again.

### Initial playable scenario

Start with a small service topology:
- A traffic generator/client.
- One load balancer or routing component.
- Two application-server instances.
- One database.
- Links with configurable latency and capacity.
- A request generator with a fixed seed.
- A limited simulation duration and a clear objective.

The first scenario should let a player observe a bottleneck, make a configuration change, rerun the same workload, and compare outcomes.

### Initial metrics

The simulation should expose only metrics the engine actually models:
- Requests offered, accepted, completed, failed, and dropped.
- Throughput.
- End-to-end latency, including percentiles when sample volume makes them meaningful.
- Queue length and queue wait.
- CPU and memory utilization or modeled consumption.
- Network-link utilization, latency, and packet loss.
- Database connection usage and query/service time, if modeled.
- Estimated infrastructure cost.
- Availability/SLO attainment and score.

Document the exact definition of every metric. For example, define whether latency is measured from request arrival to completion, whether failed requests count in latency percentiles, and how dropped requests are represented.

### Scoring principles

Score the player's ability to meet stated objectives while controlling cost and failures. Avoid opaque scoring. Publish the formula and make it reproducible. Potential score inputs include:
- SLO attainment.
- Successful request completion.
- Recovery time after an incident.
- Infrastructure cost.
- Error and drop rates.
- Security objectives in scenarios that include them.

The scoring formula is **not yet finalized** and must be documented before competitive scoring is introduced.

---

## 4. Initial technology direction

These are the agreed starting choices, subject to validation during repository setup.

| Area | Initial choice | Why it fits | Important caution |
|---|---|---|---|
| Simulation engine | Go | Strong standard library, concurrency support, testing, race detector, profiling, and straightforward deployment | Keep simulation semantics deterministic; concurrency must not make outcomes nondeterministic |
| Application/API server | Go | Can share types and domain logic with the engine initially | Separate packages/modules before considering separate services |
| Frontend | React + TypeScript | Interactive topology editor, metrics, controls, and clear type contracts | Keep UI state separate from authoritative simulation state |
| Persistence | PostgreSQL | Durable users, match metadata, scenarios, scores, and replay metadata | Persist only what is necessary; define transaction boundaries |
| Real-time updates | WebSockets where justified | Suitable for game events and live state updates | Define message schemas, authentication, limits, reconnect, and ordering behavior |
| Observability | OpenTelemetry, with suitable metrics/logging backend | Standardized instrumentation and traces/metrics where useful | Instrument meaningful boundaries; do not add telemetry with no diagnostic purpose |
| Local environment | Docker / Docker Compose | Reproducible development and integration tests | Document resource needs and health checks |
| CI | Repository-hosted CI, chosen during setup | Repeatable quality gates on changes | Pin or constrain tool versions and make commands reproducible |
| Browser tests | Tool selected when frontend begins | Exercise real player journeys | Avoid making end-to-end tests the only testing layer |
| Load testing | Tool selected when service/API is ready | Measure throughput, latency, errors, and capacity | Publish workload shape, environment, and limitations |

### Technology selection rule

Before adding a new dependency or service, document:
- The problem it solves.
- Why the current stack is insufficient.
- Alternatives considered.
- Operational and testing costs.
- How success will be measured.

Do not add Redis, Kafka, Kubernetes, a separate AI service, or multiple microservices until a specific requirement justifies them.

---

## 5. Conceptual architecture

The initial architecture should be modular, not prematurely distributed.

### Main components

1. **Frontend**
   - Topology editor and visualization.
   - Scenario setup and game controls.
   - Live metrics and incident feed.
   - Results, replay, and comparison views.
   - Sends player actions to the backend; never decides authoritative scores.

2. **API and session layer**
   - Validates identity, authorization, request shape, and rate limits.
   - Creates and manages sessions.
   - Accepts player actions.
   - Sends state updates over a real-time channel when implemented.
   - Prevents cross-session access.

3. **Simulation domain**
   - Owns the authoritative simulation state.
   - Defines entities, resources, requests, links, queues, events, faults, and scoring.
   - Applies validated actions as state transitions.
   - Exposes a narrow interface to the API layer.

4. **Discrete-event engine**
   - Advances a virtual clock to the next scheduled event.
   - Processes events using a priority queue, initially a min-heap.
   - Uses an explicit tie-breaking rule for events with equal timestamps.
   - Produces deterministic results for the same initial state, seed, and ordered actions, within the documented determinism boundary.

5. **Persistence**
   - Stores durable user/session metadata, completed results, and replay/action records as needed.
   - Does not require every transient simulation event to be written synchronously to the database.

6. **Observability**
   - Captures application metrics, errors, structured logs, traces where useful, and simulation-specific metrics.
   - Keeps simulation measurements distinguishable from actual host/server measurements.

7. **Testing and performance harness**
   - Runs unit, property/invariant, deterministic replay, integration, concurrency, fault, security, browser, load, soak, and benchmark tests.
   - Produces repeatable evidence and reports.

### High-level flow

```text
Browser (React + TypeScript)
        |
        | HTTPS / WebSocket
        v
Go API + Authentication + Session Management
        |
        | validated actions / state queries
        v
Simulation Domain + Discrete-Event Engine
        |                  |
        |                  +--> Simulation metrics/events
        |
        +--> PostgreSQL (durable records and results)
        |
        +--> OpenTelemetry instrumentation

Automated test harness --> API / engine / database / browser
```

This is a conceptual design, not a claim that these components have already been implemented. Exact process boundaries, API routes, schemas, and deployment topology remain to be decided and documented as work proceeds.

---

## 6. Simulation-engine design principles

### Discrete-event simulation

Rather than updating every entity on every real-time frame, the engine schedules events at virtual times and advances to the next event. This can reduce unnecessary work when nothing changes between events.

Potential events:
- Request arrival.
- Request queued.
- Service work started.
- Service work completed.
- Network transmission started or completed.
- Timeout.
- Retry.
- Component failure or recovery.
- Cache expiry.
- Traffic-pattern change.

The event model must define how time is represented, how equal-time events are ordered, how invalid events are handled, and how cancellation or stale events work.

### Determinism

The engine should support repeatable scenarios using:
- An explicit random seed.
- A controlled virtual clock.
- Stable event ordering.
- Explicit action ordering.
- No dependence on wall-clock sleeps for simulation progress.
- Clear rules for concurrency and state mutation.

A deterministic engine can still be hosted by a concurrent server. The important boundary is that a given simulation session's state transitions are serialized or otherwise coordinated deterministically. Different sessions can be processed concurrently without sharing mutable simulation state.

### Resource accounting

Model resource consumption explicitly. Define capacity, usage, admission, queueing, rejection, and release behavior. Add invariants so resources cannot be released twice or consumed beyond permitted limits unless the scenario intentionally models overload.

### Network abstraction

Start with a documented abstraction for latency, bandwidth/capacity, packet loss, and routing. Do not claim packet-level TCP fidelity unless those mechanisms are actually implemented. A simplified model is acceptable if its assumptions and limitations are visible.

### Request lifecycle

Define legal request states, such as:

```text
Created -> Queued -> Processing -> Completed
                         |             |
                         v             v
                       Failed        (terminal)

Other possible terminal outcomes: TimedOut, Dropped, Cancelled
```

The final state machine must specify legal transitions, retries, duplicate events, timeout races, and how terminal requests are handled.

### Replays

A replay should preserve enough information to reproduce a scenario: scenario/configuration version, random seed, initial state, ordered player actions, and relevant engine version. The exact format and compatibility policy must be designed before replay persistence is considered complete.

---

## 7. Core CS concepts and planned implementation

| Concept | Planned mechanic | How it will be verified |
|---|---|---|
| Operating systems: scheduling | FIFO and priority scheduling first; other policies only when justified | Unit tests for ordering, fairness assumptions, and waiting-time calculations |
| OS: resource contention | CPU/worker capacity and queues | Invariants and scenarios with saturation |
| Concurrency | Concurrent independent sessions; controlled mutation inside each session | Race detector, stress tests, isolation tests |
| Networking: latency and loss | Link delay, capacity, and probabilistic loss | Seeded scenarios with expected aggregate behavior and invariant checks |
| Routing | Shortest-path or another explicitly chosen algorithm | Graph fixtures, unreachable paths, ties, and route-change tests |
| Data structures | Min-heap event queue, priority queue, LRU cache where appropriate | Ordering tests, property tests, and benchmarks |
| Distributed systems | Timeouts, retries, partial failure, replicas or leader behavior if implemented | Fault injection, recovery, and consistency tests |
| Databases | Connection capacity, query delay, index/cache effects where modeled | Scenario tests; distinguish modeled behavior from real PostgreSQL behavior |
| Security | Authentication, authorization, session isolation, rate limiting, input validation | Negative tests, cross-session tests, dependency/security scans |
| Cloud/DevOps | Containers, health checks, CI, deployment and rollback process | Clean-environment build and deployment smoke tests |
| Observability | Engine and server metrics, structured logs, traces where useful | Telemetry assertions and documented diagnostic scenarios |
| LLM (optional) | Evidence-grounded incident explanation and runbook retrieval | Evaluation set, citation/evidence checks, tool authorization and failure tests |

Do not add a concept solely to fill a resume checklist. It must fit the product, be implemented honestly, and have a test or demonstration that makes its behavior understandable.

---

## 8. Testing strategy — a dedicated workstream

Testing begins with the first engine package and continues through release. It is not a final phase.

### Layer A — Unit tests

Test small, deterministic components:
- Event-queue ordering and tie-breaking.
- Resource accounting and capacity rules.
- Request state transitions.
- Scheduling policies.
- Routing algorithms.
- Cache eviction.
- Scoring formulas.
- Input validation and authorization helpers.

Keep tests fast and focused. Each bug fix should add a regression test when practical.

### Layer B — Invariants and property-based tests

Check general truths over many generated inputs. Candidate invariants:
- Virtual time never moves backwards.
- Event ordering follows the documented timestamp/tie-breaking rule.
- Terminal requests cannot transition to a nonterminal state.
- A request is not counted as successfully completed more than once.
- Resource usage follows the documented accounting rules.
- A route does not use nonexistent edges.
- One session cannot mutate another session's state.
- Replaying identical inputs within the supported determinism boundary yields identical results.

Use Go fuzzing or property-style tests where appropriate. Generated inputs should be bounded so failures remain reproducible.

### Layer C — Deterministic replay tests

For each canonical scenario:
1. Set a fixed scenario/configuration version and seed.
2. Apply the same ordered actions.
3. Run the simulation.
4. Compare relevant final state, event output, and metrics.
5. Repeat after code changes to detect unintended behavior changes.

If an algorithm intentionally changes results, update golden fixtures only after reviewing and explaining the change.

### Layer D — Integration and contract tests

Verify the boundaries between modules:
- API validation and response schemas.
- Authentication and authorization.
- Session creation, joining, closing, and reconnecting.
- Database transactions and migrations.
- WebSocket message validation and error handling.
- Duplicate, malformed, stale, or unauthorized messages.
- Frontend/backend contract compatibility.

Use isolated test data and a reproducible database environment. Tests must clean up after themselves.

### Layer E — Concurrency and race tests

- Run Go's race detector for relevant packages and integration scenarios.
- Create many independent sessions and submit concurrent actions.
- Exercise simultaneous actions to the same session.
- Test session shutdown while actions or connections are active.
- Test shared caches, registries, and connection pools for safe ownership.
- Repeat stress tests to expose intermittent problems.

A race-detector pass is useful evidence but does not prove the absence of all concurrency bugs.

### Layer F — Fault injection and recovery

Inject:
- Engine/session worker failure.
- Database interruption.
- Slow or unavailable dependencies.
- WebSocket disconnects and reconnects.
- Delayed, duplicated, stale, and malformed messages.
- Simulation-level network loss and component outages.
- Timeouts and cancellation.

For every fault, specify the expected result: fail safely, retry, recover, reject, or terminate. Record what state should survive and what can be lost.

### Layer G — Security tests

- Authentication required for protected actions.
- Authorization enforced per session and resource.
- Cross-session reads and writes denied.
- Input size and rate limits enforced.
- Malformed payloads rejected safely.
- Secrets excluded from logs and repository history.
- Database queries parameterized through the chosen library.
- Dependency and container image scans run in CI when available.
- LLM tool permissions and output validation tested if the copilot is added.

The simulator's security incidents are game mechanics; the security of the actual web application is a separate responsibility.

### Layer H — Frontend and end-to-end tests

Test key user journeys:
- Load the application and start a scenario.
- Build or configure a topology.
- Start, pause, and reset a simulation.
- Change a configuration and compare results.
- Encounter and resolve an incident.
- Finish a scenario and inspect the result.
- Disconnect and reconnect during multiplayer testing.
- Verify useful error states when the backend is unavailable.

Prefer stable selectors and deterministic test fixtures over brittle timing assumptions.

### Layer I — Performance, load, and soak tests

Define and record:
- Workload shape and arrival rate.
- Scenario complexity and number of events.
- Number of concurrent sessions and clients.
- Test environment, machine specifications, software versions, and configuration.
- Warm-up period and test duration.
- Throughput, p50/p95/p99 latency where meaningful, errors, CPU, memory, and resource exhaustion.
- Whether the test measures simulation speed, API latency, or end-to-end player latency.

Run:
- Baseline benchmarks.
- Increasing-load tests.
- Stress tests beyond expected capacity to observe graceful failure.
- Soak tests to expose leaks and degradation over time.
- Regression benchmarks after performance-sensitive changes.

Never state that NetArena supports a particular number of players or sessions until a reproducible test demonstrates it. A benchmark result is valid only for its documented environment and workload.

### Layer J — Release verification

Before a release candidate:
- Clean build succeeds.
- Automated test suites pass.
- Migrations work on a clean database.
- Container health checks and configuration are verified.
- Security checks have no unresolved release-blocking findings.
- End-to-end smoke tests pass.
- Backup/restore or data-recovery expectations are documented where relevant.
- Load and soak results satisfy the project's defined release criteria.
- Known limitations and rollback steps are documented.

### Test evidence format

For each meaningful test run, record:
- Date and commit identifier.
- Command and environment.
- Test suite/workload.
- Result and duration.
- Metrics or artifact location.
- Failures and follow-up issue.
- Whether the run is a local check, CI run, benchmark, or release gate.

---

## 9. Milestone roadmap and acceptance criteria

All milestones below are **planned**, not completed.

### Phase 0 — Project charter and repository foundation

**Purpose:** Establish continuity, reproducibility, and boundaries before feature development.

Tasks:
- Create repository and initial branch.
- Add this handbook and supporting project documents.
- Add `AGENTS.md` with Codex instructions.
- Create `TASKS.md` and verification log.
- Decide repository layout and initial tool versions.
- Add formatting/lint/test commands and minimal CI.
- Record exact local prerequisites and setup instructions.

Acceptance:
- A clean checkout can follow the documented setup.
- The initial build/test commands are reproducible.
- No feature is marked verified without evidence.
- The initial architecture and test strategy are committed.

### Phase 1 — Deterministic simulation foundation

**Purpose:** Prove the core model before adding UI complexity.

Tasks:
- Implement domain types for services, requests, resources, links, and events.
- Implement virtual time and the event priority queue.
- Define event ordering and request lifecycle.
- Implement a minimal traffic generator and service processing.
- Implement resource capacity and queue behavior.
- Add seeded randomness where required.
- Add a small canonical scenario and replay-style test.
- Add unit, invariant, and benchmark harnesses.

Acceptance:
- Same seed, configuration, and ordered actions produce repeatable results within the documented boundary.
- Core invariants are tested.
- Queue and event behavior is documented.
- The first scenario can demonstrate a measurable bottleneck.
- Relevant tests pass with recorded commands and output.

### Phase 2 — Playable single-player vertical slice

**Purpose:** Make the engine understandable and fun to use.

Tasks:
- Create React + TypeScript frontend.
- Build topology visualization and configuration controls.
- Connect the UI to the Go backend.
- Show actual simulation state and metrics.
- Add start, pause, reset, and run controls with well-defined semantics.
- Add scenario objectives and transparent scoring.
- Add a results/comparison view.
- Add frontend unit/component and end-to-end tests.

Acceptance:
- A player can complete one scenario through the browser.
- UI metrics are derived from backend simulation output.
- Invalid actions produce clear errors.
- A clean environment can build and run the vertical slice.
- The main player journey is automated in an end-to-end test.

### Phase 3 — Computer-science mechanics

**Purpose:** Add meaningful depth without losing correctness.

Potential order:
1. Multiple scheduling policies.
2. Configurable network latency, capacity, and loss.
3. Routing algorithm and route visualization.
4. Cache policy and invalidation behavior.
5. Failure injection and recovery.
6. Rate limiting and defensive controls.
7. Additional database/replication mechanics if they fit the model.

Acceptance for each mechanic:
- Assumptions and limitations documented.
- Behavior demonstrated in a scenario.
- Unit/invariant and regression tests added.
- Metrics reflect the mechanic.
- Player-facing explanation is understandable.

### Phase 4 — Multiplayer foundations

**Purpose:** Introduce multiple players and sessions only after single-player state transitions are reliable.

Tasks:
- Define the multiplayer model: cooperative, competitive, or both; shared versus separate worlds; player roles and permissions.
- Define session ownership and authoritative state.
- Specify message schemas, sequencing, idempotency, reconnect, and error handling.
- Implement authentication and authorization.
- Implement WebSocket lifecycle and session isolation.
- Decide which state is persisted and what reconnect restores.
- Add concurrency, race, disconnect, duplicate-message, and isolation tests.
- Define workload and release capacity targets.

Acceptance:
- Concurrent sessions are isolated.
- The server validates every state-changing action.
- Reconnect behavior is documented and tested.
- Duplicate or stale messages are handled predictably.
- Race tests and stress tests have recorded evidence.
- Capacity claims are based on repeatable load tests.

### Phase 5 — Observability and scale

**Purpose:** Understand and improve actual system behavior.

Tasks:
- Add structured logs, application metrics, and traces where useful.
- Add simulation-specific metrics separately from host metrics.
- Add repeatable load and soak workloads.
- Profile CPU and memory.
- Identify and fix measured bottlenecks.
- Add resource limits, graceful overload handling, and operational dashboards as needed.
- Document tested capacity and limits.

Acceptance:
- Important failure paths can be diagnosed from telemetry.
- Performance results are reproducible and include environment details.
- Optimization claims compare equivalent workloads.
- Known capacity limits and overload behavior are documented.

### Phase 6 — Deployment and portfolio release

**Purpose:** Deliver a public, maintainable, demonstrable product.

Tasks:
- Select a hosting environment based on actual requirements and cost.
- Configure secrets and production environment variables safely.
- Configure TLS, access control, health checks, logging, and telemetry.
- Automate build, test, and deployment steps.
- Add database migrations and backup/restore expectations.
- Add smoke tests, rollback instructions, and an incident runbook.
- Create architecture diagrams, quick start, demo scenarios, screenshots, and a limitations section.
- Record the exact release commit and verification evidence.

Acceptance:
- The documented deployment process works from a clean state.
- The deployed application passes smoke tests.
- Release gates pass.
- Security, capacity, and operational limitations are honestly documented.
- Public demo and README do not expose secrets or internal credentials.

### Phase 7 — Optional LLM incident copilot

**Purpose:** Add AI only if it improves the learning and diagnostic experience.

Tasks:
- Define supported questions and evidence requirements.
- Build retrieval over curated project documentation/runbooks.
- Provide only authorized, constrained tools.
- Supply structured telemetry rather than allowing invented measurements.
- Require recommendations to identify supporting evidence and uncertainty.
- Evaluate against a curated incident question set.
- Test hallucination, prompt injection, tool authorization, and unavailable-provider behavior.
- Ensure the game remains usable without the LLM.

Acceptance:
- Core gameplay works when the model provider is unavailable.
- Recommendations are grounded in actual state and documented sources.
- Unauthorized actions cannot be executed.
- Evaluation results and limitations are documented.
- The feature's latency and operating cost are understood.

---

## 10. Multiplayer model: decisions to make before implementation

These are open questions; do not silently assume answers.

- **Mode:** cooperative incident response, competitive infrastructure optimization, or both?
- **World model:** do players share one infrastructure, or does each player manage an independent topology?
- **Authority:** the server owns and validates all authoritative state.
- **Timing:** is the simulation paused/turn-based, continuously advancing, or configurable by scenario?
- **Reconnect:** how long can a disconnected player return to the same session?
- **Persistence:** which actions, snapshots, scores, and replay data survive a restart?
- **Consistency:** what ordering guarantees are required for actions within a session?
- **Capacity:** what initial workload should the release support, and what evidence is required?
- **Abuse prevention:** how are message size, action frequency, session creation, and resource consumption bounded?

Choose the simplest model that supports the intended game. Multiplayer complexity should be justified by a clear product benefit.

---

## 11. Repository layout proposal

The layout can evolve, but module boundaries should remain clear.

```text
netarena/
├── AGENTS.md
├── README.md
├── TASKS.md
├── go.mod
├── go.sum
├── Makefile
├── docker-compose.yml
├── .github/
│   └── workflows/
├── cmd/
│   └── netarena/
├── internal/
│   ├── api/
│   ├── auth/
│   ├── session/
│   ├── simulation/
│   │   ├── engine/
│   │   ├── events/
│   │   ├── model/
│   │   ├── network/
│   │   ├── scheduler/
│   │   └── scoring/
│   ├── persistence/
│   └── observability/
├── web/
│   ├── src/
│   └── tests/
├── tests/
│   ├── integration/
│   ├── e2e/
│   ├── load/
│   └── scenarios/
├── benchmarks/
├── deploy/
├── docs/
│   ├── PROJECT_PLAN.md
│   ├── architecture/
│   ├── decisions/
│   ├── development/
│   ├── testing/
│   │   └── TESTING_STRATEGY.md
│   ├── operations/
│   └── interview-guide/
└── scripts/
```

This is a proposal, not a requirement to create every directory immediately. Add folders when the corresponding work exists. Keep the Go module layout idiomatic and avoid unnecessary abstraction.

---

## 12. Architecture decision records (ADRs)

Create one short ADR for each significant architectural decision. Use files such as `docs/decisions/ADR-0001-discrete-event-engine.md`.

Each ADR should contain:
- **Status:** Proposed, Accepted, Superseded, or Rejected.
- **Context:** What problem are we solving?
- **Decision:** What are we choosing?
- **Alternatives:** What else did we consider?
- **Rationale:** Why is this option appropriate?
- **Consequences:** Benefits, costs, limitations, and risks.
- **Verification:** How will we test or validate the decision?
- **Revisit conditions:** What evidence would justify changing it?

Likely ADR topics:
- Discrete-event simulation versus fixed time-step simulation.
- Determinism and random-seed design.
- Simulation state ownership and concurrency model.
- Initial networking abstraction.
- REST and WebSocket responsibilities.
- Persistence and replay format.
- Multiplayer session model.
- Observability and performance-test strategy.
- Hosting and deployment model.

Do not create an ADR just for every small coding choice. Use them for decisions that affect architecture, correctness, operations, or future flexibility.

---

## 13. Development workflow

For every bounded task:

1. **Read:** Inspect the plan, relevant code, tasks, and recent verification history.
2. **Define:** State the goal, scope, acceptance criteria, and non-goals.
3. **Design:** Explain the intended approach and relevant trade-offs before substantial changes.
4. **Implement:** Make a small, coherent change.
5. **Test:** Run focused tests, then broader relevant suites.
6. **Inspect:** Review the diff, error paths, concurrency risks, and unintended changes.
7. **Document:** Update comments, API contracts, architecture docs, task state, and decisions as appropriate.
8. **Verify:** Record actual commands and results.
9. **Summarize:** State what changed, why, tests run, limitations, and the next task.

### Codex working rules

- Do not ask Codex to build the whole project in one prompt.
- Give one bounded milestone or task at a time.
- Ask it to inspect existing files before editing.
- Require a proposed plan for architectural changes.
- Require tests for new behavior and regressions.
- Require it to report commands run and actual outcomes.
- Review diffs; do not accept unexplained rewrites or dependency additions.
- Do not mark a task complete based solely on a model's summary.
- If a test cannot be run, record it as unverified and explain why.
- Keep credentials, tokens, and personal secrets out of source control.
- Never let an AI-generated response override these project documents without an explicit decision.

### Start-of-session prompt

Use this when resuming work in a new chat or Codex session:

> We are continuing NetArena. Before making changes, read `AGENTS.md`, `docs/PROJECT_PLAN.md`, `TASKS.md`, `docs/architecture/overview.md` if present, `docs/testing/TESTING_STRATEGY.md` if present, and the latest verification log. Inspect the repository and current Git state. Summarize the current verified milestone, architecture, known issues, and next task. Do not assume a task is complete unless the repository and verification evidence support it. Preserve the agreed project vision and technology direction. Propose a bounded plan with acceptance criteria, implement only that scope, run relevant tests, inspect the diff, and update documentation and verification records. Do not invent test results or performance claims. If a decision conflicts with the project plan, stop and explain the conflict rather than silently changing direction.

---

## 14. Challenges and solutions journal

This section records actual engineering challenges as they occur. The entries below are a template, not claims that these issues have already happened.

For each challenge, record:
- **ID and date**
- **Phase and commit**
- **Observed behavior**
- **Expected behavior**
- **Reproduction steps**
- **Root cause** (confirmed or still suspected)
- **Options considered**
- **Chosen fix and why**
- **Tests added or changed**
- **Verification evidence**
- **Trade-offs or remaining limitations**
- **What was learned**

| ID | Challenge | Status | Root cause | Solution | Verification |
|---|---|---|---|---|---|
| — | No implementation challenges recorded yet | Planned work only | Not applicable | Record the first real issue here | Not applicable |

---

## 15. Implementation and verification log

Append an entry for each meaningful milestone or verification run.

### Entry template

```text
Date:
Commit:
Phase / task:
Goal:
Files changed:
Design decision(s):
Implementation summary:
Commands run:
Tests passed:
Tests failed / skipped:
Performance results (workload and environment):
Known limitations:
Follow-up tasks:
Status: Planned / In progress / Implemented / Verified / Blocked
```

### Initial baseline

- **Project planning:** Documented in this handbook.
- **Repository:** Existing Go repository with CLI, simulation engine, HTTP API, and browser client.
- **Code implementation:** Simulation engine, CLI, HTTP API, and initial browser workflow implemented; consult the task tracker and architecture overview for scope.
- **Tests:** Go simulation/API tests and frontend workflow tests implemented; see the verification log for actual command results.
- **Benchmarks:** Do not claim performance results without reproducible measurements recorded in the verification log.
- **Deployment:** Local API/browser workflow established; hosted deployment and production readiness are not verified.
- **Open next step:** Complete milestone verification, update startup documentation, and continue the planned roadmap without overstating unfinished capabilities.

Do not change the initial baseline to a completed status until actual repository evidence exists.

---

## 16. Interview-learning record

The goal is to understand every important part well enough to explain it without relying on generated answers.

For each major feature, eventually document:

1. What problem does it solve?
2. What is the simplest correct approach?
3. Which data structures and algorithms are used?
4. What are the time and space complexity implications?
5. What alternatives were considered?
6. What failure cases exist?
7. How is correctness tested?
8. How is performance measured?
9. What trade-offs did the implementation make?
10. What would change at a much larger scale?

Create focused notes under `docs/interview-guide/`, including:
- Simulation and event scheduling.
- Concurrency and race conditions.
- Networking and queueing.
- Distributed-systems failure modes.
- API and database design.
- Security and authorization.
- Observability and profiling.
- Test strategy and benchmark interpretation.
- Deployment and operational limitations.

After each milestone, practice explaining the implementation in plain language, drawing its data flow, and defending at least one design trade-off.

---

## 17. README and portfolio release checklist

Before presenting NetArena publicly, ensure the README includes:
- Clear project summary and why it exists.
- Screenshot or short demo.
- Architecture diagram.
- Feature list distinguishing implemented features from planned ones.
- Quick-start instructions tested from a clean environment.
- Required software versions and configuration.
- Example scenario and how to play.
- Testing commands and a truthful test summary.
- Benchmark methodology and results, if available.
- Security notes and deployment instructions.
- Known limitations and what is intentionally simplified.
- Roadmap and contribution/development guidance.

Never present planned features as implemented. Avoid unsupported claims such as “production-ready,” “handles thousands of players,” or “fully realistic networking.”

---

## 18. Risk register

| Risk | Why it matters | Mitigation |
|---|---|---|
| Scope grows too quickly | Delays a working product and makes testing difficult | Deliver a vertical slice before advanced mechanics |
| Simulation is visually convincing but inaccurate | Undermines technical credibility | Document assumptions, define invariants, and test behavior |
| Nondeterministic outcomes | Makes bugs hard to reproduce and replay unreliable | Seed randomness, control event ordering, and serialize session state transitions |
| Multiplayer arrives too early | Introduces race, isolation, and recovery bugs before core behavior is stable | Complete and test the single-player engine first |
| Excessive infrastructure | Adds operational work without improving the product | Require a documented reason for every major dependency |
| AI hides weak engineering | The project looks like an API wrapper | Make the engine and its tests valuable without AI |
| Unsupported benchmark claims | Damages trust in interviews | Publish reproducible workload and environment details |
| Long-chat context loss | Decisions and progress can drift | Maintain this handbook, task list, ADRs, and verification log in Git |
| AI-generated code is not understood | Makes maintenance and interviews difficult | Review changes, add learning notes, and explain every subsystem |
| Deployment costs or complexity rise | Can block a public demo | Choose hosting only after workload and operating requirements are known |

---

## 19. Current decisions and open questions

### Agreed direction

- Project: NetArena.
- Product: distributed-infrastructure simulation and strategy game.
- Initial engine/server language: Go.
- Frontend: React + TypeScript.
- Persistence: PostgreSQL.
- Containerization: Docker.
- Observability direction: OpenTelemetry.
- Core focus: operating systems, networks, algorithms, distributed systems, security, resource management, and scaling.
- Testing: dedicated workstream from day one.
- Multiplayer: planned after a reliable single-player foundation.
- LLM: optional incident copilot after the core product works.
- Development style: bounded milestones, documented decisions, reproducible tests, and honest verification.

### Not yet decided

- Repository location and remote hosting.
- Exact Go and Node.js versions.
- Package manager and frontend tooling versions.
- API endpoints and message schemas.
- Exact event representation and equal-timestamp ordering policy.
- Initial network and CPU scheduling models.
- Scoring formula and scenario difficulty.
- Multiplayer mode and state-sharing model.
- Authentication provider and deployment host.
- Load-testing tool and release capacity target.
- LLM provider, retrieval design, and evaluation dataset.

Resolve these when the relevant implementation needs them; do not make unnecessary choices prematurely.

---

## 20. Definition of project success

NetArena succeeds as a portfolio project when it is more than a screenshot or a large repository. It should have:

- A playable, understandable scenario.
- A documented and testable simulation model.
- Meaningful CS mechanics with explicit assumptions.
- Reproducible test commands and credible verification evidence.
- A dedicated multiplayer correctness and load-testing strategy.
- Measured performance results with reproducible workloads.
- A secure, documented deployment process for the released scope.
- An architecture and decision history that explain *why* the system is built this way.
- Clear limitations and a roadmap.
- An engineer who can explain and defend the implementation.

**The most important rule:** the code, tests, documentation, and claims must agree with one another. We will prefer a smaller system we can thoroughly explain and verify over a larger system whose behavior we do not understand.
