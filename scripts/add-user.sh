#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <username> <email> [password] [workspaceSize]" >&2
  exit 1
fi

username="$1"
email="$2"
password="${3:-change-me}"
workspace_size="${4:-20Gi}"

cat <<USEREOF
- name: ${username}
  email: "${email}"
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
USEREOF
