# AC92 attune reads its specs from a macfit store and adopts the shared help layout

## Summary

Skeleton. This AC is parked so the idea is not lost; the Director scopes it before Audit, possibly several releases from now.
attune gains a way to read its YAML spec bundle from a macfit store instead of a directory in the `infra` repository, so cloud specs and their secrets live encrypted beside the Mac config files and the `infra` repository stops holding anything sensitive. attune's help output also moves to the shared layout that `vkeep`, `vconv`, `ishrink`, and `macfit` use: name and version line, one-sentence description, then `Overview`, `Usage`, `Options`, and `Notes`. attune stays a reconciler; it never learns about keychains beyond calling `internal/lockbox`.

## In Scope

Provisional. Candidate design, to be settled at scoping:

- Store-backed specs: a store flag (letter to be settled; `-s` is taken by `--specs`) plus `ATTUNE_STORE`, reusing macfit's resolution order through `internal/lockbox`. When set, attune loads every store entry whose expanded target lies under the `--specs` directory and parses its content as a spec file, reading nothing from disk. `attune.yaml` may come from the store the same way.
- No new entry kind or schema: the Director registers spec files with `macfit add` under a chosen folder, and attune filters by target prefix.
- Help layout: `attune -h` and `attune help` print the shared layout; `--version` keeps its current form or moves to `attune vX.Y.Z` as the Director decides.
- attune's `programVersion` bumps to the next MINOR.

### Files to create

- TBD at scoping.

### Files to modify

- `cmd/attune/main.go`, `cmd/attune/config.go`, `cmd/attune/spec.go`, their tests, and `cmd/attune/README.md` — TBD at scoping.

## Out Of Scope

- Retiring or emptying the `infra` repository. That is the Director's action once attune reads from the store.
- Any change to macfit or the store format.
- Migrating existing specs into a store.

## Migration findings

- None. This AC was not emitted by `govna audit`.

## Acceptance Tests

TBD at scoping. Expected shape:

**AT1** [Automated] [Pre-release gate] — `attune validate` with the store flag and a store holding the testdata specs reports the same count as the directory form.

**AT2** [Automated] [Pre-release gate] — `attune plan` with the store flag reads no spec file from disk, checked by pointing `--specs` at an empty directory.

**AT3** [Automated] [Pre-release gate] — Help output follows the shared layout and the README usage block equals it.

**AT4** [Automated] [Pre-release gate] — `./build.sh` passes.

## Status

`PENDING` — skeleton, unscoped; awaiting the Director's scoping before Audit. Parked across releases until then.
