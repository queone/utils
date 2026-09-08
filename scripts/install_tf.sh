#!/usr/bin/env bash
# install_tf.sh 1.3.1
# Install Terraform binary

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

# Check binary exists
check_binary() {
    printf "==> Checking for ${Yel}$1${Rst} ... "
    if ! command -v "$1" &>/dev/null; then
        printf "${Red}missing${Rst}!\n"
        # if [[ "$1" == "shasum" && "$OSTYPE" == "linux-gnu"* ]]; then
        #     run_with_sudo yum -y install perl-Digest-SHA
        #     if [[ -z "$(which $1 2>/dev/null)" ]]; then
        #         printf "==> Error trying to install ${Red}shasum${Rst}!\n"
        #         exit 1
        #     fi
        # fi
    else
        printf "${Gre}found${Rst}\n"
    fi
}

# Run commands with sudo
run_with_sudo() {
    if [[ "$OSTYPE" == "msys"* ]]; then
        # GitBASH doesn't use sudo
        "$@"
    else
        printf "==> Running command with sudo: ${Yel}%s${Rst}\n" "$*"
        sudo "$@"
    fi
}

# ==== MAIN =============================================================

# Confirm required binaries are available
for B in curl grep head unzip ; do
    check_binary $B
done

# Default to latest version if none is provided
if [ -z "${1:-}" ]; then
    LATEST_VERSION=$(curl -ks https://releases.hashicorp.com/terraform/ | grep -oP 'terraform/\K[0-9]+\.[0-9]+\.[0-9]+' | head -n 1)
    T_VERSION=$LATEST_VERSION
    printf "==> No version provided, installing latest stable ${Yel}$T_VERSION${Rst}\n"
else
    T_VERSION=$1
    printf "==> Installing version ${Yel}$T_VERSION${Rst}\n"
fi

# Determine the Terraform binary filename based on the OS
case "$OSTYPE" in
    "msys"*)
        FileExt="windows_amd64.zip"
        printf "==> OS is ${Blu}Windows/GitBASH${Rst}\n"
        ;;
    "linux-gnu"*)
        FileExt="linux_amd64.zip"
        printf "==> OS is ${Blu}Linux${Rst}\n"
        ;;
    "darwin"*)
        printf "==> OS is ${Blu}macOS${Rst}. Please install with ${Yel}brew install terraform${Rst}\n"
        exit 1
        ;;
    *)
        printf "==> Unknown/unsupported OSTYPE '$OSTYPE'\n"
        exit 1
        ;;
esac

# Construct the filename and download URL
Filename="terraform_${T_VERSION}_${FileExt}"
DownloadURL="https://releases.hashicorp.com/terraform/${T_VERSION}/${Filename}"

# Download and install
printf "==> Downloading Terraform from ${Yel}$DownloadURL${Rst} ...\n"
if ! curl -ksO "$DownloadURL"; then
    printf "==> ${Red}Error downloading Terraform. Please check the version or URL.${Rst}\n"
    exit 1
fi

if ! unzip -o "$Filename"; then
    printf "==> ${Red}Error extracting the Terraform archive ${Filename}.${Rst}\n"
    exit 1
fi

# Unzip should have produced a binary and LICENSE file
if [[ -f "terraform.exe" ]]; then
    TerraformBinary="terraform.exe"
else
    TerraformBinary="terraform"
fi

# Install the binary
run_with_sudo mkdir -p /usr/local/bin
run_with_sudo mv $TerraformBinary /usr/local/bin/
rm $Filename LICENSE
printf "==> Terraform version installed:\n"
printf "${Yel}"
$TerraformBinary version
printf "${Rst}"
printf "==> Make sure to put ${Yel}/usr/local/bin${Rst} in your PATH\n"

exit 0
