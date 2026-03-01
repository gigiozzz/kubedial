# Kubedial Architecture

## Overview

Kubedial uses a pull-based pattern to apply Kubernetes manifests to clusters behind firewalls. The central component (**kubecommander**) exposes a REST API; the agent (**kubedialer**) runs as a CronJob inside the private cluster and polls for commands.

```
  Admin/CI                kubecommander                  kubedialer (CronJob)
     |                    (public cluster)              (private cluster)
     |  POST /commands          |                             |
     |------------------------->|                             |
     |                          |     GET /commands/pending   |
     |                          |<----------------------------|
     |                          |     GET /commands/{id}/files|
     |                          |<----------------------------|
     |                          |        (apply/delete)       |
     |                          |     PUT /commands/{id}/result
     |                          |<----------------------------|
```

---

## 1. Data Storage (K8s Secrets)

All application data is stored in Kubernetes Secrets in the kubecommander namespace. There are four Secret types, distinguished by labels.

### Secret types

| Secret | Name pattern | Data fields | Labels |
|--------|-------------|-------------|--------|
| Agent | `agent-{UUID}` | `metadata` (JSON Agent), `token` (bearer) | `kubedial.io/type=agent` |
| Command | `cmd-{UUID}` | `metadata` (JSON Command), `result` (JSON, added after execution) | `kubedial.io/type=command`, `kubedial.io/agent-id={UUID}`, `kubedial.io/status={pending\|running\|completed\|failed}` |
| Command Files | `cmd-{UUID}-files` | `{filename}` → YAML content | `kubedial.io/type=command-files`, `kubedial.io/command-id={UUID}` |
| Users | `users` | `{userID}` → JSON `{"username":"...","token":"..."}` | (none) |

### Label-based filtering

Labels on Secrets allow efficient server-side filtering via label selectors, avoiding full list-and-filter operations in the application. Examples:

```
# Pending commands for a specific agent
kubedial.io/type=command,kubedial.io/agent-id={UUID},kubedial.io/status=pending

# All agent Secrets (for token validation)
kubedial.io/type=agent

# Files for a specific command
kubedial.io/type=command-files,kubedial.io/command-id={UUID}
```

The `kubedial.io/status` label on command Secrets is kept in sync with the `status` field inside the JSON `metadata` — both are updated atomically on each status transition.

---

## 2. Deploy ConfigMaps and Secrets

These are Kubernetes resources used to configure the application at deployment time. They are distinct from the application data Secrets described above.

### kubecommander

Defined in `kubecommander/deploy/service.yaml`:

| Resource | Name | Keys |
|----------|------|------|
| ConfigMap | `kubecommander-config` | `LOG_LEVEL` |

The `NAMESPACE` env var is injected via `fieldRef: metadata.namespace` (not from a ConfigMap). `SERVER_PORT` is hardcoded in the Deployment manifest.

### kubedialer

Defined in `kubedialer/deploy/cronjob.yaml`:

| Resource | Name | Keys |
|----------|------|------|
| ConfigMap | `kubedialer-config` | `LOG_LEVEL`, `COMMANDER_URL`, `AGENT_NAME` |
| Secret | `kubedialer-token` | `token` (the agent bearer token) |

The `kubedialer-token` Secret holds the pre-configured agent token used by the CronJob to authenticate against kubecommander. It must be populated with the token received during agent registration.

---

## 3. mTLS

Kubedial uses a single TLS listener with `tls.VerifyClientCertIfGiven` and per-route middleware to enforce different authentication schemes per endpoint group.

### Route-level enforcement

| Route group | TLS enabled | TLS disabled (backward compat) |
|-------------|-------------|-------------------------------|
| `/api/v1/agents/*` | TLS + bearer token (`AuthMiddleware`) | Bearer token |
| `/api/v1/commands/*` | mTLS only (`RequireClientCertMiddleware`, no bearer token needed) | Bearer token |

### Certificate chain

```
CA (kubedial-ca)
├── server.crt  -- kubecommander (SANs: kubecommander, kubecommander.kubedial.svc, ..., localhost)
└── client.crt  -- kubedialer (extendedKeyUsage=clientAuth)
```

Generate certificates with:
```bash
make generate-certs
make deploy-tls-secrets  # writes deploy/tls-secrets.yaml
```

### kubecommander configuration

| Env var | Description | Default |
|---------|-------------|---------|
| `TLS_ENABLED` | Enable TLS listener | `false` |
| `TLS_CERT_FILE` | Path to server certificate | `""` |
| `TLS_KEY_FILE` | Path to server private key | `""` |
| `TLS_CA_FILE` | Path to CA certificate (for client cert verification) | `""` |

### kubedialer configuration

| Env var | Description |
|---------|-------------|
| `TLS_CA_FILE` | Path to CA certificate for server verification |
| `TLS_CLIENT_CERT_FILE` | Path to client certificate |
| `TLS_CLIENT_KEY_FILE` | Path to client private key |

These can also be set via CLI flags: `--tls-ca-file`, `--tls-client-cert-file`, `--tls-client-key-file`.

### K8s Secrets

| Secret | Contents | Used by |
|--------|----------|---------|
| `kubecommander-tls` | `ca.crt`, `server.crt`, `server.key` | kubecommander Deployment |
| `kubedialer-tls` | `ca.crt`, `client.crt`, `client.key` | kubedialer CronJob |

Both Secrets are mounted at `/etc/kubedial/tls/` in their respective pods.

---

## 4. Authentication

### Token generation

Agent tokens are generated at registration time using `crypto/rand`:

```go
b := make([]byte, 32)
rand.Read(b)
token = base64.URLEncoding.EncodeToString(b)  // 44-char base64url string
```

The token is stored in the agent Secret under `Data["token"]` and returned once in the registration response. It is never retrievable again through the API.

### Two-phase validation

Every request passes through `AuthMiddleware`, which:

1. Extracts the token from `Authorization: Bearer {token}`
2. Calls `AuthRepository.ValidateToken`, which performs a two-phase lookup:
   - **Phase 1 — Agent check**: List all Secrets with `kubedial.io/type=agent`, compare `Data["token"]` with the provided token. If matched → `role=agent`, `agentID` extracted from Secret name (`agent-{UUID}`)
   - **Phase 2 — User check**: Get the `users` Secret, iterate over each entry (JSON `{"username":"...","token":"..."}`), compare tokens. If matched → `role=admin`
3. Injects `role` and `agentID` into the request context

### Roles

| Role | Source | Capabilities |
|------|--------|-------------|
| `agent` | Agent Secret token | Fetch pending commands, download files, submit results |
| `admin` | Users Secret token | All API operations (register agents, create commands, list everything) |

---

## 5. Operational Workflow

### Step-by-step flow

```
kubedialer (CronJob)                    kubecommander
      |                                      |
      |  POST /api/v1/agents/register        |
      |------------------------------------->|
      |  {agent, token}                      |  -- creates/updates agent-{UUID} Secret
      |<-------------------------------------|
      |                                      |
      |  GET /api/v1/commands/pending        |
      |    ?agentId={UUID}                   |
      |------------------------------------->|
      |  [{command1}, {command2}, ...]       |  -- label selector query
      |<-------------------------------------|
      |                                      |
      |  for each command:                   |
      |    GET /api/v1/commands/{id}/files/{filename}
      |------------------------------------->|
      |  (YAML content)                      |
      |<-------------------------------------|
      |                                      |
      |  (apply/delete on local cluster)     |
      |                                      |
      |  PUT /api/v1/commands/{id}/result    |
      |------------------------------------->|
      |  204 No Content                      |  -- updates status label + saves result
      |<-------------------------------------|
```

### Registration behavior

kubedialer registers on every CronJob invocation. If the agent Secret already exists (same agent ID), the repository updates the metadata (preserving the token) rather than creating a new Secret. The registration response includes the token only when a new agent is created.

### Status transitions

```
pending --> running --> completed
                    --> failed
```

Status is updated by kubecommander via `UpdateStatus`, which patches both the `status` field in `Data["metadata"]` and the `kubedial.io/status` label.

---

## 6. Applyer Pattern

kubedialer uses a dynamic Kubernetes client (not controller-runtime) to apply and delete manifests.

### Client setup

```
K8sApplyer
├── dynamic.Interface       -- runtime resource operations
├── discovery.DiscoveryInterface  -- API group/resource discovery
└── meta.RESTMapper         -- GVK → REST mapping (built from discovery)
```

### Apply

Two modes depending on `serverSide` flag:

| Mode | Mechanism | When resource exists |
|------|-----------|---------------------|
| Server-side | `PATCH application/apply-patch+yaml` | API server handles merge |
| Client-side | `GET` then `CREATE` or `UPDATE` | `resourceVersion` preserved for update |

Field manager is always `"kubedialer"`.

### Delete

Resources are deleted in **reverse order** relative to how they appear in the manifest, to respect dependency order (e.g., Deployment before Service).

Propagation policy:
- Default (`Force=false`): `DeletePropagationForeground` — waits for dependents to be deleted
- With `Force=true`: `DeletePropagationBackground` — returns immediately

### Namespace resolution

For namespaced resources, if the manifest does not specify a namespace and the command includes a `namespace` field, kubedialer sets it on the object before applying. Cluster-scoped resources ignore the namespace option.

---

## 7. Project Structure

```
kubedial/
├── go.work
├── common/                    # Shared library module
│   ├── models/                # Domain models (Command, Agent, CommandResult)
│   └── provider/              # Logging (zerolog) + K8s client utilities
│
├── kubecommander/             # REST API server module
│   ├── cmd/main.go
│   └── internal/
│       ├── config/
│       ├── repository/        # Secret-based implementations + tests
│       │   ├── command.go     # CommandRepository interface + impl
│       │   ├── agent.go       # AgentRepository interface + impl
│       │   └── auth.go        # AuthRepository interface + impl
│       ├── service/           # Business logic
│       │   ├── command.go
│       │   ├── agent.go       # Token generation here
│       │   └── auth.go
│       ├── endpoint/          # HTTP handlers, router, middleware
│       │   └── middleware.go  # AuthMiddleware, context helpers
│       └── server/
│
└── kubedialer/                # Agent CLI module (Cobra)
    ├── cmd/
    │   ├── root.go
    │   ├── run.go             # Main polling loop
    │   └── apply.go
    └── internal/
        ├── client/            # HTTP client for kubecommander API
        └── executor/
            └── applyer.go     # K8sApplyer (dynamic + discovery clients)
```
