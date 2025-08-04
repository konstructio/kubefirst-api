# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

The kubefirst-api is a REST API runtime implementation for the Kubefirst GitOps infrastructure and application delivery platform. It's written in Go and provides cluster management operations across multiple cloud providers (AWS, Civo, DigitalOcean, Vultr, Google Cloud, Akamai).

## Development Commands

### Building and Running

```bash
# Build the binary
make build

# Run locally with live reloading (requires air)
air

# Run directly
go run .

# Update Swagger documentation
make updateswagger
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/...

# Run tests with coverage
go test -cover ./...

# Run a specific test
go test -run TestName ./path/to/package
```

### Environment Setup

1. Ensure you have a local k3d cluster for development:
   ```bash
   k3d cluster create dev
   k3d kubeconfig write dev
   ```

2. Set required environment variables (see `.env.example`):
   - `K1_LOCAL_DEBUG=true` 
   - `K1_LOCAL_KUBECONFIG_PATH=/path/to/kubeconfig`
   - `CLUSTER_ID`, `CLUSTER_TYPE`, `INSTALL_METHOD`
   - `K1_ACCESS_TOKEN` for API authentication
   - `IS_CLUSTER_ZERO=true` if running without console

## Architecture Overview

### Core Components

- **API Layer** (`internal/router/`): RESTful API endpoints using Gin framework
  - `/api/v1/cluster` - Cluster management operations
  - `/api/v1/domain` - Domain/DNS management
  - `/api/v1/services` - Service management (GitHub, GitLab integrations)
  - `/api/v1/telemetry` - Telemetry endpoints

- **Provider Implementations** (`providers/`, `internal/`):
  - Each cloud provider has create/delete operations
  - Provider-specific logic in `internal/{aws,civo,digitalocean,vultr,google,azure}/`
  - Common interfaces for cloud operations

- **Controllers** (`internal/controller/`):
  - Orchestrates complex operations across multiple services
  - Manages state transitions and error handling
  - Handles ArgoCD, Vault, and Kubernetes operations

- **Kubernetes Integration** (`internal/k8s/`, `internal/kubernetes/`):
  - Client wrappers for Kubernetes API operations
  - Secret management and RBAC operations
  - Job and namespace management

- **GitOps** (`internal/gitShim/`, `internal/gitClient/`):
  - Git operations for repository management
  - GitOps catalog integration
  - Token and authentication handling

### Key Patterns

1. **Authentication**: Bearer token authentication via `K1_ACCESS_TOKEN` environment variable
2. **Cloud Credentials**: Can be provided via Kubernetes secrets or API parameters
3. **State Management**: Uses Kubernetes secrets to store cluster state
4. **Error Handling**: Consistent error wrapping with contextual messages
5. **Provider Abstraction**: Common interfaces with provider-specific implementations

## Dependencies

- Go 1.23.0
- Key libraries:
  - Gin for HTTP routing
  - ArgoCD for GitOps operations
  - Kubernetes client-go for K8s operations
  - Cloud provider SDKs (AWS, GCP, etc.)
  - Swagger for API documentation

## Important Notes

- Always run `swag init` after modifying API route documentation
- The API runs on port 8081 by default
- Swagger UI available at http://localhost:8081/swagger/index.html
- For local development with CLI, set `K1_LOCAL_DEBUG=true` and run console locally
- When editing error handling, wrap errors with `fmt.Errorf("context: %w", err)`