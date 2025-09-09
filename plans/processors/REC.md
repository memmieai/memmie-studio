# Final Architecture Recommendation: Hybrid-Lite for MVP

## Executive Summary

After analyzing three architectural approaches for the ReYNa Studio processor system, I recommend a **"Hybrid-Lite" approach** - a simplified version of the hybrid model optimized for MVP delivery while maintaining a clear path to scale.

## Recommendation: Hybrid-Lite Architecture

### Core Principles
1. **Start Simple**: Inline storage for MVP, add external storage later
2. **Central Metadata**: All blob metadata and relationships in State Service
3. **Schema-First**: Every blob validates against a schema from day one
4. **Event-Driven**: NATS for loose coupling between processors
5. **WebSocket Updates**: Real-time updates for responsive UI

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│                          ReYNa Studio                           │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Client Layer                          │   │
│  │                 (Web, Mobile, Future)                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  Studio API (8010)                       │   │
│  │         REST + WebSocket + Auth Integration              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│  ┌───────────────────────────┼────────────────────────────┐    │
│  │                  Core Services Layer                    │    │
│  │                                                         │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │    │
│  │  │State Service │  │Schema Service│  │Processor Svc │ │    │
│  │  │    (8006)    │  │    (8011)    │  │   (8007)     │ │    │
│  │  │              │  │              │  │              │ │    │
│  │  │• User State  │  │• Schemas     │  │• Registry    │ │    │
│  │  │• All Blobs   │  │• Validation  │  │• Routing     │ │    │
│  │  │• Relations   │  │• Versions    │  │• Configs     │ │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘ │    │
│  │         │                  │                  │         │    │
│  │         └──────────────────┼──────────────────┘        │    │
│  └─────────────────────────────┼───────────────────────────┘   │
│                                │                                │
│                         ┌──────▼──────┐                        │
│                         │    NATS     │                        │
│                         │ Event Bus   │                        │
│                         └──────┬──────┘                        │
│                                │                                │
│  ┌─────────────────────────────┼───────────────────────────┐   │
│  │                    Processor Workers                     │   │
│  │                                                         │   │
│  │  ┌──────────────────────────▼────────────────────────┐ │   │
│  │  │            Text Expansion Processor               │ │   │
│  │  │  • Subscribes to: blob.created.text-input         │ │   │
│  │  │  • Validates input schema                         │ │   │
│  │  │  • Expands text via AI                           │ │   │
│  │  │  • Creates derived blob with output schema       │ │   │
│  │  │  • Emits: blob.created.expanded-text            │ │   │
│  │  └───────────────────────────────────────────────────┘ │   │
│  │                                                         │   │
│  │  ┌───────────────────────────────────────────────────┐ │   │
│  │  │            Book Compiler Processor                │ │   │
│  │  │  • Subscribes to: book.compile.requested          │ │   │
│  │  │  • Fetches all chapter blobs                      │ │   │
│  │  │  • Compiles into book format                      │ │   │
│  │  │  • Creates compiled-book blob                     │ │   │
│  │  └───────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘

Data Storage:
┌────────────────┐  ┌────────────────┐  ┌────────────────┐
│    MongoDB     │  │   PostgreSQL   │  │     Redis      │
│                │  │                │  │                │
│ • User State   │  │ • Schemas      │  │ • Cache        │
│ • Blobs        │  │ • Processors   │  │ • Sessions     │
│ • Relations    │  │ • Workflows    │  │ • WebSocket    │
└────────────────┘  └────────────────┘  └────────────────┘
```

## Service Specifications

### 1. Schema Service (Port 8011) - NEW
**Purpose**: Central source of truth for all data schemas

**Database**: PostgreSQL (for ACID compliance and JSON support)

**Core Models**:
```go
type Schema struct {
    ID          string    `db:"id"`          // UUID
    Name        string    `db:"name"`        // Human-readable name
    ProcessorID string    `db:"processor_id"` // Owner processor
    Version     string    `db:"version"`     // Semver (1.0.0)
    
    Definition  jsonschema.Schema `db:"definition"` // JSON Schema
    
    // Metadata
    Description string    `db:"description"`
    Examples    []json.RawMessage `db:"examples"`
    
    Active      bool      `db:"active"`
    CreatedAt   time.Time `db:"created_at"`
}
```

**Key APIs**:
```yaml
POST   /schemas                 # Register new schema
GET    /schemas/:id             # Get schema definition
GET    /schemas/:id/validate    # Validate data against schema
GET    /processors/:id/schemas  # List schemas for processor
```

### 2. State Service (Port 8006) - MODIFIED
**Purpose**: Store all blobs and user state

**Changes from Current**:
- Replace generic `UserState.State map[string]interface{}` with structured blob storage
- Add blob collection with schema validation
- Maintain relationships between blobs

**Updated Models**:
```go
type Blob struct {
    ID          string                 `bson:"_id"`
    UserID      string                 `bson:"user_id"`
    ProcessorID string                 `bson:"processor_id"`
    SchemaID    string                 `bson:"schema_id"`
    
    // Dynamic data matching schema
    Data        interface{}            `bson:"data"`
    
    // Relationships
    ParentID    *string                `bson:"parent_id,omitempty"`
    DerivedIDs  []string               `bson:"derived_ids"`
    
    // Organization
    BookID      *string                `bson:"book_id,omitempty"`
    ConversationID *string             `bson:"conversation_id,omitempty"`
    
    CreatedAt   time.Time              `bson:"created_at"`
    UpdatedAt   time.Time              `bson:"updated_at"`
}

type UserState struct {
    UserID      string                 `bson:"user_id"`
    
    // Organized content
    Books       []BookProject          `bson:"books"`
    Conversations []Conversation       `bson:"conversations"`
    
    // Statistics
    BlobCount   int                    `bson:"blob_count"`
    TotalSize   int64                  `bson:"total_size_bytes"`
    
    UpdatedAt   time.Time              `bson:"updated_at"`
}
```

### 3. Processor Service (Port 8007) - RENAMED
**Purpose**: Registry and orchestration of all processors

**Key Responsibilities**:
- Register processors with their schemas
- Route events to appropriate processors
- Track processor instances per user
- Monitor processor health

**Core Models**:
```go
type Processor struct {
    ID              string    `db:"id"`
    Name            string    `db:"name"`
    
    // Schema requirements
    InputSchemaID   string    `db:"input_schema_id"`
    OutputSchemaID  string    `db:"output_schema_id"`
    
    // Event subscriptions
    SubscribeEvents []string  `db:"subscribe_events"`
    EmitEvents      []string  `db:"emit_events"`
    
    // Processing config
    WorkflowID      string    `db:"workflow_id"`
    MaxConcurrency  int       `db:"max_concurrency"`
    
    Active          bool      `db:"active"`
}

type ProcessorInstance struct {
    ID              string    `db:"id"`
    ProcessorID     string    `db:"processor_id"`
    UserID          string    `db:"user_id"`
    
    // User-specific settings
    Config          map[string]interface{} `db:"config"`
    
    Active          bool      `db:"active"`
}
```

### 4. Studio API (Port 8010) - ENHANCED
**Purpose**: Gateway, WebSocket handler, and frontend server

**New Capabilities**:
- WebSocket for real-time updates
- Event filtering per user
- Smart blob fetching
- Schema validation proxy

## Event Flow Architecture

### Event Naming Convention
```
<entity>.<action>.<schema>

Examples:
- blob.created.text-input-v1
- blob.updated.expanded-text-v1
- blob.deleted.*
- processor.completed.text-expansion
- book.compile.requested
```

### Complete Flow: Text Expansion

```yaml
1. User Types in Book Writer:
   Client → Studio API (WebSocket)
   {
     "action": "create_blob",
     "data": {
       "content": "The ship sailed.",
       "book_id": "my-novel",
       "chapter": 1
     }
   }

2. Studio API Creates Blob:
   Studio → State Service
   - Validate against schema "text-input-v1"
   - Create blob with parent references
   - Return blob ID

3. State Service Emits Event:
   State → NATS
   Event: "blob.created.text-input-v1"
   {
     "blob_id": "blob_123",
     "user_id": "user_456",
     "processor_id": "user-input"
   }

4. Text Expansion Processor Receives:
   NATS → Text Expansion Worker
   - Subscribe pattern: "blob.created.text-input-v1"
   - Fetch blob from State Service
   - Process through AI
   - Create expanded blob

5. Processor Creates Derived Blob:
   Worker → State Service
   {
     "processor_id": "text-expansion",
     "schema_id": "expanded-text-v1",
     "parent_id": "blob_123",
     "data": {
       "original": "The ship sailed.",
       "expanded": "The mighty vessel...",
       "ratio": 3.5
     }
   }

6. State Service Updates & Emits:
   - Create new blob
   - Update parent's derived_ids
   - Emit: "blob.created.expanded-text-v1"

7. WebSocket Delivers Update:
   NATS → Studio API → WebSocket → Client
   {
     "type": "blob.derived",
     "parent_id": "blob_123",
     "derived_id": "blob_789",
     "processor": "text-expansion"
   }

8. Client Updates UI:
   - Fetch new blob if needed
   - Update split view
   - Show expansion in right pane
```

## MVP Implementation Plan

### Phase 1: Foundation (Week 1)
1. **Schema Service**
   - Basic CRUD for schemas
   - JSON Schema validation
   - Version management

2. **State Service Refactor**
   - Add Blob collection
   - Implement relationships
   - Schema validation integration

3. **NATS Setup**
   - Event streaming configuration
   - Dead letter queues
   - Event replay capability

### Phase 2: Core Processors (Week 2)
1. **Text Expansion Processor**
   - Input: text-input-v1
   - Output: expanded-text-v1
   - GPT-4 integration
   - 3x-5x expansion ratio

2. **Processor Service**
   - Registration system
   - Event routing
   - Health monitoring

3. **WebSocket Integration**
   - Studio API WebSocket server
   - User-specific event filtering
   - Connection management

### Phase 3: User Experience (Week 3)
1. **Book Writer Interface**
   - Split-pane editor
   - Real-time expansion
   - Chapter organization

2. **Pitch Creator Interface**
   - Section-based input
   - Structured output
   - Export capabilities

3. **Testing & Optimization**
   - Load testing with 100 users
   - WebSocket performance
   - Query optimization

## Schema Examples

### Text Input Schema (v1)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "text-input-v1",
  "type": "object",
  "required": ["content"],
  "properties": {
    "content": {
      "type": "string",
      "minLength": 10,
      "maxLength": 50000
    },
    "metadata": {
      "type": "object",
      "properties": {
        "book_id": {"type": "string"},
        "chapter": {"type": "integer"},
        "style": {
          "type": "string",
          "enum": ["formal", "casual", "creative"]
        }
      }
    }
  }
}
```

### Expanded Text Schema (v1)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "expanded-text-v1",
  "type": "object",
  "required": ["original", "expanded", "expansion_ratio"],
  "properties": {
    "original": {
      "type": "string"
    },
    "expanded": {
      "type": "string"
    },
    "expansion_ratio": {
      "type": "number",
      "minimum": 1.0
    },
    "metrics": {
      "type": "object",
      "properties": {
        "readability_score": {"type": "number"},
        "complexity": {"type": "number"},
        "tone": {"type": "string"}
      }
    }
  }
}
```

## Why This Architecture?

### Advantages for MVP
1. **Simplicity**: Single blob storage, clear service boundaries
2. **Fast Development**: Can build incrementally
3. **Real-time Updates**: WebSocket from day one
4. **Type Safety**: Schema validation throughout
5. **Scalable Foundation**: Easy to add processors

### Trade-offs Accepted
1. **All blobs in one DB**: Acceptable for 100 users
2. **No external storage**: Fine for text content
3. **Simple caching**: Redis later if needed
4. **Basic auth**: Leverages existing auth service

### Future Growth Path
1. **Phase 4**: Add external storage for large blobs
2. **Phase 5**: Processor-specific databases
3. **Phase 6**: CDN for media content
4. **Phase 7**: Distributed processing with Temporal

## Success Metrics

### Technical Metrics
- Blob creation to UI update: <500ms
- WebSocket message delivery: <100ms
- Schema validation: <50ms
- Text expansion: <3s for 1000 words

### User Metrics
- 100 beta users active
- 1000+ blobs created daily
- 90% expansion satisfaction
- <1% error rate

## Risk Mitigation

### Technical Risks
1. **MongoDB Performance**
   - Mitigation: Proper indexing, 1MB blob limit for MVP
   
2. **WebSocket Scaling**
   - Mitigation: Sticky sessions, Redis pub/sub if needed

3. **Schema Evolution**
   - Mitigation: Versioning from day one, backward compatibility

### Operational Risks
1. **Processor Failures**
   - Mitigation: Circuit breakers, retry logic, dead letter queues

2. **Data Loss**
   - Mitigation: MongoDB replication, regular backups

## Conclusion

This Hybrid-Lite architecture provides the optimal balance for ReYNa Studio's MVP:
- **Simple enough** to build in 3 weeks
- **Robust enough** for 100 beta users
- **Flexible enough** to evolve with needs
- **Real-time** for responsive user experience

The schema-first approach ensures data consistency, while the event-driven architecture allows processors to evolve independently. Most importantly, it delivers the core book writing and pitch creation features users need while maintaining a clear path to scale.