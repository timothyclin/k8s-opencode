# AI Installation Instructions

Quick install for AI agents working with k8s-opencode.

## Prereq Checklist (AI: Verify These First)

Before proceeding, verify the human has:

- [ ] Kubernetes cluster access (your kubeconfig works)
- [ ] Helm 3.x installed (`helm version` works)
- [ ] Tailscale account with OAuth client (for ingress)

## What You Need (AI: Ask Human First)

Before running the install command, you **MUST** get these values from the human:

| Value | How to Get | Example |
|-------|-----------|---------|
| `serverPassword` | Ask human | `"my-secret-password"` |
| `providers.anthropic.apiKey` | Ask human for their Anthropic API key | `"sk-ant-api03-..."` |
| Username (workspace name) | Ask human — determines namespace as `oc-<username>` | `"timothy"` |

> **Note:** The Tailscale tailnet name is automatically derived — don't ask the human for it.

**Prompt to send to human:**
> "I need 3 values to install k8s-opencode:
> 1. `serverPassword` - What password should I set for the OpenCode web UI?
> 2. Your Anthropic API key - What's your API key (starts with `sk-ant-`)?
> 3. What username should I use? (This determines your namespace: `oc-<username>`)
> 
> Also: Do you already have the Tailscale Kubernetes Operator installed in your cluster?"

## Quick Start (After You Have the Values)

```yaml
# values.yaml — copy this and fill in the values you got from human
serverPassword: "CHANGE_ME"           # REQUIRED — get from human
providers:
  anthropic:
    enabled: true
    apiKey: "CHANGE_ME"              # REQUIRED — get from human

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

# Enable plugins (including oh-my-opencode and tarquinen/opencode-dcp)
plugins:
  enabled: true
  npm: []

# Enable Tailscale ingress (exposes OpenCode at https://opencode.<tailnet>.ts.net)
ingress:
  enabled: true
  # tailnet: "CHANGE_ME"            # Uncomment and set if you want custom subdomain

# Storage (adjust as needed)
persistence:
  workspace:
    size: 20Gi
  data:
    size: 5Gi
```

```bash
# Run this AFTER filling in the values
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f values.yaml
```

### Access URL (after install)

```
https://opencode.<your-tailnet>.ts.net
Username: opencode
Password: (the serverPassword you set)
```

## Minimal Deploy (Vanilla OpenCode)

Only if you need vanilla OpenCode without Oh-My-OpenCode:

```yaml
# values.yaml
serverPassword: "your-secure-password"
providers:
  anthropic:
    enabled: true
    apiKey: "sk-ant-your-key"
```

```bash
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f values.yaml
```

## Specific Version (if needed)

Check GitHub releases for available versions: https://github.com/timothyclin/k8s-opencode/releases

```bash
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  --version <version> -f values.yaml
```

## Key Facts (for AI reference)

| Item | Value | Notes |
|------|-------|-------|
| Chart OCI | `oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s` | Use in helm install/upgrade |
| Image | `ghcr.io/timothyclin/k8s-opencode/opencode-workspace` | Container image |
| Port | 4096 | OpenCode HTTP server port |
| Namespace | `opencode` | Required — don't use `default` |
| Username | `opencode` | Configurable via `opencode.username` |
| User UID | 1000 | Non-root user |
| Storage - Data PVC | `/home/{username}` | Config, auth, MCP credentials |
| Storage - Workspace PVC | `/home/{username}/workspace` | Project files |
| Ingress URL | `https://opencode.<tailnet>.ts.net` | When `ingress.enabled=true` |
| Upgrade command | `helm upgrade ok8s ...` | Use same `-f values.yaml` |
| Uninstall | `helm uninstall ok8s -n opencode && kubectl delete namespace opencode` | Full cleanup |

## Upgrade

```bash
helm upgrade ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode -f values.yaml
```

## Uninstall

```bash
helm uninstall ok8s -n opencode
kubectl delete namespace opencode
```

## Multi-User (Operator)

```bash
kubectl apply -f https://raw.githubusercontent.com/timothyclin/k8s-opencode/main/operator/dist/install.yaml
```