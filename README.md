# utils
A collection of small CLI utilities written in Go.

## Why
Go's tool chain is the ideal way to maintain a set of commonly used CLI utilities. They can be quickly compiled and installed whether you're in Windows, macOS, or Linux. This provides a unified and portable solution to many a scripting needs. With this setup, Go turns into a quasi-package manager for these utilities.

## Utilities

- [`bak`](cmd/bak/README.md): Create dated backups of files or directories.
- [`brew-update`](cmd/brew-update/README.md): Update, upgrade, and clean up Homebrew packages.
- [`cash5`](cmd/cash5/README.md): Analyze historical Cash5 draws and generate number recommendations.
- [`certgen`](cmd/certgen/README.md): Generate self-signed TLS certificates for local testing.
- [`certls`](cmd/certls/README.md): Show SSL/TLS certificate details for a host and port.
- [`claudecfg`](cmd/claudecfg/main.go): Manage Claude Code config — iCloud-backed memory portability and project permissions seeding/auditing.
- [`days`](cmd/days/README.md): A CLI calendar days calculator.
- [`decolor`](cmd/decolor/README.md): A utility that removes shell color escape codes from input stream or given file.
- [`dl`](cmd/dl/README.md): Download online videos using `yt-dlp` with a target filename.
- [`dos2unix`](cmd/dos2unix/README.md): Preview or convert CRLF line endings to LF.
- [`fr`](cmd/fr/README.md): A simple find/replace utility.
- [`git-cloneall`](cmd/git-cloneall/README.md): Clone all repositories from a GitHub user or organization.
- [`git-pullall`](cmd/git-pullall/README.md): Pull updates across all local Git repositories in a directory.
- [`git-remotev`](cmd/git-remotev/README.md): Print each local repository with its `origin` remote URL.
- [`git-statall`](cmd/git-statall/README.md): Show git status across local repositories.
- [`jy`](cmd/jy/README.md): A lightweight JSON and YAML converter utility.
- [`pgen`](cmd/pgen/README.md): A simple generator of memorable passwords.
- [`pman`](cmd/pman/README.md): Run authenticated Microsoft Graph and Azure REST API requests.
- [`rn`](cmd/rn/README.md): A bulk file re-namer.
- [`rncap`](cmd/rncap/README.md): Rename files by capitalizing each word in filenames.
- [`rnlower`](cmd/rnlower/README.md): Rename files by converting filenames to lowercase.
- [`sms`](cmd/sms/README.md): Send SMS messages using Twilio credentials from a local config file.
- [`tree`](cmd/tree/README.md): A lightweight directory tree printing utility.
- [`web`](cmd/web/README.md): Search DuckDuckGo and open results with an interactive selector.

## Quick Install
With Go installed, install all utilities at once:

```bash
go install github.com/queone/utils/cmd/...@latest
```

Or install a single utility:

```bash
go install github.com/queone/utils/cmd/fr@latest
```

Binaries are placed in `$GOPATH/bin` (typically `~/go/bin`), which should be in your `$PATH`.

## Getting Started
To compile the entire collection, you obviously need to have GoLang installed and properly setup in your system, with `$GOPATH` set up correctly (typically at `$HOME/go`). Also setup `$GOPATH/bin/` in your `$PATH`, since that is where all executable binaries will be placed.

To compile for the first time do: 

```bash
git clone https://github.com/queone/utils
cd utils
go mod init utils
go mod tidy
./build.sh
```

For subsequent compilation just: 

```bash
cd utils
git pull
./build.sh
```

Note that you can compile individual utilities with `./build.sh rn,web`, etc.

To build in Windows you have to have a BASH shell such as [GitBASH](https://www.git-scm.com/download/win). To build from a regular Windows Command Prompt, you may have to tweak the `build.sh` script a bit, to have it run the right `go build ...` command. 
