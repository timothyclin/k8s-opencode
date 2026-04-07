# Design: Update Ingress for Authentik Routing

## Summary

Update the Helm chart's ingress template to conditionally route traffic through Authentik when enabled, allowing Authentik to handle OIDC authentication for the shared ingress architecture.

## Requirements

- When `authentik.enabled=true`, create ingress routing to Authentik service on shared hostname (`authentik.proxy.hostname`)
- When `authentik.enabled=false`, create ingress routing to main OpenCode service on local hostname (`ingress.hostname`)
- Use proper TLS with cert-manager and nginx ingress class

## Architecture

- Conditional ingress template using Helm if-else logic
- Hostnames differentiated: shared for Authentik-enabled routing, local for direct routing
- Services: 
  - Authentik service on port 80 when enabled
  - Main OpenCode service on port 4096 when disabled
- Annotations for SSL redirect and cluster issuer

## Implementation

- Modify `chart/templates/ingress.yaml` with conditional blocks
- Add `hostname` field to `chart/values.yaml` under `ingress` section
- Ensure template renders correctly in both modes

## Testing

- Run `helm lint ./chart` to check for errors
- Run `helm template` with `authentik.enabled=true/false` to verify correct routing
- Confirm ingress routes to appropriate service based on condition