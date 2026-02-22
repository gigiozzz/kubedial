# Kubedial

Kubedial enables applying/deleting Kubernetes manifests to clusters behind firewalls using a pull-based pattern.

## Components

- **kubecommander**: REST API server that stores commands and manifest files
- **kubedialer**: CLI agent that runs as a CronJob, pulls commands, and applies manifests

## Quick Start

### Build

```bash
make build
```

### Run locally

```bash
# Start kubecommander
./bin/kubecommander

# Run kubedialer
./bin/kubedialer run --commander-url=http://localhost:8080 --agent-token=<token> --agent-name=my-agent
```

### Deploy to Kubernetes

```bash
# Create namespace
kubectl create namespace kubedial

# Deploy kubecommander
kubectl apply -f kubecommander/deploy/

# Deploy kubedialer
kubectl apply -f kubedialer/deploy/
```

## Usage

### Submit a command

```bash
curl -X POST http://localhost:8080/api/v1/commands \
  -H "Authorization: Bearer <token>" \
  -F 'metadata={"agentId":"<agent-id>","operation":"apply","namespace":"default"}' \
  -F "files=@deployment.yaml"
```

### List commands

```bash
curl http://localhost:8080/api/v1/commands \
  -H "Authorization: Bearer <token>"
```

### Get command status

```bash
curl http://localhost:8080/api/v1/commands/<command-id> \
  -H "Authorization: Bearer <token>"
```

## Configuration

### kubecommander

| Variable | Description | Default |
|----------|-------------|---------|
| LOG_LEVEL | Log level (debug, info, warn, error) | info |
| SERVER_PORT | HTTP server port | 8080 |
| NAMESPACE | Kubernetes namespace for storage | kubedial |

### kubedialer

| Variable | Description | Default |
|----------|-------------|---------|
| LOG_LEVEL | Log level (debug, info, warn, error) | info |
| COMMANDER_URL | URL of kubecommander | - |
| AGENT_TOKEN | Bearer token for authentication | - |
| AGENT_NAME | Name of this agent | - |

## Documentation

- [Architecture](docs/architecture.md) — data storage, authentication, operational workflow, applyer pattern

## License

Apache License 2.0
