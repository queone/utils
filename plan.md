# utils Plan

## Product Direction

A collection of small CLI utilities written in Go. Each utility is a single-purpose, composable tool — installable as a standalone binary via `go install`. The repo prioritizes correctness, stability, and low-friction install/use over feature breadth.

## Current Platform

- Go
- Canonical build via `./build.sh` (dispatches to `go run ./cmd/build` and `go run ./cmd/rel`)
- Multi-binary layout under `cmd/*`; shared helpers in `internal/*`

## Priorities

Active work items. Each bullet names the file(s), the one-line problem/fix, and a priority tag. Promote to a standalone AC when picked up; remove when shipped.

- **Security: shell injection in `fr`** — `cmd/fr/main.go:31` builds a shell command via `script.Exec(fmt.Sprintf(...))` using a filename; fix by switching to `exec.Command("file", "-b", "--mime-type", path)`, preserving MIME parsing (`text/*` plus XML/JSON allowances) and defining safe fallback if `file` is unavailable. **Priority:** Critical.
- **Bugfix: `dos2unix` wrong programName** — `cmd/dos2unix/main.go:13` sets `programName` to `"pman"` instead of `"dos2unix"`; correct the constant and verify usage/help output. **Priority:** Medium.
- **Test coverage expansion** — only `cmd/web/search_test.go` exists under `cmd/*`; add `*_test.go` for `cmd/rn` (dry-run vs `-f` rename), `cmd/fr` (regex match counting, replace output, file update path), and `cmd/web` (replace live-network test with mocked HTTP transport; add parsing/error tests). Use table-driven tests and temp dirs. Must be deterministic — no live network. **Priority:** Medium.
- **README placeholder cleanup** — `README.md` has lingering `<description>` placeholders and references to `cmd/*/README.md` files that do not exist; replace every placeholder with a one-line summary and ensure `cash5` is listed. Create per-utility READMEs only if scoped separately. **Priority:** Low.
- **`init()` no-op suppressor cleanup** — multiple `cmd/*/main.go` files use `func init() { _ = programName; _ = programVersion }`. Audit per command: remove the no-op only where the constants are genuinely used, otherwise remove the unused constants or surface them in usage/version output. `./build.sh` must pass for every affected utility. **Priority:** Low.
- **`cash5` refactor** — `cmd/cash5/main.go` is ~956 lines; split into `main.go` (CLI wiring), `fetch.go` (data acquisition), `display.go` (terminal output), `stats.go` (analysis), `model.go` (shared data structures). No behavior change in CLI flags or output semantics; `./build.sh cash5` must pass before broader rollout. **Priority:** Medium (high effort, moderate risk).
- **CLI framework standardization policy** — existing CLIs mix manual parsing, `go-arg`, and `cobra`; do not rewrite stable commands for consistency alone. For new commands: manual parsing for tiny tools, `go-arg` for moderate complexity, `cobra` for multi-command UX. **Priority:** Low (policy/process).

## Ideas To Explore

Pre-rubric ideas captured for future discussion. Prefix each with `IE<N>:` (sequential N) for stable references. These are not commitments and have not passed the objective-fit rubric in `AGENTS.md`. Remove entries when promoted to an AC, completed, or no longer interesting; this section is pre-rubric staging, not a historical record.

## Deferred

| ID | Description | Reason |
|----|-------------|--------|

## Constraints

- project-specific anti-patterns and guardrails here
