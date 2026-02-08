package service

import (
	"context"

	"github.com/gigiozzz/kubedial/kubecommander/internal/repository"
)

// AuthService defines the interface for authentication
type AuthService interface {
	ValidateToken(ctx context.Context, token string) (role string, agentID string, err error)
}

// authService implements AuthService
type authService struct {
	repo repository.AuthRepository
}

// NewAuthService creates a new AuthService
func NewAuthService(repo repository.AuthRepository) AuthService {
	return &authService{repo: repo}
}

// ValidateToken validates a bearer token
func (s *authService) ValidateToken(ctx context.Context, token string) (role string, agentID string, err error) {
	return s.repo.ValidateToken(ctx, token)
}
