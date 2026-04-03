# AI Installation Instructions

Quick install for AI agents working with k8s-opencode.

## Recommended Deploy (Single-User)

This includes Oh-My-OpenCode (task orchestration), plugins, and Tailscale ingress:

```yaml
# values.yaml
serverPassword: "your-secure-password"
providers:
  anthropic:
    enabled: true
    apiKey: "sk-ant-your-key"

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

# Storage (adjust as needed)
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

## Key Facts

- **Chart OCI**: `oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s`
- **Image**: `ghcr.io/timothyclin/k8s-opencode/opencode-workspace`
- **Port**: 4096
- **Storage**: 
  - Data PVC → `/home/{username}` (config, auth, MCP credentials)
  - Workspace PVC → `/home/{username}/workspace` (project files)
- **Username**: defaults to "opencode", configurable via `opencode.username`
- **User**: UID 1000 (non-root)
- **Init container**: Copies ConfigMap to data PVC on first run (config preserved across restarts)
- **Ingress URL**: `https://opencode.<your-tailnet>.ts.net` (when ingress.enabled=true)

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