# Build and Release

Reference for this repo's build pipeline, pre-release checklist, and acceptance test conventions. The enforceable one-liners live in `AGENTS.md`; this document explains the pipeline, the steps, and the rationale.

## Build

This repo has a single canonical build/test workflow: `./build.sh`.

`./build.sh` is a thin Bash dispatcher. The real implementation lives in `go run ./cmd/build` (build/test) and `go run ./cmd/rel` (release). Both `cmd/build` and `cmd/rel` are `go run` entrypoints — they are intentionally not installed as binaries.

The build pipeline runs these steps in order, fail-hard on each:

1. `mdcheck` — **fail-hard.** Scans tracked markdown files for nested-fence bugs (3-backtick outer fence containing a tagged 3-backtick inner opener). The fix is to widen the outer fence to 4+ backticks or switch to `~~~`.
2. `go mod tidy` — ensure `go.mod` and `go.sum` are consistent
3. `go fmt ./...` — **fail-hard.** If `go fmt` rewrote any file (non-empty stdout), the build fails. Re-run after committing the formatting fix.
4. `go fix ./...` — advisory; output is logged but does not break the build
5. `go vet ./...` — **fail-hard**
6. Test suite with coverage — fail-hard on any test failure
7. `staticcheck ./...` — **fail-hard.** Ensured available (installed via `go install` if not on PATH).
8. Binary build — installs utilities to `$GOPATH/bin`

To scope the run to selected packages, pass target names: `./build.sh build rel`. Validation (vet, fmt, test, staticcheck) runs only against those packages. Targets named `build` or `rel` are validated but not installed as binaries — run them with `go run` instead.

Invoking individual Go tools directly skips the tidy/fmt/lint pipeline above. A "passing" direct invocation can still produce a build that `./build.sh` would reject. The wrapper guarantees that what passes locally is what would pass in CI.

## Sandboxed Execution

Under sandboxed execution that blocks Go's build cache (look for `writing stat cache ... operation not permitted`), `staticcheck` may print a `matched no packages` warning even though it ran cleanly. Treat as advisory unless real findings appear; an unrestricted rerun confirms.

## Acceptance Tests

Every AT in an AC document must be labeled `[Automated]` or `[Manual]`.

- **Automated** — The result can be verified from CLI output, test assertions, or file inspection. Automated ATs are run during implementation and re-run as part of the pre-release checklist.
- **Manual** — Requires a live end-to-end action and must be confirmed by the user. The agent cannot self-verify these.

Default to Automated whenever the result is verifiable without a live external service. Manual ATs add friction to the release flow, so reserve them for behaviors that genuinely cannot be checked any other way.

## Pre-Release Checklist

Do not start this checklist unless the user explicitly asks to prep for release or equivalent.

The operator flow is two steps:

1. **Run `go run ./cmd/prep/ vX.Y.Z "message"`.** Stages version bumps, inserts the CHANGELOG row, deletes completed AC files (plus `-critique.md` companions), runs validation builds before and after, and prints the canonical release command. The agent determines the version (semver classification from the AC's scope) and drafts the release message (≤ 80 characters) before invoking prep.
2. **Run the printed release command (`./build.sh vX.Y.Z "message"`).** `cmd/rel` shows `git status --short`, lists every git step it will execute, and prompts for interactive confirmation. On approval it orchestrates `git add → commit → tag → push tag → push branch`. Optional: run `git diff` between the two steps if you want to inspect the CHANGELOG row wording and version-string values before committing — `cmd/rel`'s own status preview is sufficient to catch wrong-file inclusions or deletions.

Present only the release command after prep; do not add trailing commentary about wrapper routing or prompts. The director already knows.

### Appendix: what prep does

`go run ./cmd/prep/` runs nine phases internally so the operator flow above stays short. Each phase has a clear failure mode:

1. **Validate inputs.** Semver pattern (`vX.Y.Z`), message non-empty and ≤ 80 characters.
2. **Validate git state.** Inside a git work tree, target tag does not exist yet, HEAD is not at the latest tag with a clean working tree.
3. **Pre-check build.** `./build.sh` run before any writes; skipped with `--no-build` or `--dry-run`.
4. **Detect version targets.** Scans `cmd/*/main.go` for `programVersion`, `TEMPLATE_VERSION` and `internal/templates/version.go` (each presence-gated). Multi-binary repos are picked up automatically.
5. **Detect CHANGELOG targets + fail-fast idempotency guard.** Root `CHANGELOG.md` and `internal/templates/CHANGELOG.md` (template-repo case). If any target already contains a row for the target version, prep exits with a fatal error before any writes.
6. **Parse AC refs.** `AC[0-9]+` scan on the release message; composites like `AC60+AC61` yield multiple refs.
7. **Apply writes.** Version bumps (per-file idempotent no-op when the file already has the target value); CHANGELOG row insertion under `| Unreleased | |`; AC file deletions plus `-critique.md` companion deletions. Skipped when `--dry-run`.
8. **Post-check build.** `./build.sh` run after writes; skipped with `--no-build` or `--dry-run`.
9. **Print release command.** Exactly `./build.sh vX.Y.Z "message"` — nothing else.

CHANGELOG row shape (enforced by prep's insertion code and by convention):

- File shape: `# Changelog` heading, then a 2-column markdown table (`| Version | Summary |` with a `|---|---|` separator); first data row is `| Unreleased | |`, followed by one row per release (e.g., `| <version> | <AC-ref>: <one-line summary> |`).
- Summaries are single-line, ≤ 500 characters; lead with the AC reference if any.
- Versions are unprefixed (`0.29.0`, not `v0.29.0`).
- Do not backfill historical tags or invent alternative shapes (Keep-a-Changelog, sectioned `## vX.Y.Z`, etc.).

Flags: `--dry-run` (or `-n`) prints intended writes without touching the working tree; `--no-build` skips phases 3 and 8. Both are for power users or tests — the common path is plain `go run ./cmd/prep/ vX.Y.Z "message"`.

## Per-Utility programVersion Doctrine

Per-utility `programVersion` constants in `cmd/*/main.go` are independent of the repo release version. Per-utility bumps live in the AC that changes the utility — never applied by `cmd/prep`. Utilities have already diverged into meaningful per-utility SemVers (e.g., `brew-update` 1.3.5, `pgen` 1.2.3, `dl` 2.0.0); aligning them to the repo tag would discard real per-utility evolution surfaced through `--version` output.

The `programVersionRe` regex in `internal/preptool/preptool.go` matches only the inline form `const programVersion = "x.y.z"` and intentionally does **not** match the grouped form `const ( ... programVersion = "x.y.z" ... )` used by every utility in this repo. As a result, phase 4 of prep iterates `cmd/*/main.go` but yields zero `programVersion`-kind targets, and phase 7 writes nothing under `cmd/`. This silent-skip is correct for utils. **Broadening the regex to match the grouped form, without first selecting per-utility-vs-repo-tracked semantics in a successor toolchain, will trigger a mass downgrade of every utility to the release version.**

The bug was resolved upstream by governa AC100, which broadened `programVersionRe` to match both forms and added a safe auto-detect filter (1 target → bump; >1 → skip with a logged warning). preptool stays a template — the convention-coupling test in governa's `docs/library-policy.md` rejected library extraction, so no `governa-preptool` ships. utils chose the doctrine+guard mitigation pattern instead of pulling AC100's source patch; both shapes are documented in the upstream advisory at `docs/advisories/program-version-bump.md` (governa). The guard test (`TestAC26_GroupedConstFormNotBumped` in `internal/preptool/preptool_test.go`) remains permanent — it fails immediately if anyone broadens the local regex without porting the auto-detect filter alongside it.
