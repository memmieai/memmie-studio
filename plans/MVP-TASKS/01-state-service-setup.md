# Task 01: State Service MongoDB Setup

## Objective
Set up the State Service with minimal MongoDB blob storage for MVP. This service will store user content blobs.

## Prerequisites
- MongoDB running on localhost:27017
- Go environment set up
- `/home/uneid/iter3/memmieai/memmie-state` directory exists

## Task Steps

### Step 1: Create Domain Models
Create file: `/home/uneid/iter3/memmieai/memmie-state/internal/domain/blob.go`

```go
package domain

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type Blob struct {
    ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    UserID     string            `bson:"user_id" json:"user_id"`
    ProviderID string            `bson:"provider_id" json:"provider_id"` // "book" or "pitch"
    Content    string            `bson:"content" json:"content"`
    ParentID   *string           `bson:"parent_id,omitempty" json:"parent_id,omitempty"`
    Metadata   BlobMetadata      `bson:"metadata" json:"metadata"`
    CreatedAt  time.Time         `bson:"created_at" json:"created_at"`
    UpdatedAt  time.Time         `bson:"updated_at" json:"updated_at"`
}

type BlobMetadata struct {
    Title   string `bson:"title,omitempty" json:"title,omitempty"`
    Chapter int    `bson:"chapter,omitempty" json:"chapter,omitempty"` // for book
    Section string `bson:"section,omitempty" json:"section,omitempty"` // for pitch
}
```

### Step 2: Create Repository Implementation
Create file: `/home/uneid/iter3/memmieai/memmie-state/internal/repository/mongodb.go`

```go
package repository

import (
    "context"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    
    "github.com/memmieai/memmie-state/internal/domain"
)

type MongoRepository struct {
    db         *mongo.Database
    collection *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
    return &MongoRepository{
        db:         db,
        collection: db.Collection("blobs"),
    }
}

func (r *MongoRepository) CreateBlob(ctx context.Context, blob *domain.Blob) error {
    blob.ID = primitive.NewObjectID()
    blob.CreatedAt = time.Now()
    blob.UpdatedAt = time.Now()
    
    _, err := r.collection.InsertOne(ctx, blob)
    return err
}

func (r *MongoRepository) GetBlobsByUser(ctx context.Context, userID string, providerID string) ([]*domain.Blob, error) {
    filter := bson.M{
        "user_id": userID,
    }
    if providerID != "" {
        filter["provider_id"] = providerID
    }
    
    cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{"created_at", -1}}))
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var blobs []*domain.Blob
    if err := cursor.All(ctx, &blobs); err != nil {
        return nil, err
    }
    
    return blobs, nil
}

func (r *MongoRepository) GetBlob(ctx context.Context, userID string, blobID string) (*domain.Blob, error) {
    objID, err := primitive.ObjectIDFromHex(blobID)
    if err != nil {
        return nil, err
    }
    
    var blob domain.Blob
    err = r.collection.FindOne(ctx, bson.M{
        "_id":     objID,
        "user_id": userID,
    }).Decode(&blob)
    
    if err != nil {
        return nil, err
    }
    
    return &blob, nil
}

func (r *MongoRepository) UpdateBlob(ctx context.Context, userID string, blobID string, content string) error {
    objID, err := primitive.ObjectIDFromHex(blobID)
    if err != nil {
        return err
    }
    
    _, err = r.collection.UpdateOne(
        ctx,
        bson.M{
            "_id":     objID,
            "user_id": userID,
        },
        bson.M{
            "$set": bson.M{
                "content":    content,
                "updated_at": time.Now(),
            },
        },
    )
    
    return err
}
```

### Step 3: Create Service Layer
Create file: `/home/uneid/iter3/memmieai/memmie-state/internal/service/state_service.go`

```go
package service

import (
    "context"
    "fmt"
    
    "github.com/memmieai/memmie-state/internal/domain"
    "github.com/memmieai/memmie-state/internal/repository"
)

type StateService struct {
    repo *repository.MongoRepository
}

func NewStateService(repo *repository.MongoRepository) *StateService {
    return &StateService{
        repo: repo,
    }
}

func (s *StateService) CreateBlob(ctx context.Context, userID string, req CreateBlobRequest) (*domain.Blob, error) {
    blob := &domain.Blob{
        UserID:     userID,
        ProviderID: req.ProviderID,
        Content:    req.Content,
        ParentID:   req.ParentID,
        Metadata:   req.Metadata,
    }
    
    if err := s.repo.CreateBlob(ctx, blob); err != nil {
        return nil, fmt.Errorf("failed to create blob: %w", err)
    }
    
    return blob, nil
}

func (s *StateService) GetUserBlobs(ctx context.Context, userID string, providerID string) ([]*domain.Blob, error) {
    return s.repo.GetBlobsByUser(ctx, userID, providerID)
}

func (s *StateService) GetBlob(ctx context.Context, userID string, blobID string) (*domain.Blob, error) {
    return s.repo.GetBlob(ctx, userID, blobID)
}

func (s *StateService) UpdateBlob(ctx context.Context, userID string, blobID string, content string) error {
    return s.repo.UpdateBlob(ctx, userID, blobID, content)
}

// DTOs
type CreateBlobRequest struct {
    ProviderID string               `json:"provider_id"`
    Content    string               `json:"content"`
    ParentID   *string              `json:"parent_id,omitempty"`
    Metadata   domain.BlobMetadata  `json:"metadata"`
}
```

### Step 4: Create HTTP Handlers
Create file: `/home/uneid/iter3/memmieai/memmie-state/internal/handler/http.go`

```go
package handler

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/memmieai/memmie-state/internal/service"
)

type HTTPHandler struct {
    service *service.StateService
}

func NewHTTPHandler(service *service.StateService) *HTTPHandler {
    return &HTTPHandler{
        service: service,
    }
}

func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
    api := router.Group("/api/v1")
    {
        // User scoped endpoints
        api.POST("/users/:user_id/blobs", h.CreateBlob)
        api.GET("/users/:user_id/blobs", h.GetUserBlobs)
        api.GET("/users/:user_id/blobs/:blob_id", h.GetBlob)
        api.PUT("/users/:user_id/blobs/:blob_id", h.UpdateBlob)
    }
}

func (h *HTTPHandler) CreateBlob(c *gin.Context) {
    userID := c.Param("user_id")
    
    var req service.CreateBlobRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    blob, err := h.service.CreateBlob(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, blob)
}

func (h *HTTPHandler) GetUserBlobs(c *gin.Context) {
    userID := c.Param("user_id")
    providerID := c.Query("provider_id")
    
    blobs, err := h.service.GetUserBlobs(c.Request.Context(), userID, providerID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"blobs": blobs})
}

func (h *HTTPHandler) GetBlob(c *gin.Context) {
    userID := c.Param("user_id")
    blobID := c.Param("blob_id")
    
    blob, err := h.service.GetBlob(c.Request.Context(), userID, blobID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "blob not found"})
        return
    }
    
    c.JSON(http.StatusOK, blob)
}

func (h *HTTPHandler) UpdateBlob(c *gin.Context) {
    userID := c.Param("user_id")
    blobID := c.Param("blob_id")
    
    var req struct {
        Content string `json:"content"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    if err := h.service.UpdateBlob(c.Request.Context(), userID, blobID, req.Content); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"success": true})
}
```

### Step 5: Create Main Server
Create file: `/home/uneid/iter3/memmieai/memmie-state/cmd/server/main.go`

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    
    "github.com/memmieai/memmie-state/internal/handler"
    "github.com/memmieai/memmie-state/internal/repository"
    "github.com/memmieai/memmie-state/internal/service"
)

func main() {
    // Connect to MongoDB
    mongoURI := os.Getenv("MONGO_URI")
    if mongoURI == "" {
        mongoURI = "mongodb://memmie:memmiepass@localhost:27017/memmie_state?authSource=admin"
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    if err != nil {
        log.Fatal("Failed to connect to MongoDB:", err)
    }
    
    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal("Failed to ping MongoDB:", err)
    }
    
    db := client.Database("memmie_state")
    
    // Initialize layers
    repo := repository.NewMongoRepository(db)
    svc := service.NewStateService(repo)
    handler := handler.NewHTTPHandler(svc)
    
    // Setup router
    router := gin.Default()
    handler.RegisterRoutes(router)
    
    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8006"
    }
    
    log.Printf("State Service starting on port %s", port)
    if err := router.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
```

### Step 6: Create go.mod
Create file: `/home/uneid/iter3/memmieai/memmie-state/go.mod`

```go
module github.com/memmieai/memmie-state

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    go.mongodb.org/mongo-driver v1.12.1
)
```

### Step 7: Test the Service

```bash
# Terminal 1: Start the service
cd /home/uneid/iter3/memmieai/memmie-state
go mod tidy
go run cmd/server/main.go

# Terminal 2: Test endpoints
# Create a blob
curl -X POST http://localhost:8006/api/v1/users/test-user/blobs \
  -H "Content-Type: application/json" \
  -d '{
    "provider_id": "book",
    "content": "Chapter 1: The Beginning",
    "metadata": {
      "title": "My First Book",
      "chapter": 1
    }
  }'

# Get user blobs
curl http://localhost:8006/api/v1/users/test-user/blobs?provider_id=book

# Health check
curl http://localhost:8006/health
```

## Expected Output
- Service starts on port 8006
- Can create blobs with POST request
- Can retrieve blobs with GET request
- Health endpoint returns `{"status":"healthy"}`

## Success Criteria
✅ Service compiles and runs without errors
✅ MongoDB connection established
✅ Can create a blob and get a valid ID back
✅ Can retrieve created blobs by user ID
✅ Health check returns 200 OK