package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gigiozzz/kubedial/common/provider"
	"github.com/gigiozzz/kubedial/kubecommander/internal/config"
	"github.com/gigiozzz/kubedial/kubecommander/internal/repository"
	"github.com/gigiozzz/kubedial/kubecommander/internal/server"
	"github.com/gigiozzz/kubedial/kubecommander/internal/service"
	"github.com/rs/zerolog/log"
)

var Version = "dev"

func main() {
	// Initialize logging
	ctx := provider.InitLogging()
	logger := provider.FromContext(ctx)

	logger.Info().Str("version", Version).Msg("starting kubecommander")

	// Load configuration
	cfg := config.Load()

	// Create Kubernetes client
	k8sClient, err := provider.NewClientset()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create Kubernetes client")
	}

	// Create repositories
	commandRepo := repository.NewCommandRepository(k8sClient, cfg.Namespace)
	agentRepo := repository.NewAgentRepository(k8sClient, cfg.Namespace)
	authRepo := repository.NewAuthRepository(k8sClient, cfg.Namespace)

	// Create services
	commandService := service.NewCommandService(commandRepo)
	agentService := service.NewAgentService(agentRepo)
	authService := service.NewAuthService(authRepo)

	// Create and start server
	srv := server.New(cfg.ServerPort, commandService, agentService, authService)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info().Msg("received shutdown signal")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("error during shutdown")
		}
	}()

	// Start server
	if err := srv.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}

	log.Info().Msg("server stopped")
}
