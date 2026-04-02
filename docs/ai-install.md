# AI Installation Instructions

Quick install for AI agents working with k8s-opencode.

## Quick Deploy (Single-User)

```bash
# Latest version - no --version flag needed
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace \
  -f values.yaml
```

## With Custom Values

```yaml
# values.yaml
serverPassword: "secure-password"
providers:
  anthropic:
    enabled: true
    apiKey: "sk-ant-your-key"
persistence:
  workspace:
    size: 20Gi
  data:
    size: 5Gi
```

```bash
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart/ok8s -n opencode --create-namespace -f values.yaml
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
- **Storage**: PVCs mounted at `/home/opencode/workspace`, `/home/opencode/.local`, `/home/opencode/.config`
- **User**: UID 1000 (non-root)
- **Init container**: Runs as root to fix PVC permissions

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
