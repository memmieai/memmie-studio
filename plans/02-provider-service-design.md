# Provider Service Design - AI Transformer Registry

## Overview

The Provider Service (Port 8007) manages the registry of content transformers (providers) and their instances. Each provider defines how to process blobs and what UI layout to render.

## Core Responsibilities

1. **Provider Registry**: Store provider templates and capabilities
2. **Instance Management**: Users create provider instances (e.g., "book:my-novel")
3. **UI Layout Definitions**: Each provider defines its dynamic UI
4. **Execution Orchestration**: Route blob processing to appropriate workflows
5. **Marketplace**: Future provider discovery and sharing

## Data Models

### Provider Template (PostgreSQL)
```go
type ProviderTemplate struct {
    ID          string    `db:"id"`           // e.g., "text-expander"
    Name        string    `db:"name"`         // "Text Expander"
    Category    string    `db:"category"`     // "writing", "music", "research"
    Description string    `db:"description"`
    Author      string    `db:"author"`       // Creator of template
    Version     string    `db:"version"`      // semver
    
    // Capabilities
    InputTypes  []string  `db:"input_types"`  // ["text/plain", "text/markdown"]
    OutputTypes []string  `db:"output_types"` // ["text/plain"]
    
    // Processing
    WorkflowID  string    `db:"workflow_id"`  // Reference to workflow definition
    AIModel     string    `db:"ai_model"`     // "gpt-4", "claude", "custom"
    
    // UI Definition
    UILayout    UILayout  `db:"ui_layout"`    // JSON column
    
    // Configuration Schema
    ConfigSchema map[string]interface{} `db:"config_schema"` // JSON Schema for user config
    
    // Metadata
    Tags        []string  `db:"tags"`
    Icon        string    `db:"icon"`         // URL or base64
    Examples    []Example `db:"examples"`     // Usage examples
    
    CreatedAt   time.Time `db:"created_at"`
    UpdatedAt   time.Time `db:"updated_at"`
    Published   bool      `db:"published"`    // Available in marketplace
}

type ProviderInstance struct {
    ID           string    `db:"id"`          // e.g., "book:my-novel"
    UserID       string    `db:"user_id"`
    TemplateID   string    `db:"template_id"` // References ProviderTemplate
    Name         string    `db:"name"`        // User's name for instance
    
    // User Configuration
    Config       map[string]interface{} `db:"config"` // User-specific settings
    
    // Custom UI Overrides
    UIOverrides  *UILayout `db:"ui_overrides"` // Optional UI customization
    
    // State
    Active       bool      `db:"active"`
    ProcessCount int64     `db:"process_count"` // Number of blobs processed
    LastUsed     time.Time `db:"last_used"`
    
    CreatedAt    time.Time `db:"created_at"`
    UpdatedAt    time.Time `db:"updated_at"`
}
```

### UI Layout Definition
```go
type UILayout struct {
    Type        string       `json:"type"`        // "split", "tabs", "grid", "canvas"
    Orientation string       `json:"orientation"` // "horizontal", "vertical"
    Children    []UIComponent `json:"children"`
    Responsive  ResponsiveConfig `json:"responsive"`
}

type UIComponent struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"` // See component types below
    DataSource  string                 `json:"data_source"` // JSONPath to blob data
    Props       map[string]interface{} `json:"props"`
    Actions     []UIAction             `json:"actions"`
    Visibility  string                 `json:"visibility"` // Condition expression
}

// Component Types:
// - blob-editor: Text editing with syntax highlighting
// - blob-viewer: Read-only content display
// - dag-visualizer: Graph visualization of blob relationships
// - audio-player: For music/audio blobs
// - code-editor: Monaco editor for code
// - markdown-preview: Rendered markdown
// - ramble-button: Voice input button
// - image-viewer: Image display with zoom
// - split-diff: Before/after comparison
// - metrics-panel: Stats and analytics

type UIAction struct {
    Type       string                 `json:"type"`     // "transform", "create", "delete"
    Label      string                 `json:"label"`
    Icon       string                 `json:"icon"`
    Provider   string                 `json:"provider"` // Provider to execute
    Params     map[string]interface{} `json:"params"`
    Hotkey     string                 `json:"hotkey"`   // Keyboard shortcut
}
```

## API Endpoints

### Provider Template Operations
```go
// List available provider templates
GET /api/v1/providers/templates?category=writing
Response:
{
    "templates": [
        {
            "id": "text-expander",
            "name": "Text Expander",
            "category": "writing",
            "description": "Expands brief text into detailed prose",
            "input_types": ["text/plain"],
            "tags": ["ai", "writing", "expansion"]
        }
    ]
}

// Get template details
GET /api/v1/providers/templates/{template_id}
Response:
{
    "id": "text-expander",
    "ui_layout": {
        "type": "split",
        "orientation": "horizontal",
        "children": [...]
    },
    "config_schema": {
        "type": "object",
        "properties": {
            "style": {
                "type": "string",
                "enum": ["formal", "casual", "creative"]
            }
        }
    }
}
```

### Provider Instance Operations
```go
// Create provider instance
POST /api/v1/providers/instances
Request:
{
    "template_id": "text-expander",
    "name": "My Book Writer",
    "config": {
        "style": "creative",
        "target_length": "long",
        "maintain_voice": true
    }
}
Response:
{
    "id": "book:my-novel-abc123",
    "template_id": "text-expander",
    "active": true
}

// List user's provider instances
GET /api/v1/users/{user_id}/providers
Response:
{
    "providers": [
        {
            "id": "book:my-novel-abc123",
            "name": "My Book Writer",
            "template_id": "text-expander",
            "last_used": "2024-01-01T00:00:00Z"
        }
    ]
}

// Get UI layout for provider
GET /api/v1/providers/{provider_id}/ui-layout
Response:
{
    "layout": {
        "type": "split",
        "orientation": "horizontal",
        "children": [
            {
                "id": "input-pane",
                "type": "blob-editor",
                "data_source": "$.current_blob",
                "actions": [
                    {
                        "type": "transform",
                        "label": "Expand",
                        "provider": "self"
                    }
                ]
            },
            {
                "id": "output-pane",
                "type": "blob-viewer",
                "data_source": "$.processed_blob"
            }
        ]
    }
}
```

### Provider Execution
```go
// Execute provider on blob
POST /api/v1/providers/{provider_id}/execute
Request:
{
    "blob_id": "507f1f77bcf86cd799439011",
    "user_id": "user-123",
    "params": {
        "preview": false
    }
}
Response:
{
    "execution_id": "exec-123",
    "status": "processing",
    "estimated_time": 5
}

// Get execution status
GET /api/v1/executions/{execution_id}
Response:
{
    "id": "exec-123",
    "status": "completed",
    "result": {
        "new_blob_id": "507f1f77bcf86cd799439012",
        "deltas": [...],
        "metrics": {
            "processing_time": 3.2,
            "tokens_used": 1500
        }
    }
}
```

## Provider Templates

### Text Expander Template
```json
{
    "id": "text-expander",
    "name": "Text Expander",
    "ui_layout": {
        "type": "split",
        "orientation": "horizontal",
        "children": [
            {
                "id": "draft",
                "type": "blob-editor",
                "data_source": "$.input_blob",
                "props": {
                    "title": "Draft",
                    "placeholder": "Enter your text to expand..."
                }
            },
            {
                "id": "expanded",
                "type": "blob-viewer",
                "data_source": "$.output_blob",
                "props": {
                    "title": "Expanded",
                    "show_word_count": true,
                    "show_diff": true
                }
            }
        ]
    }
}
```

### Music Generator Template (Strudel)
```json
{
    "id": "music-generator",
    "name": "Music Generator",
    "ui_layout": {
        "type": "tabs",
        "children": [
            {
                "id": "description",
                "type": "blob-editor",
                "data_source": "$.description",
                "props": {
                    "title": "Description",
                    "language": "text"
                }
            },
            {
                "id": "code",
                "type": "code-editor",
                "data_source": "$.strudel_code",
                "props": {
                    "title": "Strudel Code",
                    "language": "javascript",
                    "theme": "monokai"
                }
            },
            {
                "id": "player",
                "type": "audio-player",
                "data_source": "$.audio_blob",
                "props": {
                    "title": "Preview",
                    "show_waveform": true
                }
            }
        ]
    }
}
```

### Research Assistant Template
```json
{
    "id": "research-assistant",
    "name": "Research Assistant",
    "ui_layout": {
        "type": "grid",
        "children": [
            {
                "id": "sources",
                "type": "blob-editor",
                "data_source": "$.sources",
                "props": {
                    "title": "Sources",
                    "accept": ["text/plain", "application/pdf"]
                }
            },
            {
                "id": "graph",
                "type": "dag-visualizer",
                "data_source": "$.knowledge_graph",
                "props": {
                    "title": "Knowledge Graph",
                    "interactive": true,
                    "show_labels": true
                }
            },
            {
                "id": "summary",
                "type": "markdown-preview",
                "data_source": "$.summary",
                "props": {
                    "title": "Summary"
                }
            },
            {
                "id": "citations",
                "type": "blob-viewer",
                "data_source": "$.citations",
                "props": {
                    "title": "Citations",
                    "format": "apa"
                }
            }
        ]
    }
}
```

## Service Implementation

### Repository Layer
```go
package repository

type ProviderRepository interface {
    // Templates
    GetTemplate(ctx context.Context, templateID string) (*ProviderTemplate, error)
    ListTemplates(ctx context.Context, filter TemplateFilter) ([]*ProviderTemplate, error)
    CreateTemplate(ctx context.Context, template *ProviderTemplate) error
    
    // Instances
    CreateInstance(ctx context.Context, instance *ProviderInstance) error
    GetInstance(ctx context.Context, instanceID string) (*ProviderInstance, error)
    ListUserInstances(ctx context.Context, userID string) ([]*ProviderInstance, error)
    UpdateInstance(ctx context.Context, instance *ProviderInstance) error
    DeleteInstance(ctx context.Context, instanceID string) error
}

type PostgresProviderRepository struct {
    db *sqlx.DB
}

func (r *PostgresProviderRepository) CreateInstance(ctx context.Context, instance *ProviderInstance) error {
    // Generate unique instance ID
    instance.ID = fmt.Sprintf("%s:%s-%s", 
        instance.TemplateID, 
        slug.Make(instance.Name),
        shortid.Generate())
    
    query := `
        INSERT INTO provider_instances 
        (id, user_id, template_id, name, config, active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
    
    configJSON, _ := json.Marshal(instance.Config)
    _, err := r.db.ExecContext(ctx, query,
        instance.ID,
        instance.UserID,
        instance.TemplateID,
        instance.Name,
        configJSON,
        true,
        time.Now(),
        time.Now(),
    )
    
    return err
}
```

### Service Layer
```go
package service

type ProviderService struct {
    repo         ProviderRepository
    workflowClient WorkflowClient
    stateClient  StateClient
    eventBus     EventBus
}

func (s *ProviderService) CreateInstance(ctx context.Context, userID string, req CreateInstanceRequest) (*ProviderInstance, error) {
    // Get template
    template, err := s.repo.GetTemplate(ctx, req.TemplateID)
    if err != nil {
        return nil, err
    }
    
    // Validate config against schema
    if err := s.validateConfig(req.Config, template.ConfigSchema); err != nil {
        return nil, err
    }
    
    instance := &ProviderInstance{
        UserID:     userID,
        TemplateID: req.TemplateID,
        Name:       req.Name,
        Config:     req.Config,
        Active:     true,
    }
    
    if err := s.repo.CreateInstance(ctx, instance); err != nil {
        return nil, err
    }
    
    // Publish event
    s.eventBus.Publish(ctx, "provider.created", ProviderCreatedEvent{
        ProviderID: instance.ID,
        UserID:     userID,
        TemplateID: req.TemplateID,
    })
    
    return instance, nil
}

func (s *ProviderService) ExecuteProvider(ctx context.Context, providerID string, blobID string) (*ExecutionResult, error) {
    // Get provider instance
    instance, err := s.repo.GetInstance(ctx, providerID)
    if err != nil {
        return nil, err
    }
    
    // Get template
    template, err := s.repo.GetTemplate(ctx, instance.TemplateID)
    if err != nil {
        return nil, err
    }
    
    // Get blob from State Service
    blob, err := s.stateClient.GetBlob(ctx, instance.UserID, blobID)
    if err != nil {
        return nil, err
    }
    
    // Execute workflow
    execution, err := s.workflowClient.Execute(ctx, WorkflowExecutionRequest{
        WorkflowID: template.WorkflowID,
        Input: map[string]interface{}{
            "blob":     blob,
            "config":   instance.Config,
            "provider": providerID,
        },
    })
    
    if err != nil {
        return nil, err
    }
    
    // Update usage metrics
    s.repo.UpdateInstance(ctx, &ProviderInstance{
        ID:           providerID,
        ProcessCount: instance.ProcessCount + 1,
        LastUsed:     time.Now(),
    })
    
    return &ExecutionResult{
        ExecutionID: execution.ID,
        Status:      execution.Status,
    }, nil
}

func (s *ProviderService) GetUILayout(ctx context.Context, providerID string) (*UILayout, error) {
    instance, err := s.repo.GetInstance(ctx, providerID)
    if err != nil {
        return nil, err
    }
    
    template, err := s.repo.GetTemplate(ctx, instance.TemplateID)
    if err != nil {
        return nil, err
    }
    
    // Return UI overrides if present, otherwise template UI
    if instance.UIOverrides != nil {
        return instance.UIOverrides, nil
    }
    
    return &template.UILayout, nil
}
```

## Provider Matching

```go
// Match providers that should process a blob
func (s *ProviderService) MatchProviders(ctx context.Context, blob *Blob, event string) ([]*ProviderInstance, error) {
    // Get user's active providers
    instances, err := s.repo.ListUserInstances(ctx, blob.UserID)
    if err != nil {
        return nil, err
    }
    
    var matched []*ProviderInstance
    
    for _, instance := range instances {
        if !instance.Active {
            continue
        }
        
        template, _ := s.repo.GetTemplate(ctx, instance.TemplateID)
        
        // Check if provider handles this content type
        if !contains(template.InputTypes, blob.ContentType) {
            continue
        }
        
        // Check if provider has trigger for this event
        if config, ok := instance.Config["triggers"].(map[string]interface{}); ok {
            if triggers, ok := config[event].([]interface{}); ok {
                if s.evaluateTriggers(blob, triggers) {
                    matched = append(matched, instance)
                }
            }
        }
    }
    
    return matched, nil
}
```

## Database Schema

```sql
CREATE TABLE provider_templates (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(50),
    description TEXT,
    author VARCHAR(255),
    version VARCHAR(20),
    input_types TEXT[],
    output_types TEXT[],
    workflow_id VARCHAR(100),
    ai_model VARCHAR(50),
    ui_layout JSONB NOT NULL,
    config_schema JSONB,
    tags TEXT[],
    icon TEXT,
    examples JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    published BOOLEAN DEFAULT false
);

CREATE TABLE provider_instances (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    template_id VARCHAR(100) REFERENCES provider_templates(id),
    name VARCHAR(255) NOT NULL,
    config JSONB,
    ui_overrides JSONB,
    active BOOLEAN DEFAULT true,
    process_count BIGINT DEFAULT 0,
    last_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_user_providers (user_id, active),
    INDEX idx_template (template_id)
);

CREATE TABLE provider_executions (
    id VARCHAR(100) PRIMARY KEY,
    provider_id VARCHAR(255) REFERENCES provider_instances(id),
    blob_id VARCHAR(100),
    status VARCHAR(50),
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    error TEXT,
    metrics JSONB
);
```

## Configuration

```yaml
# config/provider-service.yaml
service:
  port: 8007
  name: provider-service

database:
  url: postgresql://user:pass@localhost:5432/providers
  max_connections: 25

clients:
  workflow_service: http://localhost:8005
  state_service: http://localhost:8006
  
templates:
  auto_load: true
  directory: ./templates
  
marketplace:
  enabled: false  # Future feature
  api_url: https://marketplace.memmie.ai
  
cache:
  redis_url: redis://localhost:6379
  ttl: 10m
```

This Provider Service design enables a flexible, extensible system for content transformation with dynamic UI generation.