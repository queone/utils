# Changelog

| Version | Summary |
|---------|---------|
| Unreleased | |
| 0.14.0 | AC9: adopt governa v0.45.3 sync; first integrated-critique AC |
| 0.13.0 | AC8: adopt governa v0.45.1 sync; integrated critique + Feedback Credits |
| 0.12.0 | AC7: adopt governa v0.43.1 sync; migrate to Local Rules |
| 0.11.0 | AC6: adopt governa v0.42.0; 2-step release flow, mdcheck |
| 0.10.0 | AC5: add `claudecfg` unified Claude Code config utility — env subcommands ported from `claude-env` (memory + projects symlinks, conflict preservation, conflicts listing); perms subcommands (init/list/show/check) with embedded `go` profile + iCloud override dir; longest-prefix WARN/INFO matching; non-darwin platform gate scoped to env; remove `cmd/claude-env/`; lift flag-convention rule into AGENTS.md Project Rules; extend canonical-build rule with smoke-test guidance |
| 0.9.0 | AC4: adopt governa v0.32.1 sync — swap internal/{buildtool,reltool} to canonical packages (drop DI abstractions: CmdRunner/Pipeline/GitRunner/Release), adopt cmd/{build,rel}/main.go template headers, AGENTS.md Review Style skip-note bullet, docs/build-release.md step 9 no-trailing-commentary rule; retain AGENTS.md README-alphabetical rule as Standing Divergence; incorporate governa hotfix for buildtool_test.go t.Setenv/t.Parallel race; remove IE1; correct staticcheck install behavior note |
| 0.8.0 | AC3: adopt governa v0.31.0 sync — .gitignore Go block, docs/build-release.md step 5 `Canonical shape:` + step 4 Template Upgrade wording, plan.md restructured to six-section template with seven fix-plan items folded into Priorities; added IE1 for internal/{buildtool,reltool} canonical re-sync; Standing Divergences pruned to README + step 5 bootstrap note after v0.31.0 resolved four prior entries |
| 0.7.0 | AC2: adopt governa v0.30.0 sync — Counterparts sections in dev/qa/maintainer role docs, AGENTS.md CHANGELOG-format reference, ac-template cosmetic update; retain step 5 visual code-block as repo addition; codify per-sync feedback rule in docs/build-release.md Template Upgrade; add `.gitignore` and `docs/build-release.md` Standing Divergences entries; LICENSE copyright year |
| 0.6.0 | AC1: adopt governa v0.29.0 governance baseline — AGENTS.md as canonical contract, CLAUDE.md → symlink, role docs (dev/maintainer/qa/director), build-release/dev-cycle/dev-guidelines docs, arch.md scaffold, .gitignore extensions for governa/OS/editor artifacts, CHANGELOG.md bootstrap |
