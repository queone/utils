# AC16 Refactor `cmd/cash5/main.go`

`cmd/cash5/main.go` is ~956 lines. Split into `main.go` (CLI wiring), `fetch.go` (data acquisition), `display.go` (terminal output), `stats.go` (analysis), `model.go` (shared data structures). No behaviour change in CLI flags or output semantics; `./build.sh cash5` must pass before broader rollout. Medium priority (high effort, moderate risk).

## Summary

Code-organisation refactor of `cmd/cash5/main.go` into five topical files within the same package. Behaviour-preserving: no changes to flags, output formatting, iTerm2 image rendering, or exit codes. Intended to make the file tractable before any further feature work in `cash5`.

## Objective Fit

1. **Which part of the primary objective?** TBD — requires scoping.
2. **Why not advance a higher-priority task instead?** TBD — requires scoping.
3. **What existing decision does it depend on or risk contradicting?** TBD — requires scoping.
4. **Intentional pivot?** TBD — requires scoping.

## In Scope

- `cmd/cash5/main.go` — split into:
  - `main.go` — CLI wiring.
  - `fetch.go` — data acquisition.
  - `display.go` — terminal output.
  - `stats.go` — analysis.
  - `model.go` — shared data structures.
- No behaviour change in CLI flags or output semantics.
- `./build.sh cash5` must pass.

## Out Of Scope

TBD — requires scoping before critique gate.

## Implementation Notes

Rudimentary stub — requires further scoping before critique gate or implementation authorization.

## Acceptance Tests

TBD — requires scoping before critique gate.

## Status

`PENDING` — awaiting further scoping before critique gate or implementation authorization.
