# AC11 Fix shell-injection in `cmd/fr`

Security fix for `cmd/fr/main.go:31`: the binary classifier builds a shell command via `script.Exec(fmt.Sprintf(...))` against a caller-supplied filename, which allows shell injection through specially-crafted file names. Replace with `exec.Command("file", "-b", "--mime-type", path)` so the filename is passed as an argv token (never parsed by a shell), preserve the existing MIME parsing (`text/*` plus XML/JSON allowances), and define a safe fallback if `file(1)` is unavailable.

## Summary

Swap `script.Exec` + `fmt.Sprintf` for `os/exec` with argv-style args in `cmd/fr/main.go:31`; preserve MIME-based binary detection; add fallback behaviour when `file(1)` is missing on the host. Critical-priority security fix.

## Objective Fit

1. **Which part of the primary objective?** TBD — requires scoping.
2. **Why not advance a higher-priority task instead?** TBD — requires scoping.
3. **What existing decision does it depend on or risk contradicting?** TBD — requires scoping.
4. **Intentional pivot?** TBD — requires scoping.

## In Scope

- `cmd/fr/main.go:31` — replace `script.Exec(fmt.Sprintf(...))` with `exec.Command("file", "-b", "--mime-type", path)`.
- Preserve MIME parsing (`text/*`, XML/JSON allowances).
- Define safe fallback when `file(1)` is unavailable.

## Out Of Scope

TBD — requires scoping before critique gate.

## Implementation Notes

Rudimentary stub — requires further scoping before critique gate or implementation authorization.

## Acceptance Tests

TBD — requires scoping before critique gate.

## Status

`PENDING` — awaiting further scoping before critique gate or implementation authorization.
