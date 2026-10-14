# Architecture Overview

**Status:** Go CLI foundation implemented; the simulation and service architecture remain planned.

## Current implementation

`cmd/netarena` is a small Go executable that prints a startup message. Its test verifies the exact output. This establishes a buildable, testable Go module without implying that the simulation engine, API, persistence, or frontend exists.

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
