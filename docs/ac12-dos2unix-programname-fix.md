# AC12 Fix `dos2unix` programName constant

Bug fix for `cmd/dos2unix/main.go:13`: `programName` is set to `"pman"` instead of `"dos2unix"`. Correct the constant and verify usage/help output reflects the right tool name. Medium-priority correctness fix.

## Summary

Single-line constant fix in `cmd/dos2unix/main.go` plus verification that the `--help` / usage strings render the correct program name.

## Objective Fit

1. **Which part of the primary objective?** TBD — requires scoping.
2. **Why not advance a higher-priority task instead?** TBD — requires scoping.
3. **What existing decision does it depend on or risk contradicting?** TBD — requires scoping.
4. **Intentional pivot?** TBD — requires scoping.

## In Scope

- `cmd/dos2unix/main.go:13` — set `programName = "dos2unix"`.
- Verify usage/help output tokenization (no residual `pman` strings).

## Out Of Scope

TBD — requires scoping before critique gate.

## Implementation Notes

Rudimentary stub — requires further scoping before critique gate or implementation authorization.

## Acceptance Tests

TBD — requires scoping before critique gate.

## Status

`PENDING` — awaiting further scoping before critique gate or implementation authorization.
