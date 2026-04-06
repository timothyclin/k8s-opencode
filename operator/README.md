# OpenCode Kubernetes Operator

A Kubernetes operator that manages OpenCode workspaces as custom resources. Each `OpenCodeWorkspace` CR creates an isolated per-user environment with persistent storage, AI provider configuration, and optional Tailscale connectivity.

## What It Does

The operator watches `OpenCodeWorkspace` resources and automatically creates:

- **Namespace** — Isolated namespace (`oc-<name>`) for each workspace
- **PVCs** — Two persistent volume claims for `/workspace` and `~/.opencode`
- **ConfigMap** — `opencode.json` configuration with AI provider credentials
- **Secrets** — API keys for Anthropic, OpenAI, and OpenRouter
- **NetworkPolicy** — Pod isolation within the namespace
- **Service** — ClusterIP service exposing OpenCode on port 4096
- **StatefulSet** — Single-replica OpenCode pod with optional kubedock sidecar
- **Tailscale Ingress** — Tailnet hostname for remote access (when configured)

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.11.3+)
- Go 1.24.6+
- Docker 17.03+
- kubectl
- [Tailscale Kubernetes Operator](https://tailscale.com/k8s) installed (for ingress)

### Build and Deploy

```bash
# Build the operator image
make docker-build IMG=ghcr.io/timothyclin/k8s-opencode/operator:latest

# Push to registry
make docker-push IMG=ghcr.io/timothyclin/k8s-opencode/operator:latest

# Install CRDs
make install

# Deploy to cluster
make deploy IMG=ghcr.io/timothyclin/k8s-opencode/operator:latest
```

### Create a Workspace

```yaml
apiVersion: opencode.opencode.io/v1alpha1
kind: OpenCodeWorkspace
metadata:
  name: alice
spec:
  email: "alice@example.com"
  providers:
    anthropic:
      enabled: true
      apiKeySecretRef:
        name: alice-api-keys
        namespace: oc-alice
        key: anthropic
  storage:
    workspace: "20Gi"
    data: "5Gi"
  tailscale:
    ingressTags:
      - "tag:opencode"
```

```bash
kubectl apply -f config/samples/
```

### Access the Workspace

With Tailscale ingress enabled:
```
https://oc-alice.<your-tailnet>.ts.net
```

Or via port-forward:
```bash
kubectl port-forward -n oc-alice svc/opencode 4096:4096
```

### Delete a Workspace

```bash
kubectl delete opencodeworkspace alice
```

The operator's finalizer cleans up all created resources.

## API Reference

### OpenCodeWorkspace Spec

| Field | Type | Description |
|-------|------|-------------|
| `email` | string | **Required.** Owner email address |
| `namespacePrefix` | string | Prefix for namespace (default: "oc") |
| `providers` | ProvidersSpec | AI provider configuration |
| `providers.anthropic.enabled` | bool | Enable Anthropic |
| `providers.anthropic.model` | string | Model (default: claude-sonnet-4-20250514) |
| `providers.anthropic.apiKeySecretRef` | SecretKeyRef | Secret containing API key |
| `providers.openai` | ProviderConfig | OpenAI configuration |
| `providers.openrouter` | ProviderConfig | OpenRouter configuration |
| `resources` | ResourceRequirements | CPU/memory requests/limits |
| `storage.workspace` | resource.Quantity | PVC size for /workspace |
| `storage.data` | resource.Quantity | PVC size for ~/.opencode |
| `storage.storageClassName` | string | StorageClass for PVCs |
| `tailscale.ingressTags` | []string | Tags for Tailscale ingress |
| `tailscale.egress.enabled` | bool | Enable egress to laptop |
| `tailscale.egress.laptopHostname` | string | Laptop MagicDNS hostname |
| `tailscale.egress.mcpPorts` | []MCPPort | Laptop MCP server ports |
| `kubedock.enabled` | bool | Enable kubedock sidecar |
| `kubedock.resources` | ResourceRequirements | Kubedock CPU/memory |

### OpenCodeWorkspace Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Pending, Creating, Reconciling, Running, Failed, Terminating |
| `namespace` | string | Created namespace name |
| `ingressHostname` | string | Tailnet hostname (if ingress enabled) |
| `message` | string | Human-readable status message |
| `conditions` | []Condition | Kubernetes conditions |

## Development

```bash
# Run tests
make test

# Run locally (uses current kubeconfig)
make run

# Generate manifests
make manifests generate

# Lint
make lint-fix
```

### Architecture

The controller reconciles in this order:
1. Namespace
2. NetworkPolicy
3. Secrets (API keys)
4. ConfigMap (opencode.json)
5. PVCs (workspace + data)
6. StatefulSet (OpenCode pod)
7. Service
8. Tailscale Ingress
9. EmailMap (for ACL generation)

## Project Distribution

### YAML Bundle

```sh
make build-installer IMG=<registry>/operator:tag
```

Generates `dist/operator-install.yaml` for easy installation:
```sh
kubectl apply -f https://github.com/timothyclin/k8s-opencode/releases/latest/download/operator-install.yaml
```

### Pre-built Image

```sh
kubectl apply -f https://github.com/timothyclin/k8s-opencode/releases/latest/download/operator-install.yaml
```

## License

Copyright 2026.

Licensed under the Apache License, Version.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0