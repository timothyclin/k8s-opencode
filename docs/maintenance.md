# K8s-OMO Maintenance Guide

Operational procedures for managing an OpenCode + Oh-My-OpenCode deployment on Kubernetes.

---

## User Management (Multi-User Mode)

### Add a User

1. Edit your `values.yaml` (or per-environment overrides):

```yaml
users:
  - name: alice
    password: "secure-password-here"
    workspaceSize: 20Gi
    providers:
      anthropic:
        enabled: true
        apiKey: "sk-ant-..."
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
```

2. Upgrade the release:

```bash
helm upgrade opencode ./chart -f values.yaml
```

3. A new StatefulSet will be created for the user.

### Delete a User

1. Remove the user entry from `values.yaml`.
2. Upgrade the release:

```bash
helm upgrade opencode ./chart -f values.yaml
```

3. The user's StatefulSet will be deleted. To reclaim PVC storage:

```bash
kubectl delete pvc -l app.kubernetes.io/instance=opencode,app.kubernetes.io/user=alice
```

### Change a User's Password

1. Update the `password` field for the user in `values.yaml`.
2. Upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

> **Note**: The password is stored in the per-user Secret. Upgrading will update the Secret, and the pod will restart with the new credentials.

---

## Binary Updates

### Update OpenCode Version

1. Update the image tag in `values.yaml`:

```yaml
image:
  repository: ghcr.io/anomalyco/opencode
  tag: "1.2.3"  # Pin to a specific version
```

2. Upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

> **Recommendation**: Always pin to a specific version tag, not `latest`, for reproducible deployments.

### Update Oh-My-OpenCode

Oh-My-OpenCode is configured via `oh-my-opencode.jsonc` in the ConfigMap. To update:

1. Update the `oh-my-opencode.jsonc` schema reference in `_helpers.tpl` if a new schema version is available.
2. Update any agent model references in `values.yaml` under `omo.agents.*` and `omo.categories.*`.
3. Upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

---

## MCP Server Management

### Add a Shared MCP Server (All Users)

```yaml
sharedMcp:
  - name: context7
    url: https://mcp.context7.com/mcp
    enabled: true
```

### Add a User-Specific MCP Server

```yaml
users:
  - name: alice
    mcp:
      remote:
        - name: custom-mcp
          url: https://example.com/mcp
          enabled: true
```

### Add a Laptop MCP Server (via Tailscale)

```yaml
mcp:
  laptopServers:
    - name: playwright
      tailscaleIP: "100.x.x.x"
      port: 3000
```

> **Prerequisite**: Tailscale operator must be installed and cluster egress configured. See `README.md` for Tailscale setup.

### Remove an MCP Server

Remove the entry from `values.yaml` and upgrade.

---

## Skill Management

### Add NPM Skills (Shared)

```yaml
sharedSkills:
  npm:
    - name: "@anthropic/skill-name"
```

### Add Config Skills (Shared)

```yaml
sharedSkills:
  config:
    - name: "skill-name"
      source: "url-or-path"
```

### Add User-Specific Skills

```yaml
users:
  - name: alice
    skills:
      npm:
        - name: "@custom/skill"
      config:
        - name: "custom-rule"
          source: "https://example.com/rule.md"
```

---

## Identity Files (AGENTS.md, .cursorrules, etc.)

Identity files are managed through Helm and copied to `/workspace/` on pod startup
via an init container. Files are only copied if they don't already exist — user
edits are preserved across restarts.

### Configure Identity Files

Add files under `identity` in your values file:

```yaml
identity:
  enabled: true
  files:
    AGENTS.md: |
      # Project Identity
      Instructions for AI agents working in this workspace.
    .cursorrules: |
      # Cursor Rules
      - Always write TypeScript
      - Use functional components
```

Then upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

### Update Identity Files

1. Edit the file content under `identity.files` in your values file.
2. Upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

3. **Important**: Existing files in `/workspace/` are NOT overwritten. To push
   updated defaults to existing users, they must delete the file from their
   workspace first, then restart the pod:

```bash
kubectl exec -it <pod-name> -- rm /workspace/AGENTS.md
kubectl rollout restart deployment/opencode          # single-user
kubectl rollout restart statefulset/<release>-<user>  # multi-user
```

### Disable Identity Files

Set `identity.enabled: false` and upgrade. The init container and ConfigMap
will be removed on next upgrade. Existing files in `/workspace/` are unaffected.

---

## Provider API Keys

### Single-User Mode

Update the provider configuration in `values.yaml`:

```yaml
providers:
  anthropic:
    enabled: true
    apiKey: "sk-ant-new-key"
```

Then upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

### Multi-User Mode

Each user has their own provider configuration under `users[].providers.*`. Update the specific user's entry and upgrade.

---

## OIDC Authentication (Multi-User Mode)

For multi-user deployments, you can use OIDC authentication via oauth2-proxy instead of per-user passwords.

### How It Works

A single shared oauth2-proxy sits in front of all users. All OIDC traffic flows through one dedicated auth hostname:

- **`auth.oidc.hostname`** — a single hostname for the OAuth callback (default: `ok8s-auth`). The full callback URL is:
  `https://<hostname>-<namespace>.<tailnet>.ts.net/oauth2/callback`
- **`cookieDomain`** — the session cookie is set on the parent tailnet domain (e.g. `.lynx-beta.ts.net`) so it's valid across all per-user hostnames.

**One callback URL for ALL users.** No per-user callback URLs needed.

### Step 1: Create an OAuth Client

#### Google Workspace

1. Go to [Google Cloud Console → Credentials](https://console.cloud.google.com/apis/credentials)
2. Create or select a project
3. Configure the **OAuth consent screen**:
   - User Type: External (or Internal for Google Workspace)
   - App name: e.g. "OpenCode"
   - User support email: your email
   - Authorized domains: add your tailnet domain (e.g. `lynx-beta.ts.net`)
   - Scopes: `email`, `profile`, `openid` (default for OIDC)
   - Add test users unless you publish the app
4. Click **+ CREATE CREDENTIALS → OAuth client ID**
   - Application type: **Web application**
   - Name: e.g. "ok8s-auth"
   - Authorized redirect URI (just one):
     ```
     https://ok8s-auth-opencode.lynx-beta.ts.net/oauth2/callback
     ```
     Format: `https://<hostname>-<namespace>.<tailnet>.ts.net/oauth2/callback`
   - Click Create and note the **Client ID** and **Client Secret**

#### GitHub

1. Go to [GitHub Settings → Developer Settings → OAuth Apps](https://github.com/settings/developers)
2. Click **New OAuth App**
   - Application name: e.g. "ok8s-auth"
   - Homepage URL: `https://ok8s-auth-opencode.lynx-beta.ts.net`
   - Authorization callback URL: `https://ok8s-auth-opencode.lynx-beta.ts.net/oauth2/callback`
3. Note the **Client ID** and generate a **Client Secret**

#### Auth0

1. Go to [Auth0 Dashboard → Applications](https://manage.auth0.com/)
2. Create a **Regular Web Application**
3. Under Settings → Allowed Callback URLs, add:
   `https://ok8s-auth-opencode.lynx-beta.ts.net/oauth2/callback`
4. Note the **Domain** (used as `provider`), **Client ID**, and **Client Secret**

### Step 2: Generate a Cookie Secret

```bash
python3 -c "import base64,os; print(base64.b64encode(os.urandom(32)).decode())"
```

### Step 3: Configure the Chart

```yaml
auth:
  oidc:
    enabled: true
    provider: "https://accounts.google.com"       # OIDC issuer URL
    clientId: "your-client-id"
    clientSecret: "your-client-secret"
    cookieSecret: "base64-32-byte-secret-from-step-2"
    emailDomain: "example.com"                     # Or "*" for any domain
    hostname: "ok8s-auth"                          # Single auth hostname (default)
    cookieDomain: ".lynx-beta.ts.net"              # Your tailnet domain (with leading dot)
```

> **`cookieDomain` is required for multi-user OIDC.** It must be your tailnet domain prefixed with `.` — this allows the auth cookie to be shared across all per-user hostnames.

### Step 4: Create Tailscale Ingress for Auth

The auth hostname needs its own Tailscale ingress so users can reach the OAuth callback:

```yaml
# Add to your values.yaml
auth:
  oidc:
    ingress:
      enabled: true
```

Or create the ingress manually:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ok8s-auth-ingress
  annotations:
    tailscale.com/hostname: "ok8s-auth-opencode"
spec:
  ingressClassName: tailscale
  defaultBackend:
    service:
      name: ok8s-oauth2-proxy
      port:
        number: 4180
```

### Step 5: Upgrade

```bash
helm upgrade ok8s ./chart -n opencode -f values.yaml
```

### Adding a New User After OIDC is Configured

No OAuth changes needed — just add the user to `values.yaml` and upgrade. The single auth hostname handles all users automatically.

---

## Tailscale Management

### Update Tailscale Proxy Tag

```yaml
tailscaleOperator:
  proxyTag: "tag:k8s-new"
```

### Rotate Tailscale ACL Keys

Update your Tailscale ACL policy file in the Tailscale admin console. No Helm changes needed — the operator picks up ACL changes automatically.

---

## Kubedock: Test Containers as Kubernetes Pods

Kubedock translates the Docker API into Kubernetes Pod creation. This prevents
OOM kills from running Docker-in-Docker (DinD) sidecars by spawning test
containers as native K8s Pods instead.

### Enable Kubedock

```yaml
kubedock:
  enabled: true
  image:
    tag: "0.20.3"  # Pin to a specific version
  extraArgs: []
  # - "--port-forward"
```

Then upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

When enabled:
- **Single-user mode**: A shared kubedock deployment is created
- **Multi-user mode**: Per-user kubedock deployments with NetworkPolicy isolation
- `DOCKER_HOST`, `TESTCONTAINERS_RYUK_DISABLED`, and `TESTCONTAINERS_CHECKS_DISABLE` env vars are auto-injected into agent pods

### Disable Kubedock

Set `kubedock.enabled: false` in `values.yaml` and upgrade. The kubedock deployments and services will be removed.

### Update Kubedock Version

```yaml
kubedock:
  image:
    tag: "0.21.0"  # Check for latest stable release
```

Then upgrade:

```bash
helm upgrade opencode ./chart -f values.yaml
```

### Testcontainers Configuration

When kubedock is enabled, agents automatically use these environment variables:

```bash
DOCKER_HOST: "tcp://<kubedock-service>:2475"
TESTCONTAINERS_RYUK_DISABLED: "true"
TESTCONTAINERS_CHECKS_DISABLE: "true"
```

Your test framework (JUnit, pytest, etc.) will use `DOCKER_HOST` to connect to kubedock instead of Docker.

### Check Kubedock Status

```bash
# Single-user mode
kubectl get deployment -l app.kubernetes.io/component=kubedock
kubectl logs -l app.kubernetes.io/component=kubedock

# Multi-user mode (per-user)
kubectl get deployment -l app.kubernetes.io/component=kubedock
kubectl get networkpolicy -l app.kubernetes.io/name=ok8s
```

### Troubleshooting Kubedock

**Test containers failing to start:**

```bash
# Check kubedock logs
kubectl logs -l app.kubernetes.io/component=kubedock

# Verify DOCKER_HOST is set in agent pod
kubectl exec -it <agent-pod> -- echo $DOCKER_HOST

# Check kubedock service exists
kubectl get svc -l app.kubernetes.io/component=kubedock
```

**RBAC issues:**

```bash
# Verify kubedock service account and role
kubectl get serviceaccount -l app.kubernetes.io/component=kubedock
kubectl get role -l app.kubernetes.io/component=kubedock
kubectl get rolebinding -l app.kubernetes.io/component=kubedock
```

**NetworkPolicy blocking traffic (multi-user mode):**

```bash
kubectl get networkpolicy -l app.kubernetes.io/name=ok8s
kubectl describe networkpolicy <policy-name>
```

---

## Backup and Restore

### Manual Backup

```bash
kubectl get secret -l app.kubernetes.io/instance=opencode -n opencode -o yaml > backup-secrets.yaml
kubectl get configmap -l app.kubernetes.io/instance=opencode -n opencode -o yaml > backup-configs.yaml
```

### Restore from Backup

```bash
kubectl apply -f backup-secrets.yaml
kubectl apply -f backup-configs.yaml
```

---

## Troubleshooting

### Pod Not Starting

```bash
kubectl describe pod -l app.kubernetes.io/instance=opencode
kubectl logs -l app.kubernetes.io/instance=opencode
```

### Check Config Mount

```bash
kubectl exec -it opencode-0 -- ls -la /root/.config/opencode/
kubectl exec -it opencode-0 -- cat /root/.config/opencode/opencode.jsonc
```

### Health Check

```bash
kubectl exec -it opencode-0 -- wget -qO- http://localhost:4096/global/health
```

### Check oauth2-proxy Status

```bash
kubectl logs -l app.kubernetes.io/name=oauth2-proxy
kubectl get pods -l app.kubernetes.io/name=oauth2-proxy
```

### Tailscale Ingress Not Working

```bash
kubectl get ingress -l app.kubernetes.io/instance=opencode
kubectl describe ingress <ingress-name>
```
