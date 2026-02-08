package endpoint

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
)

func TestIntegration_ListCommands(t *testing.T) {
	cmdSvc := &mockCommandService{
		ListFunc: func(_ context.Context) ([]*models.Command, error) {
			return []*models.Command{
				{ID: "cmd-1", AgentID: "a1", Status: models.CommandStatusPending},
				{ID: "cmd-2", AgentID: "a2", Status: models.CommandStatusCompleted},
			}, nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	arr := e.GET("/api/v1/commands/").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	arr.Length().IsEqual(2)
}

func TestIntegration_GetCommand(t *testing.T) {
	cmdSvc := &mockCommandService{
		GetWithResultFunc: func(_ context.Context, id string) (*models.Command, *models.CommandResult, error) {
			if id == "cmd-1" {
				cmd := &models.Command{
					ID:            "cmd-1",
					AgentID:       "agent-1",
					OperationType: models.OperationTypeApply,
					Status:        models.CommandStatusCompleted,
				}
				result := &models.CommandResult{
					CommandID: "cmd-1",
					Output:    "deployment created",
					Success:   true,
				}
				return cmd, result, nil
			}
			return nil, nil, nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	obj := e.GET("/api/v1/commands/cmd-1").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	obj.Value("id").IsEqual("cmd-1")
	obj.Value("status").IsEqual("completed")
	obj.Value("result").Object().Value("output").IsEqual("deployment created")
}

func TestIntegration_GetCommand_NotFound(t *testing.T) {
	cmdSvc := &mockCommandService{
		GetWithResultFunc: func(_ context.Context, id string) (*models.Command, *models.CommandResult, error) {
			return nil, nil, nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	e.GET("/api/v1/commands/nonexistent").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusNotFound)
}

func TestIntegration_GetPendingCommands(t *testing.T) {
	cmdSvc := &mockCommandService{
		GetPendingFunc: func(_ context.Context, agentID string) ([]*models.Command, error) {
			if agentID == "agent-1" {
				return []*models.Command{
					{ID: "cmd-p1", AgentID: "agent-1", Status: models.CommandStatusPending},
				}, nil
			}
			return []*models.Command{}, nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	arr := e.GET("/api/v1/commands/pending").
		WithHeader("Authorization", "Bearer "+validToken).
		WithQuery("agentId", "agent-1").
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	arr.Length().IsEqual(1)
	arr.Value(0).Object().Value("id").IsEqual("cmd-p1")
}

func TestIntegration_GetPendingCommands_MissingAgentId(t *testing.T) {
	cmdSvc := &mockCommandService{}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	e.GET("/api/v1/commands/pending").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusBadRequest)
}

func TestIntegration_GetCommandFile(t *testing.T) {
	cmdSvc := &mockCommandService{
		GetFileFunc: func(_ context.Context, commandID, filename string) ([]byte, error) {
			if commandID == "cmd-1" && filename == "deploy.yaml" {
				return []byte("apiVersion: apps/v1\nkind: Deployment"), nil
			}
			return nil, nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	e.GET("/api/v1/commands/cmd-1/files/deploy.yaml").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusOK).
		ContentType("application/x-yaml").
		Body().IsEqual("apiVersion: apps/v1\nkind: Deployment")
}

func TestIntegration_GetCommandFile_NotFound(t *testing.T) {
	cmdSvc := &mockCommandService{
		GetFileFunc: func(_ context.Context, commandID, filename string) ([]byte, error) {
			return nil, nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	e.GET("/api/v1/commands/cmd-1/files/missing.yaml").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusNotFound)
}

func TestIntegration_ListCommandFiles(t *testing.T) {
	cmdSvc := &mockCommandService{
		ListFilesFunc: func(_ context.Context, commandID string) ([]string, error) {
			return []string{"deploy.yaml", "service.yaml"}, nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	arr := e.GET("/api/v1/commands/cmd-1/files").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	arr.Length().IsEqual(2)
}

func TestIntegration_UpdateResult(t *testing.T) {
	var capturedResult *models.CommandResult
	cmdSvc := &mockCommandService{
		UpdateResultFunc: func(_ context.Context, commandID string, result *models.CommandResult) error {
			capturedResult = result
			return nil
		},
	}

	e := setupTestServer(t, cmdSvc, &mockAgentService{}, defaultAuthService())

	e.PUT("/api/v1/commands/cmd-1/result").
		WithHeader("Authorization", "Bearer "+validToken).
		WithJSON(map[string]interface{}{
			"output":     "deployment.apps/nginx created",
			"success":    true,
			"executedAt": time.Now().Format(time.RFC3339),
		}).
		Expect().
		Status(http.StatusOK)

	if capturedResult == nil {
		t.Fatal("expected result to be captured")
	}
	if capturedResult.Output != "deployment.apps/nginx created" {
		t.Errorf("expected output 'deployment.apps/nginx created', got '%s'", capturedResult.Output)
	}
}
