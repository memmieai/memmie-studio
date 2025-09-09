# Schema Service Implementation Tasks

## Prerequisites
- PostgreSQL database running
- Go 1.21+ installed
- memmie-common cloned and accessible

## Task 1: Initialize Schema Service Repository
**File**: New repository setup
```bash
# Commands to execute
cd /home/uneid/iter3/memmieai
mkdir memmie-schema
cd memmie-schema
git init
go mod init github.com/memmieai/memmie-schema
```

**Dependencies to add**:
```bash
go get github.com/memmieai/memmie-common
go get github.com/lib/pq
go get github.com/golang-migrate/migrate/v4
go get github.com/xeipuuv/gojsonschema
go get github.com/gorilla/mux
go get github.com/nats-io/nats.go
go get github.com/redis/go-redis/v9
```

## Task 2: Create Project Structure
**Files to create**:
```
memmie-schema/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── models/
│   │   └── schema.go
│   ├── repository/
│   │   ├── interface.go
│   │   └── postgres.go
│   ├── validator/
│   │   ├── interface.go
│   │   └── jsonschema.go
│   ├── service/
│   │   ├── interface.go
│   │   └── schema_service.go
│   ├── handler/
│   │   ├── http.go
│   │   └── routes.go
│   └── cache/
│       ├── interface.go
│       └── redis.go
├── pkg/
│   └── client/
│       └── schema_client.go
├── migrations/
│   └── 001_create_schemas_table.sql
├── Dockerfile
├── .env.example
└── Makefile
```

## Task 3: Define Schema Models
**File**: `internal/models/schema.go`
```go
package models

import (
    "time"
    "database/sql/driver"
)

type Schema struct {
    ID           string                 `json:"id" db:"id"`
    ProcessorID  string                 `json:"processor_id" db:"processor_id"`
    Name         string                 `json:"name" db:"name"`
    Version      string                 `json:"version" db:"version"`
    Definition   string                 `json:"definition" db:"definition"`
    Status       SchemaStatus           `json:"status" db:"status"`
    Description  string                 `json:"description" db:"description"`
    Examples     JSONB                  `json:"examples" db:"examples"`
    Metadata     JSONB                  `json:"metadata" db:"metadata"`
    CreatedAt    time.Time             `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time             `json:"updated_at" db:"updated_at"`
}

type SchemaStatus string
const (
    SchemaStatusDraft      SchemaStatus = "draft"
    SchemaStatusActive     SchemaStatus = "active"
    SchemaStatusDeprecated SchemaStatus = "deprecated"
)

// JSONB handles PostgreSQL jsonb type
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
    // Implementation for database driver
}

func (j *JSONB) Scan(value interface{}) error {
    // Implementation for database scanning
}
```

## Task 4: Create Database Migrations
**File**: `migrations/001_create_schemas_table.sql`
```sql
-- Up Migration
CREATE TABLE IF NOT EXISTS schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    processor_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    definition TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'draft',
    description TEXT,
    examples JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(processor_id, name, version)
);

CREATE INDEX idx_schemas_processor ON schemas(processor_id);
CREATE INDEX idx_schemas_status ON schemas(status);
CREATE INDEX idx_schemas_name_version ON schemas(name, version);

-- Down Migration
DROP TABLE IF EXISTS schemas;
```

## Task 5: Implement Repository Layer
**File**: `internal/repository/postgres.go`
```go
package repository

import (
    "context"
    "database/sql"
    "fmt"
    
    _ "github.com/lib/pq"
    "github.com/memmieai/memmie-schema/internal/models"
)

type PostgresRepository struct {
    db *sql.DB
}

func NewPostgresRepository(connStr string) (*PostgresRepository, error) {
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Create(ctx context.Context, schema *models.Schema) error {
    query := `
        INSERT INTO schemas (processor_id, name, version, definition, status, description, examples, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, created_at, updated_at
    `
    
    err := r.db.QueryRowContext(
        ctx, query,
        schema.ProcessorID, schema.Name, schema.Version,
        schema.Definition, schema.Status, schema.Description,
        schema.Examples, schema.Metadata,
    ).Scan(&schema.ID, &schema.CreatedAt, &schema.UpdatedAt)
    
    return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*models.Schema, error) {
    // Implementation
}

func (r *PostgresRepository) GetByIdentifier(ctx context.Context, processorID, name, version string) (*models.Schema, error) {
    // Implementation
}

// Add test file: internal/repository/postgres_test.go
```

## Task 6: Implement JSON Schema Validator
**File**: `internal/validator/jsonschema.go`
```go
package validator

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/xeipuuv/gojsonschema"
)

type JSONSchemaValidator struct {
    cache map[string]*gojsonschema.Schema // Simple in-memory cache
}

func NewJSONSchemaValidator() *JSONSchemaValidator {
    return &JSONSchemaValidator{
        cache: make(map[string]*gojsonschema.Schema),
    }
}

func (v *JSONSchemaValidator) Compile(schemaStr string) (*gojsonschema.Schema, error) {
    // Check cache first
    if compiled, ok := v.cache[schemaStr]; ok {
        return compiled, nil
    }
    
    schemaLoader := gojsonschema.NewStringLoader(schemaStr)
    compiled, err := gojsonschema.NewSchema(schemaLoader)
    if err != nil {
        return nil, fmt.Errorf("failed to compile schema: %w", err)
    }
    
    // Cache compiled schema
    v.cache[schemaStr] = compiled
    return compiled, nil
}

func (v *JSONSchemaValidator) Validate(schema *gojsonschema.Schema, data interface{}) (*ValidationResult, error) {
    dataJSON, err := json.Marshal(data)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal data: %w", err)
    }
    
    dataLoader := gojsonschema.NewBytesLoader(dataJSON)
    result, err := schema.Validate(dataLoader)
    if err != nil {
        return nil, fmt.Errorf("validation error: %w", err)
    }
    
    return &ValidationResult{
        Valid:  result.Valid(),
        Errors: convertErrors(result.Errors()),
    }, nil
}

// Add test file: internal/validator/jsonschema_test.go
```

## Task 7: Implement Schema Service
**File**: `internal/service/schema_service.go`
```go
package service

import (
    "context"
    "fmt"
    
    "github.com/memmieai/memmie-schema/internal/models"
    "github.com/memmieai/memmie-schema/internal/repository"
    "github.com/memmieai/memmie-schema/internal/validator"
    "github.com/memmieai/memmie-schema/internal/cache"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type SchemaService struct {
    repo      repository.SchemaRepository
    validator validator.SchemaValidator
    cache     cache.Cache
    eventBus  EventBus
    logger    logger.Logger
}

func NewSchemaService(
    repo repository.SchemaRepository,
    validator validator.SchemaValidator,
    cache cache.Cache,
    eventBus EventBus,
    logger logger.Logger,
) *SchemaService {
    return &SchemaService{
        repo:      repo,
        validator: validator,
        cache:     cache,
        eventBus:  eventBus,
        logger:    logger,
    }
}

func (s *SchemaService) RegisterSchema(ctx context.Context, def SchemaDefinition) (*models.Schema, error) {
    // Validate the schema definition itself
    if err := s.validateSchemaDefinition(def.Definition); err != nil {
        return nil, fmt.Errorf("invalid schema definition: %w", err)
    }
    
    // Check for existing schema
    existing, _ := s.repo.GetByIdentifier(ctx, def.ProcessorID, def.Name, def.Version)
    if existing != nil {
        return nil, fmt.Errorf("schema already exists")
    }
    
    schema := &models.Schema{
        ProcessorID: def.ProcessorID,
        Name:        def.Name,
        Version:     def.Version,
        Definition:  def.Definition,
        Status:      models.SchemaStatusDraft,
        Description: def.Description,
        Examples:    def.Examples,
        Metadata:    def.Metadata,
    }
    
    if err := s.repo.Create(ctx, schema); err != nil {
        return nil, fmt.Errorf("failed to create schema: %w", err)
    }
    
    // Publish event
    s.eventBus.Publish(ctx, "schema.registered", schema)
    
    return schema, nil
}

func (s *SchemaService) ValidateData(ctx context.Context, schemaID string, data interface{}) (*ValidationResult, error) {
    // Get schema from cache or database
    schema, err := s.getSchemaWithCache(ctx, schemaID)
    if err != nil {
        return nil, fmt.Errorf("failed to get schema: %w", err)
    }
    
    // Compile and validate
    compiled, err := s.validator.Compile(schema.Definition)
    if err != nil {
        return nil, fmt.Errorf("failed to compile schema: %w", err)
    }
    
    return s.validator.Validate(compiled, data)
}

// Add test file: internal/service/schema_service_test.go
```

## Task 8: Create HTTP Handlers
**File**: `internal/handler/http.go`
```go
package handler

import (
    "encoding/json"
    "net/http"
    
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-schema/internal/service"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type HTTPHandler struct {
    service service.SchemaService
    logger  logger.Logger
}

func NewHTTPHandler(service service.SchemaService, logger logger.Logger) *HTTPHandler {
    return &HTTPHandler{
        service: service,
        logger:  logger,
    }
}

func (h *HTTPHandler) RegisterSchema(w http.ResponseWriter, r *http.Request) {
    var req RegisterSchemaRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    
    schema, err := h.service.RegisterSchema(r.Context(), req.ToDefinition())
    if err != nil {
        h.logger.Error("Failed to register schema", "error", err)
        h.respondError(w, http.StatusInternalServerError, "Failed to register schema")
        return
    }
    
    h.respondJSON(w, http.StatusCreated, schema)
}

func (h *HTTPHandler) ValidateData(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    schemaID := vars["id"]
    
    var data interface{}
    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid JSON data")
        return
    }
    
    result, err := h.service.ValidateData(r.Context(), schemaID, data)
    if err != nil {
        h.logger.Error("Validation failed", "error", err)
        h.respondError(w, http.StatusInternalServerError, "Validation failed")
        return
    }
    
    h.respondJSON(w, http.StatusOK, result)
}

// Add test file: internal/handler/http_test.go
```

## Task 9: Create Router
**File**: `internal/handler/routes.go`
```go
package handler

import (
    "github.com/gorilla/mux"
)

func (h *HTTPHandler) RegisterRoutes(router *mux.Router) {
    api := router.PathPrefix("/api/v1").Subrouter()
    
    // Schema endpoints
    api.HandleFunc("/schemas", h.RegisterSchema).Methods("POST")
    api.HandleFunc("/schemas/{id}", h.GetSchema).Methods("GET")
    api.HandleFunc("/schemas/{id}/validate", h.ValidateData).Methods("POST")
    api.HandleFunc("/schemas/{id}/activate", h.ActivateSchema).Methods("PUT")
    
    // Processor schemas
    api.HandleFunc("/processors/{id}/schemas", h.ListProcessorSchemas).Methods("GET")
    
    // Health check
    api.HandleFunc("/health", h.HealthCheck).Methods("GET")
}
```

## Task 10: Implement Redis Cache
**File**: `internal/cache/redis.go`
```go
package cache

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(addr, password string) *RedisCache {
    client := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       0,
    })
    
    return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
    val, err := c.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    
    var data interface{}
    if err := json.Unmarshal([]byte(val), &data); err != nil {
        return nil, err
    }
    
    return data, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    
    return c.client.Set(ctx, key, data, ttl).Err()
}

// Add test file: internal/cache/redis_test.go
```

## Task 11: Create Schema Client Library
**File**: `pkg/client/schema_client.go`
```go
package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type SchemaClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewSchemaClient(baseURL string) *SchemaClient {
    return &SchemaClient{
        baseURL:    baseURL,
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *SchemaClient) ValidateData(ctx context.Context, schemaID string, data interface{}) (*ValidationResult, error) {
    url := fmt.Sprintf("%s/api/v1/schemas/%s/validate", c.baseURL, schemaID)
    
    body, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result ValidationResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

// Add more client methods...
```

## Task 12: Create Configuration
**File**: `internal/config/config.go`
```go
package config

import (
    "os"
    "strconv"
)

type Config struct {
    Port         string
    DatabaseURL  string
    RedisURL     string
    NATSUrl      string
    LogLevel     string
}

func LoadConfig() *Config {
    return &Config{
        Port:        getEnv("PORT", "8011"),
        DatabaseURL: getEnv("DATABASE_URL", "postgresql://localhost/memmie_schema?sslmode=disable"),
        RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
        NATSUrl:     getEnv("NATS_URL", "nats://localhost:4222"),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

## Task 13: Create Main Server
**File**: `cmd/server/main.go`
```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-schema/internal/config"
    "github.com/memmieai/memmie-schema/internal/repository"
    "github.com/memmieai/memmie-schema/internal/validator"
    "github.com/memmieai/memmie-schema/internal/cache"
    "github.com/memmieai/memmie-schema/internal/service"
    "github.com/memmieai/memmie-schema/internal/handler"
    "github.com/memmieai/memmie-common/pkg/logger"
)

func main() {
    // Load configuration
    cfg := config.LoadConfig()
    
    // Initialize logger
    log := logger.NewConsoleLogger("schema-service", logger.InfoLevel)
    
    // Initialize repository
    repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
    if err != nil {
        log.Fatal("Failed to connect to database", "error", err)
    }
    
    // Initialize validator
    validator := validator.NewJSONSchemaValidator()
    
    // Initialize cache
    cache := cache.NewRedisCache(cfg.RedisURL, "")
    
    // Initialize event bus (implement based on NATS)
    eventBus := setupEventBus(cfg.NATSUrl)
    
    // Initialize service
    svc := service.NewSchemaService(repo, validator, cache, eventBus, log)
    
    // Initialize HTTP handler
    httpHandler := handler.NewHTTPHandler(svc, log)
    
    // Setup routes
    router := mux.NewRouter()
    httpHandler.RegisterRoutes(router)
    
    // Start server
    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: router,
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
    
    log.Info("Schema service starting", "port", cfg.Port)
    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatal("Server failed", "error", err)
    }
}
```

## Task 14: Seed System Schemas
**File**: `migrations/002_seed_system_schemas.sql`
```sql
-- Insert system schemas
INSERT INTO schemas (processor_id, name, version, definition, status, description) VALUES
('system', 'bucket-v1', '1.0.0', '{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "type"],
  "properties": {
    "name": {"type": "string", "minLength": 1},
    "type": {"type": "string"},
    "metadata": {"type": "object"}
  }
}', 'active', 'Core bucket schema'),

('system', 'blob-metadata-v1', '1.0.0', '{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "title": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}}
  }
}', 'active', 'Blob metadata schema'),

('user-input', 'text-input-v1', '1.0.0', '{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["content"],
  "properties": {
    "content": {"type": "string", "minLength": 1, "maxLength": 50000},
    "style": {"type": "string", "enum": ["formal", "casual", "creative"]},
    "context": {"type": "string"}
  }
}', 'active', 'Text input schema for user content');
```

## Task 15: Create Integration Tests
**File**: `internal/service/integration_test.go`
```go
package service_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSchemaService_FullFlow(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    // Initialize service with real dependencies
    svc := setupService(db)
    
    // Test registration
    schema, err := svc.RegisterSchema(context.Background(), SchemaDefinition{
        ProcessorID: "test-processor",
        Name:        "test-schema",
        Version:     "1.0.0",
        Definition:  `{"type": "object", "properties": {"name": {"type": "string"}}}`,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, schema.ID)
    
    // Test validation with valid data
    result, err := svc.ValidateData(context.Background(), schema.ID, map[string]interface{}{
        "name": "test",
    })
    require.NoError(t, err)
    assert.True(t, result.Valid)
    
    // Test validation with invalid data
    result, err = svc.ValidateData(context.Background(), schema.ID, map[string]interface{}{
        "name": 123, // Wrong type
    })
    require.NoError(t, err)
    assert.False(t, result.Valid)
    assert.NotEmpty(t, result.Errors)
}
```

## Task 16: Create Dockerfile
**File**: `Dockerfile`
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o schema-service cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/schema-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8011
CMD ["./schema-service"]
```

## Task 17: Create Makefile
**File**: `Makefile`
```makefile
.PHONY: build run test migrate

build:
	go build -o bin/schema-service cmd/server/main.go

run:
	go run cmd/server/main.go

test:
	go test -v ./...

test-coverage:
	go test -v -cover ./...

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

docker-build:
	docker build -t memmie-schema:latest .

docker-run:
	docker run -p 8011:8011 --env-file .env memmie-schema:latest

lint:
	golangci-lint run

fmt:
	go fmt ./...

clean:
	rm -rf bin/
```

## Testing Checklist
- [ ] Unit tests for validator
- [ ] Unit tests for repository with mocks
- [ ] Unit tests for service with mocks
- [ ] Integration tests with real database
- [ ] HTTP handler tests
- [ ] Client library tests
- [ ] Load testing for validation endpoint
- [ ] Schema compilation performance test

## Success Criteria
- [ ] Service starts on port 8011
- [ ] Can register new schemas
- [ ] Can validate data against schemas
- [ ] Validation completes in <50ms
- [ ] 80% test coverage
- [ ] Handles 1000 validation requests/second