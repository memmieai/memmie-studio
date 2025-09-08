package workflows

import (
	"encoding/json"
	"fmt"
	"time"
)

// WorkflowType represents the type of workflow for blob processing
type WorkflowType string

const (
	WorkflowTypeProcessBlob      WorkflowType = "process_blob"
	WorkflowTypeApplyDelta       WorkflowType = "apply_delta"
	WorkflowTypeProviderPipeline WorkflowType = "provider_pipeline"
	WorkflowTypeNamespaceSync    WorkflowType = "namespace_sync"
)

// BlobProcessingWorkflow defines a workflow for processing blobs through providers
type BlobProcessingWorkflow struct {
	ID          string                   `json:"id"`
	ProviderID  string                   `json:"provider_id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Type        WorkflowType             `json:"type"`
	Steps       []BlobProcessingStep     `json:"steps"`
	Config      ProcessingConfig         `json:"config"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// BlobProcessingStep represents a single step in blob processing
type BlobProcessingStep struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	ProviderID   string                 `json:"provider_id"`
	Type         string                 `json:"type"` // transform, validate, enrich, etc.
	InputMap     map[string]interface{} `json:"input_map"`
	OutputMap    map[string]interface{} `json:"output_map"`
	Config       StepConfig             `json:"config"`
	Dependencies []string               `json:"dependencies"` // IDs of steps this depends on
	Condition    string                 `json:"condition,omitempty"` // Expression to evaluate
	OnFailure    string                 `json:"on_failure"` // fail, skip, retry
	RetryPolicy  *RetryPolicy           `json:"retry_policy,omitempty"`
}

// StepConfig holds step-specific configuration
type StepConfig struct {
	Timeout           int                    `json:"timeout_seconds"`
	MaxRetries        int                    `json:"max_retries"`
	ParallelExecution bool                   `json:"parallel_execution"`
	CacheResults      bool                   `json:"cache_results"`
	CacheTTL          int                    `json:"cache_ttl_seconds"`
	Parameters        map[string]interface{} `json:"parameters"`
}

// ProcessingConfig holds workflow-level configuration
type ProcessingConfig struct {
	MaxConcurrency   int  `json:"max_concurrency"`
	StopOnError      bool `json:"stop_on_error"`
	EnableRollback   bool `json:"enable_rollback"`
	TrackLineage     bool `json:"track_lineage"`
	EmitEvents       bool `json:"emit_events"`
	AutoRetry        bool `json:"auto_retry"`
	RetryDelay       int  `json:"retry_delay_seconds"`
	MaxExecutionTime int  `json:"max_execution_time_seconds"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts       int    `json:"max_attempts"`
	BackoffMultiplier float64 `json:"backoff_multiplier"`
	InitialDelay      int    `json:"initial_delay_ms"`
	MaxDelay          int    `json:"max_delay_ms"`
}

// DeltaWorkflow defines a workflow for applying deltas to blobs
type DeltaWorkflow struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Operations []DeltaOperation    `json:"operations"`
	Validation DeltaValidation     `json:"validation"`
	Rollback   RollbackPolicy      `json:"rollback"`
	CreatedAt  time.Time           `json:"created_at"`
}

// DeltaOperation represents a single delta operation
type DeltaOperation struct {
	Type      string                 `json:"type"` // create, update, delete, transform
	Path      string                 `json:"path"` // JSON path or field path
	Value     interface{}            `json:"value,omitempty"`
	Transform string                 `json:"transform,omitempty"` // Expression for transformation
	Condition string                 `json:"condition,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// DeltaValidation defines validation rules for deltas
type DeltaValidation struct {
	SchemaValidation bool                `json:"schema_validation"`
	SchemaID         string              `json:"schema_id,omitempty"`
	CustomRules      []ValidationRule    `json:"custom_rules"`
	ConflictResolution string            `json:"conflict_resolution"` // last_write_wins, merge, fail
}

// ValidationRule defines a custom validation rule
type ValidationRule struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
	Message    string `json:"message"`
	Severity   string `json:"severity"` // error, warning, info
}

// RollbackPolicy defines how to handle rollbacks
type RollbackPolicy struct {
	Enabled          bool              `json:"enabled"`
	MaxRollbackDepth int               `json:"max_rollback_depth"`
	Strategy         string            `json:"strategy"` // immediate, deferred, manual
	CompensationMap  map[string]string `json:"compensation_map"` // Maps operations to compensations
}

// ProviderWorkflowMapping maps providers to their workflows
type ProviderWorkflowMapping struct {
	ProviderID   string   `json:"provider_id"`
	WorkflowIDs  []string `json:"workflow_ids"`
	Priority     int      `json:"priority"`
	Enabled      bool     `json:"enabled"`
	TriggerEvents []string `json:"trigger_events"` // onCreate, onUpdate, onDelete, onSchedule
}

// CreateBlobProcessingWorkflow creates a workflow definition for blob processing
func CreateBlobProcessingWorkflow(providerID, name string) (*BlobProcessingWorkflow, error) {
	return &BlobProcessingWorkflow{
		ID:          fmt.Sprintf("workflow_%s_%d", providerID, time.Now().Unix()),
		ProviderID:  providerID,
		Name:        name,
		Description: fmt.Sprintf("Blob processing workflow for %s", name),
		Type:        WorkflowTypeProcessBlob,
		Steps:       []BlobProcessingStep{},
		Config: ProcessingConfig{
			MaxConcurrency:   5,
			StopOnError:      false,
			EnableRollback:   true,
			TrackLineage:     true,
			EmitEvents:       true,
			AutoRetry:        true,
			RetryDelay:       5,
			MaxExecutionTime: 3600,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// AddStep adds a processing step to the workflow
func (w *BlobProcessingWorkflow) AddStep(step BlobProcessingStep) {
	w.Steps = append(w.Steps, step)
	w.UpdatedAt = time.Now()
}

// ToJSON converts workflow to JSON
func (w *BlobProcessingWorkflow) ToJSON() ([]byte, error) {
	return json.MarshalIndent(w, "", "  ")
}

// GetDAGOrder returns steps in DAG execution order
func (w *BlobProcessingWorkflow) GetDAGOrder() ([][]BlobProcessingStep, error) {
	// Build dependency graph
	graph := make(map[string][]string)
	stepMap := make(map[string]BlobProcessingStep)
	inDegree := make(map[string]int)
	
	for _, step := range w.Steps {
		stepMap[step.ID] = step
		inDegree[step.ID] = len(step.Dependencies)
		
		for _, dep := range step.Dependencies {
			graph[dep] = append(graph[dep], step.ID)
		}
	}
	
	// Topological sort with level grouping
	var levels [][]BlobProcessingStep
	queue := []string{}
	
	// Find nodes with no dependencies
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	
	for len(queue) > 0 {
		levelSize := len(queue)
		level := []BlobProcessingStep{}
		
		for i := 0; i < levelSize; i++ {
			current := queue[0]
			queue = queue[1:]
			level = append(level, stepMap[current])
			
			// Reduce in-degree for dependent nodes
			for _, next := range graph[current] {
				inDegree[next]--
				if inDegree[next] == 0 {
					queue = append(queue, next)
				}
			}
		}
		
		levels = append(levels, level)
	}
	
	// Check for cycles
	processedCount := 0
	for _, level := range levels {
		processedCount += len(level)
	}
	
	if processedCount != len(w.Steps) {
		return nil, fmt.Errorf("workflow contains cycles")
	}
	
	return levels, nil
}

// CreateDeltaWorkflow creates a workflow for applying deltas
func CreateDeltaWorkflow(name string, operations []DeltaOperation) *DeltaWorkflow {
	return &DeltaWorkflow{
		ID:         fmt.Sprintf("delta_%d", time.Now().Unix()),
		Name:       name,
		Operations: operations,
		Validation: DeltaValidation{
			SchemaValidation:   true,
			ConflictResolution: "last_write_wins",
			CustomRules:        []ValidationRule{},
		},
		Rollback: RollbackPolicy{
			Enabled:          true,
			MaxRollbackDepth: 10,
			Strategy:         "immediate",
			CompensationMap:  make(map[string]string),
		},
		CreatedAt: time.Now(),
	}
}