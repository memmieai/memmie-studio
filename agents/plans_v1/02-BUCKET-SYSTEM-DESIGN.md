# Bucket System Design - Dynamic Blob Organization

## Overview

Buckets are flexible, hierarchical containers for organizing blobs. They replace hard-coded concepts like `book_id` or `conversation_id` with a universal organization system that can represent ANY grouping: books, conversations, projects, research topics, music albums, recipe collections, workout plans, or any user-defined structure.

## Core Concepts

### What is a Bucket?

A bucket is a named container that:
- Groups related blobs together
- Can contain other buckets (hierarchical)
- Has its own metadata and schema
- Can be processed as a unit
- Supports any organizational pattern

### Examples of Buckets

```yaml
# Book Writing
buckets:
  - type: book
    name: "My Science Fiction Novel"
    children:
      - type: chapter
        name: "Chapter 1: The Beginning"
        blobs: [draft_1, expanded_1, edited_1]
      - type: chapter
        name: "Chapter 2: The Journey"
        blobs: [draft_2, expanded_2]
      - type: research
        name: "World Building"
        blobs: [planet_notes, character_sheets]

# Music Production
buckets:
  - type: album
    name: "Summer Vibes"
    children:
      - type: track
        name: "Opening Theme"
        blobs: [melody_1, drums_1, mix_1]
      - type: track
        name: "Sunset Dreams"
        blobs: [melody_2, bass_2, master_2]

# Research Project
buckets:
  - type: research
    name: "Climate Change Study"
    children:
      - type: papers
        name: "Literature Review"
        blobs: [paper_1, paper_2, notes_1]
      - type: data
        name: "Temperature Data"
        blobs: [dataset_1, analysis_1]
      - type: drafts
        name: "Thesis Drafts"
        blobs: [intro_draft, conclusion_draft]

# Recipe Collection
buckets:
  - type: cookbook
    name: "Family Recipes"
    children:
      - type: category
        name: "Desserts"
        children:
          - type: recipe
            name: "Chocolate Cake"
            blobs: [ingredients, instructions, photo]

# Conversation Thread
buckets:
  - type: conversation
    name: "Project Discussion"
    metadata:
      participants: ["user_1", "user_2"]
    blobs: [message_1, message_2, message_3]
```

## Data Models

### Bucket Model (MongoDB - State Service)

```go
type Bucket struct {
    ID              string                 `bson:"_id"`
    UserID          string                 `bson:"user_id"`
    
    // Identification
    Name            string                 `bson:"name"`
    Type            string                 `bson:"type"`          // user-defined type
    Description     string                 `bson:"description"`
    
    // Hierarchy
    ParentBucketID  *string                `bson:"parent_bucket_id,omitempty"`
    ChildBucketIDs  []string               `bson:"child_bucket_ids"`
    Depth           int                    `bson:"depth"`         // nesting level
    Path            string                 `bson:"path"`          // /root/book/chapter1
    
    // Content
    BlobIDs         []string               `bson:"blob_ids"`
    BlobCount       int                    `bson:"blob_count"`
    TotalSize       int64                  `bson:"total_size_bytes"`
    
    // Schema and Processing
    SchemaID        *string                `bson:"schema_id,omitempty"`     // Optional bucket schema
    ProcessorID     *string                `bson:"processor_id,omitempty"`  // Default processor
    
    // Dynamic Metadata
    Metadata        map[string]interface{} `bson:"metadata"`
    Tags            []string               `bson:"tags"`
    
    // Permissions (for future sharing)
    Visibility      string                 `bson:"visibility"`    // private, shared, public
    Collaborators   []string               `bson:"collaborators"`
    
    // Timestamps
    CreatedAt       time.Time              `bson:"created_at"`
    UpdatedAt       time.Time              `bson:"updated_at"`
    AccessedAt      time.Time              `bson:"accessed_at"`
}

// BucketTemplate for common bucket types
type BucketTemplate struct {
    ID              string                 `bson:"_id"`
    Type            string                 `bson:"type"`          // book, album, research, etc.
    Name            string                 `bson:"name"`
    Description     string                 `bson:"description"`
    
    // Default structure
    DefaultChildren []BucketTemplate       `bson:"default_children"`
    
    // Suggested processors
    SuggestedProcessors []string           `bson:"suggested_processors"`
    
    // UI hints
    Icon            string                 `bson:"icon"`
    Color           string                 `bson:"color"`
    UILayout        map[string]interface{} `bson:"ui_layout"`
}
```

### Updated Blob Model

```go
type Blob struct {
    ID              string                 `bson:"_id"`
    UserID          string                 `bson:"user_id"`
    ProcessorID     string                 `bson:"processor_id"`
    SchemaID        string                 `bson:"schema_id"`
    SchemaVersion   string                 `bson:"schema_version"`
    
    // Dynamic content
    Data            interface{}            `bson:"data"`
    
    // Relationships
    ParentID        *string                `bson:"parent_id,omitempty"`
    DerivedIDs      []string               `bson:"derived_ids"`
    
    // Bucket associations (multiple allowed)
    BucketIDs       []string               `bson:"bucket_ids"`      // Can belong to multiple buckets
    PrimaryBucketID *string                `bson:"primary_bucket_id,omitempty"`
    
    // Metadata
    Title           string                 `bson:"title"`
    Preview         string                 `bson:"preview"`
    Tags            []string               `bson:"tags"`
    ContentSize     int64                  `bson:"content_size_bytes"`
    
    // Processing
    ProcessingState string                 `bson:"processing_state"`
    ProcessingMeta  map[string]interface{} `bson:"processing_meta"`
    
    // Timestamps
    CreatedAt       time.Time              `bson:"created_at"`
    UpdatedAt       time.Time              `bson:"updated_at"`
    AccessedAt      time.Time              `bson:"accessed_at"`
}
```

## Bucket Operations

### Creating Buckets

```go
// Create a book bucket
POST /api/v1/buckets
{
  "name": "My Science Fiction Novel",
  "type": "book",
  "metadata": {
    "genre": "sci-fi",
    "target_words": 80000,
    "author": "Jane Doe"
  },
  "processor_id": "book-compiler"
}

// Create a chapter sub-bucket
POST /api/v1/buckets
{
  "name": "Chapter 1: The Beginning",
  "type": "chapter",
  "parent_bucket_id": "bucket_book_123",
  "metadata": {
    "chapter_number": 1,
    "target_words": 5000
  }
}

// Create a conversation bucket
POST /api/v1/buckets
{
  "name": "Team Discussion",
  "type": "conversation",
  "metadata": {
    "participants": ["alice", "bob", "charlie"],
    "topic": "Q4 Planning"
  }
}
```

### Adding Blobs to Buckets

```go
// Create blob and add to bucket
POST /api/v1/blobs
{
  "bucket_ids": ["bucket_chapter1_456"],
  "schema_id": "text-input-v1",
  "data": {
    "content": "The storm approached...",
    "metadata": {
      "draft_version": 1
    }
  }
}

// Add existing blob to bucket
PUT /api/v1/buckets/{bucket_id}/blobs
{
  "blob_ids": ["blob_789", "blob_012"]
}

// Move blob between buckets
PUT /api/v1/blobs/{blob_id}/buckets
{
  "add_bucket_ids": ["bucket_final_drafts"],
  "remove_bucket_ids": ["bucket_rough_drafts"]
}
```

### Querying Buckets

```go
// Get user's buckets with filters
GET /api/v1/users/{user_id}/buckets?type=book&depth=0

// Get bucket with contents
GET /api/v1/buckets/{bucket_id}?include=blobs,children

// Get bucket hierarchy
GET /api/v1/buckets/{bucket_id}/tree

// Search across buckets
GET /api/v1/buckets/search?q=science+fiction&types=book,research

// Get bucket statistics
GET /api/v1/buckets/{bucket_id}/stats
Response:
{
  "total_blobs": 42,
  "total_size_bytes": 1048576,
  "child_buckets": 5,
  "last_modified": "2024-01-01T00:00:00Z",
  "processors_used": ["text-expansion", "grammar-check"],
  "activity_heatmap": {...}
}
```

## Bucket Processing

### Processing Entire Buckets

```go
// Process all blobs in a bucket
POST /api/v1/buckets/{bucket_id}/process
{
  "processor_id": "grammar-check",
  "recursive": true,  // Include child buckets
  "filters": {
    "schema_id": "text-input-v1",
    "tags": ["draft"]
  }
}

// Compile bucket into single output
POST /api/v1/buckets/{bucket_id}/compile
{
  "processor_id": "book-compiler",
  "output_format": "pdf",
  "options": {
    "include_toc": true,
    "include_index": true
  }
}
```

## Bucket Templates

### Predefined Templates

```yaml
# Book Template
book_template:
  type: book
  name: "New Book"
  default_children:
    - type: metadata
      name: "Book Info"
    - type: chapters
      name: "Chapters"
    - type: research
      name: "Research & Notes"
    - type: characters
      name: "Character Development"
  suggested_processors:
    - text-expansion
    - grammar-check
    - book-compiler

# Research Project Template
research_template:
  type: research
  name: "Research Project"
  default_children:
    - type: literature
      name: "Literature Review"
    - type: data
      name: "Data & Analysis"
    - type: drafts
      name: "Paper Drafts"
    - type: presentations
      name: "Presentations"
  suggested_processors:
    - citation-formatter
    - data-analyzer
    - paper-compiler

# Music Album Template
album_template:
  type: album
  name: "New Album"
  default_children:
    - type: tracks
      name: "Tracks"
    - type: demos
      name: "Demos & Ideas"
    - type: samples
      name: "Samples & Loops"
    - type: masters
      name: "Final Masters"
  suggested_processors:
    - audio-mixer
    - music-generator
    - master-processor
```

## Bucket Schemas

### Dynamic Bucket Metadata Schemas

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "bucket-book-v1",
  "type": "object",
  "properties": {
    "title": {"type": "string"},
    "author": {"type": "string"},
    "genre": {"type": "string"},
    "isbn": {"type": "string"},
    "publisher": {"type": "string"},
    "publication_date": {"type": "string", "format": "date"},
    "target_words": {"type": "integer"},
    "current_words": {"type": "integer"},
    "chapters": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "number": {"type": "integer"},
          "title": {"type": "string"},
          "bucket_id": {"type": "string"}
        }
      }
    }
  }
}
```

## Bucket Events

### Event Types

```yaml
# Bucket lifecycle events
bucket.created:
  user_id: "user_123"
  bucket_id: "bucket_456"
  type: "book"
  parent_id: null

bucket.updated:
  bucket_id: "bucket_456"
  changes: ["name", "metadata.genre"]

bucket.deleted:
  bucket_id: "bucket_456"
  blob_count: 42

# Blob-bucket relationship events
bucket.blob.added:
  bucket_id: "bucket_456"
  blob_id: "blob_789"
  blob_count: 15

bucket.blob.removed:
  bucket_id: "bucket_456"
  blob_id: "blob_789"
  blob_count: 14

# Processing events
bucket.processing.started:
  bucket_id: "bucket_456"
  processor_id: "book-compiler"
  blob_count: 20

bucket.processing.completed:
  bucket_id: "bucket_456"
  processor_id: "book-compiler"
  output_blob_id: "blob_compiled_123"
```

## Use Case Examples

### 1. Book Writing with Chapters

```go
// Create book bucket
book := CreateBucket("My Novel", "book")

// Create chapter buckets
ch1 := CreateBucket("Chapter 1", "chapter", book.ID)
ch2 := CreateBucket("Chapter 2", "chapter", book.ID)

// Add drafts to chapters
draft1 := CreateBlob("Chapter 1 text...", ch1.ID)
expanded1 := ProcessBlob(draft1, "text-expansion")
AddBlobToBucket(expanded1, ch1.ID)

// Compile book
compiled := ProcessBucket(book.ID, "book-compiler")
```

### 2. Research Project

```go
// Create research bucket
research := CreateBucket("Climate Study", "research")

// Create sub-buckets
papers := CreateBucket("Papers", "papers", research.ID)
data := CreateBucket("Data", "datasets", research.ID)
analysis := CreateBucket("Analysis", "analysis", research.ID)

// Organize research materials
AddBlobsToBucket(papers.ID, pdfBlobs)
AddBlobsToBucket(data.ID, csvBlobs)

// Process entire research project
ProcessBucket(research.ID, "research-synthesizer")
```

### 3. Conversation Thread

```go
// Create conversation bucket
conv := CreateBucket("Team Chat", "conversation", nil, {
  participants: ["alice", "bob"],
  topic: "Project Planning"
})

// Add messages as blobs
msg1 := CreateBlob("Hey team, let's discuss...", conv.ID)
msg2 := CreateBlob("I think we should...", conv.ID)

// Process conversation
summary := ProcessBucket(conv.ID, "conversation-summarizer")
```

### 4. Music Album

```go
// Create album bucket
album := CreateBucket("Summer Album", "album")

// Create track buckets
track1 := CreateBucket("Track 1", "track", album.ID)
track2 := CreateBucket("Track 2", "track", album.ID)

// Add audio blobs
AddBlobToBucket(melodyBlob, track1.ID)
AddBlobToBucket(drumsBlob, track1.ID)

// Mix track
mixed := ProcessBucket(track1.ID, "audio-mixer")

// Master album
mastered := ProcessBucket(album.ID, "album-master")
```

## Bucket Permissions and Sharing

### Permission Levels

```go
type BucketPermission struct {
    BucketID        string    `bson:"bucket_id"`
    UserID          string    `bson:"user_id"`
    Permission      string    `bson:"permission"` // read, write, admin
    GrantedBy       string    `bson:"granted_by"`
    GrantedAt       time.Time `bson:"granted_at"`
}

// Share bucket with collaborator
POST /api/v1/buckets/{bucket_id}/share
{
  "user_id": "collaborator_123",
  "permission": "write"
}

// Make bucket public
PUT /api/v1/buckets/{bucket_id}/visibility
{
  "visibility": "public",
  "allow_forks": true
}
```

## Bucket Search and Discovery

### Advanced Queries

```go
// Find buckets by content
GET /api/v1/buckets/search
{
  "query": "machine learning",
  "types": ["research", "book"],
  "has_blobs_with_schema": "text-input-v1",
  "min_blob_count": 10,
  "date_range": {
    "from": "2024-01-01",
    "to": "2024-12-31"
  }
}

// Get related buckets
GET /api/v1/buckets/{bucket_id}/related
{
  "based_on": ["tags", "content", "collaborators"],
  "limit": 10
}

// Get bucket recommendations
GET /api/v1/buckets/recommendations
{
  "based_on_history": true,
  "include_public": true,
  "types": ["book", "research"]
}
```

## Performance Optimizations

### Indexing Strategy

```javascript
// MongoDB indexes for buckets
db.buckets.createIndex({ "user_id": 1, "type": 1 })
db.buckets.createIndex({ "path": 1 })
db.buckets.createIndex({ "tags": 1 })
db.buckets.createIndex({ "metadata.genre": 1 })
db.buckets.createIndex({ "created_at": -1 })
db.buckets.createIndex({ 
  "name": "text", 
  "description": "text",
  "tags": "text" 
})

// Compound index for hierarchy queries
db.buckets.createIndex({ 
  "parent_bucket_id": 1, 
  "depth": 1,
  "created_at": -1 
})
```

### Caching Strategy

```go
// Cache bucket structures
cache.Set(fmt.Sprintf("bucket:tree:%s", bucketID), treeStructure, 1*time.Hour)

// Cache bucket statistics
cache.Set(fmt.Sprintf("bucket:stats:%s", bucketID), stats, 15*time.Minute)

// Cache frequently accessed buckets
cache.Set(fmt.Sprintf("bucket:%s", bucketID), bucket, 30*time.Minute)
```

## Migration from Fixed Fields

### Before (Fixed Fields)
```go
type OldBlob struct {
    BookID         *string `bson:"book_id"`
    ConversationID *string `bson:"conversation_id"`
    ProjectID      *string `bson:"project_id"`
    // Limited to predefined types
}
```

### After (Dynamic Buckets)
```go
type NewBlob struct {
    BucketIDs []string `bson:"bucket_ids"`
    // Unlimited organization possibilities
}

// Migration script
func MigrateBlobs() {
    // Create buckets for existing organizations
    if oldBlob.BookID != nil {
        bucket := CreateBucket(oldBlob.BookID, "book")
        newBlob.BucketIDs = append(newBlob.BucketIDs, bucket.ID)
    }
    if oldBlob.ConversationID != nil {
        bucket := CreateBucket(oldBlob.ConversationID, "conversation")
        newBlob.BucketIDs = append(newBlob.BucketIDs, bucket.ID)
    }
}
```

## Benefits of Bucket System

1. **Flexibility**: Any organizational structure possible
2. **Hierarchy**: Natural parent-child relationships
3. **Multi-membership**: Blobs can belong to multiple buckets
4. **Extensibility**: New bucket types without schema changes
5. **Processing**: Process entire buckets as units
6. **Discovery**: Rich search and navigation
7. **Sharing**: Granular permissions per bucket
8. **Templates**: Reusable structures for common patterns

## Implementation Priority

### MVP Phase 1
- Basic bucket CRUD operations
- Single-level hierarchy
- Blob-bucket associations
- Book and conversation bucket types

### MVP Phase 2
- Multi-level hierarchy
- Bucket templates
- Bulk processing
- Basic search

### Post-MVP
- Advanced permissions
- Public sharing
- Bucket marketplace
- Cross-user collaboration
- Bucket versioning
- Bucket analytics dashboard