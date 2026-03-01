package main

import (
	"context"
	"os"

	"github.com/gigiozzz/kubedial/common/provider"
	"github.com/gigiozzz/kubedial/kubedialer/internal/client"
	"github.com/gigiozzz/kubedial/kubedialer/internal/executor"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	commandID string
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a specific command by ID",
	Long:  `Fetch a specific command from kubecommander and apply it.`,
	RunE:  applyExecute,
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVar(&commandID, "command-id", "", "Command ID to apply")
	cobra.CheckErr(applyCmd.MarkFlagRequired("command-id"))
}

func applyExecute(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate required flags
	if commanderURL == "" {
		commanderURL = os.Getenv("COMMANDER_URL")
	}
	if agentToken == "" {
		agentToken = os.Getenv("AGENT_TOKEN")
	}
	if tlsCAFile == "" {
		tlsCAFile = os.Getenv("TLS_CA_FILE")
	}
	if tlsClientCertFile == "" {
		tlsClientCertFile = os.Getenv("TLS_CLIENT_CERT_FILE")
	}
	if tlsClientKeyFile == "" {
		tlsClientKeyFile = os.Getenv("TLS_CLIENT_KEY_FILE")
	}

	if commanderURL == "" {
		log.Fatal().Msg("commander-url is required")
	}
	if agentToken == "" {
		log.Fatal().Msg("agent-token is required")
	}
	if commandID == "" {
		log.Fatal().Msg("command-id is required")
	}

	log.Info().
		Str("commander-url", commanderURL).
		Str("command-id", commandID).
		Msg("applying command")

	// Build TLS options if CA file is configured
	var tlsOpts *client.TLSOptions
	if tlsCAFile != "" {
		tlsOpts = &client.TLSOptions{
			CAFile:     tlsCAFile,
			ClientCert: tlsClientCertFile,
			ClientKey:  tlsClientKeyFile,
		}
	}

	// Create commander client
	commanderClient, err := client.NewCommanderClient(commanderURL, agentToken, tlsOpts)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create commander client")
	}

	// Get command
	command, err := commanderClient.GetCommand(ctx, commandID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get command")
		return err
	}

	log.Info().
		Str("operationType", string(command.OperationType)).
		Strs("filenames", command.Filenames).
		Msg("command retrieved")

	// Download files
	files := make(map[string][]byte)
	for _, filename := range command.Filenames {
		content, err := commanderClient.GetCommandFile(ctx, commandID, filename)
		if err != nil {
			log.Error().Err(err).Str("filename", filename).Msg("failed to download file")
			return err
		}
		files[filename] = content
		log.Debug().Str("filename", filename).Int("size", len(content)).Msg("file downloaded")
	}

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

	// Execute command
	result, err := exec.Execute(ctx, command, files)
	if err != nil {
		log.Error().Err(err).Msg("execution error")
	}

	// Submit result
	if err := commanderClient.SubmitResult(ctx, commandID, result); err != nil {
		log.Error().Err(err).Msg("failed to submit result")
		return err
	}

	if result.Success {
		log.Info().Str("output", result.Output).Msg("command applied successfully")
	} else {
		log.Error().Str("error", result.Error).Str("output", result.Output).Msg("command failed")
	}

	return nil
}
