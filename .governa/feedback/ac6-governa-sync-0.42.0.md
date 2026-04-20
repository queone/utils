# AC6 governa sync v0.42.0 — per-sync feedback

Per-sync feedback artifact per `docs/build-release.md` Template Upgrade rule (step 4). Captures observations about the v0.32.1 → v0.42.0 sync output that are worth routing upstream to governa for consideration. Moved to `.governa/feedback/ac6-governa-sync-0.42.0.md` at release prep (not deleted), so the feedback persists for future `enhance -r` runs.

## Upstream suggestions

### Consider adopting the canonical-build smoke-test clause into the template

The utils repo carries a clause under `AGENTS.md` Project Rules, appended to the canonical-build bullet:

> For quick smoke-testing of a single utility, use `go run ./cmd/<tool>/` or `go build -o /tmp/<name> ./cmd/<tool>/`; do not `go build ./cmd/<tool>/` from the repo root (it drops a stray binary).

This rule was added in AC5 after a stray-binary incident where an agent ran `go build ./cmd/claudecfg/` for smoke testing and left a 3.3MB Mach-O at the repo root. QA caught it before commit. The v0.42.0 template defends against *template-owned* stray binaries (`/build`, `/prep`, `/rel`) via `.gitignore` enumeration, but offers no guidance for consumer-repo tools.

Why this is worth lifting upstream:

- **Every multi-binary consumer repo faces the same footgun.** `go build ./cmd/<name>/` is a natural smoke-test instinct for any Go developer; the stray-binary output isn't obvious until after the fact.
- **Process-level guidance beats mechanical guards.** Enumerating every consumer tool in `.gitignore` is N+1 maintenance (every new tool requires a matching line). A single AGENTS.md clause covers every present and future tool without ongoing edits.
- **The template already canonicalizes `./build.sh` as the build path.** Adding the smoke-test subclause is a natural extension of the existing canonical-build rule, not a new concept.

Proposed template addition (append to the `Always use the repo's canonical build command` bullet in Project Rules):

> For quick smoke-testing of a single utility, use `go run ./cmd/<tool>/` or `go build -o /tmp/<name> ./cmd/<tool>/`; do not `go build ./cmd/<tool>/` from the repo root (it drops a stray binary).

## Observations about the sync itself

### Landed well

- **`.governa/` directory migration is seamless.** The bookkeeping layout change (`.governa-manifest` → `.governa/manifest`, `governa-sync-review.md` → `.governa/sync-review.md`, `.governa-proposed/` → `.governa/proposed/`) auto-migrates via the sync run. No manual cleanup needed on the consumer side.
- **Overlay-file landing for new template artifacts is right.** `prep.sh`, `cmd/prep/main.go`, `internal/preptool/*.go`, `docs/critique-protocol.md` all wrote directly (no diff to review) because they're new. This kept the adoption AC focused on the 14 real adopt items rather than 19.
- **Section-order advisory (AC58+AC59) caught a real drift.** `plan.md` had `Ideas To Explore ↔ Deferred` inverted relative to the template. Easy to adopt alongside the other reorder work.
- **Companion Artifacts convention (Critique, Feedback, Dispositions) is the right mental model.** Each file has an owner (QA for critique, DEV for feedback/dispositions) and a clear release-prep lifecycle (delete vs move). Made this sync's partial-adopts tractable.

### Friction surfaced during adoption

- **"24 unchanged" vs "14 adopt" accounting is ambiguous when a `keep` file has an advisory.** `plan.md` was classified `keep` but the section-order advisory effectively asks for a change. AC6 reclassified it as "reorder (from advisory)" to keep the drift arithmetic honest (`14 adopt + 1 reorder + 9 unchanged = 24`). Template documentation could be clearer that advisories on `keep` items are effectively adopt candidates, not truly-keep.
- **Sync review doesn't list the `keep` files explicitly.** Only `adopt` items are enumerated under `## Adoption Items`. For a thorough review, a consumer agent has to cross-reference `.governa/sync-review.md` and the Recommendations table. An explicit `## Kept Files (no action)` section would make completeness-checking easier.
- **Standing Divergence reconciliation has no tool support.** Each of five pre-AC6 Standing Divergences needed a manual diff against the new template to decide keep vs drop. The `governa ack` subcommand (v0.35.0) suppresses files from future sync reviews but doesn't proactively list "was this divergence subsumed by the new template?" A reconciliation helper — maybe `governa ack --review` listing each existing ack entry with a proposed keep/drop based on the new baseline — would reduce manual diffing.
- **Recommendation reason phrasing is terse.** For example, `AGENTS.md` reason is "governed sections changed: Purpose (cosmetic), Approval Boundaries (cosmetic), Review Style (cosmetic), Release Or Publish Triggers (cosmetic), Project Rules (cosmetic)." "cosmetic" doesn't convey what changed. One-line summaries per section would help the adopt/keep decision.
- **The `Purpose` section change is mislabeled as cosmetic.** Template v0.42.0's `Purpose` shifted from a governance-preamble description to a one-line repo-description. This is semantic (different intent), not cosmetic. Worth distinguishing "cosmetic wording changes" from "semantic role change" in the recommendation reason.

## Metadata

- Sync range: governa 0.32.1 → 0.42.0
- Consumer repo: utils (github.com/queone/utils)
- AC: AC6
- Generated: during AC6 implementation, author: DEV
