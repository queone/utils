#!/usr/bin/env bash
# install_go.sh 1.4.0
# Install or upgrade Go: Homebrew on a Mac when present, the official archive otherwise

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

usage() {
    printf "Install or upgrade Go. Uses Homebrew on a Mac when present, the official archive otherwise.\n\n"
    printf "Usage: install_go.sh [-n] [-a] [VERSION]\n"
    printf "  -n, --dry-run    Print the plan and exit without installing anything\n"
    printf "  -a, --archive    Use the official archive even when Homebrew is present\n"
    printf "  -h, --help       Show this help\n"
    printf "  VERSION          Exact version to install, e.g. 1.24.0; always uses the archive\n"
}

# Print an error and exit 1
die() {
    printf "==> ${Red}%s${Rst}\n" "$*" >&2
    exit 1
}

# Require each named tool on PATH, stopping at the first missing one
require_tools() {
    local t
    for t in "$@"; do
        printf "==> Checking for ${Yel}%s${Rst} ... " "$t"
        if command -v "$t" &>/dev/null; then
            printf "${Gre}found${Rst}\n"
        else
            printf "${Red}missing${Rst}\n"
            die "Required tool '$t' is missing. Install it and rerun."
        fi
    done
}

# Require sha256sum or shasum
require_digest_tool() {
    command -v sha256sum &>/dev/null || command -v shasum &>/dev/null || die "Required tool 'sha256sum' or 'shasum' is missing."
}

# Print the SHA-256 digest of a file
sha256_of() {
    if command -v sha256sum &>/dev/null; then
        sha256sum "$1" | awk '{print $1}'
    else
        shasum -a 256 "$1" | awk '{print $1}'
    fi
}

# Run a command with sudo when sudo exists and we are not root
run_privileged() {
    if [[ "$(id -u)" -eq 0 ]] || ! command -v sudo &>/dev/null; then
        "$@"
    else
        printf "==> Running with sudo: ${Yel}%s${Rst}\n" "$*"
        sudo "$@"
    fi
}

# Print the dry-run plan, one fact per line
print_plan() {
    printf "%-8s %s\n" system "$OSName" cpu "$ARCH" path "$Path" "$1" "$2"
}

# ==== MAIN

DryRun=false
ForceArchive=false
Version=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--dry-run) DryRun=true ;;
        -a|--archive) ForceArchive=true ;;
        -h|-\?|--help) usage; exit 0 ;;
        -*) die "Unknown option '$1'. Run with -h for help." ;;
        *)
            [[ -z "$Version" ]] || die "Unexpected argument '$1'. Run with -h for help."
            Version="$1"
            ;;
    esac
    shift
done

case "$OSTYPE" in
    darwin*) OS=darwin;  OSName="macOS" ;;
    linux*)  OS=linux;   OSName="Linux" ;;
    msys*)   OS=windows; OSName="Windows/GitBASH" ;;
    *) die "Unknown/unsupported OSTYPE '$OSTYPE'" ;;
esac
Machine=$(uname -m)
case "$Machine" in
    x86_64) ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) die "Unsupported architecture '$Machine'" ;;
esac
printf "==> OS is ${Blu}%s${Rst} on ${Blu}%s${Rst}\n" "$OSName" "$ARCH"

Path=archive
if [[ "$OS" == darwin && -z "$Version" && "$ForceArchive" == false ]] && command -v brew &>/dev/null; then
    Path=homebrew
fi

# ---- Homebrew path
if [[ "$Path" == homebrew ]]; then
    Formula=go
    if [[ "$DryRun" == true ]]; then
        print_plan formula "$Formula"
        exit 0
    fi
    if brew list --versions "$Formula" &>/dev/null; then
        printf "==> ${Yel}%s${Rst} is installed through Homebrew. Upgrading ...\n" "$Formula"
        brew upgrade "$Formula"
    else
        printf "==> Installing ${Yel}%s${Rst} through Homebrew ...\n" "$Formula"
        brew install "$Formula"
    fi
    printf "==> Installed: ${Gre}%s${Rst}\n" "$("$(brew --prefix)/bin/go" version)"
    printf "==> Make sure these are set, perhaps in your ${Yel}~/.bashrc${Rst} file. Homebrew manages GOROOT.\n"
    printf "${Yel}export GOPATH=~/go  # Create this directory if necessary${Rst}\n"
    printf "${Yel}export PATH=\$PATH:\$GOPATH/bin${Rst}\n"
    exit 0
fi

# ---- Archive path
require_tools curl
require_digest_tool
if [[ "$OS" == windows ]]; then
    require_tools unzip
else
    require_tools tar
fi
if [[ -z "$Version" ]]; then
    require_tools jq
    Version=$(curl -fsS 'https://go.dev/dl/?mode=json' | jq -r '.[].version' | grep -vE 'beta|rc' | sed 's/^go//' | sort -V | tail -n 1 || true)
    [[ -n "$Version" ]] || die "Could not determine the latest stable Go version"
    printf "==> No version provided, installing latest stable ${Yel}%s${Rst}\n" "$Version"
else
    printf "==> Installing version ${Yel}%s${Rst}\n" "$Version"
fi
if [[ "$OS" == windows ]]; then
    Filename="go${Version}.windows-${ARCH}.zip"
else
    Filename="go${Version}.${OS}-${ARCH}.tar.gz"
fi
if [[ "$DryRun" == true ]]; then
    print_plan archive "$Filename"
    exit 0
fi

TargetDir=/usr/local
GoDir="$TargetDir/go"
TmpDir=$(mktemp -d)
trap 'rm -rf "$TmpDir"' EXIT
DownloadURL="https://go.dev/dl/${Filename}"

printf "==> Downloading ${Blu}%s${Rst}\n" "$DownloadURL"
curl -f -L -# -o "$TmpDir/$Filename" "$DownloadURL" || die "Error downloading $DownloadURL. Check the version."

# The digest comes from the per-file .sha256 on the download host, else from the release JSON
DigestRemote=$(curl -fsSL "https://dl.google.com/go/${Filename}.sha256" 2>/dev/null | grep -oE '^[0-9a-f]{64}' || true)
if [[ -z "$DigestRemote" ]] && command -v jq &>/dev/null; then
    DigestRemote=$(curl -fsS 'https://go.dev/dl/?mode=json&include=all' | jq -r --arg f "$Filename" '.[] | .files[] | select(.filename == $f) | .sha256' || true)
fi
[[ -n "$DigestRemote" ]] || die "Could not fetch the published digest for $Filename"
DigestLocal=$(sha256_of "$TmpDir/$Filename")
printf "%-24s = %s\n" "SHA DIGEST REMOTE" "$DigestRemote" "SHA DIGEST DOWNLOADED" "$DigestLocal"
[[ "$DigestLocal" == "$DigestRemote" ]] || die "SHA digests do NOT match. Aborting!"
printf "==> ${Gre}SHA digests match. Installing ...${Rst}\n"

run_privileged mkdir -p "$TargetDir"
if [[ -d "$GoDir" ]]; then
    OldVer=$("$GoDir/bin/go" version 2>/dev/null | awk '{print $3}' || true)
    BackupDir="${TargetDir}/${OldVer:-go}_$(date +%F)_$(mktemp -u XXXXXX)"
    printf "==> Directory ${Yel}%s${Rst} exists. Moving it to ${Yel}%s${Rst}\n" "$GoDir" "$BackupDir"
    run_privileged mv "$GoDir" "$BackupDir"
fi
if [[ "$Filename" == *.zip ]]; then
    run_privileged unzip -q -d "$TargetDir" "$TmpDir/$Filename"
else
    run_privileged tar -C "$TargetDir" -xzf "$TmpDir/$Filename"
fi
printf "==> Installed: ${Gre}%s${Rst}\n" "$("$GoDir/bin/go" version)"
printf "==> Now update below essential system variables, perhaps adding them to your ${Yel}~/.bashrc${Rst} file\n"
printf "${Yel}export GOROOT=/usr/local/go${Rst}\n"
printf "${Yel}export GOPATH=~/go  # Create this directory if necessary${Rst}\n"
printf "${Yel}export PATH=\$PATH:\$GOROOT/bin:\$GOPATH/bin${Rst}\n"

exit 0
