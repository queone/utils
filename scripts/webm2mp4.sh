#!/usr/bin/env bash
# webm2mp4.sh 1.3.1
# Converts WEBM format files to MP4 format

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

Utility="$(basename $0)" # This script filename
if [[ -z "${1:-}" ]]; then
    printf "Usage: %b%s%b FILENAME\n" "${Yel}" "${Utility}" "${Rst}"
    exit 1
fi

Filename="$1"
printf "==> Converting %b%s%b\n" "${Yel}" "${Filename}" "${Rst}"
ffmpeg -i $Filename -c:v libx264 -crf 23 -preset fast -c:a aac ${Filename}.mp4
exit 0
