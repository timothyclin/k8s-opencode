#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 4 ]]; then
  echo "Usage: $0 <release> <username> <namespace> <input-tar.gz>" >&2
  exit 1
fi

release="$1"
username="$2"
namespace="$3"
input="$4"

if [[ ! -f "$input" ]]; then
  echo "Backup file not found: $input" >&2
  exit 1
fi

pod=$(kubectl get pod -n "$namespace" -l "app.kubernetes.io/instance=${release},app.kubernetes.io/name=ok8s,app.kubernetes.io/user=${username}" -o jsonpath='{.items[0].metadata.name}')

if [[ -z "$pod" ]]; then
  echo "No pod found for user ${username} in ${namespace}" >&2
  exit 1
fi

kubectl cp "$input" "$namespace/$pod:/tmp/workspace.tar.gz"
kubectl exec -n "$namespace" "$pod" -- tar -xzf /tmp/workspace.tar.gz -C /workspace
kubectl exec -n "$namespace" "$pod" -- rm -f /tmp/workspace.tar.gz
echo "Restore completed for $username"
