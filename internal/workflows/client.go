package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WorkflowClient handles communication with the workflow service
type WorkflowClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewWorkflowClient creates a new workflow client
func NewWorkflowClient(baseURL string) *WorkflowClient {
	return &WorkflowClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecutionRequest represents a workflow execution request
type ExecutionRequest struct {
	WorkflowID string                 `json:"workflow_id"`
	Input      map[string]interface{} `json:"input"`
	Context    ExecutionContext       `json:"context"`
	Priority   int                    `json:"priority"`
	Async      bool                   `json:"async"`
}

// ExecutionContext provides context for workflow execution
type ExecutionContext struct {
	UserID      string                 `json:"user_id"`
	ProviderID  string                 `json:"provider_id"`
	BlobID      string                 `json:"blob_id"`
	RequestID   string                 `json:"request_id"`
	Metadata    map[string]interface{} `json:"metadata"`
	TraceParent string                 `json:"trace_parent,omitempty"`
}

// ExecutionResponse represents the workflow execution result
type ExecutionResponse struct {
	ExecutionID string                 `json:"execution_id"`
	Status      string                 `json:"status"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       *ExecutionError        `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// ExecutionError represents an execution error
type ExecutionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	StepID  string `json:"step_id,omitempty"`
}

// ExecuteWorkflow executes a workflow
func (c *WorkflowClient) ExecuteWorkflow(ctx context.Context, req ExecutionRequest) (*ExecutionResponse, error) {
	url := fmt.Sprintf("%s/workflows/%s/execute", c.baseURL, req.WorkflowID)
	
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var result ExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// GetExecutionStatus gets the status of a workflow execution
func (c *WorkflowClient) GetExecutionStatus(ctx context.Context, executionID string) (*ExecutionResponse, error) {
	url := fmt.Sprintf("%s/executions/%s", c.baseURL, executionID)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var result ExecutionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &result, nil
}

// CancelExecution cancels a running workflow execution
func (c *WorkflowClient) CancelExecution(ctx context.Context, executionID string) error {
	url := fmt.Sprintf("%s/executions/%s/cancel", c.baseURL, executionID)
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// RegisterWorkflow registers a new workflow definition
func (c *WorkflowClient) RegisterWorkflow(ctx context.Context, workflow *BlobProcessingWorkflow) error {
	url := fmt.Sprintf("%s/workflows", c.baseURL)
	
	body, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// UpdateWorkflow updates an existing workflow definition
func (c *WorkflowClient) UpdateWorkflow(ctx context.Context, workflow *BlobProcessingWorkflow) error {
	url := fmt.Sprintf("%s/workflows/%s", c.baseURL, workflow.ID)
	
	body, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// GetWorkflow gets a workflow definition
func (c *WorkflowClient) GetWorkflow(ctx context.Context, workflowID string) (*BlobProcessingWorkflow, error) {
	url := fmt.Sprintf("%s/workflows/%s", c.baseURL, workflowID)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var workflow BlobProcessingWorkflow
	if err := json.NewDecoder(resp.Body).Decode(&workflow); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &workflow, nil
}

// ListWorkflows lists all workflows for a provider
func (c *WorkflowClient) ListWorkflows(ctx context.Context, providerID string) ([]*BlobProcessingWorkflow, error) {
	url := fmt.Sprintf("%s/workflows?provider_id=%s", c.baseURL, providerID)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var workflows []*BlobProcessingWorkflow
	if err := json.NewDecoder(resp.Body).Decode(&workflows); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return workflows, nil
}