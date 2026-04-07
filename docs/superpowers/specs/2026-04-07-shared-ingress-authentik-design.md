# Shared Ingress with Authentik for Multi-User OpenCode

## Executive Summary

This design implements a shared ingress architecture for multi-user OpenCode deployments using Authentik as an Identity-Aware Proxy (IAP). All users access OpenCode through a single shared hostname (e.g., `opencode.company.com`) with centralized OIDC authentication via Google Workspace. Authentik dynamically routes authenticated users to their isolated StatefulSet pods based on email-to-user mapping, providing secure, scalable multi-tenant access without per-user subdomains.

## Problem Statement

The current multi-user OpenCode deployment uses per-user subdomains (e.g., `opencode-alice.opencode.ts.net`) with a custom auth-router that validates hostname-to-user mapping. This requires:

- Separate DNS entries for each user
- Custom routing logic in the auth-router
- Users to know their specific subdomain

Organizations want a single, centralized access point with professional branding and simplified user experience.

## Requirements

### Functional Requirements
- **Shared Hostname**: Single domain (e.g., `opencode.company.com`) for all users
- **OIDC Authentication**: Google Workspace integration for enterprise SSO
- **Identity-Based Routing**: Route users to pods based on authenticated email claims
- **Session Persistence**: Maintain user sessions across requests
- **Security**: Zero-trust access with continuous claim validation

### Non-Functional Requirements
- **Scalability**: Support hundreds of concurrent users
- **Security**: No custom routing code; use battle-tested IAP
- **Maintainability**: Declarative configuration, minimal operational overhead
- **Compatibility**: Integrate with existing OpenCode operator and Helm chart

## Architecture Overview

```
Internet
    ↓ HTTPS (SSL termination)
[Authentik Outpost] ← Validates OIDC sessions
    ↓ (extracts user email from claims)
[Authentik Proxy] ← Dynamic backend override
    ↓ (routes to user pod)
[OpenCode StatefulSets]
    ├── opencode-alice-0 (alice@company.com)
    ├── opencode-bob-0 (bob@company.com)
    └── ...
```

## Components

### 1. Authentik Identity Provider
- **Purpose**: Central identity management and OIDC orchestration
- **Deployment**: Kubernetes deployment with PostgreSQL and Redis
- **Configuration**:
  - OIDC Source: Google Workspace
  - Scope Mappings: Extract email, compute dynamic backend URL
  - Proxy Provider: Handle HTTP traffic with dynamic routing

### 2. Authentik Outpost/Proxy
- **Purpose**: Edge proxy handling authentication and routing
- **Deployment**: Kubernetes deployment as ingress controller
- **Functionality**:
  - SSL termination
  - OIDC session validation
  - Dynamic backend target calculation
  - Request proxying to user pods

### 3. OpenCode StatefulSets (Existing)
- **Purpose**: Isolated user workspaces
- **Naming**: `opencode-{username}-0` (deterministic)
- **Network**: Headless Service for direct pod access
- **DNS**: `opencode-{username}-0.opencode-headless.{namespace}.svc.cluster.local:4096`

## Implementation Details

### Email-to-User Mapping
Authentik handles user identity through OIDC claims. Username extraction uses email prefix:

```python
# Authentik Scope Mapping expression
email = request.user.email  # e.g., "alice@company.com"
username = email.split('@')[0]  # e.g., "alice"
```

### Dynamic Backend Override
During authentication, Authentik executes secure Python expression to compute target:

```python
email = request.user.email
username = email.split('@')[0]
backend_url = f"http://opencode-{username}-0.opencode-headless.default.svc.cluster.local:4096"
return {
    "ak_proxy": {
        "backend_override": backend_url
    }
}
```

### Traffic Flow
1. **Unauthenticated Access**: User visits `https://opencode.company.com`
2. **OIDC Redirect**: Authentik redirects to Google Workspace login
3. **Token Exchange**: Google returns OIDC token with user claims
4. **Backend Calculation**: Authentik evaluates scope mapping, computes pod DNS
5. **Session Establishment**: Proxy target overridden for user's session
6. **Request Proxying**: All subsequent requests route directly to user's StatefulSet

### Error Handling
- **Unknown Email**: 403 Forbidden with "User not found" message
- **Pod Unavailable**: 502 Bad Gateway (standard reverse proxy behavior)
- **Auth Failure**: Redirect to login with error message

## Integration with Existing Codebase

### Operator Changes (Minimal)
- No changes required - operator already creates deterministic pod names
- Ensure headless services exist for DNS resolution

### Helm Chart Updates
- Add Authentik as optional component when `auth.oidc.enabled: true`
- Update ingress configuration to route through Authentik outpost
- Add OIDC configuration values:
  ```yaml
  auth:
    oidc:
      enabled: true
      provider: "oidc"
      clientId: "..."
      clientSecret: "..."
      hostname: "opencode.company.com"
  ```

### Auth-Router Replacement
- Remove existing auth-router deployment
- Authentik assumes all routing responsibilities
- Maintain email mapping ConfigMap for backward compatibility (optional)

## Security Considerations

### Authentication Security
- **OIDC Best Practices**: PKCE, secure token storage, proper scopes
- **Session Management**: Configurable session timeouts, secure cookies
- **Claim Validation**: Continuous verification of user identity per request

### Network Security
- **TLS Everywhere**: End-to-end encryption from client to pod
- **Pod Isolation**: Existing NetworkPolicies prevent cross-user communication
- **No Service Account Tokens**: Pods run without Kubernetes API access

### Authorization
- **Email-Based Access**: Only mapped emails can access corresponding pods
- **No Cross-User Access**: Users cannot access other users' pods
- **Audit Logging**: Authentik provides comprehensive access logs

## Deployment and Operations

### Prerequisites
- Google Workspace with OIDC configured
- Wildcard DNS: `*.company.com` → cluster ingress
- Tailscale (optional, for remote access)

### Deployment Steps
1. Deploy Authentik via Helm chart
2. Configure Google Workspace OIDC source
3. Create Proxy Provider with scope mappings
4. Update OpenCode ingress to use Authentik
5. Test with sample user workspaces

### Monitoring and Observability
- Authentik dashboard for auth metrics
- Kubernetes logs for proxy events
- Pod resource monitoring
- User access audit logs

### Backup and Recovery
- Authentik database backups (PostgreSQL)
- User workspace PVC backups (existing scripts)
- Configuration as code in Git

## Migration Path

### From Current Auth-Router
1. Deploy Authentik alongside existing setup
2. Test authentication and routing
3. Update DNS to point shared domain to Authentik
4. Remove old auth-router after validation

### Rollback Plan
- Keep old auth-router deployment ready
- DNS TTL considerations for quick rollback
- User communication about new access method

## Success Criteria

- ✅ Users access OpenCode via single shared URL
- ✅ Seamless Google Workspace authentication
- ✅ Automatic routing to correct user pods
- ✅ No performance degradation vs current setup
- ✅ Security audit passes enterprise requirements
- ✅ Operational monitoring in place

## Risks and Mitigations

### Risk: Authentik Complexity
**Mitigation**: Start with minimal configuration, leverage existing Helm charts

### Risk: OIDC Configuration Issues
**Mitigation**: Test thoroughly with Google Workspace admin

### Risk: DNS Resolution Failures
**Mitigation**: Validate headless service DNS before rollout

### Risk: Session State Loss
**Mitigation**: Configure appropriate session persistence

## Future Enhancements

- **User Self-Service**: Portal for users to request workspace creation
- **Admin Dashboard**: User management and access controls
- **Multi-Cluster**: Cross-cluster user routing
- **Advanced Policies**: Time-based access, IP restrictions

## Conclusion

This Authentik-based shared ingress architecture provides enterprise-grade multi-user access to OpenCode with centralized authentication and dynamic routing. By leveraging Authentik's Identity-Aware Proxy capabilities, we eliminate custom routing code while maintaining security and scalability. The design integrates cleanly with existing OpenCode operator patterns and provides a superior user experience through a single, branded access point.</content>
<parameter name="filePath">docs/superpowers/specs/2026-04-07-shared-ingress-authentik-design.md