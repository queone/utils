#!/usr/bin/env bash
# resize_image.sh 1.3.1
# Resizes given HEIC, JPEG, or JPG image file by 10% or compresses MP4 video file

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

F=$1       # Full filename
N=${F%.*}  # Base name
E=$(echo "${F##*.}" | awk '{print tolower($0)}') # Extension in lowercase
DateStr="$(date +%Y%m%d)a" # Date string for output naming

if [[ $F && "heic jpeg jpg" == *"$E"* ]]; then
  # Handle HEIC, JPEG, JPG images
  sips -s format jpeg -s formatOptions 10 "$F" -o "${N}_${DateStr}.jpg"
elif [[ $E == "mp4" ]]; then
  # Confirm the file is a valid MP4
  if ffprobe "$F" 2>&1 | grep -q "Input #0, mov,mp4"; then
    # Compress MP4 video file
    ffmpeg -i "$F" -vcodec libx264 -crf 35 "${N}_${DateStr}.mp4"
    # -i              = specifies the input file
    # -vcodec libx264 = sets the video codec to H.264, which is efficient for compression
    # -crf 35         = adjusts the quality/compression level. Lower values (e.g., 23) 
    #                   increase quality and file size, while higher values (e.g., 30)
    #                   reduce file size but lower quality. A value of 28 is a good 
    #                   starting point for balancing quality and size.
  else
    echo "Error: $F is not a valid MP4 file."
    exit 1
  fi
else
  # Usage message if file type is unsupported
  printf "usage: resize FILE.[heic|jpeg|jpg|mp4]\n"
fi

exit 0
