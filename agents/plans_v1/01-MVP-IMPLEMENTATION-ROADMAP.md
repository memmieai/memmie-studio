# MVP Implementation Roadmap - ReYNa Studio

## Overview

This document provides the complete implementation roadmap for ReYNa Studio MVP, focusing on delivering a functional creative platform with the dynamic bucket system. We're building from scratch, leveraging only what works from existing services.

## Current Service Analysis & Decisions

### Services to KEEP and ENHANCE

#### 1. **memmie-auth** (Port 8001) ✅ KEEP
- **Current**: Fully functional authentication with phone/email
- **Role in New System**: Unchanged - handles all authentication
- **No changes needed**: Works perfectly as-is

#### 2. **memmie-state** (Port 8006) ✅ TRANSFORM
- **Current**: Generic key-value storage
- **New Role**: Blob and Bucket storage engine
- **Major refactor**: Replace generic State with Blob/Bucket models

#### 3. **memmie-gateway** (Port 8000) ✅ KEEP & EXTEND
- **Current**: API gateway and routing
- **New Role**: Add WebSocket support for real-time
- **Enhancement**: Add WebSocket proxy capabilities

### Services to REPLACE

#### 4. **memmie-conversation** (Port 8002) ❌ REPLACE with Buckets
- **Current**: Chat-specific storage
- **Decision**: OBSOLETE - Buckets handle this better
- **Migration**: Convert conversations to conversation-type buckets

#### 5. **memmie-memory** (Port 8003) ❌ DEFER
- **Current**: Vector search and embeddings
- **Decision**: Not needed for MVP, add in Phase 2
- **Future**: Useful for semantic search across buckets

#### 6. **memmie-core** (Port 8004) ❌ REPLACE with Processors
- **Current**: Monolithic AI service
- **Decision**: Break into individual processors
- **New Model**: Each AI capability becomes a processor

#### 7. **memmie-workflow** (Port 8005) ❌ SIMPLIFY
- **Current**: Complex Temporal workflows
- **Decision**: Replace with simple processor chains for MVP
- **Future**: Reintroduce for complex multi-step processes

#### 8. **memmie-provider** (Port 8007) ✅ RENAME & TRANSFORM
- **Current**: Empty shell
- **New Role**: Becomes Processor Service
- **Purpose**: Registry and orchestration of all processors

#### 9. **memmie-notification** (Port 8008) ⏸️ DEFER
- **Current**: Notification system
- **Decision**: Not needed for MVP
- **Future**: Phase 2 for email/push notifications

#### 10. **memmie-media** (Port 8009) ⏸️ DEFER
- **Current**: Media storage
- **Decision**: Store media as blobs for MVP
- **Future**: CDN integration in Phase 2

### NEW Services to CREATE

#### 11. **memmie-schema** (Port 8011) 🆕 CREATE
- **Purpose**: Central schema registry and validation
- **Database**: PostgreSQL
- **Priority**: CRITICAL - Must be built first

#### 12. **memmie-studio** (Port 8010) 🆕 CREATE
- **Purpose**: Studio API with WebSocket support
- **Features**: Real-time updates, client connections
- **Priority**: HIGH - User-facing API

## Service Interface Design

### Core Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                         Client Apps                          │
│                    (Web, Mobile, Desktop)                    │
└─────────────────┬───────────────────────────┬────────────────┘
                  │                           │
                  │ HTTPS                     │ WebSocket
                  ▼                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Gateway (Port 8000)                       │
│                 Routes & Load Balancing                      │
└────┬──────┬──────┬──────┬──────┬──────┬────────────────────┘
     │      │      │      │      │      │
     ▼      ▼      ▼      ▼      ▼      ▼
┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐
│  Auth  ││ Studio ││ State  ││ Schema ││Processor││  NATS  │
│ (8001) ││ (8010) ││ (8006) ││ (8011) ││ (8007)  ││ (4222) │
└────────┘└────────┘└────────┘└────────┘└────────┘└────────┘
```

### Service Communication Patterns

#### 1. Synchronous HTTP Calls
```go
// Studio API → Auth Service
GET /api/v1/validate-token
Headers: Authorization: Bearer <token>
Response: { user_id, roles, expires_at }

// Studio API → State Service  
POST /api/v1/blobs
Body: { processor_id, schema_id, data, bucket_ids }
Response: { blob_id, created_at }

// Studio API → Schema Service
POST /api/v1/schemas/{id}/validate
Body: { data }
Response: { valid, errors }
```

#### 2. Asynchronous Events (NATS)
```yaml
# Blob creation flow
State Service → NATS: blob.created.{schema-id}
Processor → NATS: processor.started.{processor-id}
Processor → NATS: processor.completed.{processor-id}
State Service → NATS: blob.derived.{schema-id}

# Bucket events
State Service → NATS: bucket.created
State Service → NATS: bucket.blob.added
State Service → NATS: bucket.structure.changed
```

#### 3. WebSocket Messages
```javascript
// Client → Studio (subscribe)
{ "action": "subscribe", "buckets": ["bucket-123"] }

// Studio → Client (real-time update)
{ "type": "blob.created", "bucket_id": "bucket-123", "blob_id": "blob-456" }

// Client → Studio (create)
{ "action": "create_blob", "data": {...} }
```

## Implementation Phases

### Phase 0: Infrastructure Setup (Day 1-2)

1. **Setup Development Environment**
   ```bash
   # Create new services
   cd /home/uneid/iter3/memmieai
   cp -r memmie-provider memmie-schema
   cp -r memmie-provider memmie-studio
   
   # Update ports in configs
   # memmie-schema: 8011
   # memmie-studio: 8010
   ```

2. **Database Setup**
   ```sql
   -- PostgreSQL for Schema Service
   CREATE DATABASE memmie_schema;
   
   -- MongoDB indexes for State Service
   db.blobs.createIndex({ "user_id": 1, "bucket_ids": 1 })
   db.buckets.createIndex({ "user_id": 1, "type": 1 })
   ```

3. **NATS Topics Configuration**
   ```yaml
   topics:
     - blob.created.*
     - blob.updated.*
     - blob.derived.*
     - bucket.created
     - bucket.updated
     - processor.started.*
     - processor.completed.*
   ```

### Phase 1: Schema Service (Day 3-5)

**File Structure:**
```
memmie-schema/
├── cmd/server/main.go
├── internal/
│   ├── service/schema_service.go
│   ├── repository/postgres.go
│   ├── validator/json_schema.go
│   └── models/schema.go
└── pkg/client/schema_client.go
```

**Implementation Steps:**
1. Create schema models and database migrations
2. Implement CRUD operations for schemas
3. Add JSON Schema validation engine
4. Create version management system
5. Build client library for other services
6. Seed with system schemas (bucket-v1, blob-v1)

**Key Endpoints:**
```go
POST   /api/v1/schemas              // Register schema
GET    /api/v1/schemas/{id}         // Get schema
POST   /api/v1/schemas/{id}/validate // Validate data
GET    /api/v1/processors/{id}/schemas // List processor schemas
```

### Phase 2: State Service Transformation (Day 6-9)

**Current State Model:**
```go
// OLD - Generic storage
type UserState struct {
    UserID string
    State  map[string]interface{}
}
```

**New Models:**
```go
// NEW - Structured storage
type Blob struct {
    ID          primitive.ObjectID
    UserID      string
    ProcessorID string
    SchemaID    string
    Data        interface{}
    BucketIDs   []string
    ParentID    *string
    DerivedIDs  []string
}

type Bucket struct {
    ID             string
    UserID         string
    Name           string
    Type           string
    ParentBucketID *string
    ChildBucketIDs []string
    BlobIDs        []string
    Metadata       map[string]interface{}
}
```

**Migration Steps:**
1. Add new collections (blobs, buckets)
2. Create repositories for blob/bucket operations
3. Integrate schema validation client
4. Implement event publishing to NATS
5. Add bucket hierarchy management
6. Create migration script for existing data

### Phase 3: Processor Service Setup (Day 10-12)

**Transform memmie-provider into memmie-processor:**

```go
// Processor registration
type Processor struct {
    ID            string
    Name          string
    InputSchemaID string
    OutputSchemaID string
    SubscribeEvents []string
    Config        ProcessorConfig
}

// Processor instance per user
type ProcessorInstance struct {
    ProcessorID string
    UserID      string
    Settings    map[string]interface{}
    Active      bool
}
```

**Implementation:**
1. Design processor registration system
2. Create event subscription manager
3. Build processor health monitoring
4. Implement processor instance management
5. Add processor discovery endpoint

### Phase 4: Studio API with WebSocket (Day 13-16)

**Core Components:**
```go
// WebSocket connection manager
type ConnectionManager struct {
    connections map[string]*UserConnections
    mu          sync.RWMutex
}

// User connections (multiple devices)
type UserConnections struct {
    UserID      string
    Connections []*websocket.Conn
    Subscriptions []string // bucket IDs
}
```

**Implementation:**
1. Create REST API endpoints for blob/bucket CRUD
2. Implement WebSocket server
3. Build event filtering system
4. Add connection management
5. Create subscription system for buckets
6. Integrate with all backend services

### Phase 5: Text Expansion Processor (Day 17-19)

**First Real Processor:**
```go
type TextExpansionProcessor struct {
    schemaClient *schema.Client
    stateClient  *state.Client
    aiClient     *openai.Client
}

func (p *TextExpansionProcessor) Process(ctx context.Context, blob *Blob) error {
    // 1. Validate input matches schema
    // 2. Extract text content
    // 3. Call GPT-4 for expansion
    // 4. Create derived blob
    // 5. Emit completion event
}
```

### Phase 6: Basic UI (Day 20-22)

**Minimal React Interface:**
```javascript
// Components needed
- BucketTree (navigation)
- BlobEditor (text input)
- SplitView (original/expanded)
- WebSocketProvider (real-time)
```

### Phase 7: Integration & Testing (Day 23-25)

1. End-to-end testing of complete flow
2. Performance optimization
3. Error handling improvements
4. Documentation updates
5. Deployment scripts

## Service Update Guide

### Updating memmie-state

**Step 1: Add new models**
```go
// internal/models/blob.go
package models

type Blob struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    UserID      string            `bson:"user_id"`
    ProcessorID string            `bson:"processor_id"`
    SchemaID    string            `bson:"schema_id"`
    Data        interface{}       `bson:"data"`
    BucketIDs   []string          `bson:"bucket_ids"`
    ParentID    *string           `bson:"parent_id,omitempty"`
    DerivedIDs  []string          `bson:"derived_ids"`
    CreatedAt   time.Time         `bson:"created_at"`
    UpdatedAt   time.Time         `bson:"updated_at"`
}
```

**Step 2: Create repositories**
```go
// internal/repository/blob_repository.go
type BlobRepository interface {
    Create(ctx context.Context, blob *models.Blob) error
    GetByID(ctx context.Context, id string) (*models.Blob, error)
    GetByBucket(ctx context.Context, bucketID string) ([]*models.Blob, error)
    Update(ctx context.Context, blob *models.Blob) error
}
```

**Step 3: Add schema validation**
```go
// internal/service/state_service.go
func (s *StateService) CreateBlob(ctx context.Context, req CreateBlobRequest) (*Blob, error) {
    // Validate with schema service
    valid, err := s.schemaClient.Validate(ctx, req.SchemaID, req.Data)
    if !valid {
        return nil, ErrInvalidSchema
    }
    
    // Create blob
    blob := &Blob{...}
    
    // Publish event
    s.eventBus.Publish("blob.created." + req.SchemaID, blob)
    
    return blob, nil
}
```

### Updating memmie-gateway

**Add WebSocket proxy:**
```go
// internal/proxy/websocket.go
func (p *Proxy) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Upgrade to WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    
    // Proxy to Studio API
    backend, _ := url.Parse("ws://memmie-studio:8010")
    proxyWebSocket(conn, backend)
}
```

## Success Metrics

### MVP Must-Haves
- ✅ User can create buckets
- ✅ User can create blobs in buckets
- ✅ Text expansion processor works
- ✅ Real-time updates via WebSocket
- ✅ Basic web UI functional

### Performance Targets
- Blob creation: <500ms
- WebSocket latency: <100ms
- Text expansion: <3s
- Support 100 concurrent users

## Risk Mitigation

### Technical Risks
1. **Schema validation performance**
   - Solution: Cache compiled schemas
   
2. **WebSocket scalability**
   - Solution: Sticky sessions for MVP
   
3. **MongoDB performance**
   - Solution: Proper indexing strategy

### Schedule Risks
1. **Delayed AI integration**
   - Solution: Mock processor for testing
   
2. **Complex bucket operations**
   - Solution: Limit nesting depth for MVP

## Deployment Strategy

```yaml
# docker-compose.yml updates
services:
  memmie-schema:
    build: ./memmie-schema
    ports:
      - "8011:8011"
    environment:
      - DATABASE_URL=postgresql://...
      
  memmie-studio:
    build: ./memmie-studio
    ports:
      - "8010:8010"
    environment:
      - NATS_URL=nats://nats:4222
```

## Next Steps

1. Review and approve this roadmap
2. Set up new service repositories
3. Begin Phase 0 infrastructure
4. Daily progress updates
5. Weekly demos of working features