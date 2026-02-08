package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newFakeAgentRepo() AgentRepository {
	return NewAgentRepository(fake.NewSimpleClientset(), testNamespace)
}

func TestAgentRepository_Create(t *testing.T) {
	client := fake.NewSimpleClientset()
	repo := NewAgentRepository(client, testNamespace)
	ctx := context.Background()

	agent := &models.Agent{
		ID:          "agent-001",
		Name:        "test-agent",
		ClusterName: "test-cluster",
		ClusterID:   "cluster-uuid",
		LastSeen:    time.Now(),
		Status:      models.AgentStatusOnline,
	}

	if err := repo.Create(ctx, agent, "test-token-123"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify secret was created with metadata and token
	secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "agent-agent-001", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}

	if string(secret.Data["token"]) != "test-token-123" {
		t.Errorf("expected token 'test-token-123', got '%s'", string(secret.Data["token"]))
	}

	var stored models.Agent
	if err := json.Unmarshal(secret.Data["metadata"], &stored); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}
	if stored.ClusterName != "test-cluster" {
		t.Errorf("expected ClusterName 'test-cluster', got '%s'", stored.ClusterName)
	}
}

func TestAgentRepository_Create_AlreadyExists(t *testing.T) {
	repo := newFakeAgentRepo()
	ctx := context.Background()

	agent := &models.Agent{
		ID:       "agent-dup",
		Name:     "original",
		LastSeen: time.Now(),
		Status:   models.AgentStatusOnline,
	}

	if err := repo.Create(ctx, agent, "token1"); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	// Second create should update (upsert)
	agent.Name = "updated"
	if err := repo.Create(ctx, agent, "token2"); err != nil {
		t.Fatalf("second Create failed: %v", err)
	}

	got, err := repo.Get(ctx, "agent-dup")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "updated" {
		t.Errorf("expected name 'updated', got '%s'", got.Name)
	}
}

func TestAgentRepository_Get(t *testing.T) {
	repo := newFakeAgentRepo()
	ctx := context.Background()

	agent := &models.Agent{
		ID:          "agent-get",
		Name:        "get-agent",
		ClusterName: "my-cluster",
		LastSeen:    time.Now(),
		Status:      models.AgentStatusOnline,
	}
	_ = repo.Create(ctx, agent, "token")

	got, err := repo.Get(ctx, "agent-get")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected agent, got nil")
	}
	if got.Name != "get-agent" {
		t.Errorf("expected name 'get-agent', got '%s'", got.Name)
	}
	if got.ClusterName != "my-cluster" {
		t.Errorf("expected ClusterName 'my-cluster', got '%s'", got.ClusterName)
	}
}

func TestAgentRepository_Get_NotFound(t *testing.T) {
	repo := newFakeAgentRepo()
	ctx := context.Background()

	got, err := repo.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestAgentRepository_List(t *testing.T) {
	repo := newFakeAgentRepo()
	ctx := context.Background()

	for _, id := range []string{"a1", "a2", "a3"} {
		agent := &models.Agent{
			ID:       id,
			Name:     "agent-" + id,
			LastSeen: time.Now(),
			Status:   models.AgentStatusOnline,
		}
		if err := repo.Create(ctx, agent, "token-"+id); err != nil {
			t.Fatalf("Create %s failed: %v", id, err)
		}
	}

	agents, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(agents) != 3 {
		t.Errorf("expected 3 agents, got %d", len(agents))
	}
}

func TestAgentRepository_UpdateLastSeen(t *testing.T) {
	repo := newFakeAgentRepo()
	ctx := context.Background()

	before := time.Now().Add(-1 * time.Hour)
	agent := &models.Agent{
		ID:       "agent-lastseen",
		Name:     "test",
		LastSeen: before,
		Status:   models.AgentStatusOffline,
	}
	_ = repo.Create(ctx, agent, "token")

	if err := repo.UpdateLastSeen(ctx, "agent-lastseen"); err != nil {
		t.Fatalf("UpdateLastSeen failed: %v", err)
	}

	got, _ := repo.Get(ctx, "agent-lastseen")
	if got.Status != models.AgentStatusOnline {
		t.Errorf("expected status online, got %s", got.Status)
	}
	if !got.LastSeen.After(before) {
		t.Error("expected LastSeen to be updated")
	}
}

func TestAgentRepository_UpdateLastSeen_NotFound(t *testing.T) {
	repo := newFakeAgentRepo()
	ctx := context.Background()

	err := repo.UpdateLastSeen(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestAgentRepository_Create_PreservesTokenOnUpdate(t *testing.T) {
	client := fake.NewSimpleClientset()
	repo := NewAgentRepository(client, testNamespace)
	ctx := context.Background()

	agent := &models.Agent{
		ID:       "agent-token",
		Name:     "test",
		LastSeen: time.Now(),
		Status:   models.AgentStatusOnline,
	}

	_ = repo.Create(ctx, agent, "original-token")

	// Update via UpdateLastSeen (uses internal update which preserves token)
	_ = repo.UpdateLastSeen(ctx, "agent-token")

	secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "agent-agent-token", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}

	if string(secret.Data["token"]) != "original-token" {
		t.Errorf("token was not preserved, got '%s'", string(secret.Data["token"]))
	}
}
