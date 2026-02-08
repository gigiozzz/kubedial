package repository

import (
	"context"
	"testing"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "test-ns"

func newFakeCommandRepo() CommandRepository {
	return NewCommandRepository(fake.NewSimpleClientset(), testNamespace)
}

func TestCommandRepository_Create(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	cmd := &models.Command{
		ID:            "cmd-001",
		AgentID:       "agent-001",
		OperationType: models.OperationTypeApply,
		Namespace:     "default",
		Status:        models.CommandStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	files := map[string][]byte{
		"deployment.yaml": []byte("apiVersion: apps/v1\nkind: Deployment"),
		"service.yaml":    []byte("apiVersion: v1\nkind: Service"),
	}

	if err := repo.Create(ctx, cmd, files); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify command can be retrieved
	got, err := repo.Get(ctx, "cmd-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected command, got nil")
	}
	if got.ID != "cmd-001" {
		t.Errorf("expected ID cmd-001, got %s", got.ID)
	}
	if got.AgentID != "agent-001" {
		t.Errorf("expected AgentID agent-001, got %s", got.AgentID)
	}
}

func TestCommandRepository_Create_WithoutFiles(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	cmd := &models.Command{
		ID:            "cmd-002",
		AgentID:       "agent-001",
		OperationType: models.OperationTypeApply,
		Status:        models.CommandStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := repo.Create(ctx, cmd, nil); err != nil {
		t.Fatalf("Create without files failed: %v", err)
	}

	got, err := repo.Get(ctx, "cmd-002")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected command, got nil")
	}
}

func TestCommandRepository_Get_NotFound(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	got, err := repo.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestCommandRepository_List(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	for i, id := range []string{"cmd-a", "cmd-b", "cmd-c"} {
		cmd := &models.Command{
			ID:            id,
			AgentID:       "agent-001",
			OperationType: models.OperationTypeApply,
			Status:        models.CommandStatusPending,
			CreatedAt:     time.Now().Add(time.Duration(i) * time.Minute),
			UpdatedAt:     time.Now(),
		}
		if err := repo.Create(ctx, cmd, nil); err != nil {
			t.Fatalf("Create %s failed: %v", id, err)
		}
	}

	commands, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(commands) != 3 {
		t.Errorf("expected 3 commands, got %d", len(commands))
	}
}

func TestCommandRepository_GetPending(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	// Create pending command for agent-001
	cmd1 := &models.Command{
		ID:            "cmd-pending",
		AgentID:       "agent-001",
		OperationType: models.OperationTypeApply,
		Status:        models.CommandStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := repo.Create(ctx, cmd1, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create completed command for agent-001
	cmd2 := &models.Command{
		ID:            "cmd-completed",
		AgentID:       "agent-001",
		OperationType: models.OperationTypeApply,
		Status:        models.CommandStatusCompleted,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := repo.Create(ctx, cmd2, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create pending command for agent-002
	cmd3 := &models.Command{
		ID:            "cmd-other",
		AgentID:       "agent-002",
		OperationType: models.OperationTypeApply,
		Status:        models.CommandStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := repo.Create(ctx, cmd3, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should only return pending for agent-001
	pending, err := repo.GetPending(ctx, "agent-001")
	if err != nil {
		t.Fatalf("GetPending failed: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}
	if len(pending) > 0 && pending[0].ID != "cmd-pending" {
		t.Errorf("expected cmd-pending, got %s", pending[0].ID)
	}
}

func TestCommandRepository_UpdateStatus(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	cmd := &models.Command{
		ID:            "cmd-status",
		AgentID:       "agent-001",
		OperationType: models.OperationTypeApply,
		Status:        models.CommandStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := repo.Create(ctx, cmd, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.UpdateStatus(ctx, "cmd-status", models.CommandStatusRunning); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, err := repo.Get(ctx, "cmd-status")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Status != models.CommandStatusRunning {
		t.Errorf("expected status running, got %s", got.Status)
	}

	// Verify label was also updated
	client := fake.NewSimpleClientset()
	repo2 := NewCommandRepository(client, testNamespace)

	cmd2 := &models.Command{
		ID:      "cmd-label",
		AgentID: "agent-001",
		Status:  models.CommandStatusPending,
	}
	_ = repo2.Create(ctx, cmd2, nil)
	_ = repo2.UpdateStatus(ctx, "cmd-label", models.CommandStatusCompleted)

	secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "cmd-cmd-label", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if secret.Labels[labelStatus] != string(models.CommandStatusCompleted) {
		t.Errorf("expected label status completed, got %s", secret.Labels[labelStatus])
	}
}

func TestCommandRepository_UpdateStatus_NotFound(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	err := repo.UpdateStatus(ctx, "nonexistent", models.CommandStatusRunning)
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
}

func TestCommandRepository_GetFile(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	files := map[string][]byte{
		"deploy.yaml": []byte("apiVersion: apps/v1"),
	}

	cmd := &models.Command{
		ID:      "cmd-files",
		AgentID: "agent-001",
		Status:  models.CommandStatusPending,
	}
	if err := repo.Create(ctx, cmd, files); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	data, err := repo.GetFile(ctx, "cmd-files", "deploy.yaml")
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}
	if string(data) != "apiVersion: apps/v1" {
		t.Errorf("unexpected file content: %s", string(data))
	}

	// Non-existent file
	data, err = repo.GetFile(ctx, "cmd-files", "missing.yaml")
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil for missing file, got %v", data)
	}
}

func TestCommandRepository_GetFile_NoFilesSecret(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	data, err := repo.GetFile(ctx, "nonexistent", "file.yaml")
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil, got %v", data)
	}
}

func TestCommandRepository_ListFiles(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	files := map[string][]byte{
		"a.yaml": []byte("a"),
		"b.yaml": []byte("b"),
	}

	cmd := &models.Command{
		ID:      "cmd-listfiles",
		AgentID: "agent-001",
		Status:  models.CommandStatusPending,
	}
	if err := repo.Create(ctx, cmd, files); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	filenames, err := repo.ListFiles(ctx, "cmd-listfiles")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(filenames) != 2 {
		t.Errorf("expected 2 files, got %d", len(filenames))
	}
}

func TestCommandRepository_ListFiles_NoSecret(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	filenames, err := repo.ListFiles(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	if len(filenames) != 0 {
		t.Errorf("expected 0 files, got %d", len(filenames))
	}
}

func TestCommandRepository_SaveResult_GetResult(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	cmd := &models.Command{
		ID:      "cmd-result",
		AgentID: "agent-001",
		Status:  models.CommandStatusPending,
	}
	if err := repo.Create(ctx, cmd, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	result := &models.CommandResult{
		CommandID:  "cmd-result",
		Output:     "deployment created",
		Success:    true,
		ExecutedAt: time.Now(),
	}

	if err := repo.SaveResult(ctx, result); err != nil {
		t.Fatalf("SaveResult failed: %v", err)
	}

	got, err := repo.GetResult(ctx, "cmd-result")
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected result, got nil")
	}
	if got.Output != "deployment created" {
		t.Errorf("expected output 'deployment created', got %s", got.Output)
	}
	if !got.Success {
		t.Error("expected success true")
	}
}

func TestCommandRepository_GetResult_NotFound(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	got, err := repo.GetResult(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestCommandRepository_GetResult_NoResultField(t *testing.T) {
	repo := newFakeCommandRepo()
	ctx := context.Background()

	// Create command without result
	cmd := &models.Command{
		ID:      "cmd-noresult",
		AgentID: "agent-001",
		Status:  models.CommandStatusPending,
	}
	if err := repo.Create(ctx, cmd, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetResult(ctx, "cmd-noresult")
	if err != nil {
		t.Fatalf("GetResult failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil result, got %v", got)
	}
}

// Ensure the Secret has correct labels for filtering
func TestCommandRepository_Create_VerifyLabels(t *testing.T) {
	client := fake.NewSimpleClientset()
	repo := NewCommandRepository(client, testNamespace)
	ctx := context.Background()

	cmd := &models.Command{
		ID:      "cmd-labels",
		AgentID: "agent-abc",
		Status:  models.CommandStatusPending,
	}
	if err := repo.Create(ctx, cmd, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	secret, err := client.CoreV1().Secrets(testNamespace).Get(ctx, "cmd-cmd-labels", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}

	if secret.Labels[labelType] != typeCommand {
		t.Errorf("expected label type=%s, got %s", typeCommand, secret.Labels[labelType])
	}
	if secret.Labels[labelAgentID] != "agent-abc" {
		t.Errorf("expected label agent-id=agent-abc, got %s", secret.Labels[labelAgentID])
	}
	if secret.Labels[labelStatus] != string(models.CommandStatusPending) {
		t.Errorf("expected label status=pending, got %s", secret.Labels[labelStatus])
	}
}

// Suppress unused import warning
var _ = corev1.Secret{}
