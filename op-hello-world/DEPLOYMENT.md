# Deployment Guide

This guide explains how to deploy the op-hello-world operator in development and production environments.

## Configuration Structure

The configuration has been restructured into environment-specific overlays:

```
config/
├── base/                           # Common resources
│   ├── crd/                       # Custom Resource Definitions
│   ├── manager/                   # Controller deployment
│   ├── rbac/                      # Role-based access control
│   │   ├── base/                  # Core RBAC (required)
│   │   └── production/            # Additional production RBAC
│   └── samples/                   # Example HelloWorld resources
├── overlays/
│   ├── development/               # Development environment
│   └── production/                # Production environment
```

## Development Deployment

### Prerequisites
- Kubernetes cluster (local or remote)
- `kubectl` configured to access the cluster
- `kustomize` or `kubectl` with built-in kustomize support

### Features
- **Insecure HTTP metrics** on port 8080
- **No TLS certificates** required
- **Minimal RBAC** (core operator permissions only)
- **No NetworkPolicy** restrictions
- **Fast setup** for local development

### Deploy Development Environment

1. **Install CRDs:**
   ```bash
   make install-crd
   ```

2. **Deploy the operator:**
   ```bash
   # Using existing make targets (now points to development overlay)
   make deploy-development-controller
   
   # Or directly with kustomize
   kubectl apply -k config/overlays/development
   ```

3. **Create a HelloWorld resource:**
   ```bash
   make install-resource
   ```

4. **Access metrics (optional):**
   ```bash
   # Port forward to access metrics
   kubectl port-forward -n op-hello-world-system deployment/op-hello-world-controller-manager 8080:8080
   
   # View metrics
   curl http://localhost:8080/metrics
   ```

### Cleanup Development Environment
```bash
make uninstall-resource           # Remove HelloWorld resources
make undeploy-development-controller # Remove operator
```

## Production Deployment

### Prerequisites
- Kubernetes cluster
- [cert-manager](https://cert-manager.io/) installed
- Prometheus Operator (for ServiceMonitor)
- `kubectl` configured with cluster-admin permissions

### Features
- **Secure HTTPS metrics** on port 8443 with TLS certificates
- **Full RBAC** with metrics authentication/authorization
- **NetworkPolicy** for metrics endpoint protection
- **Certificate management** via cert-manager
- **ServiceMonitor** for Prometheus integration

### Deploy Production Environment

1. **Install cert-manager** (if not already installed):
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
   ```

2. **Install CRDs:**
   ```bash
   make install-crd
   ```

3. **Deploy the operator:**
   ```bash
   # Update image in production kustomization if needed
   cd config/overlays/production
   kustomize edit set image controller=ghcr.io/j7m4/op-hello-world:v1.0.0
   
   # Deploy
   kubectl apply -k config/overlays/production
   ```

4. **Verify certificate creation:**
   ```bash
   kubectl get certificate -n op-hello-world-system
   kubectl get secret metrics-server-cert -n op-hello-world-system
   ```

5. **Label namespace for metrics access** (required by NetworkPolicy):
   ```bash
   # Label the monitoring namespace to allow metrics scraping
   kubectl label namespace monitoring metrics=enabled
   ```

6. **Create HelloWorld resource:**
   ```bash
   kubectl apply -f config/base/samples/apps_v1_helloworld.yaml
   ```

### Production Configuration Details

#### TLS Certificates
- **Certificate:** `metrics-certs` managed by cert-manager
- **Secret:** `metrics-server-cert` contains TLS cert, key, and CA
- **Issuer:** Self-signed ClusterIssuer (replace with proper CA in real production)

#### NetworkPolicy
- Only allows metrics access from namespaces labeled with `metrics: enabled`
- Restricts traffic to port 8443 only

#### RBAC
Production includes additional roles:
- `metrics-auth-role`: Token and subject access review permissions
- `metrics-reader`: Read access to `/metrics` endpoint
- `secret-reader-role`: Access to container registry secrets

### Production Monitoring

1. **ServiceMonitor:** Automatically created for Prometheus scraping
2. **Metrics endpoint:** `https://op-hello-world-controller-manager-metrics-service.op-hello-world-system.svc:8443/metrics`
3. **Authentication:** Uses ServiceAccount bearer tokens

### Cleanup Production Environment
```bash
kubectl delete -k config/overlays/production
```

## Switching Between Environments

### Development to Production
```bash
# Remove development deployment
kubectl delete -k config/overlays/development

# Deploy production (ensure cert-manager is installed)
kubectl apply -k config/overlays/production
```

### Production to Development
```bash
# Remove production deployment
kubectl delete -k config/overlays/production

# Deploy development
kubectl apply -k config/overlays/development
```

## Troubleshooting

### Development Issues
1. **Metrics not accessible:** Check if port-forward is running
2. **Operator not starting:** Check logs with `kubectl logs -n op-hello-world-system deployment/op-hello-world-controller-manager`

### Production Issues
1. **Certificate not ready:** 
   ```bash
   kubectl describe certificate metrics-certs -n op-hello-world-system
   kubectl describe clusterissuer selfsigned-issuer
   ```

2. **Metrics endpoint not accessible:**
   - Check NetworkPolicy allows your monitoring namespace
   - Verify TLS certificate is properly mounted
   - Check ServiceMonitor configuration

3. **Prometheus scraping fails:**
   - Ensure monitoring namespace has `metrics: enabled` label
   - Verify ServiceMonitor is in correct namespace
   - Check TLS configuration in ServiceMonitor

## Customization

### Custom Images
Update the image in the respective kustomization.yaml:
```bash
cd config/overlays/development  # or production
kustomize edit set image controller=your-registry/op-hello-world:your-tag
```

### Custom Certificate Issuer (Production)
Replace the ClusterIssuer in `config/overlays/production/issuer.yaml` with your preferred issuer (e.g., ACME, intermediate CA).

### Additional RBAC
Add custom roles to the appropriate overlay directory and reference them in the kustomization.yaml.