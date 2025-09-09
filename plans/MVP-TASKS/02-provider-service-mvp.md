# Task 02: Provider Service MVP Setup

## Objective
Set up the Provider Service with MongoDB storage for provider templates. This service will manage provider configurations and templates for the MVP (book and pitch providers only).

## Prerequisites
- MongoDB running on localhost:27017 with auth configured
- Go environment set up
- `/home/uneid/iter3/memmieai/memmie-provider` directory exists
- State Service (Task 01) completed and running

## Task Steps

### Step 1: Create Domain Models
Create file: `/home/uneid/iter3/memmieai/memmie-provider/internal/domain/provider.go`

```go
package domain

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderType string

const (
    ProviderTypeProcessor ProviderType = "processor"
    ProviderTypeNamespace ProviderType = "namespace"
)

type Provider struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    ProviderID  string            `bson:"provider_id" json:"provider_id"` // e.g., "book", "pitch"
    Name        string            `bson:"name" json:"name"`
    Type        ProviderType      `bson:"type" json:"type"`
    Description string            `bson:"description" json:"description"`
    Template    ProviderTemplate  `bson:"template" json:"template"`
    Active      bool              `bson:"active" json:"active"`
    CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time         `bson:"updated_at" json:"updated_at"`
}

type ProviderTemplate struct {
    InputPrompt   string                 `bson:"input_prompt" json:"input_prompt"`
    SystemPrompt  string                 `bson:"system_prompt" json:"system_prompt"`
    Model         string                 `bson:"model" json:"model"` // "gpt-4", "gpt-3.5-turbo"
    Temperature   float32                `bson:"temperature" json:"temperature"`
    MaxTokens     int                    `bson:"max_tokens" json:"max_tokens"`
    UITemplate    map[string]interface{} `bson:"ui_template" json:"ui_template"`
    OutputFormat  string                 `bson:"output_format" json:"output_format"` // "text", "markdown", "json"
}

type ProviderInstance struct {
    ID           primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
    UserID       string                 `bson:"user_id" json:"user_id"`
    ProviderID   string                 `bson:"provider_id" json:"provider_id"`
    InstanceName string                 `bson:"instance_name" json:"instance_name"` // e.g., "My Novel", "Q4 Business Plan"
    Metadata     map[string]interface{} `bson:"metadata" json:"metadata"`
    CreatedAt    time.Time              `bson:"created_at" json:"created_at"`
    UpdatedAt    time.Time              `bson:"updated_at" json:"updated_at"`
}
```

### Step 2: Create Repository Implementation
Create file: `/home/uneid/iter3/memmieai/memmie-provider/internal/repository/mongodb.go`

```go
package repository

import (
    "context"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    
    "github.com/memmieai/memmie-provider/internal/domain"
)

type MongoRepository struct {
    db                *mongo.Database
    providers         *mongo.Collection
    providerInstances *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
    return &MongoRepository{
        db:                db,
        providers:         db.Collection("providers"),
        providerInstances: db.Collection("provider_instances"),
    }
}

func (r *MongoRepository) GetProvider(ctx context.Context, providerID string) (*domain.Provider, error) {
    var provider domain.Provider
    err := r.providers.FindOne(ctx, bson.M{"provider_id": providerID}).Decode(&provider)
    if err != nil {
        return nil, err
    }
    return &provider, nil
}

func (r *MongoRepository) ListProviders(ctx context.Context) ([]*domain.Provider, error) {
    cursor, err := r.providers.Find(ctx, bson.M{"active": true})
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var providers []*domain.Provider
    if err := cursor.All(ctx, &providers); err != nil {
        return nil, err
    }
    
    return providers, nil
}

func (r *MongoRepository) CreateProviderInstance(ctx context.Context, instance *domain.ProviderInstance) error {
    instance.ID = primitive.NewObjectID()
    instance.CreatedAt = time.Now()
    instance.UpdatedAt = time.Now()
    
    _, err := r.providerInstances.InsertOne(ctx, instance)
    return err
}

func (r *MongoRepository) GetUserInstances(ctx context.Context, userID string) ([]*domain.ProviderInstance, error) {
    cursor, err := r.providerInstances.Find(ctx, bson.M{"user_id": userID})
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var instances []*domain.ProviderInstance
    if err := cursor.All(ctx, &instances); err != nil {
        return nil, err
    }
    
    return instances, nil
}

func (r *MongoRepository) GetUserInstancesByProvider(ctx context.Context, userID string, providerID string) ([]*domain.ProviderInstance, error) {
    cursor, err := r.providerInstances.Find(ctx, bson.M{
        "user_id": userID,
        "provider_id": providerID,
    })
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)
    
    var instances []*domain.ProviderInstance
    if err := cursor.All(ctx, &instances); err != nil {
        return nil, err
    }
    
    return instances, nil
}

// Seed initial providers for MVP
func (r *MongoRepository) SeedProviders(ctx context.Context) error {
    // Book writer provider
    bookProvider := domain.Provider{
        ID:          primitive.NewObjectID(),
        ProviderID:  "book",
        Name:        "Book Writer",
        Type:        domain.ProviderTypeNamespace,
        Description: "AI-powered book writing assistant",
        Template: domain.ProviderTemplate{
            SystemPrompt: "You are a creative writing assistant helping to expand and improve book content. Maintain the author's voice and style.",
            InputPrompt:  "Expand the following text with more descriptive details, character development, and engaging narrative: {{content}}",
            Model:        "gpt-4",
            Temperature:  0.7,
            MaxTokens:    2000,
            OutputFormat: "markdown",
            UITemplate: map[string]interface{}{
                "type": "split_view",
                "left_panel": map[string]interface{}{
                    "type": "editor",
                    "placeholder": "Start writing your chapter...",
                },
                "right_panel": map[string]interface{}{
                    "type": "preview",
                    "title": "AI Expanded Version",
                },
            },
        },
        Active:    true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    // Business pitch provider
    pitchProvider := domain.Provider{
        ID:          primitive.NewObjectID(),
        ProviderID:  "pitch",
        Name:        "Pitch Creator",
        Type:        domain.ProviderTypeNamespace,
        Description: "Business plan and pitch deck creator",
        Template: domain.ProviderTemplate{
            SystemPrompt: "You are a business strategy consultant helping to create compelling business plans and pitch decks. Focus on clarity, data, and persuasive narrative.",
            InputPrompt:  "Transform the following business ideas into a structured pitch with problem, solution, market, and strategy sections: {{content}}",
            Model:        "gpt-4",
            Temperature:  0.5,
            MaxTokens:    2500,
            OutputFormat: "markdown",
            UITemplate: map[string]interface{}{
                "type": "split_view",
                "left_panel": map[string]interface{}{
                    "type": "editor",
                    "placeholder": "Describe your business idea...",
                },
                "right_panel": map[string]interface{}{
                    "type": "preview",
                    "title": "Structured Pitch",
                },
            },
        },
        Active:    true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    // Clear existing providers
    r.providers.DeleteMany(ctx, bson.M{})
    
    // Insert new providers
    _, err := r.providers.InsertOne(ctx, bookProvider)
    if err != nil {
        return err
    }
    
    _, err = r.providers.InsertOne(ctx, pitchProvider)
    return err
}
```

### Step 3: Create Service Layer
Create file: `/home/uneid/iter3/memmieai/memmie-provider/internal/service/provider_service.go`

```go
package service

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "strings"
    
    "github.com/memmieai/memmie-provider/internal/domain"
    "github.com/memmieai/memmie-provider/internal/repository"
)

type ProviderService struct {
    repo   *repository.MongoRepository
    apiKey string
}

func NewProviderService(repo *repository.MongoRepository) *ProviderService {
    return &ProviderService{
        repo:   repo,
        apiKey: os.Getenv("OPENAI_API_KEY"),
    }
}

func (s *ProviderService) ListProviders(ctx context.Context) ([]*domain.Provider, error) {
    return s.repo.ListProviders(ctx)
}

func (s *ProviderService) GetProvider(ctx context.Context, providerID string) (*domain.Provider, error) {
    return s.repo.GetProvider(ctx, providerID)
}

func (s *ProviderService) CreateInstance(ctx context.Context, userID string, req CreateInstanceRequest) (*domain.ProviderInstance, error) {
    // Verify provider exists
    provider, err := s.repo.GetProvider(ctx, req.ProviderID)
    if err != nil {
        return nil, fmt.Errorf("provider not found: %w", err)
    }
    
    instance := &domain.ProviderInstance{
        UserID:       userID,
        ProviderID:   provider.ProviderID,
        InstanceName: req.Name,
        Metadata:     req.Metadata,
    }
    
    if err := s.repo.CreateProviderInstance(ctx, instance); err != nil {
        return nil, fmt.Errorf("failed to create instance: %w", err)
    }
    
    return instance, nil
}

func (s *ProviderService) GetUserInstances(ctx context.Context, userID string) ([]*domain.ProviderInstance, error) {
    return s.repo.GetUserInstances(ctx, userID)
}

func (s *ProviderService) ProcessContent(ctx context.Context, userID string, req ProcessRequest) (*ProcessResponse, error) {
    provider, err := s.repo.GetProvider(ctx, req.ProviderID)
    if err != nil {
        return nil, fmt.Errorf("provider not found: %w", err)
    }
    
    // Replace template variables
    prompt := strings.Replace(provider.Template.InputPrompt, "{{content}}", req.Content, -1)
    
    // Call OpenAI API
    response, err := s.callOpenAI(ctx, provider.Template, prompt)
    if err != nil {
        return nil, fmt.Errorf("failed to process content: %w", err)
    }
    
    return &ProcessResponse{
        Original: req.Content,
        Processed: response,
        ProviderID: req.ProviderID,
        Metadata: map[string]interface{}{
            "model": provider.Template.Model,
            "temperature": provider.Template.Temperature,
        },
    }, nil
}

func (s *ProviderService) callOpenAI(ctx context.Context, template domain.ProviderTemplate, prompt string) (string, error) {
    url := "https://api.openai.com/v1/chat/completions"
    
    messages := []map[string]string{
        {"role": "system", "content": template.SystemPrompt},
        {"role": "user", "content": prompt},
    }
    
    reqBody := map[string]interface{}{
        "model":       template.Model,
        "messages":    messages,
        "temperature": template.Temperature,
        "max_tokens":  template.MaxTokens,
    }
    
    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return "", err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
    if err != nil {
        return "", err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+s.apiKey)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result struct {
        Choices []struct {
            Message struct {
                Content string `json:"content"`
            } `json:"message"`
        } `json:"choices"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }
    
    if len(result.Choices) == 0 {
        return "", fmt.Errorf("no response from AI")
    }
    
    return result.Choices[0].Message.Content, nil
}

// DTOs
type CreateInstanceRequest struct {
    ProviderID string                 `json:"provider_id"`
    Name       string                 `json:"name"`
    Metadata   map[string]interface{} `json:"metadata"`
}

type ProcessRequest struct {
    ProviderID string `json:"provider_id"`
    Content    string `json:"content"`
}

type ProcessResponse struct {
    Original   string                 `json:"original"`
    Processed  string                 `json:"processed"`
    ProviderID string                 `json:"provider_id"`
    Metadata   map[string]interface{} `json:"metadata"`
}
```

### Step 4: Create HTTP Handlers
Create file: `/home/uneid/iter3/memmieai/memmie-provider/internal/handler/http.go`

```go
package handler

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/memmieai/memmie-provider/internal/service"
)

type HTTPHandler struct {
    service *service.ProviderService
}

func NewHTTPHandler(service *service.ProviderService) *HTTPHandler {
    return &HTTPHandler{
        service: service,
    }
}

func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
    api := router.Group("/api/v1")
    {
        // Provider endpoints
        api.GET("/providers", h.ListProviders)
        api.GET("/providers/:provider_id", h.GetProvider)
        
        // User instance endpoints
        api.POST("/users/:user_id/instances", h.CreateInstance)
        api.GET("/users/:user_id/instances", h.GetUserInstances)
        
        // Processing endpoint
        api.POST("/users/:user_id/process", h.ProcessContent)
    }
}

func (h *HTTPHandler) ListProviders(c *gin.Context) {
    providers, err := h.service.ListProviders(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (h *HTTPHandler) GetProvider(c *gin.Context) {
    providerID := c.Param("provider_id")
    
    provider, err := h.service.GetProvider(c.Request.Context(), providerID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
        return
    }
    
    c.JSON(http.StatusOK, provider)
}

func (h *HTTPHandler) CreateInstance(c *gin.Context) {
    userID := c.Param("user_id")
    
    var req service.CreateInstanceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    instance, err := h.service.CreateInstance(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, instance)
}

func (h *HTTPHandler) GetUserInstances(c *gin.Context) {
    userID := c.Param("user_id")
    
    instances, err := h.service.GetUserInstances(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"instances": instances})
}

func (h *HTTPHandler) ProcessContent(c *gin.Context) {
    userID := c.Param("user_id")
    
    var req service.ProcessRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    response, err := h.service.ProcessContent(c.Request.Context(), userID, req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, response)
}
```

### Step 5: Create Main Server
Create file: `/home/uneid/iter3/memmieai/memmie-provider/cmd/server/main.go`

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
    
    "github.com/memmieai/memmie-provider/internal/handler"
    "github.com/memmieai/memmie-provider/internal/repository"
    "github.com/memmieai/memmie-provider/internal/service"
)

func main() {
    // Connect to MongoDB
    mongoURI := os.Getenv("MONGO_URI")
    if mongoURI == "" {
        mongoURI = "mongodb://memmie:memmiepass@localhost:27017/memmie_provider?authSource=admin"
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
    
    db := client.Database("memmie_provider")
    
    // Initialize layers
    repo := repository.NewMongoRepository(db)
    
    // Seed providers for MVP
    if err := repo.SeedProviders(ctx); err != nil {
        log.Printf("Warning: Failed to seed providers: %v", err)
    }
    
    svc := service.NewProviderService(repo)
    handler := handler.NewHTTPHandler(svc)
    
    // Setup router
    router := gin.Default()
    
    // Enable CORS for frontend
    router.Use(func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    })
    
    handler.RegisterRoutes(router)
    
    // Health check
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8007"
    }
    
    log.Printf("Provider Service starting on port %s", port)
    if err := router.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
```

### Step 6: Create go.mod
Create file: `/home/uneid/iter3/memmieai/memmie-provider/go.mod`

```go
module github.com/memmieai/memmie-provider

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    go.mongodb.org/mongo-driver v1.12.1
)
```

### Step 7: Create .env file
Create file: `/home/uneid/iter3/memmieai/memmie-provider/.env`

```bash
MONGO_URI=mongodb://memmie:memmiepass@localhost:27017/memmie_provider?authSource=admin
PORT=8007
OPENAI_API_KEY=your-openai-api-key-here
```

### Step 8: Test the Service

```bash
# Terminal 1: Start the service
cd /home/uneid/iter3/memmieai/memmie-provider
go mod tidy
export OPENAI_API_KEY="your-api-key"
go run cmd/server/main.go

# Terminal 2: Test endpoints
# List available providers
curl http://localhost:8007/api/v1/providers

# Get specific provider
curl http://localhost:8007/api/v1/providers/book

# Create provider instance for user
curl -X POST http://localhost:8007/api/v1/users/test-user/instances \
  -H "Content-Type: application/json" \
  -d '{
    "provider_id": "book",
    "name": "My Science Fiction Novel",
    "metadata": {
      "genre": "sci-fi",
      "target_words": 80000
    }
  }'

# Process content (requires valid OpenAI API key)
curl -X POST http://localhost:8007/api/v1/users/test-user/process \
  -H "Content-Type: application/json" \
  -d '{
    "provider_id": "book",
    "content": "The spaceship landed on the alien planet."
  }'

# Health check
curl http://localhost:8007/health
```

## Expected Output
- Service starts on port 8007
- Lists two providers: "book" and "pitch"
- Can create provider instances for users
- Can process content through OpenAI API
- Returns expanded/processed text
- Health endpoint returns `{"status":"healthy"}`

## Success Criteria
✅ Service compiles and runs without errors
✅ MongoDB connection established
✅ Providers seeded successfully (book and pitch)
✅ Can list available providers
✅ Can create provider instances for users
✅ Can process content (with valid OpenAI key)
✅ Health check returns 200 OK

## Notes
- The OpenAI API key is required for the process endpoint to work
- The service seeds two hardcoded providers on startup
- Templates are configured for GPT-4 but can be changed to GPT-3.5-turbo for cost savings
- CORS is enabled for frontend integration