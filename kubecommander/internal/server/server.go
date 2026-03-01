package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gigiozzz/kubedial/kubecommander/internal/endpoint"
	"github.com/gigiozzz/kubedial/kubecommander/internal/service"
	"github.com/rs/zerolog/log"
)

// TLSOptions holds TLS configuration for the server
type TLSOptions struct {
	Enabled  bool
	CertFile string
	KeyFile  string
	CAFile   string
}

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	router     *endpoint.ChiRouter
	tlsEnabled bool
}

// New creates a new Server
func New(
	port int,
	commandService service.CommandService,
	agentService service.AgentService,
	authService service.AuthService,
	tlsOpts TLSOptions,
) *Server {
	router := endpoint.NewChiRouter()

	// Create handlers
	commandHandler := endpoint.NewCommandHandler(commandService)
	agentHandler := endpoint.NewAgentHandler(agentService)

	// Setup routes with per-sub-router middleware
	router.Route("/api/v1", func(r endpoint.Router) {
		// /agents: bearer token auth
		r.Route("/agents", func(r endpoint.Router) {
			r.Use(endpoint.AuthMiddleware(authService))
			agentHandler.RegisterRoutes(r)
		})

		// /commands: mTLS when TLS enabled, bearer token when disabled (backward compat)
		r.Route("/commands", func(r endpoint.Router) {
			if tlsOpts.Enabled {
				r.Use(endpoint.RequireClientCertMiddleware())
			} else {
				r.Use(endpoint.AuthMiddleware(authService))
			}
			commandHandler.RegisterRoutes(r)
		})
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if tlsOpts.Enabled {
		tlsConfig, err := buildTLSConfig(tlsOpts)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to build TLS configuration")
		}
		httpServer.TLSConfig = tlsConfig
	}

	return &Server{
		httpServer: httpServer,
		router:     router,
		tlsEnabled: tlsOpts.Enabled,
	}
}

func buildTLSConfig(opts TLSOptions) (*tls.Config, error) {
	caPEM, err := os.ReadFile(opts.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}

	cert, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server cert/key: %w", err)
	}

	return &tls.Config{
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    caPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	if s.tlsEnabled {
		log.Info().Str("addr", s.httpServer.Addr).Msg("starting HTTPS server")
		// Certs already loaded into TLSConfig.Certificates; pass empty strings
		return s.httpServer.ListenAndServeTLS("", "")
	}
	log.Info().Str("addr", s.httpServer.Addr).Msg("starting HTTP server")
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("shutting down server")
	return s.httpServer.Shutdown(ctx)
}
