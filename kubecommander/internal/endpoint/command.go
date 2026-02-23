package endpoint

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gigiozzz/kubedial/common/models"
	"github.com/gigiozzz/kubedial/kubecommander/internal/service"
	"github.com/rs/zerolog/log"
)

// CommandHandler handles command-related HTTP requests
type CommandHandler struct {
	service service.CommandService
}

// NewCommandHandler creates a new CommandHandler
func NewCommandHandler(svc service.CommandService) *CommandHandler {
	return &CommandHandler{service: svc}
}

// RegisterRoutes registers command routes on the router
func (h *CommandHandler) RegisterRoutes(r Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/pending", h.GetPending)
	r.Get("/{id}", h.Get)
	r.Get("/{id}/files", h.ListFiles)
	r.Get("/{id}/files/{filename}", h.GetFile)
	r.Put("/{id}/result", h.UpdateResult)
}

// CommandMetadata represents the JSON metadata in multipart request
type CommandMetadata struct {
	AgentID       string               `json:"agentId"`
	OperationType models.OperationType `json:"operationType"`
	Namespace     string               `json:"namespace"`
	ServerSide    bool                 `json:"serverSide"`
	DryRun        bool                 `json:"dryRun"`
	Force         bool                 `json:"force"`
	Prune         bool                 `json:"prune"`
}

// Create handles command creation via multipart form
func (h *CommandHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		log.Error().Err(err).Msg("failed to parse multipart form")
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Parse metadata
	metadataStr := r.FormValue("metadata")
	if metadataStr == "" {
		http.Error(w, "missing metadata field", http.StatusBadRequest)
		return
	}

	var metadata CommandMetadata
	if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
		log.Error().Err(err).Msg("failed to parse metadata JSON")
		http.Error(w, "invalid metadata JSON", http.StatusBadRequest)
		return
	}

	// Parse files
	files := make(map[string][]byte)
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		for _, fileHeaders := range r.MultipartForm.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					log.Error().Err(err).Str("filename", fh.Filename).Msg("failed to open uploaded file")
					continue
				}
				defer f.Close()

				content, err := io.ReadAll(f)
				if err != nil {
					log.Error().Err(err).Str("filename", fh.Filename).Msg("failed to read uploaded file")
					continue
				}

				files[fh.Filename] = content
			}
		}
	}

	if len(files) == 0 {
		http.Error(w, "no files uploaded", http.StatusBadRequest)
		return
	}

	// Create command
	cmd := &models.Command{
		AgentID:       metadata.AgentID,
		OperationType: metadata.OperationType,
		Namespace:     metadata.Namespace,
		ServerSide:    metadata.ServerSide,
		DryRun:        metadata.DryRun,
		Force:         metadata.Force,
		Prune:         metadata.Prune,
	}

	created, err := h.service.Create(r.Context(), cmd, files)
	if err != nil {
		log.Error().Err(err).Msg("failed to create command")
		http.Error(w, "failed to create command", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created); err != nil {
		log.Error().Err(err).Msg("failed to encode create command response")
	}
}

// List handles listing all commands
func (h *CommandHandler) List(w http.ResponseWriter, r *http.Request) {
	commands, err := h.service.List(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list commands")
		http.Error(w, "failed to list commands", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(commands); err != nil {
		log.Error().Err(err).Msg("failed to encode commands list response")
	}
}

// GetPending handles getting pending commands for an agent
func (h *CommandHandler) GetPending(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agentId")
	if agentID == "" {
		http.Error(w, "missing agentId query parameter", http.StatusBadRequest)
		return
	}

	commands, err := h.service.GetPending(r.Context(), agentID)
	if err != nil {
		log.Error().Err(err).Str("agentId", agentID).Msg("failed to get pending commands")
		http.Error(w, "failed to get pending commands", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(commands); err != nil {
		log.Error().Err(err).Msg("failed to encode pending commands response")
	}
}

// Get handles getting a single command with its result
func (h *CommandHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := URLParam(r, "id")

	cmd, result, err := h.service.GetWithResult(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("failed to get command")
		http.Error(w, "failed to get command", http.StatusInternalServerError)
		return
	}

	if cmd == nil {
		http.Error(w, "command not found", http.StatusNotFound)
		return
	}

	response := struct {
		*models.Command
		Result *models.CommandResult `json:"result,omitempty"`
	}{
		Command: cmd,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("failed to encode get command response")
	}
}

// ListFiles handles listing files in a command
func (h *CommandHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	id := URLParam(r, "id")

	files, err := h.service.ListFiles(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("failed to list files")
		http.Error(w, "failed to list files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(files); err != nil {
		log.Error().Err(err).Msg("failed to encode files list response")
	}
}

// GetFile handles downloading a specific file
func (h *CommandHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	id := URLParam(r, "id")
	filename := URLParam(r, "filename")

	content, err := h.service.GetFile(r.Context(), id, filename)
	if err != nil {
		log.Error().Err(err).Str("id", id).Str("filename", filename).Msg("failed to get file")
		http.Error(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	if content == nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	if _, err := w.Write(content); err != nil {
		log.Error().Err(err).Msg("failed to write file content")
	}
}

// UpdateResult handles updating a command's result
func (h *CommandHandler) UpdateResult(w http.ResponseWriter, r *http.Request) {
	id := URLParam(r, "id")

	var result models.CommandResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		log.Error().Err(err).Msg("failed to decode result")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateResult(r.Context(), id, &result); err != nil {
		log.Error().Err(err).Str("id", id).Msg("failed to update result")
		http.Error(w, "failed to update result", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
