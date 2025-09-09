# Schema Service Design

## Overview

The Schema Service (Port 8011) is the central authority for all data schemas in the ReYNa Studio ecosystem. It provides schema registration, versioning, validation, and transformation capabilities for all processors and blobs.

## Core Responsibilities

1. **Schema Registry**: Store and manage all processor schemas
2. **Version Control**: Handle schema evolution with semantic versioning
3. **Validation Service**: Validate data against schemas
4. **Transformation Rules**: Define mappings between schema versions
5. **Usage Analytics**: Track schema usage across the system

## Database Design (PostgreSQL)

### Schema Tables

```sql
-- Main schema definitions
CREATE TABLE schemas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    processor_id    VARCHAR(255) NOT NULL,
    version         VARCHAR(50) NOT NULL,  -- semver: 1.0.0
    
    -- JSON Schema definition
    definition      JSONB NOT NULL,
    
    -- Validation configuration
    strict_mode     BOOLEAN DEFAULT true,
    allow_additional BOOLEAN DEFAULT false,
    
    -- Metadata
    description     TEXT,
    tags            TEXT[],
    examples        JSONB,
    
    -- Status
    status          VARCHAR(50) DEFAULT 'draft', -- draft, active, deprecated
    deprecated_at   TIMESTAMP,
    sunset_date     TIMESTAMP,
    replacement_id  UUID REFERENCES schemas(id),
    
    -- Audit
    created_by      VARCHAR(255),
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(name, processor_id, version)
);

-- Schema relationships and compatibility
CREATE TABLE schema_compatibility (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_id       UUID REFERENCES schemas(id),
    compatible_with UUID REFERENCES schemas(id),
    
    -- Compatibility type
    compatibility   VARCHAR(50), -- full, backward, forward, none
    
    -- Optional transformation
    transform_rules JSONB,
    
    created_at      TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(schema_id, compatible_with)
);

-- Schema validation history
CREATE TABLE validation_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_id       UUID REFERENCES schemas(id),
    blob_id         VARCHAR(255),
    user_id         VARCHAR(255),
    processor_id    VARCHAR(255),
    
    -- Validation result
    valid           BOOLEAN NOT NULL,
    errors          JSONB,
    warnings        JSONB,
    
    -- Performance metrics
    validation_ms   INTEGER,
    data_size_bytes INTEGER,
    
    validated_at    TIMESTAMP DEFAULT NOW(),
    
    -- Indexes for analytics
    INDEX idx_validation_schema (schema_id, validated_at),
    INDEX idx_validation_user (user_id, validated_at)
);

-- Schema usage statistics
CREATE TABLE schema_stats (
    schema_id       UUID REFERENCES schemas(id),
    date            DATE NOT NULL,
    
    -- Counters
    validation_count INTEGER DEFAULT 0,
    success_count   INTEGER DEFAULT 0,
    failure_count   INTEGER DEFAULT 0,
    
    -- Performance
    avg_validation_ms FLOAT,
    p95_validation_ms FLOAT,
    p99_validation_ms FLOAT,
    
    -- Data metrics
    total_bytes     BIGINT DEFAULT 0,
    unique_users    INTEGER DEFAULT 0,
    
    PRIMARY KEY(schema_id, date)
);

-- Indexes for performance
CREATE INDEX idx_schemas_processor ON schemas(processor_id, status);
CREATE INDEX idx_schemas_name ON schemas(name, version);
CREATE INDEX idx_schemas_status ON schemas(status) WHERE status = 'active';
```

## Domain Models

```go
package domain

import (
    "time"
    "github.com/google/uuid"
    "github.com/xeipuuv/gojsonschema"
)

type Schema struct {
    ID              uuid.UUID              `json:"id" db:"id"`
    Name            string                 `json:"name" db:"name"`
    ProcessorID     string                 `json:"processor_id" db:"processor_id"`
    Version         string                 `json:"version" db:"version"`
    
    // Schema definition
    Definition      map[string]interface{} `json:"definition" db:"definition"`
    
    // Validation config
    StrictMode      bool                   `json:"strict_mode" db:"strict_mode"`
    AllowAdditional bool                   `json:"allow_additional" db:"allow_additional"`
    
    // Metadata
    Description     string                 `json:"description" db:"description"`
    Tags            []string               `json:"tags" db:"tags"`
    Examples        []interface{}          `json:"examples" db:"examples"`
    
    // Status
    Status          SchemaStatus           `json:"status" db:"status"`
    DeprecatedAt    *time.Time             `json:"deprecated_at" db:"deprecated_at"`
    SunsetDate      *time.Time             `json:"sunset_date" db:"sunset_date"`
    ReplacementID   *uuid.UUID             `json:"replacement_id" db:"replacement_id"`
    
    // Audit
    CreatedBy       string                 `json:"created_by" db:"created_by"`
    CreatedAt       time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

type SchemaStatus string

const (
    SchemaStatusDraft      SchemaStatus = "draft"
    SchemaStatusActive     SchemaStatus = "active"
    SchemaStatusDeprecated SchemaStatus = "deprecated"
)

type ValidationResult struct {
    Valid      bool                   `json:"valid"`
    Errors     []ValidationError      `json:"errors,omitempty"`
    Warnings   []ValidationWarning    `json:"warnings,omitempty"`
    Metadata   ValidationMetadata     `json:"metadata"`
}

type ValidationError struct {
    Field       string                 `json:"field"`
    Value       interface{}            `json:"value,omitempty"`
    Message     string                 `json:"message"`
    SchemaPath  string                 `json:"schema_path"`
}

type ValidationMetadata struct {
    SchemaID        uuid.UUID          `json:"schema_id"`
    SchemaVersion   string             `json:"schema_version"`
    ValidationTime  time.Duration      `json:"validation_time"`
    DataSize        int                `json:"data_size_bytes"`
}
```

## Service Implementation

```go
package service

import (
    "context"
    "fmt"
    "time"
    
    "github.com/xeipuuv/gojsonschema"
    "github.com/memmieai/schema-service/internal/domain"
    "github.com/memmieai/schema-service/internal/repository"
)

type SchemaService struct {
    repo      repository.SchemaRepository
    cache     CacheService
    validator Validator
}

func NewSchemaService(repo repository.SchemaRepository) *SchemaService {
    return &SchemaService{
        repo:      repo,
        cache:     NewRedisCacheService(),
        validator: NewJSONSchemaValidator(),
    }
}

// RegisterSchema creates a new schema version
func (s *SchemaService) RegisterSchema(ctx context.Context, req RegisterSchemaRequest) (*domain.Schema, error) {
    // Validate schema definition itself
    if err := s.validateSchemaDefinition(req.Definition); err != nil {
        return nil, fmt.Errorf("invalid schema definition: %w", err)
    }
    
    // Check for existing versions
    existing, err := s.repo.GetLatestVersion(ctx, req.Name, req.ProcessorID)
    if err == nil && existing != nil {
        // Validate version is higher
        if !isHigherVersion(req.Version, existing.Version) {
            return nil, fmt.Errorf("version %s must be higher than %s", req.Version, existing.Version)
        }
        
        // Check compatibility if specified
        if req.CompatibleWith != nil {
            if err := s.checkCompatibility(req.Definition, existing.Definition); err != nil {
                return nil, fmt.Errorf("incompatible with previous version: %w", err)
            }
        }
    }
    
    schema := &domain.Schema{
        Name:        req.Name,
        ProcessorID: req.ProcessorID,
        Version:     req.Version,
        Definition:  req.Definition,
        Description: req.Description,
        Status:      domain.SchemaStatusDraft,
        CreatedBy:   req.CreatedBy,
        CreatedAt:   time.Now(),
    }
    
    if err := s.repo.Create(ctx, schema); err != nil {
        return nil, fmt.Errorf("failed to create schema: %w", err)
    }
    
    // Clear cache for this processor
    s.cache.InvalidatePattern(fmt.Sprintf("schema:%s:*", req.ProcessorID))
    
    return schema, nil
}

// ValidateData validates data against a schema
func (s *SchemaService) ValidateData(ctx context.Context, schemaID uuid.UUID, data interface{}) (*domain.ValidationResult, error) {
    start := time.Now()
    
    // Get schema from cache or DB
    schema, err := s.getSchemaWithCache(ctx, schemaID)
    if err != nil {
        return nil, fmt.Errorf("schema not found: %w", err)
    }
    
    // Perform validation
    result := s.validator.Validate(schema.Definition, data)
    
    // Log validation
    s.logValidation(ctx, schemaID, result, time.Since(start))
    
    return result, nil
}

// GetSchemaByID retrieves a schema
func (s *SchemaService) GetSchemaByID(ctx context.Context, id uuid.UUID) (*domain.Schema, error) {
    return s.getSchemaWithCache(ctx, id)
}

// ListProcessorSchemas lists all schemas for a processor
func (s *SchemaService) ListProcessorSchemas(ctx context.Context, processorID string) ([]*domain.Schema, error) {
    return s.repo.ListByProcessor(ctx, processorID, domain.SchemaStatusActive)
}

// ActivateSchema marks a schema as active
func (s *SchemaService) ActivateSchema(ctx context.Context, id uuid.UUID) error {
    schema, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }
    
    if schema.Status != domain.SchemaStatusDraft {
        return fmt.Errorf("only draft schemas can be activated")
    }
    
    // Run validation tests with examples
    for i, example := range schema.Examples {
        result := s.validator.Validate(schema.Definition, example)
        if !result.Valid {
            return fmt.Errorf("example %d failed validation: %v", i, result.Errors)
        }
    }
    
    schema.Status = domain.SchemaStatusActive
    schema.UpdatedAt = time.Now()
    
    return s.repo.Update(ctx, schema)
}

// Internal helpers

func (s *SchemaService) getSchemaWithCache(ctx context.Context, id uuid.UUID) (*domain.Schema, error) {
    cacheKey := fmt.Sprintf("schema:%s", id)
    
    // Check cache
    if cached, err := s.cache.Get(cacheKey); err == nil {
        return cached.(*domain.Schema), nil
    }
    
    // Load from DB
    schema, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Cache for 1 hour
    s.cache.Set(cacheKey, schema, time.Hour)
    
    return schema, nil
}

func (s *SchemaService) validateSchemaDefinition(definition map[string]interface{}) error {
    // Validate it's a valid JSON Schema
    loader := gojsonschema.NewGoLoader(definition)
    _, err := gojsonschema.NewSchema(loader)
    return err
}

func (s *SchemaService) checkCompatibility(new, old map[string]interface{}) error {
    // Simple compatibility check - ensure required fields in old are in new
    oldRequired := getRequiredFields(old)
    newRequired := getRequiredFields(new)
    
    for _, field := range oldRequired {
        if !contains(newRequired, field) {
            return fmt.Errorf("required field '%s' removed", field)
        }
    }
    
    return nil
}

func (s *SchemaService) logValidation(ctx context.Context, schemaID uuid.UUID, result *domain.ValidationResult, duration time.Duration) {
    log := &domain.ValidationLog{
        SchemaID:     schemaID,
        Valid:        result.Valid,
        ValidationMs: int(duration.Milliseconds()),
        ValidatedAt:  time.Now(),
    }
    
    if !result.Valid {
        log.Errors = result.Errors
    }
    
    // Async logging to not block validation
    go s.repo.LogValidation(context.Background(), log)
}
```

## API Endpoints

```yaml
# Schema Registration
POST /api/v1/schemas
Request:
  {
    "name": "text-input",
    "processor_id": "text-expansion",
    "version": "1.0.0",
    "definition": { /* JSON Schema */ },
    "description": "Input schema for text expansion",
    "examples": [
      {"content": "Sample text", "metadata": {}}
    ]
  }
Response:
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "text-input",
    "version": "1.0.0",
    "status": "draft"
  }

# Validate Data
POST /api/v1/schemas/{schema_id}/validate
Request:
  {
    "data": {
      "content": "The ship sailed into the storm.",
      "metadata": {"chapter": 1}
    }
  }
Response:
  {
    "valid": true,
    "metadata": {
      "schema_id": "550e8400-e29b-41d4-a716-446655440000",
      "schema_version": "1.0.0",
      "validation_time": 12,
      "data_size_bytes": 64
    }
  }

# Get Schema
GET /api/v1/schemas/{schema_id}
Response:
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "text-input",
    "processor_id": "text-expansion",
    "version": "1.0.0",
    "definition": { /* JSON Schema */ },
    "status": "active"
  }

# List Processor Schemas
GET /api/v1/processors/{processor_id}/schemas
Response:
  {
    "schemas": [
      {
        "id": "...",
        "name": "text-input",
        "version": "1.0.0",
        "status": "active"
      },
      {
        "id": "...",
        "name": "text-input",
        "version": "1.1.0",
        "status": "draft"
      }
    ]
  }

# Activate Schema
PUT /api/v1/schemas/{schema_id}/activate
Response:
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "active"
  }

# Get Schema Stats
GET /api/v1/schemas/{schema_id}/stats?days=30
Response:
  {
    "schema_id": "550e8400-e29b-41d4-a716-446655440000",
    "period": "30d",
    "total_validations": 15234,
    "success_rate": 0.98,
    "avg_validation_ms": 23,
    "unique_users": 89,
    "daily_stats": [...]
  }
```

## Integration with Other Services

### State Service Integration
```go
// State Service validates before storing blob
func (s *StateService) CreateBlob(ctx context.Context, req CreateBlobRequest) (*Blob, error) {
    // Call Schema Service for validation
    validationResult, err := s.schemaClient.ValidateData(ctx, req.SchemaID, req.Data)
    if err != nil {
        return nil, fmt.Errorf("schema validation failed: %w", err)
    }
    
    if !validationResult.Valid {
        return nil, fmt.Errorf("data validation failed: %v", validationResult.Errors)
    }
    
    // Proceed with blob creation
    blob := &Blob{
        UserID:     req.UserID,
        SchemaID:   req.SchemaID,
        Data:       req.Data,
        CreatedAt:  time.Now(),
    }
    
    return s.repo.CreateBlob(ctx, blob)
}
```

### Processor Service Integration
```go
// Processor validates input/output schemas
func (p *TextExpansionProcessor) Process(ctx context.Context, input BlobData) (*BlobData, error) {
    // Validate input
    if err := p.schemaClient.ValidateData(ctx, p.InputSchemaID, input); err != nil {
        return nil, fmt.Errorf("input validation failed: %w", err)
    }
    
    // Process data
    output := p.expandText(input)
    
    // Validate output
    if err := p.schemaClient.ValidateData(ctx, p.OutputSchemaID, output); err != nil {
        return nil, fmt.Errorf("output validation failed: %w", err)
    }
    
    return output, nil
}
```

## Schema Evolution

### Version Migration
```yaml
# Version 1.0.0
{
  "type": "object",
  "properties": {
    "content": {"type": "string"},
    "metadata": {"type": "object"}
  },
  "required": ["content"]
}

# Version 1.1.0 - Added new optional field
{
  "type": "object",
  "properties": {
    "content": {"type": "string"},
    "metadata": {"type": "object"},
    "style": {"type": "string", "enum": ["formal", "casual"]}
  },
  "required": ["content"]
}

# Version 2.0.0 - Breaking change
{
  "type": "object",
  "properties": {
    "text": {"type": "string"},        # Renamed from 'content'
    "metadata": {"type": "object"},
    "style": {"type": "string"}
  },
  "required": ["text"]
}
```

### Compatibility Rules
1. **Backward Compatible**: New version can read old data
2. **Forward Compatible**: Old version can read new data
3. **Full Compatible**: Both directions work
4. **Breaking Change**: Requires migration

## Performance Optimizations

1. **Caching Strategy**
   - Cache schemas in Redis for 1 hour
   - Cache validation results for immutable data
   - Use bloom filters for quick existence checks

2. **Validation Performance**
   - Compile schemas once and reuse
   - Parallel validation for array items
   - Short-circuit on first error in non-strict mode

3. **Database Optimizations**
   - Partial indexes on active schemas
   - JSONB GIN indexes for schema queries
   - Partitioned validation_logs by month

## Monitoring and Metrics

### Key Metrics
- Schema registration rate
- Validation success rate by schema
- Average validation latency
- Schema version adoption
- Breaking changes per month

### Alerts
- Validation success rate < 95%
- P99 validation latency > 100ms
- Schema not used in 30 days
- Deprecated schema still in use

## Implementation Priority

### MVP Phase 1
1. Basic schema CRUD
2. JSON Schema validation
3. Simple versioning

### MVP Phase 2
1. Validation logging
2. Cache layer
3. Basic statistics

### Post-MVP
1. Schema transformation
2. Advanced compatibility checking
3. Schema marketplace
4. Visual schema editor