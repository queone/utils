# Fix Plan: utils Repo Improvements (Revised)

This plan reflects current repository state and prioritizes risk reduction first.

## 1. Security Fix: Shell Injection in `fr`
**File:** `cmd/fr/main.go:31`

**Problem:** `script.Exec(fmt.Sprintf(...))` builds a shell command string using a filename, enabling shell injection via crafted paths.

**Fix:**
- Replace shell invocation with `exec.Command("file", "-b", "--mime-type", path)`.
- Keep MIME parsing behavior (`text/*`, plus XML/JSON allowances).
- Remove dependency on `github.com/bitfield/script` in this command.
- Define behavior if `file` is unavailable (e.g., safe fallback to `false` with clear error handling).

**Acceptance criteria:**
- No command string interpolation with path values.
- `script.Exec` is no longer used in `cmd/fr`.
- Existing behavior for normal text file detection remains unchanged.

**Priority:** Critical

---

## 2. Test Coverage Expansion for `cmd/` Utilities
**Current state:** only `cmd/web/search_test.go` exists in `cmd/*` tests.

**Targets:**
- `cmd/rn/`: dry-run output vs `-f` rename behavior.
- `cmd/fr/`: regex match counting, replace output, and file update path.
- `cmd/web/`: replace live-network test with mocked HTTP transport and add parsing/error tests.

**Approach:**
- Add `*_test.go` files under each command directory.
- Use table-driven tests where practical.
- Use temp directories/files for filesystem behavior.
- Mock HTTP responses for `web`; avoid live network in default tests.

**Acceptance criteria:**
- New tests pass via `./build.sh`.
- `cmd/web` tests are deterministic and do not require internet access.

**Priority:** Medium

---

## 3. README Placeholder Cleanup (Scope Corrected)
**Primary file:** `README.md`

**Problem:** Placeholder `<description>` entries remain in root README; several referenced `cmd/*/README.md` files do not currently exist.

**Fix:**
- Replace all `<description>` placeholders in `README.md` with concise one-line descriptions.
- Include missing utility `cash5` in the completed descriptions list.
- Validate links; create missing `cmd/*/README.md` files only if explicitly desired in a separate docs pass.

**Acceptance criteria:**
- No `<description>` tokens remain in `README.md`.
- Every utility listed in root README has a meaningful one-line summary.

**Priority:** Low

---

## 4. Guarded Cleanup of `init()` No-op Usage Suppressors
**Files:** multiple `cmd/*/main.go`

**Problem:** Many commands use:
```go
func init() {
    _ = programName
    _ = programVersion
}
```
Blind removal may introduce compile errors where constants are otherwise unused.

**Fix:**
- Audit each command individually.
- Remove no-op `init()` only where `programName`/`programVersion` are genuinely used.
- Where constants are unused, either remove unused constants or surface them in usage/version output.

**Acceptance criteria:**
- No no-op `init()` suppressors remain.
- Build still succeeds for all utilities via `./build.sh`.

**Priority:** Low

---

## 5. Refactor `cash5` (High Effort)
**File:** `cmd/cash5/main.go` (~956 lines)

**Goal:** Improve maintainability by separating concerns while preserving CLI behavior.

**Suggested split:**
```
cmd/cash5/
  main.go          # CLI setup, flags, command wiring
  fetch.go         # API/data acquisition
  display.go       # formatting and terminal output
  stats.go         # statistics and analysis
  model.go         # shared data structures
```

**Acceptance criteria:**
- No behavior change in CLI flags/output semantics.
- `./build.sh cash5` passes before broader rollout.

**Priority:** Medium (high effort, moderate risk)

---

## 6. CLI Framework Standardization Policy
**Current state:** mixed manual parsing, `go-arg`, and `cobra`.

**Decision (pragmatic):**
- Keep mixed tooling for existing stable commands.
- For new commands, prefer:
  - manual parsing for tiny single-purpose tools,
  - `go-arg` for moderate complexity,
  - `cobra` for multi-command/advanced UX.

**Scope note:** do not rewrite existing CLIs solely for framework consistency unless tied to feature work.

**Priority:** Low (policy/process)

---

## 7. Additional Bugfix to Add
**File:** `cmd/dos2unix/main.go:13`

**Issue:** `programName` is set to `"pman"` instead of `"dos2unix"`.

**Fix:** Correct constant value and verify usage/help output consistency.

**Priority:** Medium

---

## Recommended Execution Order

1. **Security fix** in `fr`.
2. **dos2unix programName bugfix** (small, clear correctness issue).
3. **Test coverage** (`rn`, `fr`, `web`) with deterministic tests.
4. **README placeholder cleanup** in root docs.
5. **`init()` cleanup** with per-file audit and build validation.
6. **`cash5` refactor** (optional, scoped effort).
7. **CLI policy adoption** for future work (no mass rewrite).
