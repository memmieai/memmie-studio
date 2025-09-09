# Task 03: Studio API Service Setup

## Objective
Set up the Studio API service that orchestrates State and Provider services, integrates with Auth service for authentication, and serves the React frontend.

## Prerequisites
- State Service (Task 01) running on port 8006
- Provider Service (Task 02) running on port 8007
- Auth Service running on port 8001
- Go environment set up
- `/home/uneid/iter3/memmieai/memmie-studio` directory exists

## Task Steps

### Step 1: Create Service Clients
Create file: `/home/uneid/iter3/memmieai/memmie-studio/internal/clients/auth_client.go`

```go
package clients

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type AuthClient struct {
    baseURL string
    client  *http.Client
}

func NewAuthClient(baseURL string) *AuthClient {
    return &AuthClient{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

type ValidateTokenResponse struct {
    Valid  bool   `json:"valid"`
    UserID string `json:"user_id"`
    Email  string `json:"email"`
}

func (c *AuthClient) ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
    url := fmt.Sprintf("%s/api/v1/auth/validate", c.baseURL)
    
    reqBody := map[string]string{"token": token}
    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("invalid token")
    }
    
    var result ValidateTokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}
```

Create file: `/home/uneid/iter3/memmieai/memmie-studio/internal/clients/state_client.go`

```go
package clients

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type StateClient struct {
    baseURL string
    client  *http.Client
}

func NewStateClient(baseURL string) *StateClient {
    return &StateClient{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

type Blob struct {
    ID         string                 `json:"id"`
    UserID     string                 `json:"user_id"`
    ProviderID string                 `json:"provider_id"`
    Content    string                 `json:"content"`
    ParentID   *string                `json:"parent_id,omitempty"`
    Metadata   map[string]interface{} `json:"metadata"`
    CreatedAt  string                 `json:"created_at"`
    UpdatedAt  string                 `json:"updated_at"`
}

func (c *StateClient) CreateBlob(ctx context.Context, userID string, blob map[string]interface{}) (*Blob, error) {
    url := fmt.Sprintf("%s/api/v1/users/%s/blobs", c.baseURL, userID)
    
    jsonBody, err := json.Marshal(blob)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result Blob
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

func (c *StateClient) GetUserBlobs(ctx context.Context, userID string, providerID string) ([]Blob, error) {
    url := fmt.Sprintf("%s/api/v1/users/%s/blobs", c.baseURL, userID)
    if providerID != "" {
        url += "?provider_id=" + providerID
    }
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Blobs []Blob `json:"blobs"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Blobs, nil
}

func (c *StateClient) UpdateBlob(ctx context.Context, userID string, blobID string, content string) error {
    url := fmt.Sprintf("%s/api/v1/users/%s/blobs/%s", c.baseURL, userID, blobID)
    
    reqBody := map[string]string{"content": content}
    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonBody))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

Create file: `/home/uneid/iter3/memmieai/memmie-studio/internal/clients/provider_client.go`

```go
package clients

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type ProviderClient struct {
    baseURL string
    client  *http.Client
}

func NewProviderClient(baseURL string) *ProviderClient {
    return &ProviderClient{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

type Provider struct {
    ID          string                 `json:"id"`
    ProviderID  string                 `json:"provider_id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Template    map[string]interface{} `json:"template"`
}

type ProcessResponse struct {
    Original   string                 `json:"original"`
    Processed  string                 `json:"processed"`
    ProviderID string                 `json:"provider_id"`
    Metadata   map[string]interface{} `json:"metadata"`
}

func (c *ProviderClient) ListProviders(ctx context.Context) ([]Provider, error) {
    url := fmt.Sprintf("%s/api/v1/providers", c.baseURL)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Providers []Provider `json:"providers"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Providers, nil
}

func (c *ProviderClient) ProcessContent(ctx context.Context, userID string, providerID string, content string) (*ProcessResponse, error) {
    url := fmt.Sprintf("%s/api/v1/users/%s/process", c.baseURL, userID)
    
    reqBody := map[string]string{
        "provider_id": providerID,
        "content":     content,
    }
    jsonBody, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result ProcessResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}
```

### Step 2: Create Middleware
Create file: `/home/uneid/iter3/memmieai/memmie-studio/internal/middleware/auth.go`

```go
package middleware

import (
    "net/http"
    "strings"
    
    "github.com/gin-gonic/gin"
    "github.com/memmieai/memmie-studio/internal/clients"
)

type AuthMiddleware struct {
    authClient *clients.AuthClient
}

func NewAuthMiddleware(authClient *clients.AuthClient) *AuthMiddleware {
    return &AuthMiddleware{
        authClient: authClient,
    }
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
            c.Abort()
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
            c.Abort()
            return
        }
        
        token := parts[1]
        
        // Validate token with auth service
        resp, err := m.authClient.ValidateToken(c.Request.Context(), token)
        if err != nil || !resp.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }
        
        // Store user info in context
        c.Set("user_id", resp.UserID)
        c.Set("email", resp.Email)
        c.Next()
    }
}
```

### Step 3: Create Studio Service
Create file: `/home/uneid/iter3/memmieai/memmie-studio/internal/service/studio_service.go`

```go
package service

import (
    "context"
    "fmt"
    
    "github.com/memmieai/memmie-studio/internal/clients"
)

type StudioService struct {
    stateClient    *clients.StateClient
    providerClient *clients.ProviderClient
}

func NewStudioService(stateClient *clients.StateClient, providerClient *clients.ProviderClient) *StudioService {
    return &StudioService{
        stateClient:    stateClient,
        providerClient: providerClient,
    }
}

// CreateDocument creates a new document (blob) with optional AI processing
func (s *StudioService) CreateDocument(ctx context.Context, userID string, req CreateDocumentRequest) (*DocumentResponse, error) {
    // Create the initial blob
    blobData := map[string]interface{}{
        "provider_id": req.ProviderID,
        "content":     req.Content,
        "metadata":    req.Metadata,
    }
    
    blob, err := s.stateClient.CreateBlob(ctx, userID, blobData)
    if err != nil {
        return nil, fmt.Errorf("failed to create blob: %w", err)
    }
    
    response := &DocumentResponse{
        ID:         blob.ID,
        ProviderID: blob.ProviderID,
        Original:   blob.Content,
        Metadata:   blob.Metadata,
        CreatedAt:  blob.CreatedAt,
    }
    
    // If requested, process through AI
    if req.ProcessContent {
        processed, err := s.providerClient.ProcessContent(ctx, userID, req.ProviderID, req.Content)
        if err != nil {
            // Log error but don't fail the request
            fmt.Printf("Warning: Failed to process content: %v\n", err)
        } else {
            response.Processed = &processed.Processed
            
            // Save processed content as a child blob
            processedBlob := map[string]interface{}{
                "provider_id": req.ProviderID,
                "content":     processed.Processed,
                "parent_id":   blob.ID,
                "metadata": map[string]interface{}{
                    "type":        "processed",
                    "parent_id":   blob.ID,
                    "model":       processed.Metadata["model"],
                    "temperature": processed.Metadata["temperature"],
                },
            }
            
            childBlob, err := s.stateClient.CreateBlob(ctx, userID, processedBlob)
            if err == nil {
                response.ProcessedID = &childBlob.ID
            }
        }
    }
    
    return response, nil
}

// ListDocuments lists all documents for a user, optionally filtered by provider
func (s *StudioService) ListDocuments(ctx context.Context, userID string, providerID string) (*ListDocumentsResponse, error) {
    blobs, err := s.stateClient.GetUserBlobs(ctx, userID, providerID)
    if err != nil {
        return nil, fmt.Errorf("failed to get blobs: %w", err)
    }
    
    documents := make([]DocumentSummary, 0, len(blobs))
    for _, blob := range blobs {
        // Skip processed versions (they have parent_id)
        if blob.ParentID != nil {
            continue
        }
        
        doc := DocumentSummary{
            ID:         blob.ID,
            ProviderID: blob.ProviderID,
            Title:      s.extractTitle(blob),
            Preview:    s.extractPreview(blob.Content),
            Metadata:   blob.Metadata,
            CreatedAt:  blob.CreatedAt,
            UpdatedAt:  blob.UpdatedAt,
        }
        documents = append(documents, doc)
    }
    
    return &ListDocumentsResponse{
        Documents: documents,
        Total:     len(documents),
    }, nil
}

// GetProviders returns available providers
func (s *StudioService) GetProviders(ctx context.Context) (*ProvidersResponse, error) {
    providers, err := s.providerClient.ListProviders(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get providers: %w", err)
    }
    
    return &ProvidersResponse{
        Providers: providers,
    }, nil
}

// Helper methods
func (s *StudioService) extractTitle(blob clients.Blob) string {
    if title, ok := blob.Metadata["title"].(string); ok {
        return title
    }
    if blob.Content != "" && len(blob.Content) > 0 {
        // Use first line as title
        for i, ch := range blob.Content {
            if ch == '\n' || i > 50 {
                return blob.Content[:i]
            }
        }
    }
    return "Untitled"
}

func (s *StudioService) extractPreview(content string) string {
    const maxPreview = 150
    if len(content) <= maxPreview {
        return content
    }
    return content[:maxPreview] + "..."
}

// DTOs
type CreateDocumentRequest struct {
    ProviderID     string                 `json:"provider_id"`
    Content        string                 `json:"content"`
    ProcessContent bool                   `json:"process_content"`
    Metadata       map[string]interface{} `json:"metadata"`
}

type DocumentResponse struct {
    ID          string                 `json:"id"`
    ProviderID  string                 `json:"provider_id"`
    Original    string                 `json:"original"`
    Processed   *string                `json:"processed,omitempty"`
    ProcessedID *string                `json:"processed_id,omitempty"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   string                 `json:"created_at"`
}

type DocumentSummary struct {
    ID         string                 `json:"id"`
    ProviderID string                 `json:"provider_id"`
    Title      string                 `json:"title"`
    Preview    string                 `json:"preview"`
    Metadata   map[string]interface{} `json:"metadata"`
    CreatedAt  string                 `json:"created_at"`
    UpdatedAt  string                 `json:"updated_at"`
}

type ListDocumentsResponse struct {
    Documents []DocumentSummary `json:"documents"`
    Total     int               `json:"total"`
}

type ProvidersResponse struct {
    Providers []clients.Provider `json:"providers"`
}
```

### Step 4: Create HTTP Handlers
Create file: `/home/uneid/iter3/memmieai/memmie-studio/internal/handler/http.go`

```go
package handler

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/memmieai/memmie-studio/internal/service"
)

type HTTPHandler struct {
    service *service.StudioService
}

func NewHTTPHandler(service *service.StudioService) *HTTPHandler {
    return &HTTPHandler{
        service: service,
    }
}

func (h *HTTPHandler) RegisterRoutes(router *gin.RouterGroup) {
    // Document endpoints (authenticated)
    router.POST("/documents", h.CreateDocument)
    router.GET("/documents", h.ListDocuments)
    
    // Provider endpoints (authenticated)
    router.GET("/providers", h.GetProviders)
}

func (h *HTTPHandler) CreateDocument(c *gin.Context) {
    userID, _ := c.Get("user_id")
    
    var req service.CreateDocumentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    doc, err := h.service.CreateDocument(c.Request.Context(), userID.(string), req)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, doc)
}

func (h *HTTPHandler) ListDocuments(c *gin.Context) {
    userID, _ := c.Get("user_id")
    providerID := c.Query("provider_id")
    
    docs, err := h.service.ListDocuments(c.Request.Context(), userID.(string), providerID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, docs)
}

func (h *HTTPHandler) GetProviders(c *gin.Context) {
    providers, err := h.service.GetProviders(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, providers)
}
```

### Step 5: Create Main Server
Create file: `/home/uneid/iter3/memmieai/memmie-studio/cmd/server/main.go`

```go
package main

import (
    "log"
    "os"
    
    "github.com/gin-gonic/gin"
    "github.com/memmieai/memmie-studio/internal/clients"
    "github.com/memmieai/memmie-studio/internal/handler"
    "github.com/memmieai/memmie-studio/internal/middleware"
    "github.com/memmieai/memmie-studio/internal/service"
)

func main() {
    // Service URLs
    authURL := os.Getenv("AUTH_SERVICE_URL")
    if authURL == "" {
        authURL = "http://localhost:8001"
    }
    
    stateURL := os.Getenv("STATE_SERVICE_URL")
    if stateURL == "" {
        stateURL = "http://localhost:8006"
    }
    
    providerURL := os.Getenv("PROVIDER_SERVICE_URL")
    if providerURL == "" {
        providerURL = "http://localhost:8007"
    }
    
    // Initialize clients
    authClient := clients.NewAuthClient(authURL)
    stateClient := clients.NewStateClient(stateURL)
    providerClient := clients.NewProviderClient(providerURL)
    
    // Initialize service and handler
    studioService := service.NewStudioService(stateClient, providerClient)
    httpHandler := handler.NewHTTPHandler(studioService)
    
    // Initialize middleware
    authMiddleware := middleware.NewAuthMiddleware(authClient)
    
    // Setup router
    router := gin.Default()
    
    // Enable CORS for frontend
    router.Use(func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    })
    
    // Serve static files for React app
    router.Static("/static", "./web/build/static")
    router.StaticFile("/", "./web/build/index.html")
    router.StaticFile("/favicon.ico", "./web/build/favicon.ico")
    router.StaticFile("/manifest.json", "./web/build/manifest.json")
    
    // API routes
    api := router.Group("/api/v1")
    {
        // Public endpoints
        api.GET("/health", func(c *gin.Context) {
            c.JSON(200, gin.H{
                "status": "healthy",
                "services": gin.H{
                    "auth":     authURL,
                    "state":    stateURL,
                    "provider": providerURL,
                },
            })
        })
        
        // Protected endpoints
        protected := api.Group("/")
        protected.Use(authMiddleware.RequireAuth())
        httpHandler.RegisterRoutes(protected)
    }
    
    // Catch-all route for React app (must be last)
    router.NoRoute(func(c *gin.Context) {
        c.File("./web/build/index.html")
    })
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8010"
    }
    
    log.Printf("Studio API Service starting on port %s", port)
    log.Printf("Auth Service: %s", authURL)
    log.Printf("State Service: %s", stateURL)
    log.Printf("Provider Service: %s", providerURL)
    
    if err := router.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}
```

### Step 6: Create go.mod
Create file: `/home/uneid/iter3/memmieai/memmie-studio/go.mod`

```go
module github.com/memmieai/memmie-studio

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
)
```

### Step 7: Create .env file
Create file: `/home/uneid/iter3/memmieai/memmie-studio/.env`

```bash
PORT=8010
AUTH_SERVICE_URL=http://localhost:8001
STATE_SERVICE_URL=http://localhost:8006
PROVIDER_SERVICE_URL=http://localhost:8007
```

### Step 8: Test the Service

```bash
# Terminal 1: Start all required services
# Make sure State (8006), Provider (8007), and Auth (8001) are running

# Terminal 2: Start Studio API
cd /home/uneid/iter3/memmieai/memmie-studio
go mod tidy
go run cmd/server/main.go

# Terminal 3: Test endpoints
# Health check (public)
curl http://localhost:8010/api/v1/health

# Get auth token first (assuming auth service is running)
TOKEN=$(curl -X POST http://localhost:8001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}' \
  | jq -r .token)

# List providers (authenticated)
curl http://localhost:8010/api/v1/providers \
  -H "Authorization: Bearer $TOKEN"

# Create a document
curl -X POST http://localhost:8010/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "provider_id": "book",
    "content": "Chapter 1: The Beginning",
    "process_content": true,
    "metadata": {
      "title": "My Novel",
      "chapter": 1
    }
  }'

# List documents
curl http://localhost:8010/api/v1/documents \
  -H "Authorization: Bearer $TOKEN"

# List documents by provider
curl "http://localhost:8010/api/v1/documents?provider_id=book" \
  -H "Authorization: Bearer $TOKEN"
```

## Expected Output
- Service starts on port 8010
- Health check shows connected services
- Can authenticate via Auth service
- Can list available providers
- Can create documents with optional AI processing
- Can list user's documents
- Frontend files served at root URL

## Success Criteria
✅ Service compiles and runs without errors
✅ Connects to Auth, State, and Provider services
✅ Auth middleware validates tokens correctly
✅ Can create documents (blobs) via orchestration
✅ Can process content through Provider service
✅ Can list user documents from State service
✅ Health check returns 200 OK with service status
✅ CORS configured for frontend integration

## Notes
- Auth service must be running for authentication to work
- State and Provider services must be running for full functionality
- Frontend will be served from `/web/build` directory (created in next task)
- All API endpoints except health require authentication
- The service orchestrates calls to multiple backend services