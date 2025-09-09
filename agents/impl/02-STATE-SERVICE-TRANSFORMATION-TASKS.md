# State Service Transformation Tasks

## Prerequisites
- Existing memmie-state service running
- MongoDB running
- Schema Service completed and running

## Task 1: Add Schema Client Dependency
**File**: `memmie-state/go.mod`
```bash
go get github.com/memmieai/memmie-schema/pkg/client
```

## Task 2: Create New Models for Blobs and Buckets
**File**: `internal/models/blob.go`
```go
package models

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Blob struct {
    ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
    UserID      string                 `bson:"user_id" json:"user_id"`
    ProcessorID string                 `bson:"processor_id" json:"processor_id"`
    SchemaID    string                 `bson:"schema_id" json:"schema_id"`
    Data        interface{}            `bson:"data" json:"data"`
    BucketIDs   []string              `bson:"bucket_ids" json:"bucket_ids"`
    ParentID    *string               `bson:"parent_id,omitempty" json:"parent_id,omitempty"`
    DerivedIDs  []string              `bson:"derived_ids" json:"derived_ids"`
    
    // Metadata
    Title       string                 `bson:"title" json:"title"`
    Preview     string                 `bson:"preview" json:"preview"`
    Tags        []string              `bson:"tags" json:"tags"`
    Size        int64                 `bson:"size_bytes" json:"size_bytes"`
    
    // Timestamps
    CreatedAt   time.Time             `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time             `bson:"updated_at" json:"updated_at"`
    AccessedAt  time.Time             `bson:"accessed_at" json:"accessed_at"`
}

type CreateBlobRequest struct {
    ProcessorID string                 `json:"processor_id"`
    SchemaID    string                 `json:"schema_id"`
    Data        interface{}            `json:"data"`
    BucketIDs   []string              `json:"bucket_ids,omitempty"`
    ParentID    *string               `json:"parent_id,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

**File**: `internal/models/bucket.go`
```go
package models

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Bucket struct {
    ID             primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
    UserID         string                 `bson:"user_id" json:"user_id"`
    Name           string                 `bson:"name" json:"name"`
    Type           string                 `bson:"type" json:"type"`
    
    // Hierarchy
    ParentBucketID *string               `bson:"parent_bucket_id,omitempty" json:"parent_bucket_id,omitempty"`
    ChildBucketIDs []string              `bson:"child_bucket_ids" json:"child_bucket_ids"`
    
    // Contents
    BlobIDs        []string              `bson:"blob_ids" json:"blob_ids"`
    BlobCount      int                   `bson:"blob_count" json:"blob_count"`
    TotalSize      int64                 `bson:"total_size_bytes" json:"total_size_bytes"`
    
    // Metadata
    Description    string                 `bson:"description" json:"description"`
    Icon           string                 `bson:"icon" json:"icon"`
    Color          string                 `bson:"color" json:"color"`
    Metadata       map[string]interface{} `bson:"metadata" json:"metadata"`
    
    // Settings
    SortOrder      int                   `bson:"sort_order" json:"sort_order"`
    IsPublic       bool                  `bson:"is_public" json:"is_public"`
    SharedWith     []string              `bson:"shared_with" json:"shared_with"`
    
    // Timestamps
    CreatedAt      time.Time             `bson:"created_at" json:"created_at"`
    UpdatedAt      time.Time             `bson:"updated_at" json:"updated_at"`
}

type CreateBucketRequest struct {
    Name           string                 `json:"name"`
    Type           string                 `json:"type"`
    ParentBucketID *string               `json:"parent_bucket_id,omitempty"`
    Description    string                 `json:"description,omitempty"`
    Icon           string                 `json:"icon,omitempty"`
    Color          string                 `json:"color,omitempty"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
```

## Task 3: Create Blob Repository
**File**: `internal/repository/blob_repository.go`
```go
package repository

import (
    "context"
    "fmt"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    
    "github.com/memmieai/memmie-state/internal/models"
)

type BlobRepository struct {
    collection *mongo.Collection
    logger     logger.Logger
}

func NewBlobRepository(db *mongo.Database, logger logger.Logger) *BlobRepository {
    collection := db.Collection("blobs")
    
    // Create indexes
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    indexes := []mongo.IndexModel{
        {Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
        {Keys: bson.D{{Key: "bucket_ids", Value: 1}}},
        {Keys: bson.D{{Key: "processor_id", Value: 1}}},
        {Keys: bson.D{{Key: "schema_id", Value: 1}}},
        {Keys: bson.D{{Key: "parent_id", Value: 1}}},
        {Keys: bson.D{{Key: "tags", Value: 1}}},
    }
    
    collection.Indexes().CreateMany(ctx, indexes)
    
    return &BlobRepository{
        collection: collection,
        logger:     logger,
    }
}

func (r *BlobRepository) Create(ctx context.Context, blob *models.Blob) error {
    blob.ID = primitive.NewObjectID()
    blob.CreatedAt = time.Now()
    blob.UpdatedAt = time.Now()
    blob.AccessedAt = time.Now()
    
    if blob.DerivedIDs == nil {
        blob.DerivedIDs = []string{}
    }
    if blob.Tags == nil {
        blob.Tags = []string{}
    }
    
    result, err := r.collection.InsertOne(ctx, blob)
    if err != nil {
        return fmt.Errorf("failed to create blob: %w", err)
    }
    
    blob.ID = result.InsertedID.(primitive.ObjectID)
    return nil
}

func (r *BlobRepository) GetByID(ctx context.Context, id string) (*models.Blob, error) {
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return nil, fmt.Errorf("invalid blob ID: %w", err)
    }
    
    var blob models.Blob
    err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&blob)
    if err == mongo.ErrNoDocuments {
        return nil, fmt.Errorf("blob not found")
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get blob: %w", err)
    }
    
    // Update accessed time
    r.collection.UpdateOne(ctx, 
        bson.M{"_id": objectID},
        bson.M{"$set": bson.M{"accessed_at": time.Now()}},
    )
    
    return &blob, nil
}

func (r *BlobRepository) GetByBucket(ctx context.Context, bucketID string) ([]*models.Blob, error) {
    filter := bson.M{"bucket_ids": bucketID}
    
    cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
    if err != nil {
        return nil, fmt.Errorf("failed to query blobs: %w", err)
    }
    defer cursor.Close(ctx)
    
    var blobs []*models.Blob
    if err := cursor.All(ctx, &blobs); err != nil {
        return nil, fmt.Errorf("failed to decode blobs: %w", err)
    }
    
    return blobs, nil
}

func (r *BlobRepository) AddToBucket(ctx context.Context, blobID, bucketID string) error {
    objectID, err := primitive.ObjectIDFromHex(blobID)
    if err != nil {
        return fmt.Errorf("invalid blob ID: %w", err)
    }
    
    update := bson.M{
        "$addToSet": bson.M{"bucket_ids": bucketID},
        "$set": bson.M{"updated_at": time.Now()},
    }
    
    result, err := r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
    if err != nil {
        return fmt.Errorf("failed to add blob to bucket: %w", err)
    }
    
    if result.MatchedCount == 0 {
        return fmt.Errorf("blob not found")
    }
    
    return nil
}

// Add test file: internal/repository/blob_repository_test.go
```

## Task 4: Create Bucket Repository
**File**: `internal/repository/bucket_repository.go`
```go
package repository

import (
    "context"
    "fmt"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    
    "github.com/memmieai/memmie-state/internal/models"
)

type BucketRepository struct {
    collection *mongo.Collection
    logger     logger.Logger
}

func NewBucketRepository(db *mongo.Database, logger logger.Logger) *BucketRepository {
    collection := db.Collection("buckets")
    
    // Create indexes
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    indexes := []mongo.IndexModel{
        {Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "type", Value: 1}}},
        {Keys: bson.D{{Key: "parent_bucket_id", Value: 1}}},
        {Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "name", Value: 1}}},
    }
    
    collection.Indexes().CreateMany(ctx, indexes)
    
    return &BucketRepository{
        collection: collection,
        logger:     logger,
    }
}

func (r *BucketRepository) Create(ctx context.Context, bucket *models.Bucket) error {
    bucket.ID = primitive.NewObjectID()
    bucket.CreatedAt = time.Now()
    bucket.UpdatedAt = time.Now()
    
    if bucket.ChildBucketIDs == nil {
        bucket.ChildBucketIDs = []string{}
    }
    if bucket.BlobIDs == nil {
        bucket.BlobIDs = []string{}
    }
    if bucket.SharedWith == nil {
        bucket.SharedWith = []string{}
    }
    
    result, err := r.collection.InsertOne(ctx, bucket)
    if err != nil {
        return fmt.Errorf("failed to create bucket: %w", err)
    }
    
    bucket.ID = result.InsertedID.(primitive.ObjectID)
    
    // Update parent bucket if exists
    if bucket.ParentBucketID != nil {
        r.addChildToParent(ctx, *bucket.ParentBucketID, bucket.ID.Hex())
    }
    
    return nil
}

func (r *BucketRepository) GetTree(ctx context.Context, bucketID string) (*models.BucketTree, error) {
    bucket, err := r.GetByID(ctx, bucketID)
    if err != nil {
        return nil, err
    }
    
    tree := &models.BucketTree{
        Bucket:   bucket,
        Children: []*models.BucketTree{},
    }
    
    // Recursively load children
    for _, childID := range bucket.ChildBucketIDs {
        childTree, err := r.GetTree(ctx, childID)
        if err != nil {
            r.logger.Warn("Failed to load child bucket", "id", childID, "error", err)
            continue
        }
        tree.Children = append(tree.Children, childTree)
    }
    
    return tree, nil
}

func (r *BucketRepository) MoveBucket(ctx context.Context, bucketID string, newParentID *string) error {
    bucket, err := r.GetByID(ctx, bucketID)
    if err != nil {
        return err
    }
    
    // Remove from old parent
    if bucket.ParentBucketID != nil {
        r.removeChildFromParent(ctx, *bucket.ParentBucketID, bucketID)
    }
    
    // Update bucket's parent
    update := bson.M{
        "$set": bson.M{
            "parent_bucket_id": newParentID,
            "updated_at": time.Now(),
        },
    }
    
    objectID, _ := primitive.ObjectIDFromHex(bucketID)
    _, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
    if err != nil {
        return fmt.Errorf("failed to update bucket parent: %w", err)
    }
    
    // Add to new parent
    if newParentID != nil {
        r.addChildToParent(ctx, *newParentID, bucketID)
    }
    
    return nil
}

// Add test file: internal/repository/bucket_repository_test.go
```

## Task 5: Update State Service with New Operations
**File**: `internal/service/state_service.go`
```go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/memmieai/memmie-state/internal/models"
    "github.com/memmieai/memmie-state/internal/repository"
    schemaClient "github.com/memmieai/memmie-schema/pkg/client"
    "github.com/memmieai/memmie-common/pkg/logger"
    "github.com/nats-io/nats.go"
)

type StateService struct {
    blobRepo     *repository.BlobRepository
    bucketRepo   *repository.BucketRepository
    userRepo     *repository.MongoRepository // Existing user state repo
    schemaClient *schemaClient.SchemaClient
    eventBus     *nats.Conn
    logger       logger.Logger
}

func NewStateService(
    blobRepo *repository.BlobRepository,
    bucketRepo *repository.BucketRepository,
    userRepo *repository.MongoRepository,
    schemaURL string,
    natsConn *nats.Conn,
    logger logger.Logger,
) *StateService {
    return &StateService{
        blobRepo:     blobRepo,
        bucketRepo:   bucketRepo,
        userRepo:     userRepo,
        schemaClient: schemaClient.NewSchemaClient(schemaURL),
        eventBus:     natsConn,
        logger:       logger,
    }
}

func (s *StateService) CreateBlob(ctx context.Context, userID string, req models.CreateBlobRequest) (*models.Blob, error) {
    // Validate against schema
    validationResult, err := s.schemaClient.ValidateData(ctx, req.SchemaID, req.Data)
    if err != nil {
        return nil, fmt.Errorf("schema validation failed: %w", err)
    }
    
    if !validationResult.Valid {
        return nil, fmt.Errorf("data validation failed: %v", validationResult.Errors)
    }
    
    // Calculate size
    dataBytes, _ := json.Marshal(req.Data)
    size := int64(len(dataBytes))
    
    // Extract title from data if available
    title := ""
    if dataMap, ok := req.Data.(map[string]interface{}); ok {
        if t, ok := dataMap["title"].(string); ok {
            title = t
        } else if content, ok := dataMap["content"].(string); ok && len(content) > 0 {
            // Use first line or 50 chars as title
            if len(content) > 50 {
                title = content[:50] + "..."
            } else {
                title = content
            }
        }
    }
    
    // Create preview (first 500 chars of content)
    preview := ""
    if dataMap, ok := req.Data.(map[string]interface{}); ok {
        if content, ok := dataMap["content"].(string); ok && len(content) > 0 {
            if len(content) > 500 {
                preview = content[:500] + "..."
            } else {
                preview = content
            }
        }
    }
    
    blob := &models.Blob{
        UserID:      userID,
        ProcessorID: req.ProcessorID,
        SchemaID:    req.SchemaID,
        Data:        req.Data,
        BucketIDs:   req.BucketIDs,
        ParentID:    req.ParentID,
        Title:       title,
        Preview:     preview,
        Size:        size,
    }
    
    // Extract tags from metadata
    if req.Metadata != nil {
        if tags, ok := req.Metadata["tags"].([]string); ok {
            blob.Tags = tags
        }
    }
    
    if err := s.blobRepo.Create(ctx, blob); err != nil {
        return nil, fmt.Errorf("failed to create blob: %w", err)
    }
    
    // Update parent blob if exists
    if req.ParentID != nil {
        s.updateParentDerivedIDs(ctx, *req.ParentID, blob.ID.Hex())
    }
    
    // Add to buckets
    for _, bucketID := range req.BucketIDs {
        s.bucketRepo.AddBlobToBucket(ctx, blob.ID.Hex(), bucketID)
    }
    
    // Publish event
    event := map[string]interface{}{
        "blob_id":      blob.ID.Hex(),
        "user_id":      userID,
        "processor_id": req.ProcessorID,
        "schema_id":    req.SchemaID,
        "bucket_ids":   req.BucketIDs,
        "parent_id":    req.ParentID,
    }
    
    eventData, _ := json.Marshal(event)
    s.eventBus.Publish(fmt.Sprintf("blob.created.%s", req.SchemaID), eventData)
    
    return blob, nil
}

func (s *StateService) CreateBucket(ctx context.Context, userID string, req models.CreateBucketRequest) (*models.Bucket, error) {
    // Validate bucket metadata against schema if type-specific schema exists
    if req.Metadata != nil {
        schemaID := fmt.Sprintf("bucket-metadata-%s-v1", req.Type)
        if result, err := s.schemaClient.ValidateData(ctx, schemaID, req.Metadata); err == nil {
            if !result.Valid {
                s.logger.Warn("Bucket metadata validation failed", "errors", result.Errors)
                // Don't fail, just warn - metadata schemas are optional
            }
        }
    }
    
    bucket := &models.Bucket{
        UserID:         userID,
        Name:           req.Name,
        Type:           req.Type,
        ParentBucketID: req.ParentBucketID,
        Description:    req.Description,
        Icon:           req.Icon,
        Color:          req.Color,
        Metadata:       req.Metadata,
    }
    
    if err := s.bucketRepo.Create(ctx, bucket); err != nil {
        return nil, fmt.Errorf("failed to create bucket: %w", err)
    }
    
    // Update user's root buckets if no parent
    if req.ParentBucketID == nil {
        s.updateUserRootBuckets(ctx, userID, bucket.ID.Hex())
    }
    
    // Publish event
    event := map[string]interface{}{
        "bucket_id": bucket.ID.Hex(),
        "user_id":   userID,
        "type":      req.Type,
        "parent_id": req.ParentBucketID,
    }
    
    eventData, _ := json.Marshal(event)
    s.eventBus.Publish("bucket.created", eventData)
    
    return bucket, nil
}

func (s *StateService) ExportBucket(ctx context.Context, userID, bucketID string, format string) ([]byte, error) {
    // Get bucket tree
    tree, err := s.bucketRepo.GetTree(ctx, bucketID)
    if err != nil {
        return nil, err
    }
    
    // Verify ownership
    if tree.Bucket.UserID != userID {
        return nil, fmt.Errorf("unauthorized")
    }
    
    switch format {
    case "text":
        return s.exportBucketAsText(ctx, tree)
    case "json":
        return s.exportBucketAsJSON(ctx, tree)
    default:
        return nil, fmt.Errorf("unsupported format: %s", format)
    }
}

func (s *StateService) exportBucketAsText(ctx context.Context, tree *models.BucketTree) ([]byte, error) {
    var output string
    
    // Add bucket header
    output += fmt.Sprintf("# %s\n\n", tree.Bucket.Name)
    if tree.Bucket.Description != "" {
        output += fmt.Sprintf("%s\n\n", tree.Bucket.Description)
    }
    
    // Get all blobs in this bucket
    blobs, err := s.blobRepo.GetByBucket(ctx, tree.Bucket.ID.Hex())
    if err != nil {
        return nil, err
    }
    
    // Add blob contents
    for _, blob := range blobs {
        if blob.Title != "" {
            output += fmt.Sprintf("## %s\n\n", blob.Title)
        }
        
        // Extract text content from blob data
        if dataMap, ok := blob.Data.(map[string]interface{}); ok {
            if content, ok := dataMap["content"].(string); ok {
                output += content + "\n\n"
            }
        }
    }
    
    // Recursively add children
    for _, child := range tree.Children {
        childText, err := s.exportBucketAsText(ctx, child)
        if err != nil {
            continue
        }
        output += string(childText)
    }
    
    return []byte(output), nil
}

// Add test file: internal/service/state_service_test.go
```

## Task 6: Create New HTTP Handlers for Blobs and Buckets
**File**: `internal/handler/blob_handler.go`
```go
package handler

import (
    "encoding/json"
    "net/http"
    
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-state/internal/models"
    "github.com/memmieai/memmie-state/internal/service"
)

type BlobHandler struct {
    service *service.StateService
    logger  logger.Logger
}

func NewBlobHandler(service *service.StateService, logger logger.Logger) *BlobHandler {
    return &BlobHandler{
        service: service,
        logger:  logger,
    }
}

func (h *BlobHandler) CreateBlob(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID := vars["user_id"]
    
    var req models.CreateBlobRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    blob, err := h.service.CreateBlob(r.Context(), userID, req)
    if err != nil {
        h.logger.Error("Failed to create blob", "error", err)
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    h.respondJSON(w, http.StatusCreated, blob)
}

func (h *BlobHandler) GetBlob(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID := vars["user_id"]
    blobID := vars["blob_id"]
    
    blob, err := h.service.GetBlob(r.Context(), blobID)
    if err != nil {
        h.respondError(w, http.StatusNotFound, "Blob not found")
        return
    }
    
    // Verify ownership
    if blob.UserID != userID {
        h.respondError(w, http.StatusForbidden, "Access denied")
        return
    }
    
    h.respondJSON(w, http.StatusOK, blob)
}

func (h *BlobHandler) ListBlobsByBucket(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID := vars["user_id"]
    bucketID := r.URL.Query().Get("bucket_id")
    
    if bucketID == "" {
        h.respondError(w, http.StatusBadRequest, "bucket_id required")
        return
    }
    
    // Verify bucket ownership
    bucket, err := h.service.GetBucket(r.Context(), bucketID)
    if err != nil || bucket.UserID != userID {
        h.respondError(w, http.StatusForbidden, "Access denied")
        return
    }
    
    blobs, err := h.service.GetBlobsByBucket(r.Context(), bucketID)
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, "Failed to list blobs")
        return
    }
    
    h.respondJSON(w, http.StatusOK, map[string]interface{}{
        "blobs": blobs,
        "count": len(blobs),
    })
}
```

**File**: `internal/handler/bucket_handler.go`
```go
package handler

import (
    "encoding/json"
    "net/http"
    
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-state/internal/models"
    "github.com/memmieai/memmie-state/internal/service"
)

type BucketHandler struct {
    service *service.StateService
    logger  logger.Logger
}

func NewBucketHandler(service *service.StateService, logger logger.Logger) *BucketHandler {
    return &BucketHandler{
        service: service,
        logger:  logger,
    }
}

func (h *BucketHandler) CreateBucket(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID := vars["user_id"]
    
    var req models.CreateBucketRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    bucket, err := h.service.CreateBucket(r.Context(), userID, req)
    if err != nil {
        h.logger.Error("Failed to create bucket", "error", err)
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    h.respondJSON(w, http.StatusCreated, bucket)
}

func (h *BucketHandler) GetBucketTree(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID := vars["user_id"]
    bucketID := vars["bucket_id"]
    
    tree, err := h.service.GetBucketTree(r.Context(), bucketID)
    if err != nil {
        h.respondError(w, http.StatusNotFound, "Bucket not found")
        return
    }
    
    // Verify ownership
    if tree.Bucket.UserID != userID {
        h.respondError(w, http.StatusForbidden, "Access denied")
        return
    }
    
    h.respondJSON(w, http.StatusOK, tree)
}

func (h *BucketHandler) ExportBucket(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID := vars["user_id"]
    bucketID := vars["bucket_id"]
    format := r.URL.Query().Get("format")
    
    if format == "" {
        format = "text"
    }
    
    data, err := h.service.ExportBucket(r.Context(), userID, bucketID, format)
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, "Export failed")
        return
    }
    
    // Set appropriate content type
    contentType := "text/plain"
    if format == "json" {
        contentType = "application/json"
    }
    
    w.Header().Set("Content-Type", contentType)
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-export.%s\"", bucketID, format))
    w.Write(data)
}
```

## Task 7: Update Routes
**File**: `internal/router/router.go` (update existing)
```go
func SetupRoutes(router *mux.Router, service *service.StateService, logger logger.Logger) {
    // Existing user state routes...
    
    // New blob routes
    blobHandler := handler.NewBlobHandler(service, logger)
    api.HandleFunc("/users/{user_id}/blobs", blobHandler.CreateBlob).Methods("POST")
    api.HandleFunc("/users/{user_id}/blobs/{blob_id}", blobHandler.GetBlob).Methods("GET")
    api.HandleFunc("/users/{user_id}/blobs", blobHandler.ListBlobsByBucket).Methods("GET")
    
    // New bucket routes
    bucketHandler := handler.NewBucketHandler(service, logger)
    api.HandleFunc("/users/{user_id}/buckets", bucketHandler.CreateBucket).Methods("POST")
    api.HandleFunc("/users/{user_id}/buckets/{bucket_id}", bucketHandler.GetBucket).Methods("GET")
    api.HandleFunc("/users/{user_id}/buckets/{bucket_id}/tree", bucketHandler.GetBucketTree).Methods("GET")
    api.HandleFunc("/users/{user_id}/buckets/{bucket_id}/export", bucketHandler.ExportBucket).Methods("GET")
    api.HandleFunc("/users/{user_id}/buckets/{bucket_id}/blobs", bucketHandler.AddBlobToBucket).Methods("POST")
}
```

## Task 8: Create Migration Strategy for Existing Data
**File**: `scripts/migrate_to_buckets.go`
```go
package main

import (
    "context"
    "log"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
)

func main() {
    // Connect to MongoDB
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
    if err != nil {
        log.Fatal(err)
    }
    
    db := client.Database("memmie_state")
    
    // Get all existing user states
    userStates := db.Collection("user_states")
    cursor, err := userStates.Find(context.Background(), bson.M{})
    if err != nil {
        log.Fatal(err)
    }
    
    for cursor.Next(context.Background()) {
        var state map[string]interface{}
        cursor.Decode(&state)
        
        userID := state["user_id"].(string)
        
        // Create a default bucket for existing content
        bucket := models.Bucket{
            UserID:    userID,
            Name:      "Imported Content",
            Type:      "archive",
            Metadata: map[string]interface{}{
                "imported_at": time.Now(),
                "source": "legacy_state",
            },
        }
        
        bucketRepo.Create(context.Background(), &bucket)
        
        // Convert existing state to blobs if needed
        if stateData, ok := state["state"].(map[string]interface{}); ok {
            for key, value := range stateData {
                blob := models.Blob{
                    UserID:      userID,
                    ProcessorID: "legacy",
                    SchemaID:    "legacy-data-v1",
                    Data:        map[string]interface{}{
                        "key": key,
                        "value": value,
                    },
                    BucketIDs: []string{bucket.ID.Hex()},
                }
                
                blobRepo.Create(context.Background(), &blob)
            }
        }
    }
    
    log.Println("Migration completed")
}
```

## Task 9: Create Integration Tests
**File**: `internal/service/integration_test.go`
```go
package service_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStateService_BlobBucketFlow(t *testing.T) {
    // Setup test database
    db := setupTestMongoDB(t)
    defer db.Disconnect(context.Background())
    
    // Initialize service
    svc := setupService(db)
    
    userID := "test-user-123"
    
    // Create a book bucket
    bookBucket, err := svc.CreateBucket(context.Background(), userID, models.CreateBucketRequest{
        Name: "My Novel",
        Type: "book",
        Metadata: map[string]interface{}{
            "genre": "fiction",
            "chapters": 10,
        },
    })
    require.NoError(t, err)
    assert.Equal(t, "book", bookBucket.Type)
    
    // Create a chapter bucket
    chapterBucket, err := svc.CreateBucket(context.Background(), userID, models.CreateBucketRequest{
        Name:           "Chapter 1",
        Type:           "chapter",
        ParentBucketID: &bookBucket.ID.Hex(),
    })
    require.NoError(t, err)
    
    // Create a text blob in the chapter
    blob, err := svc.CreateBlob(context.Background(), userID, models.CreateBlobRequest{
        ProcessorID: "user-input",
        SchemaID:    "text-input-v1",
        Data: map[string]interface{}{
            "content": "It was a dark and stormy night...",
            "style": "creative",
        },
        BucketIDs: []string{chapterBucket.ID.Hex()},
    })
    require.NoError(t, err)
    assert.NotEmpty(t, blob.ID)
    
    // Get bucket tree
    tree, err := svc.GetBucketTree(context.Background(), bookBucket.ID.Hex())
    require.NoError(t, err)
    assert.Len(t, tree.Children, 1)
    assert.Equal(t, "Chapter 1", tree.Children[0].Bucket.Name)
    
    // Export book as text
    exported, err := svc.ExportBucket(context.Background(), userID, bookBucket.ID.Hex(), "text")
    require.NoError(t, err)
    assert.Contains(t, string(exported), "My Novel")
    assert.Contains(t, string(exported), "dark and stormy night")
}
```

## Task 10: Update Docker Compose
**File**: `memmie-infra/docker-compose.yml` (update)
```yaml
  memmie-state:
    build: ../memmie-state
    ports:
      - "8006:8006"
    environment:
      - MONGO_URI=mongodb://mongo:27017/memmie_state
      - REDIS_URL=redis:6379
      - NATS_URL=nats://nats:4222
      - SCHEMA_SERVICE_URL=http://memmie-schema:8011
    depends_on:
      - mongodb
      - redis
      - nats
      - memmie-schema
```

## Testing Checklist
- [ ] Blob CRUD operations work
- [ ] Bucket CRUD operations work
- [ ] Schema validation integration works
- [ ] Event publishing to NATS works
- [ ] Bucket hierarchy navigation works
- [ ] Export functionality works
- [ ] Migration script tested
- [ ] Performance with 10,000 blobs
- [ ] Concurrent access handling

## Success Criteria
- [ ] Can create blobs with schema validation
- [ ] Can organize blobs in buckets
- [ ] Can export bucket as text/JSON
- [ ] Events published for all operations
- [ ] Backward compatible with existing state
- [ ] Handles 100 concurrent users