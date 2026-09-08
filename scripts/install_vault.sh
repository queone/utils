#!/usr/bin/env bash
# install_vault.sh 1.2.1
# Install HashiCorp Vault binary

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

# Check binary exists
check_binary() {
    printf "==> Checking for ${Yel}$1${Rst} ... "
    if ! command -v "$1" &>/dev/null; then
        printf "${Red}missing${Rst}!\n"
    else
        printf "${Gre}found${Rst}\n"
    fi
}

# Run commands with sudo if available
run_with_sudo() {
    if command -v sudo &>/dev/null; then
        printf "==> Running command with sudo: ${Yel}%s${Rst}\n" "$*"
        sudo "$@"
    else
        printf "==> Running command without sudo: ${Yel}%s${Rst}\n" "$*"
        "$@"
    fi
}

# ==== MAIN =============================================================

# Confirm required binaries are available
for B in curl grep unzip ; do
    check_binary $B
done

# Default to latest version if none is provided
if [ -z "${1:-}" ]; then
    printf "==> No version provided\n"
    LATEST_VERSION=$(curl -ks https://releases.hashicorp.com/vault/ | grep -oP 'vault_\K[0-9]+\.[0-9]+\.[0-9]+' | sort -V | tail -n 1)
    V_VERSION=$LATEST_VERSION
    printf "==> Installing latest stable ${Yel}$V_VERSION${Rst}\n"
else
    V_VERSION=$1
    printf "==> Installing version ${Yel}$V_VERSION${Rst}\n"
fi


# Construct the filename and download URLs
Filename="vault_${V_VERSION}_linux_amd64.zip"
DownloadURL="https://releases.hashicorp.com/vault/${V_VERSION}/${Filename}"
ChecksumURL="https://releases.hashicorp.com/vault/${V_VERSION}/vault_${V_VERSION}_SHA256SUMS"

# Download and install
printf "==> Downloading Vault from ${Yel}$DownloadURL${Rst} ...\n"
if ! curl -ksO "$DownloadURL"; then
    printf "==> ${Red}Error downloading Vault. Please check the version or URL.${Rst}\n"
    exit 1
fi

printf "==> Downloading checksum from ${Yel}$ChecksumURL${Rst} ...\n"
if ! curl -ksO "$ChecksumURL"; then
    printf "==> ${Red}Error downloading checksum. Please check the version or URL.${Rst}\n"
    exit 1
fi

# Verify the checksum
printf "==> Verifying checksum ...\n"
if ! grep "$Filename" vault_${V_VERSION}_SHA256SUMS | sha256sum -c -; then
    printf "==> ${Red}Checksum verification failed! Exiting.${Rst}\n"
    exit 1
fi

# Unzip the binary
if ! unzip -o "$Filename"; then
    printf "==> ${Red}Error extracting the Vault archive ${Filename}.${Rst}\n"
    exit 1
fi

# Install the binary
run_with_sudo mkdir -p /usr/local/bin
run_with_sudo mv vault /usr/local/bin/
rm $Filename vault_${V_VERSION}_SHA256SUMS
printf "==> Vault version installed:\n"
printf "${Yel}"
vault version
printf "${Rst}"
printf "==> Make sure to put directory ${Yel}/usr/local/bin${Rst} in your PATH\n"

exit 0

