# AGENTS.md — chart/ (Helm Chart)

**Chart name:** `ok8s` | **Version:** 0.1.0 | **AppVersion:** 1.0.0

## Overview

Helm chart deploying OpenCode on Kubernetes. Two modes: `single` (one Deployment) and `multi` (per-user StatefulSets with OIDC auth, kubedock, and Tailscale ingress).

## Template Map

| Template | Kind(s) | Mode | Gate |
|---|---|---|---|
| `deployment.yaml` | Deployment | single | `mode == "single"` |
| `statefulset.yaml` | StatefulSet ×N | multi | `mode == "multi"`, loops `users[]` |
| `service.yaml` | Service | both | always |
| `pvc.yaml` | PVC (data + workspace) | single | `mode == "single"` |
| `configmap.yaml` | ConfigMap | single | `mode == "single"` |
| `user-configmap.yaml` | ConfigMap ×N | multi | loops `users[]` |
| `user-secret.yaml` | Secret ×N | multi | `secrets.backend == "plain"` |
| `user-ingress.yaml` | Ingress ×N | multi | `ingress.enabled` |
| `identity-configmap.yaml` | ConfigMap | both | `identity.enabled` |
| `namespace.yaml` | Namespace | both | `createNamespace` |
| `resourcequota.yaml` | ResourceQuota | multi | `resourceQuota.enabled` |
| **secrets/** | | | |
| `plain-secrets.yaml` | Secret | both | `secrets.backend == "plain"` |
| `sealed-secrets.yaml` | SealedSecret | both | `secrets.backend == "sealed"` |
| `external-secrets.yaml` | ExternalSecret | both | `secrets.backend == "external"` |
| **oauth2-proxy/** | | | |
| `deployment.yaml` | Deployment | multi | `auth.oidc.enabled` |
| `service.yaml` | Service | multi | `auth.oidc.enabled` |
| `configmap.yaml` | ConfigMap | multi | `auth.oidc.enabled` |
| `auth-router.yaml` | Deployment | multi | `auth.oidc.enabled` |
| `auth-router-service.yaml` | Service | multi | `auth.oidc.enabled` |
| `auth-ingress.yaml` | Ingress | multi | `auth.oidc.enabled` |
| `email-map.yaml` | ConfigMap | multi | `auth.oidc.enabled` |
| **kubedock/** | | | |
| `user-deployment.yaml` | Deployment ×N | multi | `kubedock.enabled` |
| `user-service.yaml` | Service ×N | multi | `kubedock.enabled` |
| `role.yaml` | Role | both | `kubedock.enabled` |
| `rolebinding.yaml` | RoleBinding | both | `kubedock.enabled` |
| `serviceaccount.yaml` | ServiceAccount | both | `kubedock.enabled` |
| `deployment.yaml` | Deployment | single | `kubedock.enabled` |
| `service.yaml` | Service | single | `kubedock.enabled` |
| `networkpolicy.yaml` | NetworkPolicy | multi | `kubedock.enabled` |
| **tailscale/** | | | |
| `proxyclass.yaml` | ProxyClass | both | `mcp.laptopServers` present |
| `egress-services.yaml` | Service (ExternalName) ×N | both | `mcp.laptopServers` entries |
| `cleanup-job.yaml` | Job (pre-delete hook) | both | `ingress.enabled` |
| `cleanup-rbac.yaml` | SA + Role + RoleBinding | both | `ingress.enabled` |
| **ingress/** | | | |
| `tailscale-ingress.yaml` | Ingress | both | `ingress.enabled` |
| **tests/** | | | |
| `connection-test.yaml` | Pod (helm test) | both | always |

## Helper Functions (`_helpers.tpl`)

| Helper | Purpose | Gotcha |
|---|---|---|
| `ok8s.fullname` | Canonical resource name prefix | **FAILS if namespace is `default`** |
| `ok8s.name` | Short chart name | — |
| `ok8s.labels` | Common labels block | — |
| `ok8s.selectorLabels` | Selector labels | — |
| `ok8s.opencodeConfig` | Generates `opencode.jsonc` (single-user) | Merges `mcp.remote` + `mcp.laptopServers` with egress URLs |
| `ok8s.opencodeUserConfig` | Generates `opencode.jsonc` (per-user) | Egress URLs: `<Release.Name>-egress-<user>-<name>:<port>` |
| `ok8s.omoConfig` | Generates `oh-my-opencode.jsonc` (single) | Only when `omo.enabled` |
| `ok8s.omoUserConfig` | Generates `oh-my-opencode.jsonc` (per-user) | Merges `sharedSkills` + user skills |
| `ok8s.userConfigChecksum` | SHA256 of combined user config | **Changes to global `omo`/`sharedMcp`/`sharedSkills` roll ALL user pods** |
| `ok8s.userCount` | `len .Values.users` | — |
| `ok8s.userNameByIndex` | CSV of user names | Used by StatefulSet ordinal mapping |
| `ok8s.userList` | Joined user names | — |

## Values Structure (Key Paths)

### Required Values

| Path | When | Why |
|---|---|---|
| `-n <namespace>` (not `default`) | Always | `ok8s.fullname` refuses `default` ns |
| `serverPassword` | single mode | Protects opencode endpoint |
| `users[].name` | multi mode | Resource naming identity |
| `users[].email` | multi + OIDC | Email→user mapping for auth |
| `auth.oidc.clientId/clientSecret` | OIDC enabled | OAuth2 provider credentials |
| `auth.oidc.cookieSecret` | OIDC enabled | Session encryption |
| `auth.oidc.cookieDomain` | OIDC enabled | Must be tailnet domain with leading `.` |

### Feature Toggles

| Value | Default | Enables |
|---|---|---|
| `mode` | `"single"` | `"multi"` → per-user StatefulSets + auth |
| `auth.oidc.enabled` | `false` | oauth2-proxy + auth-router |
| `kubedock.enabled` | `false` | kubedock sidecars + RBAC |
| `ingress.enabled` | `false` | Tailscale ingress + cleanup hook |
| `identity.enabled` | `false` | Identity files ConfigMap + init container |
| `omo.enabled` | `false` | Oh-My-OpenCode config injection |
| `plugins.enabled` | `false` | Plugin entries in opencode.jsonc |
| `secrets.backend` | `"plain"` | `"sealed"` / `"external"` for GitOps |

## Resource Naming Patterns

All names derive from `ok8s.fullname` (= `<release>-ok8s` or `fullnameOverride`).

| Resource | Pattern |
|---|---|
| Single Deployment | `<fullname>` |
| Single PVCs | `<fullname>-data`, `<fullname>-workspace` |
| Single ConfigMap | `<fullname>-config` |
| Per-user StatefulSet | `<fullname>-<user.name>` |
| Per-user ConfigMap | `<fullname>-user-<user.name>-config` |
| Per-user Secret | `<fullname>-user-<user.name>-secrets` |
| Per-user Service | `<fullname>-user-<user.name>` |
| Per-user kubedock | `<fullname>-kubedock-<user.name>` |
| Egress Service | `<release>-egress-<name>` (global) or `<release>-egress-<user>-<name>` (per-user) |
| Tailscale ProxyClass | `<fullname>-egress` |
| Cleanup Job | `<fullname>-ts-cleanup` |

## Non-Obvious Behaviors

1. **Namespace guard** — `ok8s.fullname` calls `fail` if `Release.Namespace == "default"`. Always use `-n <namespace>`.

2. **Config checksum rolling** — `ok8s.userConfigChecksum` includes `sharedMcp`, `sharedSkills`, and `omo` config. Changing ANY global shared value will restart ALL user pods via annotation diff.

3. **Egress service naming** — `ok8s.opencodeUserConfig` generates URLs like `http://<Release.Name>-egress-<user>-<name>:<port>`. The `egress-services.yaml` template must create matching ExternalName Services. Verify alignment when adding per-user `mcp.laptopServers`.

4. **Tailscale = operator, not sidecar** — Chart creates CRDs (Ingress, ProxyClass) that the Tailscale Operator manages. Operator must be pre-installed in its own namespace (`tailscaleOperator.namespace`).

5. **Hot-reload** — Auth-router watches `EMAIL_MAP_PATH` via fsnotify. ConfigMap updates propagate without pod restart. No deployment annotation hash needed (intentional).

6. **Identity init container** — When `identity.enabled`, copies files (AGENTS.md, etc.) to `/workspace` only if they don't already exist. Preserves user edits across restarts.

7. **nodeSelector default** — `kubernetes.io/arch: arm64`. Override for amd64 clusters.

8. **Secret backend migration** — Switching `secrets.backend` after install may leave orphaned resources. Migrate manually.

## Anti-Patterns

- **Never** rename `ok8s.*` helpers without updating ALL templates
- **Never** use `secrets.backend: plain` in GitOps — credentials in plaintext
- **Never** reorder `users[]` without understanding PVC ordinal mapping
- **Never** set `cookieDomain` without leading `.` — breaks OIDC cross-subdomain cookies
- **Never** commit real API keys in values files or examples
- **Never** enable OIDC without setting all 4 required values (clientId, clientSecret, cookieSecret, cookieDomain)
