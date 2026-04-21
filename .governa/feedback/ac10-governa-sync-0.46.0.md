# AC10 governa sync v0.46.0 — per-sync feedback

Per-sync feedback artifact per `docs/build-release.md` Template Upgrade rule. Moved to `.governa/feedback/ac10-governa-sync-0.46.0.md` at release prep (not deleted), so the feedback persists for future `enhance -r` runs.

## Upstream proposal: extend IE semantics to support AC pointers

The current template (v0.46.0, `docs/development-cycle.md` Notes) defines `Ideas To Explore` as pre-rubric staging only: *"record loose, pre-rubric follow-on ideas in plan.md under Ideas To Explore with an IE<N>: prefix"* and *"remove IE entries when promoted to an AC or completed; the list is staging, not history"*. That rule makes `plan.md` unable to surface post-draft pre-implementation work — once an item crosses the rubric and gets drafted as an AC, plan.md loses it.

During AC10 the director wanted `plan.md` to remain a **one-page backlog glance**: a single place to recall everything that has been batched up for upcoming work, without having to enumerate `docs/ac*.md` files or grep commit history. Eagerly promoting 7 stub ACs while keeping the Priorities-era glance surface required a semantics change.

### Proposal

Allow IE entries to persist as pointers to their originating stub ACs. Specifically, relax the Notes bullet so IE entries are **removed at AC completion rather than AC draft**. An IE entry would then legitimately take either shape:

1. Pre-rubric idea: `IE<N>: <short description>` — same as today.
2. Stub-AC pointer: `IE<N>: <one-line problem statement> — tracked as docs/ac<M>-<slug>.md.` — persists until the referenced AC ships (or the stub is deleted during de-prioritization).

The `IE<N>` prefix stays stable across the lifecycle, so director references like "IE3" remain meaningful between draft and ship.

Until adopted upstream, utils codifies this locally in `docs/development-cycle.md` `## Local Rules` with a temporary marker tied to the governa version that ships this change (tracked as governa AC74 in the director's intent).

## Related observation: stub-AC pattern is worth lifting into `docs/ac-template.md`

AC10 exercised a **stub-AC** pattern — ACs drafted with `TBD — requires scoping before critique gate` placeholders in Out Of Scope and Acceptance Tests, plus a literal marker in Implementation Notes, and `PENDING` status. The pattern let the director eagerly allocate AC numbers (AC11–AC17) and write glance-level summaries for 7 backlog items without incurring the full scoping cost up front.

Governa itself uses `PENDING`-state stubs informally, but neither `docs/ac-template.md` nor `docs/sync-methodology.md` codifies "stub-AC" as a first-class variant. That leaves each consumer to either invent the pattern ad hoc or lean on a Local Rule (utils' current posture for this sync — see the second `## Local Rules` bullet in `docs/development-cycle.md`).

Suggested upstream: lift stub-AC into the template. Two possible shapes — (a) amend `docs/ac-template.md` preamble to list `TBD — requires scoping before critique gate` as an accepted placeholder in Out Of Scope and Acceptance Tests, or (b) add a dedicated `## Stub ACs` subsection to `docs/ac-template.md` that defines the minimum shape and the exit criteria. Either way, a single codified spec would prevent the pattern from drifting across consumers. Not urgent — utils' Local Rules entry covers our need today.

## Observations about the sync itself

### Landed well

- **Clean one-file adopt, no surprises in AC9→AC10 transition.** The v0.45.3 → v0.46.0 sync produced a single-file adopt (`docs/development-cycle.md`) with a tight 3-line diff (step 1 rewrite, Notes bullet removal, promotion-path update). No bookkeeping churn: `TEMPLATE_VERSION` and `.governa/manifest` updated cleanly; `.governa/feedback/ac9-governa-sync-0.45.3.md` was not auto-advised for removal (governa AC73 did not address our outstanding v0.45.3 feedback about recommendation-reason phrasing on `docs/roles/qa.md` — observation preserved).
- **Integrated-critique cycle at full maturity.** AC10 is the second utils AC under the integrated-critique model (AC9 was the first). Round 1 surfaced 6 findings (1 blocker + 4 major + 1 minor); Round 2 cleared cleanly with a `no blockers` verdict. DEV response → AC revision → Disposition Log → QA verification chain ran without any protocol friction.
- **AC73's Notes-bullet removal is minimally disruptive on the dev-cycle.md side**, but the plan.md knock-on is substantial for consumers carrying real Priorities content. Utils had 7 pre-rubric-cleared items in `Priorities` at the time of adoption; promoting them to stub ACs was non-trivial scope addition, but feasible inside a single AC once the stub-AC pattern was established. A shorter path from `Priorities` content to stub ACs (e.g., a governa-side smoke test / migration helper) would reduce the burst cost for future consumers hitting this upgrade.

### Friction surfaced during adoption

- **`## Constraints` removal in the template skeleton implies a routing decision that governa does not spell out.** v0.46.0's plan.md skeleton drops `Constraints`, but `AC73: simplify plan.md to Product Direction + Ideas To Explore` does not say where real constraint content should go if a consumer had it. Director's reasoning during AC10 (architectural invariants → `arch.md`; governance rules → `AGENTS.md` `## Project Rules`) is sound but not in any upstream doc. Suggested upstream: a one-line note in the template's `plan.md` or in `docs/development-cycle.md` Notes saying "For real constraints content, prefer `arch.md` (architectural invariants) or `AGENTS.md` `## Project Rules` (governance rules); the plan.md Constraints section was removed because such content is better homed in those files." Saves future consumers a judgment call.

## Metadata

- Sync range: governa 0.45.3 → 0.46.0
- Consumer repo: utils (github.com/queone/utils)
- AC: AC10
- Generated: during AC10 implementation, author: DEV
