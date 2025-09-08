# Implementation Roadmap for Memmie Studio

## Overview
This document outlines the complete implementation strategy for Memmie Studio, from initial setup to production deployment.

## Phase 1: Foundation (Week 1-2)

### 1.1 Core Data Models
```go
// Location: internal/domain/

// Blob.go
type Blob struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Content     []byte
    ContentHash string
    ContentType string
    Size        int64
    Version     int
    
    // DAG relationships
    ParentID    *uuid.UUID
    RootID      uuid.UUID // Original blob in the chain
    Depth       int        // Distance from root
    
    // Provider metadata
    CreatedBy   string // Provider ID or "user"
    ProcessedBy map[string]ProcessingInfo
    
    // Timestamps
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time
}

// Delta.go
type Delta struct {
    ID          uuid.UUID
    BlobID      uuid.UUID
    ProviderID  string
    UserID      uuid.UUID
    
    // Operation details
    Operation   DeltaOperation
    Patch       json.RawMessage
    
    // Causality
    PreviousDelta *uuid.UUID
    CausedBy      *uuid.UUID // Event or parent delta
    
    // Versioning
    FromVersion int
    ToVersion   int
    
    // Status
    Status      DeltaStatus
    AppliedAt   *time.Time
    Error       *string
    
    CreatedAt   time.Time
}

// Provider.go
type Provider struct {
    ID              string
    Name            string
    Description     string
    Type            ProviderType
    
    // Configuration
    WorkflowID      string
    InputSchema     json.RawMessage
    OutputSchema    json.RawMessage
    Config          map[string]interface{}
    
    // Capabilities
    SupportedTypes  []string
    MaxInputSize    int64
    Timeout         time.Duration
    
    // Event subscriptions
    TriggerEvents   []EventType
    TriggerPatterns []string // Regex patterns for content
    
    // State
    Status          ProviderStatus
    Version         string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### 1.2 Repository Layer
```go
// Location: internal/repository/

// BlobRepository interface
type BlobRepository interface {
    // Basic CRUD
    Create(ctx context.Context, blob *Blob) error
    Get(ctx context.Context, id uuid.UUID) (*Blob, error)
    GetByUser(ctx context.Context, userID uuid.UUID, opts ListOptions) ([]*Blob, error)
    Update(ctx context.Context, blob *Blob) error
    Delete(ctx context.Context, id uuid.UUID) error
    
    // Version management
    GetVersion(ctx context.Context, id uuid.UUID, version int) (*Blob, error)
    GetHistory(ctx context.Context, id uuid.UUID) ([]*Blob, error)
    
    // DAG operations
    GetChildren(ctx context.Context, parentID uuid.UUID) ([]*Blob, error)
    GetAncestors(ctx context.Context, blobID uuid.UUID) ([]*Blob, error)
    GetDescendants(ctx context.Context, blobID uuid.UUID) ([]*Blob, error)
    GetDAG(ctx context.Context, rootID uuid.UUID) (*DAG, error)
}

// DeltaRepository interface
type DeltaRepository interface {
    Create(ctx context.Context, delta *Delta) error
    Get(ctx context.Context, id uuid.UUID) (*Delta, error)
    GetByBlob(ctx context.Context, blobID uuid.UUID) ([]*Delta, error)
    GetPending(ctx context.Context, limit int) ([]*Delta, error)
    
    // Delta chain operations
    GetChain(ctx context.Context, blobID uuid.UUID, fromVersion, toVersion int) ([]*Delta, error)
    GetLatest(ctx context.Context, blobID uuid.UUID) (*Delta, error)
    
    // Status updates
    MarkApplied(ctx context.Context, id uuid.UUID) error
    MarkFailed(ctx context.Context, id uuid.UUID, err error) error
}
```

### 1.3 Service Layer
```go
// Location: internal/service/

// StudioService orchestrates blob operations
type StudioService struct {
    blobRepo     BlobRepository
    deltaRepo    DeltaRepository
    providerRepo ProviderRepository
    
    storage      StorageBackend
    eventBus     EventBus
    cache        Cache
    
    deltaEngine  *DeltaEngine
    dagProcessor *DAGProcessor
}

// Core operations
func (s *StudioService) CreateBlob(ctx context.Context, input CreateBlobInput) (*Blob, error) {
    // 1. Validate input
    if err := s.validateBlobInput(input); err != nil {
        return nil, err
    }
    
    // 2. Store content
    contentID, err := s.storage.Store(ctx, input.Content)
    if err != nil {
        return nil, err
    }
    
    // 3. Create blob record
    blob := &Blob{
        ID:          uuid.New(),
        UserID:      input.UserID,
        ContentID:   contentID,
        ContentType: input.ContentType,
        Size:        int64(len(input.Content)),
        Version:     1,
        CreatedBy:   "user",
        CreatedAt:   time.Now(),
    }
    
    if err := s.blobRepo.Create(ctx, blob); err != nil {
        return nil, err
    }
    
    // 4. Create initial delta
    delta := &Delta{
        ID:        uuid.New(),
        BlobID:    blob.ID,
        Operation: OpCreate,
        Patch:     json.RawMessage(fmt.Sprintf(`{"content_id": "%s"}`, contentID)),
        ToVersion: 1,
        CreatedAt: time.Now(),
    }
    
    if err := s.deltaRepo.Create(ctx, delta); err != nil {
        return nil, err
    }
    
    // 5. Emit creation event
    event := BlobEvent{
        Type:    EventBlobCreated,
        BlobID:  blob.ID,
        UserID:  blob.UserID,
        DeltaID: delta.ID,
    }
    
    if err := s.eventBus.Publish(ctx, event); err != nil {
        log.Error("Failed to publish event", "error", err)
    }
    
    return blob, nil
}
```

## Phase 2: Delta Engine (Week 2-3)

### 2.1 Delta Application System
```go
// Location: internal/engine/

type DeltaEngine struct {
    deltaRepo DeltaRepository
    blobRepo  BlobRepository
    storage   StorageBackend
}

func (e *DeltaEngine) ApplyDelta(ctx context.Context, delta *Delta) error {
    // 1. Get current blob state
    blob, err := e.blobRepo.Get(ctx, delta.BlobID)
    if err != nil {
        return err
    }
    
    // 2. Validate delta can be applied
    if blob.Version != delta.FromVersion {
        return ErrVersionMismatch
    }
    
    // 3. Apply the delta based on operation
    switch delta.Operation {
    case OpUpdate:
        if err := e.applyUpdate(ctx, blob, delta); err != nil {
            return err
        }
        
    case OpTransform:
        if err := e.applyTransform(ctx, blob, delta); err != nil {
            return err
        }
        
    case OpDelete:
        if err := e.applyDelete(ctx, blob, delta); err != nil {
            return err
        }
        
    case OpRevert:
        if err := e.applyRevert(ctx, blob, delta); err != nil {
            return err
        }
    }
    
    // 4. Update blob version
    blob.Version = delta.ToVersion
    blob.UpdatedAt = time.Now()
    
    // 5. Save updated blob
    if err := e.blobRepo.Update(ctx, blob); err != nil {
        return err
    }
    
    // 6. Mark delta as applied
    if err := e.deltaRepo.MarkApplied(ctx, delta.ID); err != nil {
        return err
    }
    
    return nil
}

// Conflict resolution
func (e *DeltaEngine) ResolveConflicts(ctx context.Context, deltas []*Delta) ([]*Delta, error) {
    // Implement Operational Transformation (OT) or CRDT-based resolution
    // For now, use last-write-wins with causality tracking
    
    sort.Slice(deltas, func(i, j int) bool {
        // Sort by causality chain, then timestamp
        if deltas[i].PreviousDelta != nil && *deltas[i].PreviousDelta == deltas[j].ID {
            return false // j comes before i
        }
        if deltas[j].PreviousDelta != nil && *deltas[j].PreviousDelta == deltas[i].ID {
            return true // i comes before j
        }
        return deltas[i].CreatedAt.Before(deltas[j].CreatedAt)
    })
    
    return deltas, nil
}
```

### 2.2 Materialization Engine
```go
// Rebuild blob state from delta history
func (e *DeltaEngine) Materialize(ctx context.Context, blobID uuid.UUID, targetVersion *int) (*Blob, error) {
    // 1. Get delta chain
    var deltas []*Delta
    if targetVersion != nil {
        deltas, _ = e.deltaRepo.GetChain(ctx, blobID, 0, *targetVersion)
    } else {
        deltas, _ = e.deltaRepo.GetByBlob(ctx, blobID)
    }
    
    // 2. Start with empty state
    blob := &Blob{
        ID:          blobID,
        ProcessedBy: make(map[string]ProcessingInfo),
    }
    
    // 3. Apply each delta in sequence
    for _, delta := range deltas {
        if err := e.applyDeltaToState(blob, delta); err != nil {
            return nil, fmt.Errorf("failed at delta %s: %w", delta.ID, err)
        }
    }
    
    return blob, nil
}
```

## Phase 3: Provider System (Week 3-4)

### 3.1 Provider Registry
```go
// Location: internal/provider/

type ProviderRegistry struct {
    providers map[string]*Provider
    mu        sync.RWMutex
    repo      ProviderRepository
}

func (r *ProviderRegistry) Register(ctx context.Context, provider *Provider) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // Validate provider
    if err := r.validateProvider(provider); err != nil {
        return err
    }
    
    // Store in database
    if err := r.repo.Create(ctx, provider); err != nil {
        return err
    }
    
    // Cache in memory
    r.providers[provider.ID] = provider
    
    // Register workflow with Temporal
    if err := r.registerWorkflow(provider); err != nil {
        return err
    }
    
    return nil
}
```

### 3.2 Provider Execution Framework
```go
// Location: internal/provider/executor/

type ProviderExecutor struct {
    registry       *ProviderRegistry
    workflowClient client.Client
    taskQueue      string
}

func (e *ProviderExecutor) Execute(ctx context.Context, input ProviderInput) (*ProviderOutput, error) {
    // 1. Get provider
    provider, err := e.registry.Get(input.ProviderID)
    if err != nil {
        return nil, err
    }
    
    // 2. Validate input against schema
    if err := e.validateInput(provider, input); err != nil {
        return nil, err
    }
    
    // 3. Start workflow
    workflowOptions := client.StartWorkflowOptions{
        ID:        fmt.Sprintf("%s-%s-%d", provider.ID, input.BlobID, time.Now().Unix()),
        TaskQueue: e.taskQueue,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 3,
            BackoffCoefficient: 2.0,
        },
    }
    
    we, err := e.workflowClient.ExecuteWorkflow(
        ctx,
        workflowOptions,
        provider.WorkflowID,
        input,
    )
    if err != nil {
        return nil, err
    }
    
    // 4. Wait for result (with timeout)
    var output ProviderOutput
    err = we.Get(ctx, &output)
    
    return &output, err
}
```

## Phase 4: DAG Processing (Week 4-5)

### 4.1 DAG Manager
```go
// Location: internal/dag/

type DAGManager struct {
    blobRepo BlobRepository
    cache    Cache
}

func (m *DAGManager) BuildDAG(ctx context.Context, rootID uuid.UUID) (*DAG, error) {
    // Check cache
    if cached, ok := m.cache.Get(fmt.Sprintf("dag:%s", rootID)); ok {
        return cached.(*DAG), nil
    }
    
    dag := &DAG{
        RootID: rootID,
        Nodes:  make(map[uuid.UUID]*Node),
        Edges:  make([]*Edge, 0),
    }
    
    // Recursive build
    visited := make(map[uuid.UUID]bool)
    if err := m.buildRecursive(ctx, rootID, dag, visited); err != nil {
        return nil, err
    }
    
    // Calculate levels for parallel processing
    dag.Levels = m.calculateLevels(dag)
    
    // Cache result
    m.cache.Set(fmt.Sprintf("dag:%s", rootID), dag, 5*time.Minute)
    
    return dag, nil
}

func (m *DAGManager) PropagateUpdate(ctx context.Context, blobID uuid.UUID, delta *Delta) error {
    // 1. Get all descendants
    descendants, err := m.blobRepo.GetDescendants(ctx, blobID)
    if err != nil {
        return err
    }
    
    // 2. Group by level for parallel processing
    levels := m.groupByDepth(descendants)
    
    // 3. Process each level
    for _, level := range levels {
        // Process nodes at same level in parallel
        g, ctx := errgroup.WithContext(ctx)
        
        for _, blob := range level {
            blob := blob // Capture for goroutine
            g.Go(func() error {
                return m.reprocessBlob(ctx, blob, delta)
            })
        }
        
        if err := g.Wait(); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Phase 5: Event System Integration (Week 5)

### 5.1 Event-Driven Architecture
```go
// Location: internal/events/

type EventManager struct {
    bus           EventBus
    registry      *ProviderRegistry
    executor      *ProviderExecutor
    subscriptions map[string][]Subscription
}

func (m *EventManager) Initialize(ctx context.Context) error {
    // Subscribe to blob events
    topics := []string{
        "studio.blob.created",
        "studio.blob.updated",
        "studio.blob.deleted",
        "studio.delta.applied",
    }
    
    for _, topic := range topics {
        if err := m.bus.Subscribe(topic, m.handleEvent); err != nil {
            return err
        }
    }
    
    return nil
}

func (m *EventManager) handleEvent(event Event) {
    // 1. Parse event
    var blobEvent BlobEvent
    if err := json.Unmarshal(event.Data, &blobEvent); err != nil {
        log.Error("Failed to parse event", "error", err)
        return
    }
    
    // 2. Find interested providers
    providers := m.registry.GetProvidersForEvent(blobEvent.Type)
    
    // 3. Execute providers
    for _, provider := range providers {
        go func(p *Provider) {
            input := ProviderInput{
                ProviderID: p.ID,
                BlobID:     blobEvent.BlobID,
                Event:      blobEvent,
            }
            
            if _, err := m.executor.Execute(context.Background(), input); err != nil {
                log.Error("Provider execution failed", 
                    "provider", p.ID, 
                    "blob", blobEvent.BlobID,
                    "error", err)
            }
        }(provider)
    }
}
```

## Phase 6: Storage Optimization (Week 6)

### 6.1 Content-Addressed Storage
```go
// Location: internal/storage/

type ContentAddressedStorage struct {
    backend StorageBackend
    hasher  hash.Hash
}

func (s *ContentAddressedStorage) Store(ctx context.Context, content []byte) (string, error) {
    // 1. Calculate content hash
    s.hasher.Reset()
    s.hasher.Write(content)
    contentHash := hex.EncodeToString(s.hasher.Sum(nil))
    
    // 2. Check if already exists (deduplication)
    if exists, _ := s.backend.Exists(ctx, contentHash); exists {
        return contentHash, nil
    }
    
    // 3. Compress if beneficial
    compressed := content
    if len(content) > 1024 { // Only compress if > 1KB
        compressed = s.compress(content)
    }
    
    // 4. Store with hash as key
    if err := s.backend.Put(ctx, contentHash, compressed); err != nil {
        return "", err
    }
    
    return contentHash, nil
}
```

### 6.2 Tiered Storage
```go
type TieredStorage struct {
    hot  StorageBackend // Redis for recent/frequent
    warm StorageBackend // PostgreSQL for medium
    cold StorageBackend // S3 for archival
}

func (s *TieredStorage) Get(ctx context.Context, key string) ([]byte, error) {
    // Try hot tier first
    if data, err := s.hot.Get(ctx, key); err == nil {
        return data, nil
    }
    
    // Try warm tier
    if data, err := s.warm.Get(ctx, key); err == nil {
        // Promote to hot tier
        go s.hot.Put(context.Background(), key, data)
        return data, nil
    }
    
    // Fall back to cold tier
    data, err := s.cold.Get(ctx, key)
    if err != nil {
        return nil, err
    }
    
    // Promote to warm tier
    go s.warm.Put(context.Background(), key, data)
    
    return data, nil
}
```

## Phase 7: API Implementation (Week 7)

### 7.1 RESTful API
```go
// Location: internal/handler/

type StudioHandler struct {
    studioSvc *StudioService
    auth      *AuthMiddleware
}

func (h *StudioHandler) RegisterRoutes(r *mux.Router) {
    // Blob endpoints
    r.HandleFunc("/api/v1/studio/blobs", h.CreateBlob).Methods("POST")
    r.HandleFunc("/api/v1/studio/blobs/{id}", h.GetBlob).Methods("GET")
    r.HandleFunc("/api/v1/studio/blobs/{id}", h.UpdateBlob).Methods("PATCH")
    r.HandleFunc("/api/v1/studio/blobs/{id}", h.DeleteBlob).Methods("DELETE")
    
    // Version endpoints
    r.HandleFunc("/api/v1/studio/blobs/{id}/versions", h.GetVersions).Methods("GET")
    r.HandleFunc("/api/v1/studio/blobs/{id}/versions/{version}", h.GetVersion).Methods("GET")
    
    // DAG endpoints
    r.HandleFunc("/api/v1/studio/blobs/{id}/dag", h.GetDAG).Methods("GET")
    r.HandleFunc("/api/v1/studio/blobs/{id}/children", h.GetChildren).Methods("GET")
    
    // Delta endpoints
    r.HandleFunc("/api/v1/studio/blobs/{id}/deltas", h.GetDeltas).Methods("GET")
    r.HandleFunc("/api/v1/studio/deltas/{id}", h.GetDelta).Methods("GET")
    
    // Provider endpoints
    r.HandleFunc("/api/v1/studio/providers", h.ListProviders).Methods("GET")
    r.HandleFunc("/api/v1/studio/providers", h.RegisterProvider).Methods("POST")
    r.HandleFunc("/api/v1/studio/providers/{id}/process", h.ProcessBlob).Methods("POST")
}
```

### 7.2 WebSocket Support
```go
// Real-time updates
func (h *StudioHandler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // Get user ID from auth
    userID := h.auth.GetUserID(r)
    
    // Subscribe to user's blob events
    sub := h.eventBus.Subscribe(fmt.Sprintf("user.%s.blobs", userID))
    defer sub.Unsubscribe()
    
    // Stream events to client
    for event := range sub.Channel() {
        if err := conn.WriteJSON(event); err != nil {
            break
        }
    }
}
```

## Phase 8: Testing & Quality Assurance (Week 8)

### 8.1 Unit Tests
```go
// Location: internal/service/studio_test.go

func TestCreateBlob(t *testing.T) {
    // Setup
    ctx := context.Background()
    mockRepo := mocks.NewMockBlobRepository()
    mockStorage := mocks.NewMockStorage()
    svc := NewStudioService(mockRepo, mockStorage)
    
    // Test
    input := CreateBlobInput{
        UserID:      uuid.New(),
        Content:     []byte("test content"),
        ContentType: "text/plain",
    }
    
    blob, err := svc.CreateBlob(ctx, input)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, blob)
    assert.Equal(t, input.UserID, blob.UserID)
    assert.Equal(t, 1, blob.Version)
}
```

### 8.2 Integration Tests
```go
func TestProviderProcessing(t *testing.T) {
    // Setup full system
    ctx := context.Background()
    system := setupTestSystem(t)
    defer system.Cleanup()
    
    // Register test provider
    provider := &Provider{
        ID:   "test-expander",
        Name: "Test Expander",
        WorkflowID: "expand-workflow",
    }
    system.Registry.Register(ctx, provider)
    
    // Create blob
    blob := system.CreateTestBlob("Short text")
    
    // Wait for processing
    time.Sleep(2 * time.Second)
    
    // Check derived blob was created
    children, err := system.BlobRepo.GetChildren(ctx, blob.ID)
    assert.NoError(t, err)
    assert.Len(t, children, 1)
    assert.Contains(t, string(children[0].Content), "Expanded:")
}
```

## Deployment Strategy

### Docker Configuration
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o studio cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/studio .
EXPOSE 8010
CMD ["./studio"]
```

### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: memmie-studio
spec:
  replicas: 3
  selector:
    matchLabels:
      app: memmie-studio
  template:
    metadata:
      labels:
        app: memmie-studio
    spec:
      containers:
      - name: studio
        image: memmieai/studio:latest
        ports:
        - containerPort: 8010
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: studio-secrets
              key: database-url
        - name: TEMPORAL_HOST
          value: temporal:7233
```

## Success Metrics

1. **Performance**
   - Delta application < 100ms
   - DAG traversal < 200ms for 100 nodes
   - Provider execution < 5s average

2. **Reliability**
   - 99.9% uptime
   - Zero data loss
   - Successful delta application rate > 99%

3. **Scalability**
   - Support 10,000 concurrent users
   - Process 1M blobs/day
   - Handle DAGs with 1000+ nodes

This roadmap provides a clear path from initial implementation to production deployment, with each phase building on the previous one.