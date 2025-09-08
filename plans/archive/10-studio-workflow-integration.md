# Memmie Studio - Workflow Service Integration Guide

## Overview

This guide details how Memmie Studio integrates with the Workflow Service to enable delta-driven blob processing through provider pipelines.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│                 │     │                  │     │                 │
│  Memmie Studio  │────▶│ Workflow Service │────▶│  Other Services │
│   (Port 8010)   │     │   (Port 8005)    │     │  (Ports 8001-9) │
│                 │     │                  │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
        │                        │                         │
        ▼                        ▼                         ▼
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Blob Storage   │     │ Workflow Defs    │     │   Provider APIs │
│   (PostgreSQL)  │     │   (PostgreSQL)   │     │                 │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

## Integration Flow

### 1. Service Startup

When Memmie Studio starts, it:

1. **Connects to Workflow Service**
   ```go
   workflowClient := workflows.NewWorkflowClient("http://localhost:8005")
   ```

2. **Loads YAML Definitions**
   ```go
   loader := workflows.NewWorkflowLoader(
       workflowClient,
       "./workflows",
       "./schemas",
       "./providers",
   )
   ```

3. **Registers Workflows and Schemas**
   ```go
   if err := loader.LoadAndRegisterAll(ctx); err != nil {
       log.Fatal("Failed to register workflows:", err)
   }
   ```

### 2. Blob Processing Trigger

When a blob is created or updated:

1. **Event Published to NATS**
   ```go
   event := BlobEvent{
       Type:       "blob.created",
       BlobID:     blob.ID,
       UserID:     userID,
       ProviderID: providerID,
       Metadata:   blob.Metadata,
   }
   nats.Publish("studio.blob.events", event)
   ```

2. **Provider Matching**
   ```go
   providers := orchestrator.GetTriggeredProviders(event.Type)
   ```

3. **Workflow Execution**
   ```go
   for _, provider := range providers {
       req := ExecutionRequest{
           WorkflowID: provider.WorkflowID,
           Input: map[string]interface{}{
               "blob_id":      event.BlobID,
               "user_id":      event.UserID,
               "provider_ids": []string{provider.ID},
           },
       }
       resp, err := workflowClient.ExecuteWorkflow(ctx, req)
   }
   ```

### 3. Delta Generation

Workflow outputs are transformed into deltas:

```go
deltas := transformer.TransformWorkflowOutput(
    workflowOutput,
    blobID,
    providerID,
)

for _, delta := range deltas {
    deltaStorage.Store(ctx, delta)
}
```

## Service Endpoints

### Memmie Studio Endpoints (Port 8010)

```yaml
# Blob Management
GET    /api/v1/blobs/{id}           # Get blob by ID
POST   /api/v1/blobs                # Create new blob
PUT    /api/v1/blobs/{id}           # Update blob
DELETE /api/v1/blobs/{id}           # Delete blob

# Provider Management
GET    /api/v1/providers            # List providers
POST   /api/v1/providers            # Register provider
GET    /api/v1/providers/{id}       # Get provider details
POST   /api/v1/providers/match      # Match providers for blob

# Workflow Integration
POST   /api/v1/providers/execute-chain  # Execute provider chain
POST   /api/v1/providers/trigger        # Trigger provider processing
POST   /api/v1/providers/rollback       # Rollback provider changes

# Delta Management
POST   /api/v1/deltas/generate      # Generate deltas from changes
POST   /api/v1/deltas/apply         # Apply deltas to blob
GET    /api/v1/deltas/history/{id}  # Get delta history for blob

# Validation & Analysis
POST   /api/v1/blobs/validate       # Validate blob structure
POST   /api/v1/validation/chapter   # Validate chapter structure
POST   /api/v1/analysis/characters  # Extract character information
POST   /api/v1/consistency/check    # Check consistency

# Events
POST   /api/v1/events/emit          # Emit processing events
WS     /api/v1/events/stream        # WebSocket event stream
```

### Workflow Service Integration Points

The Workflow Service needs these endpoints to call back to Studio:

```yaml
# Called by workflow steps
POST /api/v1/studio/callbacks/step-complete
POST /api/v1/studio/callbacks/step-failed
POST /api/v1/studio/callbacks/workflow-complete
```

## Configuration

### Studio Service Configuration

```yaml
# config/studio.yaml
service:
  name: memmie-studio
  port: 8010
  
workflow:
  service_url: http://localhost:8005
  timeout: 120s
  max_retries: 3
  
storage:
  postgres:
    url: postgresql://user:pass@localhost:5432/studio?sslmode=disable
    max_connections: 25
    
nats:
  url: nats://memmie:memmiepass@localhost:4222
  subjects:
    blob_events: studio.blob.events
    delta_events: studio.delta.events
    
providers:
  config_dir: ./providers
  workflows_dir: ./workflows
  schemas_dir: ./schemas
  auto_load: true
```

### Environment Variables

```bash
# Studio Service
STUDIO_PORT=8010
STUDIO_DATABASE_URL=postgresql://user:pass@localhost:5432/studio?sslmode=disable

# Workflow Service Connection
WORKFLOW_SERVICE_URL=http://localhost:8005
WORKFLOW_API_KEY=studio-api-key

# Other Service URLs
AUTH_SERVICE_URL=http://localhost:8001
CORE_SERVICE_URL=http://localhost:8004
MEMORY_SERVICE_URL=http://localhost:8003

# NATS Configuration
NATS_URL=nats://memmie:memmiepass@localhost:4222

# Redis (for caching)
REDIS_URL=redis://localhost:6379/0
```

## Implementation Checklist

### Phase 1: Core Integration (Week 1)
- [ ] Fix workflow service compilation errors
- [ ] Create Studio service structure
- [ ] Implement workflow client
- [ ] Set up YAML loading system
- [ ] Create basic API endpoints

### Phase 2: Blob Processing (Week 2)
- [ ] Implement blob storage with PostgreSQL
- [ ] Create delta generation system
- [ ] Build provider matching logic
- [ ] Implement workflow execution triggers
- [ ] Add event publishing to NATS

### Phase 3: Provider System (Week 3)
- [ ] Create provider registration
- [ ] Implement namespace providers
- [ ] Build provider-workflow mappings
- [ ] Add provider configuration management
- [ ] Create provider marketplace structure

### Phase 4: Advanced Features (Week 4)
- [ ] Implement DAG processing
- [ ] Add compensation/rollback support
- [ ] Create WebSocket real-time updates
- [ ] Build monitoring and metrics
- [ ] Add caching layer with Redis

## Testing Strategy

### Unit Tests
```go
// internal/workflows/orchestrator_test.go
func TestOrchestratorProcessBlob(t *testing.T) {
    // Test blob processing through providers
}

func TestDeltaGeneration(t *testing.T) {
    // Test delta generation from workflow output
}
```

### Integration Tests
```go
// tests/integration/workflow_integration_test.go
func TestEndToEndBlobProcessing(t *testing.T) {
    // 1. Create blob
    // 2. Trigger workflow
    // 3. Verify deltas generated
    // 4. Check blob state updated
}
```

### Load Tests
```go
// tests/performance/load_test.go
func TestConcurrentBlobProcessing(t *testing.T) {
    // Test with 100 concurrent blob operations
}
```

## Monitoring & Observability

### Metrics to Track
- Workflow execution time
- Delta generation rate
- Provider processing time
- Blob storage operations/sec
- Error rates by provider
- Queue depth for async operations

### Logging
```go
logger.Info("workflow.executed",
    "workflow_id", workflowID,
    "blob_id", blobID,
    "duration_ms", duration,
    "deltas_generated", len(deltas),
)
```

### Health Checks
```go
// GET /health
{
  "status": "healthy",
  "services": {
    "workflow": "connected",
    "database": "healthy",
    "nats": "connected",
    "redis": "connected"
  },
  "providers": {
    "total": 15,
    "active": 12,
    "failed": 3
  }
}
```

## Example: Complete Book Chapter Processing

1. **User creates chapter blob**
   ```json
   POST /api/v1/blobs
   {
     "namespace_id": "book:my-novel",
     "content": "Chapter 1 content...",
     "metadata": {
       "type": "chapter",
       "chapter_number": 1,
       "status": "draft"
     }
   }
   ```

2. **Studio triggers workflow**
   - Matches book-writer provider
   - Executes book_chapter_processing workflow
   - Workflow calls AI service for expansion
   - Checks consistency
   - Generates summary

3. **Deltas generated and applied**
   ```json
   {
     "deltas": [
       {
         "type": "create_derived",
         "path": "/derived/expanded",
         "value": {
           "blob_id": "expanded-chapter-uuid",
           "content": "Expanded chapter content..."
         }
       },
       {
         "type": "update",
         "path": "/metadata/consistency_score",
         "value": 0.92
       }
     ]
   }
   ```

4. **User sees updated chapter with expansions**

## Troubleshooting

### Common Issues

1. **Workflow service not responding**
   - Check if service is running on port 8005
   - Verify DATABASE_URL is correct
   - Check for compilation errors

2. **Deltas not being generated**
   - Verify workflow output format
   - Check delta transformer logic
   - Ensure database migrations ran

3. **Providers not triggering**
   - Check provider registration
   - Verify trigger conditions
   - Review NATS event publishing

4. **Performance issues**
   - Enable Redis caching
   - Increase workflow concurrency
   - Optimize database queries
   - Use async processing for non-critical paths

## Next Steps

1. Implement the workflow service fixes
2. Create the Studio service with basic endpoints
3. Test end-to-end blob processing
4. Add more provider types
5. Build the client SDKs
6. Create documentation and examples