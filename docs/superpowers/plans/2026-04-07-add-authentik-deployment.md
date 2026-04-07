# Add Authentik Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Authentik main server deployment and configuration with secure secret key requirement to prevent production vulnerabilities.

**Architecture:** Update Helm chart values.yaml to include secretKey configuration, modify secret template to require user-provided secret key instead of random generation, verify templates render correctly.

**Tech Stack:** Helm charts, Kubernetes manifests, Authentik identity provider.

---

### Task 1: Update values.yaml with secretKey configuration

**Files:**
- Modify: chart/values.yaml

- [ ] **Step 1: Add secretKey field to authentik section**

Add the following under the authentik section in chart/values.yaml:

```yaml
# -- Authentik secret key for encryption (REQUIRED for security)
secretKey: ""
```

- [ ] **Step 2: Verify the change**

Check that the secretKey is added after the image section in the authentik block.

---

### Task 2: Update secret.yaml to require user-provided secretKey

**Files:**
- Modify: chart/templates/authentik/secret.yaml

- [ ] **Step 1: Update secretKey generation to require user input**

Change line 9 in chart/templates/authentik/secret.yaml from:

```yaml
secretKey: {{ randAlphaNum 32 | b64enc }}
```

to:

```yaml
secretKey: {{ required "authentik.secretKey must be set to prevent production vulnerabilities" .Values.authentik.secretKey | b64enc }}
```

- [ ] **Step 2: Verify the change**

Ensure the required function is used with the appropriate error message.

---

### Task 3: Test templates render correctly

**Files:**
- Test: chart/templates/authentik/*

- [ ] **Step 1: Run helm template with test values**

Run: `helm template my-release chart/ -f chart/values.yaml --set authentik.enabled=true --set authentik.secretKey=testSecretKey123 --namespace opencode`

Expected: Templates render without errors, producing Kubernetes manifests for authentik deployment, service, configmap, and secret.

- [ ] **Step 2: Verify authentik resources are generated**

Check that the output includes:
- Deployment with authentik container
- Service with ports 80/443
- ConfigMap with authentik.yml
- Secret with encoded secretKey

---

### Task 4: Commit changes

- [ ] **Step 1: Add modified files**

```bash
git add chart/values.yaml chart/templates/authentik/secret.yaml
```

- [ ] **Step 2: Commit with conventional message**

```bash
git commit -m "feat: add Authentik main server deployment and configuration"
```

- [ ] **Step 3: Verify commit**

Run `git log --oneline -1` to confirm the commit message.</content>
<parameter name="filePath">docs/superpowers/plans/2026-04-07-add-authentik-deployment.md