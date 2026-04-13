#!/usr/bin/env bash
set -euo pipefail

# install-utils.sh — install all queone/utils CLI utilities via go install
#
# One-liner:
#   curl -sL https://raw.githubusercontent.com/queone/utils/main/install-utils.sh | bash

MODULE="github.com/queone/utils"

if ! command -v go &>/dev/null; then
    echo "Error: Go is not installed. See https://go.dev/dl/"
    exit 1
fi

GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

echo "Installing all utilities from ${MODULE}..."
go install "${MODULE}/cmd/...@latest"

# List what was installed
echo ""
echo "Installed to ${GOBIN}:"
for bin in "${GOBIN}"/*; do
    name=$(basename "$bin")
    # Check if this binary belongs to our module by matching known utilities
    case "$name" in
        bak|brew-update|cash5|certgen|certls|claude-env|days|decolor|dl|dos2unix|\
        fr|git-cloneall|git-pullall|git-remotev|git-statall|jy|pgen|pman|\
        rn|rncap|rnlower|sms|tree|web)
            echo "  $name"
            ;;
    esac
done

# Check PATH
if [[ ":$PATH:" != *":${GOBIN}:"* ]]; then
    echo ""
    echo "Warning: ${GOBIN} is not in your PATH."
    echo "Add this to your shell profile:"
    echo "  export PATH=\"\$PATH:${GOBIN}\""
fi
