#!/bin/bash
# Script to preload images into Kind cluster to avoid re-downloading

set -e

# Load common configuration
source "$(dirname "$0")/../config.env"

echo "🚀 Preloading images for Kind cluster: $CLUSTER_NAME"

# List of images to preload
IMAGES=(

  "redis:8.0.3"

  # Observability stack
  "grafana/otel-lgtm:0.11.6"
  "grafana/pyroscope:main-6d0f426"
  "grafana/alloy:v1.5.0"
  "otel/opentelemetry-collector-contrib:0.130.1"
)

# Pull images to local Docker first
echo "📥 Pulling images to local Docker..."
for image in "${IMAGES[@]}"; do
  if docker image inspect "$image" >/dev/null 2>&1; then
    echo "✓ Image already cached: $image"
  else
    echo "⬇️  Pulling: $image"
    if ! docker pull "$image"; then
      echo "❌ Failed to pull: $image"
    fi
  fi
done

# Load images into Kind cluster
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  echo ""
  echo "📦 Loading images into Kind cluster..."
  for image in "${IMAGES[@]}"; do
    echo "Loading: $image"
    if ! kind load docker-image "$image" --name "$CLUSTER_NAME"; then
      echo "❌ Failed to load image: $image"
    else
      echo "✅ Successfully loaded: $image"
    fi
  done
  echo ""
  echo "✅ Images preload complete, check for preload errors above!"
else
  echo ""
  echo "⚠️  Kind cluster '$CLUSTER_NAME' not found. Create it first with:"
  echo "   kind create cluster --name $CLUSTER_NAME"
fi
