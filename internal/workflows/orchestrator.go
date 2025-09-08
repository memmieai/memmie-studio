package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	
	"github.com/google/uuid"
)

// Orchestrator coordinates workflow execution for blob processing
type Orchestrator struct {
	client          *WorkflowClient
	providers       map[string]*Provider
	workflows       map[string]*BlobProcessingWorkflow
	eventBus        EventBus
	deltaProcessor  *DeltaProcessor
	mu              sync.RWMutex
}

// Provider represents a blob processing provider
type Provider struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"` // namespace, processor, hybrid
	NamespaceID string            `json:"namespace_id,omitempty"`
	WorkflowIDs []string          `json:"workflow_ids"`
	Triggers    []TriggerConfig   `json:"triggers"`
	Config      ProviderConfig    `json:"config"`
	Active      bool              `json:"active"`
}

// TriggerConfig defines when a provider should be triggered
type TriggerConfig struct {
	Event      string                 `json:"event"` // onCreate, onUpdate, onDelete
	Conditions []TriggerCondition     `json:"conditions"`
	Priority   int                    `json:"priority"`
	Async      bool                   `json:"async"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// TriggerCondition defines conditions for triggering
type TriggerCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, contains, regex
	Value    interface{} `json:"value"`
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	MaxConcurrentJobs int                    `json:"max_concurrent_jobs"`
	RateLimitPerMin   int                    `json:"rate_limit_per_min"`
	TimeoutSeconds    int                    `json:"timeout_seconds"`
	RetryPolicy       *RetryPolicy           `json:"retry_policy"`
	Parameters        map[string]interface{} `json:"parameters"`
}

// EventBus interface for event publishing
type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(ctx context.Context, handler EventHandler) error
}

// Event represents a blob event
type Event struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	BlobID     string                 `json:"blob_id"`
	UserID     string                 `json:"user_id"`
	ProviderID string                 `json:"provider_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
}

// EventHandler handles events
type EventHandler func(ctx context.Context, event Event) error

// DeltaProcessor processes blob deltas
type DeltaProcessor struct {
	storage DeltaStorage
}

// DeltaStorage interface for delta storage
type DeltaStorage interface {
	Store(ctx context.Context, delta Delta) error
	GetByBlobID(ctx context.Context, blobID string) ([]Delta, error)
	ApplyDeltas(ctx context.Context, blobID string, deltas []Delta) error
}

// Delta represents a blob state change
type Delta struct {
	ID         string                 `json:"id"`
	BlobID     string                 `json:"blob_id"`
	ProviderID string                 `json:"provider_id"`
	Type       string                 `json:"type"` // create, update, delete, transform
	Path       string                 `json:"path"`
	OldValue   interface{}            `json:"old_value,omitempty"`
	NewValue   interface{}            `json:"new_value,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  time.Time              `json:"timestamp"`
	Sequence   int64                  `json:"sequence"`
}

// NewOrchestrator creates a new workflow orchestrator
func NewOrchestrator(workflowURL string, eventBus EventBus, deltaStorage DeltaStorage) *Orchestrator {
	return &Orchestrator{
		client:         NewWorkflowClient(workflowURL),
		providers:      make(map[string]*Provider),
		workflows:      make(map[string]*BlobProcessingWorkflow),
		eventBus:       eventBus,
		deltaProcessor: &DeltaProcessor{storage: deltaStorage},
	}
}

// RegisterProvider registers a provider with its workflows
func (o *Orchestrator) RegisterProvider(ctx context.Context, provider *Provider) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	// Register workflows for this provider
	for _, workflowID := range provider.WorkflowIDs {
		workflow, err := o.client.GetWorkflow(ctx, workflowID)
		if err != nil {
			return fmt.Errorf("failed to get workflow %s: %w", workflowID, err)
		}
		o.workflows[workflowID] = workflow
	}
	
	o.providers[provider.ID] = provider
	return nil
}

// ProcessBlob processes a blob through applicable providers
func (o *Orchestrator) ProcessBlob(ctx context.Context, blobID, userID string, eventType string) error {
	o.mu.RLock()
	providers := o.getTriggeredProviders(eventType)
	o.mu.RUnlock()
	
	// Create execution context
	execCtx := ExecutionContext{
		UserID:    userID,
		BlobID:    blobID,
		RequestID: uuid.New().String(),
		Metadata: map[string]interface{}{
			"event_type": eventType,
			"timestamp":  time.Now().Unix(),
		},
	}
	
	// Process through each provider
	var wg sync.WaitGroup
	errors := make(chan error, len(providers))
	
	for _, provider := range providers {
		if !provider.Active {
			continue
		}
		
		// Check if should run async
		async := o.shouldRunAsync(provider, eventType)
		
		if async {
			wg.Add(1)
			go func(p *Provider) {
				defer wg.Done()
				if err := o.executeProviderWorkflows(ctx, p, execCtx); err != nil {
					errors <- fmt.Errorf("provider %s: %w", p.ID, err)
				}
			}(provider)
		} else {
			if err := o.executeProviderWorkflows(ctx, provider, execCtx); err != nil {
				return fmt.Errorf("provider %s: %w", provider.ID, err)
			}
		}
	}
	
	// Wait for async executions
	wg.Wait()
	close(errors)
	
	// Collect errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("multiple errors during processing: %v", errs)
	}
	
	return nil
}

// executeProviderWorkflows executes all workflows for a provider
func (o *Orchestrator) executeProviderWorkflows(ctx context.Context, provider *Provider, execCtx ExecutionContext) error {
	execCtx.ProviderID = provider.ID
	
	for _, workflowID := range provider.WorkflowIDs {
		workflow, exists := o.workflows[workflowID]
		if !exists {
			continue
		}
		
		// Build input from blob and provider config
		input := o.buildWorkflowInput(provider, execCtx)
		
		req := ExecutionRequest{
			WorkflowID: workflowID,
			Input:      input,
			Context:    execCtx,
			Priority:   o.getProviderPriority(provider),
			Async:      true,
		}
		
		// Execute workflow
		resp, err := o.client.ExecuteWorkflow(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to execute workflow %s: %w", workflowID, err)
		}
		
		// Process workflow output to generate deltas
		if err := o.processWorkflowOutput(ctx, resp, provider.ID, execCtx.BlobID); err != nil {
			return fmt.Errorf("failed to process output: %w", err)
		}
	}
	
	return nil
}

// processWorkflowOutput processes workflow output and generates deltas
func (o *Orchestrator) processWorkflowOutput(ctx context.Context, resp *ExecutionResponse, providerID, blobID string) error {
	if resp.Error != nil {
		return fmt.Errorf("workflow execution error: %s", resp.Error.Message)
	}
	
	// Extract deltas from output
	deltas := o.extractDeltas(resp.Output, providerID, blobID)
	
	// Store deltas
	for _, delta := range deltas {
		if err := o.deltaProcessor.storage.Store(ctx, delta); err != nil {
			return fmt.Errorf("failed to store delta: %w", err)
		}
	}
	
	// Apply deltas to blob
	if err := o.deltaProcessor.storage.ApplyDeltas(ctx, blobID, deltas); err != nil {
		return fmt.Errorf("failed to apply deltas: %w", err)
	}
	
	// Publish delta events
	for _, delta := range deltas {
		event := Event{
			ID:         uuid.New().String(),
			Type:       "delta.applied",
			BlobID:     blobID,
			ProviderID: providerID,
			Timestamp:  time.Now(),
			Data: map[string]interface{}{
				"delta_id":   delta.ID,
				"delta_type": delta.Type,
				"path":       delta.Path,
			},
		}
		
		if err := o.eventBus.Publish(ctx, event); err != nil {
			// Log error but don't fail
			fmt.Printf("failed to publish delta event: %v\n", err)
		}
	}
	
	return nil
}

// extractDeltas extracts deltas from workflow output
func (o *Orchestrator) extractDeltas(output map[string]interface{}, providerID, blobID string) []Delta {
	var deltas []Delta
	
	// Check if output contains deltas field
	if deltasData, ok := output["deltas"]; ok {
		if deltasList, ok := deltasData.([]interface{}); ok {
			for _, deltaData := range deltasList {
				if deltaMap, ok := deltaData.(map[string]interface{}); ok {
					delta := Delta{
						ID:         uuid.New().String(),
						BlobID:     blobID,
						ProviderID: providerID,
						Timestamp:  time.Now(),
					}
					
					// Parse delta fields
					if t, ok := deltaMap["type"].(string); ok {
						delta.Type = t
					}
					if p, ok := deltaMap["path"].(string); ok {
						delta.Path = p
					}
					if v, ok := deltaMap["old_value"]; ok {
						delta.OldValue = v
					}
					if v, ok := deltaMap["new_value"]; ok {
						delta.NewValue = v
					}
					if m, ok := deltaMap["metadata"].(map[string]interface{}); ok {
						delta.Metadata = m
					}
					
					deltas = append(deltas, delta)
				}
			}
		}
	}
	
	// If no explicit deltas, create one from the entire output
	if len(deltas) == 0 && len(output) > 0 {
		delta := Delta{
			ID:         uuid.New().String(),
			BlobID:     blobID,
			ProviderID: providerID,
			Type:       "transform",
			Path:       "/",
			NewValue:   output,
			Timestamp:  time.Now(),
			Metadata: map[string]interface{}{
				"source": "workflow_output",
			},
		}
		deltas = append(deltas, delta)
	}
	
	return deltas
}

// getTriggeredProviders gets providers triggered by an event
func (o *Orchestrator) getTriggeredProviders(eventType string) []*Provider {
	var providers []*Provider
	
	for _, provider := range o.providers {
		for _, trigger := range provider.Triggers {
			if trigger.Event == eventType {
				if o.evaluateTriggerConditions(trigger.Conditions) {
					providers = append(providers, provider)
					break
				}
			}
		}
	}
	
	return providers
}

// evaluateTriggerConditions evaluates trigger conditions
func (o *Orchestrator) evaluateTriggerConditions(conditions []TriggerCondition) bool {
	// If no conditions, trigger is always active
	if len(conditions) == 0 {
		return true
	}
	
	// Evaluate all conditions (AND logic)
	for _, condition := range conditions {
		// TODO: Implement condition evaluation logic
		// For now, return true
		_ = condition
	}
	
	return true
}

// shouldRunAsync determines if provider should run asynchronously
func (o *Orchestrator) shouldRunAsync(provider *Provider, eventType string) bool {
	for _, trigger := range provider.Triggers {
		if trigger.Event == eventType {
			return trigger.Async
		}
	}
	return false
}

// getProviderPriority gets the priority for a provider
func (o *Orchestrator) getProviderPriority(provider *Provider) int {
	maxPriority := 0
	for _, trigger := range provider.Triggers {
		if trigger.Priority > maxPriority {
			maxPriority = trigger.Priority
		}
	}
	return maxPriority
}

// buildWorkflowInput builds input for workflow execution
func (o *Orchestrator) buildWorkflowInput(provider *Provider, ctx ExecutionContext) map[string]interface{} {
	return map[string]interface{}{
		"blob_id":     ctx.BlobID,
		"user_id":     ctx.UserID,
		"provider_id": provider.ID,
		"parameters":  provider.Config.Parameters,
		"metadata":    ctx.Metadata,
	}
}

// GetProviderDAG returns the DAG of providers and their dependencies
func (o *Orchestrator) GetProviderDAG(ctx context.Context) (map[string][]string, error) {
	dag := make(map[string][]string)
	
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	for providerID, provider := range o.providers {
		var dependencies []string
		
		// Analyze workflows to determine dependencies
		for _, workflowID := range provider.WorkflowIDs {
			if workflow, exists := o.workflows[workflowID]; exists {
				// Extract provider dependencies from workflow steps
				for _, step := range workflow.Steps {
					if step.ProviderID != "" && step.ProviderID != providerID {
						dependencies = append(dependencies, step.ProviderID)
					}
				}
			}
		}
		
		dag[providerID] = dependencies
	}
	
	return dag, nil
}