package endpoint

import (
	"context"
	"net/http"
	"testing"

	"github.com/gigiozzz/kubedial/common/models"
)

func TestIntegration_AuthMiddleware_ValidToken(t *testing.T) {
	agentSvc := &mockAgentService{
		ListFunc: func(_ context.Context) ([]*models.Agent, error) {
			return []*models.Agent{}, nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	e.GET("/api/v1/agents/").
		WithHeader("Authorization", "Bearer "+validToken).
		Expect().
		Status(http.StatusOK)
}

func TestIntegration_AuthMiddleware_MissingHeader(t *testing.T) {
	agentSvc := &mockAgentService{
		ListFunc: func(_ context.Context) ([]*models.Agent, error) {
			return []*models.Agent{}, nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	e.GET("/api/v1/agents/").
		Expect().
		Status(http.StatusUnauthorized)
}

func TestIntegration_AuthMiddleware_InvalidFormat(t *testing.T) {
	agentSvc := &mockAgentService{
		ListFunc: func(_ context.Context) ([]*models.Agent, error) {
			return []*models.Agent{}, nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	e.GET("/api/v1/agents/").
		WithHeader("Authorization", "InvalidFormat").
		Expect().
		Status(http.StatusUnauthorized)
}

func TestIntegration_AuthMiddleware_InvalidToken(t *testing.T) {
	agentSvc := &mockAgentService{
		ListFunc: func(_ context.Context) ([]*models.Agent, error) {
			return []*models.Agent{}, nil
		},
	}

	e := setupTestServer(t, &mockCommandService{}, agentSvc, defaultAuthService())

	e.GET("/api/v1/agents/").
		WithHeader("Authorization", "Bearer invalid-token").
		Expect().
		Status(http.StatusUnauthorized)
}
