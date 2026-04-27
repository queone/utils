# Development Guidelines

Engineering guidance for any agent or contributor working in this repo.
These are durable coding practices, not workflow or process rules.
For workflow, see `development-cycle.md`. For validation, see `build-release.md`.

utils is a collection of small, single-purpose CLI utilities written in Go — each installable as a standalone binary via `go install`. The repo prioritizes correctness, stability, and low-friction install/use over feature breadth. See `plan.md` for product direction.

## Identifier Strategy

- Choose a primary key strategy early and document it in `arch.md`
- Prefer surrogate keys for internal identity; keep external IDs as indexed attributes
- When integrating multiple external ID systems, maintain an explicit mapping layer rather than assuming IDs are interchangeable

## Schema And Data Migrations

- Treat schema changes as first-class events: version them, document them, test the migration path
- Never assume old data fits new schemas — write migration logic or fail explicitly
- When a migration changes identity or key structure, audit all foreign key references in the same change

## External Integration Patterns

- Validate external data at the boundary; do not trust upstream shape or completeness
- When reconciling data from multiple sources, define a clear precedence order and document it
- Cache external data locally with explicit TTL or versioning; never silently serve stale data as fresh

## Generated Artifact Propagation

- When source-of-truth code is duplicated into templates or rendered examples, fixes must propagate to all copies in the same change
- Grep the full repo for the pattern being changed before considering a fix complete
- If a template and its rendered output diverge, the template is authoritative
- Exported functions in shared packages (`internal/buildtool`, `internal/reltool`, `internal/color`) carry godoc single-line comments to keep the public surface self-documenting.

The convention is "what it does, not how" — the *how* changes with refactors and the comment goes stale; the *what* is the contract.

## Program Version Declaration

- Every installable `cmd/<name>/main.go` must declare a non-empty `const programVersion` string literal
- Script-only helper entrypoints (`build`, `rel`) are exempt
- The build tool validates this before compiling installable binaries; missing or empty declarations fail the build

### Semver application

| Bump  | When                                                                                  |
|-------|---------------------------------------------------------------------------------------|
| PATCH | Bug fixes, tests, internal refactors, tooling — anything invisible to users at the CLI level. Batch when possible. |
| MINOR | User-visible features, new commands, new flags, behavioral changes, governance/template/docs scaffolding adoption. Reset patch to 0. |
| MAJOR | Not applicable before 1.0.                                                            |

The "would a user notice this in `--help`?" test catches most ambiguous cases. New flags, renamed flags, new commands, changed output formats — all MINOR. Internal refactors, dependency bumps that don't change behavior, additional tests, build-tooling changes — all PATCH. The "batch when possible" guidance for PATCH means you can accumulate several small fixes into one PATCH release rather than tagging each one.

## Error Handling And Validation

- Validate at system boundaries (user input, external APIs, file I/O); trust internal code
- Fail explicitly rather than silently degrading — a clear error is better than wrong output
- Static analysis and linting errors are build failures, not warnings
- Wrap user-facing errors with operation context and recovery guidance — bare `return err` at command boundaries is unacceptable

The user sees the error message verbatim, and Go errors from the standard library or third-party packages have no actionable context. Wrapping with `fmt.Errorf("operation: %w (recovery hint)", err)` gives the user a fighting chance to fix the problem themselves.

## Testing Expectations

- Tests are part of implementation, not a follow-up step
- Every new function and error path should have a test before the work is presented as complete
- If a code path cannot be tested without mocking infrastructure that is out of scope, document the coverage gap explicitly rather than silently skipping it
- Label tests that require live systems or manual verification as `[Manual]`

When refactoring, treat existing tests as load-bearing assertions. They encode behavior that someone thought was worth pinning down. If a test is genuinely obsolete, delete it explicitly with a note in the commit; don't let it rot.

## Dependency And Import Hygiene

- Prefer standard library over external dependencies when the capability is equivalent
- When adding a dependency, justify it — convenience alone is not sufficient
- Keep import paths consistent after renames or reorganizations; grep for stale references

## CLI Usage Formatting

- All commands must accept `-h`, `-?`, and `--help` as help flags
- Help output uses a shared formatting function for consistent layout
- "Usage:" is rendered in bold white
- Each flag line is indented 2 spaces; descriptions align at column 38
- Short and long flag forms are combined on one line (e.g. `-v, --verbose`)
- When adding new flags, add the entry to the shared usage formatter — do not rely on framework defaults

## Documentation Alignment

- Docs ship with the code change that introduces the behavior
- If a doc references a function, flag, or file path, verify it still exists before publishing
- Architecture docs (`arch.md`) reflect what is built, not what is planned; `plan.md` is forward-looking only
