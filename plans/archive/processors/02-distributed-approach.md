# Processor Architecture Plan 2: Distributed Blob Storage

## Overview

Each processor maintains its own database for blobs it creates. The State Service acts as a central index and stores only user-created blobs. This provides strong isolation and allows processors to optimize their storage for their specific needs.

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
│            WebSocket + REST Gateway + Aggregator             │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│State Service │    │Schema Service│    │Processor Svc │
│    (8006)    │    │    (8011)    │    │   (8007)     │
│              │    │              │    │              │
│ User Blobs   │    │ All Schemas  │    │ Registry     │
│ Blob Index   │    │ Validation   │    │ Router       │
└──────────────┘    └──────────────┘    └──────────────┘
        │                                       │
        └──────────────┬────────────────────────┘
                       ▼
                ┌─────────────┐
                │    NATS     │
                │Event Stream │
                └─────────────┘
                       ▲
        ┌──────────────┼──────────────────┐
        ▼              ▼                  ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│Text Expansion│ │Book Compiler │ │Pitch Builder │
│   Service    │ │   Service    │ │   Service    │
│              │ │              │ │              │
│  Own MongoDB │ │  Own MongoDB │ │  PostgreSQL  │
└──────────────┘ └──────────────┘ └──────────────┘
```

## Data Models

### Blob Index (State Service - MongoDB)
```go
// Central index of all blobs across the system
type BlobIndex struct {
    ID              string              `bson:"_id"`
    UserID          string              `bson:"user_id"`
    
    // Location information
    ProcessorID     string              `bson:"processor_id"`    // Which processor owns this
    StorageLocation string              `bson:"storage_location"` // Connection string/identifier
    RemoteBlobID    string              `bson:"remote_blob_id"`   // ID in processor's database
    
    // Schema reference
    SchemaID        string              `bson:"schema_id"`
    SchemaVersion   string              `bson:"schema_version"`
    
    // Relationships (IDs only)
    ParentID        *string             `bson:"parent_id,omitempty"`
    DerivedIDs      []string            `bson:"derived_ids"`
    
    // Cached metadata for queries
    Type            string              `bson:"type"`       // chapter, message, pitch, etc.
    Title           string              `bson:"title"`      // For display
    Preview         string              `bson:"preview"`    // First 200 chars
    Size            int64               `bson:"size_bytes"`
    
    CreatedAt       time.Time           `bson:"created_at"`
    UpdatedAt       time.Time           `bson:"updated_at"`
}

// User's original content blobs (not processor-derived)
type UserBlob struct {
    ID              string              `bson:"_id"`
    UserID          string              `bson:"user_id"`
    SchemaID        string              `bson:"schema_id"`
    
    // Actual data stored here for user-created content
    Data            interface{}         `bson:"data"`
    
    // Organizational structures
    ConversationID  *string             `bson:"conversation_id,omitempty"`
    BookID          *string             `bson:"book_id,omitempty"`
    
    ProcessingState string              `bson:"processing_state"`
    CreatedAt       time.Time           `bson:"created_at"`
}

type UserState struct {
    UserID          string              `bson:"user_id"`
    
    // Statistics across all processors
    TotalBlobs      int                 `bson:"total_blobs"`
    BlobsByProcessor map[string]int     `bson:"blobs_by_processor"`
    
    // Quick access structures
    Books           []BookStructure     `bson:"books"`
    Conversations   []ConversationMeta  `bson:"conversations"`
    
    UpdatedAt       time.Time           `bson:"updated_at"`
}
```

### Text Expansion Service Database (MongoDB)
```go
// Processor-specific blob storage
type ExpandedTextBlob struct {
    ID              string              `bson:"_id"`
    UserID          string              `bson:"user_id"`
    ParentBlobID    string              `bson:"parent_blob_id"`   // Reference to source
    
    // Schema-compliant data
    Data            ExpandedTextData    `bson:"data"`
    
    // Processing metadata
    ProcessingTime  int                 `bson:"processing_time_ms"`
    ModelUsed       string              `bson:"model_used"`
    TokensUsed      int                 `bson:"tokens_used"`
    
    CreatedAt       time.Time           `bson:"created_at"`
}

type ExpandedTextData struct {
    Original        string              `bson:"original"`
    Expanded        string              `bson:"expanded"`
    ExpansionRatio  float64             `bson:"expansion_ratio"`
    StyleAnalysis   StyleMetrics        `bson:"style_analysis"`
    Sections        []TextSection       `bson:"sections"`
}

type StyleMetrics struct {
    Tone            string              `bson:"tone"`
    Complexity      float64             `bson:"complexity"`
    ReadabilityScore float64            `bson:"readability_score"`
    VocabularyLevel string              `bson:"vocabulary_level"`
}
```

### Book Compiler Service Database (PostgreSQL)
```sql
-- Compiled book storage with relational structure
CREATE TABLE compiled_books (
    id              UUID PRIMARY KEY,
    user_id         VARCHAR(255) NOT NULL,
    book_id         VARCHAR(255) NOT NULL,
    version         INTEGER NOT NULL,
    
    -- Compiled content
    full_text       TEXT,
    word_count      INTEGER,
    chapter_count   INTEGER,
    
    -- References to source blobs
    source_blob_ids JSONB,  -- Array of blob IDs
    
    -- Metadata
    compile_config  JSONB,
    created_at      TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(book_id, version)
);

CREATE TABLE book_chapters (
    id              UUID PRIMARY KEY,
    book_id         UUID REFERENCES compiled_books(id),
    chapter_num     INTEGER NOT NULL,
    title           VARCHAR(500),
    
    -- Content references
    draft_blob_id   VARCHAR(255),
    expanded_blob_id VARCHAR(255),
    
    -- Compiled chapter
    compiled_text   TEXT,
    word_count      INTEGER,
    
    UNIQUE(book_id, chapter_num)
);
```

### Processor Registry (Processor Service - PostgreSQL)
```go
type ProcessorRegistration struct {
    ID                  string          `db:"id"`
    Name                string          `db:"name"`
    Description         string          `db:"description"`
    
    // Storage configuration
    StorageType         string          `db:"storage_type"`      // mongodb, postgresql, redis
    ConnectionString    string          `db:"connection_string"` // Encrypted
    DatabaseName        string          `db:"database_name"`
    
    // Schema requirements
    InputSchemaIDs      []string        `db:"input_schema_ids"`  // Can process multiple
    OutputSchemaID      string          `db:"output_schema_id"`
    
    // API endpoints for direct access
    BaseURL             string          `db:"base_url"`          // https://text-expansion:8012
    HealthEndpoint      string          `db:"health_endpoint"`
    QueryEndpoint       string          `db:"query_endpoint"`
    
    // Processing configuration
    EventSubscriptions  []string        `db:"event_subscriptions"`
    MaxConcurrency      int             `db:"max_concurrency"`
    TimeoutSeconds      int             `db:"timeout_seconds"`
    
    Active              bool            `db:"active"`
    CreatedAt           time.Time       `db:"created_at"`
}
```

## Event Flow

### 1. User Creates Content
```yaml
Client → Studio API:
  POST /api/v1/blobs
  {
    "type": "chapter_draft",
    "schema_id": "text-input-v1",
    "data": {
      "content": "The storm approached.",
      "metadata": {
        "book_id": "my-novel",
        "chapter_num": 1
      }
    }
  }

Studio API → State Service:
  - Store in UserBlob collection
  - Create BlobIndex entry
  - Set processing_state: "pending"

State Service → NATS:
  Event: blob.created
  {
    "blob_id": "blob_123",
    "user_id": "user_456",
    "schema_id": "text-input-v1",
    "location": "state-service"
  }
```

### 2. Processor Handles Event
```yaml
Text Expansion Service:
  - Receive NATS event
  - Fetch blob from State Service
  - Validate against input schema
  - Process and expand text
  - Store in own database
  - Register with State Service index

Service → Own Database:
  {
    "user_id": "user_456",
    "parent_blob_id": "blob_123",
    "data": {
      "original": "The storm approached.",
      "expanded": "Dark clouds gathered on the horizon...",
      "expansion_ratio": 4.2
    }
  }

Service → State Service:
  POST /api/v1/blob-index
  {
    "user_id": "user_456",
    "processor_id": "text-expansion",
    "storage_location": "text-expansion-db",
    "remote_blob_id": "expanded_789",
    "parent_id": "blob_123",
    "schema_id": "expanded-text-v1",
    "preview": "Dark clouds gathered..."
  }

State Service:
  - Create index entry
  - Update parent's derived_ids
  - Emit event
```

### 3. Client Queries Blobs
```yaml
Client → Studio API:
  GET /api/v1/users/456/blobs?book_id=my-novel

Studio API:
  1. Query State Service for index
  2. Group by processor_id
  3. Parallel fetch from each processor:
     - GET text-expansion:8012/blobs/expanded_789
     - GET book-compiler:8013/books/my-novel
  4. Aggregate results
  5. Return unified response

Response:
  {
    "blobs": [
      {
        "id": "blob_123",
        "type": "user_draft",
        "data": {...},
        "derived": [
          {
            "id": "expanded_789",
            "processor": "text-expansion",
            "preview": "Dark clouds..."
          }
        ]
      }
    ]
  }
```

## Text Expansion Processor Implementation

### Service Structure
```
text-expansion-service/
├── cmd/server/main.go
├── internal/
│   ├── storage/          # MongoDB client
│   ├── processor/        # Core expansion logic
│   ├── schema/           # Schema validation
│   ├── api/              # REST endpoints
│   └── events/           # NATS subscriber
└── schemas/
    ├── input-v1.json
    └── output-v1.json
```

### Processing Pipeline
```go
type TextExpansionProcessor struct {
    storage     *mongo.Database
    schemaClient *SchemaClient
    stateClient *StateClient
    aiClient    *AIClient
}

func (p *TextExpansionProcessor) ProcessBlob(event BlobCreatedEvent) error {
    // 1. Fetch source blob
    sourceBlob, err := p.stateClient.GetBlob(event.BlobID)
    
    // 2. Validate schema
    if err := p.schemaClient.Validate(sourceBlob.Data, p.InputSchemaID); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    // 3. Extract text content
    text := sourceBlob.Data["content"].(string)
    
    // 4. Analyze style
    style := p.aiClient.AnalyzeStyle(text)
    
    // 5. Expand text
    expanded := p.aiClient.ExpandText(text, style)
    
    // 6. Create output blob
    outputBlob := ExpandedTextBlob{
        ID:           uuid.New().String(),
        UserID:       event.UserID,
        ParentBlobID: event.BlobID,
        Data: ExpandedTextData{
            Original: text,
            Expanded: expanded,
            ExpansionRatio: float64(len(expanded)) / float64(len(text)),
            StyleAnalysis: style,
        },
    }
    
    // 7. Store in own database
    if err := p.storage.Collection("expanded_texts").InsertOne(outputBlob); err != nil {
        return err
    }
    
    // 8. Register with State Service
    return p.stateClient.RegisterBlobIndex(BlobIndex{
        UserID:          event.UserID,
        ProcessorID:     "text-expansion",
        StorageLocation: "text-expansion-db",
        RemoteBlobID:    outputBlob.ID,
        ParentID:        &event.BlobID,
        SchemaID:        p.OutputSchemaID,
    })
}
```

### Direct Query API
```go
// Processors expose query endpoints for their data
func (s *TextExpansionService) QueryEndpoints() {
    // Get specific blob
    router.GET("/blobs/:id", func(c *gin.Context) {
        blob := s.storage.FindOne(bson.M{"_id": c.Param("id")})
        c.JSON(200, blob)
    })
    
    // Query by user
    router.GET("/users/:user_id/blobs", func(c *gin.Context) {
        blobs := s.storage.Find(bson.M{
            "user_id": c.Param("user_id"),
            "created_at": bson.M{"$gte": c.Query("since")},
        })
        c.JSON(200, blobs)
    })
    
    // Batch fetch
    router.POST("/blobs/batch", func(c *gin.Context) {
        var ids []string
        c.BindJSON(&ids)
        blobs := s.storage.Find(bson.M{"_id": bson.M{"$in": ids}})
        c.JSON(200, blobs)
    })
}
```

## Book Writing Flow

### Complete Workflow
```yaml
1. User creates book project:
   State Service:
     - Create BookStructure in UserState
     - Initialize chapter list

2. User writes chapter draft:
   State Service:
     - Store UserBlob with chapter content
     - Emit blob.created event

3. Text Expansion processes:
   Text Expansion Service:
     - Fetch draft from State Service
     - Expand text with AI
     - Store in own database
     - Register index with State Service

4. User triggers book compilation:
   Book Compiler Service:
     - Query State Service for all chapter indexes
     - Fetch drafts from State Service
     - Fetch expanded texts from Text Expansion Service
     - Compile into book format
     - Store compiled book in PostgreSQL
     - Register index with State Service

5. Client displays book:
   Studio API:
     - Query State Service for book structure
     - Parallel fetch:
       - Drafts from State Service
       - Expanded from Text Expansion
       - Compiled from Book Compiler
     - Return aggregated view
```

## Advantages

1. **Isolation**: Each processor manages its own data
2. **Optimization**: Processors can use optimal storage (MongoDB, PostgreSQL, Redis)
3. **Scalability**: Can scale processors independently
4. **Fault Tolerance**: Processor failure doesn't affect others
5. **Schema Evolution**: Processors can evolve schemas independently

## Disadvantages

1. **Complexity**: Multiple databases to manage
2. **Query Overhead**: Need to aggregate from multiple sources
3. **Consistency**: Distributed transactions are complex
4. **Network Latency**: Cross-service queries add latency
5. **Operational Cost**: More infrastructure to maintain

## Implementation Phases

1. **Phase 1: Infrastructure**
   - Schema Service setup
   - State Service with index
   - Processor Service registry
   - NATS event streaming

2. **Phase 2: First Processor**
   - Text Expansion Service
   - Own MongoDB instance
   - API endpoints
   - Event handlers

3. **Phase 3: Aggregation**
   - Studio API aggregator
   - Parallel query optimization
   - Response caching
   - WebSocket updates

4. **Phase 4: Additional Processors**
   - Book Compiler (PostgreSQL)
   - Pitch Builder (PostgreSQL)
   - Image Processor (S3 + metadata)