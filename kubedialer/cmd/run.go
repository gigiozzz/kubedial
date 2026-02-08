package main

import (
	"context"
	"os"
	"time"

	"github.com/gigiozzz/kubedial/common/provider"
	"github.com/gigiozzz/kubedial/common/models"
	"github.com/gigiozzz/kubedial/kubedialer/internal/client"
	"github.com/gigiozzz/kubedial/kubedialer/internal/executor"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a single execution cycle",
	Long: `Fetch pending commands from kubecommander, execute them, and report results.
This command is designed to be run as a CronJob.`,
	RunE: runExecute,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runExecute(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate required flags
	if commanderURL == "" {
		commanderURL = os.Getenv("COMMANDER_URL")
	}
	if agentToken == "" {
		agentToken = os.Getenv("AGENT_TOKEN")
	}
	if agentName == "" {
		agentName = os.Getenv("AGENT_NAME")
	}
	if clusterName == "" {
		clusterName = os.Getenv("CLUSTER_NAME")
	}

	if commanderURL == "" {
		log.Fatal().Msg("commander-url is required")
	}
	if agentToken == "" {
		log.Fatal().Msg("agent-token is required")
	}
	if agentName == "" {
		log.Fatal().Msg("agent-name is required")
	}

	log.Info().
		Str("commander-url", commanderURL).
		Str("agent-name", agentName).
		Msg("starting kubedialer run")

	// Create commander client
	commanderClient := client.NewCommanderClient(commanderURL, agentToken)

	// Register agent
	agent := &models.Agent{
		Name:        agentName,
		ClusterName: clusterName,
		LastSeen:    time.Now(),
		Status:      models.AgentStatusOnline,
	}

	registered, token, err := commanderClient.RegisterAgent(ctx, agent)
	if err != nil {
		log.Error().Err(err).Msg("failed to register agent")
		return err
	}
	log.Info().Str("id", registered.ID).Msg("agent registered")

	if token != "" {
		log.Info().Msg("received agent token from registration")
	}

	// Get pending commands
	commands, err := commanderClient.GetPendingCommands(ctx, registered.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get pending commands")
		return err
	}

	if len(commands) == 0 {
		log.Info().Msg("no pending commands")
		return nil
	}

	log.Info().Int("count", len(commands)).Msg("found pending commands")

	// Create executor
	k8sConfig, err := provider.GetConfig()
	if err != nil {
		log.Error().Err(err).Msg("failed to get kubernetes config")
		return err
	}

	applyer, err := executor.NewK8sApplyer(k8sConfig)
	if err != nil {
		log.Error().Err(err).Msg("failed to create applyer")
		return err
	}

	exec := executor.NewManifestExecutor(applyer)

	// Process each command
	for _, command := range commands {
		log.Info().
			Str("id", command.ID).
			Str("operationType", string(command.OperationType)).
			Msg("processing command")

		// Download files
		files := make(map[string][]byte)
		for _, filename := range command.Filenames {
			content, err := commanderClient.GetCommandFile(ctx, command.ID, filename)
			if err != nil {
				log.Error().Err(err).
					Str("commandId", command.ID).
					Str("filename", filename).
					Msg("failed to download file")
				continue
			}
			files[filename] = content
		}

		// Execute command
		result, err := exec.Execute(ctx, command, files)
		if err != nil {
			log.Error().Err(err).Str("id", command.ID).Msg("execution error")
		}

		// Submit result
		if err := commanderClient.SubmitResult(ctx, command.ID, result); err != nil {
			log.Error().Err(err).Str("id", command.ID).Msg("failed to submit result")
		} else {
			log.Info().
				Str("id", command.ID).
				Bool("success", result.Success).
				Msg("result submitted")
		}
	}

	log.Info().Msg("run completed")
	return nil
}
