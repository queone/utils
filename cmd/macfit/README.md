## macfit
Keep Mac config files in one encrypted store and restore them on any Mac. macOS only.

The store is a single sealed file. By default it lives in `~/.local/share/macfit/`, which stays on one Mac. Point it at a synced folder once with `init -s PATH` and every Mac that sees the folder can open it. A folder inside iCloud Drive, such as `~/Library/Mobile Documents/com~apple~CloudDocs/etc/`, is a good choice; Dropbox, Google Drive, or any other synced folder works the same way.

```text
macfit init -N -s ~/data/etc/macfit.store     # first Mac: create the store and its key, remember the path
macfit add ~/.bash_logout ~/.bashrc ~/.profile # register live files and capture them
macfit push                                   # send changed live files into the store
macfit diff                                   # show what differs between the store and this Mac
macfit pull                                   # plan the restore: what would be written, nothing touched
macfit pull -f ~/.bashrc                      # write one file, or all of them with no target
macfit init -s ~/data/etc/macfit.store        # another Mac: unlock with the passphrase, remember the path
```

`push` sends live files up into the store. `pull` brings the store down onto the Mac. The store is the remote, as in git.

### Where the store is

Each command finds the store in this order: `-s PATH` on the command line, then `MACFIT_STORE`, then the path that `init -s` remembered in `$XDG_CONFIG_HOME/macfit/store`, then `$XDG_DATA_HOME/macfit/macfit.store`. Only `init` writes the remembered path, and only when it was run with `-s`. `init -N` creates the default folder when needed; for any other location the folder must already exist, so a missing synced folder is never faked.

### What a store holds

The store is a SQLite database sealed inside an authenticated encryption envelope (XChaCha20-Poly1305). It is always encrypted on disk. A command decrypts it into memory, does its work, seals it again, and replaces the file atomically. Nothing but the store file is written. That is what makes a synced folder safe: the sync client only ever sees whole files.

Every registered file is stored as a target template, a file mode, an optional host binding, and every version captured so far. Targets use the directory variables of this machine, so a file at `~/.config/git/config` is stored as `$XDG_CONFIG_HOME/git/config` and lands wherever that variable points on the Mac that pulls it. The variables recognized are `$XDG_CONFIG_HOME`, `$XDG_DATA_HOME`, `$XDG_STATE_HOME`, `$XDG_CACHE_HOME`, `$CLAUDE_CONFIG_DIR`, and `~`, with the standard XDG fallbacks when a variable is unset. `add -l` keeps the path spelled under `~` instead.

The list of files is data inside your store, created by your `add` commands. macfit itself carries no path of yours.

### The key

`init -N` draws a random 256-bit key, saves it in the login keychain through the `security` command, and asks for a recovery passphrase. A copy of the key wrapped with that passphrase (Argon2id, then XChaCha20-Poly1305) sits in the store header. On another Mac that sees the same store file, `init` asks for the passphrase once and saves the key to that Mac's keychain. From then on every command works without a prompt.

`macfit key` maintains that keychain item: `key show` reports the store path, the key id, whether the item exists, and whether it opens the store, without printing the secret; `key restore` puts the key back with the passphrase; `key rm` deletes the item after a `[y/N]` prompt (`-f` skips it); `key passphrase` changes the recovery passphrase.

The key is protected exactly as well as your login keychain: any process running as your user can read it with the same `security` command, without a prompt. iCloud Keychain does not carry it; the passphrase is the cross-Mac path. During `init -N` and `key restore` the key passes to `security` as a command-line argument, so it is visible in the process list for that instant.

If the passphrase or the key ever leaks, create a new store with `init -N -s NEWPATH`, `add` your files again, and delete the old store and its keychain item with `key rm`.

### Daily use

- `macfit add PATH... [-H HOST] [-l]` registers regular files and captures their content and mode. `-H` binds the entries to one Mac's LocalHostName; a host-bound entry wins over the unbound one on that Mac and is ignored elsewhere.
- `macfit push [TARGET...]` captures live files whose content or mode changed as new versions. Older versions stay in the store.
- `macfit pull [TARGET...]` prints the plan and writes nothing: `unchanged`, `would write` for a missing file, `would overwrite` for a differing one, `symlink` for a live symlink. `macfit pull -f` writes the plan, creating parent directories (0700 for a 0600 file, else 0755) and setting the mode; it never writes through a symlink. `-n` is accepted and means the plan, even next to `-f`.
- `macfit diff [TARGET...] [-V]` prints `=` same, `M` differs, `?` live file missing, one line per entry, and exits 1 when anything drifted. `-V` adds a unified diff block, set off by blank lines.
- Colors on a terminal, plain when piped: grey for `=` and `unchanged`, yellow for `M`, `differs`, `would overwrite`, and `missing`, orange for `?`, green for `would write`, `restored`, `added`, and `updated`, red for `symlink`. Inside a diff block the headers are dark grey, unchanged lines grey, and removed and added lines light yellow.
- `macfit ls` lists every entry with its mode, host, and last capture time. `macfit rm TARGET [-H HOST]` removes one.

A TARGET is either the template as `ls` shows it (`$XDG_CONFIG_HOME/git/config`) or the live path.

### Things to know

- Two Macs writing the store while offline produce a sync conflict copy, named `macfit 2.store`, `macfit (1).store`, or similar depending on the sync client. macfit warns when one sits beside the store, and a stale writer that opened an older generation is refused rather than allowed to overwrite the newer file.
- A synced folder is not end-to-end encrypted unless the provider says so. The store is encrypted before it reaches the folder, so that does not matter for its contents.
- `init` and the `key` prompts need a terminal.
- Files only: no directories, globs, or symlinks. macOS `defaults` settings are a planned addition.

### Usage

```text
macfit v1.2.0
Keep Mac config files in one encrypted store and restore them on any Mac.

Overview
  The store is a single sealed file. Keep it in a synced folder and every Mac
  that sees the folder can open it. push sends live files into the store,
  pull restores them from the store, and diff shows what differs. The key
  lives in the login keychain, with a passphrase-wrapped copy in the store
  for other Macs.

Usage
  macfit init [-N]                    unlock an existing store, or create one with -N
  macfit add PATH... [-H HOST] [-l]   register live files and capture them
  macfit rm TARGET [-H HOST]          forget a file and its stored versions
  macfit ls                           list entries
  macfit push [TARGET...]             send changed live files into the store
  macfit pull [TARGET...] [-f]        plan the restore, or write it with -f
  macfit diff [TARGET...] [-V]        show drift between the store and this Mac
  macfit key show                     store path, key id, keychain and store state
  macfit key restore                  put the key back in the keychain with the passphrase
  macfit key rm [-f]                  delete the keychain item after a prompt
  macfit key passphrase               change the recovery passphrase

Options
  -s, --store PATH   Store file for this command; init -s also remembers it
  -N, --new          Create a new store (init)
  -H, --host NAME    Bind the entry to one Mac (add, rm)
  -l, --literal      Keep the path under ~ instead of an XDG variable (add)
  -n, --dry-run      Print the pull plan; the default, kept for scripts
  -f, --force        Write the pull plan, overwriting live files that differ; skip the key rm prompt
  -V, --verbose      Add a unified diff to diff output
  -v, --version      Print macfit v1.2.0 and exit
  -h, -?, --help     Show this help message and exit

Notes
  Store path order: -s, then MACFIT_STORE, then the path init -s remembered in
  $XDG_CONFIG_HOME/macfit/store, then $XDG_DATA_HOME/macfit/macfit.store.
  A TARGET is the template ls shows ($XDG_CONFIG_HOME/git/config) or the live path.
  init needs a terminal for the passphrase prompt and creates only the default folder.
  Files only: no directories, globs, or symlinks. macOS defaults settings are a planned addition.
```
