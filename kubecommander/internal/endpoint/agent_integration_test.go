package endpoint

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
)

func TestIntegration_RegisterAgent(t *testing.T) {
	agentSvc := &mockAgentService{
		RegisterFunc: func(_ context.Context, agent *models.Agent) (*models.Agent, string, error) {
			agent.ID = "new-agent-id"
			agent.LastSeen = time.Now()
			agent.Status = models.AgentStatusOnline
			return agent, "generated-token-xyz", nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	obj := e.POST("/api/v1/agents/register").
		WithHeader("Authorization", "Bearer "+validToken).
		WithJSON(map[string]string{
			"name":        "test-agent",
			"clusterName": "my-cluster",
		}).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()

	obj.Value("id").IsEqual("new-agent-id")
	obj.Value("name").IsEqual("test-agent")
	obj.Value("clusterName").IsEqual("my-cluster")
	obj.Value("token").IsEqual("generated-token-xyz")
	obj.Value("status").IsEqual("online")
}

func TestIntegration_ListAgents(t *testing.T) {
	agentSvc := &mockAgentService{
		ListFunc: func(_ context.Context) ([]*models.Agent, error) {
			return []*models.Agent{
				{ID: "a1", Name: "agent-1", Status: models.AgentStatusOnline},
				{ID: "a2", Name: "agent-2", Status: models.AgentStatusOffline},
			}, nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	arr := e.GET("/api/v1/agents/").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusOK).
		JSON().Array()

	arr.Length().IsEqual(2)
	arr.Value(0).Object().Value("id").IsEqual("a1")
	arr.Value(1).Object().Value("id").IsEqual("a2")
}

func TestIntegration_GetAgent(t *testing.T) {
	agentSvc := &mockAgentService{
		GetFunc: func(_ context.Context, id string) (*models.Agent, error) {
			if id == "existing" {
				return &models.Agent{
					ID:          "existing",
					Name:        "found-agent",
					ClusterName: "prod",
					Status:      models.AgentStatusOnline,
				}, nil
			}
			return nil, nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	obj := e.GET("/api/v1/agents/existing").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	obj.Value("id").IsEqual("existing")
	obj.Value("name").IsEqual("found-agent")
	obj.Value("clusterName").IsEqual("prod")
}

func TestIntegration_GetAgent_NotFound(t *testing.T) {
	agentSvc := &mockAgentService{
		GetFunc: func(_ context.Context, id string) (*models.Agent, error) {
			return nil, nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	e.GET("/api/v1/agents/nonexistent").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusNotFound)
}
