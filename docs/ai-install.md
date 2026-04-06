# AI Installation Instructions

Quick install for AI agents working with k8s-opencode.

## Prereq Checklist (AI: Verify These First)

Before proceeding, verify the human has:

- [ ] Kubernetes cluster access (your kubeconfig works)
- [ ] Helm 3.x installed (`helm version` works)
- [ ] Tailscale account with OAuth client (for ingress)

---

## Step 1: Ask Deployment Mode

**Use the `question` tool to ask which mode they want:**

```json
{
  "questions": [{
    "question": "Which deployment mode do you want?",
    "header": "Deployment Mode",
    "options": [
      {"label": "Single-user (Recommended)", "description": "Simple Helm install, one OpenCode instance for personal use. Quick setup, no operator needed."},
      {"label": "Multi-user", "description": "Install the Kubernetes operator first. Creates isolated workspaces per user via CRD. Dynamic provisioning, each user gets their own namespace."}
    ]
  }]
}
```

---

## Step 2: Gather Required Info (Based on Mode)

### Single-User Mode

**MUST ask for ALL of these values** using the question tool — do not skip optional fields:

| Value | Required | How to Get | Default | Example |
|-------|----------|-----------|---------|---------|
| `serverPassword` | ✅ Yes | Ask human | (none) | `"my-secret-password"` |
| `Username` | ❌ No | Ask human | `"opencode"` | `"timothy"` |
| Memory limit | ❌ No | Ask human | `"2Gi"` | `"4Gi"` |
| Kubedock | ❌ No | Ask human | `true` (enabled) | `false` to disable |

> **IMPORTANT:** The question tool MUST include ALL fields — do not skip "optional" fields. Defaults are applied automatically if not provided. Asking ensures the user is aware of configurable options.

> **API key is optional** — Don't ask for it. After logging in, run `/connect` to authenticate with 75+ providers.
> 
> **Note:** Username determines ingress hostname. For username "alice", URL is `https://oc-alice.<tailnet>.ts.net`. You don't need to ask for the tailnet name.

**Use the `question` tool to collect values interactively:**

> **⚠️ CRITICAL: Include ALL questions.** Do NOT skip optional fields — defaults are applied automatically. The question tool below includes ALL four fields for a reason. Ask all of them to ensure the user is aware of configurable options.

Use your agent's `question` tool to show an interactive form. The `question` tool collects values as free-text answers ("Type your own answer") — there is no masked/password input type. Label the password field clearly so the human knows it's a secret value. Example:

```json
{
  "questions": [
    {
      "question": "What password should I set for the OpenCode web UI? (This will be stored as a Kubernetes secret)",
      "header": "Server Password",
      "options": []
    },
    {
      "question": "What username? Defaults to 'opencode'. This sets your home directory and access URL.",
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

> **Do NOT ask for the Tailscale tailnet name** — it is determined automatically.

If your agent platform does not support interactive input fields, ask for the password in a clearly labeled field and note that it will be used as a secret value.

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

Defaults now include Oh-My-OpenCode, Superpowers skills, Context7 MCP, user skills, kubedock, and 2Gi memory. Just set your password:

```yaml
# values.yaml — fill in serverPassword
serverPassword: "CHANGE_ME"

# All defaults are pre-configured:
# - Oh-My-OpenCode (omo.enabled: true)
# - Context7 MCP server
# - User skills (empty by default - use npm skill packages)
# - Kubedock enabled (test containers as K8s pods)
# - 2Gi memory limit
# - 20Gi workspace / 5Gi data storage

# Enable Tailscale ingress (recommended)
ingress:
  enabled: true
```

Optional overrides (only if needed):

```yaml
# Disable Oh-My-OpenCode
omo:
  enabled: false

# Or change memory limit
resources:
  limits:
    memory: 4Gi

# Or disable kubedock
kubedock:
  enabled: false
```

```bash
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f values.yaml
```

Or with all options explicitly shown (for reference):

```yaml
# values.yaml — full example with defaults
serverPassword: "CHANGE_ME"

# Oh-My-OpenCode: enabled by default
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

# Plugins: enabled by default (oh-my-opencode, @tarquinen/opencode-dcp, superpowers)
plugins:
  enabled: true

# MCP servers: context7 enabled by default
mcp:
  remote:
    - name: context7
      url: https://mcp.context7.com/mcp
      enabled: true

# Skills: none by default - install via npm packages as needed
skills:
  npm: []
  config: []

# Kubedock: enabled by default (runs test containers as K8s pods)
kubedock:
  enabled: true

# Container resources: 2Gi memory by default
resources:
  limits:
    memory: 2Gi

# Tailscale ingress
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
https://oc-<username>.<your-tailnet>.ts.net
# Example (default username): https://oc-opencode.<your-tailnet>.ts.net
# Example (username=alice): https://oc-alice.<your-tailnet>.ts.net
Username: (your username - defaults to opencode)
Password: (serverPassword you set)
```

### Multi-User

Each workspace gets its own namespace. Access depends on auth configuration:

**With OIDC auth enabled** (recommended for teams):
- Users access via shared endpoint: `https://<hostname>.<namespace>.<tailnet>.ts.net`
- Auth router validates OIDC session from cookie and routes to correct user pod

**Without OIDC auth** (basic mode):
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
| Ingress URL | `https://oc-<username>.<tailnet>.ts.net` | When `ingress.enabled=true` |

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