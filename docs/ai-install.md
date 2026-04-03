# AI Installation Instructions

Quick install for AI agents working with k8s-opencode.

## Prereq Checklist (AI: Verify These First)

Before proceeding, verify the human has:

- [ ] Kubernetes cluster access (your kubeconfig works)
- [ ] Helm 3.x installed (`helm version` works)
- [ ] Tailscale account with OAuth client (for ingress)

---

## Step 1: Ask Deployment Mode

**First, ask the human which mode they want:**

> "k8s-opencode supports two deployment modes:
> 
> 1. **Single-user** (recommended for personal use)
>    - Simple Helm install, one OpenCode instance for you
>    - Quick setup, no operator needed
> 
> 2. **Multi-user** (for teams/enterprises)
>    - Install the Kubernetes operator first
>    - Creates isolated workspaces per user via Custom Resource Definition
>    - Dynamic provisioning, each user gets their own namespace
> 
> Which mode would you like?"

---

## Step 2: Gather Required Info (Based on Mode)

### Single-User Mode

If they choose single-user, ask for:

| Value | How to Get | Example |
|-------|-----------|---------|
| `serverPassword` | Ask human | `"my-secret-password"` |
| Username | Optional — defaults to `opencode` | `"timothy"` |

> **API key is optional** — Don't ask for it. After logging in, run `/connect` to authenticate with 75+ providers.
> 
> **Note:** The Tailscale tailnet name is automatically derived — don't ask.

**Prompt:**
> "I need 1 value to install single-user k8s-opencode:
> - `serverPassword` - What password should I set for the OpenCode web UI?
> 
> (Optional: What username should I use? Defaults to 'opencode'. This determines your home directory name.)"

### Multi-User Mode

If they choose multi-user:

1. **First install the operator** (one-time):
```bash
kubectl apply -f https://raw.githubusercontent.com/timothyclin/k8s-opencode/main/operator/dist/install.yaml
```

2. **Then create workspaces** — ask for per-user values:

> **For multiple users**: Provide one workspace YAML per user. You can collect all values at once or sequentially.

| Value | How to Get | Example |
|-------|-----------|---------|
| Workspace name | Ask human | `"alice"` |
| Email | Ask human | `"alice@example.com"` |
| Storage sizes | Optional, defaults work | workspace: 20Gi, data: 5Gi |

> **API key is optional** — Don't ask. After login, each user runs `/connect` to authenticate.

**Prompt for multiple users:**
> "I need the following for each user (provide as many as needed):
> - Workspace name (e.g., 'alice', 'bob')
> - Email address for each user
> 
> Storage defaults to 20Gi workspace / 5Gi data. Say if anyone needs different sizes."

---

## Step 3: Deploy

### Single-User Deploy

```yaml
# values.yaml — fill in serverPassword
serverPassword: "CHANGE_ME"

# API key optional - run /connect after login to authenticate
# providers:
#   anthropic:
#     enabled: true
#     apiKey: "sk-ant-your-key"

# Enable Oh-My-OpenCode (task orchestration with Sisyphus, Oracle, Librarian)
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

# Enable plugins
plugins:
  enabled: true

# Enable Tailscale ingress
ingress:
  enabled: true

# Storage
persistence:
  workspace:
    size: 20Gi
  data:
    size: 5Gi
```

```bash
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f values.yaml
```

### Multi-User Deploy (Create Workspace)

> **For multiple users**: Currently, create one `OpenCodeWorkspace` YAML per user. See [Bulk Import (Roadmap)](#bulk-import-roadmap) for future enhancement.

```yaml
# alice-workspace.yaml
apiVersion: opencode.opencode.io/v1alpha1
kind: OpenCodeWorkspace
metadata:
  name: CHANGE_ME          # Workspace name from human
spec:
  email: "CHANGE_ME"       # Email from human
  providers:
    anthropic:
      enabled: true        # Optional - can use /connect instead
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

---

## Bulk Import (Roadmap)

For teams with multiple users, the operator will eventually support bulk import via a CSV file:

**Future workflow:**
1. Human provides a CSV with columns: `name,email,storage_workspace,storage_data`
2. AI creates a ConfigMap with the CSV content
3. Operator watches ConfigMap and creates workspaces automatically

**Sample CSV (future):**
```csv
name,email,storage_workspace,storage_data
alice,alice@example.com,20Gi,5Gi
bob,bob@example.com,20Gi,10Gi
charlie,charlie@example.com,10Gi,5Gi
```

**Until then**: Create one `OpenCodeWorkspace` resource per user.

---

## Access

### Single-User

```
https://opencode.<your-tailnet>.ts.net
Username: opencode
Password: (serverPassword you set)
```

### Multi-User

Each workspace gets its own namespace. Access via:

```bash
# Replace with workspace name
kubectl port-forward -n oc-alice svc/opencode 4096:4096
```

Then open `http://localhost:4096`

---

## Key Facts (for AI reference)

| Item | Value | Notes |
|------|-------|-------|
| Chart OCI | `oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s` | Use in helm install/upgrade |
| Image | `ghcr.io/timothyclin/k8s-opencode/opencode-workspace` | Container image |
| Port | 4096 | OpenCode HTTP server port |
| Single-user namespace | `opencode` | Helm chart deploys here |
| Multi-user namespace prefix | `oc-` | Operator creates `oc-<workspace>` |
| Username | `opencode` | Configurable via `opencode.username` |
| User UID | 1000 | Non-root user |
| Storage - Data PVC | `/home/{username}` | Config, auth, MCP credentials |
| Storage - Workspace PVC | `/home/{username}/workspace` | Project files |
| Ingress URL | `https://opencode.<tailnet>.ts.net` | When `ingress.enabled=true` |

## Upgrade

```bash
# Single-user
helm upgrade ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode -f values.yaml
```

## Uninstall

```bash
# Single-user
helm uninstall ok8s -n opencode
kubectl delete namespace opencode

# Multi-user (operator stays, delete workspaces)
kubectl delete opencodeworkspace <name>
# Or remove operator entirely
kubectl delete -f https://raw.githubusercontent.com/timothyclin/k8s-opencode/main/operator/dist/install.yaml
```