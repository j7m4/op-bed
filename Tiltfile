# Tiltfile for Grafana LGTM + Alloy Stack

# Load environment variables from config.env
load('ext://dotenv', 'dotenv')
dotenv('./config.env')

# Update settings
update_settings(max_parallel_updates=3)

# Allow Kubernetes context
allow_k8s_contexts('kind-op-bed')

# Load Kubernetes manifests for Grafana LGTM stack
k8s_yaml([
    'k8s/pyroscope/config-configmap.yaml',
    'k8s/pyroscope/service.yaml',
    'k8s/pyroscope/deployment.yaml',
    'k8s/grafana-lgtm/datasources-configmap.yaml',
    'k8s/grafana-lgtm/deployment.yaml',
    'k8s/alloy/config-configmap.yaml',
    'k8s/alloy/rbac.yaml',
    'k8s/alloy/service.yaml',
    'k8s/alloy/deployment.yaml'
])

# Configure resources
k8s_resource('pyroscope',
    port_forwards=['4040:4040'],
    labels=['observability']
)

k8s_resource('lgtm',
    port_forwards=[
        '3000:3000',  # Grafana
        '3100:3100',  # Loki
        '9090:9090',  # Prometheus
        '3200:3200',  # Tempo
        '4317:4317',  # OTLP gRPC
        '4318:4318'   # OTLP HTTP
    ],
    labels=['observability'],
    resource_deps=['pyroscope']
)

k8s_resource('alloy',
    port_forwards=['12345:12345'],
    labels=['observability'],
    resource_deps=['lgtm', 'pyroscope']
)

# Include op-hello-world if it exists
if os.path.exists('op-hello-world'):
    local_resource(
        'build-ohw',
        cmd="""
        make -C op-hello-world build
        echo "Build complete"
        tilt trigger docker-build-push-ohw
        """,
        #deps=['./op-hello-world/api'],
        labels=['op-hello-world'],
        trigger_mode=TRIGGER_MODE_MANUAL,
        auto_init=False
    )

    local_resource(
        'docker-build-push-ohw',
        cmd="""
        make -C op-hello-world docker-build docker-push
        echo "Docker build and push complete"
        tilt trigger kind-load-image-ohw
        """,
        resource_deps=['build-ohw'],
        labels=['op-hello-world']
    )

    local_resource(
        'kind-load-image-ohw',
        cmd="""
        make -C op-hello-world kind-load-image
        echo "Image loaded into kind cluster"
        tilt trigger deploy-controller-ohw
        """,
        resource_deps=['docker-build-push-ohw'],
        labels=['op-hello-world']
    )

    local_resource(
        'deploy-controller-ohw',
        cmd="""
        make -C op-hello-world deploy-controller
        echo "Controller deployed"
        tilt trigger install-resource-ohw
        """,
        resource_deps=['kind-load-image-ohw'],
        labels=['op-hello-world']
    )

    local_resource(
        'undeploy-controller-ohw',
        'make -C op-hello-world undeploy-controller; echo "Controller undeployed"',
        labels=['op-hello-world'],
        trigger_mode=TRIGGER_MODE_MANUAL,
        auto_init=False
    )

    local_resource(
        'install-resource-ohw',
        'make -C op-hello-world install-resource; echo "Resource installed"',
        resource_deps=['deploy-controller-ohw'],
        labels=['op-hello-world']
    )

    local_resource(
       'uninstall-resource-ohw',
       'make -C op-hello-world uninstall-resource; echo "Resource uninstalled"',
       labels=['op-hello-world'],
       trigger_mode=TRIGGER_MODE_MANUAL,
       auto_init=False
   )


