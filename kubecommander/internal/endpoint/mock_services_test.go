package endpoint

import (
	"context"

	"github.com/gigiozzz/kubedial/common/models"
)

// mockCommandService is a mock implementation of service.CommandService
type mockCommandService struct {
	CreateFunc      func(ctx context.Context, cmd *models.Command, files map[string][]byte) (*models.Command, error)
	GetFunc         func(ctx context.Context, id string) (*models.Command, error)
	ListFunc        func(ctx context.Context) ([]*models.Command, error)
	GetPendingFunc  func(ctx context.Context, agentID string) ([]*models.Command, error)
	GetFileFunc     func(ctx context.Context, commandID, filename string) ([]byte, error)
	ListFilesFunc   func(ctx context.Context, commandID string) ([]string, error)
	UpdateResultFunc func(ctx context.Context, commandID string, result *models.CommandResult) error
	GetWithResultFunc func(ctx context.Context, id string) (*models.Command, *models.CommandResult, error)
}

func (m *mockCommandService) Create(ctx context.Context, cmd *models.Command, files map[string][]byte) (*models.Command, error) {
	return m.CreateFunc(ctx, cmd, files)
}

func (m *mockCommandService) Get(ctx context.Context, id string) (*models.Command, error) {
	return m.GetFunc(ctx, id)
}

func (m *mockCommandService) List(ctx context.Context) ([]*models.Command, error) {
	return m.ListFunc(ctx)
}

func (m *mockCommandService) GetPending(ctx context.Context, agentID string) ([]*models.Command, error) {
	return m.GetPendingFunc(ctx, agentID)
}

func (m *mockCommandService) GetFile(ctx context.Context, commandID, filename string) ([]byte, error) {
	return m.GetFileFunc(ctx, commandID, filename)
}

func (m *mockCommandService) ListFiles(ctx context.Context, commandID string) ([]string, error) {
	return m.ListFilesFunc(ctx, commandID)
}

func (m *mockCommandService) UpdateResult(ctx context.Context, commandID string, result *models.CommandResult) error {
	return m.UpdateResultFunc(ctx, commandID, result)
}

func (m *mockCommandService) GetWithResult(ctx context.Context, id string) (*models.Command, *models.CommandResult, error) {
	return m.GetWithResultFunc(ctx, id)
}

// mockAgentService is a mock implementation of service.AgentService
type mockAgentService struct {
	RegisterFunc     func(ctx context.Context, agent *models.Agent) (*models.Agent, string, error)
	GetFunc          func(ctx context.Context, id string) (*models.Agent, error)
	ListFunc         func(ctx context.Context) ([]*models.Agent, error)
	UpdateLastSeenFunc func(ctx context.Context, id string) error
}

func (m *mockAgentService) Register(ctx context.Context, agent *models.Agent) (*models.Agent, string, error) {
	return m.RegisterFunc(ctx, agent)
}

func (m *mockAgentService) Get(ctx context.Context, id string) (*models.Agent, error) {
	return m.GetFunc(ctx, id)
}

func (m *mockAgentService) List(ctx context.Context) ([]*models.Agent, error) {
	return m.ListFunc(ctx)
}

func (m *mockAgentService) UpdateLastSeen(ctx context.Context, id string) error {
	return m.UpdateLastSeenFunc(ctx, id)
}

// mockAuthService is a mock implementation of service.AuthService
type mockAuthService struct {
	ValidateTokenFunc func(ctx context.Context, token string) (string, string, error)
}

func (m *mockAuthService) ValidateToken(ctx context.Context, token string) (string, string, error) {
	return m.ValidateTokenFunc(ctx, token)
}
