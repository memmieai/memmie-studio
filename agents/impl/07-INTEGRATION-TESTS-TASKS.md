# Integration Test Tasks

## Overview
Integration tests to validate the complete MVP functionality across all services.

## Task 1: Setup Test Infrastructure

### Files to Create:
- `tests/integration/setup/docker-compose.test.yml`
- `tests/integration/setup/test-data.sql`
- `tests/integration/setup/test-env.sh`

### Implementation:
```yaml
# docker-compose.test.yml
version: '3.8'
services:
  postgres-test:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: test_schemas
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
    ports:
      - "5433:5432"
  
  mongodb-test:
    image: mongo:6
    environment:
      MONGO_INITDB_DATABASE: test_state
      MONGO_INITDB_ROOT_USERNAME: test
      MONGO_INITDB_ROOT_PASSWORD: test
    ports:
      - "27018:27017"
  
  nats-test:
    image: nats:latest
    ports:
      - "4223:4222"
```

### Acceptance Criteria:
- Test databases spin up cleanly
- Test data loads successfully
- Environment isolation from development

---

## Task 2: End-to-End User Registration and Login Test

### Files to Create:
- `tests/integration/auth/registration_test.go`
- `tests/integration/auth/login_test.go`

### Implementation:
```go
func TestUserRegistrationFlow(t *testing.T) {
    // Setup
    client := setupTestClient()
    
    // Register user
    regReq := RegisterRequest{
        Email:    "test@example.com",
        Password: "Test123!",
        Username: "testuser",
    }
    
    regResp, err := client.Register(regReq)
    assert.NoError(t, err)
    assert.NotEmpty(t, regResp.UserID)
    assert.NotEmpty(t, regResp.Token)
    
    // Verify user can login
    loginReq := LoginRequest{
        Email:    "test@example.com",
        Password: "Test123!",
    }
    
    loginResp, err := client.Login(loginReq)
    assert.NoError(t, err)
    assert.Equal(t, regResp.UserID, loginResp.UserID)
    
    // Verify token is valid
    user, err := client.ValidateToken(loginResp.Token)
    assert.NoError(t, err)
    assert.Equal(t, "test@example.com", user.Email)
}
```

### Acceptance Criteria:
- User can register with email/password
- User receives valid JWT token
- User can login with credentials
- Token validation works correctly

---

## Task 3: Schema Registration and Validation Test

### Files to Create:
- `tests/integration/schema/registration_test.go`
- `tests/integration/schema/validation_test.go`

### Implementation:
```go
func TestSchemaLifecycle(t *testing.T) {
    client := setupSchemaClient()
    
    // Register book schema
    bookSchema := SchemaDefinition{
        Name:    "book",
        Version: "1.0.0",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "title": map[string]interface{}{
                    "type": "string",
                },
                "chapters": map[string]interface{}{
                    "type": "array",
                    "items": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "title":   map[string]interface{}{"type": "string"},
                            "content": map[string]interface{}{"type": "string"},
                        },
                    },
                },
            },
            "required": []string{"title"},
        },
    }
    
    schema, err := client.RegisterSchema(bookSchema)
    assert.NoError(t, err)
    assert.NotEmpty(t, schema.ID)
    
    // Validate valid data
    validData := map[string]interface{}{
        "title": "My Book",
        "chapters": []map[string]interface{}{
            {
                "title":   "Chapter 1",
                "content": "Content here",
            },
        },
    }
    
    result, err := client.ValidateData(schema.ID, validData)
    assert.NoError(t, err)
    assert.True(t, result.IsValid)
    
    // Validate invalid data
    invalidData := map[string]interface{}{
        "chapters": "not an array",
    }
    
    result, err = client.ValidateData(schema.ID, invalidData)
    assert.NoError(t, err)
    assert.False(t, result.IsValid)
    assert.Contains(t, result.Errors[0], "title is required")
}
```

### Acceptance Criteria:
- Schema registration succeeds
- Valid data passes validation
- Invalid data fails with clear errors
- Schema versioning works

---

## Task 4: Blob and Bucket Operations Test

### Files to Create:
- `tests/integration/state/blob_test.go`
- `tests/integration/state/bucket_test.go`

### Implementation:
```go
func TestBlobBucketOperations(t *testing.T) {
    client := setupStateClient()
    userID := "test-user-123"
    
    // Create bucket hierarchy
    rootBucket, err := client.CreateBucket(CreateBucketRequest{
        UserID: userID,
        Name:   "My Books",
        Type:   "library",
    })
    assert.NoError(t, err)
    
    bookBucket, err := client.CreateBucket(CreateBucketRequest{
        UserID:   userID,
        Name:     "Adventure Novel",
        Type:     "book",
        ParentID: &rootBucket.ID,
    })
    assert.NoError(t, err)
    
    // Create blob in bucket
    blob, err := client.CreateBlob(CreateBlobRequest{
        UserID:      userID,
        ProcessorID: "text-processor",
        SchemaID:    "book-schema-v1",
        Data: map[string]interface{}{
            "title": "Adventure Novel",
            "chapters": []map[string]interface{}{
                {
                    "title":   "The Beginning",
                    "content": "It was a dark and stormy night...",
                },
            },
        },
        BucketIDs: []string{bookBucket.ID},
    })
    assert.NoError(t, err)
    assert.NotEmpty(t, blob.ID)
    
    // Query blobs in bucket
    blobs, err := client.QueryBlobs(QueryBlobsRequest{
        UserID:   userID,
        BucketID: bookBucket.ID,
    })
    assert.NoError(t, err)
    assert.Len(t, blobs, 1)
    assert.Equal(t, blob.ID, blobs[0].ID)
    
    // Move blob to different bucket
    anotherBucket, _ := client.CreateBucket(CreateBucketRequest{
        UserID:   userID,
        Name:     "Archived",
        Type:     "archive",
        ParentID: &rootBucket.ID,
    })
    
    err = client.MoveBlobToBucket(blob.ID, bookBucket.ID, anotherBucket.ID)
    assert.NoError(t, err)
    
    // Verify move
    blobs, _ = client.QueryBlobs(QueryBlobsRequest{
        UserID:   userID,
        BucketID: anotherBucket.ID,
    })
    assert.Len(t, blobs, 1)
}
```

### Acceptance Criteria:
- Bucket hierarchy creation works
- Blobs can be created with schema validation
- Blobs can be queried by bucket
- Blobs can be moved between buckets

---

## Task 5: Processor Execution Test

### Files to Create:
- `tests/integration/processor/text_expansion_test.go`
- `tests/integration/processor/event_flow_test.go`

### Implementation:
```go
func TestProcessorExecution(t *testing.T) {
    processorClient := setupProcessorClient()
    stateClient := setupStateClient()
    eventBus := setupEventBus()
    
    userID := "test-user-123"
    
    // Register processor
    processor, err := processorClient.RegisterProcessor(RegisterProcessorRequest{
        Name:         "test-expander",
        Type:         "text-expansion",
        InputSchema:  "outline-schema-v1",
        OutputSchema: "chapter-schema-v1",
        Config: map[string]interface{}{
            "model": "gpt-3.5-turbo",
        },
    })
    assert.NoError(t, err)
    
    // Create input blob
    inputBlob, err := stateClient.CreateBlob(CreateBlobRequest{
        UserID:      userID,
        ProcessorID: processor.ID,
        SchemaID:    "outline-schema-v1",
        Data: map[string]interface{}{
            "title":   "Chapter 1",
            "outline": "Hero discovers mysterious artifact",
        },
    })
    assert.NoError(t, err)
    
    // Subscribe to completion events
    completeChan := make(chan ProcessingCompleteEvent)
    eventBus.Subscribe("processing.complete", func(event ProcessingCompleteEvent) {
        if event.InputBlobID == inputBlob.ID {
            completeChan <- event
        }
    })
    
    // Execute processor
    job, err := processorClient.Execute(ExecuteRequest{
        ProcessorID: processor.ID,
        InputBlobID: inputBlob.ID,
        UserID:      userID,
    })
    assert.NoError(t, err)
    assert.Equal(t, "pending", job.Status)
    
    // Wait for completion
    select {
    case event := <-completeChan:
        assert.Equal(t, "completed", event.Status)
        assert.NotEmpty(t, event.OutputBlobID)
        
        // Verify output blob
        outputBlob, err := stateClient.GetBlob(event.OutputBlobID)
        assert.NoError(t, err)
        assert.Equal(t, "chapter-schema-v1", outputBlob.SchemaID)
        
        data := outputBlob.Data.(map[string]interface{})
        assert.Contains(t, data["content"].(string), "artifact")
        
    case <-time.After(30 * time.Second):
        t.Fatal("Processor execution timeout")
    }
}
```

### Acceptance Criteria:
- Processor registration works
- Processor execution creates jobs
- Events are published correctly
- Output blobs are created with correct schema

---

## Task 6: WebSocket Real-time Updates Test

### Files to Create:
- `tests/integration/websocket/connection_test.go`
- `tests/integration/websocket/updates_test.go`

### Implementation:
```go
func TestWebSocketUpdates(t *testing.T) {
    // Setup WebSocket client
    ws, err := websocket.Dial("ws://localhost:8000/ws", "", "http://localhost/")
    assert.NoError(t, err)
    defer ws.Close()
    
    userID := "test-user-123"
    token := getTestToken(userID)
    
    // Authenticate
    authMsg := WSMessage{
        Type: "auth",
        Data: map[string]interface{}{
            "token": token,
        },
    }
    err = websocket.JSON.Send(ws, authMsg)
    assert.NoError(t, err)
    
    // Subscribe to bucket
    bucketID := "test-bucket-123"
    subMsg := WSMessage{
        Type: "subscribe",
        Data: map[string]interface{}{
            "bucket_id": bucketID,
        },
    }
    err = websocket.JSON.Send(ws, subMsg)
    assert.NoError(t, err)
    
    // Create blob in another connection
    go func() {
        time.Sleep(1 * time.Second)
        stateClient := setupStateClient()
        stateClient.CreateBlob(CreateBlobRequest{
            UserID:    userID,
            BucketIDs: []string{bucketID},
            Data: map[string]interface{}{
                "test": "data",
            },
        })
    }()
    
    // Receive update
    var update WSMessage
    err = websocket.JSON.Receive(ws, &update)
    assert.NoError(t, err)
    assert.Equal(t, "blob.created", update.Type)
    assert.Equal(t, bucketID, update.Data["bucket_id"])
}
```

### Acceptance Criteria:
- WebSocket connection establishes
- Authentication works
- Bucket subscriptions work
- Real-time updates received

---

## Task 7: Book Creation and Export Test

### Files to Create:
- `tests/integration/book/creation_test.go`
- `tests/integration/book/export_test.go`

### Implementation:
```go
func TestBookCreationAndExport(t *testing.T) {
    client := setupStudioClient()
    userID := "test-user-123"
    
    // Create book
    book, err := client.CreateBook(CreateBookRequest{
        UserID: userID,
        Title:  "My Adventure",
        Metadata: map[string]interface{}{
            "genre":  "fantasy",
            "author": "Test Author",
        },
    })
    assert.NoError(t, err)
    
    // Add chapters
    chapters := []Chapter{
        {
            Title:   "The Beginning",
            Content: "Once upon a time...",
            Order:   1,
        },
        {
            Title:   "The Journey",
            Content: "The hero set forth...",
            Order:   2,
        },
        {
            Title:   "The End",
            Content: "And they lived happily ever after.",
            Order:   3,
        },
    }
    
    for _, chapter := range chapters {
        _, err := client.AddChapter(book.ID, chapter)
        assert.NoError(t, err)
    }
    
    // Export as text
    exportReq := ExportBookRequest{
        BookID: book.ID,
        Format: "text",
    }
    
    result, err := client.ExportBook(exportReq)
    assert.NoError(t, err)
    
    // Verify export content
    content := result.Content
    assert.Contains(t, content, "My Adventure")
    assert.Contains(t, content, "Chapter 1: The Beginning")
    assert.Contains(t, content, "Once upon a time...")
    assert.Contains(t, content, "Chapter 2: The Journey")
    assert.Contains(t, content, "Chapter 3: The End")
    
    // Test markdown export
    exportReq.Format = "markdown"
    result, err = client.ExportBook(exportReq)
    assert.NoError(t, err)
    assert.Contains(t, result.Content, "# My Adventure")
    assert.Contains(t, result.Content, "## Chapter 1: The Beginning")
}
```

### Acceptance Criteria:
- Book creation with metadata
- Chapter management (add, update, reorder)
- Text export with proper formatting
- Markdown export with headers

---

## Task 8: Performance and Load Test

### Files to Create:
- `tests/integration/performance/load_test.go`
- `tests/integration/performance/metrics.go`

### Implementation:
```go
func TestConcurrentOperations(t *testing.T) {
    client := setupStudioClient()
    
    // Test concurrent user operations
    numUsers := 50
    numOpsPerUser := 10
    
    var wg sync.WaitGroup
    errors := make(chan error, numUsers*numOpsPerUser)
    
    start := time.Now()
    
    for i := 0; i < numUsers; i++ {
        wg.Add(1)
        go func(userNum int) {
            defer wg.Done()
            
            userID := fmt.Sprintf("user-%d", userNum)
            
            // Each user creates books and blobs
            for j := 0; j < numOpsPerUser; j++ {
                // Create book
                book, err := client.CreateBook(CreateBookRequest{
                    UserID: userID,
                    Title:  fmt.Sprintf("Book %d", j),
                })
                if err != nil {
                    errors <- err
                    continue
                }
                
                // Create blob
                _, err = client.CreateBlob(CreateBlobRequest{
                    UserID: userID,
                    Data: map[string]interface{}{
                        "book_id": book.ID,
                        "content": "Test content",
                    },
                })
                if err != nil {
                    errors <- err
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    duration := time.Since(start)
    
    // Check errors
    var errorCount int
    for err := range errors {
        errorCount++
        t.Logf("Error: %v", err)
    }
    
    // Performance assertions
    assert.Less(t, errorCount, numUsers) // Allow some errors but not all
    assert.Less(t, duration, 30*time.Second) // Should complete within 30 seconds
    
    opsPerSecond := float64(numUsers*numOpsPerUser) / duration.Seconds()
    t.Logf("Performance: %.2f ops/second", opsPerSecond)
    assert.Greater(t, opsPerSecond, 10.0) // At least 10 ops/second
}
```

### Acceptance Criteria:
- System handles 50 concurrent users
- Operations complete within timeout
- Error rate below 10%
- Throughput above minimum threshold

---

## Task 9: Error Handling and Recovery Test

### Files to Create:
- `tests/integration/resilience/error_test.go`
- `tests/integration/resilience/recovery_test.go`

### Implementation:
```go
func TestErrorHandlingAndRecovery(t *testing.T) {
    client := setupStudioClient()
    
    // Test invalid schema validation
    t.Run("InvalidSchemaData", func(t *testing.T) {
        _, err := client.CreateBlob(CreateBlobRequest{
            UserID:   "test-user",
            SchemaID: "strict-schema",
            Data: map[string]interface{}{
                "invalid": "data",
            },
        })
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "validation failed")
    })
    
    // Test database connection recovery
    t.Run("DatabaseRecovery", func(t *testing.T) {
        // Simulate database outage
        stopDatabase()
        
        // Attempt operation (should fail)
        _, err := client.CreateBook(CreateBookRequest{
            UserID: "test-user",
            Title:  "Test Book",
        })
        assert.Error(t, err)
        
        // Restart database
        startDatabase()
        time.Sleep(5 * time.Second) // Wait for reconnection
        
        // Operation should succeed now
        book, err := client.CreateBook(CreateBookRequest{
            UserID: "test-user",
            Title:  "Test Book After Recovery",
        })
        assert.NoError(t, err)
        assert.NotEmpty(t, book.ID)
    })
    
    // Test event bus recovery
    t.Run("EventBusRecovery", func(t *testing.T) {
        // Setup event listener
        received := make(chan bool, 1)
        eventBus := setupEventBus()
        eventBus.Subscribe("test.event", func(event interface{}) {
            received <- true
        })
        
        // Disconnect NATS
        stopNATS()
        
        // Publish event (should be queued)
        eventBus.Publish("test.event", map[string]interface{}{
            "test": "data",
        })
        
        // Restart NATS
        startNATS()
        time.Sleep(5 * time.Second)
        
        // Event should be delivered
        select {
        case <-received:
            // Success
        case <-time.After(10 * time.Second):
            t.Fatal("Event not received after recovery")
        }
    })
}
```

### Acceptance Criteria:
- Invalid data rejected with clear errors
- System recovers from database outages
- Event delivery resumes after NATS recovery
- No data loss during temporary outages

---

## Task 10: Security and Authentication Test

### Files to Create:
- `tests/integration/security/auth_test.go`
- `tests/integration/security/permissions_test.go`

### Implementation:
```go
func TestSecurityAndPermissions(t *testing.T) {
    client := setupStudioClient()
    
    // Test unauthorized access
    t.Run("UnauthorizedAccess", func(t *testing.T) {
        unauthClient := setupUnauthenticatedClient()
        
        _, err := unauthClient.CreateBook(CreateBookRequest{
            UserID: "test-user",
            Title:  "Unauthorized Book",
        })
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "unauthorized")
    })
    
    // Test cross-user access prevention
    t.Run("CrossUserAccess", func(t *testing.T) {
        // Create book as user1
        user1Client := setupClientForUser("user1")
        book, err := user1Client.CreateBook(CreateBookRequest{
            UserID: "user1",
            Title:  "User1's Book",
        })
        assert.NoError(t, err)
        
        // Try to access as user2
        user2Client := setupClientForUser("user2")
        _, err = user2Client.GetBook(book.ID)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "forbidden")
        
        // Try to modify as user2
        err = user2Client.UpdateBook(book.ID, UpdateBookRequest{
            Title: "Hacked Title",
        })
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "forbidden")
    })
    
    // Test token expiration
    t.Run("TokenExpiration", func(t *testing.T) {
        expiredToken := generateExpiredToken()
        clientWithExpired := setupClientWithToken(expiredToken)
        
        _, err := clientWithExpired.CreateBook(CreateBookRequest{
            UserID: "test-user",
            Title:  "Test Book",
        })
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "token expired")
    })
    
    // Test rate limiting
    t.Run("RateLimiting", func(t *testing.T) {
        client := setupClientForUser("rate-test-user")
        
        // Make many rapid requests
        var lastErr error
        for i := 0; i < 100; i++ {
            _, lastErr = client.CreateBook(CreateBookRequest{
                UserID: "rate-test-user",
                Title:  fmt.Sprintf("Book %d", i),
            })
            if lastErr != nil && strings.Contains(lastErr.Error(), "rate limit") {
                break
            }
        }
        
        assert.Error(t, lastErr)
        assert.Contains(t, lastErr.Error(), "rate limit exceeded")
    })
}
```

### Acceptance Criteria:
- Unauthorized requests rejected
- Users cannot access other users' data
- Expired tokens rejected
- Rate limiting enforced

---

## Task 11: Data Consistency Test

### Files to Create:
- `tests/integration/consistency/transaction_test.go`
- `tests/integration/consistency/sync_test.go`

### Implementation:
```go
func TestDataConsistency(t *testing.T) {
    // Test transactional consistency
    t.Run("TransactionalOperations", func(t *testing.T) {
        client := setupStudioClient()
        userID := "test-user"
        
        // Create book with chapters atomically
        bookReq := CreateBookWithChaptersRequest{
            UserID: userID,
            Title:  "Atomic Book",
            Chapters: []Chapter{
                {Title: "Chapter 1", Content: "Content 1"},
                {Title: "Chapter 2", Content: "Content 2"},
            },
        }
        
        // Simulate partial failure
        simulateChapterCreationFailure(2) // Fail on second chapter
        
        book, err := client.CreateBookWithChapters(bookReq)
        assert.Error(t, err)
        
        // Verify rollback - no book should exist
        books, err := client.ListBooks(userID)
        assert.NoError(t, err)
        assert.Empty(t, books)
    })
    
    // Test eventual consistency
    t.Run("EventualConsistency", func(t *testing.T) {
        stateClient := setupStateClient()
        processorClient := setupProcessorClient()
        
        userID := "test-user"
        
        // Create blob
        blob, err := stateClient.CreateBlob(CreateBlobRequest{
            UserID: userID,
            Data: map[string]interface{}{
                "content": "Original content",
            },
        })
        assert.NoError(t, err)
        
        // Process blob (async)
        job, err := processorClient.Execute(ExecuteRequest{
            ProcessorID: "async-processor",
            InputBlobID: blob.ID,
            UserID:      userID,
        })
        assert.NoError(t, err)
        
        // Poll for consistency
        var outputBlob *Blob
        for i := 0; i < 10; i++ {
            time.Sleep(1 * time.Second)
            
            job, _ = processorClient.GetJob(job.ID)
            if job.Status == "completed" {
                outputBlob, _ = stateClient.GetBlob(job.OutputBlobID)
                break
            }
        }
        
        assert.NotNil(t, outputBlob)
        assert.Contains(t, outputBlob.Data, "processed")
    })
}
```

### Acceptance Criteria:
- Transactional operations maintain consistency
- Failed operations properly rollback
- Eventual consistency achieved
- No orphaned data after failures

---

## Task 12: Migration and Upgrade Test

### Files to Create:
- `tests/integration/migration/schema_migration_test.go`
- `tests/integration/migration/data_migration_test.go`

### Implementation:
```go
func TestSchemaMigration(t *testing.T) {
    schemaClient := setupSchemaClient()
    stateClient := setupStateClient()
    
    // Register v1 schema
    schemaV1 := SchemaDefinition{
        Name:    "user-profile",
        Version: "1.0.0",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "name": map[string]interface{}{"type": "string"},
                "age":  map[string]interface{}{"type": "number"},
            },
        },
    }
    
    v1, err := schemaClient.RegisterSchema(schemaV1)
    assert.NoError(t, err)
    
    // Create data with v1
    blob, err := stateClient.CreateBlob(CreateBlobRequest{
        UserID:   "test-user",
        SchemaID: v1.ID,
        Data: map[string]interface{}{
            "name": "John Doe",
            "age":  30,
        },
    })
    assert.NoError(t, err)
    
    // Register v2 schema with migration
    schemaV2 := SchemaDefinition{
        Name:    "user-profile",
        Version: "2.0.0",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "name":      map[string]interface{}{"type": "string"},
                "birthYear": map[string]interface{}{"type": "number"},
                "email":     map[string]interface{}{"type": "string"},
            },
        },
        Migration: &MigrationDefinition{
            FromVersion: "1.0.0",
            Transform: `
                function migrate(data) {
                    return {
                        name: data.name,
                        birthYear: new Date().getFullYear() - data.age,
                        email: ""
                    };
                }
            `,
        },
    }
    
    v2, err := schemaClient.RegisterSchema(schemaV2)
    assert.NoError(t, err)
    
    // Migrate blob
    migratedBlob, err := stateClient.MigrateBlob(blob.ID, v2.ID)
    assert.NoError(t, err)
    
    // Verify migration
    data := migratedBlob.Data.(map[string]interface{})
    assert.Equal(t, "John Doe", data["name"])
    assert.Equal(t, float64(1994), data["birthYear"]) // Assuming current year 2024
    assert.Equal(t, "", data["email"])
}
```

### Acceptance Criteria:
- Schema versions tracked correctly
- Migration functions execute properly
- Data transformed accurately
- Backward compatibility maintained

---

## Task 13: Monitoring and Observability Test

### Files to Create:
- `tests/integration/monitoring/metrics_test.go`
- `tests/integration/monitoring/health_test.go`

### Implementation:
```go
func TestMonitoringAndHealth(t *testing.T) {
    // Test health endpoints
    t.Run("HealthChecks", func(t *testing.T) {
        services := []string{
            "http://localhost:8000/health", // Gateway
            "http://localhost:8001/health", // Auth
            "http://localhost:8006/health", // State
            "http://localhost:8010/health", // Studio
            "http://localhost:8011/health", // Schema
            "http://localhost:8012/health", // Processor
        }
        
        for _, url := range services {
            resp, err := http.Get(url)
            assert.NoError(t, err)
            assert.Equal(t, 200, resp.StatusCode)
            
            var health HealthResponse
            json.NewDecoder(resp.Body).Decode(&health)
            assert.Equal(t, "healthy", health.Status)
            assert.NotEmpty(t, health.Version)
        }
    })
    
    // Test metrics collection
    t.Run("MetricsCollection", func(t *testing.T) {
        // Make some requests
        client := setupStudioClient()
        for i := 0; i < 10; i++ {
            client.CreateBook(CreateBookRequest{
                UserID: "metrics-test",
                Title:  fmt.Sprintf("Book %d", i),
            })
        }
        
        // Check metrics endpoint
        resp, err := http.Get("http://localhost:8000/metrics")
        assert.NoError(t, err)
        
        body, _ := ioutil.ReadAll(resp.Body)
        metrics := string(body)
        
        // Verify key metrics present
        assert.Contains(t, metrics, "http_requests_total")
        assert.Contains(t, metrics, "http_request_duration_seconds")
        assert.Contains(t, metrics, "db_queries_total")
        assert.Contains(t, metrics, "websocket_connections_active")
    })
    
    // Test distributed tracing
    t.Run("DistributedTracing", func(t *testing.T) {
        client := setupStudioClientWithTracing()
        
        // Make request with trace ID
        traceID := "test-trace-123"
        ctx := context.WithValue(context.Background(), "trace-id", traceID)
        
        book, err := client.CreateBookWithContext(ctx, CreateBookRequest{
            UserID: "trace-test",
            Title:  "Traced Book",
        })
        assert.NoError(t, err)
        
        // Verify trace propagated through services
        traces := getTracesForID(traceID)
        assert.Greater(t, len(traces), 3) // Should have spans from multiple services
        
        // Verify span hierarchy
        assert.Contains(t, traces[0].Service, "gateway")
        assert.Contains(t, traces[1].Service, "studio")
        assert.Contains(t, traces[2].Service, "state")
    })
}
```

### Acceptance Criteria:
- All health endpoints responsive
- Metrics collected accurately
- Distributed tracing works
- Performance metrics available

---

## Running the Tests

### Setup Script:
```bash
#!/bin/bash
# tests/run-integration-tests.sh

# Start test infrastructure
docker-compose -f tests/integration/setup/docker-compose.test.yml up -d

# Wait for services
./tests/integration/setup/wait-for-services.sh

# Load test data
psql -h localhost -p 5433 -U test -d test_schemas < tests/integration/setup/test-data.sql

# Run tests
go test -v ./tests/integration/... -tags=integration

# Cleanup
docker-compose -f tests/integration/setup/docker-compose.test.yml down
```

### CI/CD Integration:
```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - name: Run Integration Tests
        run: ./tests/run-integration-tests.sh
```