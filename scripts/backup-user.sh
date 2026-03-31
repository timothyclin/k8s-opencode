#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 4 ]]; then
  echo "Usage: $0 <release> <username> <namespace> <output-tar.gz>" >&2
  exit 1
fi

release="$1"
username="$2"
namespace="$3"
output="$4"

pod=$(kubectl get pod -n "$namespace" -l "app.kubernetes.io/instance=${release},app.kubernetes.io/name=k8s-omo,app.kubernetes.io/user=${username}" -o jsonpath='{.items[0].metadata.name}')

if [[ -z "$pod" ]]; then
  echo "No pod found for user ${username} in ${namespace}" >&2
  exit 1
fi

kubectl exec -n "$namespace" "$pod" -- tar -czf /tmp/workspace.tar.gz -C /workspace .
kubectl cp "$namespace/$pod:/tmp/workspace.tar.gz" "$output"
kubectl exec -n "$namespace" "$pod" -- rm -f /tmp/workspace.tar.gz
echo "Backup saved to $output"
