package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
	"github.com/gigiozzz/kubedial/kubecommander/internal/repository"
	"github.com/google/uuid"
)

// AgentService defines the interface for agent business logic
type AgentService interface {
	Register(ctx context.Context, agent *models.Agent) (*models.Agent, string, error)
	Get(ctx context.Context, id string) (*models.Agent, error)
	List(ctx context.Context) ([]*models.Agent, error)
	UpdateLastSeen(ctx context.Context, id string) error
}

// agentService implements AgentService
type agentService struct {
	repo repository.AgentRepository
}

// NewAgentService creates a new AgentService
func NewAgentService(repo repository.AgentRepository) AgentService {
	return &agentService{repo: repo}
}

// Register registers a new agent, generates a token and returns it
func (s *agentService) Register(ctx context.Context, agent *models.Agent) (*models.Agent, string, error) {
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	agent.LastSeen = time.Now()
	agent.Status = models.AgentStatusOnline

	token, err := generateToken()
	if err != nil {
		return nil, "", err
	}

	if err := s.repo.Create(ctx, agent, token); err != nil {
		return nil, "", err
	}

	return agent, token, nil
}

// generateToken generates a secure random token (32 bytes, base64url encoded)
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Get retrieves an agent by ID
func (s *agentService) Get(ctx context.Context, id string) (*models.Agent, error) {
	return s.repo.Get(ctx, id)
}

// List retrieves all agents
func (s *agentService) List(ctx context.Context) ([]*models.Agent, error) {
	return s.repo.List(ctx)
}

// UpdateLastSeen updates an agent's last seen timestamp
func (s *agentService) UpdateLastSeen(ctx context.Context, id string) error {
	return s.repo.UpdateLastSeen(ctx, id)
}
