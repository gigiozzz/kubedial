package repository

import (
	"context"
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAuthRepository_ValidateToken_AgentToken(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	// Create agent secret with token
	agentSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-agent-001",
			Namespace: testNamespace,
			Labels: map[string]string{
				labelType: typeAgent,
			},
		},
		Data: map[string][]byte{
			"metadata": []byte(`{"id":"agent-001","name":"test-agent"}`),
			"token":    []byte("agent-token-abc"),
		},
	}
	if _, err := client.CoreV1().Secrets(testNamespace).Create(ctx, agentSecret, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create agent secret: %v", err)
	}

	repo := NewAuthRepository(client, testNamespace)

	role, agentID, err := repo.ValidateToken(ctx, "agent-token-abc")
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if role != "agent" {
		t.Errorf("expected role 'agent', got '%s'", role)
	}
	if agentID != "agent-001" {
		t.Errorf("expected agentID 'agent-001', got '%s'", agentID)
	}
}

func TestAuthRepository_ValidateToken_UserToken(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	// Create users secret
	userData, _ := json.Marshal(map[string]string{
		"username": "admin-user",
		"token":    "user-token-xyz",
	})

	usersSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "users",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"admin1": userData,
		},
	}
	if _, err := client.CoreV1().Secrets(testNamespace).Create(ctx, usersSecret, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create users secret: %v", err)
	}

	repo := NewAuthRepository(client, testNamespace)

	role, agentID, err := repo.ValidateToken(ctx, "user-token-xyz")
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if role != "admin" {
		t.Errorf("expected role 'admin', got '%s'", role)
	}
	if agentID != "" {
		t.Errorf("expected empty agentID, got '%s'", agentID)
	}
}

func TestAuthRepository_ValidateToken_InvalidToken(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	repo := NewAuthRepository(client, testNamespace)

	_, _, err := repo.ValidateToken(ctx, "invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestAuthRepository_ValidateToken_AgentPriority(t *testing.T) {
	client := fake.NewSimpleClientset()
	ctx := context.Background()

	// Create agent secret
	agentSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-agent-002",
			Namespace: testNamespace,
			Labels: map[string]string{
				labelType: typeAgent,
			},
		},
		Data: map[string][]byte{
			"metadata": []byte(`{"id":"agent-002","name":"test-agent"}`),
			"token":    []byte("shared-token"),
		},
	}
	if _, err := client.CoreV1().Secrets(testNamespace).Create(ctx, agentSecret, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create agent secret: %v", err)
	}

	// Create users secret with same token
	userData, _ := json.Marshal(map[string]string{
		"username": "admin",
		"token":    "shared-token",
	})
	usersSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "users",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"admin1": userData,
		},
	}
	if _, err := client.CoreV1().Secrets(testNamespace).Create(ctx, usersSecret, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create users secret: %v", err)
	}

	repo := NewAuthRepository(client, testNamespace)

	// Agent tokens are checked first
	role, agentID, err := repo.ValidateToken(ctx, "shared-token")
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if role != "agent" {
		t.Errorf("expected role 'agent' (agent priority), got '%s'", role)
	}
	if agentID != "agent-002" {
		t.Errorf("expected agentID 'agent-002', got '%s'", agentID)
	}
}
