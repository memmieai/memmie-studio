# State Service Design - Blob Storage System (UPDATED)

## Overview

The State Service (Port 8006) is the central repository for all user-created and processor-derived blobs. It uses MongoDB for flexible schema storage, maintains blob relationships, and integrates with the Schema Service for validation.

## Core Responsibilities

1. **Blob Storage**: Store all blobs with dynamic schema-validated data
2. **Relationship Management**: Maintain parent-child and cross-references
3. **Schema Integration**: Validate all blob data against registered schemas
4. **Event Emission**: Publish NATS events for processor triggers
5. **User State**: Track books, conversations, and organized content

## Data Models

### Blob Document (MongoDB)
```go
type Blob struct {
    ID          string              `bson:"_id"`
    UserID      string              `bson:"user_id"`
    ProcessorID string              `bson:"processor_id"` // Which processor created this
    SchemaID    string              `bson:"schema_id"`    // References Schema Service
    SchemaVersion string            `bson:"schema_version"`
    
    // Dynamic content matching schema
    Data        interface{}         `bson:"data"`         // Validated against schema
    
    // Relationships
    ParentID    *string             `bson:"parent_id,omitempty"`
    DerivedIDs  []string            `bson:"derived_ids"`  // Blobs derived from this
    
    // Organization
    BookID      *string             `bson:"book_id,omitempty"`
    ConversationID *string          `bson:"conversation_id,omitempty"`
    CollectionID *string            `bson:"collection_id,omitempty"`
    
    // Metadata for queries
    Title       string              `bson:"title"`
    Preview     string              `bson:"preview"`      // First 500 chars
    Tags        []string            `bson:"tags"`
    ContentSize int64               `bson:"content_size_bytes"`
    
    // Processing state
    ProcessingState string          `bson:"processing_state"` // pending, processing, completed, failed
    ProcessingMeta map[string]interface{} `bson:"processing_meta"`
    
    // Timestamps
    CreatedAt   time.Time           `bson:"created_at"`
    UpdatedAt   time.Time           `bson:"updated_at"`
    AccessedAt  time.Time           `bson:"accessed_at"`
}
```

### UserState Document
```go
type UserState struct {
    ID              string            `bson:"_id"`
    UserID          string            `bson:"user_id"`
    
    // Statistics
    TotalBlobs      int               `bson:"total_blobs"`
    TotalSize       int64             `bson:"total_size_bytes"`
    BlobsByProcessor map[string]int   `bson:"blobs_by_processor"`
    
    // Organized content
    Books           []BookProject     `bson:"books"`
    Conversations   []Conversation    `bson:"conversations"`
    Collections     []BlobCollection  `bson:"collections"`
    
    // Processor instances
    ProcessorInstances []ProcessorInstance `bson:"processor_instances"`
    
    // Quotas
    MaxBlobs        int               `bson:"max_blobs"`
    MaxSizeBytes    int64             `bson:"max_size_bytes"`
    
    UpdatedAt       time.Time         `bson:"updated_at"`
}

type BookProject struct {
    ID              string            `bson:"id"`
    Title           string            `bson:"title"`
    Description     string            `bson:"description"`
    Chapters        []ChapterRef      `bson:"chapters"`
    CreatedAt       time.Time         `bson:"created_at"`
}

type ChapterRef struct {
    ChapterNum      int               `bson:"chapter_num"`
    Title           string            `bson:"title"`
    DraftBlobID     string            `bson:"draft_blob_id"`
    ProcessedBlobIDs map[string]string `bson:"processed_blob_ids"` // processor_id -> blob_id
}
```

## API Endpoints

### Blob Operations
```go
// Create a new blob
POST /api/v1/users/{user_id}/blobs
Request:
{
    "content": "Chapter 1 draft",
    "content_type": "text/plain",
    "provider_id": "book:my-novel",
    "parent_id": "optional-parent-blob-id",
    "metadata": {
        "chapter": 1,
        "status": "draft"
    }
}
Response:
{
    "id": "507f1f77bcf86cd799439011",
    "version": 1,
    "created_at": "2024-01-01T00:00:00Z"
}

// Get blob by ID
GET /api/v1/users/{user_id}/blobs/{blob_id}
Response:
{
    "id": "507f1f77bcf86cd799439011",
    "content": "Chapter 1 draft",
    "metadata": {...},
    "parent_id": null,
    "children_ids": ["507f1f77bcf86cd799439012"],
    "version": 1
}

// List user's blobs with filtering
GET /api/v1/users/{user_id}/blobs?provider_id=book:my-novel&depth=0
Response:
{
    "blobs": [...],
    "total": 42,
    "page": 1
}

// Get user's DAG structure
GET /api/v1/users/{user_id}/dag
Response:
{
    "roots": [
        {
            "id": "...",
            "content": "...",
            "children": [
                {
                    "id": "...",
                    "content": "...",
                    "children": []
                }
            ]
        }
    ],
    "total_nodes": 42,
    "max_depth": 5
}
```

### Delta Operations
```go
// Apply delta to blob
POST /api/v1/users/{user_id}/blobs/{blob_id}/deltas
Request:
{
    "type": "update",
    "path": "/content",
    "new_value": "Updated chapter content",
    "provider_id": "text-expander"
}
Response:
{
    "blob_id": "507f1f77bcf86cd799439011",
    "new_version": 2,
    "delta_id": "delta-123"
}

// Get blob history
GET /api/v1/users/{user_id}/blobs/{blob_id}/history
Response:
{
    "deltas": [
        {
            "id": "delta-123",
            "type": "update",
            "applied_at": "2024-01-01T00:00:00Z",
            "provider_id": "text-expander"
        }
    ]
}

// Revert to specific version
POST /api/v1/users/{user_id}/blobs/{blob_id}/revert
Request:
{
    "target_version": 1
}
```

## MongoDB Indexes

```javascript
// Optimize blob queries
db.blobs.createIndex({ "user_id": 1, "created_at": -1 })
db.blobs.createIndex({ "user_id": 1, "provider_id": 1 })
db.blobs.createIndex({ "parent_id": 1 })
db.blobs.createIndex({ "user_id": 1, "depth": 1 })
db.blobs.createIndex({ "metadata.status": 1 })
db.blobs.createIndex({ "tags": 1 })

// Text search
db.blobs.createIndex({ "content": "text" })

// User state queries
db.user_blob_states.createIndex({ "user_id": 1 }, { unique: true })
```

## Service Implementation

### Repository Layer
```go
package repository

import (
    "context"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
)

type BlobRepository interface {
    Create(ctx context.Context, blob *Blob) error
    GetByID(ctx context.Context, userID string, blobID primitive.ObjectID) (*Blob, error)
    GetByUser(ctx context.Context, userID string, filter BlobFilter) ([]*Blob, error)
    GetDAG(ctx context.Context, userID string) (*DAGStructure, error)
    ApplyDelta(ctx context.Context, blobID primitive.ObjectID, delta Delta) error
    GetChildren(ctx context.Context, blobID primitive.ObjectID) ([]*Blob, error)
}

type MongoBlobRepository struct {
    db         *mongo.Database
    collection *mongo.Collection
}

func (r *MongoBlobRepository) Create(ctx context.Context, blob *Blob) error {
    blob.ID = primitive.NewObjectID()
    blob.CreatedAt = time.Now()
    blob.UpdatedAt = time.Now()
    blob.Version = 1
    
    // Calculate depth based on parent
    if blob.ParentID != nil {
        parent, err := r.GetByID(ctx, blob.UserID, *blob.ParentID)
        if err != nil {
            return err
        }
        blob.Depth = parent.Depth + 1
        
        // Update parent's children
        update := bson.M{
            "$push": bson.M{"children_ids": blob.ID},
            "$set": bson.M{"updated_at": time.Now()},
        }
        r.collection.UpdateByID(ctx, parent.ID, update)
    }
    
    _, err := r.collection.InsertOne(ctx, blob)
    return err
}

func (r *MongoBlobRepository) GetDAG(ctx context.Context, userID string) (*DAGStructure, error) {
    // Get root blobs (depth = 0)
    filter := bson.M{
        "user_id": userID,
        "depth": 0,
    }
    
    cursor, err := r.collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    
    var roots []*Blob
    if err := cursor.All(ctx, &roots); err != nil {
        return nil, err
    }
    
    // Recursively build tree
    dag := &DAGStructure{
        Roots: make([]*BlobNode, 0, len(roots)),
    }
    
    for _, root := range roots {
        node := r.buildDAGNode(ctx, root)
        dag.Roots = append(dag.Roots, node)
        dag.TotalNodes++
        if root.Depth > dag.MaxDepth {
            dag.MaxDepth = root.Depth
        }
    }
    
    return dag, nil
}
```

### Service Layer
```go
package service

type StateService struct {
    blobRepo BlobRepository
    eventBus EventBus
}

func (s *StateService) CreateBlob(ctx context.Context, userID string, req CreateBlobRequest) (*Blob, error) {
    // Check user quotas
    state, err := s.getUserState(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    if state.TotalBlobs >= state.MaxBlobs {
        return nil, ErrQuotaExceeded
    }
    
    blob := &Blob{
        UserID:      userID,
        Content:     req.Content,
        ContentType: req.ContentType,
        ProviderID:  req.ProviderID,
        ParentID:    req.ParentID,
        Metadata:    req.Metadata,
        Size:        int64(len(req.Content)),
    }
    
    if err := s.blobRepo.Create(ctx, blob); err != nil {
        return nil, err
    }
    
    // Publish event
    s.eventBus.Publish(ctx, "blob.created", BlobCreatedEvent{
        BlobID:     blob.ID.Hex(),
        UserID:     userID,
        ProviderID: req.ProviderID,
        ParentID:   req.ParentID,
    })
    
    return blob, nil
}

func (s *StateService) ApplyDelta(ctx context.Context, blobID string, delta Delta) error {
    // Apply delta
    if err := s.blobRepo.ApplyDelta(ctx, blobID, delta); err != nil {
        return err
    }
    
    // Publish event for downstream processing
    s.eventBus.Publish(ctx, "delta.applied", DeltaAppliedEvent{
        BlobID:  blobID,
        DeltaID: delta.ID,
        Type:    delta.Type,
    })
    
    // Trigger reprocessing of child blobs if needed
    if delta.Type == "update" || delta.Type == "transform" {
        s.cascadeUpdate(ctx, blobID)
    }
    
    return nil
}
```

## Performance Optimizations

### Caching Strategy
```go
// Use Redis for frequently accessed blobs
type CachedBlobRepository struct {
    mongo BlobRepository
    redis *redis.Client
}

func (r *CachedBlobRepository) GetByID(ctx context.Context, userID string, blobID string) (*Blob, error) {
    // Check cache first
    key := fmt.Sprintf("blob:%s:%s", userID, blobID)
    cached, err := r.redis.Get(ctx, key).Result()
    if err == nil {
        var blob Blob
        json.Unmarshal([]byte(cached), &blob)
        return &blob, nil
    }
    
    // Fallback to MongoDB
    blob, err := r.mongo.GetByID(ctx, userID, blobID)
    if err != nil {
        return nil, err
    }
    
    // Cache for 5 minutes
    data, _ := json.Marshal(blob)
    r.redis.Set(ctx, key, data, 5*time.Minute)
    
    return blob, nil
}
```

### Batch Operations
```go
// Batch create for bulk imports
func (s *StateService) CreateBlobsBatch(ctx context.Context, userID string, blobs []*Blob) error {
    // Use MongoDB bulk write
    models := make([]mongo.WriteModel, len(blobs))
    for i, blob := range blobs {
        blob.ID = primitive.NewObjectID()
        blob.UserID = userID
        models[i] = mongo.NewInsertOneModel().SetDocument(blob)
    }
    
    _, err := s.blobRepo.BulkWrite(ctx, models)
    return err
}
```

## Event Integration

### NATS Events Published
```go
// blob.created - When new blob is created
type BlobCreatedEvent struct {
    BlobID     string    `json:"blob_id"`
    UserID     string    `json:"user_id"`
    ProviderID string    `json:"provider_id"`
    ParentID   *string   `json:"parent_id,omitempty"`
    CreatedAt  time.Time `json:"created_at"`
}

// blob.updated - When blob content changes
type BlobUpdatedEvent struct {
    BlobID    string    `json:"blob_id"`
    UserID    string    `json:"user_id"`
    Version   int       `json:"version"`
    DeltaID   string    `json:"delta_id"`
    UpdatedAt time.Time `json:"updated_at"`
}

// dag.modified - When DAG structure changes
type DAGModifiedEvent struct {
    UserID       string   `json:"user_id"`
    AffectedIDs  []string `json:"affected_ids"`
    OperationType string  `json:"operation_type"`
}
```

## Error Handling

```go
var (
    ErrBlobNotFound    = errors.New("blob not found")
    ErrQuotaExceeded   = errors.New("user quota exceeded")
    ErrInvalidParent   = errors.New("invalid parent blob")
    ErrCyclicReference = errors.New("cyclic reference detected")
    ErrUnauthorized    = errors.New("unauthorized access")
)
```

## Configuration

```yaml
# config/state-service.yaml
service:
  port: 8006
  name: state-service

mongodb:
  uri: mongodb://user:pass@localhost:27017
  database: memmie_state
  
redis:
  url: redis://localhost:6379
  cache_ttl: 5m
  
quotas:
  default_max_blobs: 10000
  default_max_size_mb: 1000
  
nats:
  url: nats://localhost:4222
  subjects:
    blob_created: state.blob.created
    blob_updated: state.blob.updated
    dag_modified: state.dag.modified
```

## Testing

```go
func TestBlobCreation(t *testing.T) {
    // Test blob creation with parent
    parent := createTestBlob(t, userID, nil)
    child := createTestBlob(t, userID, &parent.ID)
    
    assert.Equal(t, parent.Depth + 1, child.Depth)
    assert.Contains(t, parent.ChildrenIDs, child.ID)
}

func TestDAGConstruction(t *testing.T) {
    // Create complex DAG
    root := createTestBlob(t, userID, nil)
    child1 := createTestBlob(t, userID, &root.ID)
    child2 := createTestBlob(t, userID, &root.ID)
    grandchild := createTestBlob(t, userID, &child1.ID)
    
    dag, err := service.GetDAG(ctx, userID)
    assert.NoError(t, err)
    assert.Equal(t, 4, dag.TotalNodes)
    assert.Equal(t, 2, dag.MaxDepth)
}
```

This State Service design provides efficient blob storage with DAG management, delta tracking, and seamless integration with the rest of the Memmie platform.