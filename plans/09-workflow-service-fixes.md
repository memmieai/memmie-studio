# Workflow Service Fixes and Integration

## Current Issues

### 1. Compilation Errors in memmie-workflow

The workflow service has several compilation errors that need to be fixed:

#### Missing EnqueueExecution Implementation
The `PostgresRepository` doesn't implement the `EnqueueExecution` method defined in the Repository interface.

#### QueuedExecution Structure Mismatch
The `DequeueExecution` method references fields that don't exist in the `QueuedExecution` struct:
- Missing fields: `ID`, `Status`, `RetryCount`, `MaxRetries`, `CreatedAt`, `UpdatedAt`, `Metadata`
- Existing fields: `ExecutionID`, `WorkflowID`, `Priority`, `QueuedAt`, `Attempt`

### 2. Service Connection Issues
- NATS authorization failures (already fixed in dev.sh)
- MongoDB authentication issues (already fixed in dev.sh)

## Fixes Required

### Fix 1: Update QueuedExecution Structure

```go
// internal/domain/models.go
type QueuedExecution struct {
    ID          uuid.UUID              `json:"id" db:"id"`
    ExecutionID uuid.UUID              `json:"execution_id" db:"execution_id"`
    WorkflowID  string                 `json:"workflow_id" db:"workflow_id"`
    Priority    int                    `json:"priority" db:"priority"`
    Status      string                 `json:"status" db:"status"`
    RetryCount  int                    `json:"retry_count" db:"retry_count"`
    MaxRetries  int                    `json:"max_retries" db:"max_retries"`
    QueuedAt    time.Time              `json:"queued_at" db:"queued_at"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
    Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
}
```

### Fix 2: Implement EnqueueExecution Method

```go
// internal/repository/postgres.go
func (r *PostgresRepository) EnqueueExecution(ctx context.Context, execution *domain.QueuedExecution) error {
    query := `
        INSERT INTO queued_executions (
            id, execution_id, workflow_id, priority, status, 
            retry_count, max_retries, created_at, updated_at, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
    
    metadata, err := json.Marshal(execution.Metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }
    
    execution.ID = uuid.New()
    execution.CreatedAt = time.Now()
    execution.UpdatedAt = time.Now()
    execution.Status = "pending"
    
    _, err = r.db.ExecContext(ctx, query,
        execution.ID,
        execution.ExecutionID,
        execution.WorkflowID,
        execution.Priority,
        execution.Status,
        execution.RetryCount,
        execution.MaxRetries,
        execution.CreatedAt,
        execution.UpdatedAt,
        metadata,
    )
    
    if err != nil {
        return fmt.Errorf("failed to enqueue execution: %w", err)
    }
    
    return nil
}
```

### Fix 3: Create Database Migration

```sql
-- migrations/004_add_queue_tables.sql
CREATE TABLE IF NOT EXISTS queued_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL,
    workflow_id VARCHAR(255) NOT NULL,
    priority INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB,
    
    INDEX idx_queued_status (status),
    INDEX idx_queued_priority (priority DESC, created_at ASC)
);
```

## YAML-Based Workflow System

The workflow service uses a JSON-based structure internally but can support YAML workflow definitions. Here's how it works:

### Current Workflow Structure

```go
type Workflow struct {
    ID             string         `json:"id"`
    ProviderID     string         `json:"provider_id"`
    Name           string         `json:"name"`
    Description    string         `json:"description"`
    InputSchemaID  string         `json:"input_schema_id"`
    OutputSchemaID string         `json:"output_schema_id"`
    Steps          []WorkflowStep `json:"steps"`
    Active         bool           `json:"active"`
}

type WorkflowStep struct {
    ID           string                 `json:"id"`
    Name         string                 `json:"name"`
    Type         StepType               `json:"type"`
    Service      string                 `json:"service"`
    Endpoint     string                 `json:"endpoint"`
    Method       string                 `json:"method"`
    InputMap     map[string]interface{} `json:"input_map"`
    OutputMap    map[string]interface{} `json:"output_map"`
    Condition    string                 `json:"condition"`
    Variables    []string               `json:"variables"`
    Compensation *CompensationAction    `json:"compensation"`
    Retry        *RetryPolicy           `json:"retry"`
    Timeout      int                    `json:"timeout_seconds"`
    OnFailure    StepFailureAction      `json:"on_failure"`
}
```

### YAML Workflow Definition Format

```yaml
# workflows/blob-processing.yaml
id: blob_processing_workflow
provider_id: memmie-studio
name: Blob Processing Pipeline
description: Process blobs through provider transformations
input_schema_id: blob_input_schema_v1
output_schema_id: blob_output_schema_v1
active: true

steps:
  - id: validate_blob
    name: Validate Blob Structure
    type: api_call
    service: studio
    endpoint: /blobs/validate
    method: POST
    input_map:
      blob_id: $.input.blob_id
      schema_id: $.input.schema_id
    timeout_seconds: 30
    on_failure: fail
    retry:
      max_attempts: 2
      backoff_ms: 1000

  - id: extract_metadata
    name: Extract Blob Metadata
    type: api_call
    service: studio
    endpoint: /blobs/metadata
    method: POST
    input_map:
      blob_id: $.input.blob_id
      extract_fields:
        - content_type
        - size
        - checksum
    condition: $.steps.validate_blob.output.valid == true
    timeout_seconds: 20

  - id: trigger_providers
    name: Trigger Provider Processing
    type: api_call
    service: studio
    endpoint: /providers/trigger
    method: POST
    input_map:
      blob_id: $.input.blob_id
      provider_ids: $.input.provider_ids
      metadata: $.steps.extract_metadata.output
    timeout_seconds: 60
    on_failure: continue
    compensation:
      service: studio
      endpoint: /providers/rollback
      method: POST
      input_map:
        blob_id: $.input.blob_id
        execution_id: $.execution.id

  - id: generate_deltas
    name: Generate State Deltas
    type: api_call
    service: studio
    endpoint: /deltas/generate
    method: POST
    input_map:
      blob_id: $.input.blob_id
      provider_outputs: $.steps.trigger_providers.output
    output_map:
      deltas: $.deltas
      new_blob_ids: $.created_blobs
    timeout_seconds: 45
```

### Schema Definition in YAML

```yaml
# schemas/blob-input-schema.yaml
id: blob_input_schema_v1
provider_id: memmie-studio
name: Blob Input Schema
version: "1.0"
type: input
description: Schema for blob processing input

definition:
  type: object
  required:
    - blob_id
    - user_id
    - provider_ids
  properties:
    blob_id:
      type: string
      format: uuid
      description: The blob to process
    user_id:
      type: string
      format: uuid
      description: The user who owns the blob
    provider_ids:
      type: array
      items:
        type: string
      description: List of provider IDs to trigger
    metadata:
      type: object
      additionalProperties: true
      description: Additional metadata for processing
```

## Integration with Memmie Studio

### 1. Workflow Registration

Memmie Studio needs to register its workflows with the workflow service on startup:

```go
// internal/workflows/registration.go
package workflows

import (
    "context"
    "fmt"
    "gopkg.in/yaml.v3"
    "io/ioutil"
    "path/filepath"
)

type WorkflowRegistrar struct {
    client       *WorkflowClient
    workflowsDir string
    schemasDir   string
}

func NewWorkflowRegistrar(client *WorkflowClient, workflowsDir, schemasDir string) *WorkflowRegistrar {
    return &WorkflowRegistrar{
        client:       client,
        workflowsDir: workflowsDir,
        schemasDir:   schemasDir,
    }
}

func (r *WorkflowRegistrar) RegisterAll(ctx context.Context) error {
    // Register schemas first
    if err := r.registerSchemas(ctx); err != nil {
        return fmt.Errorf("failed to register schemas: %w", err)
    }
    
    // Then register workflows
    if err := r.registerWorkflows(ctx); err != nil {
        return fmt.Errorf("failed to register workflows: %w", err)
    }
    
    return nil
}

func (r *WorkflowRegistrar) registerSchemas(ctx context.Context) error {
    files, err := filepath.Glob(filepath.Join(r.schemasDir, "*.yaml"))
    if err != nil {
        return err
    }
    
    for _, file := range files {
        data, err := ioutil.ReadFile(file)
        if err != nil {
            return fmt.Errorf("failed to read schema file %s: %w", file, err)
        }
        
        var schema Schema
        if err := yaml.Unmarshal(data, &schema); err != nil {
            return fmt.Errorf("failed to parse schema file %s: %w", file, err)
        }
        
        if err := r.client.RegisterSchema(ctx, &schema); err != nil {
            return fmt.Errorf("failed to register schema %s: %w", schema.ID, err)
        }
    }
    
    return nil
}

func (r *WorkflowRegistrar) registerWorkflows(ctx context.Context) error {
    files, err := filepath.Glob(filepath.Join(r.workflowsDir, "*.yaml"))
    if err != nil {
        return err
    }
    
    for _, file := range files {
        data, err := ioutil.ReadFile(file)
        if err != nil {
            return fmt.Errorf("failed to read workflow file %s: %w", file, err)
        }
        
        var workflow Workflow
        if err := yaml.Unmarshal(data, &workflow); err != nil {
            return fmt.Errorf("failed to parse workflow file %s: %w", file, err)
        }
        
        if err := r.client.RegisterWorkflow(ctx, &workflow); err != nil {
            return fmt.Errorf("failed to register workflow %s: %w", workflow.ID, err)
        }
    }
    
    return nil
}
```

### 2. Provider-Workflow Mapping

Each provider in memmie-studio can define its workflows:

```yaml
# providers/book-writer.yaml
provider:
  id: book-writer
  name: Book Writing Assistant
  type: hybrid  # namespace + processor
  
workflows:
  - workflow_id: book_chapter_processing
    triggers:
      - event: onCreate
        conditions:
          - field: metadata.type
            operator: eq
            value: chapter
      - event: onUpdate
        conditions:
          - field: metadata.status
            operator: eq
            value: draft
    priority: 10
    
  - workflow_id: book_outline_generation
    triggers:
      - event: onCreate
        conditions:
          - field: metadata.type
            operator: eq
            value: outline_request
    priority: 5
```

### 3. Delta Generation from Workflow Output

The workflow service outputs are transformed into deltas for the blob storage:

```go
// internal/workflows/delta_transformer.go
package workflows

import (
    "encoding/json"
    "fmt"
    "github.com/google/uuid"
    "time"
)

type DeltaTransformer struct {
    storage DeltaStorage
}

func (t *DeltaTransformer) TransformWorkflowOutput(
    ctx context.Context,
    workflowOutput map[string]interface{},
    blobID string,
    providerID string,
) ([]Delta, error) {
    var deltas []Delta
    
    // Check for explicit delta operations in output
    if deltaOps, ok := workflowOutput["delta_operations"].([]interface{}); ok {
        for _, op := range deltaOps {
            delta := t.parseDeltaOperation(op, blobID, providerID)
            deltas = append(deltas, delta)
        }
    }
    
    // Check for blob transformations
    if transform, ok := workflowOutput["transformed_content"]; ok {
        delta := Delta{
            ID:         uuid.New().String(),
            BlobID:     blobID,
            ProviderID: providerID,
            Type:       "transform",
            Path:       "/content",
            NewValue:   transform,
            Timestamp:  time.Now(),
            Metadata: map[string]interface{}{
                "workflow_execution_id": workflowOutput["execution_id"],
            },
        }
        deltas = append(deltas, delta)
    }
    
    // Check for new blob creation
    if newBlobs, ok := workflowOutput["created_blobs"].([]interface{}); ok {
        for _, blob := range newBlobs {
            delta := Delta{
                ID:         uuid.New().String(),
                BlobID:     blobID,
                ProviderID: providerID,
                Type:       "create_derived",
                Path:       "/derived",
                NewValue:   blob,
                Timestamp:  time.Now(),
            }
            deltas = append(deltas, delta)
        }
    }
    
    return deltas, nil
}
```

## Workflow Execution Flow

1. **Blob Created/Updated** → Event published to NATS
2. **Studio Service** → Receives event, determines triggered providers
3. **For each provider**:
   - Get associated workflows
   - Execute workflows via workflow service
   - Transform outputs to deltas
   - Apply deltas to blob storage
   - Publish delta events

## Example: Book Writing Workflow

```yaml
# Complete book writing workflow
id: book_chapter_expansion
provider_id: book:my-novel
name: Chapter Expansion Pipeline
description: Expands chapter content and updates book structure

steps:
  - id: load_chapter
    name: Load Chapter Content
    type: api_call
    service: studio
    endpoint: /blobs/{blob_id}
    method: GET
    output_map:
      content: $.content
      metadata: $.metadata

  - id: expand_content
    name: AI Content Expansion
    type: api_call
    service: core
    endpoint: /generate
    method: POST
    input_map:
      prompt: |
        Expand the following chapter content with more descriptive details,
        character development, and sensory descriptions while maintaining
        the author's voice and style:
        
        {$.steps.load_chapter.output.content}
      model: gpt-4
      temperature: 0.7
      max_tokens: 3000
    timeout_seconds: 60

  - id: check_consistency
    name: Check Story Consistency
    type: api_call
    service: studio
    endpoint: /consistency/check
    method: POST
    input_map:
      book_id: $.provider_id
      chapter_content: $.steps.expand_content.output.text
      chapter_number: $.steps.load_chapter.output.metadata.chapter_number
    timeout_seconds: 45

  - id: save_expansion
    name: Save Expanded Content
    type: api_call
    service: studio
    endpoint: /blobs
    method: POST
    input_map:
      parent_id: $.input.blob_id
      content: $.steps.expand_content.output.text
      metadata:
        type: expanded_chapter
        original_id: $.input.blob_id
        consistency_score: $.steps.check_consistency.output.score
        word_count: $.steps.expand_content.output.word_count
    output_map:
      new_blob_id: $.id
      deltas: $.deltas
```

## Implementation Steps

1. **Fix Workflow Service Compilation** (Immediate)
   - Update QueuedExecution structure
   - Implement missing repository methods
   - Run database migrations

2. **Create YAML Workflow Definitions** (Phase 1)
   - Define schemas for blob processing
   - Create workflow templates for common operations
   - Set up provider-workflow mappings

3. **Integrate Studio with Workflow Service** (Phase 2)
   - Implement workflow registration on startup
   - Create workflow execution triggers
   - Build delta transformation pipeline

4. **Testing and Optimization** (Phase 3)
   - Test end-to-end workflow execution
   - Optimize for performance
   - Add monitoring and metrics

This architecture provides a flexible, scalable system for blob processing through workflows, with full version history via deltas and support for complex provider ecosystems.