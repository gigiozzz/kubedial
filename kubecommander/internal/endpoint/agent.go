package endpoint

import (
	"encoding/json"
	"net/http"

	"github.com/gigiozzz/kubedial/common/models"
	"github.com/gigiozzz/kubedial/kubecommander/internal/service"
	"github.com/rs/zerolog/log"
)

// AgentHandler handles agent-related HTTP requests
type AgentHandler struct {
	service service.AgentService
}

// NewAgentHandler creates a new AgentHandler
func NewAgentHandler(svc service.AgentService) *AgentHandler {
	return &AgentHandler{service: svc}
}

// RegisterRoutes registers agent routes on the router
func (h *AgentHandler) RegisterRoutes(r Router) {
	r.Post("/register", h.Register)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
}

// RegisterResponse is the response for agent registration
type RegisterResponse struct {
	models.Agent
	Token string `json:"token"`
}

// Register handles agent registration
func (h *AgentHandler) Register(w http.ResponseWriter, r *http.Request) {
	var agent models.Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		log.Error().Err(err).Msg("failed to decode agent registration request")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	registered, token, err := h.service.Register(r.Context(), &agent)
	if err != nil {
		log.Error().Err(err).Msg("failed to register agent")
		http.Error(w, "failed to register agent", http.StatusInternalServerError)
		return
	}

	resp := RegisterResponse{
		Agent: *registered,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// List handles listing all agents
func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	agents, err := h.service.List(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list agents")
		http.Error(w, "failed to list agents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// Get handles getting a single agent
func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := URLParam(r, "id")

	agent, err := h.service.Get(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("failed to get agent")
		http.Error(w, "failed to get agent", http.StatusInternalServerError)
		return
	}

	if agent == nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}
