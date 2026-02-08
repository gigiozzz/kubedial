package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gigiozzz/kubedial/common/models"
)

// CommanderClient defines the interface for kubecommander API client
type CommanderClient interface {
	// RegisterAgent registers this agent with the commander and returns the agent token
	RegisterAgent(ctx context.Context, agent *models.Agent) (*models.Agent, string, error)

	// GetPendingCommands retrieves pending commands for this agent
	GetPendingCommands(ctx context.Context, agentID string) ([]*models.Command, error)

	// GetCommand retrieves a command by ID
	GetCommand(ctx context.Context, commandID string) (*models.Command, error)

	// GetCommandFile retrieves a file from a command
	GetCommandFile(ctx context.Context, commandID, filename string) ([]byte, error)

	// SubmitResult submits the result of a command execution
	SubmitResult(ctx context.Context, commandID string, result *models.CommandResult) error
}

// commanderClient implements CommanderClient
type commanderClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewCommanderClient creates a new CommanderClient
func NewCommanderClient(baseURL, token string) CommanderClient {
	return &commanderClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *commanderClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// registerResponse matches the kubecommander RegisterResponse
type registerResponse struct {
	models.Agent
	Token string `json:"token"`
}

// RegisterAgent registers this agent with the commander and returns the agent token
func (c *commanderClient) RegisterAgent(ctx context.Context, agent *models.Agent) (*models.Agent, string, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/agents/register", agent)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("failed to register agent: %s - %s", resp.Status, string(body))
	}

	var result registerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Agent, result.Token, nil
}

// GetPendingCommands retrieves pending commands for this agent
func (c *commanderClient) GetPendingCommands(ctx context.Context, agentID string) ([]*models.Command, error) {
	path := "/api/v1/commands/pending?agentId=" + url.QueryEscape(agentID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get pending commands: %s - %s", resp.Status, string(body))
	}

	var commands []*models.Command
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return commands, nil
}

// GetCommand retrieves a command by ID
func (c *commanderClient) GetCommand(ctx context.Context, commandID string) (*models.Command, error) {
	path := "/api/v1/commands/" + url.PathEscape(commandID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get command: %s - %s", resp.Status, string(body))
	}

	var cmd models.Command
	if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &cmd, nil
}

// GetCommandFile retrieves a file from a command
func (c *commanderClient) GetCommandFile(ctx context.Context, commandID, filename string) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/commands/%s/files/%s",
		url.PathEscape(commandID),
		url.PathEscape(filename))

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get file: %s - %s", resp.Status, string(body))
	}

	return io.ReadAll(resp.Body)
}

// SubmitResult submits the result of a command execution
func (c *commanderClient) SubmitResult(ctx context.Context, commandID string, result *models.CommandResult) error {
	path := fmt.Sprintf("/api/v1/commands/%s/result", url.PathEscape(commandID))
	resp, err := c.doRequest(ctx, http.MethodPut, path, result)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to submit result: %s - %s", resp.Status, string(body))
	}

	return nil
}
