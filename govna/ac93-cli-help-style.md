# AC93 One help header and section style for every gkit utility, with a recommendation for the other queone projects

## Summary

Skeleton. This AC is parked so the idea is not lost; the Director scopes it before Audit.
Every gkit utility prints the same first two help lines, in skout's format and coloring: the utility name in bold white followed by ` v<version>` in plain text, then a one-sentence description in gray. Section headings below them use one shared vocabulary, written capitalized rather than all caps (`Overview`, `Usage`, `Options`, `Notes`) and rendered in bold white. One helper renders the header and sections so no utility hand-colors its help. The AC ends with a short written recommendation for the other queone projects, skout among them, that names the header format, the section set, the ordering, and the `--version` form.
Doc impact: every `cmd/<name>/README.md` usage block, and one new style document whose home is settled at scoping.

## Observed today

- skout v0.14.1: line one is `skout` in bold white (256-color 231) plus ` v0.14.1` plain; line two is `Fantasy Baseball advisor — github.com/queone/skout` in gray (245); headings `USAGE` and `COMMANDS` are all caps in white (255), not bold; nested flags are indented under their command; `--version` prints `skout 0.14.1` without the `v` that the header shows.
- gkit: `vkeep`, `vconv`, `ishrink`, and `macfit` print `name vX.Y.Z`, a description line, then `Overview`, `Usage`, `Options`, `Notes` in plain text; `attune` prints `attune — description` then `Usage:` and `Flags:`; older utilities vary further. `--version` prints `name X.Y.Z` in some and `name vX.Y.Z` in others.

## In Scope

Provisional. Candidate design, to be settled at scoping:

- Header: line one `<name>` in bold white then ` v<version>` plain; line two the one-sentence description in gray. Whether line two also carries ` — github.com/queone/gkit/cmd/<name>` is a scoping decision.
- Headings: capitalized words, bold white, through `internal/color` (`color.Bold(color.Gra10(...))` or the equivalent settled at scoping), plain when output is not a terminal.
- Section vocabulary and order: `Overview` (optional), `Usage` (required, synopsis lines), `Commands` (optional, multi-command utilities, with nested flags indented as skout does), `Options` (required when flags exist; replaces `Flags`), utility-specific sections such as `Cheatsheet` or `Timestamps` (optional, after `Options`), `Examples` (optional), `Notes` (optional, last). The basic set every CLI carries is `Usage` and `Options`.
- One renderer, `internal/usage` or a function in `internal/color`, takes name, version, description, and ordered sections and returns the text, so each utility's `usage()` becomes data.
- `--version` prints `<name> v<version>` everywhere; `-v`, `-h`, `-?`, `--help`, `help`, and `version` aliases as the shared convention.
- Every `cmd/<name>/main.go` usage text, its tests, and its README usage block move to the renderer. Scoping decides whether the migration ships in one AC or in batches by utility family.
- The recommendation document for other queone projects, including the note that skout's `--version` should print `skout v0.14.1` to match its own header and that its headings move from all caps to capitalized bold white.

### Files to create

- `internal/usage/usage.go` and `usage_test.go` — TBD at scoping.
- The style document — location TBD at scoping (a gkit doc, a bits entry, or govna canon).

### Files to modify

- Every `cmd/<name>/main.go`, `main_test.go`, and `README.md` — TBD at scoping, possibly batched.

## Out Of Scope

- Changing any utility's behavior, flags, or output other than help and version text.
- Editing skout or any other queone project; they receive the recommendation.

## Migration findings

- None. This AC was not emitted by `govna audit`.

## Acceptance Tests

TBD at scoping. Expected shape:

**AT1** [Automated] [Pre-release gate] — Every installed utility's help starts with `<name> v<version>` and a non-empty description line, checked by one test that iterates `cmd/*`.

**AT2** [Automated] [Pre-release gate] — With color enabled, the name carries the bold white sequence and every heading carries the bold white sequence; with color disabled, no escape sequence appears.

**AT3** [Automated] [Pre-release gate] — Every `--version` prints `<name> v<version>`, checked by `./build.sh`.

**AT4** [Automated] [Pre-release gate] — Every README usage block equals its utility's help text.

**AT5** [Automated] [Pre-release gate] — `./build.sh` passes.

## Status

`PENDING` — skeleton, unscoped; awaiting the Director's scoping before Audit. Parked across releases until then.
