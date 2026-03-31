#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <username> [password] [workspaceSize]" >&2
  exit 1
fi

username="$1"
password="${2:-change-me}"
workspace_size="${3:-20Gi}"

cat <<EOF
- name: ${username}
  password: "${password}"
  workspaceSize: ${workspace_size}
  providers:
    anthropic:
      enabled: false
      apiKey: ""
    openai:
      enabled: false
      apiKey: ""
    google:
      enabled: false
      apiKey: ""
      projectId: ""
  mcp:
    remote: []
    laptopServers: []
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: "2"
      memory: 2Gi
  skills:
    npm: []
    config: []
EOF
