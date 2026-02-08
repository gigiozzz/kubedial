package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gigiozzz/kubedial/kubecommander/internal/endpoint"
	"github.com/gigiozzz/kubedial/kubecommander/internal/service"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	router     *endpoint.ChiRouter
}

// New creates a new Server
func New(
	port int,
	commandService service.CommandService,
	agentService service.AgentService,
	authService service.AuthService,
) *Server {
	router := endpoint.NewChiRouter()

	// Create handlers
	commandHandler := endpoint.NewCommandHandler(commandService)
	agentHandler := endpoint.NewAgentHandler(agentService)

	// Setup routes with authentication middleware
	router.Route("/api/v1", func(r endpoint.Router) {
		r.Use(endpoint.AuthMiddleware(authService))

		r.Route("/agents", func(r endpoint.Router) {
			agentHandler.RegisterRoutes(r)
		})

		r.Route("/commands", func(r endpoint.Router) {
			commandHandler.RegisterRoutes(r)
		})
	})

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		router: router,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Info().Str("addr", s.httpServer.Addr).Msg("starting HTTP server")
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
