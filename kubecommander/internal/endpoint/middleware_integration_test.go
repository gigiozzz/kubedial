package endpoint

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
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

func TestIntegration_RequireClientCertMiddleware_NoTLS(t *testing.T) {
	handler := RequireClientCertMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// r.TLS is nil (plain HTTP)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestIntegration_RequireClientCertMiddleware_TLSNoVerifiedChains(t *testing.T) {
	handler := RequireClientCertMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// TLS set but no verified chains (unverified client cert)
	req.TLS = &tls.ConnectionState{
		VerifiedChains: nil,
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", rr.Code)
	}
}

func TestIntegration_RequireClientCertMiddleware_TLSWithVerifiedChains(t *testing.T) {
	handler := RequireClientCertMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// TLS set with a verified chain (simulated valid client cert)
	req.TLS = &tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{}},
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
}
