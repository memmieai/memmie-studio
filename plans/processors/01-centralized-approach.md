# Processor Architecture Plan 1: Centralized Blob Storage

## Overview

All blobs (user-created and processor-derived) are stored in a single MongoDB database within the State Service. Processors register with the Schema Service and subscribe to NATS events to process matching blobs.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         Clients                              │
│                    (Web, Mobile, AR)                         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Studio API (8010)                         │
│                 WebSocket + REST Gateway                     │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│State Service │    │Schema Service│    │Processor Svc │
│    (8006)    │    │    (8011)    │    │   (8007)     │
│              │◄───│              │◄───│              │
│ All Blobs    │    │ All Schemas  │    │ Registry     │
│ User State   │    │ Validation   │    │ Orchestrator │
└──────────────┘    └──────────────┘    └──────────────┘
        │                                       │
        └──────────────┬────────────────────────┘
                       ▼
                ┌─────────────┐
                │    NATS     │
                │Event Stream │
                └─────────────┘
                       ▲
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│Text Expansion│ │Book Compiler │ │Pitch Builder │
│   Worker     │ │   Worker     │ │   Worker     │
└──────────────┘ └──────────────┘ └──────────────┘
```

## Data Models

### Blob (State Service - MongoDB)
```go
type Blob struct {
    ID           string                 `bson:"_id"`
    UserID       string                 `bson:"user_id"`
    ProcessorID  string                 `bson:"processor_id"`  // Which processor created this
    SchemaID     string                 `bson:"schema_id"`     // References Schema Service
    
    // Dynamic content matching schema
    Data         interface{}            `bson:"data"`          // Validated against schema
    
    // Relationships
    ParentID     *string                `bson:"parent_id,omitempty"`
    DerivedIDs   []string               `bson:"derived_ids"`   // Blobs derived from this
    ConversationID *string              `bson:"conversation_id,omitempty"`
    
    // Metadata
    Tags         []string               `bson:"tags"`
    Version      int                    `bson:"version"`
    ProcessingState string              `bson:"processing_state"` // pending, processing, completed
    
    CreatedAt    time.Time              `bson:"created_at"`
    UpdatedAt    time.Time              `bson:"updated_at"`
}

type UserState struct {
    UserID       string                 `bson:"user_id"`
    BlobCount    int                    `bson:"blob_count"`
    TotalSize    int64                  `bson:"total_size_bytes"`
    
    // Quick access indexes
    Conversations []ConversationIndex   `bson:"conversations"`
    Books        []BookIndex            `bson:"books"`
    
    UpdatedAt    time.Time              `bson:"updated_at"`
}

type ConversationIndex struct {
    ID           string                 `bson:"id"`
    Title        string                 `bson:"title"`
    LastBlobID   string                 `bson:"last_blob_id"`
    BlobCount    int                    `bson:"blob_count"`
    UpdatedAt    time.Time              `bson:"updated_at"`
}
```

### Schema (Schema Service - PostgreSQL)
```go
type Schema struct {
    ID           string                 `db:"id"`           // UUID
    ProcessorID  string                 `db:"processor_id"` // Owner processor
    Name         string                 `db:"name"`
    Version      string                 `db:"version"`      // semver
    Type         string                 `db:"type"`         // input, output, intermediate
    
    // JSON Schema definition
    Definition   map[string]interface{} `db:"definition"`
    
    // Examples for validation testing
    Examples     []interface{}          `db:"examples"`
    
    Active       bool                   `db:"active"`
    CreatedAt    time.Time              `db:"created_at"`
}
```

### Processor Registration (Processor Service - PostgreSQL)
```go
type Processor struct {
    ID              string             `db:"id"`           // e.g., "text-expansion"
    Name            string             `db:"name"`
    Description     string             `db:"description"`
    
    // Schema requirements
    InputSchemaID   string             `db:"input_schema_id"`
    OutputSchemaID  string             `db:"output_schema_id"`
    
    // Processing configuration
    WorkflowID      string             `db:"workflow_id"`
    Priority        int                `db:"priority"`
    MaxConcurrency  int                `db:"max_concurrency"`
    
    // Event subscriptions
    EventPatterns   []string           `db:"event_patterns"` // NATS subjects to subscribe
    
    Active          bool               `db:"active"`
    CreatedAt       time.Time          `db:"created_at"`
}

type ProcessorInstance struct {
    ID              string             `db:"id"`
    ProcessorID     string             `db:"processor_id"`
    UserID          string             `db:"user_id"`
    
    // User-specific configuration
    Config          map[string]interface{} `db:"config"`
    
    Active          bool               `db:"active"`
    CreatedAt       time.Time          `db:"created_at"`
}
```

## Event Flow

### 1. User Creates Content
```yaml
Client → Studio API:
  POST /api/v1/blobs
  {
    "processor_id": "user-input",
    "schema_id": "text-input-v1",
    "data": {
      "content": "The ship sailed into the storm.",
      "metadata": {
        "type": "chapter",
        "book_id": "my-novel"
      }
    }
  }

Studio API → State Service:
  - Create blob with schema validation
  - Set processing_state: "pending"

State Service → NATS:
  Event: blob.created
  {
    "blob_id": "blob_123",
    "user_id": "user_456",
    "schema_id": "text-input-v1",
    "processor_id": "user-input"
  }
```

### 2. Processor Receives Event
```yaml
Text Expansion Worker:
  - Subscribe: blob.created.text-input-v1
  - Receive event
  - Fetch blob from State Service
  - Validate against input schema
  - Process through workflow
  - Create derived blob

Worker → State Service:
  POST /api/v1/blobs
  {
    "processor_id": "text-expansion",
    "schema_id": "expanded-text-v1",
    "parent_id": "blob_123",
    "data": {
      "original": "The ship sailed into the storm.",
      "expanded": "The mighty vessel, its weathered hull creaking...",
      "expansion_ratio": 3.5,
      "metadata": {
        "model": "gpt-4",
        "temperature": 0.7
      }
    }
  }

State Service:
  - Create new blob
  - Update parent blob's derived_ids
  - Emit blob.created event
```

### 3. Client Receives Update
```yaml
State Service → NATS:
  Event: blob.derived
  {
    "parent_id": "blob_123",
    "derived_id": "blob_789",
    "processor_id": "text-expansion"
  }

NATS → Studio API (WebSocket):
  - Receive event
  - Filter for user's subscriptions
  - Send to client via WebSocket

WebSocket → Client:
  {
    "type": "blob.derived",
    "data": {
      "parent_id": "blob_123",
      "derived_id": "blob_789",
      "preview": "The mighty vessel..."
    }
  }
```

## Text Expansion Processor Example

### Registration
```yaml
processor:
  id: text-expansion
  name: Text Expansion Processor
  input_schema_id: schema-text-input-v1
  output_schema_id: schema-expanded-text-v1
  event_patterns:
    - blob.created.text-input-v1
    - blob.updated.text-input-v1
  workflow_id: wf-text-expansion
```

### Input Schema
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["content"],
  "properties": {
    "content": {
      "type": "string",
      "minLength": 10,
      "maxLength": 10000
    },
    "style": {
      "type": "string",
      "enum": ["formal", "casual", "creative", "technical"]
    },
    "metadata": {
      "type": "object",
      "properties": {
        "type": {"type": "string"},
        "book_id": {"type": "string"},
        "chapter_num": {"type": "integer"}
      }
    }
  }
}
```

### Output Schema
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["original", "expanded", "expansion_ratio"],
  "properties": {
    "original": {"type": "string"},
    "expanded": {"type": "string"},
    "expansion_ratio": {"type": "number"},
    "style_analysis": {
      "type": "object",
      "properties": {
        "tone": {"type": "string"},
        "complexity": {"type": "number"},
        "readability_score": {"type": "number"}
      }
    },
    "metadata": {
      "type": "object",
      "properties": {
        "model": {"type": "string"},
        "temperature": {"type": "number"},
        "processing_time_ms": {"type": "integer"}
      }
    }
  }
}
```

### Workflow Definition
```yaml
id: wf-text-expansion
name: Text Expansion Workflow
steps:
  - id: validate-input
    type: schema-validation
    schema_id: schema-text-input-v1
    
  - id: analyze-style
    type: api-call
    service: ai-service
    endpoint: /analyze/style
    input_map:
      text: "$.data.content"
    output_map:
      style: "$.style_analysis"
      
  - id: expand-text
    type: api-call
    service: ai-service
    endpoint: /generate/expand
    input_map:
      original: "$.data.content"
      style: "$.data.style || $.style_analysis.detected_style"
      context: "$.data.metadata"
    output_map:
      expanded: "$.expanded_text"
      
  - id: calculate-metrics
    type: computation
    operations:
      - expansion_ratio: "len($.expanded_text) / len($.data.content)"
      - word_count_original: "len($.data.content.split())"
      - word_count_expanded: "len($.expanded_text.split())"
      
  - id: create-output-blob
    type: state-update
    operation: create-blob
    schema_id: schema-expanded-text-v1
    data_map:
      original: "$.data.content"
      expanded: "$.expanded_text"
      expansion_ratio: "$.expansion_ratio"
      style_analysis: "$.style_analysis"
      metadata:
        model: "gpt-4"
        temperature: 0.7
        processing_time_ms: "$.execution_time"
```

## Book Writing Use Case

### User Flow
1. User creates a new book project
2. System creates a book container (conversation-like structure)
3. User writes chapter drafts as blobs
4. Each draft triggers text-expansion processor
5. Expanded versions are linked as derived blobs
6. User can view original and expanded side-by-side
7. Book compiler processor can combine chapters

### Data Structure
```yaml
Book Container:
  - ID: book_my-novel
  - Title: "My Novel"
  - Chapters: [
      {
        draft_blob_id: blob_001,
        expanded_blob_id: blob_002,
        chapter_num: 1,
        title: "The Beginning"
      },
      {
        draft_blob_id: blob_003,
        expanded_blob_id: blob_004,
        chapter_num: 2,
        title: "The Journey"
      }
    ]
```

## Advantages

1. **Simplicity**: Single source of truth for all blobs
2. **Query Performance**: Can easily query across all user data
3. **Consistency**: ACID transactions within MongoDB
4. **Flexibility**: Easy to add new relationships and indexes
5. **Debugging**: All data in one place for troubleshooting

## Disadvantages

1. **Scalability**: Single database could become bottleneck
2. **Isolation**: Processor failures could affect shared database
3. **Schema Evolution**: Changes affect entire database
4. **Performance**: Large blob collections might slow queries
5. **Security**: All processors access same database

## Implementation Steps

1. **Phase 1: Core Services**
   - Schema Service with PostgreSQL
   - Update State Service for blob storage
   - Processor Service registry

2. **Phase 2: Event System**
   - NATS event streaming setup
   - Event routing based on schemas
   - Dead letter queue for failures

3. **Phase 3: Processors**
   - Text expansion processor
   - Book compiler processor
   - Pitch builder processor

4. **Phase 4: WebSocket**
   - Real-time event delivery
   - Client subscriptions
   - Optimistic updates