#!/usr/bin/env bash

source "$(dirname "$0")/../config.env"

if [ -z "$DOCKER_USERNAME" ] || [ -z "$DOCKER_PASSWORD" ] || [ -z "$DOCKER_REGISTRY" ]; then
  echo "Error: DOCKER_USERNAME, DOCKER_PASSWORD and DOCKER_REGISTRY is not set in config.env"
  exit 1
fi

echo "$DOCKER_PASSWORD" | docker login "$DOCKER_REGISTRY" -u "$DOCKER_USERNAME" --password-stdin