#!/usr/bin/env bash
set -e

source "$(dirname "$0")/../config.env"

echo "🔐 Setting up registry credentials in Kind cluster..."

# Ensure we're using the correct context
kubectl config use-context "$KUBECTL_CONTEXT"

# Create the docker-registry secret in all necessary namespaces
NAMESPACES=("default" "op-hello-world-system")

for NAMESPACE in "${NAMESPACES[@]}"; do
  # Create namespace if it doesn't exist
  kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
  
  # Delete existing secret if it exists
  kubectl delete secret ghcr-login -n "$NAMESPACE" --ignore-not-found=true
  
  # Create the secret from the local Docker config
  kubectl create secret generic ghcr-login \
    --from-file=.dockerconfigjson="$HOME/.docker/config.json" \
    --type=kubernetes.io/dockerconfigjson \
    -n "$NAMESPACE"
  
  echo "✅ Registry secret created in namespace: $NAMESPACE"
done

# Patch default service account to use the secret for image pulls
for NAMESPACE in "${NAMESPACES[@]}"; do
  kubectl patch serviceaccount default -n "$NAMESPACE" \
    -p '{"imagePullSecrets": [{"name": "ghcr-login"}]}' || true
done

echo "✅ Registry credentials configured successfully!"