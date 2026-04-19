# claudecfg profiles

Each `*.json` file in this directory is an embedded profile for `claudecfg perms init`. The shape mirrors Claude Code's `settings.json`:

```json
{
  "permissions": {
    "allow": ["Bash(go list *)", "Bash(staticcheck *)"]
  }
}
```

The filename stem (without `.json`) is the profile name used with `-p <name>`. The entries under `permissions.allow` are appended to the target project's `.claude/settings.json` (deduplicated by exact string).

## Adding a profile

- **For upstream inclusion**: add a file to this directory. It will be compiled into the binary via `//go:embed`.
- **For personal-machine use**: drop a JSON file into `~/data/etc/claude/perms-starters/` (iCloud-synced). The filename stem becomes the profile name. Override-dir entries shadow embedded ones.

Malformed JSON files in the override dir are skipped with a `WARNING:` to stderr and don't block other profiles from loading.
