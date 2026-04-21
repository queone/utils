# AC9 governa sync v0.45.3 — per-sync feedback

Per-sync feedback artifact per `docs/build-release.md` Template Upgrade rule (step 4). Moved to `.governa/feedback/ac9-governa-sync-0.45.3.md` at release prep (not deleted), so the feedback persists for future `enhance -r` runs.

## Observations about the sync itself

### Landed well

- **Feedback-loop round-trip 2 of 2 closed cleanly.** Our AC8 `-feedback.md` observations (governa ack --review bug, sync-review reason ambiguity, edge-triggered sync -f) surfaced to governa via AC8 release prep. Governa addressed them in v0.45.2 (AC71 "ack+advisor+verbosity fixes (addresses utils feedback from v0.45.1 syncs)"). v0.45.3's AC63 scanner then auto-flagged `.governa/feedback/ac8-governa-sync-0.45.1.md` under `## Advisory Notes`, and AC71's level-triggered `sync -f` fix correctly pruned it during AC9 implementation. End-to-end: feedback surfaced → addressed → credited → flagged → pruned. The designed loop works as advertised. (First full round-trip was AC6 → AC69/AC70; AC8 → AC71/AC72 is the second.)
- **AC71's three runtime fixes validated on utils.** During pre-AC-draft sync run (the post-v0.13.0 sync that informed the AC9 scope), all three AC71-addressed behaviors were exercised and observed working:
  1. **Credit scanner flags ac8.** `.governa/sync-review.md` `## Advisory Notes` emitted: *"`.governa/feedback/ac8-governa-sync-0.45.1.md` — addressed by governa v0.45.2; review and delete if resolved."* Previously our AC8 manually-rm'd a feedback file that the v0.42.0 sync had flagged but the v0.43.1 sync no longer did (edge-triggered loss).
  2. **`governa ack --review` accepts no-path invocation.** Returns *"no acknowledged-drift entries in manifest"* (exit 0). Previously errored *"path is required: use `governa ack <path>`"* — AC8's `-feedback.md` surfaced the help-text-vs-runtime discrepancy; AC71 fixed the runtime.
  3. **`sync -f` is level-triggered and prunes as advised.** `governa sync -f` on the post-v0.13.0 tree emitted *"prune: removed /Users/tek1/code/utils/.governa/feedback/ac8-governa-sync-0.45.1.md"* and the file is gone. Under the prior edge-triggered behavior the advisory would have been cleared by the intervening baseline-commit sync, and a later `-f` run would have reported *"no addressed feedback files to remove"* without acting.

  The AT verifies the feedback file contains these notes; it does not re-run the behaviors (separate smoke-testing is out of scope per AC9 Out Of Scope).
- **Integrated-critique in practice.** AC9 is the first consumer AC using `## Critique` inside the AC itself rather than a standalone `docs/ac9-*-critique.md` file. The single-file model collapses context, makes the QA-round history inspectable in one `git diff`, and removes the file-lifecycle tracking overhead. Round 1 and Round 2 both landed cleanly into the AC's Critique section; the Disposition Log under Implementation Notes captured DEV's responses symmetrically. No friction observed during the transition.

### Friction surfaced during adoption

- **Sync-review `(cosmetic: bullet added)` reason phrasing on `docs/roles/qa.md` still understates the scope.** The actual v0.45.3 diff adds a new `Calibrate verbosity` bullet AND reworks the existing QA write-surface rule to match the integrated-critique convention (wording change from "chat or `docs/ac<N>-<slug>-critique.md`" to "chat only"). The recommendation-reason characterizes this as one bullet added, but it's add-one + reword-one. AC71's fixes targeted ack/advisor/verbosity, not recommendation-reason clarity — so this is a still-outstanding echo of AC8's bullet-reason-ambiguity observation. Suggested upstream: extend the sync-review recommendation-reason schema to distinguish `(bullet added)` from `(bullet added + existing rule reworded)` or similar, so a consumer reviewer reading only the reason line doesn't miss the reword.

## Metadata

- Sync range: governa 0.45.1 → 0.45.3
- Consumer repo: utils (github.com/queone/utils)
- AC: AC9
- Generated: during AC9 implementation, author: DEV
