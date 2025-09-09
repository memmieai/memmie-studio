# Core Interfaces Design

## Overview
This document defines all interfaces that services will implement and depend on. These interfaces enable dependency injection, testing, and clean architecture.

## Schema Service Interfaces

```go
// pkg/interfaces/schema.go
package interfaces

import (
    "context"
    "time"
)

// SchemaService defines the contract for schema operations
type SchemaService interface {
    // Schema CRUD
    RegisterSchema(ctx context.Context, schema SchemaDefinition) (*Schema, error)
    GetSchema(ctx context.Context, schemaID string) (*Schema, error)
    GetSchemaByName(ctx context.Context, processorID, name, version string) (*Schema, error)
    ListSchemas(ctx context.Context, filter SchemaFilter) ([]*Schema, error)
    UpdateSchemaStatus(ctx context.Context, schemaID string, status SchemaStatus) error
    
    // Validation
    ValidateData(ctx context.Context, schemaID string, data interface{}) (*ValidationResult, error)
    ValidateWithSchema(ctx context.Context, schema *Schema, data interface{}) (*ValidationResult, error)
    
    // Version Management
    GetLatestVersion(ctx context.Context, processorID, name string) (*Schema, error)
    GetVersionHistory(ctx context.Context, processorID, name string) ([]*Schema, error)
    CheckCompatibility(ctx context.Context, oldSchemaID, newSchemaID string) (*CompatibilityResult, error)
}

// SchemaRepository defines persistence operations
type SchemaRepository interface {
    Create(ctx context.Context, schema *Schema) error
    GetByID(ctx context.Context, id string) (*Schema, error)
    GetByIdentifier(ctx context.Context, processorID, name, version string) (*Schema, error)
    List(ctx context.Context, filter SchemaFilter) ([]*Schema, error)
    Update(ctx context.Context, schema *Schema) error
    Delete(ctx context.Context, id string) error
}

// SchemaValidator defines validation operations
type SchemaValidator interface {
    Compile(schema string) (CompiledSchema, error)
    Validate(compiled CompiledSchema, data interface{}) (*ValidationResult, error)
}

// Schema represents a registered schema
type Schema struct {
    ID           string                 `json:"id"`
    ProcessorID  string                 `json:"processor_id"`
    Name         string                 `json:"name"`
    Version      string                 `json:"version"`
    Definition   string                 `json:"definition"` // JSON Schema as string
    Status       SchemaStatus           `json:"status"`
    Description  string                 `json:"description"`
    Examples     []interface{}          `json:"examples"`
    Metadata     map[string]interface{} `json:"metadata"`
    CreatedAt    time.Time             `json:"created_at"`
    UpdatedAt    time.Time             `json:"updated_at"`
}

type SchemaStatus string
const (
    SchemaStatusDraft   SchemaStatus = "draft"
    SchemaStatusActive  SchemaStatus = "active"
    SchemaStatusDeprecated SchemaStatus = "deprecated"
)

type ValidationResult struct {
    Valid    bool                   `json:"valid"`
    Errors   []ValidationError      `json:"errors,omitempty"`
    Warnings []ValidationWarning    `json:"warnings,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## State Service Interfaces

```go
// pkg/interfaces/state.go
package interfaces

import (
    "context"
    "time"
)

// StateService defines operations for blob and bucket management
type StateService interface {
    // Blob Operations
    CreateBlob(ctx context.Context, req CreateBlobRequest) (*Blob, error)
    GetBlob(ctx context.Context, blobID string) (*Blob, error)
    UpdateBlob(ctx context.Context, blobID string, req UpdateBlobRequest) (*Blob, error)
    DeleteBlob(ctx context.Context, blobID string) error
    ListBlobs(ctx context.Context, filter BlobFilter) ([]*Blob, error)
    GetBlobsByBucket(ctx context.Context, bucketID string) ([]*Blob, error)
    
    // Bucket Operations
    CreateBucket(ctx context.Context, req CreateBucketRequest) (*Bucket, error)
    GetBucket(ctx context.Context, bucketID string) (*Bucket, error)
    UpdateBucket(ctx context.Context, bucketID string, req UpdateBucketRequest) (*Bucket, error)
    DeleteBucket(ctx context.Context, bucketID string) error
    ListBuckets(ctx context.Context, filter BucketFilter) ([]*Bucket, error)
    GetBucketTree(ctx context.Context, bucketID string) (*BucketTree, error)
    
    // Relationships
    AddBlobToBucket(ctx context.Context, blobID, bucketID string) error
    RemoveBlobFromBucket(ctx context.Context, blobID, bucketID string) error
    MoveBucket(ctx context.Context, bucketID string, newParentID *string) error
    
    // User State
    GetUserState(ctx context.Context, userID string) (*UserState, error)
    GetUserQuotas(ctx context.Context, userID string) (*UserQuotas, error)
}

// BlobRepository defines blob persistence
type BlobRepository interface {
    Create(ctx context.Context, blob *Blob) error
    GetByID(ctx context.Context, id string) (*Blob, error)
    GetByUser(ctx context.Context, userID string, filter BlobFilter) ([]*Blob, error)
    GetByBucket(ctx context.Context, bucketID string) ([]*Blob, error)
    Update(ctx context.Context, blob *Blob) error
    Delete(ctx context.Context, id string) error
}

// BucketRepository defines bucket persistence
type BucketRepository interface {
    Create(ctx context.Context, bucket *Bucket) error
    GetByID(ctx context.Context, id string) (*Bucket, error)
    GetByUser(ctx context.Context, userID string, filter BucketFilter) ([]*Bucket, error)
    GetChildren(ctx context.Context, parentID string) ([]*Bucket, error)
    Update(ctx context.Context, bucket *Bucket) error
    Delete(ctx context.Context, id string) error
}

// Blob represents stored content
type Blob struct {
    ID          string                 `json:"id"`
    UserID      string                 `json:"user_id"`
    ProcessorID string                 `json:"processor_id"`
    SchemaID    string                 `json:"schema_id"`
    Data        interface{}            `json:"data"`
    BucketIDs   []string              `json:"bucket_ids"`
    ParentID    *string               `json:"parent_id,omitempty"`
    DerivedIDs  []string              `json:"derived_ids"`
    Metadata    map[string]interface{} `json:"metadata"`
    CreatedAt   time.Time             `json:"created_at"`
    UpdatedAt   time.Time             `json:"updated_at"`
}

// Bucket represents an organizational container
type Bucket struct {
    ID             string                 `json:"id"`
    UserID         string                 `json:"user_id"`
    Name           string                 `json:"name"`
    Type           string                 `json:"type"`
    ParentBucketID *string               `json:"parent_bucket_id,omitempty"`
    ChildBucketIDs []string              `json:"child_bucket_ids"`
    BlobIDs        []string              `json:"blob_ids"`
    Metadata       map[string]interface{} `json:"metadata"`
    Settings       BucketSettings        `json:"settings"`
    CreatedAt      time.Time             `json:"created_at"`
    UpdatedAt      time.Time             `json:"updated_at"`
}
```

## Processor Service Interfaces

```go
// pkg/interfaces/processor.go
package interfaces

import (
    "context"
)

// ProcessorService manages processor registration and execution
type ProcessorService interface {
    // Registration
    RegisterProcessor(ctx context.Context, processor ProcessorDefinition) (*Processor, error)
    GetProcessor(ctx context.Context, processorID string) (*Processor, error)
    ListProcessors(ctx context.Context) ([]*Processor, error)
    UpdateProcessor(ctx context.Context, processorID string, update ProcessorUpdate) error
    
    // Instances
    CreateInstance(ctx context.Context, userID, processorID string, config map[string]interface{}) (*ProcessorInstance, error)
    GetUserInstances(ctx context.Context, userID string) ([]*ProcessorInstance, error)
    UpdateInstanceConfig(ctx context.Context, instanceID string, config map[string]interface{}) error
    
    // Execution
    GetProcessorForSchema(ctx context.Context, schemaID string) (*Processor, error)
    ValidateInput(ctx context.Context, processorID string, data interface{}) error
    ValidateOutput(ctx context.Context, processorID string, data interface{}) error
}

// ProcessorExecutor defines how processors execute
type ProcessorExecutor interface {
    Execute(ctx context.Context, input ProcessorInput) (*ProcessorOutput, error)
    ValidateInput(ctx context.Context, data interface{}) error
    ValidateOutput(ctx context.Context, data interface{}) error
}

// Processor represents a registered processor
type Processor struct {
    ID              string   `json:"id"`
    Name            string   `json:"name"`
    Description     string   `json:"description"`
    InputSchemaID   string   `json:"input_schema_id"`
    OutputSchemaID  string   `json:"output_schema_id"`
    SubscribeEvents []string `json:"subscribe_events"`
    EmitEvents      []string `json:"emit_events"`
    Config          ProcessorConfig `json:"config"`
    Active          bool     `json:"active"`
}

// ProcessorInstance represents a user's processor configuration
type ProcessorInstance struct {
    ID          string                 `json:"id"`
    UserID      string                 `json:"user_id"`
    ProcessorID string                 `json:"processor_id"`
    Config      map[string]interface{} `json:"config"`
    Active      bool                   `json:"active"`
}
```

## Studio API Interfaces

```go
// pkg/interfaces/studio.go
package interfaces

import (
    "context"
    "github.com/gorilla/websocket"
)

// StudioService defines the main API operations
type StudioService interface {
    // Blob Operations (delegates to StateService)
    CreateBlob(ctx context.Context, userID string, req CreateBlobRequest) (*Blob, error)
    GetBlob(ctx context.Context, userID, blobID string) (*Blob, error)
    ListBlobs(ctx context.Context, userID string, filter BlobFilter) ([]*Blob, error)
    
    // Bucket Operations (delegates to StateService)
    CreateBucket(ctx context.Context, userID string, req CreateBucketRequest) (*Bucket, error)
    GetBucket(ctx context.Context, userID, bucketID string) (*Bucket, error)
    ListBuckets(ctx context.Context, userID string, filter BucketFilter) ([]*Bucket, error)
    GetBucketTree(ctx context.Context, userID, bucketID string) (*BucketTree, error)
    
    // Book Operations (convenience methods)
    CreateBook(ctx context.Context, userID string, req CreateBookRequest) (*Bucket, error)
    AddChapter(ctx context.Context, userID, bookID string, req AddChapterRequest) (*Bucket, error)
    ExportBook(ctx context.Context, userID, bookID string, format ExportFormat) ([]byte, error)
    
    // WebSocket Management
    HandleWebSocket(ctx context.Context, userID string, conn *websocket.Conn) error
    BroadcastToUser(ctx context.Context, userID string, message interface{}) error
}

// WebSocketManager handles real-time connections
type WebSocketManager interface {
    AddConnection(userID string, conn *websocket.Conn) error
    RemoveConnection(userID string, conn *websocket.Conn) error
    SendToUser(userID string, message interface{}) error
    SendToConnections(connections []*websocket.Conn, message interface{}) error
    GetUserConnections(userID string) []*websocket.Conn
}

// EventHandler processes events from NATS
type EventHandler interface {
    HandleBlobCreated(ctx context.Context, event BlobCreatedEvent) error
    HandleBlobUpdated(ctx context.Context, event BlobUpdatedEvent) error
    HandleBucketCreated(ctx context.Context, event BucketCreatedEvent) error
    HandleProcessorCompleted(ctx context.Context, event ProcessorCompletedEvent) error
}
```

## Event Bus Interface

```go
// pkg/interfaces/eventbus.go
package interfaces

import "context"

// EventBus defines pub/sub operations
type EventBus interface {
    Publish(ctx context.Context, topic string, event interface{}) error
    Subscribe(ctx context.Context, topic string, handler EventHandler) error
    SubscribeWithGroup(ctx context.Context, topic, group string, handler EventHandler) error
    Unsubscribe(subscription Subscription) error
}

// EventHandler processes events
type EventHandler func(ctx context.Context, event []byte) error

// Subscription represents an active subscription
type Subscription interface {
    Unsubscribe() error
    Topic() string
    Group() string
}
```

## Cache Interface

```go
// pkg/interfaces/cache.go
package interfaces

import (
    "context"
    "time"
)

// Cache defines caching operations
type Cache interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    TTL(ctx context.Context, key string) (time.Duration, error)
}
```

## Auth Client Interface

```go
// pkg/interfaces/auth.go
package interfaces

import "context"

// AuthClient validates tokens with auth service
type AuthClient interface {
    ValidateToken(ctx context.Context, token string) (*TokenValidation, error)
    GetUser(ctx context.Context, userID string) (*User, error)
}

// TokenValidation contains token validation results
type TokenValidation struct {
    Valid   bool   `json:"valid"`
    UserID  string `json:"user_id"`
    Email   string `json:"email"`
    Phone   string `json:"phone"`
    Expires int64  `json:"expires"`
}

// User represents basic user information
type User struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Phone    string `json:"phone"`
    Status   string `json:"status"`
}
```

## Testing Interfaces

```go
// pkg/mocks/mocks.go
package mocks

// All interfaces should have mock implementations for testing
// Use mockgen or similar tool to generate these

// Example mock generation command:
// mockgen -source=pkg/interfaces/schema.go -destination=pkg/mocks/schema_mock.go -package=mocks
```

## Dependency Injection Structure

```go
// Example service initialization with DI
type SchemaService struct {
    repo      SchemaRepository
    validator SchemaValidator
    cache     Cache
    eventBus  EventBus
    logger    Logger
}

func NewSchemaService(
    repo SchemaRepository,
    validator SchemaValidator,
    cache Cache,
    eventBus EventBus,
    logger Logger,
) *SchemaService {
    return &SchemaService{
        repo:      repo,
        validator: validator,
        cache:     cache,
        eventBus:  eventBus,
        logger:    logger,
    }
}
```

## Interface Principles

1. **Single Responsibility**: Each interface has one clear purpose
2. **Dependency Inversion**: Depend on interfaces, not implementations
3. **Interface Segregation**: Small, focused interfaces over large ones
4. **Testability**: All interfaces have mock implementations
5. **Context First**: All methods accept context for cancellation/timeout
6. **Error Handling**: All operations return explicit errors
7. **Immutability**: Return new objects rather than modifying parameters