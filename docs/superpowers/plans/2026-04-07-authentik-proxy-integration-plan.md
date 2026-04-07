# Authentik Proxy Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable Authentik Identity-Aware Proxy for shared ingress with dynamic user routing based on OIDC authentication.

**Architecture:** Modify Helm values to enable Authentik proxy, update multi-user services to be headless for DNS resolution, deploy configuration changes, and manually configure Authentik UI components for OIDC authentication and dynamic backend routing.

**Tech Stack:** Kubernetes, Helm, Authentik, OIDC, NGINX Ingress

---

### Task 1: Enable Authentik Proxy in Values

**Files:**
- Modify: `chart/values.yaml` (authentik section)

- [ ] **Step 1: Update authentik.proxy.enabled to true**

Modify the authentik section in `chart/values.yaml`:

```yaml
authentik:
  enabled: true
  proxy:
    enabled: true  # Change from false to true
    hostname: "opencode.company.com"
    cookieDomain: ".company.com"
```

- [ ] **Step 2: Configure OIDC settings**

Add OIDC configuration below the proxy section:

```yaml
  oidc:
    enabled: true  # Change from false to true
    provider: "oidc"
    clientId: ""  # To be filled by user
    clientSecret: ""  # To be filled by user
    issuerUrl: ""  # To be filled by user
    scopes: ["openid", "email", "profile"]
```

- [ ] **Step 3: Validate values.yaml syntax**

Run: `helm template test ./chart --set authentik.enabled=true --dry-run`
Expected: No YAML syntax errors

- [ ] **Step 4: Commit proxy configuration**

```bash
git add chart/values.yaml
git commit -m "feat: enable Authentik proxy and add OIDC configuration"
```

### Task 2: Modify Multi-User Services for Headless DNS

**Files:**
- Modify: `chart/templates/service.yaml` (multi-user section)

- [ ] **Step 1: Update multi-user service spec to be headless**

Modify the multi-user service section in `chart/templates/service.yaml`. Find the multi-user service spec and add `clusterIP: None`:

```yaml
{{- if eq .Values.mode "multi" }}
{{- $root := . }}
{{- range $index, $user := .Values.users }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ include "ok8s.fullname" $root }}-user-{{ $user.name }}
  namespace: {{ $root.Release.Namespace }}
  labels:
    {{- include "ok8s.labels" $root | nindent 4 }}
    app.kubernetes.io/user: {{ $user.name | quote }}
spec:
  clusterIP: None  # Add this line to make service headless
  selector:
    {{- include "ok8s.selectorLabels" $root | nindent 4 }}
    app.kubernetes.io/user: {{ $user.name | quote }}
  ports:
  - name: http
    port: 4096
    targetPort: 4096
    protocol: TCP
{{- end }}
{{- end }}
```

- [ ] **Step 2: Test service template rendering**

Run: `helm template test ./chart --set mode=multi --set users[0].name=testuser --dry-run`
Expected: Service rendered with `clusterIP: None`

- [ ] **Step 3: Commit service changes**

```bash
git add chart/templates/service.yaml
git commit -m "feat: make multi-user services headless for Authentik DNS resolution"
```

### Task 3: Deploy Updated Configuration

**Files:**
- Deploy: Kubernetes cluster via Helm

- [ ] **Step 1: Deploy updated chart to test namespace**

Run: `helm upgrade --install ok8s-test ./chart --namespace opencode-authentik-test --set authentik.enabled=true --set mode=multi --set users[0].name=testuser --set authentik.secretKey=your-secret-key-here`

Expected: Deployment succeeds, Authentik pod running

- [ ] **Step 2: Verify Authentik pod status**

Run: `kubectl get pods -n opencode-authentik-test -l app.kubernetes.io/name=authentik`
Expected: Pod in Running state

- [ ] **Step 3: Check headless service DNS resolution**

Run: `kubectl run test-dns --image=busybox --rm -it --restart=Never -- nslookup opencode-testuser-0.ok8s-test-user-testuser.opencode-authentik-test.svc.cluster.local`
Expected: DNS resolution succeeds

### Task 4: Configure Authentik OIDC Source

**Files:**
- Configure: Authentik web UI

- [ ] **Step 1: Access Authentik admin interface**

Navigate to: `https://opencode.company.com/admin` (using configured hostname)

Expected: Authentik login page loads

- [ ] **Step 2: Create OIDC Source**

In Authentik UI:
1. Go to Directory → Federation & Social login
2. Click "Create" → OAuth2/OpenID Provider
3. Configure:
   - Name: "Google Workspace" (or your provider)
   - Provider type: Select your OIDC provider
   - Client ID: Enter from identity provider
   - Client Secret: Enter from identity provider
   - Scopes: `openid email profile`
4. Save the configuration

- [ ] **Step 3: Test OIDC source**

Click "Test" or "Check" in the OIDC source configuration
Expected: Successful connection to identity provider

### Task 5: Create Scope Mapping for Dynamic Routing

**Files:**
- Configure: Authentik web UI

- [ ] **Step 1: Navigate to Property Mappings**

In Authentik UI:
1. Go to Customization → Property Mappings
2. Click "Create" → Authentik: Scope Mapping

- [ ] **Step 2: Configure scope mapping**

Create new scope mapping with:
- Name: "OpenCode Dynamic Routing"
- Scope name: `ak_proxy`
- Expression:

```python
email = request.user.email
username = email.split('@')[0]
backend_url = f"http://opencode-{username}-0.ok8s-test-user-{username}.opencode-authentik-test.svc.cluster.local:4096"
return {"ak_proxy": {"backend_override": backend_url}}
```

- [ ] **Step 3: Save and validate mapping**

Save the scope mapping
Expected: No syntax errors in expression

### Task 6: Create Proxy Provider

**Files:**
- Configure: Authentik web UI

- [ ] **Step 1: Navigate to Providers**

In Authentik UI:
1. Go to Applications → Providers
2. Click "Create" → Proxy Provider

- [ ] **Step 2: Configure proxy provider**

Configure with:
- Name: "OpenCode Proxy"
- Authentication: Select the OIDC source created in Task 4
- Authorization: Allow all authenticated users
- Mode: "Forward auth (single application)"
- Upstream URL: `http://placeholder:4096` (will be overridden)
- Cookie domain: `.company.com`
- Additional scopes: Select the scope mapping from Task 5

- [ ] **Step 3: Save proxy provider**

Save the configuration
Expected: Provider created successfully

### Task 7: Create Application

**Files:**
- Configure: Authentik web UI

- [ ] **Step 1: Navigate to Applications**

In Authentik UI:
1. Go to Applications → Applications
2. Click "Create"

- [ ] **Step 2: Configure application**

Configure with:
- Name: "OpenCode"
- Provider: Select the proxy provider from Task 6
- External host: `opencode.company.com`
- Redirect URIs: `https://opencode.company.com/*`

- [ ] **Step 3: Save application**

Save the configuration
Expected: Application created and accessible

### Task 8: Test Authentication and Routing

**Files:**
- Test: Browser access to application

- [ ] **Step 1: Access shared hostname**

Navigate to: `https://opencode.company.com`

Expected: Redirected to OIDC provider login

- [ ] **Step 2: Complete authentication**

Log in with test user credentials

Expected: Successful authentication, redirect back to OpenCode

- [ ] **Step 3: Verify correct routing**

Check that user is routed to their specific OpenCode instance

Expected: User's OpenCode workspace loads with correct data

- [ ] **Step 4: Test user isolation**

Attempt to access another user's workspace

Expected: Access denied or routed to own workspace

### Task 9: Validate Security and Performance

**Files:**
- Test: Security and performance validation

- [ ] **Step 1: Check HTTPS enforcement**

Run: `curl -I http://opencode.company.com`
Expected: 301 redirect to HTTPS

- [ ] **Step 2: Verify session persistence**

Access application, close browser, reopen
Expected: Still authenticated (within session timeout)

- [ ] **Step 3: Test performance**

Run load test with multiple concurrent users
Expected: No significant performance degradation

- [ ] **Step 4: Review Authentik logs**

Check Authentik logs for authentication events
Expected: Clean logs with no errors

- [ ] **Step 5: Commit validation results**

```bash
git add docs/testing/authentik-integration-test-results.md  # If created
git commit -m "docs: add Authentik integration test results"
```

---

**Implementation Notes:**

- **OIDC Provider Setup:** The user must configure their identity provider (Google Workspace, Azure AD, etc.) separately and obtain client credentials before Task 4.

- **Hostname Configuration:** Update `authentik.proxy.hostname` in values.yaml to match your actual domain.

- **DNS Requirements:** Ensure wildcard DNS (`*.company.com`) points to your ingress controller.

- **Namespace Awareness:** The scope mapping expression uses the Helm release namespace - adjust if using different namespace.

- **Security:** Never commit actual client secrets to git - use Helm secrets or external secret management.

- **Testing:** Test with a single user first before enabling for multiple users.

**Success Criteria:**
- Users can authenticate via OIDC to shared hostname
- Automatic routing to correct user pods based on email
- Headless services enable direct pod DNS resolution
- Security and performance meet requirements</content>
<parameter name="filePath">docs/superpowers/plans/2026-04-07-authentik-proxy-integration-plan.md