# Architecture Overview

**Status:** Deterministic simulation engine, Go CLI, HTTP API, and initial React/TypeScript browser workflow implemented. Persistence, authentication, multiplayer, and the complete game remain planned.

## Current implementation

- `cmd/netarena` provides the simulation CLI, including service-capacity sweeps.
- `cmd/netarena-server` starts the HTTP API. Its listen address is configurable through `NETARENA_ADDR`; the default is `127.0.0.1:8080`.
- `internal/api` implements `GET /api/v1/health` and `POST /api/v1/simulations`, including request validation and JSON error responses.
- `internal/sim` contains the deterministic discrete-event simulation, routing, request lifecycle, failure/recovery, resilience policies, metrics, scenarios, and experiment reporting.
- `web` contains the React and TypeScript browser client. It configures simulation scenarios, supports network-topology editing and routing-algorithm selection, submits requests to the API, and displays simulation results, route traces, and errors. The browser validates topology input before submission; the simulation engine remains authoritative for execution.
- Automated Go and frontend tests cover the implemented components. Verification evidence and outstanding limitations belong in `docs/development/verification-log.md`.

The browser workflow is an initial vertical slice, not a complete game. Durable persistence, authentication, multiplayer, and real-time updates are not implemented by this milestone.

The module path is provisionally `example.com/netarena` because no Git remote is configured. Replace it with the canonical repository path before publishing or introducing imports that depend on it.

## Initial architecture

```text
React + TypeScript browser client
        | HTTPS / WebSocket (real-time channel when introduced)
        v
Go API + authentication + session management
        |
        v
Simulation domain + deterministic discrete-event engine
        |                         |
        v                         v
PostgreSQL durable records    Simulation metrics/events
        |
        v
OpenTelemetry instrumentation at useful application boundaries

Automated test harness -> engine / API / database / browser / load scenarios
```

## Boundaries

- The frontend presents state and sends player actions; it is not authoritative for scores or game state.
- The API validates identity, authorization, input, and session access.
- The simulation domain owns authoritative simulation state.
- The engine uses virtual time and explicit event ordering. Equal-timestamp tie-breaking must be specified and tested.
- Independent sessions should be isolated and may run concurrently; state transitions within a session must follow a documented consistency model.
- PostgreSQL stores durable records/results/replay data as required, not necessarily every transient event.
- Instrumentation must distinguish simulated metrics from real host/server metrics.

## First simulation model

Begin with a small topology: traffic generator, routing/load-balancing component, two application instances, database, and links with modeled latency/capacity. Use a fixed seed and repeatable workload.

## Open decisions

- Exact domain types and package boundaries.
- Event timestamp representation and tie-breaking.
- Request state machine and cancellation semantics.
- Initial network and CPU scheduling abstractions.
- REST/WebSocket contracts.
- Replay persistence format.
- Multiplayer model and deployment topology.

Record significant decisions in `docs/decisions/` using an ADR.
