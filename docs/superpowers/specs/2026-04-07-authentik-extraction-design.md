# Authentik Extraction to Separate Repository Design

## Overview
Extract Authentik Identity-Aware Proxy components from the main OpenCode Helm chart into a standalone repository (`k8s-opencode-authentik`) for independent maintenance and deployment.

## Goals
- Create a standalone Helm chart for Authentik with PostgreSQL, Redis, and Tailscale ingress
- Enable independent versioning and deployment of Authentik
- Maintain compatibility with OpenCode multi-user setup for OIDC authentication
- Ensure secure defaults and production-ready configuration

## Repository Structure
```
k8s-opencode-authentik/
├── charts/authentik/           # Main chart directory
│   ├── Chart.yaml              # Chart metadata (version, dependencies)
│   ├── values.yaml             # Default configuration values
│   └── templates/              # Kubernetes manifest templates
│       ├── deployment.yaml     # Authentik server deployment
│       ├── service.yaml        # Authentik service
│       ├── configmap.yaml      # Authentik configuration
│       ├── secret.yaml         # Authentik secrets
│       ├── postgres-deployment.yaml  # PostgreSQL database
│       ├── postgres-service.yaml     # PostgreSQL service
│       ├── postgres-pvc.yaml         # PostgreSQL persistence
│       ├── redis-deployment.yaml     # Redis cache
│       ├── redis-service.yaml        # Redis service
│       ├── ingress.yaml              # Tailscale ingress
│       └── networkpolicy.yaml        # Network policies
├── docs/
│   ├── README.md               # Installation and usage guide
│   └── opencode-integration.md # OpenCode OIDC setup guide
├── .github/workflows/
│   ├── publish-chart.yml       # OCI chart publishing to GHCR
│   ├── publish-images.yml      # Authentik image builds (if needed)
│   └── test-chart.yml          # Chart testing and validation
├── scripts/
│   ├── version-sync.sh         # Version bumping and tagging
│   ├── test-install.sh         # Local testing script
│   └── migrate-from-main.sh    # Migration helper from main repo
└── AGENTS.md                   # Agent maintenance instructions
```

## Chart Architecture

### Components
- **Authentik Server**: Main application deployment with web UI and API
- **PostgreSQL**: Primary database for user data and configurations
- **Redis**: Session storage and background task queuing
- **Tailscale Ingress**: Secure external access via MagicDNS

### Key Features
- Standalone installation: `helm install authentik ./charts/authentik`
- Configurable domain and Tailscale integration
- OIDC provider setup for external applications
- Automatic bootstrap with admin user creation
- Security hardening with secret key validation

### Dependencies
- No external Helm dependencies (PostgreSQL and Redis included)
- Requires Tailscale operator in cluster for ingress

## Configuration

### Values Structure
```yaml
# Authentik server configuration
domain: "authentik.example.ts.net"
secretKey: "CHANGE-ME-TO-SECURE-32-CHAR-STRING"

# Database configuration
postgres:
  enabled: true
  database: "authentik"
  username: "authentik"
  password: "authentikpassword"

# Cache configuration
redis:
  enabled: true
  password: "redispassword"

# Ingress configuration
ingress:
  enabled: true
  hostname: "authentik"
  cookieDomain: ".example.ts.net"

# OIDC configuration (for OpenCode integration)
oidc:
  enabled: false
  clientId: ""
  clientSecret: ""
```

### Security Considerations
- Secret key must be cryptographically secure (32+ characters)
- Database passwords use strong defaults but should be overridden
- Network policies restrict pod-to-pod communication
- Containers run as non-root users
- TLS termination at ingress level

## Migration from Main Chart

### Source Files to Extract
- `chart/templates/authentik/*` → `charts/authentik/templates/`
- Authentik section from `chart/values.yaml` → `charts/authentik/values.yaml`
- Authentik documentation from `docs/` → new repo docs

### Main Chart Cleanup
- Remove `chart/templates/authentik/` directory
- Remove authentik values from main `values.yaml`
- Update Chart.yaml to remove any Authentik dependencies

### Version Synchronization
- Independent semantic versioning for Authentik chart
- OCI publishing to `ghcr.io/timothyclin/k8s-opencode-authentik/chart/authentik`
- Tag-based releases trigger automated version updates

## Deployment Scenarios

### Standalone Authentik
```bash
helm install authentik ./charts/authentik \
  -n authentik --create-namespace \
  -f values.yaml
```

### With OpenCode Integration
1. Deploy Authentik first
2. Configure OIDC provider in Authentik admin UI
3. Deploy OpenCode with OIDC settings pointing to Authentik

## Testing Strategy

### Unit Tests
- Helm template validation: `helm template test ./charts/authentik`
- YAML syntax and structure checks
- Variable substitution verification

### Integration Tests
- Local cluster deployment with `helm install --dry-run`
- Database connectivity tests
- Admin interface accessibility
- OIDC flow validation

### End-to-End Tests
- Complete authentication flow from external client
- Session persistence across pod restarts
- Ingress routing and SSL termination

## Success Criteria
- Authentik deploys successfully as standalone chart
- Admin interface accessible via configured domain
- OIDC provider functional for external applications
- Clean separation from main OpenCode chart
- Independent CI/CD pipeline operational
- Documentation covers installation and OpenCode integration

## Risk Mitigation
- Backward compatibility maintained through OIDC configuration
- Comprehensive testing before first release
- Rollback strategy: keep Authentik in main chart until standalone proven
- Monitoring: Include Prometheus metrics if needed for production monitoring

## Future Enhancements
- LDAP integration support
- SAML provider configuration
- Multi-tenant organization setup
- Backup and restore automation
- High availability deployment options