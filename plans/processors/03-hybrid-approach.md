# Processor Architecture Plan 3: Hybrid Storage Model

## Overview

This hybrid approach stores all blob metadata and relationships in the State Service for fast querying, while large blob content is stored in processor-optimized storage. Small blobs (<1MB) are stored directly in State Service for low latency, while large blobs use distributed storage with content URLs.

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
│              WebSocket + REST + Smart Router                 │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│State Service │    │Schema Service│    │Processor Svc │
│    (8006)    │    │    (8011)    │    │   (8007)     │
│              │    │              │    │              │
│ Metadata +   │    │ All Schemas  │    │ Registry     │
│ Small Blobs  │    │ Validation   │    │ Orchestrator │
│ Relationships│    │ Versioning   │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
        │                   │                   │
        │            ┌──────┴──────┐           │
        │            │  Schema DB  │           │
        │            │ PostgreSQL  │           │
        │            └─────────────┘           │
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
│Text Expansion│ │Book Compiler │ │Media Process │
│   Worker     │ │   Worker     │ │   Worker     │
│              │ │              │ │              │
│ Content Store│ │ Content Store│ │  S3 Storage  │
│   (Redis)    │ │  (MongoDB)   │ │              │
└──────────────┘ └──────────────┘ └──────────────┘
```

## Data Models

### Unified Blob Model (State Service - MongoDB)
```go
type Blob struct {
    ID              string                 `bson:"_id"`
    UserID          string                 `bson:"user_id"`
    ProcessorID     string                 `bson:"processor_id"`
    SchemaID        string                 `bson:"schema_id"`
    SchemaVersion   string                 `bson:"schema_version"`
    
    // Content storage strategy
    StorageType     string                 `bson:"storage_type"`     // inline, url, reference
    
    // For small blobs (<1MB): inline storage
    InlineData      interface{}            `bson:"inline_data,omitempty"`
    
    // For large blobs: external storage
    ContentURL      *string                `bson:"content_url,omitempty"`
    ContentSize     int64                  `bson:"content_size_bytes"`
    ContentHash     string                 `bson:"content_hash"`     // SHA256 for integrity
    
    // Relationships (always stored)
    ParentID        *string                `bson:"parent_id,omitempty"`
    DerivedIDs      []string               `bson:"derived_ids"`
    ConversationID  *string                `bson:"conversation_id,omitempty"`
    BookID          *string                `bson:"book_id,omitempty"`
    
    // Searchable metadata (always stored)
    Title           string                 `bson:"title"`
    Preview         string                 `bson:"preview"`         // First 500 chars
    Tags            []string               `bson:"tags"`
    SearchVector    []float32              `bson:"search_vector"`   // For similarity search
    
    // Processing metadata
    ProcessingState string                 `bson:"processing_state"`
    ProcessingMeta  map[string]interface{} `bson:"processing_meta"`
    
    // Timestamps
    CreatedAt       time.Time              `bson:"created_at"`
    UpdatedAt       time.Time              `bson:"updated_at"`
    AccessedAt      time.Time              `bson:"accessed_at"`     // For cache management
}

type UserState struct {
    UserID          string                 `bson:"user_id"`
    
    // Storage quotas and usage
    StorageUsed     int64                  `bson:"storage_used_bytes"`
    StorageQuota    int64                  `bson:"storage_quota_bytes"`
    BlobCount       int                    `bson:"blob_count"`
    
    // Organized structures
    Books           []BookProject          `bson:"books"`
    Conversations   []Conversation         `bson:"conversations"`
    Collections     []BlobCollection       `bson:"collections"`      // User-defined groups
    
    // Processor configurations
    ProcessorConfigs map[string]interface{} `bson:"processor_configs"`
    
    UpdatedAt       time.Time              `bson:"updated_at"`
}

type BookProject struct {
    ID              string                 `bson:"id"`
    Title           string                 `bson:"title"`
    Description     string                 `bson:"description"`
    
    // Chapter organization
    Chapters        []ChapterRef           `bson:"chapters"`
    
    // Processing pipelines
    DefaultPipeline []string               `bson:"default_pipeline"` // e.g., ["text-expansion", "grammar-check"]
    
    CreatedAt       time.Time              `bson:"created_at"`
    UpdatedAt       time.Time              `bson:"updated_at"`
}

type ChapterRef struct {
    ChapterNum      int                    `bson:"chapter_num"`
    Title           string                 `bson:"title"`
    DraftBlobID     string                 `bson:"draft_blob_id"`
    ProcessedBlobIDs map[string]string     `bson:"processed_blob_ids"` // processor_id -> blob_id
}
```

### Schema Model (Schema Service - PostgreSQL)
```sql
-- Schema definitions with versioning
CREATE TABLE schemas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_name     VARCHAR(255) NOT NULL,
    processor_id    VARCHAR(255) NOT NULL,
    version         VARCHAR(50) NOT NULL, -- semver: 1.0.0
    
    -- JSON Schema definition
    definition      JSONB NOT NULL,
    
    -- Validation rules
    strict_mode     BOOLEAN DEFAULT true,  -- Reject extra fields
    required_fields TEXT[],
    
    -- Compatibility
    compatible_with TEXT[],  -- List of schema IDs this can transform from
    
    -- Metadata
    description     TEXT,
    examples        JSONB,
    deprecated      BOOLEAN DEFAULT false,
    created_at      TIMESTAMP DEFAULT NOW(),
    created_by      VARCHAR(255),
    
    UNIQUE(schema_name, processor_id, version)
);

-- Schema transformations
CREATE TABLE schema_transformations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_schema_id  UUID REFERENCES schemas(id),
    to_schema_id    UUID REFERENCES schemas(id),
    
    -- Transformation rules (JSONPath mappings)
    transform_rules JSONB NOT NULL,
    
    -- Optional custom transformation function
    transform_func  TEXT,  -- JavaScript/Lua function
    
    created_at      TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(from_schema_id, to_schema_id)
);

-- Schema usage tracking
CREATE TABLE schema_usage (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_id       UUID REFERENCES schemas(id),
    processor_id    VARCHAR(255),
    blob_count      BIGINT DEFAULT 0,
    last_used       TIMESTAMP,
    
    -- Performance metrics
    avg_validation_ms FLOAT,
    validation_failures INTEGER DEFAULT 0,
    
    PRIMARY KEY(schema_id, processor_id)
);
```

### Processor Configuration (Processor Service - PostgreSQL)
```go
type ProcessorConfig struct {
    ID                  string                 `db:"id"`
    Name                string                 `db:"name"`
    Description         string                 `db:"description"`
    
    // Schema configuration
    InputSchemas        []SchemaRef            `db:"input_schemas"`   // Multiple versions supported
    OutputSchema        SchemaRef              `db:"output_schema"`
    
    // Storage strategy
    StorageStrategy     string                 `db:"storage_strategy"` // inline, external, hybrid
    StorageThreshold    int64                  `db:"storage_threshold_bytes"`
    
    // External storage config (if used)
    StorageBackend      string                 `db:"storage_backend"`  // redis, s3, mongodb
    StorageConfig       map[string]interface{} `db:"storage_config"`   // Encrypted
    
    // Processing configuration
    WorkflowID          string                 `db:"workflow_id"`
    Concurrency         int                    `db:"concurrency"`
    BatchSize           int                    `db:"batch_size"`
    TimeoutSeconds      int                    `db:"timeout_seconds"`
    
    // Event configuration
    InputEvents         []string               `db:"input_events"`
    OutputEvents        []string               `db:"output_events"`
    
    // Cost tracking
    CostPerExecution    float64                `db:"cost_per_execution"`
    MonthlyBudget       float64                `db:"monthly_budget"`
    CurrentSpend        float64                `db:"current_spend"`
    
    Active              bool                   `db:"active"`
    CreatedAt           time.Time              `db:"created_at"`
}

type SchemaRef struct {
    SchemaID            string                 `json:"schema_id"`
    SchemaName          string                 `json:"schema_name"`
    Version             string                 `json:"version"`
    MinVersion          string                 `json:"min_version"`    // Minimum compatible version
}
```

## Content Storage Strategies

### 1. Inline Storage (State Service)
```go
// For small blobs < 1MB
blob := Blob{
    StorageType: "inline",
    InlineData: map[string]interface{}{
        "content": "Short chapter text...",
        "metadata": metadata,
    },
}
```

### 2. External URL Storage (S3/CDN)
```go
// For media files and large documents
blob := Blob{
    StorageType: "url",
    ContentURL: aws.String("https://cdn.reyna.ai/blobs/user123/blob456.json"),
    ContentSize: 5242880, // 5MB
    ContentHash: "sha256:abcd1234...",
}
```

### 3. Processor Cache Storage (Redis)
```go
// For frequently accessed processed content
type CachedContent struct {
    BlobID      string
    Content     []byte
    TTL         time.Duration
}

// Text expansion service uses Redis for recent expansions
redis.Set(ctx, fmt.Sprintf("expanded:%s", blobID), content, 24*time.Hour)
```

## Event Flow

### 1. Small Content Creation (Inline)
```yaml
Client → Studio API:
  POST /api/v1/blobs
  {
    "schema_id": "text-input-v1",
    "data": {
      "content": "Quick note for chapter 3",
      "book_id": "my-novel"
    }
  }

Studio API:
  - Check content size: 28 bytes < 1MB
  - Validate against schema
  - Store inline in State Service

State Service:
  {
    "storage_type": "inline",
    "inline_data": {
      "content": "Quick note for chapter 3",
      "book_id": "my-novel"
    },
    "content_size": 28
  }

State Service → NATS:
  Event: blob.created.inline
  Subscribers can fetch directly from State Service
```

### 2. Large Content Processing (External)
```yaml
Client → Studio API:
  POST /api/v1/blobs
  {
    "schema_id": "chapter-draft-v1",
    "data": {
      "content": "... 50KB of chapter text ...",
      "book_id": "my-novel"
    }
  }

Studio API:
  - Check size: 50KB < 1MB (still inline)
  - Store in State Service
  - Emit event

Text Expansion Worker:
  - Receive event
  - Fetch from State Service (inline data)
  - Process and expand (result: 200KB)
  - Store expanded content externally

Worker → Redis:
  SET expanded:blob789 <200KB content> EX 86400

Worker → State Service:
  POST /api/v1/blobs
  {
    "processor_id": "text-expansion",
    "parent_id": "blob123",
    "storage_type": "reference",
    "content_url": "redis://expanded:blob789",
    "content_size": 204800,
    "inline_data": {
      "preview": "First 500 chars...",
      "metrics": {
        "expansion_ratio": 4.1,
        "readability": 8.5
      }
    }
  }
```

### 3. Client Fetching Strategy
```yaml
Client → Studio API:
  GET /api/v1/blobs/blob789

Studio API → State Service:
  - Fetch blob metadata
  - Check storage_type

If inline:
  Return immediately with full content

If external:
  Options based on size:
    < 100KB: Fetch and return
    100KB-1MB: Return URL with temporary signed access
    > 1MB: Stream response or return CDN URL

Response:
  {
    "id": "blob789",
    "storage_type": "reference",
    "inline_data": {
      "preview": "First 500 chars...",
      "metrics": {...}
    },
    "content_url": "https://cdn.reyna.ai/temp/blob789?token=xyz&expires=...",
    "content_size": 204800
  }
```

## Text Expansion Processor - Hybrid Implementation

### Processing Pipeline
```go
type TextExpansionProcessor struct {
    stateClient  *StateClient
    schemaClient *SchemaClient
    cache        *redis.Client
    storage      *S3Client
    aiClient     *AIClient
}

func (p *TextExpansionProcessor) ProcessBlob(event BlobCreatedEvent) error {
    ctx := context.Background()
    
    // 1. Fetch source blob
    sourceBlob, err := p.stateClient.GetBlob(ctx, event.BlobID)
    if err != nil {
        return fmt.Errorf("fetch blob: %w", err)
    }
    
    // 2. Extract content based on storage type
    var content string
    switch sourceBlob.StorageType {
    case "inline":
        content = sourceBlob.InlineData["content"].(string)
    case "url":
        data, err := p.fetchExternalContent(sourceBlob.ContentURL)
        if err != nil {
            return err
        }
        content = data["content"].(string)
    }
    
    // 3. Check cache for recent processing
    cacheKey := fmt.Sprintf("expanded:%s:v2", sourceBlob.ContentHash)
    if cached, err := p.cache.Get(ctx, cacheKey).Result(); err == nil {
        // Found in cache, create blob with cached content
        return p.createOutputBlob(sourceBlob, []byte(cached), "cache")
    }
    
    // 4. Validate against input schema
    inputSchema, err := p.schemaClient.GetSchema(ctx, sourceBlob.SchemaID)
    if err := p.schemaClient.Validate(sourceBlob.InlineData, inputSchema); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // 5. Process with AI
    expanded, metrics := p.aiClient.ExpandText(content, sourceBlob.InlineData["metadata"])
    
    // 6. Determine storage strategy based on size
    expandedSize := len(expanded)
    var storageType string
    var contentURL *string
    var inlineData interface{}
    
    if expandedSize < 1024*1024 { // < 1MB: inline
        storageType = "inline"
        inlineData = map[string]interface{}{
            "original": content,
            "expanded": expanded,
            "metrics":  metrics,
        }
    } else { // >= 1MB: external
        storageType = "url"
        
        // Store in S3 for permanent storage
        s3Key := fmt.Sprintf("expanded/%s/%s.json", event.UserID, uuid.New())
        url, err := p.storage.Upload(s3Key, map[string]interface{}{
            "expanded": expanded,
            "metrics":  metrics,
        })
        if err != nil {
            return err
        }
        contentURL = &url
        
        // Store preview inline
        inlineData = map[string]interface{}{
            "preview": expanded[:500],
            "metrics": metrics,
        }
    }
    
    // 7. Cache for quick access
    p.cache.Set(ctx, cacheKey, expanded, 24*time.Hour)
    
    // 8. Create output blob in State Service
    outputBlob := Blob{
        ID:            uuid.New().String(),
        UserID:        event.UserID,
        ProcessorID:   "text-expansion",
        SchemaID:      "expanded-text-v1",
        StorageType:   storageType,
        InlineData:    inlineData,
        ContentURL:    contentURL,
        ContentSize:   int64(expandedSize),
        ContentHash:   p.hash(expanded),
        ParentID:      &event.BlobID,
        Title:         fmt.Sprintf("Expanded: %s", sourceBlob.Title),
        Preview:       expanded[:min(500, len(expanded))],
        ProcessingMeta: map[string]interface{}{
            "model":      "gpt-4",
            "tokens":     metrics["tokens"],
            "duration":   metrics["duration_ms"],
        },
    }
    
    return p.stateClient.CreateBlob(ctx, outputBlob)
}
```

## Book Writing Complete Flow

### Data Organization
```yaml
UserState.Books:
  - id: "my-novel"
    title: "My Science Fiction Novel"
    chapters:
      - chapter_num: 1
        title: "The Beginning"
        draft_blob_id: "blob_001"
        processed_blob_ids:
          text-expansion: "blob_002"
          grammar-check: "blob_003"
          
State Service Blobs:
  blob_001: # User's draft (inline)
    storage_type: "inline"
    inline_data:
      content: "Chapter 1 draft text..."
      
  blob_002: # Expanded version (external)
    storage_type: "url"
    content_url: "s3://expanded/blob_002.json"
    inline_data:
      preview: "First 500 chars..."
      metrics: {expansion_ratio: 3.5}
      
  blob_003: # Grammar checked (inline)
    storage_type: "inline"
    inline_data:
      content: "Grammar corrected text..."
      corrections: [...]
```

### Query Optimization
```go
// Studio API: Fetch complete book with all versions
func (s *StudioAPI) GetBookChapters(userID, bookID string) (*BookResponse, error) {
    // 1. Get book structure from UserState
    userState := s.stateClient.GetUserState(userID)
    book := userState.GetBook(bookID)
    
    // 2. Batch fetch all blob IDs
    blobIDs := []string{}
    for _, chapter := range book.Chapters {
        blobIDs = append(blobIDs, chapter.DraftBlobID)
        for _, processedID := range chapter.ProcessedBlobIDs {
            blobIDs = append(blobIDs, processedID)
        }
    }
    
    // 3. Single query to State Service
    blobs := s.stateClient.GetBlobs(blobIDs)
    
    // 4. Group by storage type for optimized fetching
    inlineBlobs := filterInline(blobs)
    externalBlobs := filterExternal(blobs)
    
    // 5. Parallel fetch external content if needed
    contents := make(map[string]interface{})
    
    // Inline blobs already have content
    for _, blob := range inlineBlobs {
        contents[blob.ID] = blob.InlineData
    }
    
    // Fetch external content in parallel
    var wg sync.WaitGroup
    for _, blob := range externalBlobs {
        wg.Add(1)
        go func(b Blob) {
            defer wg.Done()
            if blob.ContentSize < 100*1024 { // <100KB: fetch now
                content := s.fetchContent(b.ContentURL)
                contents[b.ID] = content
            } else { // Large: return URL
                contents[b.ID] = map[string]interface{}{
                    "url": s.generateSignedURL(b.ContentURL),
                    "size": b.ContentSize,
                }
            }
        }(blob)
    }
    wg.Wait()
    
    return &BookResponse{
        Book:     book,
        Chapters: contents,
    }, nil
}
```

## Advantages

1. **Performance**: Small blobs are fast (inline), large blobs don't bloat DB
2. **Flexibility**: Processors can choose optimal storage
3. **Query Efficiency**: All metadata in one place for fast queries
4. **Cost Optimization**: Use appropriate storage for content size
5. **Cache Layer**: Redis for frequently accessed content
6. **Consistency**: Single source of truth for relationships

## Disadvantages

1. **Complexity**: Multiple storage backends to manage
2. **Cache Invalidation**: Keeping caches synchronized
3. **Storage Overhead**: Some duplication between inline previews and full content
4. **Migration Complexity**: Moving between storage types as content grows

## Implementation Phases

1. **Phase 1: Core Infrastructure**
   - Schema Service with PostgreSQL
   - State Service with hybrid storage
   - Basic inline storage only

2. **Phase 2: External Storage**
   - S3 integration for large blobs
   - Redis cache layer
   - Content URL generation

3. **Phase 3: Smart Routing**
   - Storage strategy optimizer
   - Automatic migration between storage types
   - CDN integration

4. **Phase 4: Advanced Features**
   - Content deduplication
   - Compression for text blobs
   - P2P content sharing between users