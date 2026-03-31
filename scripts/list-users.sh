#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <release> <namespace>" >&2
  exit 1
fi

release="$1"
namespace="$2"

kubectl get pods -n "$namespace" -l "app.kubernetes.io/instance=${release},app.kubernetes.io/name=ok8s" -o wide
