# utils Plan

## Product Direction

A collection of small CLI utilities written in Go. Each utility is a single-purpose, composable tool — installable as a standalone binary via `go install`. The repo prioritizes correctness, stability, and low-friction install/use over feature breadth.

## Ideas To Explore

Ideas captured for future reference. Prefix each with `IE<N>:` (sequential N) for stable references. Entries come in two shapes: (a) **pre-rubric idea** — `IE<N>: <one-liner>` awaiting director discussion and the objective-fit rubric (see `AGENTS.md` Approval Boundaries); (b) **pointer to a drafted AC stub** not yet scoped through the critique cycle — `IE<N>: <one-liner> → docs/ac<N>-<slug>.md`. A shape (a) entry that clears the rubric converts to shape (b) at AC-draft time (keeping the same `IE<N>` number) rather than being removed, so the entry persists as a pointer until the pointed-to AC ships. Remove entries when the underlying idea is closed: rejected, retired, or (for AC pointers) the pointed-to AC has shipped and its file has been deleted. This section is not a historical record.

- IE1: Security shell-injection fix in `cmd/fr/main.go` — tracked as `docs/ac11-fr-shell-injection-fix.md`.
- IE2: Bugfix `dos2unix` wrong `programName` constant — tracked as `docs/ac12-dos2unix-programname-fix.md`.
- IE3: Test coverage expansion for `cmd/rn`, `cmd/fr`, `cmd/web` — tracked as `docs/ac13-cmd-test-coverage.md`.
- IE4: README placeholder and stale `cmd/*/README.md` reference cleanup — tracked as `docs/ac14-readme-placeholder-cleanup.md`.
- IE5: Audit of `init()` no-op suppressor functions across `cmd/*/main.go` — tracked as `docs/ac15-init-noop-suppressor-audit.md`.
- IE6: `cmd/cash5/main.go` refactor — split ~956 lines into `main.go` / `fetch.go` / `display.go` / `stats.go` / `model.go` — tracked as `docs/ac16-cash5-refactor.md`.
- IE7: CLI framework standardization policy (manual / go-arg / cobra) — tracked as `docs/ac17-cli-framework-policy.md`.
- IE10: Migrate `utils` from local `internal/preptool` copy to imported `governa-preptool` once the library ships (governed by AC96 library policy and landed via the downstream preptool-extraction AC; trigger is library shipping, not AC96 alone); reconcile per-utility-vs-repo-tracked mode selection at adoption.
