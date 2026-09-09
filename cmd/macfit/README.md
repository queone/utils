## macfit
Keep Mac config files in one encrypted store and restore them on any Mac. macOS only.

The store is a single sealed file. Keep it in iCloud Drive, the default location, and every Mac on the account sees it. Any other synced folder works the same way.

```text
macfit init                         # first Mac: create the store and its key; other Macs: unlock with the passphrase
macfit add ~/.bashrc                # register a live file and capture it
macfit push                         # send changed live files into the store
macfit diff                         # show what differs between the store and this Mac
macfit pull                         # restore files from the store onto this Mac
```

`push` sends live files up into the store. `pull` brings the store down onto the Mac. The store is the remote, as in git.

### What a store holds

The store is a SQLite database sealed inside an authenticated encryption envelope (XChaCha20-Poly1305). By default it is `~/Library/Mobile Documents/com~apple~CloudDocs/macfit.store`; point macfit elsewhere with `-s PATH` or `MACFIT_STORE`. It is always encrypted on disk. A command decrypts it into memory, does its work, seals it again, and replaces the file atomically. Nothing but the store file is written.

Every registered file is stored as a target template, a file mode, an optional host binding, and every version captured so far. Targets use the directory variables of this machine, so a file at `~/.config/git/config` is stored as `$XDG_CONFIG_HOME/git/config` and lands wherever that variable points on the Mac that pulls it. The variables recognized are `$XDG_CONFIG_HOME`, `$XDG_DATA_HOME`, `$XDG_STATE_HOME`, `$XDG_CACHE_HOME`, `$CLAUDE_CONFIG_DIR`, and `~`, with the standard XDG fallbacks when a variable is unset. `add -l` keeps the path spelled under `~` instead.

The list of files is data inside your store, created by your `add` commands. macfit itself carries no path of yours.

### The key

`init` on the first Mac draws a random 256-bit key, saves it in the login keychain through the `security` command, and asks for a recovery passphrase. A copy of the key wrapped with that passphrase (Argon2id, then XChaCha20-Poly1305) sits in the store header. On another Mac that sees the same store file, `init` asks for the passphrase once and saves the key to that Mac's keychain. From then on every command works without a prompt.

The key is protected exactly as well as your login keychain: any process running as your user can read it with the same `security` command, without a prompt. iCloud Keychain does not carry it; the passphrase is the cross-Mac path. During `init` the key passes to `security` as a command-line argument, so it is visible in the process list for that instant.

If the passphrase or the key ever leaks, create a new store with `init -s NEWPATH`, `add` your files again, and delete the old store and its keychain item (`security delete-generic-password -s macfit`).

### Daily use

- `macfit add PATH [-H HOST] [-l]` registers one regular file and captures its content and mode. `-H` binds the entry to one Mac's LocalHostName; a host-bound entry wins over the unbound one on that Mac and is ignored elsewhere.
- `macfit push [TARGET...]` captures live files whose content or mode changed as new versions. Older versions stay in the store.
- `macfit pull [TARGET...] [-n] [-f]` writes the latest version to the live path, creating parent directories (0700 for a 0600 file, else 0755) and setting the mode. It refuses to overwrite a live file that differs unless `-f`, and never writes through a symlink. `-n` prints the plan and writes nothing.
- `macfit diff [TARGET...] [-V]` prints `=` same, `M` differs, `?` live file missing, one line per entry, and exits 1 when anything drifted. `-V` adds a unified diff.
- `macfit ls` lists every entry with its mode, host, and last capture time. `macfit rm TARGET [-H HOST]` removes one.

A TARGET is either the template as `ls` shows it (`$XDG_CONFIG_HOME/git/config`) or the live path.

### Things to know

- Two Macs writing the store while offline produce an iCloud conflict copy (`macfit 2.store`). macfit warns when one sits beside the store, and a stale writer that opened an older generation is refused rather than allowed to overwrite the newer file.
- iCloud Drive is not end-to-end encrypted unless Advanced Data Protection is on. The store is encrypted before it reaches iCloud, so that does not matter for its contents.
- `init` needs a terminal for the passphrase prompt, and it will not create the iCloud Drive folder: if `Mobile Documents` is missing, turn on iCloud Drive or pass `-s PATH`.
- Files only: no directories, globs, or symlinks. macOS `defaults` settings are a planned addition.

### Usage

```text
macfit v1.0.0
Keep Mac config files in one encrypted store and restore them on any Mac.

Overview
  The store is a single sealed file. Keep it in iCloud Drive, the default
  location, and every Mac on the account sees it; any other synced folder
  works the same way. push sends live files into the store, pull restores
  them from the store, and diff shows what differs. The key lives in the
  login keychain, with a passphrase-wrapped copy in the store for other Macs.

Usage
  macfit init                         create the store and key, or unlock an existing store
  macfit add PATH [-H HOST] [-l]      register a live file and capture it
  macfit rm TARGET [-H HOST]          forget a file and its stored versions
  macfit ls                           list entries
  macfit push [TARGET...]             send changed live files into the store
  macfit pull [TARGET...] [-n] [-f]   restore files from the store
  macfit diff [TARGET...] [-V]        show drift between the store and this Mac

Options
  -s, --store PATH   Store file (default: MACFIT_STORE, else macfit.store in iCloud Drive)
  -H, --host NAME    Bind the entry to one Mac (add, rm)
  -l, --literal      Keep the path under ~ instead of an XDG variable (add)
  -n, --dry-run      Print what pull would write and write nothing
  -f, --force        Let pull overwrite a live file that differs
  -V, --verbose      Add a unified diff to diff output
  -v, --version      Print macfit v1.0.0 and exit
  -h, -?, --help     Show this help message and exit

Notes
  A TARGET is the template ls shows ($XDG_CONFIG_HOME/git/config) or the live path.
  init needs a terminal for the passphrase prompt and never creates the iCloud Drive folder.
  Files only: no directories, globs, or symlinks. macOS defaults settings are a planned addition.
```
