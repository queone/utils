# AC17 Document CLI framework standardization policy

Existing CLIs mix manual parsing, `go-arg`, and `cobra`. Do not rewrite stable commands for consistency alone. For new commands: manual parsing for tiny tools, `go-arg` for moderate complexity, `cobra` for multi-command UX. Low priority (policy/process).

## Summary

Policy doc capturing the current decision-tree for picking a flag-parsing approach on new CLI tools. Does not migrate any existing tool; guides future additions only. Lands as a new section in `docs/development-guidelines.md` (or a dedicated file if scoping decides otherwise).

## Objective Fit

1. **Which part of the primary objective?** TBD — requires scoping.
2. **Why not advance a higher-priority task instead?** TBD — requires scoping.
3. **What existing decision does it depend on or risk contradicting?** TBD — requires scoping.
4. **Intentional pivot?** TBD — requires scoping.

## In Scope

- Document the manual / `go-arg` / `cobra` decision tree for new CLI tools.
- State explicitly that existing stable commands are not migrated for consistency alone.
- Place the policy in `docs/development-guidelines.md` (or a dedicated policy file — scoping decision).

## Out Of Scope

TBD — requires scoping before critique gate.

## Implementation Notes

Rudimentary stub — requires further scoping before critique gate or implementation authorization.

## Acceptance Tests

TBD — requires scoping before critique gate.

## Status

`PENDING` — awaiting further scoping before critique gate or implementation authorization.
