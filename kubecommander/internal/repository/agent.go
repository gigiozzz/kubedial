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

// AgentRepository defines the interface for agent data access
type AgentRepository interface {
	Create(ctx context.Context, agent *models.Agent, token string) error
	Get(ctx context.Context, id string) (*models.Agent, error)
	List(ctx context.Context) ([]*models.Agent, error)
	UpdateLastSeen(ctx context.Context, id string) error
}

// agentRepositoryImpl implements AgentRepository using Secrets
type agentRepositoryImpl struct {
	client    kubernetes.Interface
	namespace string
}

// NewAgentRepository creates a new AgentRepository
func NewAgentRepository(c kubernetes.Interface, namespace string) AgentRepository {
	return &agentRepositoryImpl{
		client:    c,
		namespace: namespace,
	}
}

// Create creates a new agent with its token
func (r *agentRepositoryImpl) Create(ctx context.Context, agent *models.Agent, token string) error {
	agentData, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("agent-%s", agent.ID),
			Namespace: r.namespace,
			Labels: map[string]string{
				labelType: typeAgent,
			},
		},
		Data: map[string][]byte{
			"metadata": agentData,
			"token":    []byte(token),
		},
	}

	if _, err := r.client.CoreV1().Secrets(r.namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			return r.update(ctx, agent)
		}
		return fmt.Errorf("failed to create agent secret: %w", err)
	}

	return nil
}

func (r *agentRepositoryImpl) update(ctx context.Context, agent *models.Agent) error {
	agentData, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent: %w", err)
	}

	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("agent-%s", agent.ID), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get agent secret: %w", err)
	}

	// Preserve the token field during updates
	secret.Data["metadata"] = agentData
	if _, err := r.client.CoreV1().Secrets(r.namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update agent secret: %w", err)
	}

	return nil
}

// Get retrieves an agent by ID
func (r *agentRepositoryImpl) Get(ctx context.Context, id string) (*models.Agent, error) {
	secret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, fmt.Sprintf("agent-%s", id), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get agent secret: %w", err)
	}

	var agent models.Agent
	if err := json.Unmarshal(secret.Data["metadata"], &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	return &agent, nil
}

// List retrieves all agents
func (r *agentRepositoryImpl) List(ctx context.Context) ([]*models.Agent, error) {
	secretList, err := r.client.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labelType, typeAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list agent secrets: %w", err)
	}

	agents := make([]*models.Agent, 0, len(secretList.Items))
	for _, secret := range secretList.Items {
		var agent models.Agent
		if err := json.Unmarshal(secret.Data["metadata"], &agent); err != nil {
			continue
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// UpdateLastSeen updates an agent's last seen timestamp
func (r *agentRepositoryImpl) UpdateLastSeen(ctx context.Context, id string) error {
	agent, err := r.Get(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.LastSeen = time.Now()
	agent.Status = models.AgentStatusOnline

	return r.update(ctx, agent)
}
