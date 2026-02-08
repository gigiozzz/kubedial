package executor

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
	"github.com/rs/zerolog/log"
)

// Executor defines the interface for manifest execution
type Executor interface {
	// Execute executes a command with its manifest files
	Execute(ctx context.Context, cmd *models.Command, files map[string][]byte) (*models.CommandResult, error)
}

// manifestExecutor implements Executor
type manifestExecutor struct {
	applyer Applyer
}

// NewManifestExecutor creates a new ManifestExecutor
func NewManifestExecutor(applyer Applyer) Executor {
	return &manifestExecutor{applyer: applyer}
}

// Execute executes a command with its manifest files
func (e *manifestExecutor) Execute(ctx context.Context, cmd *models.Command, files map[string][]byte) (*models.CommandResult, error) {
	result := &models.CommandResult{
		CommandID:  cmd.ID,
		ExecutedAt: time.Now(),
	}

	// Concatenate all manifest files
	var manifests bytes.Buffer
	for filename, content := range files {
		log.Debug().Str("filename", filename).Msg("processing manifest file")
		manifests.Write(content)
		manifests.WriteString("\n---\n")
	}

	var output string
	var err error

	switch cmd.OperationType {
	case models.OperationTypeApply:
		opts := ApplyOptions{
			Namespace:  cmd.Namespace,
			ServerSide: cmd.ServerSide,
			DryRun:     cmd.DryRun,
			Force:      cmd.Force,
		}
		output, err = e.applyer.Apply(ctx, manifests.Bytes(), opts)

	case models.OperationTypeDelete:
		opts := DeleteOptions{
			Namespace: cmd.Namespace,
			Force:     cmd.Force,
		}
		output, err = e.applyer.Delete(ctx, manifests.Bytes(), opts)

	default:
		err = fmt.Errorf("unknown operation type: %s", cmd.OperationType)
	}

	result.Output = output
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	return result, nil
}
