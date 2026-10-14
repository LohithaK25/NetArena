# NetArena AI-Assisted Development Rules

Before changing code:
1. Read `docs/PROJECT_PLAN.md`, `TASKS.md`, this file, and the relevant architecture/testing/verification documents.
2. Inspect the current Git state and existing files. Never assume previous work is absent.
3. State the bounded task, acceptance criteria, intended approach, and non-goals.
4. Preserve the agreed product: a systems-engineering simulation game, not a superficial game or an AI wrapper.
5. Preserve the initial stack direction: Go, React + TypeScript, PostgreSQL, Docker, and OpenTelemetry unless a documented decision changes it.
6. Treat testing as a first-class workstream from the first implementation.
7. Add or update tests for new behavior and bug fixes. Run the relevant tests and report exact commands/results.
8. Never invent test results, coverage, capacity, latency, or benchmark numbers. If a check was not run, state that it remains unverified.
9. Review the diff and avoid unrelated rewrites, unnecessary dependencies, and premature microservices/infrastructure.
10. Update `TASKS.md`, the verification log, architecture docs, and decision records when appropriate.
11. Explain important design choices in terms of the problem, alternatives, rationale, trade-offs, and verification.
12. Do not commit, push, delete, reset, or overwrite user work unless explicitly asked.
13. Keep secrets and credentials out of source control.

When resuming a session, summarize the last verified milestone, current Git state, known issues, and next task before implementation.
