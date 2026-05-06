# utils Plan

## Product Direction

A collection of small CLI utilities written in Go. Each utility is a single-purpose, composable tool — installable as a standalone binary via `go install`. The repo prioritizes correctness, stability, and low-friction install/use over feature breadth.

## Ideas To Explore

Ideas captured for future reference. Prefix each with `IE<N>:` (sequential N) for stable references. Entries are either a **pre-rubric IE** — `IE<N>: <one-liner>` awaiting director discussion and the objective-fit rubric (see `AGENTS.md` Approval Boundaries); or an **AC-pointer** — `IE<N>: <one-liner> → docs/ac<N>-<slug>.md` pointing to a drafted AC stub not yet scoped through the critique cycle. A pre-rubric IE that clears the rubric converts to an AC-pointer at AC-draft time (keeping the same `IE<N>` number) rather than being removed, so the entry persists as a pointer until the pointed-to AC ships. Remove entries when the underlying idea is closed: rejected, retired, or (for AC-pointers) the pointed-to AC has shipped and its file has been deleted. This section is not a historical record.

- IE10: Migrate `tfe` off `queone/utl` and absorb it into `utils/` as another utility.
- IE11: Migrate `azm` off `queone/utl` (placeholder; relocate to `azm/plan.md` once azm adopts governa).
- IE12: Evaluate replacing iq's `internal/color` with `queone/governa-color`.
- IE14: Consolidate `utils` on `queone/governa-color`; enhance governa-color with the colors / helpers (e.g. `ClearCode`, syntax-highlight palette) consumers need to drop direct `gookit/color` use.
- IE15: Sweep `TestAT<N>_` identifiers in `cmd/claudecfg/main_test.go` to behavior-describing names — AC34's sweep covered comments + error messages but not `func Test*` identifiers; 27 occurrences in that one file.
