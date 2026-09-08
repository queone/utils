#!/usr/bin/env bash
# install_vault.sh 1.3.0
# Install or upgrade HashiCorp Vault: Homebrew on a Mac, HashiCorp's apt repository on Debian or Ubuntu, the official archive otherwise

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

Product="HashiCorp Vault"
Tool="vault"
Formula="hashicorp/tap/vault"

usage() {
    printf "Install or upgrade %s. Uses Homebrew on a Mac, HashiCorp's apt repository on Debian or Ubuntu, the official archive otherwise.\n\n" "$Product"
    printf "Usage: install_vault.sh [-n] [-a] [VERSION]\n"
    printf "  -n, --dry-run    Print the plan and exit without installing anything\n"
    printf "  -a, --archive    Use the official archive even when Homebrew or apt is present\n"
    printf "  -h, --help       Show this help\n"
    printf "  VERSION          Exact version to install, e.g. 1.9.8; always uses the archive\n"
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
elif [[ "$OS" == linux && -z "$Version" && "$ForceArchive" == false ]] && command -v apt-get &>/dev/null; then
    Path=apt
fi

# ---- Homebrew path
if [[ "$Path" == homebrew ]]; then
    if [[ "$DryRun" == true ]]; then
        print_plan formula "$Formula"
        exit 0
    fi
    brew tap | grep -qx hashicorp/tap || brew tap hashicorp/tap
    if brew list --versions "$Tool" &>/dev/null; then
        printf "==> ${Yel}%s${Rst} is installed through Homebrew. Upgrading ...\n" "$Tool"
        brew upgrade "$Formula"
    else
        printf "==> Installing ${Yel}%s${Rst} through Homebrew ...\n" "$Formula"
        brew install "$Formula"
    fi
    printf "==> Installed: ${Gre}%s${Rst}\n" "$("$(brew --prefix)/bin/$Tool" version | head -n 1)"
    exit 0
fi

# ---- apt path
if [[ "$Path" == apt ]]; then
    require_tools curl gpg apt-get
    if [[ "$DryRun" == true ]]; then
        print_plan package "$Tool"
        exit 0
    fi
    KeyFile=/usr/share/keyrings/hashicorp-archive-keyring.gpg
    ListFile=/etc/apt/sources.list.d/hashicorp.list
    if [[ ! -f "$KeyFile" || ! -f "$ListFile" ]]; then
        Codename=$(. /etc/os-release 2>/dev/null && printf '%s' "${VERSION_CODENAME:-}")
        [[ -n "$Codename" ]] || die "Could not read VERSION_CODENAME from /etc/os-release"
        printf "==> Adding HashiCorp's apt repository for ${Yel}%s${Rst}\n" "$Codename"
        TmpDir=$(mktemp -d)
        trap 'rm -rf "$TmpDir"' EXIT
        curl -fsSL https://apt.releases.hashicorp.com/gpg | gpg --dearmor > "$TmpDir/keyring.gpg" || die "Error fetching HashiCorp's signing key"
        printf 'deb [signed-by=%s] https://apt.releases.hashicorp.com %s main\n' "$KeyFile" "$Codename" > "$TmpDir/hashicorp.list"
        run_privileged install -m 644 "$TmpDir/keyring.gpg" "$KeyFile"
        run_privileged install -m 644 "$TmpDir/hashicorp.list" "$ListFile"
    fi
    run_privileged apt-get update
    run_privileged apt-get install -y "$Tool"
    printf "==> Installed: ${Gre}%s${Rst}\n" "$("$Tool" version | head -n 1)"
    exit 0
fi

# ---- Archive path
require_tools curl unzip
require_digest_tool
if [[ -z "$Version" ]]; then
    Version=$(curl -fsS "https://releases.hashicorp.com/${Tool}/" | grep -oE "${Tool}/[0-9]+\.[0-9]+\.[0-9]+/" | cut -d/ -f2 | sort -V | tail -n 1 || true)
    [[ -n "$Version" ]] || die "Could not determine the latest stable $Product version"
    printf "==> No version provided, installing latest stable ${Yel}%s${Rst}\n" "$Version"
else
    printf "==> Installing version ${Yel}%s${Rst}\n" "$Version"
fi
ArchiveArch="$ARCH"
if [[ "$OS" == windows ]]; then
    ArchiveArch=amd64  # HashiCorp publishes no Windows ARM build
fi
Filename="${Tool}_${Version}_${OS}_${ArchiveArch}.zip"
if [[ "$DryRun" == true ]]; then
    print_plan archive "$Filename"
    exit 0
fi

BaseURL="https://releases.hashicorp.com/${Tool}/${Version}"
TmpDir=$(mktemp -d)
trap 'rm -rf "$TmpDir"' EXIT

printf "==> Downloading ${Blu}%s${Rst}\n" "$BaseURL/$Filename"
curl -f -L -# -o "$TmpDir/$Filename" "$BaseURL/$Filename" || die "Error downloading $BaseURL/$Filename. Check the version."
curl -fsS -o "$TmpDir/SHA256SUMS" "$BaseURL/${Tool}_${Version}_SHA256SUMS" || die "Error downloading the digest file for $Product $Version"
DigestRemote=$(grep " ${Filename}\$" "$TmpDir/SHA256SUMS" | awk '{print $1}' || true)
[[ -n "$DigestRemote" ]] || die "No published digest for $Filename"
DigestLocal=$(sha256_of "$TmpDir/$Filename")
printf "%-24s = %s\n" "SHA DIGEST REMOTE" "$DigestRemote" "SHA DIGEST DOWNLOADED" "$DigestLocal"
[[ "$DigestLocal" == "$DigestRemote" ]] || die "SHA digests do NOT match. Aborting!"
printf "==> ${Gre}SHA digests match. Installing ...${Rst}\n"

unzip -q -o -d "$TmpDir" "$TmpDir/$Filename" || die "Error extracting $Filename"
Binary="$Tool"
if [[ "$OS" == windows ]]; then
    Binary="${Tool}.exe"
fi
run_privileged mkdir -p /usr/local/bin
run_privileged install -m 755 "$TmpDir/$Binary" "/usr/local/bin/$Binary"
printf "==> Installed: ${Gre}%s${Rst}\n" "$("/usr/local/bin/$Binary" version | head -n 1)"
printf "==> Make sure ${Yel}/usr/local/bin${Rst} is in your PATH\n"

exit 0
