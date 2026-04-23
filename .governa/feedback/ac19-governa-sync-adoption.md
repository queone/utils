# AC19 governa sync v0.47.0 — per-sync feedback

Per-sync feedback artifact per `docs/build-release.md` Template Upgrade rule.

## Landed well

- **AC74 closed the feedback loop cleanly.** The IE dual-use semantics shipped exactly as proposed in the AC10 feedback artifact. The adoption diff was 3 lines in Notes — zero ambiguity, zero judgment calls. This is the ideal sync experience: consumer surfaces a proposal, upstream ships it, next sync is mechanical.
- **Advisory note for feedback pruning was accurate and actionable.** The sync-review correctly flagged `ac10-governa-sync-0.46.0.md` as addressed, and the director confirmed the two unaddressed sections could be intentionally dropped (they'll resurface naturally if a future sync reproduces them).

## Observations

- **mdcheck scans `git ls-files *.md`, so unstaged deletions cause build failures.** When an AC deletes a tracked `.md` file, the deletion must be staged before `./build.sh` will pass. Not a governa issue per se, but worth noting: the build pipeline's mdcheck step and the sync workflow's file-deletion step interact in a way that requires staging awareness. This came up during AC19 implementation — the feedback file deletion was applied to disk but not staged, causing mdcheck to fail on the missing path.

## Metadata

- Sync range: governa 0.46.0 → 0.47.0
- Consumer repo: utils (github.com/queone/utils)
- AC: AC19
- Generated: during AC19 implementation, author: DEV
