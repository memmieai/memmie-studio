# Memmie Studio

A reactive blob processing system with DAG-based transformations and delta-driven state management for the Memmie platform.

## Overview

Memmie Studio provides a flexible, provider-driven data processing pipeline where:
- **Blobs** are versioned data units stored per user
- **Providers** are processors that react to blob changes
- **Workflows** define transformation pipelines that produce deltas
- **DAGs** represent the relationships between original and derived blobs
- **Deltas** provide an audit trail and enable version control

## Architecture

### Core Concepts

#### 1. Blob Storage
- User-scoped blob stores with versioned content
- Each blob contains:
  - Raw data (content)
  - Metadata (type, created_at, version)
  - Processing state (which providers have processed it)
  - DAG relationships (parent/child nodes)
  - Delta history (all changes over time)

#### 2. Delta System
Instead of directly mutating blobs, all changes are applied through deltas:
```go
type Delta struct {
    ID          uuid.UUID
    BlobID      uuid.UUID
    ProviderID  string
    Operation   string // create, update, delete, transform
    Patch       json.RawMessage
    Metadata    map[string]interface{}
    AppliedAt   time.Time
}
```

The current blob state is materialized from the delta history, providing:
- Complete audit trail
- Time-travel capabilities
- Conflict resolution
- Rollback functionality

#### 3. Provider System
Providers are autonomous processors that:
- Subscribe to blob events (onCreate, onEdit, onDelete)
- Execute workflows to process blobs
- Produce deltas that create derived blobs
- Maintain their processing state

Example providers:
- **ContentExpander**: Expands brief text into detailed content
- **Summarizer**: Creates summaries of long content
- **Translator**: Translates content to other languages
- **Validator**: Checks content against rules
- **Enhancer**: Improves writing quality

#### 4. DAG Processing
Blob transformations form a Directed Acyclic Graph:
```
Original Blob
    ├── Expanded Version (by ContentExpander)
    │   └── Summary (by Summarizer)
    ├── Translation (by Translator)
    └── Enhanced Version (by Enhancer)
        └── Final Review (by Validator)
```

When a parent blob is edited:
1. Edit event propagates through the DAG
2. Child blobs are marked for reprocessing
3. Providers re-execute their workflows
4. New deltas update the derived blobs

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway                          │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                      Studio Service                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Blob Manager │  │Delta Engine  │  │ DAG Processor│     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│Workflow Service│    │Provider Service│    │ Event Bus    │
│  (Temporal)    │    │   (Registry)   │    │   (NATS)    │
└───────────────┘    └───────────────┘    └───────────────┘
        │                      │                      │
┌───────────────────────────────────────────────────────────┐
│                     PostgreSQL + S3                        │
│  Deltas | Metadata | DAG Relations | Blob Storage         │
└───────────────────────────────────────────────────────────┘
```

## API Design

### Blob Operations

```http
# Create blob
POST /api/v1/studio/blobs
{
  "content": "The quick brown fox",
  "type": "text/plain",
  "metadata": {}
}

# Get blob (with version)
GET /api/v1/studio/blobs/{blobId}?version=3

# Update blob (creates delta)
PATCH /api/v1/studio/blobs/{blobId}
{
  "delta": {
    "operation": "update",
    "patch": { "content": "The quick brown fox jumps" }
  }
}

# Get blob history
GET /api/v1/studio/blobs/{blobId}/history

# Get DAG relationships
GET /api/v1/studio/blobs/{blobId}/dag
```

### Provider Operations

```http
# Register provider
POST /api/v1/studio/providers
{
  "id": "content-expander",
  "name": "Content Expander",
  "description": "Expands brief text into detailed content",
  "events": ["onCreate", "onEdit"],
  "workflow": "expand-content-workflow"
}

# Process blob (provider endpoint)
POST /api/v1/studio/providers/{providerId}/process
{
  "blobId": "123",
  "event": "onCreate"
}
```

## Database Schema

### Core Tables

```sql
-- Blobs table (current materialized state)
CREATE TABLE blobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    content BYTEA,
    content_type VARCHAR(255),
    version INTEGER DEFAULT 1,
    parent_blob_id UUID REFERENCES blobs(id),
    provider_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Deltas table (event sourcing)
CREATE TABLE deltas (
    id UUID PRIMARY KEY,
    blob_id UUID REFERENCES blobs(id),
    provider_id VARCHAR(255),
    operation VARCHAR(50),
    patch JSONB,
    metadata JSONB,
    applied_at TIMESTAMP,
    created_by UUID
);

-- DAG edges
CREATE TABLE blob_edges (
    id UUID PRIMARY KEY,
    parent_blob_id UUID REFERENCES blobs(id),
    child_blob_id UUID REFERENCES blobs(id),
    provider_id VARCHAR(255),
    transform_type VARCHAR(255),
    created_at TIMESTAMP
);

-- Provider processing state
CREATE TABLE provider_state (
    id UUID PRIMARY KEY,
    blob_id UUID REFERENCES blobs(id),
    provider_id VARCHAR(255),
    status VARCHAR(50), -- pending, processing, completed, failed
    last_processed_version INTEGER,
    processed_at TIMESTAMP,
    error TEXT,
    UNIQUE(blob_id, provider_id)
);
```

## Integration with Workflow Service

The existing Temporal-based workflow service will be extended to support provider workflows:

### Provider Workflow Definition
```go
type ProviderWorkflow struct {
    ProviderID   string
    BlobID       string
    Event        string // onCreate, onEdit, onDelete
    ParentDelta  *Delta // For tracking causality
}

// Workflow activities
func ProcessBlobActivity(ctx context.Context, input ProviderWorkflow) (*Delta, error) {
    // Provider-specific processing logic
    // Returns a delta to be applied
}

func ApplyDeltaActivity(ctx context.Context, delta *Delta) error {
    // Apply delta to blob store
    // Update materialized view
    // Emit events for downstream providers
}
```

### Event Flow

1. **Blob Creation/Edit**
   ```
   User Action → Studio Service → Delta Created → Event Emitted
   ```

2. **Provider Processing**
   ```
   Event → Provider Service → Workflow Triggered → Processing → Delta Generated
   ```

3. **DAG Propagation**
   ```
   Delta Applied → Child Blobs Identified → Events Emitted → Providers Notified
   ```

## Use Case: Book Writing Assistant

### Providers
1. **OutlineGenerator**: Creates chapter outlines from concepts
2. **ContentExpander**: Expands outlines into full chapters
3. **StyleEditor**: Adjusts writing style and tone
4. **FactChecker**: Validates claims and adds citations
5. **GrammarChecker**: Fixes grammar and syntax

### Workflow Example
```
1. User writes: "Chapter 1: Introduction to AI"
2. OutlineGenerator creates derived blob with section breakdown
3. ContentExpander creates full chapter text from outline
4. StyleEditor refines the expanded content
5. GrammarChecker produces final polished version

If user edits original title:
- All derived blobs receive onEdit events
- Providers reprocess in dependency order
- DAG ensures consistency
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1-2)
- [ ] Delta engine implementation
- [ ] Blob storage with versioning
- [ ] Basic CRUD operations
- [ ] Event system integration

### Phase 2: Provider System (Week 3-4)
- [ ] Provider registry
- [ ] Workflow integration
- [ ] Event subscription system
- [ ] Processing state tracking

### Phase 3: DAG Processing (Week 5-6)
- [ ] DAG relationship management
- [ ] Cascading updates
- [ ] Dependency resolution
- [ ] Parallel processing optimization

### Phase 4: Advanced Features (Week 7-8)
- [ ] Conflict resolution
- [ ] Time-travel queries
- [ ] Batch processing
- [ ] Provider marketplace

## Configuration

```yaml
# config/studio.yaml
service:
  port: 8010
  name: memmie-studio

storage:
  blobs:
    backend: s3  # or filesystem
    bucket: memmie-studio-blobs
  metadata:
    backend: postgres

providers:
  max_concurrent: 10
  timeout: 30s
  retry_policy:
    max_attempts: 3
    backoff: exponential

events:
  backend: nats
  topics:
    blob_created: studio.blob.created
    blob_updated: studio.blob.updated
    blob_deleted: studio.blob.deleted
    delta_applied: studio.delta.applied

workflow:
  temporal_host: localhost:7233
  namespace: memmie-studio
  task_queue: studio-providers
```

## Development Setup

```bash
# Clone repository
git clone https://github.com/memmieai/memmie-studio.git
cd memmie-studio

# Install dependencies
go mod init github.com/memmieai/memmie-studio
go mod tidy

# Run migrations
migrate -path migrations -database $DATABASE_URL up

# Start service
go run cmd/server/main.go
```

## Testing Strategy

1. **Unit Tests**: Delta engine, DAG processor
2. **Integration Tests**: Provider workflows, event propagation
3. **E2E Tests**: Complete blob lifecycle with multiple providers
4. **Performance Tests**: DAG processing with deep hierarchies

## Security Considerations

- User-scoped blob isolation
- Provider sandboxing
- Delta validation before application
- Rate limiting per provider
- Audit logging for all operations

## Monitoring

- Delta application latency
- Provider processing times
- DAG depth and breadth metrics
- Event queue depth
- Storage usage per user

## Future Enhancements

1. **Provider Marketplace**: Community-contributed providers
2. **Collaborative Editing**: Multi-user blob editing with CRDTs
3. **AI Provider Templates**: Pre-built AI-powered providers
4. **Webhooks**: External system integration
5. **GraphQL API**: Flexible querying of DAG structures