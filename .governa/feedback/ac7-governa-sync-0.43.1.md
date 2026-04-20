# AC7 governa sync v0.43.1 — per-sync feedback

Per-sync feedback artifact per `docs/build-release.md` Template Upgrade rule (step 4). Captures observations about the v0.42.0 → v0.43.1 sync output that are worth routing upstream to governa for consideration. Moved to `.governa/feedback/ac7-governa-sync-0.43.1.md` at release prep (not deleted), so the feedback persists for future `enhance -r` runs.

## Upstream suggestions

### Consider inline-location metadata for Local Rules entries

The v0.43.0 Local Rules convention (`docs/development-cycle.md` § Local Rules) solves the drift problem elegantly — `## Local Rules` at the end of a supplementary governance doc is sync-aware and repo-owned, no advisory noise on future syncs. But it creates a reader-ergonomics cost: rules can't stay topically adjacent to the content they extend.

Concrete example from AC7: the CHANGELOG bootstrap rule naturally belongs *inside* the `CHANGELOG row shape` bullet list in `docs/build-release.md`'s `Pre-Release Checklist` Appendix. It extends that list with guidance for the zero-state case. Under the current Local Rules convention, it must live in a file-end `## Local Rules` section instead. A reader scanning the CHANGELOG row shape list for bootstrap guidance now has to know to scroll to the end of the doc to find it.

Proposed template enhancement: let Local Rules entries carry optional inline-location metadata pointing at the section (or list, or anchor) they extend. Something like:

```
- Bootstrap: if CHANGELOG.md does not yet exist, create it... <!-- local-rule: anchor=changelog-row-shape -->
```

Or a more structured form:

```
- <!-- local-rule: id=ac3-bootstrap anchor=changelog-row-shape -->
  Bootstrap: if CHANGELOG.md does not yet exist, create it...
```

The scorer stays happy (the rule is still under `## Local Rules`); the metadata gives tooling (or a future agent) the signal to optionally render the rule inline at the anchor point in IDE preview, docs builds, or similar surfaces. Consumers who don't want the indirection can ignore the metadata.

Why this is worth lifting upstream:

- Any consumer repo that needs to extend a template bullet list faces the same adjacency loss — not unique to utils.
- The metadata convention is additive and optional; repos that don't use it aren't affected.
- It preserves the Local Rules section as the canonical registry while restoring topical adjacency for readers.

## Observations about the sync itself

### Landed well

- **v0.43.1's skeleton-policy fix for `plan.md` worked end-to-end.** After the previous v0.43.0 sync attempt re-flagged `plan.md` as "stable standing drift" (despite being legitimate repo-specific roadmap content), the hotfix correctly reclassifies it as `keep` with the reason *"plan.md skeleton sections only — standing drift as expected (Product Direction, Priorities)."* No ack needed; no adoption work; the file just sits cleanly. This is the right behavior — consumer-repo roadmap content shouldn't require an ack dance every sync.
- **Local Rules convention is the right shape.** Much cleaner than the ad-hoc `## Standing Divergences from Template` section we'd introduced in AC6 as a bridge mechanism. Section-granular, sync-aware, template-blessed — exactly what was needed.
- **Template Changes summary in the review document is useful.** Seeing "0.43.1 — AC67: plan.md skeleton policy end-to-end fix" + "0.43.0 — AC66: plan.md skeleton coverage + `## Local Rules` convention" at the top of `.governa/sync-review.md` made the adoption scope immediately clear.

### Friction surfaced during adoption

- **"Not AGENTS.md" rule is easy to miss.** The Local Rules convention's "Not AGENTS.md" paragraph is load-bearing — without it, the natural instinct on a migration like AC7 would be to add `## Local Rules` to AGENTS.md for consistency, which would violate its `## Governed Sections` invariant. The rule is clearly stated in the template, but buried after the intro paragraph. Consider leading with it or adding a short reiteration in `## Governed Sections` itself so the invariant is visible from both directions.
- **Topical-adjacency loss is structural.** See the upstream suggestion above.

## Metadata

- Sync range: governa 0.42.0 → 0.43.1
- Consumer repo: utils (github.com/queone/utils)
- AC: AC7
- Generated: during AC7 implementation, author: DEV
