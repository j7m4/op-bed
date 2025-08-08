# Tiltfile for Grafana LGTM + Alloy Stack

# Load environment variables from config.env
load('ext://dotenv', 'dotenv')
dotenv('./config.env')

# Update settings
update_settings(max_parallel_updates=3)

# Allow Kubernetes context
allow_k8s_contexts('kind-op-bed')

# Install Prometheus Operator CRDs
# This is required for ServiceMonitor resources used by op-hello-world and potentially other operators
local_resource(
    'prometheus-operator-crds',
    'helm repo add prometheus-community https://prometheus-community.github.io/helm-charts && ' +
    'helm repo update && ' +
    'helm upgrade --install prometheus-crds prometheus-community/prometheus-operator-crds --create-namespace --namespace monitoring',
    labels=['observability']
)

# Load Kubernetes manifests in order
# 1. ConfigMaps first
k8s_yaml([
    'k8s/loki/config-configmap.yaml',
    'k8s/tempo/config-configmap.yaml',
    'k8s/mimir/config-configmap.yaml',
    'k8s/pyroscope/config-configmap.yaml',
    'k8s/grafana/datasources-configmap.yaml',
    'k8s/grafana/dashboards-configmap.yaml',
    'k8s/alloy/config-configmap.yaml'
])

# 2. RBAC for Alloy
k8s_yaml('k8s/alloy/rbac.yaml')

# 3. Services
k8s_yaml([
    'k8s/loki/service.yaml',
    'k8s/tempo/service.yaml',
    'k8s/mimir/service.yaml',
    'k8s/pyroscope/service.yaml',
    'k8s/grafana/service.yaml',
    'k8s/alloy/service.yaml'
])

# 4. Deployments
k8s_yaml([
    'k8s/loki/deployment.yaml',
    'k8s/tempo/deployment.yaml',
    'k8s/mimir/deployment.yaml',
    'k8s/pyroscope/deployment.yaml',
    'k8s/alloy/deployment.yaml',
    'k8s/grafana/deployment.yaml'
])

# Configure resources
k8s_resource('loki', 
    port_forwards=['3100:3100'],
    labels=['observability']
)

k8s_resource('tempo',
    port_forwards=['3200:3200', '14317:4317', '14318:4318'],
    labels=['observability']
)

k8s_resource('mimir',
    port_forwards=['9009:9009'],
    labels=['observability']
)

k8s_resource('pyroscope',
    port_forwards=['4040:4040'],
    labels=['observability']
)

k8s_resource('alloy',
    port_forwards=['12345:12345'],
    labels=['observability'],
    resource_deps=['loki', 'tempo', 'mimir', 'pyroscope']
)

k8s_resource('grafana',
    port_forwards=['3000:3000'],
    labels=['observability'],
    resource_deps=['loki', 'tempo', 'mimir', 'pyroscope']
)

# Include op-hello-world if it exists
if os.path.exists('op-hello-world'):
    # Build and load the op-hello-world operator Docker image into Kind
    local_resource(
        'op-hello-world-docker',
        'cd op-hello-world && make docker-load-image',
        deps=['./op-hello-world'],
        labels=['operator']
    )

    # Apply operator manifests
    k8s_yaml(kustomize('./op-hello-world/config/default'))

    k8s_resource('op-hello-world-controller-manager',
        port_forwards=['8080:8080'],
        labels=['operator'],
        resource_deps=['alloy', 'op-hello-world-docker', 'prometheus-operator-crds']
    )
