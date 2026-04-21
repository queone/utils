# AC8 governa sync v0.45.1 — per-sync feedback

Per-sync feedback artifact per `docs/build-release.md` Template Upgrade rule (step 4). Moved to `.governa/feedback/ac8-governa-sync-0.45.1.md` at release prep (not deleted), so the feedback persists for future `enhance -r` runs.

## Observations about the sync itself

### Landed well

- **Full feedback-loop round-trip validated.** AC6-era observations surfaced via `.governa/feedback/ac6-governa-sync-0.42.0.md` → governa addressed them upstream in v0.45.0 (AC69 "enhance feedback loop + utils feedback + integrated critique default") and v0.45.1 (AC70 "Feedback Credits enforcement") → v0.45.1 sync's AC63 scanner flagged `ac6-governa-sync-0.42.0.md` under `## Advisory Notes` as *"addressed by governa v0.45.1; review and delete if resolved."* End-to-end credit-backfill + scanner-detection works as designed.
- **Integrated-critique default (AC69) is a workflow simplification.** Moving critique content into the AC's `## Critique` section collapses the current two-file pattern (`ac<N>.md` + `ac<N>-critique.md`) into one file, reduces cross-reference churn, and makes the AC's critique history inspectable in a single git diff. The heading-level changes (`## Round N` → `### Round N`, `### F<N>` → `#### F<N>`) reflect the embedded-in-AC structure cleanly.
- **Feedback Credits format is load-bearing in the right way.** AC70's enforcement tightens the feedback loop: `prep.sh` reads the AC's `## Feedback Credits` section and requires the release message to contain a matching `(addresses <consumer> feedback from v<X.Y.Z> syncs)` credit per entry. Makes the loop auditable — upstream can grep CHANGELOG for consumer-credit patterns and reconcile against a registry.

### Friction surfaced during adoption

- **Sync-review recommendation reason line is load-bearing but ambiguous.** For `AGENTS.md` this sync, the reason was *"governed sections changed: Governed Sections (structural), Approval Boundaries (cosmetic), Project Rules (cosmetic: bullet removed)"*. `(cosmetic: bullet removed)` doesn't identify *which* bullet. A consumer agent trusting the sync-review could silently drop a legitimate repo-specific rule (in our case, AC4 README-alphabetical). Direct `diff` inspection is the only way to know. Consider naming the bullet or adding a pointer to the repo-specific-reason when the template proposes to remove a bullet that the repo has added.
- **`governa ack --review` runtime contradicts `--help`.** `governa ack --help` documents `--review` as *"read-only: propose keep/drop for each acked entry (no path required)"*, but invoking `governa ack --review` (or `governa ack -r`, or `governa ack --review --target <dir>`) errors with *"path is required: use `governa ack <path>`"* in all three forms. Exit code 0 despite the error message. The feature is non-functional as documented; either the help text overstates what works, or the runtime was skipped for the no-path branch. AC8's planned hygiene-check step using this command had to be dropped.
- **`governa sync -f` advisory is edge-triggered, not level-triggered.** The advisory flag for closed feedback files (like `ac6-governa-sync-0.42.0.md`) appears only on the sync run where the CHANGELOG credit is first detected (v0.44.0 → v0.45.1 transition here). Once that sync completes and the baseline is committed, a subsequent `governa sync -f` reports *"prune: no addressed feedback files to remove"* even though the file still sits on disk unpruned. Consumers who don't pass `-f` on the first flagging sync have to delete the file manually on subsequent adoptions. Consider making the advisory level-triggered (flag whenever a feedback file's credit version is `≤ current baseline`) so `-f` remains a working recovery path across multiple sync cycles.

## Metadata

- Sync range: governa 0.44.0 → 0.45.1
- Consumer repo: utils (github.com/queone/utils)
- AC: AC8
- Generated: during AC8 implementation, author: DEV
