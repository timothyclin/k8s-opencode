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

## Quick Start

### Prerequisites

- Kubernetes cluster (ARM64 or amd64)
- Helm 3.x
- Tailscale account (free tier works)

> No API key required at install time. After logging in, run `/connect` to authenticate.

### For Humans

**Option 1: Let an LLM agent do it (Recommended)**

Copy and paste this to your LLM agent (Claude Code, Cursor, OpenCode, etc.):

```
Install k8s-opencode by following the instructions here:
https://raw.githubusercontent.com/timothyclin/k8s-opencode/main/docs/ai-install.md
```

**Option 2: Manual install**

```bash
# Create values.yaml with your serverPassword, then:
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f values.yaml
```

> **Security Note:** Use a values file instead of `--set` to avoid exposing your password in shell history.

See [Manual Installation - Single-User](#manual-installation---single-user) for full configuration options.

### For LLM Agents

Fetch the installation guide and follow it:

```
curl -s https://raw.githubusercontent.com/timothyclin/k8s-opencode/main/docs/ai-install.md
```

---

## Manual Installation - Single-User

One OpenCode instance for personal use. Uses a standard Kubernetes `Deployment`.

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

## Manual Installation - Multi-User

Dynamic user provisioning via Kubernetes CRD. Each user gets an isolated
workspace with dedicated storage, config, and network policy.

### Prerequisites

- Kubernetes cluster (ARM64 or amd64)
- kubectl configured
- Tailscale Kubernetes Operator installed

### Install the Operator

```bash
kubectl apply -f https://github.com/timothyclin/k8s-opencode/releases/latest/download/install.yaml
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
  
  # LLM providers (optional - can use /connect instead)
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
  
  # Plugins (default: oh-my-opencode, opencode-dcp, superpowers)
  plugins:
    enabled: true
    npm:
      - "superpowers@git+https://github.com/obra/superpowers.git"
  
  # MCP servers
  mcp:
    remote:
      - name: context7
        url: https://mcp.context7.com/mcp
        enabled: true
    # Laptop MCP servers (via Tailscale egress)
    # laptopServers:
    #   - name: playwright
    #     tailscaleFqdn: my-laptop.tail12345.ts.net
    #     port: 3000
    #     enabled: true
  
  # Skills configuration
  skills:
    npm: []  # Add npm skill packages as needed
    # config: []  # Or inline skill configs
  
  # Storage
  storage:
    workspace: "20Gi"
    data: "5Gi"
  
  # Kubedock (test containers as K8s pods)
  kubedock:
    enabled: true
```

```bash
kubectl apply -f alice-workspace.yaml
```

### What the Operator Creates

For each `OpenCodeWorkspace` CR, the operator reconciles:

- **Namespace** — `oc-<name>` (isolated per user)
- **PVCs** — `workspace-pvc` and `data-pvc` for persistent storage
- **ConfigMap** — `opencode-config` with complete `opencode.json` configuration including:
  - Default plugins (oh-my-opencode@latest, @tarquinen/opencode-dcp@latest)
  - User-specified npm plugins
  - MCP servers (remote URLs and laptop servers via Tailscale)
  - Skills (npm packages and inline configs)
  - LLM provider configuration and API keys
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

### Enable Ingress (Multi-User Operator)

To expose a workspace via Tailscale, add the `tailscale` spec to your OpenCodeWorkspace:

```yaml
apiVersion: opencode.opencode.io/v1alpha1
kind: OpenCodeWorkspace
metadata:
  name: alice
spec:
  email: "alice@example.com"
  tailscale:
    ingressTags:
      - "tag:your-tag"  # Must be permitted in your Tailscale ACL
```

The operator will create a Tailscale Ingress that exposes the workspace at:
```
https://oc-<workspace>-<prefix>.<your-tailnet>.ts.net
```

**Setting up Tailscale Tags:**

Tailscale ingress proxies are created as devices in your tailnet. Each device must have a tag that is permitted in your ACL policy.

1. **Create a tag** in [Tailscale Admin Console](https://login.tailscale.com/admin/settings/tags)
2. **Add tag owners** in your ACL policy:

```json
{
  "tagOwners": {
    "tag:opencode": ["your-email@example.com"]
  }
}
```

3. **Use your tag** in the OpenCodeWorkspace spec (as shown above)

See [Tailscale Tags Documentation](https://tailscale.com/docs/features/tags) for full details.

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

## Known Issues

### Stale Tailscale DNS Entries (oc-user-2, oc-user-3, etc.)

**Symptom**: After reinstalling the chart, the ingress hostname gets assigned a `-2`, `-3`, etc. suffix instead of the expected hostname.

**Root Cause**: This is a known bug in the Tailscale Kubernetes operator (v1.94.2 and earlier). When the operator encounters an optimistic lock error during reconciliation, it may delete and recreate the proxy StatefulSet with a new random suffix. Each new pod registers a new Tailscale device with a new hostname, but the old device is never deleted from the tailnet.

See: [tailscale/tailscale#18922](https://github.com/tailscale/tailscale/issues/18922)

**Impact**:
- Stale DNS entries accumulate (`oc-username`, `oc-username-1`, `oc-username-2`, etc.)
- Each stale entry corresponds to a Tailscale machine that must be manually removed

**Workarounds**:

1. **Manual cleanup** (recommended):
   - Go to [Tailscale Admin Console](https://login.tailscale.com/admin/machines)
   - Delete the stale machine entries for your namespace
   - Reinstall the chart - the correct hostname should be assigned

2. **Avoid rapid reinstalls**:
   - Wait at least 30 seconds between `helm uninstall` and `helm install`
   - Avoid triggering concurrent modifications during reconciliation

3. **The chart's cleanup job** (improved in v0.2.10):
   - Adds graceful termination with 30s grace period
   - Waits 15s for Tailscale control plane to process deregistration
   - This helps but cannot fully prevent the issue due to the operator bug

**Note**: Upgrading the Tailscale operator to a version that fixes this issue (when available) is the long-term solution.

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
