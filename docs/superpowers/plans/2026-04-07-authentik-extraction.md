# Authentik Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract Authentik components from the main OpenCode Helm chart into a standalone repository for independent deployment and maintenance.

**Architecture:** Create a new GitHub repository with a complete Helm chart structure, migrate Authentik-specific templates and values, establish CI/CD pipelines for independent publishing, and clean up the main repository.

**Tech Stack:** Helm 3, Kubernetes, GitHub Actions, OCI chart publishing, Tailscale operator.

---

### Task 1: Create New Repository Structure

**Files:**
- Create: `../k8s-opencode-authentik/.gitignore`
- Create: `../k8s-opencode-authentik/README.md`
- Create: `../k8s-opencode-authentik/AGENTS.md`
- Create: `../k8s-opencode-authentik/charts/authentik/Chart.yaml`
- Create: `../k8s-opencode-authentik/charts/authentik/values.yaml`
- Create: `../k8s-opencode-authentik/charts/authentik/templates/_helpers.tpl`

- [ ] **Step 1: Create new repository directory**

```bash
mkdir -p ../k8s-opencode-authentik
cd ../k8s-opencode-authentik
git init
```

- [ ] **Step 2: Add .gitignore**

```bash
cat > .gitignore << 'EOF'
# Helm
*.tgz
charts/*/charts/

# Kubernetes
*.yaml.tmp

# Python
__pycache__/
*.pyc

# OS
.DS_Store
Thumbs.db
EOF
```

- [ ] **Step 3: Add README.md**

```bash
cat > README.md << 'EOF'
# k8s-opencode-authentik

Standalone Helm chart for deploying Authentik Identity-Aware Proxy in Kubernetes clusters.

## Installation

```bash
helm install authentik ./charts/authentik \
  -n authentik --create-namespace \
  -f values.yaml
```

## Configuration

See `charts/authentik/values.yaml` for all configuration options.

## Requirements

- Kubernetes 1.19+
- Helm 3.0+
- Tailscale operator (for ingress)
EOF
```

- [ ] **Step 4: Add AGENTS.md**

```bash
cat > AGENTS.md << 'EOF'
# AGENTS.md — Authentik Chart Maintenance

Guidelines for AI agents maintaining the k8s-opencode-authentik repository.

## Worktrees for ALL Edits

Every code edit MUST use a worktree. No exceptions.

```bash
cd /path/to/repo
git worktree add ../authentik-<task> -b agent/<task>
cd ../authentik-<task>
# Make changes, commit frequently
git add . && git commit -m "feat: description"
# Integrate back
cd /path/to/repo
git merge agent/<task>
git worktree remove ../authentik-<task>
```

## Testing

- Run `helm template test ./charts/authentik` for validation
- Test locally with `helm install --dry-run`
- Verify in cluster with `helm test`

## Version Sync

When tagging vX.Y.Z, CI updates Chart.yaml version and appVersion automatically.
EOF
```

- [ ] **Step 5: Create Chart.yaml**

```bash
cat > charts/authentik/Chart.yaml << 'EOF'
apiVersion: v2
name: authentik
description: Authentik Identity-Aware Proxy
type: application
version: 0.1.0
appVersion: "2026.2.2"

keywords:
  - authentication
  - identity
  - oidc
  - saml

maintainers:
  - name: Timothy Lin
    email: tim@timlin.dev
EOF
```

- [ ] **Step 6: Create values.yaml**

```bash
cat > charts/authentik/values.yaml << 'EOF'
# Authentik domain (must match Tailscale ingress)
domain: "authentik.example.ts.net"

# CRITICAL: Change this to a secure 32+ character random string
secretKey: "CHANGE-ME-TO-SECURE-32-CHAR-STRING"

# Proxy configuration for cookie domain
proxy:
  enabled: true
  hostname: "authentik.example.ts.net"
  cookieDomain: ".example.ts.net"

# Authentik server image
image:
  repository: ghcr.io/goauthentik/server
  tag: "2026.2.2"
  pullPolicy: IfNotPresent

replicaCount: 1

resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: "2"
    memory: 2Gi

# PostgreSQL configuration
postgres:
  enabled: true
  image:
    repository: postgres
    tag: "15"
  database: "authentik"
  username: "authentik"
  password: "authentikpassword"
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  persistence:
    enabled: true
    size: 10Gi

# Redis configuration
redis:
  enabled: true
  image:
    repository: redis
    tag: "7-alpine"
  password: "redispassword"
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 500m
      memory: 512Mi

# Tailscale ingress
ingress:
  enabled: true
  hostname: "authentik"
  tailscale:
    proxyTag: "tag:k8s"

# OIDC configuration (for external apps like OpenCode)
oidc:
  enabled: false
  issuerUrl: ""
  clientId: ""
  clientSecret: ""
  scopes: ["openid", "email", "profile"]
EOF
```

- [ ] **Step 7: Create _helpers.tpl**

```bash
cat > charts/authentik/templates/_helpers.tpl << 'EOF'
{{/*
Expand the name of the chart.
*/}}
{{- define "authentik.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "authentik.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "authentik.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "authentik.labels" -}}
helm.sh/chart: {{ include "authentik.chart" . }}
{{ include "authentik.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "authentik.selectorLabels" -}}
app.kubernetes.io/name: {{ include "authentik.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "authentik.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "authentik.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
EOF
```

- [ ] **Step 8: Initialize git and commit**

```bash
cd ../k8s-opencode-authentik
git add .
git commit -m "feat: initialize Authentik Helm chart repository"
```

### Task 2: Migrate Authentik Templates

**Files:**
- Copy: `chart/templates/authentik/deployment.yaml` → `charts/authentik/templates/deployment.yaml`
- Copy: `chart/templates/authentik/service.yaml` → `charts/authentik/templates/service.yaml`
- Copy: `chart/templates/authentik/configmap.yaml` → `charts/authentik/templates/configmap.yaml`
- Copy: `chart/templates/authentik/secret.yaml` → `charts/authentik/templates/secret.yaml`
- Copy: `chart/templates/authentik/postgres-deployment.yaml` → `charts/authentik/templates/postgres-deployment.yaml`
- Copy: `chart/templates/authentik/postgres-service.yaml` → `charts/authentik/templates/postgres-service.yaml`
- Copy: `chart/templates/authentik/postgres-pvc.yaml` → `charts/authentik/templates/postgres-pvc.yaml`
- Copy: `chart/templates/authentik/redis-deployment.yaml` → `charts/authentik/templates/redis-deployment.yaml`
- Copy: `chart/templates/authentik/redis-service.yaml` → `charts/authentik/templates/redis-service.yaml`
- Copy: `chart/templates/authentik/ingress.yaml` → `charts/authentik/templates/ingress.yaml`
- Copy: `chart/templates/authentik/networkpolicy.yaml` → `charts/authentik/templates/networkpolicy.yaml`

- [ ] **Step 1: Copy all Authentik templates**

```bash
cp -r chart/templates/authentik/* charts/authentik/templates/
```

- [ ] **Step 2: Update template references**

In all copied files, replace:
- `{{ include "ok8s.fullname" . }}` → `{{ include "authentik.fullname" . }}`
- `{{ include "ok8s.authentik.fullname" . }}` → `{{ include "authentik.fullname" . }}`
- `{{ .Values.authentik.` → `{{ .Values.`

- [ ] **Step 3: Update namespace references**

In all templates, ensure namespace is set to `{{ .Release.Namespace }}`

- [ ] **Step 4: Update ConfigMap for standalone values**

In `configmap.yaml`, update domain reference:
```yaml
domain: {{ .Values.domain }}
```

- [ ] **Step 5: Update Secret template**

In `secret.yaml`, ensure secret key uses standalone value:
```yaml
AUTHENTIK_SECRET_KEY: {{ .Values.secretKey | b64enc }}
```

- [ ] **Step 6: Update ingress template**

In `ingress.yaml`, update hostname and cookie domain:
```yaml
spec:
  tls:
  - hosts:
    - {{ .Values.ingress.hostname }}
  rules:
  - host: {{ .Values.ingress.hostname }}
```

- [ ] **Step 7: Commit migrated templates**

```bash
git add charts/authentik/templates/
git commit -m "feat: migrate and adapt Authentik templates for standalone chart"
```

### Task 3: Create CI/CD Pipelines

**Files:**
- Create: `.github/workflows/publish-chart.yml`
- Create: `.github/workflows/test-chart.yml`
- Create: `scripts/version-sync.sh`

- [ ] **Step 1: Create publish-chart.yml**

```bash
mkdir -p .github/workflows
cat > .github/workflows/publish-chart.yml << 'EOF'
name: Publish Chart

on:
  push:
    tags:
      - 'v*'

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Install Helm
      run: |
        curl https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz -o helm.tar.gz
        tar -zxvf helm.tar.gz
        sudo mv linux-amd64/helm /usr/local/bin/helm

    - name: Update Chart Version
      run: |
        VERSION=${GITHUB_REF#refs/tags/v}
        yq -i ".version = \"$VERSION\"" charts/authentik/Chart.yaml
        yq -i ".appVersion = \"$VERSION\"" charts/authentik/Chart.yaml

    - name: Package Chart
      run: |
        helm package charts/authentik

    - name: Login to GHCR
      run: |
        echo ${{ secrets.GITHUB_TOKEN }} | helm registry login ghcr.io -u ${{ github.actor }} --password-stdin

    - name: Push Chart
      run: |
        helm push authentik-*.tgz oci://ghcr.io/timothyclin/k8s-opencode-authentik/chart
EOF
```

- [ ] **Step 2: Create test-chart.yml**

```bash
cat > .github/workflows/test-chart.yml << 'EOF'
name: Test Chart

on:
  pull_request:
    paths:
      - 'charts/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4

    - name: Install Helm
      run: |
        curl https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz -o helm.tar.gz
        tar -zxvf helm.tar.gz
        sudo mv linux-amd64/helm /usr/local/bin/helm

    - name: Install chart-testing
      run: |
        curl -Lo ct.tar.gz https://github.com/helm/chart-testing/releases/download/v3.8.0/chart-testing_3.8.0_linux_amd64.tar.gz
        tar -xzf ct.tar.gz
        sudo mv ct /usr/local/bin/ct

    - name: Run chart-testing
      run: ct lint --charts charts/authentik
EOF
```

- [ ] **Step 3: Create version-sync.sh**

```bash
mkdir -p scripts
cat > scripts/version-sync.sh << 'EOF'
#!/bin/bash
set -e

# Get version from git tag
VERSION=${1:-$(git describe --tags --abbrev=0)}

# Update Chart.yaml
yq -i ".version = \"$VERSION\"" charts/authentik/Chart.yaml
yq -i ".appVersion = \"$VERSION\"" charts/authentik/Chart.yaml

echo "Updated chart version to $VERSION"
EOF
chmod +x scripts/version-sync.sh
```

- [ ] **Step 4: Commit CI/CD setup**

```bash
git add .github/ scripts/
git commit -m "feat: add CI/CD pipelines and version sync script"
```

### Task 4: Create Documentation

**Files:**
- Create: `docs/README.md`
- Create: `docs/opencode-integration.md`

- [ ] **Step 1: Create docs/README.md**

```bash
mkdir -p docs
cat > docs/README.md << 'EOF'
# Authentik Deployment Guide

## Prerequisites

- Kubernetes cluster with Tailscale operator installed
- Helm 3.0+
- Cluster admin access

## Installation

### Quick Start

```bash
# Clone the repository
git clone https://github.com/timothyclin/k8s-opencode-authentik.git
cd k8s-opencode-authentik

# Install with default values
helm install authentik ./charts/authentik \
  -n authentik --create-namespace \
  -f charts/authentik/values.yaml
```

### Configuration

Edit `charts/authentik/values.yaml`:

- Set `domain` to your desired Tailscale hostname
- Change `secretKey` to a secure 32+ character string
- Configure `proxy.cookieDomain` for your tailnet

### Accessing Authentik

After installation, access the admin interface at:
`https://<domain>`

Default credentials:
- Username: admin
- Password: admin

## Troubleshooting

### Bootstrap Issues
If the admin user isn't created:
1. Check pod logs: `kubectl logs -n authentik deployment/authentik`
2. Verify `domain` and `secretKey` are set correctly
3. Ensure PostgreSQL and Redis are running

### Ingress Issues
If you can't access the UI:
1. Check Tailscale operator status
2. Verify domain configuration
3. Check ingress resource: `kubectl get ingress -n authentik`
EOF
```

- [ ] **Step 2: Create opencode-integration.md**

```bash
cat > docs/opencode-integration.md << 'EOF'
# OpenCode Integration

Configure Authentik as OIDC provider for OpenCode authentication.

## Setup Steps

1. **Deploy Authentik** (see main README)

2. **Create OIDC Application in Authentik**
   - Go to Authentik admin UI
   - Navigate to Applications → Create
   - Name: "OpenCode"
   - Provider: Create new OIDC provider
     - Client Type: Confidential
     - Scopes: openid, email, profile
     - Redirect URIs: `https://<opencode-domain>/oauth2/callback`

3. **Get Client Credentials**
   - Copy Client ID and Client Secret from the provider

4. **Configure OpenCode**
   In your OpenCode `values.yaml`:

   ```yaml
   auth:
     oidc:
       enabled: true
       provider: "oidc"
       clientId: "<client-id-from-authentik>"
       clientSecret: "<client-secret-from-authentik>"
       issuerUrl: "https://<authentik-domain>/application/o/opencode/"
       cookieSecret: "<generate-32-char-secret>"
       emailDomain: "*"
       hostname: "<opencode-hostname>"
   ```

5. **Deploy OpenCode**
   ```bash
   helm upgrade opencode ./chart -f values.yaml
   ```

## User Flow

1. User visits OpenCode URL
2. Redirected to Authentik login
3. After authentication, redirected back to OpenCode
4. User has access based on Authentik permissions
EOF
```

- [ ] **Step 3: Commit documentation**

```bash
git add docs/
git commit -m "docs: add deployment guide and OpenCode integration docs"
```

### Task 5: Clean Up Main Repository

**Files:**
- Delete: `chart/templates/authentik/`
- Modify: `chart/values.yaml` (remove authentik section)
- Modify: `docs/` (update references)

- [ ] **Step 1: Remove Authentik templates**

```bash
rm -rf chart/templates/authentik/
```

- [ ] **Step 2: Remove Authentik values**

Edit `chart/values.yaml` to remove the entire `authentik:` section (lines 306-367)

- [ ] **Step 3: Update documentation**

Remove Authentik references from docs and update README to mention separate chart

- [ ] **Step 4: Commit cleanup**

```bash
git add .
git commit -m "feat: remove Authentik components from main chart"
```

### Task 6: Test Standalone Installation

**Files:**
- Test: `charts/authentik/`

- [ ] **Step 1: Validate chart structure**

```bash
cd charts/authentik
helm template test . --dry-run
```

- [ ] **Step 2: Test template rendering**

```bash
helm template authentik . -f values.yaml > /tmp/authentik-manifests.yaml
kubectl apply --dry-run=client -f /tmp/authentik-manifests.yaml
```

- [ ] **Step 3: Verify no main chart references**

Search for "ok8s" in templates - should find none

- [ ] **Step 4: Test with sample values**

Create test-values.yaml with minimal config and verify rendering

- [ ] **Step 5: Commit test validation**

```bash
git add .
git commit -m "test: validate standalone chart functionality"
```

### Task 7: Push to GitHub

**Files:**
- Push: Repository to GitHub

- [ ] **Step 1: Create GitHub repository**

Go to GitHub and create public repository `k8s-opencode-authentik`

- [ ] **Step 2: Add remote**

```bash
git remote add origin https://github.com/timothyclin/k8s-opencode-authentik.git
```

- [ ] **Step 3: Push initial commit**

```bash
git push -u origin main
```

- [ ] **Step 4: Verify repository**

Check GitHub repository has all files and CI/CD workflows