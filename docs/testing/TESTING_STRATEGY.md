# NetArena Testing Strategy

**Status:** Test strategy established; the Go CLI has a focused unit test, and the initial Go CI checks are defined.

The CLI unit test verifies its startup output. It is not evidence for simulation, API, persistence, frontend, or concurrency behavior; those tests remain future work.

Testing is a dedicated workstream from the first engine package through deployment.

## Test layers

1. **Unit tests:** event queue, state transitions, scheduling, resource accounting, routing, cache policies, scoring, validation.
2. **Invariant/property tests:** virtual time does not move backwards; terminal requests do not become active again; resources follow accounting rules; routes use valid edges; sessions remain isolated.
3. **Deterministic replay tests:** fixed configuration, seed, and ordered actions produce repeatable outcomes within the documented determinism boundary.
4. **Integration/contract tests:** APIs, schemas, database transactions/migrations, authentication, session lifecycle, WebSocket messages.
5. **Concurrency/race tests:** Go race detector, simultaneous actions, session shutdown, concurrent sessions, shared-state ownership.
6. **Fault/recovery tests:** dependency failures, disconnect/reconnect, timeouts, duplicated/stale messages, worker failure, simulation-level component outages.
7. **Security tests:** authorization, cross-session access denial, malformed and oversized payloads, rate limits, secret hygiene, dependency/container scans.
8. **Frontend/E2E tests:** scenario creation, configuration, run/pause/reset, incident handling, results, reconnect/error states.
9. **Performance/load/soak tests:** defined workloads, concurrent sessions, throughput, latency percentiles, CPU/memory, overload behavior, sustained degradation.
10. **Release verification:** clean build, migrations, health checks, smoke tests, security findings, recovery/rollback documentation, load evidence.

## Rules

- Add regression tests for fixed bugs where practical.
- Make random behavior seedable and testable.
- Record exact commands, environment, commit, duration, results, failures, and artifacts.
- Performance claims require reproducible workloads and environment details.
- A race-detector pass is useful but does not prove all concurrency behavior correct.
- Multiplayer release requires evidence for session isolation, authoritative server state, duplicate/stale message handling, reconnect semantics, security, and tested capacity.
- The current Go CLI startup behavior has a passing focused unit test. Engine, API, persistence, frontend, integration, and meaningful concurrency tests do not exist yet; do not infer their status from the CLI test.
