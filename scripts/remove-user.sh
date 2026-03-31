#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <values.yaml> <username> [namespace]" >&2
  exit 1
fi

values_file="$1"
username="$2"
namespace="${3:-default}"

if [[ ! -f "$values_file" ]]; then
  echo "Values file not found: $values_file" >&2
  exit 1
fi

tmp_file="${values_file}.tmp"
awk -v user="$username" '
  BEGIN { skip=0 }
  /^- name: / {
    if ($3 == user) { skip=1; next }
  }
  skip {
    if ($0 ~ /^- name: /) { skip=0; print }
    next
  }
  { print }
' "$values_file" > "$tmp_file"

mv "$tmp_file" "$values_file"

echo "Removed user $username from $values_file"
echo "Deleting PVCs (if they exist)..."
kubectl delete pvc "opencode-${username}-data" "opencode-${username}-workspace" -n "$namespace" --ignore-not-found
