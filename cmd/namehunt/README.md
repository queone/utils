## namehunt

Find free usernames on any site with a predictable profile URL. Give it a site and one name, and it tells you whether that name is free. Give it a pattern, and it prints every free name the pattern expands to.

### Usage

```
namehunt [flags] SITE CANDIDATE
```

Run `namehunt` with no arguments, or with `-h`, to see the help screen.

| Flag | Meaning |
|------|---------|
| `-d, --distinct` | Keep only names with no repeated character. |
| `-r, --require CHARS` | Keep only names containing every character in `CHARS`. |
| `-n, --dry-run` | Print the names and send no request. |
| `-f, --free CODES` | Status codes that mean free, comma-separated. Default `404`. |
| `-w, --wait MS` | Pause `MS` milliseconds between requests. Default `0`. |
| `-l, --list` | List known sites and exit. |
| `-v, --version` | Print the version and exit. |
| `-h, --help` | Show the help screen. |

`SITE` is one of:

- a built-in site name: `github` or `lichess`
- a name from your sites file (see below)
- a URL template with `{}` where the username goes, such as `https://www.reddit.com/user/{}`. The template must start with `http://` or `https://`.

`CANDIDATE` is one of:

- one name, such as `kaqe`
- a pattern, such as `[qk][aeou][qk][aeou]`. Each `[...]` class stands for one character from the class. Every other character is literal.
- `-` to read one name per line from stdin

### Examples

```bash
namehunt github kaqe                                  # is kaqe free on GitHub?
namehunt -d lichess '[qk][aeiou][qk][aeiou]'          # free 4-letter names, no repeated letters
namehunt 'https://www.reddit.com/user/{}' kaqe        # any site, by template
cat names.txt | namehunt -w 200 github -              # names from a file, paced 200 ms apart
namehunt -n -d -r qk github '[qkaeou][qkaeou][qkaeou][qkaeou]'   # just list the candidates
```

Three scans worth keeping:

```bash
# GitHub, every three-letter name from the letters a e o u q k t p m (729 names)
namehunt github '[aeouqktpm][aeouqktpm][aeouqktpm]'

# GitHub, four distinct letters from q k a e o u that include both q and k (144 names)
namehunt -d -r qk github '[qkaeou][qkaeou][qkaeou][qkaeou]'

# Lichess, q or k alternating with a vowel, no repeats (40 names)
namehunt -d lichess '[qk][aeiou][qk][aeiou]'
```

### Sites file

Name your own sites in `~/.config/namehunt/sites` (or `$XDG_CONFIG_HOME/namehunt/sites` when that variable is set to an absolute path). One site per line: the name, the URL template, and optionally the comma-separated status codes that mean free. Blank lines and lines starting with `#` are skipped. A line with a built-in name replaces the built-in.

```
# name     profile URL template               codes that mean free
github     https://github.com/{}
lichess    https://lichess.org/@/{}
reddit     https://www.reddit.com/user/{}     404
```

`namehunt -l` prints every known site and whether it is built in or from the file. A bad line stops the tool and names the line number.

### How "free" is decided

For each name, `namehunt` sends a `HEAD` request to the profile URL with a 5 second timeout and follows redirects.

- A status in the free list (`-f`, then the site's file entry, then `404`) means free.
- Any other 2xx status means taken.
- A 405 is retried once with `GET`.
- A 429 or a network error is retried up to five times, waiting 100 ms and doubling each time.
- Anything else, or five failed attempts, means unknown, and the status or error is shown.

Output and exit codes:

- One name: one line, `NAME: free`, `NAME: taken`, or `NAME: unknown (HTTP 403)`. Exit 0 free, 1 taken, 2 unknown.
- Pattern or stdin: free names one per line on stdout. Unknown names go to stderr with their reason, followed by `Checked N: F free, T taken, U unknown.` Exit 0 when nothing is unknown, else 2.
- Bad input: one line on stderr, exit 2, no request sent.

Limits: the answer comes from the status code only. A site that answers 200 for a missing profile cannot be told apart from a taken name. Pick a site whose missing-profile status differs, or set that status with `-f` or in the sites file.
