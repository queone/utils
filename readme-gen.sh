#!/usr/bin/env bash
# readme-gen 0.2.0
set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

for d in cmd/*; do
  [ -d "$d" ] || continue
  main="$d/main.go"
  [ -f "$main" ] || continue

  util="$(basename "$d")"

  program_name="$(
    grep -E 'program_name[[:space:]]*=' "$main" \
    | head -n1 \
    | sed -E 's/.*=[[:space:]]*["“]?([^"”]+)["”]?.*/\1/'
  )"

  printf -- '- [`%s`](cmd/%s/README.md): %s.\n' \
    "$util" "$util" "<description>"
done

