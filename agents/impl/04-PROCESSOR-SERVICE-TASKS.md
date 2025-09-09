# Processor Service Implementation Tasks

## Prerequisites
- Schema Service running on port 8011
- State Service with blob support on port 8006
- NATS running on port 4222

## Task 1: Transform Provider Service to Processor Service
**Note**: We're transforming the existing memmie-provider service
```bash
cd /home/uneid/iter3/memmieai/memmie-provider
# Service already exists, we'll refactor it
```

**Update go.mod dependencies**:
```bash
go get github.com/memmieai/memmie-schema/pkg/client
go get github.com/memmieai/memmie-state/pkg/client
```

## Task 2: Define Processor Models
**File**: `internal/models/processor.go`
```go
package models

import (
    "time"
)

type Processor struct {
    ID              string            `json:"id" db:"id"`
    Name            string            `json:"name" db:"name"`
    Description     string            `json:"description" db:"description"`
    InputSchemaID   string            `json:"input_schema_id" db:"input_schema_id"`
    OutputSchemaID  string            `json:"output_schema_id" db:"output_schema_id"`
    
    // Event configuration
    SubscribeEvents []string          `json:"subscribe_events" db:"subscribe_events"`
    EmitEvents      []string          `json:"emit_events" db:"emit_events"`
    
    // Processing configuration
    Config          ProcessorConfig   `json:"config" db:"config"`
    MaxConcurrency  int              `json:"max_concurrency" db:"max_concurrency"`
    TimeoutSeconds  int              `json:"timeout_seconds" db:"timeout_seconds"`
    RetryPolicy     RetryPolicy      `json:"retry_policy" db:"retry_policy"`
    
    // Status
    Active          bool             `json:"active" db:"active"`
    Version         string           `json:"version" db:"version"`
    
    CreatedAt       time.Time        `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

type ProcessorConfig struct {
    Type            string                 `json:"type"` // "ai", "transform", "aggregate"
    AIModel         string                 `json:"ai_model,omitempty"`
    Temperature     float64                `json:"temperature,omitempty"`
    MaxTokens       int                    `json:"max_tokens,omitempty"`
    CustomSettings  map[string]interface{} `json:"custom_settings,omitempty"`
}

type ProcessorInstance struct {
    ID              string                 `json:"id" db:"id"`
    ProcessorID     string                 `json:"processor_id" db:"processor_id"`
    UserID          string                 `json:"user_id" db:"user_id"`
    
    // User-specific configuration
    UserConfig      map[string]interface{} `json:"user_config" db:"user_config"`
    
    // Status
    Active          bool                   `json:"active" db:"active"`
    LastExecutedAt  *time.Time            `json:"last_executed_at" db:"last_executed_at"`
    ExecutionCount  int                   `json:"execution_count" db:"execution_count"`
    
    CreatedAt       time.Time             `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time             `json:"updated_at" db:"updated_at"`
}

type ProcessorExecution struct {
    ID              string                 `json:"id"`
    ProcessorID     string                 `json:"processor_id"`
    UserID          string                 `json:"user_id"`
    InputBlobID     string                 `json:"input_blob_id"`
    OutputBlobID    *string               `json:"output_blob_id,omitempty"`
    
    Status          ExecutionStatus        `json:"status"`
    Error           *string               `json:"error,omitempty"`
    
    StartedAt       time.Time             `json:"started_at"`
    CompletedAt     *time.Time            `json:"completed_at,omitempty"`
    DurationMs      int                   `json:"duration_ms"`
}

type ExecutionStatus string
const (
    ExecutionStatusPending   ExecutionStatus = "pending"
    ExecutionStatusRunning   ExecutionStatus = "running"
    ExecutionStatusCompleted ExecutionStatus = "completed"
    ExecutionStatusFailed    ExecutionStatus = "failed"
    ExecutionStatusRetrying  ExecutionStatus = "retrying"
)
```

## Task 3: Create Processor Registry
**File**: `internal/registry/processor_registry.go`
```go
package registry

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/memmieai/memmie-provider/internal/models"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type ProcessorRegistry struct {
    processors map[string]ProcessorExecutor
    configs    map[string]*models.Processor
    mu         sync.RWMutex
    logger     logger.Logger
}

type ProcessorExecutor interface {
    Execute(ctx context.Context, input ProcessorInput) (*ProcessorOutput, error)
    ValidateInput(data interface{}) error
    ValidateOutput(data interface{}) error
    GetInfo() ProcessorInfo
}

type ProcessorInput struct {
    UserID      string
    BlobID      string
    Data        interface{}
    Config      map[string]interface{}
    BucketIDs   []string
}

type ProcessorOutput struct {
    Data        interface{}
    Metadata    map[string]interface{}
    BucketIDs   []string
}

func NewProcessorRegistry(logger logger.Logger) *ProcessorRegistry {
    return &ProcessorRegistry{
        processors: make(map[string]ProcessorExecutor),
        configs:    make(map[string]*models.Processor),
        logger:     logger,
    }
}

func (r *ProcessorRegistry) Register(processor *models.Processor, executor ProcessorExecutor) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, exists := r.processors[processor.ID]; exists {
        return fmt.Errorf("processor %s already registered", processor.ID)
    }
    
    r.processors[processor.ID] = executor
    r.configs[processor.ID] = processor
    
    r.logger.Info("Processor registered", "id", processor.ID, "name", processor.Name)
    return nil
}

func (r *ProcessorRegistry) Get(processorID string) (ProcessorExecutor, *models.Processor, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    executor, ok := r.processors[processorID]
    if !ok {
        return nil, nil, fmt.Errorf("processor %s not found", processorID)
    }
    
    config := r.configs[processorID]
    return executor, config, nil
}

func (r *ProcessorRegistry) List() []*models.Processor {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    processors := make([]*models.Processor, 0, len(r.configs))
    for _, p := range r.configs {
        processors = append(processors, p)
    }
    return processors
}
```

## Task 4: Implement Text Expansion Processor
**File**: `internal/processors/text_expansion.go`
```go
package processors

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    
    "github.com/memmieai/memmie-provider/internal/registry"
    "github.com/memmieai/memmie-common/pkg/logger"
    openai "github.com/sashabaranov/go-openai"
)

type TextExpansionProcessor struct {
    openaiClient *openai.Client
    logger       logger.Logger
}

func NewTextExpansionProcessor(apiKey string, logger logger.Logger) *TextExpansionProcessor {
    return &TextExpansionProcessor{
        openaiClient: openai.NewClient(apiKey),
        logger:       logger,
    }
}

func (p *TextExpansionProcessor) Execute(ctx context.Context, input registry.ProcessorInput) (*registry.ProcessorOutput, error) {
    // Extract text content from input
    var inputData map[string]interface{}
    dataBytes, err := json.Marshal(input.Data)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal input data: %w", err)
    }
    
    if err := json.Unmarshal(dataBytes, &inputData); err != nil {
        return nil, fmt.Errorf("failed to unmarshal input data: %w", err)
    }
    
    content, ok := inputData["content"].(string)
    if !ok {
        return nil, fmt.Errorf("content field not found or not a string")
    }
    
    style := "creative"
    if s, ok := inputData["style"].(string); ok {
        style = s
    }
    
    // Prepare expansion prompt
    prompt := p.buildPrompt(content, style, input.Config)
    
    // Call OpenAI API
    resp, err := p.openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: openai.GPT4,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleSystem,
                Content: "You are a creative writing assistant that expands brief text into rich, detailed prose while maintaining the original meaning and intent.",
            },
            {
                Role:    openai.ChatMessageRoleUser,
                Content: prompt,
            },
        },
        Temperature: 0.7,
        MaxTokens:   2000,
    })
    
    if err != nil {
        return nil, fmt.Errorf("OpenAI API error: %w", err)
    }
    
    if len(resp.Choices) == 0 {
        return nil, fmt.Errorf("no response from OpenAI")
    }
    
    expandedText := resp.Choices[0].Message.Content
    
    // Calculate metrics
    originalWords := len(strings.Fields(content))
    expandedWords := len(strings.Fields(expandedText))
    expansionRatio := float64(expandedWords) / float64(originalWords)
    
    // Prepare output
    output := &registry.ProcessorOutput{
        Data: map[string]interface{}{
            "original":        content,
            "expanded":        expandedText,
            "expansion_ratio": expansionRatio,
            "style":          style,
            "word_count": map[string]int{
                "original": originalWords,
                "expanded": expandedWords,
            },
        },
        Metadata: map[string]interface{}{
            "processor": "text-expansion",
            "model":     "gpt-4",
            "style":     style,
        },
        BucketIDs: input.BucketIDs,
    }
    
    return output, nil
}

func (p *TextExpansionProcessor) buildPrompt(content, style string, config map[string]interface{}) string {
    var styleGuide string
    switch style {
    case "formal":
        styleGuide = "Use formal, professional language with sophisticated vocabulary."
    case "casual":
        styleGuide = "Use casual, conversational language that feels natural and approachable."
    case "creative":
        styleGuide = "Use creative, vivid language with rich descriptions and engaging narrative."
    case "technical":
        styleGuide = "Use precise, technical language with clear explanations."
    case "poetic":
        styleGuide = "Use poetic, lyrical language with metaphors and beautiful imagery."
    default:
        styleGuide = "Maintain a balanced, engaging writing style."
    }
    
    targetExpansion := "3-5 times"
    if ratio, ok := config["expansion_ratio"].(float64); ok {
        targetExpansion = fmt.Sprintf("%.1f times", ratio)
    }
    
    return fmt.Sprintf(`Expand the following text to approximately %s its original length. %s

Original text:
%s

Requirements:
1. Maintain the original meaning and key points
2. Add relevant details, descriptions, and context
3. Ensure smooth flow and readability
4. Keep the same narrative voice and tense
5. Do not add information that contradicts the original

Expanded text:`, targetExpansion, styleGuide, content)
}

func (p *TextExpansionProcessor) ValidateInput(data interface{}) error {
    dataMap, ok := data.(map[string]interface{})
    if !ok {
        return fmt.Errorf("input must be a map")
    }
    
    if _, ok := dataMap["content"].(string); !ok {
        return fmt.Errorf("content field is required and must be a string")
    }
    
    return nil
}

func (p *TextExpansionProcessor) ValidateOutput(data interface{}) error {
    dataMap, ok := data.(map[string]interface{})
    if !ok {
        return fmt.Errorf("output must be a map")
    }
    
    required := []string{"original", "expanded", "expansion_ratio"}
    for _, field := range required {
        if _, ok := dataMap[field]; !ok {
            return fmt.Errorf("required field %s not found", field)
        }
    }
    
    return nil
}

func (p *TextExpansionProcessor) GetInfo() registry.ProcessorInfo {
    return registry.ProcessorInfo{
        ID:          "text-expansion",
        Name:        "Text Expansion Processor",
        Description: "Expands brief text into detailed prose using AI",
        Version:     "1.0.0",
    }
}
```

## Task 5: Create Processor Worker Pool
**File**: `internal/worker/worker_pool.go`
```go
package worker

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/memmieai/memmie-provider/internal/models"
    "github.com/memmieai/memmie-provider/internal/registry"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type WorkerPool struct {
    registry     *registry.ProcessorRegistry
    stateClient  StateClient
    schemaClient SchemaClient
    workers      int
    jobQueue     chan *Job
    wg           sync.WaitGroup
    logger       logger.Logger
}

type Job struct {
    ProcessorID string
    UserID      string
    InputBlobID string
    Context     context.Context
}

func NewWorkerPool(
    registry *registry.ProcessorRegistry,
    stateClient StateClient,
    schemaClient SchemaClient,
    workers int,
    logger logger.Logger,
) *WorkerPool {
    return &WorkerPool{
        registry:     registry,
        stateClient:  stateClient,
        schemaClient: schemaClient,
        workers:      workers,
        jobQueue:     make(chan *Job, workers*10),
        logger:       logger,
    }
}

func (p *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(ctx, i)
    }
    
    p.logger.Info("Worker pool started", "workers", p.workers)
}

func (p *WorkerPool) Stop() {
    close(p.jobQueue)
    p.wg.Wait()
    p.logger.Info("Worker pool stopped")
}

func (p *WorkerPool) Submit(job *Job) error {
    select {
    case p.jobQueue <- job:
        return nil
    default:
        return fmt.Errorf("job queue is full")
    }
}

func (p *WorkerPool) worker(ctx context.Context, id int) {
    defer p.wg.Done()
    
    p.logger.Info("Worker started", "id", id)
    
    for {
        select {
        case <-ctx.Done():
            p.logger.Info("Worker stopping", "id", id)
            return
            
        case job, ok := <-p.jobQueue:
            if !ok {
                p.logger.Info("Worker stopped", "id", id)
                return
            }
            
            p.processJob(job)
        }
    }
}

func (p *WorkerPool) processJob(job *Job) {
    startTime := time.Now()
    
    // Create execution record
    execution := &models.ProcessorExecution{
        ProcessorID:  job.ProcessorID,
        UserID:       job.UserID,
        InputBlobID:  job.InputBlobID,
        Status:       models.ExecutionStatusRunning,
        StartedAt:    startTime,
    }
    
    // Get processor
    executor, config, err := p.registry.Get(job.ProcessorID)
    if err != nil {
        p.handleError(execution, err)
        return
    }
    
    // Get input blob
    inputBlob, err := p.stateClient.GetBlob(job.Context, job.InputBlobID)
    if err != nil {
        p.handleError(execution, err)
        return
    }
    
    // Validate input against schema
    if err := p.schemaClient.ValidateData(job.Context, config.InputSchemaID, inputBlob.Data); err != nil {
        p.handleError(execution, fmt.Errorf("input validation failed: %w", err))
        return
    }
    
    // Execute processor
    input := registry.ProcessorInput{
        UserID:    job.UserID,
        BlobID:    job.InputBlobID,
        Data:      inputBlob.Data,
        Config:    make(map[string]interface{}), // Load user config if needed
        BucketIDs: inputBlob.BucketIDs,
    }
    
    output, err := executor.Execute(job.Context, input)
    if err != nil {
        p.handleError(execution, err)
        return
    }
    
    // Validate output against schema
    if err := p.schemaClient.ValidateData(job.Context, config.OutputSchemaID, output.Data); err != nil {
        p.handleError(execution, fmt.Errorf("output validation failed: %w", err))
        return
    }
    
    // Create output blob
    outputBlob, err := p.stateClient.CreateBlob(job.Context, job.UserID, CreateBlobRequest{
        ProcessorID: job.ProcessorID,
        SchemaID:    config.OutputSchemaID,
        Data:        output.Data,
        BucketIDs:   output.BucketIDs,
        ParentID:    &job.InputBlobID,
        Metadata:    output.Metadata,
    })
    
    if err != nil {
        p.handleError(execution, fmt.Errorf("failed to create output blob: %w", err))
        return
    }
    
    // Update execution record
    completedAt := time.Now()
    execution.Status = models.ExecutionStatusCompleted
    execution.OutputBlobID = &outputBlob.ID
    execution.CompletedAt = &completedAt
    execution.DurationMs = int(completedAt.Sub(startTime).Milliseconds())
    
    // Publish completion event
    p.publishCompletionEvent(execution)
    
    p.logger.Info("Job processed successfully",
        "processor", job.ProcessorID,
        "user", job.UserID,
        "duration_ms", execution.DurationMs,
    )
}

func (p *WorkerPool) handleError(execution *models.ProcessorExecution, err error) {
    errorStr := err.Error()
    execution.Status = models.ExecutionStatusFailed
    execution.Error = &errorStr
    
    p.logger.Error("Job processing failed",
        "processor", execution.ProcessorID,
        "user", execution.UserID,
        "error", err,
    )
    
    // Publish failure event
    p.publishFailureEvent(execution)
}
```

## Task 6: Create Event Listener for Processing
**File**: `internal/events/processor_listener.go`
```go
package events

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/memmieai/memmie-provider/internal/worker"
    "github.com/memmieai/memmie-common/pkg/logger"
    "github.com/nats-io/nats.go"
)

type ProcessorListener struct {
    natsConn   *nats.Conn
    workerPool *worker.WorkerPool
    processors map[string]ProcessorSubscription
    logger     logger.Logger
}

type ProcessorSubscription struct {
    ProcessorID string
    SchemaID    string
    Subscription *nats.Subscription
}

func NewProcessorListener(
    natsConn *nats.Conn,
    workerPool *worker.WorkerPool,
    logger logger.Logger,
) *ProcessorListener {
    return &ProcessorListener{
        natsConn:   natsConn,
        workerPool: workerPool,
        processors: make(map[string]ProcessorSubscription),
        logger:     logger,
    }
}

func (l *ProcessorListener) RegisterProcessor(processorID, inputSchemaID string) error {
    topic := fmt.Sprintf("blob.created.%s", inputSchemaID)
    
    sub, err := l.natsConn.Subscribe(topic, func(msg *nats.Msg) {
        l.handleBlobCreated(processorID, msg.Data)
    })
    
    if err != nil {
        return fmt.Errorf("failed to subscribe to %s: %w", topic, err)
    }
    
    l.processors[processorID] = ProcessorSubscription{
        ProcessorID:  processorID,
        SchemaID:     inputSchemaID,
        Subscription: sub,
    }
    
    l.logger.Info("Processor registered for events", 
        "processor", processorID, 
        "schema", inputSchemaID,
        "topic", topic,
    )
    
    return nil
}

func (l *ProcessorListener) handleBlobCreated(processorID string, data []byte) {
    var event struct {
        BlobID      string `json:"blob_id"`
        UserID      string `json:"user_id"`
        ProcessorID string `json:"processor_id"`
        SchemaID    string `json:"schema_id"`
    }
    
    if err := json.Unmarshal(data, &event); err != nil {
        l.logger.Error("Failed to unmarshal event", "error", err)
        return
    }
    
    // Don't process our own outputs (avoid loops)
    if event.ProcessorID == processorID {
        return
    }
    
    // Submit job to worker pool
    job := &worker.Job{
        ProcessorID: processorID,
        UserID:      event.UserID,
        InputBlobID: event.BlobID,
        Context:     context.Background(),
    }
    
    if err := l.workerPool.Submit(job); err != nil {
        l.logger.Error("Failed to submit job", "error", err)
    }
}

func (l *ProcessorListener) UnregisterProcessor(processorID string) error {
    if sub, ok := l.processors[processorID]; ok {
        if err := sub.Subscription.Unsubscribe(); err != nil {
            return err
        }
        delete(l.processors, processorID)
    }
    return nil
}
```

## Task 7: Create Processor Service
**File**: `internal/service/processor_service.go`
```go
package service

import (
    "context"
    "fmt"
    
    "github.com/memmieai/memmie-provider/internal/events"
    "github.com/memmieai/memmie-provider/internal/models"
    "github.com/memmieai/memmie-provider/internal/registry"
    "github.com/memmieai/memmie-provider/internal/repository"
    "github.com/memmieai/memmie-provider/internal/worker"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type ProcessorService struct {
    repo         repository.ProcessorRepository
    registry     *registry.ProcessorRegistry
    workerPool   *worker.WorkerPool
    listener     *events.ProcessorListener
    logger       logger.Logger
}

func NewProcessorService(
    repo repository.ProcessorRepository,
    registry *registry.ProcessorRegistry,
    workerPool *worker.WorkerPool,
    listener *events.ProcessorListener,
    logger logger.Logger,
) *ProcessorService {
    return &ProcessorService{
        repo:       repo,
        registry:   registry,
        workerPool: workerPool,
        listener:   listener,
        logger:     logger,
    }
}

func (s *ProcessorService) RegisterProcessor(ctx context.Context, req RegisterProcessorRequest) (*models.Processor, error) {
    processor := &models.Processor{
        ID:              req.ID,
        Name:            req.Name,
        Description:     req.Description,
        InputSchemaID:   req.InputSchemaID,
        OutputSchemaID:  req.OutputSchemaID,
        SubscribeEvents: []string{fmt.Sprintf("blob.created.%s", req.InputSchemaID)},
        EmitEvents:      []string{fmt.Sprintf("blob.created.%s", req.OutputSchemaID)},
        Config:          req.Config,
        MaxConcurrency:  req.MaxConcurrency,
        TimeoutSeconds:  req.TimeoutSeconds,
        Active:          true,
        Version:         req.Version,
    }
    
    // Save to database
    if err := s.repo.Create(ctx, processor); err != nil {
        return nil, fmt.Errorf("failed to save processor: %w", err)
    }
    
    // Register executor if provided
    if req.Executor != nil {
        if err := s.registry.Register(processor, req.Executor); err != nil {
            return nil, fmt.Errorf("failed to register executor: %w", err)
        }
        
        // Start listening for events
        if err := s.listener.RegisterProcessor(processor.ID, processor.InputSchemaID); err != nil {
            return nil, fmt.Errorf("failed to register event listener: %w", err)
        }
    }
    
    s.logger.Info("Processor registered", "id", processor.ID, "name", processor.Name)
    return processor, nil
}

func (s *ProcessorService) CreateUserInstance(ctx context.Context, userID, processorID string, config map[string]interface{}) (*models.ProcessorInstance, error) {
    // Verify processor exists
    processor, err := s.repo.GetByID(ctx, processorID)
    if err != nil {
        return nil, fmt.Errorf("processor not found: %w", err)
    }
    
    instance := &models.ProcessorInstance{
        ProcessorID: processorID,
        UserID:      userID,
        UserConfig:  config,
        Active:      true,
    }
    
    if err := s.repo.CreateInstance(ctx, instance); err != nil {
        return nil, fmt.Errorf("failed to create instance: %w", err)
    }
    
    return instance, nil
}

func (s *ProcessorService) ExecuteProcessor(ctx context.Context, userID, processorID, blobID string) error {
    // Verify processor exists and is active
    processor, err := s.repo.GetByID(ctx, processorID)
    if err != nil {
        return fmt.Errorf("processor not found: %w", err)
    }
    
    if !processor.Active {
        return fmt.Errorf("processor is not active")
    }
    
    // Submit job to worker pool
    job := &worker.Job{
        ProcessorID: processorID,
        UserID:      userID,
        InputBlobID: blobID,
        Context:     ctx,
    }
    
    return s.workerPool.Submit(job)
}

func (s *ProcessorService) ListProcessors(ctx context.Context) ([]*models.Processor, error) {
    return s.repo.List(ctx)
}

func (s *ProcessorService) GetProcessorStatus(ctx context.Context, processorID string) (*ProcessorStatus, error) {
    processor, err := s.repo.GetByID(ctx, processorID)
    if err != nil {
        return nil, err
    }
    
    // Get execution stats
    stats, err := s.repo.GetExecutionStats(ctx, processorID)
    if err != nil {
        return nil, err
    }
    
    return &ProcessorStatus{
        Processor: processor,
        Stats:     stats,
    }, nil
}
```

## Task 8: Initialize Built-in Processors
**File**: `internal/processors/init.go`
```go
package processors

import (
    "os"
    
    "github.com/memmieai/memmie-provider/internal/models"
    "github.com/memmieai/memmie-provider/internal/service"
    "github.com/memmieai/memmie-common/pkg/logger"
)

func InitializeBuiltinProcessors(svc *service.ProcessorService, logger logger.Logger) error {
    // Text Expansion Processor
    openaiKey := os.Getenv("OPENAI_API_KEY")
    if openaiKey != "" {
        textExpander := NewTextExpansionProcessor(openaiKey, logger)
        
        _, err := svc.RegisterProcessor(context.Background(), service.RegisterProcessorRequest{
            ID:             "text-expansion",
            Name:           "Text Expansion",
            Description:    "Expands brief text into detailed prose",
            InputSchemaID:  "text-input-v1",
            OutputSchemaID: "expanded-text-v1",
            Config: models.ProcessorConfig{
                Type:        "ai",
                AIModel:     "gpt-4",
                Temperature: 0.7,
                MaxTokens:   2000,
            },
            MaxConcurrency: 10,
            TimeoutSeconds: 30,
            Version:        "1.0.0",
            Executor:       textExpander,
        })
        
        if err != nil {
            return fmt.Errorf("failed to register text expansion processor: %w", err)
        }
    }
    
    // Grammar Check Processor (placeholder)
    // grammarChecker := NewGrammarCheckProcessor(logger)
    // svc.RegisterProcessor(...)
    
    // Style Analyzer Processor (placeholder)
    // styleAnalyzer := NewStyleAnalyzerProcessor(logger)
    // svc.RegisterProcessor(...)
    
    // Book Compiler Processor (placeholder)
    // bookCompiler := NewBookCompilerProcessor(logger)
    // svc.RegisterProcessor(...)
    
    logger.Info("Built-in processors initialized")
    return nil
}
```

## Task 9: Create Repository Layer
**File**: `internal/repository/postgres.go`
```go
package repository

import (
    "context"
    "database/sql"
    "encoding/json"
    
    _ "github.com/lib/pq"
    "github.com/memmieai/memmie-provider/internal/models"
)

type PostgresRepository struct {
    db *sql.DB
}

func NewPostgresRepository(connStr string) (*PostgresRepository, error) {
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }
    
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Create(ctx context.Context, processor *models.Processor) error {
    subscribeEvents, _ := json.Marshal(processor.SubscribeEvents)
    emitEvents, _ := json.Marshal(processor.EmitEvents)
    config, _ := json.Marshal(processor.Config)
    
    query := `
        INSERT INTO processors (
            id, name, description, input_schema_id, output_schema_id,
            subscribe_events, emit_events, config, max_concurrency,
            timeout_seconds, active, version
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        RETURNING created_at, updated_at
    `
    
    err := r.db.QueryRowContext(ctx, query,
        processor.ID, processor.Name, processor.Description,
        processor.InputSchemaID, processor.OutputSchemaID,
        subscribeEvents, emitEvents, config,
        processor.MaxConcurrency, processor.TimeoutSeconds,
        processor.Active, processor.Version,
    ).Scan(&processor.CreatedAt, &processor.UpdatedAt)
    
    return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*models.Processor, error) {
    processor := &models.Processor{}
    var subscribeEvents, emitEvents, config []byte
    
    query := `
        SELECT id, name, description, input_schema_id, output_schema_id,
               subscribe_events, emit_events, config, max_concurrency,
               timeout_seconds, active, version, created_at, updated_at
        FROM processors
        WHERE id = $1
    `
    
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &processor.ID, &processor.Name, &processor.Description,
        &processor.InputSchemaID, &processor.OutputSchemaID,
        &subscribeEvents, &emitEvents, &config,
        &processor.MaxConcurrency, &processor.TimeoutSeconds,
        &processor.Active, &processor.Version,
        &processor.CreatedAt, &processor.UpdatedAt,
    )
    
    if err != nil {
        return nil, err
    }
    
    json.Unmarshal(subscribeEvents, &processor.SubscribeEvents)
    json.Unmarshal(emitEvents, &processor.EmitEvents)
    json.Unmarshal(config, &processor.Config)
    
    return processor, nil
}

func (r *PostgresRepository) CreateInstance(ctx context.Context, instance *models.ProcessorInstance) error {
    userConfig, _ := json.Marshal(instance.UserConfig)
    
    query := `
        INSERT INTO processor_instances (processor_id, user_id, user_config, active)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at
    `
    
    err := r.db.QueryRowContext(ctx, query,
        instance.ProcessorID, instance.UserID, userConfig, instance.Active,
    ).Scan(&instance.ID, &instance.CreatedAt, &instance.UpdatedAt)
    
    return err
}
```

## Task 10: Create Database Migrations
**File**: `migrations/001_create_processors_tables.sql`
```sql
-- Processors table
CREATE TABLE IF NOT EXISTS processors (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    input_schema_id VARCHAR(255) NOT NULL,
    output_schema_id VARCHAR(255) NOT NULL,
    subscribe_events JSONB,
    emit_events JSONB,
    config JSONB,
    max_concurrency INT DEFAULT 1,
    timeout_seconds INT DEFAULT 30,
    active BOOLEAN DEFAULT true,
    version VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Processor instances table
CREATE TABLE IF NOT EXISTS processor_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    processor_id VARCHAR(255) REFERENCES processors(id),
    user_id VARCHAR(255) NOT NULL,
    user_config JSONB,
    active BOOLEAN DEFAULT true,
    last_executed_at TIMESTAMP,
    execution_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(processor_id, user_id)
);

-- Execution history table
CREATE TABLE IF NOT EXISTS processor_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    processor_id VARCHAR(255) REFERENCES processors(id),
    user_id VARCHAR(255) NOT NULL,
    input_blob_id VARCHAR(255) NOT NULL,
    output_blob_id VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    error TEXT,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    duration_ms INT,
    
    INDEX idx_executions_processor (processor_id, started_at),
    INDEX idx_executions_user (user_id, started_at),
    INDEX idx_executions_status (status)
);
```

## Task 11: Update Main Server
**File**: `cmd/server/main.go`
```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-provider/internal/config"
    "github.com/memmieai/memmie-provider/internal/events"
    "github.com/memmieai/memmie-provider/internal/handler"
    "github.com/memmieai/memmie-provider/internal/processors"
    "github.com/memmieai/memmie-provider/internal/registry"
    "github.com/memmieai/memmie-provider/internal/repository"
    "github.com/memmieai/memmie-provider/internal/service"
    "github.com/memmieai/memmie-provider/internal/worker"
    "github.com/memmieai/memmie-common/pkg/logger"
    "github.com/nats-io/nats.go"
)

func main() {
    cfg := config.LoadConfig()
    log := logger.NewConsoleLogger("processor-service", logger.InfoLevel)
    
    // Initialize repository
    repo, err := repository.NewPostgresRepository(cfg.DatabaseURL)
    if err != nil {
        log.Fatal("Failed to connect to database", "error", err)
    }
    
    // Initialize processor registry
    processorRegistry := registry.NewProcessorRegistry(log)
    
    // Initialize clients
    stateClient := clients.NewStateClient(cfg.StateServiceURL)
    schemaClient := clients.NewSchemaClient(cfg.SchemaServiceURL)
    
    // Initialize worker pool
    workerPool := worker.NewWorkerPool(
        processorRegistry,
        stateClient,
        schemaClient,
        cfg.WorkerCount,
        log,
    )
    
    // Connect to NATS
    natsConn, err := nats.Connect(cfg.NATSUrl)
    if err != nil {
        log.Fatal("Failed to connect to NATS", "error", err)
    }
    defer natsConn.Close()
    
    // Initialize event listener
    listener := events.NewProcessorListener(natsConn, workerPool, log)
    
    // Initialize service
    processorService := service.NewProcessorService(
        repo,
        processorRegistry,
        workerPool,
        listener,
        log,
    )
    
    // Initialize built-in processors
    if err := processors.InitializeBuiltinProcessors(processorService, log); err != nil {
        log.Error("Failed to initialize built-in processors", "error", err)
    }
    
    // Start worker pool
    workerPool.Start(context.Background())
    
    // Initialize HTTP handler
    httpHandler := handler.NewHTTPHandler(processorService, log)
    
    // Setup routes
    router := mux.NewRouter()
    httpHandler.RegisterRoutes(router)
    
    // Start HTTP server
    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: router,
    }
    
    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan
        
        log.Info("Shutting down processor service")
        workerPool.Stop()
        srv.Shutdown(context.Background())
    }()
    
    log.Info("Processor service starting", "port", cfg.Port)
    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Fatal("Server failed", "error", err)
    }
}
```

## Testing Checklist
- [ ] Processor registration works
- [ ] Text expansion processor executes
- [ ] Event subscription triggers processing
- [ ] Worker pool handles concurrent jobs
- [ ] Schema validation on input/output
- [ ] Derived blobs created correctly
- [ ] Error handling and retries
- [ ] Performance under load

## Success Criteria
- [ ] Can register new processors
- [ ] Text expansion works end-to-end
- [ ] Processes blobs within 3 seconds
- [ ] Handles 100 concurrent jobs
- [ ] Graceful shutdown of workers
- [ ] Events trigger processing automatically