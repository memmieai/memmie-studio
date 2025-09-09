# State Service Design - Blob Storage System with Dynamic Buckets

## Overview

The State Service (Port 8006) is the central repository for all user-created and processor-derived blobs. It uses MongoDB for flexible schema storage, maintains blob relationships, and integrates with the Schema Service for validation. The service uses a dynamic bucket system for flexible organization of blobs into any hierarchical structure.

## Core Responsibilities

1. **Blob Storage**: Store all blobs with dynamic schema-validated data
2. **Bucket Management**: Organize blobs into flexible, hierarchical buckets
3. **Relationship Management**: Maintain parent-child and cross-references
4. **Schema Integration**: Validate all blob data against registered schemas
5. **Event Emission**: Publish NATS events for processor triggers
6. **User State**: Track buckets and organized content

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
    
    // Bucket Organization (flexible containers)
    BucketIDs   []string            `bson:"bucket_ids"`   // Which buckets contain this blob
    
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

// Bucket - Flexible container for organizing blobs
type Bucket struct {
    ID              string                 `bson:"_id"`
    UserID          string                 `bson:"user_id"`
    Name            string                 `bson:"name"`
    Type            string                 `bson:"type"`        // book, album, research, conversation, etc.
    
    // Hierarchical structure
    ParentBucketID  *string                `bson:"parent_bucket_id,omitempty"`
    ChildBucketIDs  []string               `bson:"child_bucket_ids"`
    
    // Contained blobs
    BlobIDs         []string               `bson:"blob_ids"`
    BlobCount       int                    `bson:"blob_count"`
    
    // User-defined metadata
    Metadata        map[string]interface{} `bson:"metadata"`
    
    // Display properties
    Description     string                 `bson:"description"`
    Icon            string                 `bson:"icon"`
    Color           string                 `bson:"color"`
    SortOrder       int                    `bson:"sort_order"`
    
    // Access control
    IsPublic        bool                   `bson:"is_public"`
    SharedWith      []string               `bson:"shared_with"`  // User IDs
    
    CreatedAt       time.Time              `bson:"created_at"`
    UpdatedAt       time.Time              `bson:"updated_at"`
}
```

### UserState Document
```go
type UserState struct {
    ID              string            `bson:"_id"`
    UserID          string            `bson:"user_id"`
    
    // Statistics
    TotalBlobs      int               `bson:"total_blobs"`
    TotalBuckets    int               `bson:"total_buckets"`
    TotalSize       int64             `bson:"total_size_bytes"`
    BlobsByProcessor map[string]int   `bson:"blobs_by_processor"`
    
    // Root buckets (top-level organization)
    RootBucketIDs   []string          `bson:"root_bucket_ids"`
    
    // Processor instances
    ProcessorInstances []ProcessorInstance `bson:"processor_instances"`
    
    // Quotas
    MaxBlobs        int               `bson:"max_blobs"`
    MaxBuckets      int               `bson:"max_buckets"`
    MaxSizeBytes    int64             `bson:"max_size_bytes"`
    
    UpdatedAt       time.Time         `bson:"updated_at"`
}
```

## API Endpoints

### Blob Operations
```go
// Create a new blob
POST /api/v1/users/{user_id}/blobs
Request:
{
    "processor_id": "user-input",
    "schema_id": "text-input-v1",
    "data": {
        "content": "Chapter 1 draft",
        "style": "creative"
    },
    "bucket_ids": ["bucket-123"],  // Optional: add to specific buckets
    "parent_id": "optional-parent-blob-id",
    "metadata": {
        "title": "Chapter 1",
        "tags": ["draft", "fiction"]
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
GET /api/v1/users/{user_id}/blobs?bucket_id=bucket-123&processor_id=text-expansion
Response:
{
    "blobs": [...],
    "total": 42,
    "page": 1
}

// Get user's DAG structure
GET /api/v1/users/{user_id}/dag?bucket_id=bucket-123  // Optional: filter by bucket
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

### Bucket Operations
```go
// Create a new bucket
POST /api/v1/users/{user_id}/buckets
Request:
{
    "name": "My Novel",
    "type": "book",
    "parent_bucket_id": null,
    "metadata": {
        "genre": "science fiction",
        "target_word_count": 80000
    },
    "description": "My first science fiction novel",
    "icon": "📚",
    "color": "#4A90E2"
}

// Get bucket with contents
GET /api/v1/users/{user_id}/buckets/{bucket_id}
Response:
{
    "id": "bucket-123",
    "name": "My Novel",
    "type": "book",
    "blob_ids": ["blob-1", "blob-2"],
    "child_bucket_ids": ["chapter-1", "chapter-2"],
    "metadata": {...}
}

// List user's buckets
GET /api/v1/users/{user_id}/buckets?type=book
Response:
{
    "buckets": [
        {
            "id": "bucket-123",
            "name": "My Novel",
            "type": "book",
            "blob_count": 42,
            "child_count": 12
        }
    ]
}

// Add blob to bucket
POST /api/v1/users/{user_id}/buckets/{bucket_id}/blobs
Request:
{
    "blob_id": "blob-456"
}

// Move bucket to new parent
PUT /api/v1/users/{user_id}/buckets/{bucket_id}/parent
Request:
{
    "parent_bucket_id": "bucket-789"  // null for root level
}

// Get bucket hierarchy
GET /api/v1/users/{user_id}/buckets/{bucket_id}/tree
Response:
{
    "id": "bucket-123",
    "name": "My Novel",
    "children": [
        {
            "id": "chapter-1",
            "name": "Chapter 1",
            "children": []
        }
    ]
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
db.blobs.createIndex({ "user_id": 1, "processor_id": 1 })
db.blobs.createIndex({ "bucket_ids": 1 })
db.blobs.createIndex({ "parent_id": 1 })
db.blobs.createIndex({ "schema_id": 1 })
db.blobs.createIndex({ "tags": 1 })

// Bucket queries
db.buckets.createIndex({ "user_id": 1, "type": 1 })
db.buckets.createIndex({ "parent_bucket_id": 1 })
db.buckets.createIndex({ "blob_ids": 1 })

// Text search
db.blobs.createIndex({ "title": "text", "preview": "text" })

// User state queries
db.user_states.createIndex({ "user_id": 1 }, { unique: true })
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
    GetByBucket(ctx context.Context, bucketID string) ([]*Blob, error)
    GetDAG(ctx context.Context, userID string, bucketID *string) (*DAGStructure, error)
    ApplyDelta(ctx context.Context, blobID primitive.ObjectID, delta Delta) error
    GetChildren(ctx context.Context, blobID primitive.ObjectID) ([]*Blob, error)
    AddToBucket(ctx context.Context, blobID, bucketID string) error
    RemoveFromBucket(ctx context.Context, blobID, bucketID string) error
}

type BucketRepository interface {
    Create(ctx context.Context, bucket *Bucket) error
    GetByID(ctx context.Context, userID string, bucketID string) (*Bucket, error)
    GetByUser(ctx context.Context, userID string, filter BucketFilter) ([]*Bucket, error)
    GetRootBuckets(ctx context.Context, userID string) ([]*Bucket, error)
    GetChildren(ctx context.Context, bucketID string) ([]*Bucket, error)
    GetTree(ctx context.Context, bucketID string) (*BucketTree, error)
    Update(ctx context.Context, bucket *Bucket) error
    Delete(ctx context.Context, bucketID string) error
    MoveToParent(ctx context.Context, bucketID string, newParentID *string) error
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
    blobRepo   BlobRepository
    bucketRepo BucketRepository
    schemaClient SchemaClient
    eventBus   EventBus
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
    
    // Validate data against schema
    if err := s.schemaClient.Validate(ctx, req.SchemaID, req.Data); err != nil {
        return nil, fmt.Errorf("schema validation failed: %w", err)
    }
    
    blob := &Blob{
        UserID:      userID,
        ProcessorID: req.ProcessorID,
        SchemaID:    req.SchemaID,
        Data:        req.Data,
        BucketIDs:   req.BucketIDs,
        ParentID:    req.ParentID,
        Title:       req.Metadata.Title,
        Tags:        req.Metadata.Tags,
        ContentSize: calculateDataSize(req.Data),
    }
    
    if err := s.blobRepo.Create(ctx, blob); err != nil {
        return nil, err
    }
    
    // Add to buckets if specified
    for _, bucketID := range req.BucketIDs {
        s.bucketRepo.AddBlob(ctx, bucketID, blob.ID)
    }
    
    // Publish event with schema information
    s.eventBus.Publish(ctx, fmt.Sprintf("blob.created.%s", req.SchemaID), BlobCreatedEvent{
        BlobID:     blob.ID.Hex(),
        UserID:     userID,
        ProcessorID: req.ProcessorID,
        SchemaID:   req.SchemaID,
        BucketIDs:  req.BucketIDs,
        ParentID:   req.ParentID,
    })
    
    return blob, nil
}

func (s *StateService) CreateBucket(ctx context.Context, userID string, req CreateBucketRequest) (*Bucket, error) {
    // Check user quotas
    state, err := s.getUserState(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    if state.TotalBuckets >= state.MaxBuckets {
        return nil, ErrBucketQuotaExceeded
    }
    
    bucket := &Bucket{
        ID:             generateBucketID(),
        UserID:         userID,
        Name:           req.Name,
        Type:           req.Type,
        ParentBucketID: req.ParentBucketID,
        Metadata:       req.Metadata,
        Description:    req.Description,
        Icon:           req.Icon,
        Color:          req.Color,
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }
    
    if err := s.bucketRepo.Create(ctx, bucket); err != nil {
        return nil, err
    }
    
    // Update parent bucket if exists
    if req.ParentBucketID != nil {
        parent, err := s.bucketRepo.GetByID(ctx, userID, *req.ParentBucketID)
        if err != nil {
            return nil, err
        }
        parent.ChildBucketIDs = append(parent.ChildBucketIDs, bucket.ID)
        s.bucketRepo.Update(ctx, parent)
    } else {
        // Add to user's root buckets
        s.updateUserRootBuckets(ctx, userID, bucket.ID)
    }
    
    // Publish event
    s.eventBus.Publish(ctx, "bucket.created", BucketCreatedEvent{
        BucketID:   bucket.ID,
        UserID:     userID,
        Type:       req.Type,
        ParentID:   req.ParentBucketID,
    })
    
    return bucket, nil
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
// blob.created.<schema-id> - When new blob is created
type BlobCreatedEvent struct {
    BlobID      string    `json:"blob_id"`
    UserID      string    `json:"user_id"`
    ProcessorID string    `json:"processor_id"`
    SchemaID    string    `json:"schema_id"`
    BucketIDs   []string  `json:"bucket_ids"`
    ParentID    *string   `json:"parent_id,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// blob.updated - When blob content changes
type BlobUpdatedEvent struct {
    BlobID    string    `json:"blob_id"`
    UserID    string    `json:"user_id"`
    SchemaID  string    `json:"schema_id"`
    Version   int       `json:"version"`
    DeltaID   string    `json:"delta_id"`
    UpdatedAt time.Time `json:"updated_at"`
}

// bucket.created - When new bucket is created
type BucketCreatedEvent struct {
    BucketID  string    `json:"bucket_id"`
    UserID    string    `json:"user_id"`
    Type      string    `json:"type"`
    ParentID  *string   `json:"parent_id,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}

// bucket.blob.added - When blob is added to bucket
type BucketBlobAddedEvent struct {
    BucketID  string    `json:"bucket_id"`
    BlobID    string    `json:"blob_id"`
    UserID    string    `json:"user_id"`
    AddedAt   time.Time `json:"added_at"`
}

// bucket.structure.changed - When bucket hierarchy changes
type BucketStructureChangedEvent struct {
    BucketID      string    `json:"bucket_id"`
    UserID        string    `json:"user_id"`
    OldParentID   *string   `json:"old_parent_id,omitempty"`
    NewParentID   *string   `json:"new_parent_id,omitempty"`
    ChangedAt     time.Time `json:"changed_at"`
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