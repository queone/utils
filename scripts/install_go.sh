#!/usr/bin/env bash
# install_go.sh 1.3.1
# Install and setup Go

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

# Check binary exists
check_binary() {
    printf "==> Checking for ${Yel}$1${Rst} ... "
    if [[ -z "$(which $1 2>/dev/null)" ]]; then
        printf "${Red}missing${Rst}!\n"
        if [[ "$1" == "shasum" && "$OSTYPE" == "linux-gnu"* ]]; then
            run_with_sudo yum -y install perl-Digest-SHA
            if [[ -z "$(which $1 2>/dev/null)" ]]; then
                printf "==> Error trying to install ${Red}shasum${Rst}!\n"
                exit 1
            fi
        fi
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

# ==== MAIN

# Determine the GoLang version to be downloaded
if [[ -z "${1:-}" ]]; then
    GOVER=$(curl -ks https://go.dev/dl/?mode=json | jq -r '.[].version' | grep -vE 'beta|rc' | sed 's/^go//' | sort -V | tail -n 1)
    printf "==> No version provided, installing 2nd to last latest version ${Yel}$GOVER${Rst}\n"
else
    GOVER="$1"
    printf "==> Installing version ${Yel}$GOVER${Rst}\n"
fi

# Determine the GoLang file package to be downloaded for this OS
ARCH=$(uname -m)
case "$ARCH" in
    "x86_64") ARCH="amd64" ;;
    "aarch64") ARCH="arm64" ;;
    *) printf "==> Unsupported architecture: ${Red}$ARCH${Rst}\n" && exit 1 ;;
esac
case "$OSTYPE" in
    "msys"*)
        Filename="go${GOVER}.windows-${ARCH}.zip"
        printf "==> OS is ${Blu}Windows/GitBASH${Rst}\n"
        ;;
    "linux-gnu"*)
        Filename="go${GOVER}.linux-${ARCH}.tar.gz"
        printf "==> OS is ${Blu}Linux${Rst}\n"
        ;;
    "darwin"*)
        #Filename="${GOVER}.darwin-amd64.tar.gz"
        printf "==> OS is ${Blu}macOS${Rst}. Please install with ${Yel}brew install go${Rst}\n"
        exit 1
        ;;
    *)
        printf "==> Unknown/unsupported OSTYPE '$OSTYPE'\n"
        exit 1
        ;;
esac

# Confirm required binaries are available
for B in curl jq awk grep shasum ; do
    check_binary $B
done

TargetDir="/usr/local"
printf "==> Making sure ${Yel}$TargetDir${Rst} exists... \n"
run_with_sudo mkdir -p $TargetDir
GoDir="$TargetDir/go"

# Backup existing install if it exists
if [[ -d "$GoDir" ]]; then
    OldVer="$(${GoDir}/bin/go version | awk '{print $3}')"
    BackupDir="${TargetDir}/${OldVer}_$(date +%F)_temp-$(mktemp -u XXXXXX)"
    printf "==> Directory ${Yel}$GoDir${Rst} exists. Backing it up to a temp directory.\n"
    run_with_sudo mv "$GoDir" "$BackupDir"
fi

cd $TargetDir

Files=$(curl -ks "https://go.dev/dl/?mode=json")
printf "==> Downloading ${Blu}https://go.dev/dl/${Filename}${Rst}\n\n"
DownloadFile="/tmp/${Filename}"
curl -# -kLo $DownloadFile https://go.dev/dl/${Filename}
if [[ -e "$DownloadFile" ]]; then
    DigestLocal=$(shasum -a 256 $DownloadFile | awk '{print $1}')
else
    printf "==> Error downloading ${Red}${DownloadFile}${Rst}\n" && exit 1
fi
if [[ -n ${Files} ]]; then
    DigestRemote=$(echo "$Files" | jq -r '.[] | .files[] | "\(.filename) \(.sha256)"' | grep "$Filename" | awk '{print $2}')
else
    printf "==> Error getting JSON string with all digests\n" && exit 1
fi

printf "\n%-24s = %s\n" "SHA DIGEST REMOTE" "$DigestRemote"
printf "%-24s = %s\n" "SHA DIGEST DOWNLOADED" "$DigestLocal"
[[ "$DigestLocal" != "$DigestRemote" ]] && printf "\n==> ${Red}SHA digest do NOT match. Aborting!${Rst}\n" && exit 1
printf "\n==> ${Gre}SHA digests do match. Installing ...${Rst}\n"

if [[ "${DownloadFile}" == *.tar.gz ]]; then
    run_with_sudo tar xzf $DownloadFile
elif [[ "${DownloadFile}" == *.zip ]]; then
    run_with_sudo unzip -q $DownloadFile
else
    printf "==> Unknown ${Red}${DownloadFile}${Rst} extension\n" && exit 1
fi

printf "==> Now update below essential system variables, perhaps adding them to your ${Yel}~/.bashrc${Rst} file\n"
printf "${Yel}export GOROOT=/usr/local/go${Rst}\n"
printf "${Yel}export GOPATH=~/go  # Create this directory if necesary${Rst}\n"
printf "${Yel}export PATH=\$PATH:\$GOROOT/bin:\$GOPATH/bin${Rst}\n"

exit 0
