# Update Ingress for Authentik Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the Helm chart ingress to conditionally route through Authentik for shared hostname authentication.

**Architecture:** Conditional ingress template using Helm if-else, with different hostnames for enabled/disabled states.

**Tech Stack:** Helm, Kubernetes Ingress, Authentik

---

### Task 1: Add hostname for direct ingress

**Files:**
- Modify: `chart/values.yaml`

- [ ] **Step 1: Add hostname field under ingress section**

Add the following to the `ingress` section in `chart/values.yaml`:

```yaml
ingress:
  enabled: false
  hostname: "opencode.local"
```

- [ ] **Step 2: Commit the values update**

```bash
git add chart/values.yaml
git commit -m "feat: add hostname for direct ingress routing"
```

### Task 2: Update ingress template hostnames

**Files:**
- Modify: `chart/templates/ingress.yaml`

- [ ] **Step 1: Update else block to use ingress.hostname**

In `chart/templates/ingress.yaml`, change the host in the else block from `{{ .Values.authentik.proxy.hostname }}` to `{{ .Values.ingress.hostname }}` in both the `tls.hosts` and `rules.host` sections.

The updated else block should be:

```yaml
# Original ingress logic for direct routing to OpenCode
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "ok8s.fullname" . }}
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - {{ .Values.ingress.hostname }}
    secretName: {{ include "ok8s.fullname" . }}-tls
  rules:
  - host: {{ .Values.ingress.hostname }}
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: {{ include "ok8s.fullname" . }}
            port:
              number: 4096
```

- [ ] **Step 2: Commit the ingress template update**

```bash
git add chart/templates/ingress.yaml
git commit -m "feat: update ingress to use conditional hostnames for Authentik routing"
```

### Task 3: Test ingress template rendering

**Files:**
- Test: `chart/`

- [ ] **Step 1: Lint the chart**

Run: `helm lint ./chart`
Expected: No errors

- [ ] **Step 2: Test template with Authentik enabled**

Run: `helm template test ./chart --set authentik.enabled=true --set authentik.proxy.hostname=opencode.company.com`
Expected: Ingress routes to Authentik service on shared hostname

- [ ] **Step 3: Test template with Authentik disabled**

Run: `helm template test ./chart --set authentik.enabled=false`
Expected: Ingress routes to main service on local hostname

- [ ] **Step 4: Commit test verification**

If tests pass, commit any changes if needed, but likely no changes.

```bash
git commit --allow-empty -m "test: verify ingress template renders correctly for both modes"
```