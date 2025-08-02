# Kubernetes Operators Project

This project contains multiple Kubernetes operators built with Kubebuilder. Each operator is organized 
in its own directory with an `op-` prefix.

## Requirements

### Prerequisites

- **Go** (version 1.21 or later)
- **Docker** (for building container images)
- **kind** (Kubernetes in Docker) - already configured
- **kubectl** (for interacting with Kubernetes clusters)
- **Kubebuilder** (for operator development)

### Development Environment

- Each operator directory follows the naming convention: `op-<operator-name>`
- Operators are built using the Kubebuilder framework
- Local development and testing use kind clusters

#### Setting up the environment

Setup: 
* creates the kind cluster, if necessary
* sets the current context to the cluster
* preinstalls images defined in `scripts/preload-images.sh`

#### Tearing down the environment

It can be helpful to start from a clean slate by deleting the kind cluster.

```bash
./teardown.sh
```

### Getting Started

1. Ensure all prerequisites are installed
2. Navigate to the specific operator directory (e.g., `op-example`)
3. Follow the operator-specific README for build and deployment instructions

### Project Structure

```
op-bed/
├── op-<operator1>/     # First operator
├── op-<operator2>/     # Second operator
├── op-<operator3>/     # Third operator
└── ...                 # Additional operators
```

Each operator directory contains its own Kubebuilder-generated structure with controllers, APIs, and configuration files.