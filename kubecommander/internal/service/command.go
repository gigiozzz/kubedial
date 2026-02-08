package service

import (
	"context"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
	"github.com/gigiozzz/kubedial/kubecommander/internal/repository"
	"github.com/google/uuid"
)

// CommandService defines the interface for command business logic
type CommandService interface {
	Create(ctx context.Context, cmd *models.Command, files map[string][]byte) (*models.Command, error)
	Get(ctx context.Context, id string) (*models.Command, error)
	List(ctx context.Context) ([]*models.Command, error)
	GetPending(ctx context.Context, agentID string) ([]*models.Command, error)
	GetFile(ctx context.Context, commandID, filename string) ([]byte, error)
	ListFiles(ctx context.Context, commandID string) ([]string, error)
	UpdateResult(ctx context.Context, commandID string, result *models.CommandResult) error
	GetWithResult(ctx context.Context, id string) (*models.Command, *models.CommandResult, error)
}

// commandService implements CommandService
type commandService struct {
	repo repository.CommandRepository
}

// NewCommandService creates a new CommandService
func NewCommandService(repo repository.CommandRepository) CommandService {
	return &commandService{repo: repo}
}

// Create creates a new command
func (s *commandService) Create(ctx context.Context, cmd *models.Command, files map[string][]byte) (*models.Command, error) {
	cmd.ID = uuid.New().String()
	cmd.Status = models.CommandStatusPending
	cmd.CreatedAt = time.Now()
	cmd.UpdatedAt = time.Now()

	// Extract filenames from the files map
	filenames := make([]string, 0, len(files))
	for name := range files {
		filenames = append(filenames, name)
	}
	cmd.Filenames = filenames

	if err := s.repo.Create(ctx, cmd, files); err != nil {
		return nil, err
	}

	return cmd, nil
}

// Get retrieves a command by ID
func (s *commandService) Get(ctx context.Context, id string) (*models.Command, error) {
	return s.repo.Get(ctx, id)
}

// List retrieves all commands
func (s *commandService) List(ctx context.Context) ([]*models.Command, error) {
	return s.repo.List(ctx)
}

// GetPending retrieves pending commands for an agent
func (s *commandService) GetPending(ctx context.Context, agentID string) ([]*models.Command, error) {
	return s.repo.GetPending(ctx, agentID)
}

// GetFile retrieves a file from a command
func (s *commandService) GetFile(ctx context.Context, commandID, filename string) ([]byte, error) {
	return s.repo.GetFile(ctx, commandID, filename)
}

// ListFiles lists all files in a command
func (s *commandService) ListFiles(ctx context.Context, commandID string) ([]string, error) {
	return s.repo.ListFiles(ctx, commandID)
}

// UpdateResult updates a command with its result
func (s *commandService) UpdateResult(ctx context.Context, commandID string, result *models.CommandResult) error {
	result.CommandID = commandID
	result.ExecutedAt = time.Now()

	// Update command status based on result
	status := models.CommandStatusCompleted
	if !result.Success {
		status = models.CommandStatusFailed
	}

	if err := s.repo.UpdateStatus(ctx, commandID, status); err != nil {
		return err
	}

	return s.repo.SaveResult(ctx, result)
}

// GetWithResult retrieves a command with its result
func (s *commandService) GetWithResult(ctx context.Context, id string) (*models.Command, *models.CommandResult, error) {
	cmd, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if cmd == nil {
		return nil, nil, nil
	}

	result, err := s.repo.GetResult(ctx, id)
	if err != nil {
		return cmd, nil, err
	}

	return cmd, result, nil
}
