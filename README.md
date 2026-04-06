# k8s-opencode

OpenCode on Kubernetes, with Tailscale connectivity for remote access and laptop
MCP servers.

## AI Installation

For AI agents, see [docs/ai-install.md](docs/ai-install.md) for the full installation guide.

**Quick summary for AI agents:**

1. **First ask: Single or Multi-user?** — k8s-opencode has two modes:
   - **Single-user**: Simple Helm install, one OpenCode for personal use
   - **Multi-user**: Install operator first, then create workspaces per user

2. **Gather required info based on mode:**
   - Single-user: just `serverPassword`
   - Multi-user: operator installed first, then `name` + `email` per user

   > **API key is optional** — Don't ask for it. After login, run `/connect` to authenticate.

3. **Then deploy** — Use the recommended values from `docs/ai-install.md`

4. **After install** — Tell human the access URL (the tailnet name is determined automatically from the cluster — do NOT ask for it):
   ```
   https://oc-<username>.<tailnet>.ts.net
   # Default username: https://oc-opencode.<tailnet>.ts.net
   Username: (username, defaults to opencode)
   Password: (the serverPassword you set)
   ```
   > The `<tailnet>` placeholder is filled automatically by Tailscale — you don't need to know it in advance.

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

> No API key required at install time. After logging in, run `/connect` to authenticate.

### Install

**No version required** — omit `--version` to use the latest published version:

```bash
# Latest version (recommended - no --version flag needed)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --set serverPassword=your-secure-password

# Specific version (only if you need an exact version - check GitHub releases)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --version <version> \
  --set serverPassword=your-secure-password
```

Or with a values file:

```bash
# Latest version (recommended)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f my-values.yaml

# Specific version (check GitHub releases for available versions)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --version <version> \
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
https://oc-<username>.<your-tailnet>.ts.net
# Example (default username): https://oc-opencode.<your-tailnet>.ts.net
# Example (username=alice): https://oc-alice.<your-tailnet>.ts.net
Password: (your serverPassword value)
```

### Configuration

See [chart/values.yaml](chart/values.yaml) for all options. Key sections:

| Section | Purpose |
|---------|---------|
| `serverPassword` | HTTP auth for the OpenCode server |
| `opencode.username` | Home directory user (default: opencode) |
| `resources.limits.memory` | Container memory limit (default: 2Gi) |
| `mcp.remote[]` | Remote MCP servers (URLs) |
| `mcp.laptopServers[]` | Laptop MCP servers (via Tailscale egress) |
| `ingress.enabled` | Expose OpenCode UI to tailnet |
| `kubedock.enabled` | Enable kubedock (default: true) |
| `persistence.*` | Storage for home dir and workspace |

> **Note:** LLM API keys are optional. After logging in, run `/connect` to authenticate with 75+ providers.

### Enabling Oh-My-OpenCode, Skills, MCPs, and Plugins

By default, the chart enables Oh-My-OpenCode, Context7 MCP, and user skills. To disable or customize, add the following to your values file:

```yaml
# Disable Oh-My-OpenCode (task orchestration, specialized agents)
omo:
  enabled: false
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

# Override default skills (empty by default - install via npm packages as needed)
skills:
  npm: []
  config: []

# Override default MCP servers (default: context7)
mcp:
  remote: []
  laptopServers: []

# Disable plugins (default: oh-my-opencode, @tarquinen/opencode-dcp, superpowers)
plugins:
  enabled: false
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
        namespace: oc-alice
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

- **Namespace** — `oc-<name>` (isolated per user)
- **PVCs** — `workspace-pvc` and `data-pvc` for persistent storage
- **ConfigMap** — `opencode-config` with `opencode.json` configuration
- **NetworkPolicy** — isolates user workloads
- **Service** — ClusterIP on port 4096
- **StatefulSet** — single-replica OpenCode pod

### API Key Management

API keys can be provided in three ways:

**1. Via Secret reference (pre-configured):**

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

**2. Via /connect command (after login - recommended):**

After accessing the workspace, run `/connect` in the terminal to link your OpenCode
account. OpenCode supports 75+ providers including Anthropic, OpenAI, Google,
OpenRouter, and many more. This is the simplest approach - no secret management needed.

```bash
/connect
```

**3. OAuth (provider-specific):**

```yaml
spec:
  providers:
    anthropic:
      enabled: true
      # No apiKeySecretRef — uses OAuth
```

### Access User Workspace

Each workspace gets its own namespace. Access via shared Tailscale frontend with auth router:

When OIDC auth is enabled (`auth.oidc.enabled: true`), all users access via a shared endpoint:
```
https://<hostname>.<namespace>.<tailnet>.ts.net
```

The auth router validates the OIDC session from the cookie and routes to the correct user pod internally — users don't need to know which pod they're on.

If OIDC is not enabled, access via port-forward:

```bash
kubectl port-forward -n oc-alice svc/opencode 4096:4096
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
    ├── oc-alice namespace
    │   ├── StatefulSet (1 replica)
    │   ├── PVCs (workspace + data)
    │   ├── ConfigMap (opencode.json)
    │   ├── NetworkPolicy
    │   └── Service
    │
    └── oc-bob namespace
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

### How to Help (AI Agent)

When a human asks you to install or configure k8s-opencode:

**1. Ask deployment mode first** — Use the `question` tool:

```json
{
  "questions": [{
    "question": "Which deployment mode do you want?",
    "header": "Deployment Mode",
    "options": [
      {"label": "Single-user (Recommended)", "description": "Simple Helm install, one OpenCode instance for personal use"},
      {"label": "Multi-user", "description": "Install Kubernetes operator first, then create isolated workspaces per user"}
    ]
  }]
}
```

**2. Gather required values** — Use the `question` tool to collect ALL values (single-user example):

> **⚠️ CRITICAL: Ask ALL questions.** Do not skip optional fields — the question tool below is complete for a reason. Defaults are applied automatically if not provided.

```json
{
  "questions": [
    {
      "question": "What password should I set for the OpenCode web UI? (This will be stored as a Kubernetes secret)",
      "header": "Server Password",
      "options": []
    },
    {
      "question": "What username? Defaults to 'opencode'. Sets your home directory and access URL.",
      "header": "Username (optional)",
      "options": []
    },
    {
      "question": "Memory limit? Defaults to 2Gi. Increase for large projects (e.g. 4Gi).",
      "header": "Memory Limit (optional)",
      "options": []
    },
    {
      "question": "Disable kubedock? It's enabled by default to run test containers as K8s pods.",
      "header": "Kubedock",
      "options": [
        {"label": "Keep enabled (Recommended)", "description": "Run test containers as Kubernetes pods"},
        {"label": "Disable", "description": "Disable kubedock"}
      ]
    }
  ]
}
```

> API key is optional — don't ask. After login, run `/connect` to authenticate.
> 
> **Do NOT ask for the Tailscale tailnet name** — it is determined automatically from the cluster.
> 
> **Note:** The `question` tool collects free-text answers via "Type your own answer" — there is no masked/password input type. Just label the field clearly as a password/secret.

**3. Use the recommended deploy** — See [docs/ai-install.md](docs/ai-install.md) for the full config with Oh-My-OpenCode enabled.

**4. Tell them the access URL after install:**
```
https://oc-<username>.<tailnet>.ts.net
# Default username: https://oc-opencode.<tailnet>.ts.net
Username: (username, defaults to opencode)
Password: (the serverPassword you set)
```


