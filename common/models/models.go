package models

import "time"

// OperationType represents the type of operation to perform on manifests
type OperationType string

const (
	// OperationTypeApply applies the manifests to the cluster
	OperationTypeApply OperationType = "apply"
	// OperationTypeDelete deletes the manifests from the cluster
	OperationTypeDelete OperationType = "delete"
)

// CommandStatus represents the execution status of a command
type CommandStatus string

const (
	// CommandStatusPending indicates the command is waiting to be executed
	CommandStatusPending CommandStatus = "pending"
	// CommandStatusRunning indicates the command is currently being executed
	CommandStatusRunning CommandStatus = "running"
	// CommandStatusCompleted indicates the command was executed successfully
	CommandStatusCompleted CommandStatus = "completed"
	// CommandStatusFailed indicates the command execution failed
	CommandStatusFailed CommandStatus = "failed"
)

// AgentStatus represents the status of an agent
type AgentStatus string

const (
	// AgentStatusOnline indicates the agent is connected and responsive
	AgentStatusOnline AgentStatus = "online"
	// AgentStatusOffline indicates the agent is not responding
	AgentStatusOffline AgentStatus = "offline"
)

// Command represents a manifest command to be executed by an agent
type Command struct {
	// ID is the unique identifier of the command (UUID v4)
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`

	// AgentID is the unique identifier of the target agent (UUID v4)
	AgentID string `json:"agentId" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`

	// OperationType specifies whether to apply or delete manifests
	OperationType OperationType `json:"operationType" example:"apply"`

	// Namespace is the target Kubernetes namespace for the manifests
	Namespace string `json:"namespace" example:"default"`

	// ServerSide enables server-side apply mode
	ServerSide bool `json:"serverSide" example:"true"`

	// DryRun enables dry-run mode (no actual changes)
	DryRun bool `json:"dryRun" example:"false"`

	// Force enables force delete (immediate deletion without graceful period)
	Force bool `json:"force" example:"false"`

	// Prune enables pruning of resources not in the manifests
	Prune bool `json:"prune" example:"false"`

	// Filenames is the list of manifest file names attached to this command
	Filenames []string `json:"filenames" example:"deployment.yaml,service.yaml"`

	// Status is the current execution status of the command
	Status CommandStatus `json:"status" example:"pending"`

	// CreatedAt is the timestamp when the command was created (RFC3339 format)
	CreatedAt time.Time `json:"createdAt" example:"2024-01-15T10:30:00Z"`

	// UpdatedAt is the timestamp when the command was last updated (RFC3339 format)
	UpdatedAt time.Time `json:"updatedAt" example:"2024-01-15T10:35:00Z"`
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	// CommandID is the unique identifier of the executed command (UUID v4)
	CommandID string `json:"commandId" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Output contains the stdout/stderr output from the execution
	Output string `json:"output" example:"deployment.apps/nginx created"`

	// Error contains the error message if execution failed
	Error string `json:"error,omitempty" example:"resource not found"`

	// Success indicates whether the execution was successful
	Success bool `json:"success" example:"true"`

	// ExecutedAt is the timestamp when the command was executed (RFC3339 format)
	ExecutedAt time.Time `json:"executedAt" example:"2024-01-15T10:35:00Z"`
}

// Agent represents a kubedialer agent
type Agent struct {
	// ID is the unique identifier of the agent (UUID v4)
	ID string `json:"id" example:"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`

	// Name is the human-readable name of the agent
	Name string `json:"name" example:"production-cluster-agent"`

	// ClusterName is the human-readable name of the Kubernetes cluster
	ClusterName string `json:"clusterName" example:"prod-eu-west-1"`

	// ClusterID is the UUID of the kube-system namespace in the cluster
	ClusterID string `json:"clusterId" example:"d4735e3a-265e-16d6-a3e6-8d4f2b2e0c9a"`

	// LastSeen is the timestamp when the agent last contacted the server (RFC3339 format)
	LastSeen time.Time `json:"lastSeen" example:"2024-01-15T10:30:00Z"`

	// Status is the current status of the agent
	Status AgentStatus `json:"status" example:"online"`
}
