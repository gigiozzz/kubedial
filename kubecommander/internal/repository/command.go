package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	labelType      = "kubedial.io/type"
	labelCommandID = "kubedial.io/command-id"
	labelAgentID   = "kubedial.io/agent-id"
	labelStatus    = "kubedial.io/status"

	typeCommand      = "command"
	typeCommandFiles = "command-files"
	typeAgent        = "agent"
	typeUsers        = "users"
)

// CommandRepository defines the interface for command data access
type CommandRepository interface {
	Create(ctx context.Context, cmd *models.Command, files map[string][]byte) error
	Get(ctx context.Context, id string) (*models.Command, error)
	List(ctx context.Context) ([]*models.Command, error)
	GetPending(ctx context.Context, agentID string) ([]*models.Command, error)
	UpdateStatus(ctx context.Context, id string, status models.CommandStatus) error
	GetFile(ctx context.Context, commandID, filename string) ([]byte, error)
	ListFiles(ctx context.Context, commandID string) ([]string, error)
	SaveResult(ctx context.Context, result *models.CommandResult) error
	GetResult(ctx context.Context, commandID string) (*models.CommandResult, error)
}

// commandRepositoryImpl implements CommandRepository using Secrets
type commandRepositoryImpl struct {
	client    kubernetes.Interface
	namespace string
}

// NewCommandRepository creates a new CommandRepository
func NewCommandRepository(c kubernetes.Interface, namespace string) CommandRepository {
	return &commandRepositoryImpl{
		client:    c,
		namespace: namespace,
	}
}

// Create creates a new command with its files
func (r *commandRepositoryImpl) Create(ctx context.Context, cmd *models.Command, files map[string][]byte) error {
	cmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	cmdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("cmd-%s", cmd.ID),
			Namespace: r.namespace,
			Labels: map[string]string{
				labelType:    typeCommand,
				labelAgentID: cmd.AgentID,
				labelStatus:  string(cmd.Status),
			},
		},
		Data: map[string][]byte{
			"metadata": cmdData,
		},
	}

	if _, err := r.client.CoreV1().Secrets(r.namespace).Create(ctx, cmdSecret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create command secret: %w", err)
	}

	// Create files Secret
	if len(files) > 0 {
		filesSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cmd-%s-files", cmd.ID),
				Namespace: r.namespace,
				Labels: map[string]string{
					labelType:      typeCommandFiles,
					labelCommandID: cmd.ID,
				},
			},
			Data: files,
		}

		if _, err := r.client.CoreV1().Secrets(r.namespace).Create(ctx, filesSecret, metav1.CreateOptions{}); err != nil {
			_ = r.client.CoreV1().Secrets(r.namespace).Delete(ctx, cmdSecret.Name, metav1.DeleteOptions{})
			return fmt.Errorf("failed to create files secret: %w", err)
		}
	}

	return nil
}

// Get retrieves a command by ID
func (r *commandRepositoryImpl) Get(ctx context.Context, id string) (*models.Command, error) {
	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("cmd-%s", id), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get command secret: %w", err)
	}

	var cmd models.Command
	if err := json.Unmarshal(secret.Data["metadata"], &cmd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command: %w", err)
	}

	return &cmd, nil
}

// List retrieves all commands
func (r *commandRepositoryImpl) List(ctx context.Context) ([]*models.Command, error) {
	secretList, err := r.client.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labelType, typeCommand),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list command secrets: %w", err)
	}

	commands := make([]*models.Command, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var cmd models.Command
		if err := json.Unmarshal(secret.Data["metadata"], &cmd); err != nil {
			continue
		}
		commands = append(commands, &cmd)
	}

	return commands, nil
}

// GetPending retrieves pending commands for a specific agent using label selectors
func (r *commandRepositoryImpl) GetPending(ctx context.Context, agentID string) ([]*models.Command, error) {
	selector := fmt.Sprintf("%s=%s,%s=%s,%s=%s",
		labelType, typeCommand,
		labelAgentID, agentID,
		labelStatus, string(models.CommandStatusPending),
	)

	secretList, err := r.client.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pending command secrets: %w", err)
	}

	commands := make([]*models.Command, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var cmd models.Command
		if err := json.Unmarshal(secret.Data["metadata"], &cmd); err != nil {
			continue
		}
		commands = append(commands, &cmd)
	}

	return commands, nil
}

// UpdateStatus updates a command's status (both metadata and label)
func (r *commandRepositoryImpl) UpdateStatus(ctx context.Context, id string, status models.CommandStatus) error {
	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("cmd-%s", id), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("command not found: %s", id)
		}
		return fmt.Errorf("failed to get command secret: %w", err)
	}

	var cmd models.Command
	if err := json.Unmarshal(secret.Data["metadata"], &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	cmd.Status = status
	cmd.UpdatedAt = time.Now()

	cmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	secret.Data["metadata"] = cmdData
	secret.Labels[labelStatus] = string(status)

	if _, err := r.client.CoreV1().Secrets(r.namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update command secret: %w", err)
	}

	return nil
}

// GetFile retrieves a file from a command
func (r *commandRepositoryImpl) GetFile(ctx context.Context, commandID, filename string) ([]byte, error) {
	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("cmd-%s-files", commandID), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get files secret: %w", err)
	}

	data, ok := secret.Data[filename]
	if !ok {
		return nil, nil
	}

	return data, nil
}

// ListFiles lists all files in a command
func (r *commandRepositoryImpl) ListFiles(ctx context.Context, commandID string) ([]string, error) {
	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("cmd-%s-files", commandID), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get files secret: %w", err)
	}

	files := make([]string, 0, len(secret.Data))
	for name := range secret.Data {
		files = append(files, name)
	}

	return files, nil
}

// SaveResult saves a command result in the command Secret itself
func (r *commandRepositoryImpl) SaveResult(ctx context.Context, result *models.CommandResult) error {
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("cmd-%s", result.CommandID), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get command secret: %w", err)
	}

	secret.Data["result"] = resultData

	if _, err := r.client.CoreV1().Secrets(r.namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update command secret with result: %w", err)
	}

	return nil
}

// GetResult retrieves a command result from the command Secret
func (r *commandRepositoryImpl) GetResult(ctx context.Context, commandID string) (*models.CommandResult, error) {
	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("cmd-%s", commandID), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get command secret: %w", err)
	}

	resultData, ok := secret.Data["result"]
	if !ok {
		return nil, nil
	}

	var result models.CommandResult
	if err := json.Unmarshal(resultData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}
