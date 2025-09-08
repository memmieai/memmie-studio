package workflows

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	
	"gopkg.in/yaml.v3"
)

// YAMLWorkflow represents a workflow definition in YAML format
type YAMLWorkflow struct {
	ID             string         `yaml:"id"`
	ProviderID     string         `yaml:"provider_id"`
	Name           string         `yaml:"name"`
	Description    string         `yaml:"description"`
	InputSchemaID  string         `yaml:"input_schema_id"`
	OutputSchemaID string         `yaml:"output_schema_id"`
	Active         bool           `yaml:"active"`
	Steps          []YAMLStep     `yaml:"steps"`
}

// YAMLStep represents a workflow step in YAML format
type YAMLStep struct {
	ID           string                 `yaml:"id"`
	Name         string                 `yaml:"name"`
	Type         string                 `yaml:"type"`
	Service      string                 `yaml:"service"`
	Endpoint     string                 `yaml:"endpoint"`
	Method       string                 `yaml:"method"`
	InputMap     map[string]interface{} `yaml:"input_map"`
	OutputMap    map[string]interface{} `yaml:"output_map"`
	Condition    string                 `yaml:"condition"`
	Variables    []string               `yaml:"variables"`
	Compensation *YAMLCompensation      `yaml:"compensation"`
	Retry        *YAMLRetry             `yaml:"retry"`
	Timeout      int                    `yaml:"timeout_seconds"`
	OnFailure    string                 `yaml:"on_failure"`
}

// YAMLCompensation represents compensation configuration
type YAMLCompensation struct {
	Service  string                 `yaml:"service"`
	Endpoint string                 `yaml:"endpoint"`
	Method   string                 `yaml:"method"`
	InputMap map[string]interface{} `yaml:"input_map"`
}

// YAMLRetry represents retry configuration
type YAMLRetry struct {
	MaxAttempts  int `yaml:"max_attempts"`
	BackoffMs    int `yaml:"backoff_ms"`
	MaxBackoffMs int `yaml:"max_backoff_ms"`
}

// YAMLSchema represents a schema definition in YAML format
type YAMLSchema struct {
	ID          string                 `yaml:"id"`
	ProviderID  string                 `yaml:"provider_id"`
	Name        string                 `yaml:"name"`
	Version     string                 `yaml:"version"`
	Type        string                 `yaml:"type"`
	Description string                 `yaml:"description"`
	Definition  map[string]interface{} `yaml:"definition"`
}

// YAMLProvider represents a provider definition in YAML format
type YAMLProvider struct {
	Provider struct {
		ID          string            `yaml:"id"`
		Name        string            `yaml:"name"`
		Type        string            `yaml:"type"`
		Description string            `yaml:"description"`
		Namespace   *NamespaceConfig  `yaml:"namespace"`
		Processor   *ProcessorConfig  `yaml:"processor"`
	} `yaml:"provider"`
	
	Workflows []WorkflowMapping     `yaml:"workflows"`
	Config    ProviderConfiguration `yaml:"config"`
	Permissions map[string][]string  `yaml:"permissions"`
	Metadata  map[string]interface{} `yaml:"metadata"`
}

// NamespaceConfig represents namespace configuration in YAML
type NamespaceConfig struct {
	Name               string                 `yaml:"name"`
	AllowSubNamespaces bool                   `yaml:"allow_sub_namespaces"`
	Schema             map[string]interface{} `yaml:"schema"`
	OnCreateProviders  []string               `yaml:"on_create_providers"`
	OnEditProviders    []string               `yaml:"on_edit_providers"`
}

// ProcessorConfig represents processor configuration in YAML
type ProcessorConfig struct {
	Capabilities []string `yaml:"capabilities"`
}

// WorkflowMapping represents workflow trigger mapping
type WorkflowMapping struct {
	WorkflowID string    `yaml:"workflow_id"`
	Triggers   []Trigger `yaml:"triggers"`
}

// Trigger represents a workflow trigger
type Trigger struct {
	Event      string      `yaml:"event"`
	Conditions []Condition `yaml:"conditions"`
	Priority   int         `yaml:"priority"`
	Async      bool        `yaml:"async"`
}

// Condition represents a trigger condition
type Condition struct {
	Field    string      `yaml:"field"`
	Operator string      `yaml:"operator"`
	Value    interface{} `yaml:"value"`
}

// ProviderConfiguration represents provider configuration
type ProviderConfiguration struct {
	MaxConcurrentJobs int                    `yaml:"max_concurrent_jobs"`
	RateLimitPerMin   int                    `yaml:"rate_limit_per_min"`
	TimeoutSeconds    int                    `yaml:"timeout_seconds"`
	RetryPolicy       *RetryPolicy           `yaml:"retry_policy"`
	Parameters        map[string]interface{} `yaml:"parameters"`
}

// WorkflowLoader handles loading and registering YAML workflows
type WorkflowLoader struct {
	client       *WorkflowClient
	workflowsDir string
	schemasDir   string
	providersDir string
}

// NewWorkflowLoader creates a new workflow loader
func NewWorkflowLoader(client *WorkflowClient, workflowsDir, schemasDir, providersDir string) *WorkflowLoader {
	return &WorkflowLoader{
		client:       client,
		workflowsDir: workflowsDir,
		schemasDir:   schemasDir,
		providersDir: providersDir,
	}
}

// LoadAndRegisterAll loads and registers all YAML definitions
func (l *WorkflowLoader) LoadAndRegisterAll(ctx context.Context) error {
	// Load and register schemas first
	if err := l.loadSchemas(ctx); err != nil {
		return fmt.Errorf("failed to load schemas: %w", err)
	}
	
	// Load and register workflows
	if err := l.loadWorkflows(ctx); err != nil {
		return fmt.Errorf("failed to load workflows: %w", err)
	}
	
	// Load and register providers
	if err := l.loadProviders(ctx); err != nil {
		return fmt.Errorf("failed to load providers: %w", err)
	}
	
	return nil
}

// loadSchemas loads all schema YAML files
func (l *WorkflowLoader) loadSchemas(ctx context.Context) error {
	pattern := filepath.Join(l.schemasDir, "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob schema files: %w", err)
	}
	
	for _, file := range files {
		if err := l.loadSchemaFile(ctx, file); err != nil {
			return fmt.Errorf("failed to load schema %s: %w", file, err)
		}
	}
	
	return nil
}

// loadSchemaFile loads a single schema YAML file
func (l *WorkflowLoader) loadSchemaFile(ctx context.Context, filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	var schema YAMLSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	
	// Convert to internal format and register
	// This would call the workflow service API to register the schema
	fmt.Printf("Loaded schema: %s from %s\n", schema.ID, filename)
	
	return nil
}

// loadWorkflows loads all workflow YAML files
func (l *WorkflowLoader) loadWorkflows(ctx context.Context) error {
	pattern := filepath.Join(l.workflowsDir, "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob workflow files: %w", err)
	}
	
	for _, file := range files {
		if err := l.loadWorkflowFile(ctx, file); err != nil {
			return fmt.Errorf("failed to load workflow %s: %w", file, err)
		}
	}
	
	return nil
}

// loadWorkflowFile loads a single workflow YAML file
func (l *WorkflowLoader) loadWorkflowFile(ctx context.Context, filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	var workflow YAMLWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	
	// Convert YAML workflow to internal BlobProcessingWorkflow format
	bpWorkflow := l.convertYAMLToWorkflow(workflow)
	
	// Register with workflow service
	if err := l.client.RegisterWorkflow(ctx, bpWorkflow); err != nil {
		return fmt.Errorf("failed to register workflow: %w", err)
	}
	
	fmt.Printf("Loaded workflow: %s from %s\n", workflow.ID, filename)
	
	return nil
}

// convertYAMLToWorkflow converts YAML workflow to internal format
func (l *WorkflowLoader) convertYAMLToWorkflow(yaml YAMLWorkflow) *BlobProcessingWorkflow {
	workflow := &BlobProcessingWorkflow{
		ID:          yaml.ID,
		ProviderID:  yaml.ProviderID,
		Name:        yaml.Name,
		Description: yaml.Description,
		Type:        WorkflowTypeProcessBlob,
		Steps:       make([]BlobProcessingStep, 0, len(yaml.Steps)),
		Config: ProcessingConfig{
			MaxConcurrency:   5,
			StopOnError:      false,
			EnableRollback:   true,
			TrackLineage:     true,
			EmitEvents:       true,
			AutoRetry:        true,
			RetryDelay:       5,
			MaxExecutionTime: 600,
		},
	}
	
	// Convert steps
	for _, yamlStep := range yaml.Steps {
		step := BlobProcessingStep{
			ID:         yamlStep.ID,
			Name:       yamlStep.Name,
			ProviderID: yaml.ProviderID,
			Type:       yamlStep.Type,
			InputMap:   yamlStep.InputMap,
			OutputMap:  yamlStep.OutputMap,
			Condition:  yamlStep.Condition,
			OnFailure:  yamlStep.OnFailure,
			Config: StepConfig{
				Timeout: yamlStep.Timeout,
			},
		}
		
		// Convert retry policy
		if yamlStep.Retry != nil {
			step.RetryPolicy = &RetryPolicy{
				MaxAttempts:       yamlStep.Retry.MaxAttempts,
				BackoffMultiplier: 2.0,
				InitialDelay:      yamlStep.Retry.BackoffMs,
				MaxDelay:          yamlStep.Retry.MaxBackoffMs,
			}
		}
		
		// Extract dependencies from conditions
		if strings.Contains(yamlStep.Condition, "$.steps.") {
			// Parse dependencies from condition expressions
			// This is a simplified version - real implementation would parse properly
			step.Dependencies = l.extractDependencies(yamlStep.Condition)
		}
		
		workflow.Steps = append(workflow.Steps, step)
	}
	
	return workflow
}

// extractDependencies extracts step dependencies from condition expressions
func (l *WorkflowLoader) extractDependencies(condition string) []string {
	var deps []string
	
	// Simple extraction - finds patterns like $.steps.step_id.
	// Real implementation would use proper expression parsing
	parts := strings.Split(condition, "$.steps.")
	for i := 1; i < len(parts); i++ {
		endIdx := strings.IndexAny(parts[i], ". ")
		if endIdx == -1 {
			endIdx = len(parts[i])
		}
		stepID := parts[i][:endIdx]
		if stepID != "" && !contains(deps, stepID) {
			deps = append(deps, stepID)
		}
	}
	
	return deps
}

// loadProviders loads all provider YAML files
func (l *WorkflowLoader) loadProviders(ctx context.Context) error {
	pattern := filepath.Join(l.providersDir, "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob provider files: %w", err)
	}
	
	for _, file := range files {
		if err := l.loadProviderFile(ctx, file); err != nil {
			return fmt.Errorf("failed to load provider %s: %w", file, err)
		}
	}
	
	return nil
}

// loadProviderFile loads a single provider YAML file
func (l *WorkflowLoader) loadProviderFile(ctx context.Context, filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	var provider YAMLProvider
	if err := yaml.Unmarshal(data, &provider); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	
	// Convert and register provider
	// This would integrate with the provider registration system
	fmt.Printf("Loaded provider: %s from %s\n", provider.Provider.ID, filename)
	
	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}