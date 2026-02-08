package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// AuthRepository defines the interface for authentication token access
type AuthRepository interface {
	ValidateToken(ctx context.Context, token string) (role string, agentID string, err error)
}

// authRepositoryImpl implements AuthRepository using Secrets
type authRepositoryImpl struct {
	client    kubernetes.Interface
	namespace string
}

// NewAuthRepository creates a new AuthRepository
func NewAuthRepository(c kubernetes.Interface, namespace string) AuthRepository {
	return &authRepositoryImpl{
		client:    c,
		namespace: namespace,
	}
}

// ValidateToken validates a bearer token by checking agent Secrets and users Secret
func (r *authRepositoryImpl) ValidateToken(ctx context.Context, token string) (role string, agentID string, err error) {
	// 1. Check agent Secrets (label kubedial.io/type=agent)
	agentSecrets, err := r.client.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", labelType, typeAgent),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to list agent secrets: %w", err)
	}

	for _, secret := range agentSecrets.Items {
		storedToken := string(secret.Data["token"])
		if storedToken == token {
			// Extract agent ID from secret name (agent-{UUID})
			id := secret.Name[len("agent-"):]
			return "agent", id, nil
		}
	}

	// 2. Check users Secret
	usersSecret, err := r.client.CoreV1().Secrets(r.namespace).Get(ctx, "users", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", "", fmt.Errorf("invalid token")
		}
		return "", "", fmt.Errorf("failed to get users secret: %w", err)
	}

	for _, userData := range usersSecret.Data {
		var user struct {
			Username string `json:"username"`
			Token    string `json:"token"`
		}
		if err := json.Unmarshal(userData, &user); err != nil {
			continue
		}
		if user.Token == token {
			return "admin", "", nil
		}
	}

	return "", "", fmt.Errorf("invalid token")
}
