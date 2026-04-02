# k8s-opencode

OpenCode on Kubernetes, with Tailscale connectivity for remote access and laptop
MCP servers.

## What This Is

Deploy [OpenCode](https://opencode.ai) (AI coding agent) on your Kubernetes
cluster with two deployment options:

- **Single-user**: Helm chart → simple personal deployment
- **Multi-user**: Kubernetes operator → dynamic user provisioning via CRD

Tailscale provides secure access from anywhere — no public ports, no ingress
controllers, no TLS cert management.

**Two-way connectivity:**

- **Your laptop → Cluster**: Access the OpenCode web UI via Tailscale MagicDNS
- **Cluster → Your laptop**: OpenCode agents call MCP servers running locally on
  your machine

## Deployment Options

| Mode | Tool | Use Case |
|------|------|----------|
| Single-user | Helm chart | Personal use, simple setup |
| Multi-user | Operator + CRD | Teams, dynamic provisioning, enterprise |

---

## Single-User Mode (Helm Chart)

One OpenCode instance for personal use. Uses a standard Kubernetes `Deployment`.

### Prerequisites

- Kubernetes cluster (ARM64 or amd64)
- Helm 3.x
- Tailscale account (free tier works)
- At least one LLM provider API key

### Install

**No version required** — omit `--version` to use the latest published version:

```bash
# Latest version (recommended - no --version flag needed)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --set providers.anthropic.enabled=true \
  --set providers.anthropic.apiKey=sk-ant-your-key \
  --set serverPassword=your-secure-password

# Specific version (only if you need an exact version)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --version 0.1.5 \
  --set providers.anthropic.enabled=true \
  --set providers.anthropic.apiKey=sk-ant-your-key \
  --set serverPassword=your-secure-password
```

Or with a values file:

```bash
# Latest version (recommended)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f my-values.yaml

# Specific version
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --version 0.1.5 \
  -f my-values.yaml
```

> **Namespace is required** — the chart will fail if installed into `default`.

### Verify

```bash
helm test ok8s -n opencode
```

### Access OpenCode

After enabling Tailscale ingress (see [Tailscale Setup](#tailscale-setup)):

```
https://opencode.<your-tailnet>.ts.net
Password: (your serverPassword value)
```

### Configuration

See [chart/values.yaml](chart/values.yaml) for all options. Key sections:

| Section | Purpose |
|---------|---------|
| `providers.*` | LLM API keys (anthropic, openai, google) |
| `serverPassword` | HTTP auth for the OpenCode server |
| `mcp.remote[]` | Remote MCP servers (URLs) |
| `mcp.laptopServers[]` | Laptop MCP servers (via Tailscale egress) |
| `ingress.enabled` | Expose OpenCode UI to tailnet |
| `kubedock.*` | Docker API → K8s Pod translation |
| `persistence.*` | Storage for config and workspace |

### Enabling Oh-My-OpenCode, Skills, MCPs, and Plugins

By default, the chart installs OpenCode with **no plugins, skills, or MCP servers**. To enable them, add the following to your values file:

```yaml
# Enable Oh-My-OpenCode (task orchestration, specialized agents)
omo:
  enabled: true
  sisyphus:
    maxConcurrentTasks: 2
    taskTimeout: 300
  agents:
    oracle:
      model: "github-copilot/gpt-5.2"
      promptAppend: "Provide concise tradeoffs and a clear recommendation"
    librarian:
      model: "github-copilot/gpt-5-mini"
  categories:
    quick:
      model: "github-copilot/gpt-4.1"
    visualEngineering:
      model: "github-copilot/claude-sonnet-4.6"

# Install npm skills (optional)
skills:
  npm: []
  # Example: ["@my-org/skills@latest"]
  config: []

# Configure MCP servers (optional)
mcp:
  remote: []
  # Example:
  # - name: context7
  #   url: https://mcp.context7.com/mcp
  #   enabled: true
  laptopServers: []
  # Example:
  # - name: playwright
  #   tailscaleIP: "100.x.x.x"
  #   port: 3000

# Additional plugins (optional)
plugins:
  enabled: true
  npm: []
  # Example: ["@tarquinen/opencode-dcp@latest"]
```

---

## Multi-User Mode (Operator)

Dynamic user provisioning via Kubernetes CRD. Each user gets an isolated
workspace with dedicated storage, config, and network policy.

### Prerequisites

- Kubernetes cluster (ARM64 or amd64)
- kubectl configured
- Tailscale Kubernetes Operator installed

### Install the Operator

```bash
kubectl apply -f https://raw.githubusercontent.com/timothyclin/k8s-opencode/main/operator/dist/install.yaml
```

Or build from source:

```bash
cd operator
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
        namespace: opencode-system
        key: anthropic
    openai:
      enabled: false
    openrouter:
      enabled: false
  storage:
    workspace: "20Gi"
    data: "5Gi"
```

```bash
kubectl apply -f alice-workspace.yaml
```

### What the Operator Creates

For each `OpenCodeWorkspace` CR, the operator reconciles:

- **Namespace** — `opencode-<name>` (isolated per user)
- **PVCs** — `workspace-pvc` and `data-pvc` for persistent storage
- **ConfigMap** — `opencode-config` with `opencode.json` configuration
- **NetworkPolicy** — isolates user workloads
- **Service** — ClusterIP on port 4096
- **StatefulSet** — single-replica OpenCode pod

### API Key Management

API keys can be provided via a Secret reference:

```yaml
spec:
  providers:
    anthropic:
      enabled: true
      apiKeySecretRef:
        name: my-secret       # Secret name
        namespace: default    # Secret namespace
        key: anthropic-key    # Key within the Secret
```

Or use OAuth (no API key needed):

```yaml
spec:
  providers:
    anthropic:
      enabled: true
      # No apiKeySecretRef — uses OAuth
```

### Access User Workspace

Each workspace gets its own namespace. Access via port-forward or Tailscale
ingress:

```bash
kubectl port-forward -n opencode-alice svc/opencode 4096:4096
```

### Delete a Workspace

```bash
kubectl delete opencodeworkspace alice
```

The operator's finalizer automatically cleans up the user namespace and all
resources.

---

## Tailscale Setup

### Install the Tailscale Kubernetes Operator

```bash
helm repo add tailscale https://pkgs.tailscale.com/helmcharts
helm install tailscale-operator tailscale/tailscale-operator \
  --namespace tailscale \
  --create-namespace \
  --set-string oauth.clientId=tskey-client-xxx \
  --set-string oauth.clientSecret=tskey-client-xxx
```

Get your OAuth client from
[Tailscale Admin Console](https://login.tailscale.com/admin/settings/oauth).
Required scopes: `devices`, `services`, `keys`.

### Enable Ingress (Single-User Helm)

```bash
helm upgrade ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode \
  -f my-values.yaml \
  --set ingress.enabled=true
```

### Expose Laptop MCP Servers

For each MCP server running on your laptop:

```yaml
# my-values.yaml (Helm) or in CRD spec (Operator)
mcp:
  laptopServers:
    - name: playwright
      tailscaleIP: "100.x.x.x" # Your laptop's Tailscale IP (run: tailscale ip -4)
      port: 3000
```

The chart/operator creates a Tailscale egress proxy so the cluster can reach
your laptop's MCP server.

---

## Kubedock: Test Containers as Kubernetes Pods

Kubedock translates the Docker API into Kubernetes Pod creation. This prevents
OOM kills from running Docker-in-Docker (DinD) sidecars by spawning test
containers as native K8s Pods instead.

### Enable Kubedock (Single-User)

```yaml
kubedock:
  enabled: true
```

### Testcontainers Configuration

These environment variables are auto-injected when kubedock is enabled:

```yaml
DOCKER_HOST: "tcp://<kubedock-service>:2475"
TESTCONTAINERS_RYUK_DISABLED: "true"
TESTCONTAINERS_CHECKS_DISABLE: "true"
```

---

## Architecture

### Single-user mode (Helm)

```
Your Tailnet
├── Laptop (Tailscale node)
│   └── Local MCP servers (Playwright, browser tools, etc.)
│
└── Kubernetes Cluster
    ├── Tailscale Operator
    │   ├── Ingress proxy  ← exposes OpenCode UI to tailnet
    │   └── Egress proxies ← routes cluster traffic to laptop MCPs
    │
    └── OpenCode Pod (Deployment)
        ├── opencode serve :4096
        ├── kubedock (optional)
        └── MCP config
```

### Multi-user mode (Operator)

```
Your Tailnet
├── Laptop (Tailscale node)
│   └── Local MCP servers
│
└── Kubernetes Cluster
    ├── Tailscale Operator
    │   └── Per-user ingress proxies
    │
    ├── OpenCode Operator (operator-system namespace)
    │   └── Watches OpenCodeWorkspace CRs
    │
    ├── opencode-alice namespace
    │   ├── StatefulSet (1 replica)
    │   ├── PVCs (workspace + data)
    │   ├── ConfigMap (opencode.json)
    │   ├── NetworkPolicy
    │   └── Service
    │
    └── opencode-bob namespace
        └── (same structure)
```

---

## Secret Management (Single-User Helm)

Four backends supported:

```yaml
# Plain Kubernetes secrets (default — fine for personal clusters)
secrets:
  backend: "plain"

# Bitnami sealed-secrets (for GitOps)
secrets:
  backend: "sealed"

# Mozilla SOPS (for encrypted values in Git)
secrets:
  backend: "sops"

# external-secrets-operator (for Vault, AWS SM, GCP SM, etc.)
secrets:
  backend: "external"
  externalSecretStore: "my-secret-store"
```

---

## Development

### Build Operator Locally

```bash
cd operator
make manifests generate  # Regenerate CRDs and code
make test                # Run unit tests
make run                 # Run locally against current kubeconfig
```

### Build and Push Operator Image

```bash
cd operator
make docker-build docker-push IMG=ghcr.io/timothyclin/k8s-opencode/operator:dev
```

### Test in Kind

```bash
kind create cluster --name opencode-test
make deploy IMG=ghcr.io/timothyclin/k8s-opencode/operator:dev
kubectl apply -f config/samples/
```

---

# AI Instructions

## Context

This repository provides two ways to deploy OpenCode on Kubernetes:

1. **Helm chart** (`chart/`) — for single-user deployments
2. **Kubernetes operator** (`operator/`) — for multi-user deployments with CRD

## Key Architecture Facts

- OpenCode runs as `opencode serve` — a headless HTTP server on port 4096
- Single replica per user (stateful — stores sessions and workspace on disk)
- Tailscale handles all external connectivity (no public ingress)
- Default Docker image: `ghcr.io/timothyclin/k8s-opencode/opencode-workspace`
- Operator image: `ghcr.io/timothyclin/k8s-opencode/operator`

## File Structure

```
chart/                           # Helm chart for single-user mode
├── Chart.yaml
├── values.yaml
└── templates/

operator/                        # Kubernetes operator for multi-user mode
├── api/v1alpha1/               # CRD types (OpenCodeWorkspace)
├── internal/controller/        # Reconciliation logic
├── config/crd/                 # Generated CRDs
├── config/samples/             # Example CRs
├── Dockerfile
└── Makefile
```

## CRD Spec (OpenCodeWorkspace)

```yaml
spec:
  email: string                  # Required, must match ^[^@]+@[^@]+$
  providers:
    anthropic/openai/openrouter:
      enabled: boolean
      apiKeySecretRef:           # Optional — omit for OAuth
        name: string
        namespace: string
        key: string
  storage:
    workspace: string            # e.g., "20Gi"
    data: string                 # e.g., "5Gi"
    storageClassName: string     # Optional
```

## How to Help

**For Helm chart changes:**
1. Edit `chart/values.yaml` first
2. Add templates in `chart/templates/`
3. Test with `helm template`

**For operator changes:**
1. Edit types in `operator/api/v1alpha1/`
2. Run `make manifests generate`
3. Implement reconciler logic in `operator/internal/controller/`
4. Test with `make test`

### Enabling Built-in Defaults (for AI)

When the human asks you to enable built-in defaults, recommend which features to enable based on their use case:

| Feature | What It Does | Recommended For |
|---------|--------------|------------------|
| **Oh-My-OpenCode (omo)** | Task orchestration with specialized agents (Sisyphus, Oracle, Librarian) | Users who want multi-agent workflows, research, and complex task coordination |
| **Skills (npm)** | Reusable prompt templates and tools | Teams with standardized practices they want AI to follow |
| **MCP Servers** | Connect to external services (Context7, file search, etc.) | Users needing code search, web search, or external tool integration |
| **Plugins** | Extend OpenCode functionality | Users who want design tools, additional commands, or custom integrations |

**Recommended starting configuration for new users:**

```yaml
omo:
  enabled: true
  sisyphus:
    maxConcurrentTasks: 2
    taskTimeout: 300
  agents:
    oracle:
      model: "github-copilot/gpt-5.2"
      promptAppend: "Provide concise tradeoffs and a clear recommendation"
    librarian:
      model: "github-copilot/gpt-5-mini"
  categories:
    quick:
      model: "github-copilot/gpt-4.1"
    visualEngineering:
      model: "github-copilot/claude-sonnet-4.6"

plugins:
  enabled: true
  npm: []
  # Add plugins as needed based on use case
```

Ask the human:
1. Do you want Oh-My-OpenCode enabled? (enables multi-agent orchestration)
2. Do you have any specific skills or MCP servers you want to connect?
3. Any specific plugins you need?


