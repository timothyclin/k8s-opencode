# AGENTS.md — k8s-opencode Knowledge Base

**Generated:** 2026-04-01 | **Commit:** 6832d82 | **Branch:** main

## Overview

Helm chart + container images for deploying [OpenCode](https://github.com/anomalyco/opencode) on Kubernetes with Tailscale connectivity. Supports single-user and multi-user modes with per-user isolation, OIDC auth, kubedock, and MCP laptop server egress.

## Structure

```
k8s-opencode/
├── chart/                  # Helm chart (see chart/AGENTS.md for deep template docs)
│   ├── Chart.yaml          # name: ok8s, version: 0.1.0
│   ├── values.yaml         # ALL config — single source of truth
│   └── templates/          # 19 template files across 6 subdirs
├── images/
│   ├── router/             # Go auth-router (main.go + Dockerfile)
│   └── workspace/          # OpenCode workspace image (Dockerfile only)
├── scripts/                # User lifecycle: add/remove/list/backup/restore
├── docs/                   # Architecture, maintenance, customization, multi-user
├── examples/               # values-minimal.yaml, values-full.yaml, values-multi.yaml
└── .github/workflows/      # publish-images.yml, publish-chart.yml (tag-driven)
```

## Where to Look

| Task | Location | Notes |
|---|---|---|
| Change chart defaults | `chart/values.yaml` | Every value documented inline |
| Add/modify K8s resources | `chart/templates/` | See `chart/AGENTS.md` for template map |
| Change helper functions | `chart/templates/_helpers.tpl` | `ok8s.*` helpers used everywhere |
| Modify auth router logic | `images/router/main.go` | Single-file Go app, rebuild image after |
| Modify workspace image | `images/workspace/Dockerfile` | Debian-based, copies opencode from upstream |
| Add/remove users (ops) | `scripts/add-user.sh` etc. | Manipulate values.yaml + helm upgrade |
| Example configs | `examples/values-*.yaml` | Minimal, full, and multi-user samples |
| CI/CD | `.github/workflows/` | Both trigger on `v*` tag push only |

## Naming Conventions (CRITICAL)

Four distinct names — do NOT confuse:

| Name | Where | Meaning |
|---|---|---|
| `opencode` | Namespace, hostPrefix, pod names, resource names | Product runtime identity |
| `ok8s` | Chart name, helper prefix (`ok8s.fullname`), Go module (`ok8s-auth-router`), release name | Chart/infra shorthand |
| `omo` | Config key (`omo:` in values), oh-my-opencode.jsonc | Oh-My-OpenCode agent config |
| `k8s-opencode` | Repo name, GHCR path (`ghcr.io/timothyclin/k8s-opencode/`) | Repository/registry identity |

## Images

### auth-router (`images/router/`)

Go reverse proxy for multi-user OIDC auth. Single file: `main.go`.

- **Listens:** `:8080`
- **Auth flow:** Calls oauth2-proxy `/oauth2/auth` with request cookies → extracts `X-Auth-Request-Email` → maps email→user via JSON file → proxies to `http://opencode-user-{user}:4096`
- **Env vars:** `OAUTH2_PROXY_URL`, `SIGNIN_URL`, `EMAIL_MAP_PATH`, `HOST_PREFIX`
- **Hot-reload:** Watches `EMAIL_MAP_PATH` via fsnotify, reloads without restart
- **No health endpoint** — probes must use TCP `:8080` or add `/health`

### opencode-workspace (`images/workspace/`)

Debian bookworm + opencode binary + dev tools (git, python3, ripgrep, jq, tmux, build-essential, etc.).

- **ARG:** `OPENCODE_IMAGE=ghcr.io/anomalyco/opencode:latest` (source of opencode binary)
- **Entrypoint:** `opencode` (serves on `:4096` when run by chart)
- **WORKDIR:** `/workspace`

### CI Build

Both images built multi-arch (`linux/amd64,linux/arm64`) via `docker buildx`. Tagging: `v1.2.3` → tags `1.2.3`, `1.2`, `1`. Published to GHCR on `v*` tag push.

## Git Workflow

### Branch-First (Mandatory)

Never commit to `main`. Always feature branch → PR.

```bash
git checkout -b feat/my-feature
git add -A && git commit -m "feat: add my feature"
git push -u origin feat/my-feature
```

### Worktrees

Use for parallel work (PR review, hotfixes). Name: `<repo>-<purpose>`. Clean up with `git worktree prune`.

### Commits

[Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `refactor:`, `chore:`. One logical change per commit.

## Code Changes

- **Minimal diffs** — change only what's needed, don't refactor unrelated code
- **Type safety** — never `as any`, `@ts-ignore`, `@ts-expect-error`
- **Error handling** — no empty `catch` blocks, log with context
- **Bugfixes** — fix minimally, never refactor while fixing

## Verification Commands

```bash
# Lint chart
helm lint chart/

# Render templates locally
helm template ok8s chart/ -f chart/values.yaml -n opencode

# Dry-run against cluster
helm install ok8s chart/ --dry-run --debug -n opencode

# Install from GHCR (production)
helm install ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart -n opencode --create-namespace \
  --version 0.1.0 -f values.yaml

# Upgrade from GHCR (production)
helm upgrade ok8s oci://ghcr.io/timothyclin/k8s-opencode/chart -n opencode -f values.yaml
```

> Local `./chart` paths are for development only. Published deployments use the OCI URI.

## Anti-Patterns

- **Never install in `default` namespace** — `ok8s.fullname` helper will fail
- **Never commit real credentials** in values files or examples
- **Never use `plain` secrets backend in GitOps** — use `sealed` or `external`
- **Never rename `ok8s.*` helpers** without updating all templates
- **Never reorder `users[]` array** without understanding StatefulSet ordinal→user mapping
- **Don't confuse naming** — `omo` ≠ `opencode` ≠ `ok8s` ≠ `k8s-opencode`

## Component Dependencies

```
auth-router ──→ oauth2-proxy ──→ OIDC provider (Google/GitHub/etc.)
     │                │
     │                └── cookieDomain + clientId/Secret required
     │
     └── per-user Services (opencode-user-{name}:4096)
              │
              └── StatefulSets (one per user in multi mode)
                       │
                       ├── kubedock sidecar (optional, per-user Deployment)
                       ├── Tailscale Ingress (operator-managed, not sidecar)
                       └── egress ExternalName Services (laptop MCP servers)
```

**Tailscale Operator** must be installed separately. Chart creates CRDs (Ingress, ProxyClass) that instruct the operator.

## Agent Delegation

| Task Type | Delegate To |
|---|---|
| Codebase search, pattern discovery | `explore` agent |
| External docs, library lookups | `librarian` agent |
| Architecture decisions, debugging | `oracle` agent |
| Frontend / UI / styling | `visual-engineering` category |
| Trivial single-file edits | `quick` category |
| Complex logic, algorithms | `ultrabrain` category |

- Fire explore/librarian **in parallel**, always `run_in_background=true`
- After delegating a search, **never repeat it manually** — wait for agent result
- Use `session_id` to continue with a sub-agent (preserves context, saves tokens)
- If delegation fails twice: **stop, report to user, triage together**

## PR Etiquette

- One concern per PR
- Summary of **what** and **why**
- Link related issues
- Self-review diff before requesting review

## File Organization

- Docs → `docs/`
- Examples → `examples/`
- Scripts → `scripts/`
- Keep repo root clean
