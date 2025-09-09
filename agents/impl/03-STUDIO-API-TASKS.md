# Studio API Implementation Tasks

## Prerequisites
- Schema Service running on port 8011
- State Service updated with blob/bucket support on port 8006
- Auth Service running on port 8001
- NATS running on port 4222

## Task 1: Initialize Studio Service Repository
**File**: New repository setup
```bash
cd /home/uneid/iter3/memmieai
mkdir memmie-studio-api
cd memmie-studio-api
git init
go mod init github.com/memmieai/memmie-studio-api
```

**Dependencies**:
```bash
go get github.com/memmieai/memmie-common
go get github.com/memmieai/memmie-schema/pkg/client
go get github.com/gorilla/mux
go get github.com/gorilla/websocket
go get github.com/nats-io/nats.go
go get github.com/redis/go-redis/v9
go get golang.org/x/sync/errgroup
```

## Task 2: Create Project Structure
```
memmie-studio-api/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── models/
│   │   ├── requests.go
│   │   └── responses.go
│   ├── clients/
│   │   ├── auth_client.go
│   │   ├── state_client.go
│   │   └── schema_client.go
│   ├── websocket/
│   │   ├── hub.go
│   │   ├── client.go
│   │   ├── message.go
│   │   └── manager.go
│   ├── service/
│   │   ├── studio_service.go
│   │   ├── blob_service.go
│   │   ├── bucket_service.go
│   │   └── book_service.go
│   ├── handler/
│   │   ├── http.go
│   │   ├── websocket.go
│   │   └── routes.go
│   ├── events/
│   │   ├── subscriber.go
│   │   └── publisher.go
│   └── middleware/
│       ├── auth.go
│       ├── cors.go
│       └── logging.go
├── pkg/
│   └── client/
│       └── studio_client.go
├── Dockerfile
├── .env.example
└── Makefile
```

## Task 3: Define WebSocket Message Types
**File**: `internal/models/messages.go`
```go
package models

import (
    "encoding/json"
    "time"
)

// WebSocket message types
const (
    // Client → Server
    MessageTypeSubscribe   = "subscribe"
    MessageTypeUnsubscribe = "unsubscribe"
    MessageTypeCreateBlob  = "create_blob"
    MessageTypeUpdateBlob  = "update_blob"
    MessageTypeCreateBucket = "create_bucket"
    MessageTypeHeartbeat   = "heartbeat"
    
    // Server → Client
    MessageTypeBlobCreated   = "blob.created"
    MessageTypeBlobUpdated   = "blob.updated"
    MessageTypeBlobDerived   = "blob.derived"
    MessageTypeBucketCreated = "bucket.created"
    MessageTypeBucketUpdated = "bucket.updated"
    MessageTypeError         = "error"
    MessageTypeAck           = "ack"
)

// ClientMessage represents messages from client to server
type ClientMessage struct {
    ID        string          `json:"id"`        // Client-generated request ID
    Type      string          `json:"type"`      // Message type
    Action    string          `json:"action"`    // Action to perform
    Data      json.RawMessage `json:"data"`      // Payload
    Timestamp int64           `json:"timestamp"` // Client timestamp
}

// ServerMessage represents messages from server to client
type ServerMessage struct {
    ID          string          `json:"id,omitempty"`     // Request ID if responding
    Type        string          `json:"type"`             // Message type
    Data        interface{}     `json:"data,omitempty"`   // Payload
    Error       string          `json:"error,omitempty"`  // Error message if any
    Timestamp   int64           `json:"timestamp"`        // Server timestamp
}

// Subscription represents what a client wants to listen to
type Subscription struct {
    UserID    string   `json:"user_id"`
    BucketIDs []string `json:"bucket_ids,omitempty"` // Subscribe to specific buckets
    Types     []string `json:"types,omitempty"`      // Subscribe to specific event types
}

// CreateBlobMessage for WebSocket blob creation
type CreateBlobMessage struct {
    ProcessorID string                 `json:"processor_id"`
    SchemaID    string                 `json:"schema_id"`
    Data        interface{}            `json:"data"`
    BucketIDs   []string              `json:"bucket_ids,omitempty"`
    ParentID    *string               `json:"parent_id,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

## Task 4: Implement WebSocket Hub
**File**: `internal/websocket/hub.go`
```go
package websocket

import (
    "context"
    "encoding/json"
    "sync"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type Hub struct {
    clients    map[string]map[*Client]bool // userID -> clients
    register   chan *Client
    unregister chan *Client
    broadcast  chan *BroadcastMessage
    mu         sync.RWMutex
    logger     logger.Logger
}

type Client struct {
    hub          *Hub
    conn         *websocket.Conn
    userID       string
    send         chan []byte
    subscriptions map[string]bool // bucket IDs
    mu           sync.RWMutex
}

type BroadcastMessage struct {
    UserID    string
    BucketIDs []string
    Message   []byte
}

func NewHub(logger logger.Logger) *Hub {
    return &Hub{
        clients:    make(map[string]map[*Client]bool),
        register:   make(chan *Client, 100),
        unregister: make(chan *Client, 100),
        broadcast:  make(chan *BroadcastMessage, 1000),
        logger:     logger,
    }
}

func (h *Hub) Run(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
            
        case client := <-h.register:
            h.mu.Lock()
            if h.clients[client.userID] == nil {
                h.clients[client.userID] = make(map[*Client]bool)
            }
            h.clients[client.userID][client] = true
            h.mu.Unlock()
            h.logger.Info("Client connected", "user_id", client.userID)
            
        case client := <-h.unregister:
            h.mu.Lock()
            if clients, ok := h.clients[client.userID]; ok {
                if _, ok := clients[client]; ok {
                    delete(clients, client)
                    close(client.send)
                    if len(clients) == 0 {
                        delete(h.clients, client.userID)
                    }
                }
            }
            h.mu.Unlock()
            h.logger.Info("Client disconnected", "user_id", client.userID)
            
        case msg := <-h.broadcast:
            h.sendToUser(msg)
            
        case <-ticker.C:
            h.pingClients()
        }
    }
}

func (h *Hub) sendToUser(msg *BroadcastMessage) {
    h.mu.RLock()
    clients := h.clients[msg.UserID]
    h.mu.RUnlock()
    
    for client := range clients {
        // Check if client is subscribed to any of the buckets
        if len(msg.BucketIDs) > 0 {
            subscribed := false
            client.mu.RLock()
            for _, bucketID := range msg.BucketIDs {
                if client.subscriptions[bucketID] {
                    subscribed = true
                    break
                }
            }
            client.mu.RUnlock()
            
            if !subscribed {
                continue
            }
        }
        
        select {
        case client.send <- msg.Message:
        default:
            // Client's send channel is full, close it
            h.unregister <- client
        }
    }
}

func (h *Hub) BroadcastToUser(userID string, message interface{}) error {
    data, err := json.Marshal(message)
    if err != nil {
        return err
    }
    
    h.broadcast <- &BroadcastMessage{
        UserID:  userID,
        Message: data,
    }
    
    return nil
}
```

## Task 5: Implement WebSocket Client Handler
**File**: `internal/websocket/client.go`
```go
package websocket

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/memmieai/memmie-studio-api/internal/models"
)

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    maxMessageSize = 1024 * 1024 // 1MB
)

func (c *Client) ReadPump(handler MessageHandler) {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    
    for {
        var msg models.ClientMessage
        err := c.conn.ReadJSON(&msg)
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                c.hub.logger.Error("WebSocket error", "error", err)
            }
            break
        }
        
        // Handle message
        go c.handleMessage(handler, &msg)
    }
}

func (c *Client) WritePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    
    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            c.conn.WriteMessage(websocket.TextMessage, message)
            
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

func (c *Client) handleMessage(handler MessageHandler, msg *models.ClientMessage) {
    ctx := context.WithValue(context.Background(), "user_id", c.userID)
    
    response := &models.ServerMessage{
        ID:        msg.ID,
        Timestamp: time.Now().Unix(),
    }
    
    switch msg.Type {
    case models.MessageTypeSubscribe:
        var sub models.Subscription
        if err := json.Unmarshal(msg.Data, &sub); err != nil {
            response.Type = models.MessageTypeError
            response.Error = "Invalid subscription data"
        } else {
            c.mu.Lock()
            for _, bucketID := range sub.BucketIDs {
                c.subscriptions[bucketID] = true
            }
            c.mu.Unlock()
            response.Type = models.MessageTypeAck
        }
        
    case models.MessageTypeCreateBlob:
        var req models.CreateBlobMessage
        if err := json.Unmarshal(msg.Data, &req); err != nil {
            response.Type = models.MessageTypeError
            response.Error = "Invalid blob data"
        } else {
            blob, err := handler.CreateBlob(ctx, c.userID, req)
            if err != nil {
                response.Type = models.MessageTypeError
                response.Error = err.Error()
            } else {
                response.Type = models.MessageTypeBlobCreated
                response.Data = blob
            }
        }
        
    case models.MessageTypeHeartbeat:
        response.Type = models.MessageTypeAck
        
    default:
        response.Type = models.MessageTypeError
        response.Error = "Unknown message type"
    }
    
    // Send response
    if data, err := json.Marshal(response); err == nil {
        select {
        case c.send <- data:
        default:
            // Channel full, client will be disconnected
        }
    }
}

type MessageHandler interface {
    CreateBlob(ctx context.Context, userID string, req models.CreateBlobMessage) (*models.Blob, error)
    CreateBucket(ctx context.Context, userID string, req models.CreateBucketMessage) (*models.Bucket, error)
}
```

## Task 6: Create Service Clients
**File**: `internal/clients/state_client.go`
```go
package clients

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type StateClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewStateClient(baseURL string) *StateClient {
    return &StateClient{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *StateClient) CreateBlob(ctx context.Context, userID string, req CreateBlobRequest) (*Blob, error) {
    url := fmt.Sprintf("%s/api/v1/users/%s/blobs", c.baseURL, userID)
    
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        return nil, fmt.Errorf("failed to create blob: status %d", resp.StatusCode)
    }
    
    var blob Blob
    if err := json.NewDecoder(resp.Body).Decode(&blob); err != nil {
        return nil, err
    }
    
    return &blob, nil
}

func (c *StateClient) CreateBucket(ctx context.Context, userID string, req CreateBucketRequest) (*Bucket, error) {
    // Similar implementation
}

func (c *StateClient) GetBucketTree(ctx context.Context, userID, bucketID string) (*BucketTree, error) {
    // Implementation
}

func (c *StateClient) ExportBucket(ctx context.Context, userID, bucketID, format string) ([]byte, error) {
    url := fmt.Sprintf("%s/api/v1/users/%s/buckets/%s/export?format=%s", 
        c.baseURL, userID, bucketID, format)
    
    httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to export bucket: status %d", resp.StatusCode)
    }
    
    return io.ReadAll(resp.Body)
}
```

## Task 7: Implement Studio Service
**File**: `internal/service/studio_service.go`
```go
package service

import (
    "context"
    "fmt"
    
    "github.com/memmieai/memmie-studio-api/internal/clients"
    "github.com/memmieai/memmie-studio-api/internal/models"
    "github.com/memmieai/memmie-studio-api/internal/websocket"
    "github.com/memmieai/memmie-common/pkg/logger"
    "github.com/nats-io/nats.go"
)

type StudioService struct {
    stateClient  *clients.StateClient
    schemaClient *clients.SchemaClient
    authClient   *clients.AuthClient
    wsHub        *websocket.Hub
    natsConn     *nats.Conn
    logger       logger.Logger
}

func NewStudioService(
    stateURL, schemaURL, authURL string,
    wsHub *websocket.Hub,
    natsConn *nats.Conn,
    logger logger.Logger,
) *StudioService {
    return &StudioService{
        stateClient:  clients.NewStateClient(stateURL),
        schemaClient: clients.NewSchemaClient(schemaURL),
        authClient:   clients.NewAuthClient(authURL),
        wsHub:        wsHub,
        natsConn:     natsConn,
        logger:       logger,
    }
}

// Implement MessageHandler interface for WebSocket
func (s *StudioService) CreateBlob(ctx context.Context, userID string, req models.CreateBlobMessage) (*models.Blob, error) {
    // Validate with schema service
    validation, err := s.schemaClient.ValidateData(ctx, req.SchemaID, req.Data)
    if err != nil {
        return nil, fmt.Errorf("schema validation failed: %w", err)
    }
    
    if !validation.Valid {
        return nil, fmt.Errorf("data validation failed: %v", validation.Errors)
    }
    
    // Create blob via state service
    stateReq := clients.CreateBlobRequest{
        ProcessorID: req.ProcessorID,
        SchemaID:    req.SchemaID,
        Data:        req.Data,
        BucketIDs:   req.BucketIDs,
        ParentID:    req.ParentID,
        Metadata:    req.Metadata,
    }
    
    blob, err := s.stateClient.CreateBlob(ctx, userID, stateReq)
    if err != nil {
        return nil, err
    }
    
    // Broadcast to user's other connections
    s.wsHub.BroadcastToUser(userID, models.ServerMessage{
        Type:      models.MessageTypeBlobCreated,
        Data:      blob,
        Timestamp: time.Now().Unix(),
    })
    
    return blob, nil
}

func (s *StudioService) CreateBook(ctx context.Context, userID string, req models.CreateBookRequest) (*models.Bucket, error) {
    // Create book bucket
    bucketReq := clients.CreateBucketRequest{
        Name: req.Title,
        Type: "book",
        Metadata: map[string]interface{}{
            "author":      req.Author,
            "genre":       req.Genre,
            "description": req.Description,
            "chapters_planned": req.ChaptersPlanned,
        },
    }
    
    book, err := s.stateClient.CreateBucket(ctx, userID, bucketReq)
    if err != nil {
        return nil, err
    }
    
    // Create chapter buckets if requested
    if req.CreateChapters > 0 {
        for i := 1; i <= req.CreateChapters; i++ {
            chapterReq := clients.CreateBucketRequest{
                Name:           fmt.Sprintf("Chapter %d", i),
                Type:           "chapter",
                ParentBucketID: &book.ID,
                Metadata: map[string]interface{}{
                    "chapter_number": i,
                    "status": "draft",
                },
            }
            
            _, err := s.stateClient.CreateBucket(ctx, userID, chapterReq)
            if err != nil {
                s.logger.Warn("Failed to create chapter", "number", i, "error", err)
            }
        }
    }
    
    return book, nil
}

func (s *StudioService) ExportBook(ctx context.Context, userID, bookID string, format string) ([]byte, error) {
    // Get book bucket
    book, err := s.stateClient.GetBucket(ctx, userID, bookID)
    if err != nil {
        return nil, err
    }
    
    if book.Type != "book" {
        return nil, fmt.Errorf("bucket is not a book")
    }
    
    // Export via state service
    return s.stateClient.ExportBucket(ctx, userID, bookID, format)
}
```

## Task 8: Create Event Subscriber
**File**: `internal/events/subscriber.go`
```go
package events

import (
    "context"
    "encoding/json"
    
    "github.com/memmieai/memmie-studio-api/internal/models"
    "github.com/memmieai/memmie-studio-api/internal/websocket"
    "github.com/memmieai/memmie-common/pkg/logger"
    "github.com/nats-io/nats.go"
)

type EventSubscriber struct {
    natsConn *nats.Conn
    wsHub    *websocket.Hub
    logger   logger.Logger
}

func NewEventSubscriber(natsConn *nats.Conn, wsHub *websocket.Hub, logger logger.Logger) *EventSubscriber {
    return &EventSubscriber{
        natsConn: natsConn,
        wsHub:    wsHub,
        logger:   logger,
    }
}

func (s *EventSubscriber) Start(ctx context.Context) error {
    // Subscribe to blob events
    s.natsConn.Subscribe("blob.created.*", func(msg *nats.Msg) {
        s.handleBlobCreated(msg.Data)
    })
    
    s.natsConn.Subscribe("blob.updated.*", func(msg *nats.Msg) {
        s.handleBlobUpdated(msg.Data)
    })
    
    s.natsConn.Subscribe("blob.derived.*", func(msg *nats.Msg) {
        s.handleBlobDerived(msg.Data)
    })
    
    // Subscribe to bucket events
    s.natsConn.Subscribe("bucket.created", func(msg *nats.Msg) {
        s.handleBucketCreated(msg.Data)
    })
    
    s.natsConn.Subscribe("bucket.updated", func(msg *nats.Msg) {
        s.handleBucketUpdated(msg.Data)
    })
    
    // Subscribe to processor events
    s.natsConn.Subscribe("processor.completed.*", func(msg *nats.Msg) {
        s.handleProcessorCompleted(msg.Data)
    })
    
    s.logger.Info("Event subscriber started")
    
    <-ctx.Done()
    return nil
}

func (s *EventSubscriber) handleBlobCreated(data []byte) {
    var event struct {
        BlobID    string   `json:"blob_id"`
        UserID    string   `json:"user_id"`
        BucketIDs []string `json:"bucket_ids"`
    }
    
    if err := json.Unmarshal(data, &event); err != nil {
        s.logger.Error("Failed to unmarshal blob created event", "error", err)
        return
    }
    
    // Broadcast to user
    message := models.ServerMessage{
        Type: models.MessageTypeBlobCreated,
        Data: map[string]interface{}{
            "blob_id":    event.BlobID,
            "bucket_ids": event.BucketIDs,
        },
        Timestamp: time.Now().Unix(),
    }
    
    s.wsHub.BroadcastToUser(event.UserID, message)
}

func (s *EventSubscriber) handleBlobDerived(data []byte) {
    var event struct {
        ParentID    string   `json:"parent_id"`
        DerivedID   string   `json:"derived_id"`
        UserID      string   `json:"user_id"`
        ProcessorID string   `json:"processor_id"`
        BucketIDs   []string `json:"bucket_ids"`
    }
    
    if err := json.Unmarshal(data, &event); err != nil {
        s.logger.Error("Failed to unmarshal blob derived event", "error", err)
        return
    }
    
    // Broadcast to user with bucket filter
    message := models.ServerMessage{
        Type: models.MessageTypeBlobDerived,
        Data: map[string]interface{}{
            "parent_id":    event.ParentID,
            "derived_id":   event.DerivedID,
            "processor_id": event.ProcessorID,
            "bucket_ids":   event.BucketIDs,
        },
        Timestamp: time.Now().Unix(),
    }
    
    s.wsHub.BroadcastToUserBuckets(event.UserID, event.BucketIDs, message)
}
```

## Task 9: Create HTTP Handlers
**File**: `internal/handler/http.go`
```go
package handler

import (
    "encoding/json"
    "net/http"
    
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-studio-api/internal/models"
    "github.com/memmieai/memmie-studio-api/internal/service"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type HTTPHandler struct {
    service *service.StudioService
    logger  logger.Logger
}

func NewHTTPHandler(service *service.StudioService, logger logger.Logger) *HTTPHandler {
    return &HTTPHandler{
        service: service,
        logger:  logger,
    }
}

func (h *HTTPHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(string)
    
    var req models.CreateBookRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    book, err := h.service.CreateBook(r.Context(), userID, req)
    if err != nil {
        h.logger.Error("Failed to create book", "error", err)
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    h.respondJSON(w, http.StatusCreated, book)
}

func (h *HTTPHandler) ExportBook(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(string)
    bookID := mux.Vars(r)["book_id"]
    format := r.URL.Query().Get("format")
    
    if format == "" {
        format = "text"
    }
    
    data, err := h.service.ExportBook(r.Context(), userID, bookID, format)
    if err != nil {
        h.logger.Error("Failed to export book", "error", err)
        h.respondError(w, http.StatusInternalServerError, "Export failed")
        return
    }
    
    contentType := "text/plain"
    if format == "json" {
        contentType = "application/json"
    } else if format == "markdown" {
        contentType = "text/markdown"
    }
    
    w.Header().Set("Content-Type", contentType)
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"book-%s.%s\"", bookID, format))
    w.Write(data)
}

func (h *HTTPHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(string)
    
    books, err := h.service.ListBucketsByType(r.Context(), userID, "book")
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, "Failed to list books")
        return
    }
    
    h.respondJSON(w, http.StatusOK, map[string]interface{}{
        "books": books,
        "count": len(books),
    })
}
```

## Task 10: Create WebSocket Handler
**File**: `internal/handler/websocket.go`
```go
package handler

import (
    "net/http"
    
    "github.com/gorilla/websocket"
    "github.com/memmieai/memmie-studio-api/internal/service"
    ws "github.com/memmieai/memmie-studio-api/internal/websocket"
    "github.com/memmieai/memmie-common/pkg/logger"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // Configure CORS properly in production
        return true
    },
}

type WebSocketHandler struct {
    hub     *ws.Hub
    service *service.StudioService
    logger  logger.Logger
}

func NewWebSocketHandler(hub *ws.Hub, service *service.StudioService, logger logger.Logger) *WebSocketHandler {
    return &WebSocketHandler{
        hub:     hub,
        service: service,
        logger:  logger,
    }
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Get user ID from context (set by auth middleware)
    userID, ok := r.Context().Value("user_id").(string)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Upgrade connection
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error("Failed to upgrade connection", "error", err)
        return
    }
    
    // Create client
    client := &ws.Client{
        Hub:           h.hub,
        Conn:          conn,
        UserID:        userID,
        Send:          make(chan []byte, 256),
        Subscriptions: make(map[string]bool),
    }
    
    // Register client
    h.hub.Register <- client
    
    // Start goroutines
    go client.WritePump()
    go client.ReadPump(h.service)
}
```

## Task 11: Create Auth Middleware
**File**: `internal/middleware/auth.go`
```go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/memmieai/memmie-studio-api/internal/clients"
)

type AuthMiddleware struct {
    authClient *clients.AuthClient
    logger     logger.Logger
}

func NewAuthMiddleware(authURL string, logger logger.Logger) *AuthMiddleware {
    return &AuthMiddleware{
        authClient: clients.NewAuthClient(authURL),
        logger:     logger,
    }
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Missing authorization header", http.StatusUnauthorized)
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
            return
        }
        
        token := parts[1]
        
        // Validate token
        validation, err := m.authClient.ValidateToken(r.Context(), token)
        if err != nil || !validation.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        // Add user ID to context
        ctx := context.WithValue(r.Context(), "user_id", validation.UserID)
        ctx = context.WithValue(ctx, "user_email", validation.Email)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Task 12: Setup Routes
**File**: `internal/handler/routes.go`
```go
package handler

import (
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-studio-api/internal/middleware"
)

func SetupRoutes(
    router *mux.Router,
    httpHandler *HTTPHandler,
    wsHandler *WebSocketHandler,
    authMiddleware *middleware.AuthMiddleware,
) {
    // Health check (no auth)
    router.HandleFunc("/health", httpHandler.HealthCheck).Methods("GET")
    
    // WebSocket endpoint (auth via query param or first message)
    router.HandleFunc("/ws", wsHandler.HandleWebSocket)
    
    // API routes (all require auth)
    api := router.PathPrefix("/api/v1").Subrouter()
    api.Use(authMiddleware.Authenticate)
    
    // Book operations
    api.HandleFunc("/books", httpHandler.CreateBook).Methods("POST")
    api.HandleFunc("/books", httpHandler.ListBooks).Methods("GET")
    api.HandleFunc("/books/{book_id}", httpHandler.GetBook).Methods("GET")
    api.HandleFunc("/books/{book_id}/export", httpHandler.ExportBook).Methods("GET")
    api.HandleFunc("/books/{book_id}/chapters", httpHandler.AddChapter).Methods("POST")
    
    // Blob operations
    api.HandleFunc("/blobs", httpHandler.CreateBlob).Methods("POST")
    api.HandleFunc("/blobs/{blob_id}", httpHandler.GetBlob).Methods("GET")
    api.HandleFunc("/blobs", httpHandler.ListBlobs).Methods("GET")
    
    // Bucket operations
    api.HandleFunc("/buckets", httpHandler.CreateBucket).Methods("POST")
    api.HandleFunc("/buckets/{bucket_id}", httpHandler.GetBucket).Methods("GET")
    api.HandleFunc("/buckets/{bucket_id}/tree", httpHandler.GetBucketTree).Methods("GET")
    api.HandleFunc("/buckets/{bucket_id}/blobs", httpHandler.AddBlobToBucket).Methods("POST")
}
```

## Task 13: Create Main Server
**File**: `cmd/server/main.go`
```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-studio-api/internal/config"
    "github.com/memmieai/memmie-studio-api/internal/events"
    "github.com/memmieai/memmie-studio-api/internal/handler"
    "github.com/memmieai/memmie-studio-api/internal/middleware"
    "github.com/memmieai/memmie-studio-api/internal/service"
    "github.com/memmieai/memmie-studio-api/internal/websocket"
    "github.com/memmieai/memmie-common/pkg/logger"
    "github.com/nats-io/nats.go"
)

func main() {
    // Load config
    cfg := config.LoadConfig()
    
    // Initialize logger
    log := logger.NewConsoleLogger("studio-api", logger.InfoLevel)
    
    // Connect to NATS
    natsConn, err := nats.Connect(cfg.NATSUrl)
    if err != nil {
        log.Fatal("Failed to connect to NATS", "error", err)
    }
    defer natsConn.Close()
    
    // Initialize WebSocket hub
    wsHub := websocket.NewHub(log)
    
    // Initialize service
    studioService := service.NewStudioService(
        cfg.StateServiceURL,
        cfg.SchemaServiceURL,
        cfg.AuthServiceURL,
        wsHub,
        natsConn,
        log,
    )
    
    // Initialize event subscriber
    eventSub := events.NewEventSubscriber(natsConn, wsHub, log)
    
    // Initialize handlers
    httpHandler := handler.NewHTTPHandler(studioService, log)
    wsHandler := handler.NewWebSocketHandler(wsHub, studioService, log)
    
    // Initialize middleware
    authMiddleware := middleware.NewAuthMiddleware(cfg.AuthServiceURL, log)
    corsMiddleware := middleware.NewCORSMiddleware()
    
    // Setup routes
    router := mux.NewRouter()
    router.Use(corsMiddleware.Handle)
    router.Use(middleware.LoggingMiddleware(log))
    
    handler.SetupRoutes(router, httpHandler, wsHandler, authMiddleware)
    
    // Start WebSocket hub
    go wsHub.Run(context.Background())
    
    // Start event subscriber
    go eventSub.Start(context.Background())
    
    // Start HTTP server
    srv := &http.Server{
        Addr:         ":" + cfg.Port,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }
    
    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan
        
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        
        if err := srv.Shutdown(ctx); err != nil {
            log.Error("Server shutdown failed", "error", err)
        }
    }()
    
    log.Info("Studio API starting", "port", cfg.Port)
    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatal("Server failed", "error", err)
    }
}
```

## Task 14: Create Integration Tests
**File**: `internal/service/integration_test.go`
```go
package service_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStudioService_BookCreationFlow(t *testing.T) {
    // Setup test environment
    svc, cleanup := setupTestService(t)
    defer cleanup()
    
    userID := "test-user-123"
    ctx := context.Background()
    
    // Create a book
    book, err := svc.CreateBook(ctx, userID, models.CreateBookRequest{
        Title:           "Test Novel",
        Author:          "Test Author",
        Genre:           "Fiction",
        Description:     "A test book",
        ChaptersPlanned: 10,
        CreateChapters:  3,
    })
    require.NoError(t, err)
    assert.Equal(t, "book", book.Type)
    
    // Add content to chapter 1
    blob, err := svc.CreateBlob(ctx, userID, models.CreateBlobMessage{
        ProcessorID: "user-input",
        SchemaID:    "text-input-v1",
        Data: map[string]interface{}{
            "content": "Chapter 1 content...",
        },
        BucketIDs: []string{book.ChildBucketIDs[0]}, // First chapter
    })
    require.NoError(t, err)
    
    // Export book
    exported, err := svc.ExportBook(ctx, userID, book.ID, "text")
    require.NoError(t, err)
    assert.Contains(t, string(exported), "Test Novel")
    assert.Contains(t, string(exported), "Chapter 1 content")
}

func TestStudioService_WebSocketFlow(t *testing.T) {
    // Setup WebSocket connection
    ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:8010/ws", nil)
    require.NoError(t, err)
    defer ws.Close()
    
    // Subscribe to bucket
    err = ws.WriteJSON(map[string]interface{}{
        "id":   "req-1",
        "type": "subscribe",
        "data": map[string]interface{}{
            "bucket_ids": []string{"test-bucket"},
        },
    })
    require.NoError(t, err)
    
    // Read acknowledgment
    var response map[string]interface{}
    err = ws.ReadJSON(&response)
    require.NoError(t, err)
    assert.Equal(t, "ack", response["type"])
    
    // Create blob via WebSocket
    err = ws.WriteJSON(map[string]interface{}{
        "id":   "req-2",
        "type": "create_blob",
        "data": map[string]interface{}{
            "processor_id": "user-input",
            "schema_id":    "text-input-v1",
            "data": map[string]interface{}{
                "content": "Test content",
            },
            "bucket_ids": []string{"test-bucket"},
        },
    })
    require.NoError(t, err)
    
    // Read blob created response
    err = ws.ReadJSON(&response)
    require.NoError(t, err)
    assert.Equal(t, "blob.created", response["type"])
}
```

## Task 15: Create Dockerfile
**File**: `Dockerfile`
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o studio-api cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/studio-api .

EXPOSE 8010
CMD ["./studio-api"]
```

## Testing Checklist
- [ ] Book creation and organization
- [ ] Blob creation with validation
- [ ] WebSocket connection handling
- [ ] Real-time event delivery
- [ ] Export functionality
- [ ] Auth integration
- [ ] Concurrent WebSocket connections
- [ ] Event subscription filtering
- [ ] Error handling

## Success Criteria
- [ ] Can create books with chapters
- [ ] WebSocket delivers updates <100ms
- [ ] Handles 100 concurrent connections
- [ ] Events filtered by bucket subscription
- [ ] Export produces valid text/JSON
- [ ] Graceful reconnection handling