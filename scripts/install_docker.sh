#!/usr/bin/env bash
# install_docker.sh 1.0.0
# Install or upgrade Docker: the docker CLI and colima through Homebrew on a Mac, Docker's apt repository on Debian or Ubuntu, Docker's dnf repository on Fedora and RHEL-family systems

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

Product="Docker"
Formulae="docker colima"
Packages="docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin"

usage() {
    printf "Install or upgrade %s. Uses the docker CLI and colima through Homebrew on a Mac, Docker's apt repository on Debian or Ubuntu, Docker's dnf repository on Fedora and RHEL-family systems.\n\n" "$Product"
    printf "Usage: install_docker.sh [-n]\n"
    printf "  -n, --dry-run    Print the plan and exit without installing anything\n"
    printf "  -h, --help       Show this help\n"
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

# Print one field from /etc/os-release, empty when absent
os_release_field() {
    ( . /etc/os-release 2>/dev/null && printf '%s' "${!1:-}" )
}

# Enable the service when systemd is present, add the caller to the docker group, print the result
finish_linux() {
    if command -v systemctl &>/dev/null; then
        run_privileged systemctl enable --now docker
    else
        printf "==> ${Yel}systemctl${Rst} not found; start the docker service by hand\n"
    fi
    run_privileged usermod -aG docker "$(id -un)"
    printf "==> Installed: ${Gre}%s${Rst}\n" "$(docker --version)"
    printf "==> Log out and back in so the ${Yel}docker${Rst} group takes effect\n"
}

# ==== MAIN

DryRun=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--dry-run) DryRun=true ;;
        -h|-\?|--help) usage; exit 0 ;;
        -*) die "Unknown option '$1'. Run with -h for help." ;;
        *) die "Unexpected argument '$1'. Run with -h for help." ;;
    esac
    shift
done

case "$OSTYPE" in
    darwin*) OS=darwin;  OSName="macOS" ;;
    linux*)  OS=linux;   OSName="Linux" ;;
    *) die "Unknown/unsupported OSTYPE '$OSTYPE'" ;;
esac
Machine=$(uname -m)
case "$Machine" in
    x86_64) ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) die "Unsupported architecture '$Machine'" ;;
esac
printf "==> OS is ${Blu}%s${Rst} on ${Blu}%s${Rst}\n" "$OSName" "$ARCH"

Path=""
if [[ "$OS" == darwin ]] && command -v brew &>/dev/null; then
    Path=homebrew
elif [[ "$OS" == linux ]] && command -v apt-get &>/dev/null; then
    Path=apt
elif [[ "$OS" == linux ]] && command -v dnf &>/dev/null; then
    Path=dnf
fi
[[ -n "$Path" ]] || die "No supported install path: this needs Homebrew on macOS, or apt-get or dnf on Linux"

# ---- Homebrew path
if [[ "$Path" == homebrew ]]; then
    if [[ "$DryRun" == true ]]; then
        print_plan formula "$Formulae"
        exit 0
    fi
    for Formula in $Formulae; do
        if brew list --versions "$Formula" &>/dev/null; then
            printf "==> ${Yel}%s${Rst} is installed through Homebrew. Upgrading ...\n" "$Formula"
            brew upgrade "$Formula"
        else
            printf "==> Installing ${Yel}%s${Rst} through Homebrew ...\n" "$Formula"
            brew install "$Formula"
        fi
    done
    printf "==> Installed: ${Gre}%s${Rst}\n" "$("$(brew --prefix)/bin/docker" --version)"
    printf "==> Run ${Yel}colima start${Rst} before first use; the docker CLI talks to the colima VM\n"
    exit 0
fi

# ---- apt path
if [[ "$Path" == apt ]]; then
    require_tools curl gpg apt-get dpkg
    DistroId=$(os_release_field ID)
    Codename=$(os_release_field VERSION_CODENAME)
    case "$DistroId" in
        ubuntu|debian) ;;
        *) die "Docker's apt repository covers Ubuntu and Debian; this system reports ID '$DistroId'" ;;
    esac
    [[ -n "$Codename" ]] || die "Could not read VERSION_CODENAME from /etc/os-release"
    if [[ "$DryRun" == true ]]; then
        print_plan package "$Packages"
        exit 0
    fi
    KeyFile=/etc/apt/keyrings/docker.gpg
    ListFile=/etc/apt/sources.list.d/docker.list
    if [[ ! -f "$KeyFile" || ! -f "$ListFile" ]]; then
        printf "==> Adding Docker's apt repository for ${Yel}%s %s${Rst}\n" "$DistroId" "$Codename"
        TmpDir=$(mktemp -d)
        trap 'rm -rf "$TmpDir"' EXIT
        curl -fsSL "https://download.docker.com/linux/${DistroId}/gpg" | gpg --dearmor > "$TmpDir/keyring.gpg" || die "Error fetching Docker's signing key"
        printf 'deb [arch=%s signed-by=%s] https://download.docker.com/linux/%s %s stable\n' "$(dpkg --print-architecture)" "$KeyFile" "$DistroId" "$Codename" > "$TmpDir/docker.list"
        run_privileged install -d -m 755 /etc/apt/keyrings
        run_privileged install -m 644 "$TmpDir/keyring.gpg" "$KeyFile"
        run_privileged install -m 644 "$TmpDir/docker.list" "$ListFile"
    fi
    run_privileged apt-get update
    # shellcheck disable=SC2086  # Packages is a deliberate word list
    run_privileged apt-get install -y $Packages
    finish_linux
    exit 0
fi

# ---- dnf path
if [[ "$Path" == dnf ]]; then
    require_tools dnf
    DistroId=$(os_release_field ID)
    case "$DistroId" in
        fedora) RepoOS=fedora ;;
        rhel)   RepoOS=rhel ;;
        *)      RepoOS=centos ;;
    esac
    if [[ "$DryRun" == true ]]; then
        print_plan package "$Packages"
        exit 0
    fi
    RepoURL="https://download.docker.com/linux/${RepoOS}/docker-ce.repo"
    RepoFile=/etc/yum.repos.d/docker-ce.repo
    if [[ ! -f "$RepoFile" ]]; then
        printf "==> Adding Docker's dnf repository for ${Yel}%s${Rst}\n" "$RepoOS"
        run_privileged dnf install -y dnf-plugins-core
        # dnf 4 and dnf 5 spell the add-repo command differently
        run_privileged dnf config-manager --add-repo "$RepoURL" 2>/dev/null \
            || run_privileged dnf config-manager addrepo --from-repofile="$RepoURL"
    fi
    # shellcheck disable=SC2086  # Packages is a deliberate word list
    run_privileged dnf install -y $Packages
    finish_linux
    exit 0
fi

die "Unhandled install path '$Path'"
