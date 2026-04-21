# AC13 Expand test coverage under `cmd/*`

Only `cmd/web/search_test.go` exists under `cmd/*`. Add `*_test.go` for `cmd/rn` (dry-run vs `-f` rename), `cmd/fr` (regex match counting, replace output, file-update path), and `cmd/web` (replace the live-network test with a mocked HTTP transport; add parsing/error tests). Use table-driven tests and temp dirs; must be deterministic — no live network.

## Summary

Add deterministic, table-driven tests for three CLIs currently lacking coverage. Retire any live-network test in `cmd/web` in favour of a mocked HTTP transport. Medium priority.

## Objective Fit

1. **Which part of the primary objective?** TBD — requires scoping.
2. **Why not advance a higher-priority task instead?** TBD — requires scoping.
3. **What existing decision does it depend on or risk contradicting?** TBD — requires scoping.
4. **Intentional pivot?** TBD — requires scoping.

## In Scope

- `cmd/rn` — tests for dry-run vs `-f` rename paths.
- `cmd/fr` — tests for regex match counting, replace output, file-update path.
- `cmd/web` — replace live-network test with mocked HTTP transport; add parsing/error tests.
- Shared: table-driven shape, temp dirs, deterministic, no live network.

## Out Of Scope

TBD — requires scoping before critique gate.

## Implementation Notes

Rudimentary stub — requires further scoping before critique gate or implementation authorization.

## Acceptance Tests

TBD — requires scoping before critique gate.

## Status

`PENDING` — awaiting further scoping before critique gate or implementation authorization.
