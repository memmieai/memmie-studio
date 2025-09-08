# Studio API Design - Orchestration Layer

## Overview

The Studio Service (Port 8010) acts as an orchestration layer that:
1. Serves the React frontend application
2. Aggregates data from backend services (State, Provider, Workflow)
3. Constructs optimized DTOs for client consumption  
4. Manages WebSocket connections for real-time updates
5. Handles authentication and session management

## Core Architecture

```go
// Studio Service does NOT proxy requests directly
// Instead, it orchestrates multiple service calls and builds responses

type StudioService struct {
    // Service Clients
    stateClient    *stateclient.Client
    providerClient *providerclient.Client
    workflowClient *workflowclient.Client
    authClient     *authclient.Client
    coreClient     *coreclient.Client
    mediaClient    *mediaclient.Client
    
    // WebSocket Manager
    wsHub          *WebSocketHub
    
    // Event Subscription
    natsConn       *nats.Conn
    
    // Cache
    cache          *redis.Client
}
```

## API Endpoints

### Blob Management (Orchestrated)
```go
// Create blob with provider processing
POST /api/v1/blobs
Request:
{
    "content": "Chapter 1: The beginning",
    "provider_id": "book:my-novel",
    "metadata": {
        "type": "chapter",
        "chapter_number": 1
    },
    "auto_process": true
}

// Studio Service orchestration:
func (s *StudioService) CreateBlob(ctx context.Context, req CreateBlobRequest) (*BlobResponse, error) {
    // 1. Validate user session
    userID := getUserID(ctx)
    
    // 2. Create blob in State Service
    blob, err := s.stateClient.CreateBlob(ctx, &state.CreateBlobRequest{
        UserID:     userID,
        Content:    req.Content,
        ProviderID: req.ProviderID,
        Metadata:   req.Metadata,
    })
    
    // 3. If auto_process, trigger provider
    if req.AutoProcess && req.ProviderID != "" {
        execution, _ := s.providerClient.Execute(ctx, &provider.ExecuteRequest{
            ProviderID: req.ProviderID,
            BlobID:     blob.ID,
        })
    }
    
    // 4. Get provider UI layout for response
    var uiLayout *UILayout
    if req.ProviderID != "" {
        layout, _ := s.providerClient.GetUILayout(ctx, req.ProviderID)
        uiLayout = layout
    }
    
    // 5. Construct optimized DTO
    return &BlobResponse{
        Blob:      blob,
        UILayout:  uiLayout,
        Processing: execution != nil,
    }, nil
}
```

### DAG Visualization (Aggregated)
```go
// Get complete DAG with provider info
GET /api/v1/dag

// Studio Service aggregation:
func (s *StudioService) GetDAG(ctx context.Context) (*DAGResponse, error) {
    userID := getUserID(ctx)
    
    // 1. Get DAG structure from State Service
    dag, err := s.stateClient.GetUserDAG(ctx, userID)
    
    // 2. Get provider instances from Provider Service
    providers, _ := s.providerClient.ListUserProviders(ctx, userID)
    
    // 3. Enrich DAG nodes with provider info
    for _, node := range dag.Nodes {
        if provider, ok := providers[node.ProviderID]; ok {
            node.ProviderName = provider.Name
            node.ProviderIcon = provider.Icon
        }
    }
    
    // 4. Calculate statistics
    stats := calculateDAGStats(dag)
    
    return &DAGResponse{
        DAG:       dag,
        Providers: providers,
        Stats:     stats,
    }, nil
}
```

### Speech Input (Multi-Service)
```go
// Process speech to blob
POST /api/v1/ramble
Content-Type: multipart/form-data
Body: audio file

// Studio Service orchestration:
func (s *StudioService) ProcessRamble(ctx context.Context, audio io.Reader) (*RambleResponse, error) {
    userID := getUserID(ctx)
    
    // 1. Send audio to Media Service for transcription
    transcription, err := s.mediaClient.TranscribeAudio(ctx, &media.TranscribeRequest{
        Audio:  audio,
        Model:  "whisper-1",
    })
    
    // 2. Create blob from transcription
    blob, err := s.stateClient.CreateBlob(ctx, &state.CreateBlobRequest{
        UserID:  userID,
        Content: transcription.Text,
        Metadata: map[string]interface{}{
            "source": "ramble",
            "duration": transcription.Duration,
        },
    })
    
    // 3. Optional: Process with AI for enhancement
    if req.Enhance {
        enhanced, _ := s.coreClient.Process(ctx, &core.ProcessRequest{
            Input: transcription.Text,
            Prompt: "Clean up this transcribed speech, fixing grammar and structure",
        })
        
        // Create enhanced version as child blob
        enhancedBlob, _ := s.stateClient.CreateBlob(ctx, &state.CreateBlobRequest{
            UserID:   userID,
            Content:  enhanced.Output,
            ParentID: &blob.ID,
            Metadata: map[string]interface{}{
                "type": "enhanced_ramble",
            },
        })
    }
    
    return &RambleResponse{
        Transcription: transcription.Text,
        BlobID:        blob.ID,
        Duration:      transcription.Duration,
    }, nil
}
```

### Dynamic UI Configuration
```go
// Get UI layout for current context
GET /api/v1/ui/layout?provider_id=book:my-novel&blob_id=xxx

Response:
{
    "layout": {
        "type": "split",
        "orientation": "horizontal",
        "children": [...],
        "data_bindings": {
            "input_blob": "blob_id_123",
            "output_blob": "blob_id_456"
        }
    },
    "actions": [
        {
            "id": "expand",
            "label": "Expand Text",
            "hotkey": "cmd+e",
            "provider": "text-expander"
        }
    ],
    "theme": "light"
}
```

## WebSocket Real-Time Updates

```go
// WebSocket connection endpoint
WS /api/v1/ws

// Message types
type WSMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

// Client → Server messages
{
    "type": "subscribe",
    "payload": {
        "channels": ["blobs", "providers", "dag"]
    }
}

{
    "type": "ramble_start",
    "payload": {
        "provider_id": "book:my-novel"
    }
}

// Server → Client messages  
{
    "type": "blob_created",
    "payload": {
        "blob": {...},
        "parent_id": "..."
    }
}

{
    "type": "processing_complete",
    "payload": {
        "blob_id": "...",
        "result": {...}
    }
}

{
    "type": "dag_updated",
    "payload": {
        "affected_nodes": [...],
        "operation": "add_child"
    }
}
```

## Frontend Serving

```go
func (s *StudioService) SetupRoutes(router *gin.Engine) {
    // API routes
    api := router.Group("/api/v1")
    {
        api.POST("/blobs", s.CreateBlob)
        api.GET("/blobs/:id", s.GetBlob)
        api.GET("/dag", s.GetDAG)
        api.POST("/ramble", s.ProcessRamble)
        api.GET("/ui/layout", s.GetUILayout)
        
        // WebSocket
        api.GET("/ws", s.HandleWebSocket)
    }
    
    // Serve React app
    router.Static("/static", "./web/build/static")
    router.StaticFile("/manifest.json", "./web/build/manifest.json")
    router.StaticFile("/favicon.ico", "./web/build/favicon.ico")
    
    // Catch-all for React Router
    router.NoRoute(func(c *gin.Context) {
        // Don't serve index.html for API routes
        if !strings.HasPrefix(c.Request.URL.Path, "/api") {
            c.File("./web/build/index.html")
        }
    })
}
```

## Service Client Integration

```go
// Example: State Service Client
package stateclient

type Client struct {
    baseURL string
    http    *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        http: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (c *Client) CreateBlob(ctx context.Context, req *CreateBlobRequest) (*Blob, error) {
    url := fmt.Sprintf("%s/api/v1/users/%s/blobs", c.baseURL, req.UserID)
    
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := c.http.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var blob Blob
    json.NewDecoder(resp.Body).Decode(&blob)
    return &blob, nil
}
```

## Authentication & Session Management

```go
func (s *StudioService) AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get token from header or cookie
        token := c.GetHeader("Authorization")
        if token == "" {
            token = c.Cookie("session_token")
        }
        
        // Validate with Auth Service
        user, err := s.authClient.ValidateToken(c.Request.Context(), token)
        if err != nil {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        
        // Store user context
        c.Set("user_id", user.ID)
        c.Set("user", user)
        c.Next()
    }
}
```

## Response DTOs

```go
// Optimized DTOs that combine data from multiple services

type BlobResponse struct {
    Blob       *Blob                  `json:"blob"`
    Provider   *ProviderInfo          `json:"provider,omitempty"`
    UILayout   *UILayout              `json:"ui_layout,omitempty"`
    Children   []*BlobSummary         `json:"children,omitempty"`
    Processing bool                   `json:"processing"`
    Metrics    *BlobMetrics           `json:"metrics,omitempty"`
}

type DAGResponse struct {
    Nodes      []*DAGNode             `json:"nodes"`
    Edges      []*DAGEdge             `json:"edges"`
    Providers  map[string]*Provider   `json:"providers"`
    Stats      *DAGStats              `json:"stats"`
}

type WorkspaceResponse struct {
    Blobs      []*BlobSummary         `json:"blobs"`
    Providers  []*ProviderInstance    `json:"providers"`
    UILayout   *UILayout              `json:"ui_layout"`
    QuickActions []*Action            `json:"quick_actions"`
}
```

## Caching Strategy

```go
func (s *StudioService) GetBlobWithCache(ctx context.Context, userID, blobID string) (*BlobResponse, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("blob:%s:%s", userID, blobID)
    cached, err := s.cache.Get(ctx, cacheKey).Result()
    if err == nil {
        var response BlobResponse
        json.Unmarshal([]byte(cached), &response)
        return &response, nil
    }
    
    // Build response from services
    blob, _ := s.stateClient.GetBlob(ctx, userID, blobID)
    provider, _ := s.providerClient.GetInstance(ctx, blob.ProviderID)
    uiLayout, _ := s.providerClient.GetUILayout(ctx, blob.ProviderID)
    
    response := &BlobResponse{
        Blob:     blob,
        Provider: provider,
        UILayout: uiLayout,
    }
    
    // Cache for 5 minutes
    data, _ := json.Marshal(response)
    s.cache.Set(ctx, cacheKey, data, 5*time.Minute)
    
    return response, nil
}
```

## Error Handling

```go
func (s *StudioService) ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            
            // Map service errors to HTTP status codes
            switch {
            case errors.Is(err, stateclient.ErrBlobNotFound):
                c.JSON(404, gin.H{"error": "Blob not found"})
            case errors.Is(err, providerclient.ErrProviderNotFound):
                c.JSON(404, gin.H{"error": "Provider not found"})
            case errors.Is(err, ErrUnauthorized):
                c.JSON(401, gin.H{"error": "Unauthorized"})
            default:
                c.JSON(500, gin.H{"error": "Internal server error"})
            }
        }
    }
}
```

## Configuration

```yaml
# config/studio.yaml
service:
  port: 8010
  name: studio-service
  
frontend:
  build_path: ./web/build
  dev_mode: false
  
services:
  auth_url: http://localhost:8001
  state_url: http://localhost:8006
  provider_url: http://localhost:8007
  workflow_url: http://localhost:8005
  core_url: http://localhost:8004
  media_url: http://localhost:8009
  
websocket:
  max_connections: 10000
  ping_interval: 30s
  
cache:
  redis_url: redis://localhost:6379
  default_ttl: 5m
  
nats:
  url: nats://localhost:4222
  subscriptions:
    - state.blob.*
    - provider.execution.*
    - workflow.completed
```

## Performance Optimizations

### Parallel Service Calls
```go
func (s *StudioService) GetWorkspace(ctx context.Context, userID string) (*WorkspaceResponse, error) {
    var (
        blobs     []*Blob
        providers []*ProviderInstance
        blobsErr  error
        provErr   error
    )
    
    // Parallel calls to services
    var wg sync.WaitGroup
    wg.Add(2)
    
    go func() {
        defer wg.Done()
        blobs, blobsErr = s.stateClient.GetRecentBlobs(ctx, userID, 20)
    }()
    
    go func() {
        defer wg.Done()
        providers, provErr = s.providerClient.ListUserProviders(ctx, userID)
    }()
    
    wg.Wait()
    
    if blobsErr != nil {
        return nil, blobsErr
    }
    if provErr != nil {
        return nil, provErr
    }
    
    return &WorkspaceResponse{
        Blobs:     blobs,
        Providers: providers,
    }, nil
}
```

### Request Batching
```go
func (s *StudioService) GetBlobsBatch(ctx context.Context, userID string, blobIDs []string) ([]*BlobResponse, error) {
    // Single call to State Service for multiple blobs
    blobs, err := s.stateClient.GetBlobsBatch(ctx, userID, blobIDs)
    if err != nil {
        return nil, err
    }
    
    // Build responses
    responses := make([]*BlobResponse, len(blobs))
    for i, blob := range blobs {
        responses[i] = &BlobResponse{
            Blob: blob,
        }
    }
    
    return responses, nil
}
```

This Studio API design provides a clean orchestration layer that aggregates data from multiple services and presents it in an optimized format for the frontend.