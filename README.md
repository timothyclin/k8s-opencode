# k8s-omo

OpenCode + Oh-My-OpenCode on Kubernetes, with Tailscale connectivity for remote
access and laptop MCP servers.

## What This Is

A Helm chart that deploys [OpenCode](https://opencode.ai) (AI coding agent) with
optional [Oh-My-OpenAgent](https://github.com/code-yeongyu/oh-my-openagent)
orchestration on your Kubernetes cluster. Tailscale provides secure access from
anywhere — no public ports, no ingress controllers, no TLS cert management.

**Two-way connectivity:**

- **Your laptop → Cluster**: Access the OpenCode web UI via Tailscale MagicDNS
- **Cluster → Your laptop**: OpenCode agents call MCP servers running locally on
  your machine

## Modes

### Single-User Mode (default)

One OpenCode instance. Simple, personal use. Uses a standard `Deployment`.

```yaml
mode: "single"
```

### Multi-User Mode

Per-user isolation with dedicated StatefulSet, PVCs, ConfigMap, Secret, and
ingress per user. Shared oauth2-proxy for OIDC authentication and an internal
router for host-based routing to each user service.

```yaml
mode: "multi"
users:
  - name: alice
    password: "secure-password"
    providers:
      anthropic:
        enabled: true
        apiKey: "sk-ant-..."
```

## Quick Start

### Prerequisites

- Kubernetes cluster (ARM64 or amd64)
- Helm 3.x
- Tailscale account (free tier works)
- At least one LLM provider API key

### 1. Install

```bash
helm install ok8s ./chart -n opencode --create-namespace \
  --set providers.anthropic.enabled=true \
  --set providers.anthropic.apiKey=sk-ant-your-key \
  --set serverPassword=your-secure-password
```

Or with a values file:

```bash
helm install ok8s ./chart -n opencode --create-namespace -f my-values.yaml
```

> **Namespace is required** — the chart will fail if installed into `default`. Always use `-n <namespace>`.

### 2. Verify

```bash
helm test opencode
```

### 3. Access OpenCode

After enabling Tailscale ingress (see [Tailscale Setup](#tailscale-setup)):

```
https://opencode.<your-tailnet>.ts.net
Password: (your serverPassword value)
```

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

### Enable Ingress

```bash
helm upgrade opencode ./chart -f my-values.yaml \
  --set ingress.enabled=true
```

### Expose Laptop MCP Servers

For each MCP server running on your laptop:

```yaml
# my-values.yaml
mcp:
  laptopServers:
    - name: playwright
      tailscaleIP: "100.x.x.x" # Your laptop's Tailscale IP (run: tailscale ip -4)
      port: 3000
```

The chart creates a Tailscale egress proxy so the cluster can reach your
laptop's MCP server.

## Configuration

See [values.yaml](chart/values.yaml) for all options. Key sections:

| Section               | Purpose                                                  |
| --------------------- | -------------------------------------------------------- |
| `mode`                | `"single"` (Deployment) or `"multi"` (StatefulSet)       |
| `users[]`             | Per-user config (multi-user mode only)                   |
| `sharedMcp[]`         | MCP servers shared across all users                      |
| `sharedSkills`        | Skills shared across all users                           |
| `providers.*`         | LLM API keys (anthropic, openai, google)                 |
| `serverPassword`      | HTTP auth for the OpenCode server                        |
| `auth.oidc.*`         | OIDC authentication via oauth2-proxy (multi-user mode)   |
| `omo.*`               | Oh-My-OpenCode agent config                              |
| `mcp.remote[]`        | Remote MCP servers (URLs)                                |
| `mcp.laptopServers[]` | Laptop MCP servers (via Tailscale cluster egress)        |
| `ingress.enabled`     | Expose OpenCode UI to tailnet                            |
| `secrets.backend`     | Secret management: `plain`, `sealed`, `sops`, `external` |
| `kubedock.*`          | Docker API → K8s Pod translation for test containers     |
| `identity.*`          | Default identity files (AGENTS.md, .cursorrules, etc.)   |
| `persistence.*`       | Storage for config data and workspace                    |

## Architecture

### Single-user mode

```
Your Tail net
├── Laptop (Tailscale node)
│   └── Local MCP servers (Playwright, browser tools, etc.)
│
└── ARM64 k8s Cluster
    ├── Tailscale Operator
    │   ├── Ingress proxy  ← exposes OpenCode UI to tailnet
    │   └── Egress proxies ← routes cluster traffic to laptop MCPs
    │
    └── OpenCode Pod
        ├── opencode serve :4096
        ├── oh-my-opencode plugin
        ├── kubedock (optional) ← Docker API → K8s Pod translation
        └── MCP config (remote + laptop)
```

### Multi-user mode (StatefulSet + oauth2-proxy)

In multi-user mode (`mode: "multi"`), each user gets:

- **Dedicated StatefulSet** — one replica per user, fully isolated
- **Per-user ConfigMap** — `opencode.jsonc` and `oh-my-opencode.jsonc`
- **Per-user Secret** — server password and API keys
- **Per-user PVCs** — data and workspace storage
- **Per-user Ingress** — Tailscale ingress with user-specific hostname
- **Shared oauth2-proxy** — single OIDC proxy in front of all users

```
Your Tailnet
├── Laptop (Tailscale node)
│   └── Local MCP servers (Playwright, browser tools, etc.)
│
└── ARM64 k8s Cluster
    ├── Tailscale Operator
    │   ├── Ingress proxies  ← one per user (e.g., opencode-alice-<ns>.ts.net)
    │   └── Egress proxies   ← routes cluster traffic to laptop MCPs
    │
    ├── oauth2-proxy (shared)
    │   └── OIDC authentication for all users
    ├── Nginx router
    │   └── Host-based routing to per-user services
    │
    ├── alice StatefulSet
    │   ├── ConfigMap (opencode.jsonc)
    │   ├── Secret (alice API keys)
    │   ├── kubedock Deployment (optional)
    │   └── PVCs (data + workspace)
    │
    └── bob StatefulSet
        ├── ConfigMap (opencode.jsonc)
        ├── Secret (bob API keys)
        ├── kubedock Deployment (optional)
        └── PVCs (data + workspace)
```

## Multi-User Setup

### Enable multi-user mode

```yaml
mode: "multi"
users:
  - name: alice
    password: "secure-password"
    workspaceSize: 20Gi
    providers:
      anthropic:
        enabled: true
        apiKey: "sk-ant-..."
```

Each user gets their own URL:
- Alice: `https://opencode-alice-<namespace>.<tailnet>.ts.net`
- Bob: `https://opencode-bob-<namespace>.<tailnet>.ts.net`

### oauth2-proxy OIDC configuration

See [docs/maintenance.md#oidc-authentication-multi-user-mode](docs/maintenance.md#oidc-authentication-multi-user-mode) for full setup instructions.

Quick config reference:

```yaml
auth:
  oidc:
    enabled: true
    provider: "https://accounts.google.com"        # OIDC issuer URL
    clientId: "your-client-id"
    clientSecret: "your-client-secret"
    cookieSecret: "base64-32-byte-secret"
    emailDomain: "example.com"                     # Or "*" for any domain
    hostname: "ok8s-auth"                          # Single auth hostname (default)
    cookieDomain: ".<tailnet>.ts.net"              # REQUIRED — your tailnet domain with leading dot
    ingress:
      enabled: true                                # Creates Tailscale ingress for auth
```

> **One callback URL for ALL users:** `https://ok8s-auth-<namespace>.<tailnet>.ts.net/oauth2/callback`

### Maintenance operations

Add user:

```bash
./scripts/add-user.sh alice | tee -a my-values.yaml
helm upgrade opencode ./chart -f my-values.yaml
```

Remove user:

```bash
./scripts/remove-user.sh my-values.yaml alice default
helm upgrade opencode ./chart -f my-values.yaml
```

Backup and restore:

```bash
./scripts/backup-user.sh opencode alice default ./alice-workspace.tar.gz
./scripts/restore-user.sh opencode alice default ./alice-workspace.tar.gz
```

List users:

```bash
./scripts/list-users.sh opencode default
```

## Kubedock: Test Containers as Kubernetes Pods

Kubedock translates the Docker API into Kubernetes Pod creation. This prevents
OOM kills from running Docker-in-Docker (DinD) sidecars by spawning test
containers as native K8s Pods instead.

### Enable Kubedock

```yaml
kubedock:
  enabled: true
  # Optional: extra arguments for kubedock
  extraArgs: []
  # - "--port-forward"
```

When enabled, the chart:
- **Single-user mode**: Deploys a shared kubedock instance
- **Multi-user mode**: Deploys per-user kubedock instances with NetworkPolicy
  isolation between users
- Injects `DOCKER_HOST`, `TESTCONTAINERS_RYUK_DISABLED`, and
  `TESTCONTAINERS_CHECKS_DISABLE` env vars into agent pods

### Testcontainers Configuration

Set these environment variables in your test framework (auto-injected when
kubedock is enabled):

```yaml
DOCKER_HOST: "tcp://<kubedock-service>:2475"
TESTCONTAINERS_RYUK_DISABLED: "true"
TESTCONTAINERS_CHECKS_DISABLE: "true"
```

## Secret Management

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

## Roadmap

- [ ] **Dynamic user management** — Replace static values.yaml user definitions
  with an external identity provider (Authentik, Keycloak, Authelia) or custom
  CRD + operator for Kubernetes-native user lifecycle management. See
  [docs/maintenance.md](docs/maintenance.md) for current procedures.

---

# AI Instructions

## Context

This is a Helm chart for deploying OpenCode with Oh-My-OpenCode on Kubernetes.
The chart is designed to be open-source and reusable, with all user-specific
decisions templatized through `values.yaml`.

## Key Architecture Facts

- OpenCode runs as `opencode serve` — a headless HTTP server on port 4096
- Single replica per user (stateful — stores sessions and workspace on disk)
- Tailscale handles all external connectivity (no public ingress)
- Two Tailscale directions: **ingress** (laptop→cluster UI) and **egress**
  (cluster→laptop MCPs)
- Official Docker image: `ghcr.io/anomalyco/opencode` (multi-arch, includes
  ARM64)
- Oh-My-OpenCode is configured via `oh-my-opencode.jsonc` in a ConfigMap
- MCP servers are configured in `opencode.jsonc` — both remote (URLs) and laptop
  (Tailscale egress)
- Multi-user mode uses per-user StatefulSets with shared oauth2-proxy + auth router for OIDC
- Auth router validates OIDC sessions and enforces email→user binding — users can only access their own workspace
- Each user must have an `email` field matching their Google Workspace email
- `cookieDomain` must be set to the tailnet domain (e.g. `.lynx-beta.ts.net`)

## File Structure

```
chart/
├── Chart.yaml
├── values.yaml                    # All configurable values with descriptions
├── templates/
│   ├── _helpers.tpl               # Named templates for config generation
│   ├── namespace.yaml
│   ├── deployment.yaml            # Single-user Deployment
│   ├── statefulset.yaml           # Multi-user per-user StatefulSets
│   ├── service.yaml               # ClusterIP :4096
│   ├── configmap.yaml             # Single-user: opencode.jsonc + oh-my-opencode.jsonc
│   ├── user-configmap.yaml        # Multi-user: per-user configs
│   ├── user-secret.yaml           # Multi-user: per-user secrets
│   ├── user-ingress.yaml          # Multi-user: per-user Tailscale ingress
│   ├── resourcequota.yaml         # Multi-user: namespace resource quota
│   ├── identity-configmap.yaml    # Identity files (AGENTS.md, etc.)
│   ├── pvc.yaml                   # Single-user: Data + workspace PVCs
│   ├── secrets/
│   │   ├── plain-secrets.yaml     # Standard K8s secrets
│   │   ├── sealed-secrets.yaml    # Bitnami sealed-secrets (conditional)
│   │   └── external-secrets.yaml  # external-secrets-operator (conditional)
│   ├── ingress/
│   │   └── tailscale-ingress.yaml # Tailscale Ingress (conditional)
│   ├── tailscale/
│   │   ├── proxyclass.yaml        # ProxyClass for route acceptance
│   │   └── egress-services.yaml   # ExternalName Services per laptop MCP
│   ├── kubedock/
│   │   ├── serviceaccount.yaml    # Kubedock service account
│   │   ├── role.yaml              # RBAC role for pod creation
│   │   ├── rolebinding.yaml       # Role binding
│   │   ├── deployment.yaml        # Single-user kubedock Deployment
│   │   ├── service.yaml           # Single-user kubedock Service
│   │   ├── user-deployment.yaml   # Multi-user per-user kubedock Deployments
│   │   ├── user-service.yaml      # Multi-user per-user kubedock Services
│   │   └── networkpolicy.yaml     # NetworkPolicy for user isolation
│   ├── oauth2-proxy/
│   │   ├── configmap.yaml         # oauth2-proxy configuration
│   │   ├── deployment.yaml        # oauth2-proxy deployment
│   │   ├── service.yaml           # oauth2-proxy service
│   │   ├── router-configmap.yaml  # host-based router config
│   │   ├── router-deployment.yaml # host-based router deployment
│   │   └── router-service.yaml    # host-based router service
│   └── tests/
│       └── connection-test.yaml   # helm test
```

## How to Help

When asked to modify this chart:

1. **Always edit `values.yaml` first** if adding new configuration — add the
   value with a `# --` comment prefix (for helm-docs)
2. **Add the template** in the appropriate `templates/` subdirectory
3. **Use conditional rendering** — every optional feature must be gated by a
   values flag
4. **Follow existing patterns** — look at how `ingress.enabled` or
   `secrets.backend` gates work
5. **Never hardcode** — all user-specific values (API keys, IPs, hostnames,
   sizes) must be in values.yaml
6. **Test with `helm template`** — verify rendered output before claiming
   completion

When asked to deploy or configure:

1. Start from `examples/values-minimal.yaml` and add only what's needed
2. **Always use `-n <namespace>`** — the chart rejects installation into `default`
   ```bash
   helm install ok8s ./chart -n opencode --create-namespace -f my-values.yaml
   ```
3. Tailscale operator must be installed separately — this chart does not install
   it
4. For laptop MCPs: user needs to provide their laptop's Tailscale IP or
   MagicDNS name
5. Secret backend choice depends on their GitOps setup — `plain` for personal,
   `sealed`/`external` for teams

When asked to set up OIDC for multi-user mode:

1. There is ONE shared OAuth client for ALL users — do NOT create per-user OAuth clients
2. Register exactly ONE callback URL on that OAuth client:
   `https://ok8s-auth-<namespace>.<tailnet>.ts.net/oauth2/callback`
   (e.g. `https://ok8s-auth-opencode.lynx-beta.ts.net/oauth2/callback`)
3. Generate a cookie secret: `python3 -c "import base64,os; print(base64.b64encode(os.urandom(32)).decode())"`
4. Set `auth.oidc.cookieDomain` to the tailnet domain with a leading dot (e.g. `.lynx-beta.ts.net`)
5. The auth Tailscale ingress is created automatically when `auth.oidc.enabled: true` in multi-user mode
6. See [docs/maintenance.md#oidc-authentication-multi-user-mode](docs/maintenance.md#oidc-authentication-multi-user-mode) for full provider setup steps

## Common Tasks

### Add a new LLM provider

1. Add to `values.yaml` under `providers.*` with `enabled` and `apiKey` fields
2. Add env var block in `templates/deployment.yaml` (follow existing pattern)
3. Add secret key in `templates/secrets/plain-secrets.yaml`

### Add a new optional component

1. Add values under a new key in `values.yaml` with `# --` description
2. Create template with `{{- if .Values.yourKey.enabled }}` guard
3. Add to an example values file if it's a user-facing feature

### Change the OpenCode image

- User overrides via `image.repository` and `image.tag` in their values file
- Do not change defaults in `values.yaml` without discussing
