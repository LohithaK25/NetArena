# NetArena

NetArena is a distributed-systems simulation and strategy game. Players build and operate simulated infrastructure under traffic spikes, resource contention, network problems, failures, and security incidents.

## Project status

**Phase 0 — Repository foundation.** The Go CLI and its startup-message test are the initial executable/test foundation. No simulation, API, or frontend features are implemented yet.

## Source of truth

- `docs/PROJECT_PLAN.md` â€” project vision, roadmap, architecture direction, testing, and engineering journal.
- `AGENTS.md` â€” instructions for AI-assisted development.
- `TASKS.md` â€” current milestone and next actions.
- `docs/architecture/overview.md` â€” current architecture and boundaries.
- `docs/testing/TESTING_STRATEGY.md` â€” dedicated testing workstream.
- `docs/development/verification-log.md` â€” commands run and evidence.

## Development principles

- Correctness before complexity.
- Tests and documentation from the first milestone.
- Single-player correctness before multiplayer expansion.
- Measure performance; never invent benchmark claims.
- Mark work verified only when acceptance criteria and evidence support it.

## Planned starting stack

Go simulation engine/API, React + TypeScript frontend, PostgreSQL persistence, Docker for reproducible environments, and OpenTelemetry for useful instrumentation. The inspected tool versions are recorded in `docs/development/verification-log.md`.

## Go development

The Go module currently uses the provisional path `example.com/netarena` because this repository has no configured Git remote. Replace it with the repository's canonical import path before publishing the module or adding packages that import it.

The current Go foundation uses Go 1.27.0 and only the standard library. From the repository root in PowerShell:

```powershell
# Add the Go installation directory if it is not already on PATH.
$env:Path = 'C:\Program Files\Go\bin;' + $env:Path

# Check formatting without modifying files.
$unformatted = Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -l $_.FullName }
if ($unformatted) { $unformatted; throw "Go source files are not formatted." }

go vet ./...
go test -race ./...
go build -o (Join-Path $env:TEMP 'netarena.exe') ./cmd/netarena
go run ./cmd/netarena
```

To format Go source files before checking them:

```powershell
Get-ChildItem -Recurse -Filter *.go | ForEach-Object { gofmt -w $_.FullName }
```

GitHub Actions runs the formatting check, vet, race-enabled tests, and build on pushes and pull requests targeting `main`. A local pass does not verify a hosted CI run.
