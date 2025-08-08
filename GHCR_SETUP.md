# GitHub Container Registry Setup

This guide explains how to authenticate with GitHub Container Registry (ghcr.io) to push the op-hello-world controller image.

## Prerequisites

1. GitHub Personal Access Token (PAT) with `write:packages` scope
2. Docker installed and running

## Authentication Steps

1. Create a GitHub Personal Access Token:
   - Go to GitHub Settings → Developer settings → Personal access tokens
   - Click "Generate new token (classic)"
   - Select the `write:packages` scope (this will auto-select `read:packages`)
   - Generate and copy the token
   - Save this in `config.env` as `GHCR_PAT`; also, save `GHCR_USER` to your GitHub username.
   - In `config.env`, ensure `DOCKER_REGISTRY`, `DOCKER_USERNAME`, and `DOCKER_PASSWORD` are set in `config.env` from the `GHCR_*` values.

2. Log in to GitHub Container Registry:
   ```bash
   source config.env
   echo $DOCKER_PASSWORD | docker login $DOCKER_REGISTRY -u $DOCKER_USERNAME --password-stdin
   ```
   
   Or, use `./scripts/docker-login.sh`

## Building and Pushing the Controller Image

In each operator directory (e.g., `op-hello-world`), the `Makefile` is set up to build and push the Docker image.

1. Build the Docker image:
   ```bash
   make docker-build
   ```

2. Push the image to GitHub Container Registry:
   ```bash
   make docker-push
   ```

   Or do both in one command:
   ```bash
   make docker-build docker-push
   ```

## Using a Different Tag

To use a specific tag instead of `latest`:

```bash
make docker-build docker-push IMG=ghcr.io/${GHRC_USER}/op-hello-world:v1.0.0
```

## Deploying to Kubernetes

Deploy the controller using the GitHub Container Registry image:

```bash
# Install CRDs
make install

# Deploy controller (uses the IMG from Makefile by default)
make deploy

# Or with a specific image tag
make deploy IMG=ghcr.io/${GHRC_USER}/op-hello-world:v1.0.0
```

## Making the Image Public (Optional)

By default, GitHub Container Registry images are private. To make the image public:

1. Go to https://github.com/${GHRC_USER}?tab=packages
2. Find the `op-hello-world` package
3. Click on "Package settings"
4. Scroll to "Danger Zone" and click "Change visibility"
5. Select "Public" and confirm

This allows anyone to pull the image without authentication, which is useful for open-source projects.