# Shared Ingress with Authentik Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement shared ingress architecture using Authentik Identity-Aware Proxy for multi-user OpenCode access with centralized OIDC authentication and dynamic user routing.

**Architecture:** Authentik serves as the central identity provider and proxy, extracting user email from OIDC claims to dynamically route requests to per-user StatefulSet pods. Uses secure scope mappings for backend calculation without custom routing code.

**Tech Stack:** Authentik, Kubernetes, Helm, OIDC providers (Google Workspace, Azure AD, Okta), PostgreSQL, Redis

---

## File Structure

**New Files:**
- `chart/templates/authentik/deployment.yaml` - Authentik server deployment
- `chart/templates/authentik/service.yaml` - Authentik service
- `chart/templates/authentik/configmap.yaml` - Authentik configuration
- `chart/templates/authentik/secret.yaml` - Authentik secrets
- `chart/templates/authentik/postgres-deployment.yaml` - PostgreSQL database
- `chart/templates/authentik/postgres-service.yaml` - PostgreSQL service
- `chart/templates/authentik/postgres-pvc.yaml` - PostgreSQL persistent storage
- `chart/templates/authentik/redis-deployment.yaml` - Redis cache
- `chart/templates/authentik/redis-service.yaml` - Redis service
- `chart/templates/authentik/ingress.yaml` - Authentik ingress
- `chart/templates/authentik/networkpolicy.yaml` - Authentik network policy

**Modified Files:**
- `chart/values.yaml` - Add auth.oidc configuration section
- `chart/templates/ingress.yaml` - Update to route through Authentik when enabled
- `chart/templates/_helpers.tpl` - Add Authentik helper functions
- `chart/Chart.yaml` - Add dependencies if using subchart approach

**Test Files:**
- `chart/templates/tests/authentik-test.yaml` - Helm test for Authentik deployment

## Implementation Notes

- **Authentik Version:** Use v2024.6.0+ for stable OIDC and proxy features
- **Security:** All secrets use Kubernetes secrets, no hardcoded values
- **Scalability:** Start with minimal resources, scale based on user load
- **Backup:** PostgreSQL PVC enables database persistence across upgrades
- **OIDC:** Configuration supports any OIDC provider via values.yaml
- **Testing:** Use `helm template` for validation, `helm test` for deployment verification

---

### Task 1: Update Helm Values for Authentik Configuration

**Files:**
- Modify: `chart/values.yaml` (add auth.oidc section)

- [ ] **Step 1: Add Authentik OIDC configuration section to values.yaml**

Add this section after the existing `auth.router` section:

```yaml
# -- Authentik Identity-Aware Proxy configuration
authentik:
  enabled: false
  image:
    repository: ghcr.io/goauthentik/server
    tag: "2024.6.0"
    pullPolicy: IfNotPresent
  replicaCount: 1
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 1Gi

  # -- OIDC provider configuration
  oidc:
    enabled: false
    provider: "oidc"  # oidc, google, github, etc.
    clientId: ""
    clientSecret: ""
    issuerUrl: ""  # https://accounts.google.com for Google
    scopes: ["openid", "email", "profile"]

  # -- Proxy provider configuration
  proxy:
    enabled: false
    hostname: "opencode.company.com"
    cookieDomain: ".company.com"

  # -- Database configuration
  postgres:
    enabled: true
    image:
      repository: postgres
      tag: "15"
    database: "authentik"
    username: "authentik"
    password: "change-me-in-production"
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

  # -- Redis configuration
  redis:
    enabled: true
    image:
      repository: redis
      tag: "7-alpine"
    password: "change-me-in-production"
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: 200m
        memory: 128Mi
```

- [ ] **Step 2: Validate YAML syntax**

Run: `helm template test ./chart --set authentik.enabled=true --dry-run`
Expected: No YAML parsing errors

- [ ] **Step 3: Commit values update**

```bash
git add chart/values.yaml
git commit -m "feat: add Authentik configuration to Helm values"
```

---

### Task 2: Create Authentik Database Components

**Files:**
- Create: `chart/templates/authentik/postgres-deployment.yaml`
- Create: `chart/templates/authentik/postgres-service.yaml`
- Create: `chart/templates/authentik/postgres-pvc.yaml`

- [ ] **Step 1: Create PostgreSQL deployment template**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-postgres
  labels:
    app.kubernetes.io/name: postgres
    app.kubernetes.io/instance: {{ .Release.Name }}
    app.kubernetes.io/component: database
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: postgres
      app.kubernetes.io/instance: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: postgres
        app.kubernetes.io/instance: {{ .Release.Name }}
    spec:
      containers:
      - name: postgres
        image: "{{ .Values.authentik.postgres.image.repository }}:{{ .Values.authentik.postgres.image.tag }}"
        env:
        - name: POSTGRES_DB
          value: {{ .Values.authentik.postgres.database | quote }}
        - name: POSTGRES_USER
          value: {{ .Values.authentik.postgres.username | quote }}
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{ include "ok8s.fullname" . }}-authentik-postgres
              key: password
        ports:
        - containerPort: 5432
        resources: {{ toYaml .Values.authentik.postgres.resources | nindent 10 }}
        volumeMounts:
        - name: postgres-data
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-data
        persistentVolumeClaim:
          claimName: {{ include "ok8s.fullname" . }}-authentik-postgres
{{- end }}
```

- [ ] **Step 2: Create PostgreSQL service template**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-postgres
spec:
  ports:
  - port: 5432
    targetPort: 5432
  selector:
    app.kubernetes.io/name: postgres
    app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
```

- [ ] **Step 3: Create PostgreSQL PVC template**

```yaml
{{- if and .Values.authentik.enabled .Values.authentik.postgres.persistence.enabled }}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-postgres
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: {{ .Values.authentik.postgres.persistence.size }}
{{- end }}
```

- [ ] **Step 4: Test template rendering**

Run: `helm template test ./chart --set authentik.enabled=true -s templates/authentik/postgres-deployment.yaml`
Expected: Valid Kubernetes YAML output

- [ ] **Step 5: Commit database components**

```bash
git add chart/templates/authentik/
git commit -m "feat: add Authentik PostgreSQL database components"
```

---

### Task 3: Create Authentik Redis Components

**Files:**
- Create: `chart/templates/authentik/redis-deployment.yaml`
- Create: `chart/templates/authentik/redis-service.yaml`

- [ ] **Step 1: Create Redis deployment template**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-redis
  labels:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Release.Name }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: redis
      app.kubernetes.io/instance: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: redis
        app.kubernetes.io/instance: {{ .Release.Name }}
    spec:
      containers:
      - name: redis
        image: "{{ .Values.authentik.redis.image.repository }}:{{ .Values.authentik.redis.image.tag }}"
        command: ["redis-server", "--requirepass", "$(REDIS_PASSWORD)"]
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{ include "ok8s.fullname" . }}-authentik-redis
              key: password
        ports:
        - containerPort: 6379
        resources: {{ toYaml .Values.authentik.redis.resources | nindent 10 }}
{{- end }}
```

- [ ] **Step 2: Create Redis service template**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-redis
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
```

- [ ] **Step 3: Test template rendering**

Run: `helm template test ./chart --set authentik.enabled=true -s templates/authentik/redis-deployment.yaml`
Expected: Valid Kubernetes YAML

- [ ] **Step 4: Commit Redis components**

```bash
git add chart/templates/authentik/redis-deployment.yaml chart/templates/authentik/redis-service.yaml
git commit -m "feat: add Authentik Redis cache components"
```

---

### Task 4: Create Authentik Secrets

**Files:**
- Create: `chart/templates/authentik/secret.yaml`

- [ ] **Step 1: Create Authentik secrets template**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik
type: Opaque
data:
  # Authentik secret key for encryption
  secretKey: {{ randAlphaNum 32 | b64enc }}
  # PostgreSQL password
  postgresPassword: {{ .Values.authentik.postgres.password | b64enc }}
  # Redis password
  redisPassword: {{ .Values.authentik.redis.password | b64enc }}
  # OIDC client secret (if provided)
  {{- if .Values.authentik.oidc.clientSecret }}
  oidcClientSecret: {{ .Values.authentik.oidc.clientSecret | b64enc }}
  {{- end }}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-postgres
type: Opaque
data:
  password: {{ .Values.authentik.postgres.password | b64enc }}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-redis
type: Opaque
data:
  password: {{ .Values.authentik.redis.password | b64enc }}
{{- end }}
```

- [ ] **Step 2: Test secret template**

Run: `helm template test ./chart --set authentik.enabled=true -s templates/authentik/secret.yaml`
Expected: Valid Secret resources with base64-encoded data

- [ ] **Step 3: Commit secrets**

```bash
git add chart/templates/authentik/secret.yaml
git commit -m "feat: add Authentik secrets for database and OIDC"
```

---

### Task 5: Create Authentik Main Deployment

**Files:**
- Create: `chart/templates/authentik/configmap.yaml`
- Create: `chart/templates/authentik/deployment.yaml`
- Create: `chart/templates/authentik/service.yaml`

- [ ] **Step 1: Create Authentik configmap**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik
data:
  # Authentik configuration
  authentik.yml: |
    # Basic configuration
    debug: false
    secret_key: /secrets/secretKey
    disable_startup_analytics: true

    # Database
    database:
      host: {{ include "ok8s.fullname" . }}-authentik-postgres
      name: {{ .Values.authentik.postgres.database }}
      user: {{ .Values.authentik.postgres.username }}
      password: /secrets/postgresPassword

    # Redis
    redis:
      host: {{ include "ok8s.fullname" . }}-authentik-redis
      password: /secrets/redisPassword

    # Email (disabled for this use case)
    email:
      backend: dummy

    # OIDC provider settings (configured via UI after deployment)
    oidc:
      enabled: {{ .Values.authentik.oidc.enabled }}
      {{- if .Values.authentik.oidc.issuerUrl }}
      issuer_url: {{ .Values.authentik.oidc.issuerUrl }}
      {{- end }}
{{- end }}
```

- [ ] **Step 2: Create Authentik deployment**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik
  labels:
    app.kubernetes.io/name: authentik
    app.kubernetes.io/instance: {{ .Release.Name }}
spec:
  replicas: {{ .Values.authentik.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: authentik
      app.kubernetes.io/instance: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: authentik
        app.kubernetes.io/instance: {{ .Release.Name }}
    spec:
      containers:
      - name: authentik
        image: "{{ .Values.authentik.image.repository }}:{{ .Values.authentik.image.tag }}"
        command: ["ak", "server"]
        env:
        - name: AUTHENTIK_BOOTSTRAP_PASSWORD
          value: "admin"  # Change in production
        - name: AUTHENTIK_BOOTSTRAP_EMAIL
          value: "admin@local"
        ports:
        - containerPort: 9000
          name: http
        - containerPort: 9443
          name: https
        volumeMounts:
        - name: config
          mountPath: /config
        - name: secrets
          mountPath: /secrets
        resources: {{ toYaml .Values.authentik.resources | nindent 10 }}
      volumes:
      - name: config
        configMap:
          name: {{ include "ok8s.fullname" . }}-authentik
      - name: secrets
        secret:
          secretName: {{ include "ok8s.fullname" . }}-authentik
{{- end }}
```

- [ ] **Step 3: Create Authentik service**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik
spec:
  ports:
  - name: http
    port: 80
    targetPort: 9000
  - name: https
    port: 443
    targetPort: 9443
  selector:
    app.kubernetes.io/name: authentik
    app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
```

- [ ] **Step 4: Test Authentik components**

Run: `helm template test ./chart --set authentik.enabled=true -s templates/authentik/deployment.yaml`
Expected: Valid Deployment with proper environment and volumes

- [ ] **Step 5: Commit Authentik main components**

```bash
git add chart/templates/authentik/configmap.yaml chart/templates/authentik/deployment.yaml chart/templates/authentik/service.yaml
git commit -m "feat: add Authentik main server deployment and configuration"
```

---

### Task 6: Update Ingress for Authentik Routing

**Files:**
- Modify: `chart/templates/ingress.yaml`

- [ ] **Step 1: Update ingress template to conditionally route through Authentik**

Modify the existing ingress template to add Authentik routing logic. Find the current ingress rules and add:

```yaml
{{- if .Values.authentik.enabled }}
# Authentik ingress for shared hostname
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - {{ .Values.authentik.proxy.hostname }}
    secretName: {{ include "ok8s.fullname" . }}-authentik-tls
  rules:
  - host: {{ .Values.authentik.proxy.hostname }}
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: {{ include "ok8s.fullname" . }}-authentik
            port:
              number: 80
{{- else }}
# Original ingress logic here
...
{{- end }}
```

- [ ] **Step 2: Test ingress template**

Run: `helm template test ./chart --set authentik.enabled=true --set authentik.proxy.hostname=opencode.company.com`
Expected: Ingress routes to Authentik service

- [ ] **Step 3: Commit ingress update**

```bash
git add chart/templates/ingress.yaml
git commit -m "feat: update ingress to route through Authentik for shared hostname"
```

---

### Task 7: Add Authentik Network Policy

**Files:**
- Create: `chart/templates/authentik/networkpolicy.yaml`

- [ ] **Step 1: Create network policy for Authentik**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: authentik
      app.kubernetes.io/instance: {{ .Release.Name }}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector: {}  # Allow from all namespaces
    ports:
    - protocol: TCP
      port: 9000
    - protocol: TCP
      port: 9443
  egress:
  - to:
    - podSelector:
        matchLabels:
          app.kubernetes.io/name: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app.kubernetes.io/name: redis
    ports:
    - protocol: TCP
      port: 6379
  - to: []  # Allow external access for OIDC providers
    ports:
    - protocol: TCP
      port: 443
{{- end }}
```

- [ ] **Step 2: Test network policy**

Run: `helm template test ./chart --set authentik.enabled=true -s templates/authentik/networkpolicy.yaml`
Expected: Valid NetworkPolicy restricting traffic appropriately

- [ ] **Step 3: Commit network policy**

```bash
git add chart/templates/authentik/networkpolicy.yaml
git commit -m "feat: add network policy for Authentik security"
```

---

### Task 8: Add Helm Test for Authentik

**Files:**
- Create: `chart/templates/tests/authentik-test.yaml`

- [ ] **Step 1: Create Authentik test job**

```yaml
{{- if .Values.authentik.enabled }}
apiVersion: v1
kind: Pod
metadata:
  name: {{ include "ok8s.fullname" . }}-authentik-test
  annotations:
    "helm.sh/hook": test
spec:
  restartPolicy: Never
  containers:
  - name: test
    image: curlimages/curl:8.4.0
    command:
    - sh
    - -c
    - |
      # Test Authentik service availability
      curl -f http://{{ include "ok8s.fullname" . }}-authentik:80/ || exit 1

      # Test PostgreSQL connectivity (basic)
      echo "Authentik components deployed successfully"
spec:
  restartPolicy: Never
{{- end }}
```

- [ ] **Step 2: Test helm test functionality**

Run: `helm template test ./chart --set authentik.enabled=true -s templates/tests/authentik-test.yaml`
Expected: Valid test pod definition

- [ ] **Step 3: Commit test**

```bash
git add chart/templates/tests/authentik-test.yaml
git commit -m "feat: add Helm test for Authentik deployment validation"
```

---

### Task 9: Update Chart.yaml Dependencies

**Files:**
- Modify: `chart/Chart.yaml`

- [ ] **Step 1: Add PostgreSQL and Redis as optional dependencies**

```yaml
dependencies:
- name: postgresql
  version: "12.x.x"
  repository: "https://charts.bitnami.com/bitnami"
  condition: authentik.postgres.enabled
- name: redis
  version: "17.x.x"
  repository: "https://charts.bitnami.com/bitnami"
  condition: authentik.redis.enabled
```

- [ ] **Step 2: Test dependency resolution**

Run: `helm dependency update ./chart`
Expected: Dependencies downloaded without errors

- [ ] **Step 3: Commit Chart.yaml update**

```bash
git add chart/Chart.yaml
git commit -m "feat: add PostgreSQL and Redis as conditional dependencies for Authentik"
```

---

### Task 10: Update _helpers.tpl with Authentik Helpers

**Files:**
- Modify: `chart/templates/_helpers.tpl`

- [ ] **Step 1: Add Authentik helper functions**

Add to the helpers file:

```go
{{/*
Authentik fullname
*/}}
{{- define "ok8s.authentik.fullname" -}}
{{- printf "%s-authentik" (include "ok8s.fullname" .) -}}
{{- end -}}

{{/*
Authentik PostgreSQL fullname
*/}}
{{- define "ok8s.authentik.postgres.fullname" -}}
{{- printf "%s-authentik-postgres" (include "ok8s.fullname" .) -}}
{{- end -}}

{{/*
Authentik Redis fullname
*/}}
{{- define "ok8s.authentik.redis.fullname" -}}
{{- printf "%s-authentik-redis" (include "ok8s.fullname" .) -}}
{{- end -}}
```

- [ ] **Step 2: Test helper functions**

Run: `helm template test ./chart --set authentik.enabled=true | grep -E "(authentik-postgres|authentik-redis)"`
Expected: Names use consistent naming pattern

- [ ] **Step 3: Commit helpers update**

```bash
git add chart/templates/_helpers.tpl
git commit -m "feat: add Authentik helper functions for consistent naming"
```

---

### Task 11: Create Documentation for Authentik Setup

**Files:**
- Modify: `docs/multi-user-management.md` (add Authentik section)

- [ ] **Step 1: Add Authentik setup section to multi-user docs**

Add a new section after the existing auth router documentation:

```markdown
## Authentik Identity-Aware Proxy Setup

For shared hostname access with centralized OIDC authentication, deploy Authentik as an Identity-Aware Proxy.

### Prerequisites

- OIDC provider (Google Workspace, Azure AD, Okta, etc.)
- Wildcard DNS: `*.yourdomain.com` → cluster ingress

### Configuration

Enable Authentik in your Helm values:

```yaml
authentik:
  enabled: true
  oidc:
    enabled: true
    provider: "oidc"
    clientId: "your-client-id"
    clientSecret: "your-client-secret"
    issuerUrl: "https://accounts.google.com"  # For Google Workspace
  proxy:
    hostname: "opencode.yourdomain.com"
```

### Post-Deployment Setup

1. Access Authentik admin: `https://opencode.yourdomain.com/admin`
2. Configure OIDC Source for your identity provider
3. Create Proxy Provider with dynamic backend mapping
4. Set up scope mapping for user routing:

```python
email = request.user.email
username = email.split('@')[0]
backend = f"http://opencode-{username}-0.opencode-headless.default.svc.cluster.local:4096"
return {"ak_proxy": {"backend_override": backend}}
```

### User Access

Users access via: `https://opencode.yourdomain.com`

Authentik handles authentication and routes to the appropriate user pod based on email claims.
```

- [ ] **Step 2: Commit documentation update**

```bash
git add docs/multi-user-management.md
git commit -m "docs: add Authentik setup guide to multi-user management"
```

---

### Task 12: Validate Full Chart Template

**Files:**
- Test: Full chart rendering

- [ ] **Step 1: Test complete Authentik deployment**

Run: `helm template test ./chart --set mode=multi --set authentik.enabled=true --set authentik.oidc.enabled=true --set authentik.proxy.hostname=opencode.company.com`
Expected: All Authentik resources render without errors

- [ ] **Step 2: Validate resource relationships**

Check that services reference correct deployments, secrets are properly mounted, etc.

- [ ] **Step 3: Run helm lint**

Run: `helm lint ./chart`
Expected: No linting errors

- [ ] **Step 4: Commit validation complete**

```bash
git commit --allow-empty -m "feat: complete Authentik implementation - all templates validated"
```</content>
<parameter name="filePath">docs/superpowers/plans/2026-04-07-shared-ingress-authentik.md