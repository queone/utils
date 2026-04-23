# utils Plan

## Product Direction

A collection of small CLI utilities written in Go. Each utility is a single-purpose, composable tool — installable as a standalone binary via `go install`. The repo prioritizes correctness, stability, and low-friction install/use over feature breadth.

## Ideas To Explore

Pre-rubric ideas captured for future discussion. Prefix each with `IE<N>:` (sequential N) for stable references. These are not commitments and have not passed the objective-fit rubric in `AGENTS.md`. Remove entries when promoted to an AC, completed, or no longer interesting; this section is pre-rubric staging, not a historical record.

Per the `IE-as-pointer` Local Rule in `docs/development-cycle.md`, IE entries in this repo may also persist as pointers to stub ACs that have been drafted but not yet scoped — those entries are removed only when the referenced AC ships or is deleted.

- IE1: Security shell-injection fix in `cmd/fr/main.go` — tracked as `docs/ac11-fr-shell-injection-fix.md`.
- IE2: Bugfix `dos2unix` wrong `programName` constant — tracked as `docs/ac12-dos2unix-programname-fix.md`.
- IE3: Test coverage expansion for `cmd/rn`, `cmd/fr`, `cmd/web` — tracked as `docs/ac13-cmd-test-coverage.md`.
- IE4: README placeholder and stale `cmd/*/README.md` reference cleanup — tracked as `docs/ac14-readme-placeholder-cleanup.md`.
- IE5: Audit of `init()` no-op suppressor functions across `cmd/*/main.go` — tracked as `docs/ac15-init-noop-suppressor-audit.md`.
- IE6: `cmd/cash5/main.go` refactor — split ~956 lines into `main.go` / `fetch.go` / `display.go` / `stats.go` / `model.go` — tracked as `docs/ac16-cash5-refactor.md`.
- IE7: CLI framework standardization policy (manual / go-arg / cobra) — tracked as `docs/ac17-cli-framework-policy.md`.
- IE8: Replace cash5 WINNING GEOMETRIES with just the WINNING CIRCLE
- IE9: Create new utility `moneycon` that does exactly what ./moneycon.py does — tracked as `docs/ac18-moneycon-utility.md`.
