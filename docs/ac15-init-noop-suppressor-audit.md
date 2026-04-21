# AC15 Audit `init()` no-op suppressors across `cmd/*/main.go`

Multiple `cmd/*/main.go` files use `func init() { _ = programName; _ = programVersion }`. Audit each command: remove the no-op only where the constants are genuinely used, otherwise remove the unused constants or surface them in usage/version output. `./build.sh` must pass for every affected utility. Low priority.

## Summary

Per-utility audit sweep. Decide for each `cmd/*/main.go` whether `programName`/`programVersion` are real program info (surface in help/version) or dead weight (remove both the constants and the no-op init). End state: no suppressor patterns left; every remaining constant has a real use.

## Objective Fit

1. **Which part of the primary objective?** TBD — requires scoping.
2. **Why not advance a higher-priority task instead?** TBD — requires scoping.
3. **What existing decision does it depend on or risk contradicting?** TBD — requires scoping.
4. **Intentional pivot?** TBD — requires scoping.

## In Scope

- `cmd/*/main.go` — enumerate every file using `func init() { _ = programName; _ = programVersion }`.
- For each: either surface the constants in usage/version output, or remove the constants + no-op init together.
- `./build.sh` must pass for every affected utility.

## Out Of Scope

TBD — requires scoping before critique gate.

## Implementation Notes

Rudimentary stub — requires further scoping before critique gate or implementation authorization.

## Acceptance Tests

TBD — requires scoping before critique gate.

## Status

`PENDING` — awaiting further scoping before critique gate or implementation authorization.
