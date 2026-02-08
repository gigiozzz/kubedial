package endpoint

import (
	"context"
	"net/http"
	"strings"

	"github.com/gigiozzz/kubedial/kubecommander/internal/service"
)

type contextKey string

const (
	contextKeyRole    contextKey = "role"
	contextKeyAgentID contextKey = "agentID"
)

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			role, agentID, err := authService.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyRole, role)
			ctx = context.WithValue(ctx, contextKeyAgentID, agentID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRole extracts the role from the request context
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(contextKeyRole).(string); ok {
		return role
	}
	return ""
}

// GetAgentID extracts the agent ID from the request context
func GetAgentID(ctx context.Context) string {
	if agentID, ok := ctx.Value(contextKeyAgentID).(string); ok {
		return agentID
	}
	return ""
}
