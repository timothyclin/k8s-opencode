# Authentik Proxy Integration for Shared Ingress

## Executive Summary

This design implements the integration of Authentik Identity-Aware Proxy (IAP) with the existing OpenCode multi-user infrastructure. By enabling Authentik's proxy capabilities and configuring OIDC authentication with dynamic backend routing, users can access their isolated OpenCode instances through a single shared hostname (`opencode.company.com`) with seamless authentication via Google Workspace or other OIDC providers.

## Problem Statement

The current OpenCode deployment has Authentik infrastructure in place but not configured as an IAP. The proxy functionality is disabled, and the multi-user services are not set up for headless DNS resolution required for Authentik's dynamic routing. Users cannot currently access OpenCode through the shared ingress architecture described in the design specifications.

## Requirements

### Functional Requirements
- **Enable Authentik Proxy**: Configure Authentik deployment to act as an identity-aware proxy
- **OIDC Integration**: Support authentication via enterprise identity providers (Google Workspace, Azure AD, Okta)
- **Dynamic Routing**: Route authenticated users to their OpenCode pods based on email-to-username mapping
- **Headless Services**: Modify multi-user services to enable direct pod DNS resolution
- **Shared Hostname**: Single access point for all users through configured hostname

### Non-Functional Requirements
- **Security**: Maintain existing pod isolation and zero-trust access
- **Compatibility**: Work with existing multi-user StatefulSet naming conventions
- **Maintainability**: Declarative configuration with minimal manual setup
- **Scalability**: Support dynamic user routing without service restarts

## Architecture Overview

```
Internet
    ↓ HTTPS (SSL termination via Ingress)
[NGINX Ingress Controller]
    ↓ (routes to Authentik when enabled)
[Authentik Proxy] ← OIDC authentication + dynamic routing
    ↓ (backend_override based on email)
[OpenCode StatefulSets]
    ├── opencode-alice-0 (alice@company.com)
    ├── opencode-bob-0 (bob@company.com)
    └── ...
```

## Implementation Details

### 1. Authentik Configuration Updates

**Helm Values Changes:**
- Enable `authentik.proxy.enabled: true`
- Configure OIDC source settings:
  ```yaml
  authentik:
    proxy:
      enabled: true
      hostname: "opencode.company.com"
    oidc:
      enabled: true
      provider: "oidc"
      clientId: "..."
      clientSecret: "..."
      issuerUrl: "..."
  ```

### 2. Service Modifications for Headless DNS

**Current State:** Multi-user mode creates regular ClusterIP services per user
**Required State:** Headless services to enable pod-level DNS resolution

Modify `chart/templates/service.yaml` to make multi-user services headless:
```yaml
spec:
  clusterIP: None  # Makes it headless
  selector:
    app.kubernetes.io/user: {{ $user.name | quote }}
```

This enables DNS resolution: `opencode-{username}-0.{service-name}.{namespace}.svc.cluster.local`

### 3. Authentik UI Configuration (Post-Deployment)

**OIDC Source Setup:**
- Provider: Google Workspace / Azure AD / Okta
- Client ID/Secret from identity provider
- Scopes: `openid`, `email`, `profile`

**Scope Mapping for Dynamic Routing:**
```python
email = request.user.email
username = email.split('@')[0]
backend_url = f"http://opencode-{username}-0.opencode-headless.{namespace}.svc.cluster.local:4096"
return {"ak_proxy": {"backend_override": backend_url}}
```

**Proxy Provider Configuration:**
- Mode: Forward Auth (Single Application)
- Authentication: OIDC Source
- Authorization: Allow authenticated users
- Upstream URL: Placeholder (overridden by scope mapping)

**Application Setup:**
- Provider: Proxy Provider
- External Host: `opencode.company.com`
- Redirect URIs: `https://opencode.company.com/*`

### 4. StatefulSet Compatibility

**Existing Infrastructure:** StatefulSets are named `opencode-{username}` with serviceName matching the headless service
**DNS Resolution:** Pod `opencode-{username}-0` is directly addressable via headless service DNS
**No Changes Required:** Current naming convention perfectly aligns with Authentik's routing logic

## Integration with Existing Codebase

### Operator Compatibility
- No operator changes required
- Operator continues to create deterministic StatefulSet names
- Headless services provide DNS resolution for Authentik routing

### Helm Chart Updates
- `chart/values.yaml`: Enable proxy and add OIDC configuration
- `chart/templates/service.yaml`: Modify multi-user services to be headless
- `chart/templates/ingress.yaml`: Already supports conditional Authentik routing

### Authentik Templates
- Existing Authentik deployment templates work unchanged
- Proxy configuration handled via UI and scope mappings
- No additional Kubernetes resources needed

## Security Considerations

### Authentication Security
- **OIDC Best Practices**: PKCE, secure token handling, proper scopes
- **Session Management**: Configurable timeouts, secure cookies
- **Claim Validation**: Authentik validates OIDC claims on every request

### Network Security
- **TLS End-to-End**: HTTPS from client to Authentik, then to pods
- **Pod Isolation**: Existing NetworkPolicies prevent cross-user access
- **No Service Account Tokens**: Pods run without Kubernetes API access

### Authorization
- **Email-Based Routing**: Only mapped emails can access corresponding pods
- **No Cross-User Access**: Users cannot reach other users' pods
- **Audit Logging**: Authentik provides comprehensive access logs

## Deployment and Operations

### Prerequisites
- Authentik deployed and accessible via web UI
- Enterprise OIDC identity provider configured
- Wildcard DNS pointing to cluster ingress
- Multi-user OpenCode deployment with headless services

### Deployment Steps
1. Update `values.yaml` to enable Authentik proxy and configure OIDC
2. Modify service template for headless configuration
3. Deploy updated Helm chart
4. Access Authentik admin UI (`https://opencode.company.com/admin`)
5. Configure OIDC source, scope mapping, proxy provider, and application
6. Test authentication and routing with sample user

### Configuration Validation
- Verify headless service DNS resolution
- Test OIDC authentication flow
- Confirm dynamic backend routing works
- Validate user isolation and security

### Monitoring
- Authentik access logs for authentication events
- Kubernetes pod logs for routing verification
- Network policy enforcement monitoring

## Migration Path

### From Current State
1. Enable Authentik proxy in values.yaml
2. Update services to headless mode
3. Deploy configuration changes
4. Configure Authentik UI components
5. Test and validate routing
6. Update DNS if needed

### Rollback Plan
- Disable Authentik proxy in values.yaml
- Revert service changes
- Redeploy to restore direct ingress routing
- Keep Authentik running for future use

## Success Criteria

- ✅ Users authenticate via OIDC to shared hostname
- ✅ Automatic routing to correct user pods based on email
- ✅ Headless services enable direct pod DNS resolution
- ✅ No performance degradation vs current setup
- ✅ Security audit passes enterprise requirements
- ✅ Operational monitoring in place

## Risks and Mitigations

### Risk: Authentik Configuration Complexity
**Mitigation**: Document step-by-step UI configuration process

### Risk: DNS Resolution Issues
**Mitigation**: Test headless service DNS before rollout

### Risk: OIDC Provider Compatibility
**Mitigation**: Test with target identity provider in staging

### Risk: Routing Logic Errors
**Mitigation**: Implement comprehensive testing of routing logic

## Future Enhancements

- **User Self-Service**: Portal for workspace provisioning
- **Admin Dashboard**: User management and access controls
- **Multi-Provider Support**: Multiple OIDC sources
- **Advanced Policies**: Time-based access, IP restrictions

## Conclusion

This integration enables the shared ingress architecture by leveraging Authentik's built-in proxy capabilities with dynamic backend override. The solution maintains security, scalability, and user experience while integrating cleanly with the existing OpenCode multi-user infrastructure. The headless service modification and Authentik UI configuration complete the implementation of centralized authentication and routing.</content>
<parameter name="filePath">docs/superpowers/specs/2026-04-07-authentik-proxy-integration-design.md