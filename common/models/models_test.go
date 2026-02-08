package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCommand_JSONMarshalTime(t *testing.T) {
	// Create a command with specific timestamps
	createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC)

	cmd := Command{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		AgentID:       "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		OperationType: OperationTypeApply,
		Namespace:     "default",
		ServerSide:    true,
		DryRun:        false,
		Force:         false,
		Prune:         false,
		Filenames:     []string{"deployment.yaml", "service.yaml"},
		Status:        CommandStatusPending,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}

	// Marshal to JSON
	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("failed to marshal command: %v", err)
	}

	// Verify RFC3339 format in JSON
	jsonStr := string(data)

	// Expected RFC3339 format: "2024-01-15T10:30:00Z"
	expectedCreatedAt := `"createdAt":"2024-01-15T10:30:00Z"`
	expectedUpdatedAt := `"updatedAt":"2024-01-15T10:35:00Z"`

	if !strings.Contains(jsonStr, expectedCreatedAt) {
		t.Errorf("createdAt not in RFC3339 format.\nExpected to contain: %s\nGot: %s", expectedCreatedAt, jsonStr)
	}

	if !strings.Contains(jsonStr, expectedUpdatedAt) {
		t.Errorf("updatedAt not in RFC3339 format.\nExpected to contain: %s\nGot: %s", expectedUpdatedAt, jsonStr)
	}
}

func TestCommand_JSONUnmarshalTime(t *testing.T) {
	// JSON with RFC3339 timestamps
	jsonData := `{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"agentId": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"operationType": "apply",
		"namespace": "default",
		"serverSide": true,
		"dryRun": false,
		"force": false,
		"prune": false,
		"filenames": ["deployment.yaml", "service.yaml"],
		"status": "pending",
		"createdAt": "2024-01-15T10:30:00Z",
		"updatedAt": "2024-01-15T10:35:00Z"
	}`

	var cmd Command
	if err := json.Unmarshal([]byte(jsonData), &cmd); err != nil {
		t.Fatalf("failed to unmarshal command: %v", err)
	}

	// Verify timestamps
	expectedCreatedAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	expectedUpdatedAt := time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC)

	if !cmd.CreatedAt.Equal(expectedCreatedAt) {
		t.Errorf("createdAt mismatch.\nExpected: %v\nGot: %v", expectedCreatedAt, cmd.CreatedAt)
	}

	if !cmd.UpdatedAt.Equal(expectedUpdatedAt) {
		t.Errorf("updatedAt mismatch.\nExpected: %v\nGot: %v", expectedUpdatedAt, cmd.UpdatedAt)
	}
}

func TestCommandResult_JSONMarshalTime(t *testing.T) {
	executedAt := time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC)

	result := CommandResult{
		CommandID:  "550e8400-e29b-41d4-a716-446655440000",
		Output:     "deployment.apps/nginx created",
		Success:    true,
		ExecutedAt: executedAt,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	jsonStr := string(data)
	expectedExecutedAt := `"executedAt":"2024-01-15T10:35:00Z"`

	if !strings.Contains(jsonStr, expectedExecutedAt) {
		t.Errorf("executedAt not in RFC3339 format.\nExpected to contain: %s\nGot: %s", expectedExecutedAt, jsonStr)
	}
}

func TestCommandResult_JSONUnmarshalTime(t *testing.T) {
	jsonData := `{
		"commandId": "550e8400-e29b-41d4-a716-446655440000",
		"output": "deployment.apps/nginx created",
		"success": true,
		"executedAt": "2024-01-15T10:35:00Z"
	}`

	var result CommandResult
	if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	expectedExecutedAt := time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC)

	if !result.ExecutedAt.Equal(expectedExecutedAt) {
		t.Errorf("executedAt mismatch.\nExpected: %v\nGot: %v", expectedExecutedAt, result.ExecutedAt)
	}
}

func TestAgent_JSONMarshalTime(t *testing.T) {
	lastSeen := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	agent := Agent{
		ID:          "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		Name:        "production-cluster-agent",
		ClusterName: "prod-eu-west-1",
		ClusterID:   "d4735e3a-265e-16d6-a3e6-8d4f2b2e0c9a",
		LastSeen:    lastSeen,
		Status:      AgentStatusOnline,
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal agent: %v", err)
	}

	jsonStr := string(data)
	expectedLastSeen := `"lastSeen":"2024-01-15T10:30:00Z"`
	expectedClusterName := `"clusterName":"prod-eu-west-1"`

	if !strings.Contains(jsonStr, expectedLastSeen) {
		t.Errorf("lastSeen not in RFC3339 format.\nExpected to contain: %s\nGot: %s", expectedLastSeen, jsonStr)
	}

	if !strings.Contains(jsonStr, expectedClusterName) {
		t.Errorf("clusterName not in JSON.\nExpected to contain: %s\nGot: %s", expectedClusterName, jsonStr)
	}
}

func TestAgent_JSONUnmarshalTime(t *testing.T) {
	jsonData := `{
		"id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"name": "production-cluster-agent",
		"clusterName": "prod-eu-west-1",
		"clusterId": "d4735e3a-265e-16d6-a3e6-8d4f2b2e0c9a",
		"lastSeen": "2024-01-15T10:30:00Z",
		"status": "online"
	}`

	var agent Agent
	if err := json.Unmarshal([]byte(jsonData), &agent); err != nil {
		t.Fatalf("failed to unmarshal agent: %v", err)
	}

	expectedLastSeen := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	if !agent.LastSeen.Equal(expectedLastSeen) {
		t.Errorf("lastSeen mismatch.\nExpected: %v\nGot: %v", expectedLastSeen, agent.LastSeen)
	}

	if agent.ClusterName != "prod-eu-west-1" {
		t.Errorf("clusterName mismatch.\nExpected: prod-eu-west-1\nGot: %s", agent.ClusterName)
	}

	if agent.ClusterID != "d4735e3a-265e-16d6-a3e6-8d4f2b2e0c9a" {
		t.Errorf("clusterId mismatch.\nExpected: d4735e3a-265e-16d6-a3e6-8d4f2b2e0c9a\nGot: %s", agent.ClusterID)
	}
}
