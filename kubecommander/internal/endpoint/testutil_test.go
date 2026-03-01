package endpoint

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
)

const validToken = "test-valid-token"

func setupTestServer(t *testing.T, cmdSvc *mockCommandService, agentSvc *mockAgentService, authSvc *mockAuthService) *httpexpect.Expect {
	t.Helper()

	router := NewChiRouter()

	commandHandler := NewCommandHandler(cmdSvc)
	agentHandler := NewAgentHandler(agentSvc)

	router.Route("/api/v1", func(r Router) {
		r.Route("/agents", func(r Router) {
			r.Use(AuthMiddleware(authSvc))
			agentHandler.RegisterRoutes(r)
		})

		r.Route("/commands", func(r Router) {
			r.Use(AuthMiddleware(authSvc))
			commandHandler.RegisterRoutes(r)
		})
	})

	return httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(router),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	})
}

func defaultAuthService() *mockAuthService {
	return &mockAuthService{
		ValidateTokenFunc: func(_ context.Context, token string) (string, string, error) {
			if token == validToken {
				return "admin", "", nil
			}
			return "", "", fmt.Errorf("invalid token")
		},
	}
}
