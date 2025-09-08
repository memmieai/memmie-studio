# Namespace-Based Provider Architecture

## Conceptual Shift: Providers as Namespaces

Instead of providers being purely functional processors, they can also serve as organizational namespaces. This creates a powerful dual-purpose system where:
- **Functional Providers**: Process and transform content (text-expander, summarizer, etc.)
- **Namespace Providers**: Organize and scope content (books, projects, topics, folders)

## Architecture

### Provider Types
```go
type ProviderType string

const (
    ProviderTypeProcessor  ProviderType = "processor"  // Transforms content
    ProviderTypeNamespace  ProviderType = "namespace"  // Organizes content
    ProviderTypeHybrid     ProviderType = "hybrid"     // Both organizes and processes
)

type Provider struct {
    ID          string       // e.g., "book:harry-potter", "project:thesis", "folder:research"
    Type        ProviderType
    ParentID    *string      // For nested namespaces
    
    // Namespace properties
    Namespace   NamespaceConfig
    
    // Processing properties (optional for namespace providers)
    Processor   *ProcessorConfig
}

type NamespaceConfig struct {
    Name        string
    Description string
    Schema      BlobSchema    // Enforced structure for blobs in this namespace
    Permissions Permissions   // Access control
    Metadata    map[string]interface{}
    
    // Hierarchical organization
    AllowSubNamespaces bool
    MaxDepth          int
    
    // Auto-processing rules
    OnCreateProviders []string // Auto-trigger these providers for new blobs
    OnEditProviders   []string // Auto-trigger these providers for edits
}
```

## Namespace Examples

### Book as Provider
```yaml
provider:
  id: "book:the-great-novel"
  type: namespace
  namespace:
    name: "The Great Novel"
    description: "My science fiction masterpiece"
    schema:
      required_fields:
        - chapter_number
        - section_type  # outline, draft, final
        - word_count
    on_create_providers:
      - "grammar-checker"
      - "word-counter"
    on_edit_providers:
      - "consistency-checker"
      - "character-tracker"
    metadata:
      genre: "science fiction"
      target_words: 80000
      author: "user_123"
```

### Project as Provider
```yaml
provider:
  id: "project:memmie-v2"
  type: namespace
  namespace:
    name: "Memmie V2 Development"
    description: "Next version development"
    schema:
      required_fields:
        - document_type  # spec, code, test, doc
        - version
        - status        # draft, review, approved
    allow_sub_namespaces: true
    sub_namespaces:
      - "project:memmie-v2:frontend"
      - "project:memmie-v2:backend"
      - "project:memmie-v2:docs"
```

### Research Topic as Provider
```yaml
provider:
  id: "topic:quantum-computing"
  type: namespace
  namespace:
    name: "Quantum Computing Research"
    description: "Collection of quantum computing resources"
    schema:
      required_fields:
        - resource_type  # paper, note, summary, annotation
        - source_url
        - date_added
        - tags
    on_create_providers:
      - "pdf-extractor"
      - "citation-parser"
      - "summarizer"
```

## Blob Organization with Namespace Providers

### Blob Storage Structure
```go
type Blob struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    
    // Namespace scoping
    ProviderID  string      // The namespace this blob belongs to
    Path        string      // Hierarchical path within namespace
    
    // Example paths:
    // "book:harry-potter/chapter-1/draft"
    // "project:thesis/research/paper-1"
    // "folder:personal/journal/2024-01-15"
    
    Content     []byte
    Metadata    map[string]interface{}
    
    // Relationships within namespace
    ParentID    *uuid.UUID  // Parent blob in same namespace
    Children    []uuid.UUID // Child blobs in same namespace
}
```

### Namespace-Scoped Operations
```go
// Get all blobs in a namespace
func (s *StudioService) GetNamespaceBlobs(
    ctx context.Context,
    userID uuid.UUID,
    providerID string,
    options ListOptions,
) ([]*Blob, error) {
    return s.blobRepo.GetByProvider(ctx, userID, providerID, options)
}

// Create blob in namespace
func (s *StudioService) CreateNamespacedBlob(
    ctx context.Context,
    providerID string,
    input CreateBlobInput,
) (*Blob, error) {
    // Validate against namespace schema
    provider, err := s.providerRepo.Get(ctx, providerID)
    if err != nil {
        return nil, err
    }
    
    if err := s.validateAgainstSchema(input, provider.Namespace.Schema); err != nil {
        return nil, err
    }
    
    // Create blob with namespace
    blob := &Blob{
        ProviderID: providerID,
        Path:       input.Path,
        // ... other fields
    }
    
    // Trigger auto-processing providers
    for _, processorID := range provider.Namespace.OnCreateProviders {
        s.triggerProvider(ctx, processorID, blob.ID)
    }
    
    return blob, nil
}
```

## Use Cases

### 1. Book Writing System
```go
// Create a book namespace
bookProvider := &Provider{
    ID:   "book:my-novel",
    Type: ProviderTypeNamespace,
    Namespace: NamespaceConfig{
        Name: "My Novel",
        Schema: BlobSchema{
            RequiredFields: []string{"chapter", "status"},
            ValidStatuses:  []string{"outline", "draft", "edited", "final"},
        },
        OnCreateProviders: []string{"word-counter", "reading-time"},
        OnEditProviders:   []string{"grammar-checker", "consistency-checker"},
    },
}

// Create chapters within the book
chapter1 := CreateNamespacedBlob(ctx, "book:my-novel", CreateBlobInput{
    Path:    "chapters/01",
    Content: []byte("Chapter 1 content..."),
    Metadata: map[string]interface{}{
        "chapter": 1,
        "status":  "draft",
        "title":   "The Beginning",
    },
})

// Auto-triggers word-counter and reading-time providers
// Results stored as child blobs in the same namespace
```

### 2. Research Organization
```go
// Create research topic namespace
researchProvider := &Provider{
    ID:   "research:ai-ethics",
    Type: ProviderTypeNamespace,
    Namespace: NamespaceConfig{
        Name: "AI Ethics Research",
        AllowSubNamespaces: true,
        OnCreateProviders: []string{
            "citation-extractor",
            "key-points-extractor",
            "related-papers-finder",
        },
    },
}

// Add papers to the research namespace
paper1 := CreateNamespacedBlob(ctx, "research:ai-ethics", CreateBlobInput{
    Path:    "papers/2024/bias-in-llms",
    Content: pdfContent,
    Metadata: map[string]interface{}{
        "type":     "paper",
        "authors":  []string{"Smith", "Jones"},
        "year":     2024,
        "doi":      "10.1234/...",
    },
})

// Auto-extracts citations, key points, and finds related papers
```

### 3. Project Management
```go
// Create project namespace with sub-namespaces
projectProvider := &Provider{
    ID:   "project:new-app",
    Type: ProviderTypeNamespace,
    Namespace: NamespaceConfig{
        Name: "New App Development",
        AllowSubNamespaces: true,
    },
}

// Sub-namespaces for organization
subNamespaces := []string{
    "project:new-app:specs",
    "project:new-app:frontend",
    "project:new-app:backend",
    "project:new-app:testing",
    "project:new-app:docs",
}

// Add spec document
spec := CreateNamespacedBlob(ctx, "project:new-app:specs", CreateBlobInput{
    Path:    "api/v1/user-management",
    Content: []byte("API Specification..."),
    Metadata: map[string]interface{}{
        "type":    "openapi",
        "version": "3.0",
        "status":  "draft",
    },
})
```

## Hybrid Providers

Some providers can be both namespace AND processor:

```go
type BookWriterProvider struct {
    Provider
}

func NewBookWriterProvider(bookID string) *BookWriterProvider {
    return &BookWriterProvider{
        Provider: Provider{
            ID:   fmt.Sprintf("book:%s", bookID),
            Type: ProviderTypeHybrid,
            
            // Namespace configuration
            Namespace: NamespaceConfig{
                Name: bookID,
                Schema: BookSchema,
                OnCreateProviders: []string{"grammar-checker"},
            },
            
            // Processing configuration
            Processor: &ProcessorConfig{
                Capabilities: []string{
                    "generate-outline",
                    "expand-chapter",
                    "check-consistency",
                    "track-characters",
                },
            },
        },
    }
}

// Can both organize content AND process it
func (p *BookWriterProvider) Process(ctx context.Context, blob *Blob) (*Blob, error) {
    switch blob.Metadata["action"] {
    case "generate-outline":
        return p.generateOutline(ctx, blob)
    case "expand-chapter":
        return p.expandChapter(ctx, blob)
    case "check-consistency":
        return p.checkConsistency(ctx, blob)
    }
    return blob, nil
}
```

## Benefits of Namespace Providers

1. **Organization**: Natural hierarchical organization of content
2. **Isolation**: Different projects/books/topics don't interfere
3. **Schema Enforcement**: Each namespace can enforce its own structure
4. **Auto-Processing**: Namespace-specific processing pipelines
5. **Permissions**: Fine-grained access control per namespace
6. **Scalability**: Easy to add new namespaces without affecting others
7. **Discovery**: Browse and search within specific namespaces

## Implementation Considerations

### Database Schema Updates
```sql
-- Add namespace support to blobs table
ALTER TABLE blobs ADD COLUMN provider_namespace VARCHAR(255);
ALTER TABLE blobs ADD COLUMN namespace_path TEXT;

CREATE INDEX idx_blobs_namespace ON blobs(provider_namespace, user_id);
CREATE INDEX idx_blobs_namespace_path ON blobs(provider_namespace, namespace_path);

-- Add namespace configuration to providers
ALTER TABLE providers ADD COLUMN namespace_config JSONB;
ALTER TABLE providers ADD COLUMN parent_provider_id VARCHAR(255);

CREATE INDEX idx_providers_parent ON providers(parent_provider_id);
```

### API Updates
```http
# Create namespaced blob
POST /api/v1/namespaces/{namespace-id}/blobs
{
  "path": "chapters/01/draft",
  "content": "...",
  "metadata": {
    "chapter": 1,
    "status": "draft"
  }
}

# List namespace contents
GET /api/v1/namespaces/{namespace-id}/blobs
?path=/chapters/
&recursive=true

# Get namespace tree
GET /api/v1/namespaces/{namespace-id}/tree

# Create sub-namespace
POST /api/v1/namespaces/{namespace-id}/sub-namespaces
{
  "id": "frontend",
  "name": "Frontend Code",
  "schema": {...}
}
```

### Client SDK Updates
```typescript
// TypeScript SDK
const book = await studio.namespaces.create({
  id: 'book:my-novel',
  type: 'namespace',
  name: 'My Novel',
  schema: {
    requiredFields: ['chapter', 'status']
  }
});

// Add content to namespace
const chapter = await book.blobs.create({
  path: 'chapters/01',
  content: 'Chapter content...',
  metadata: {
    chapter: 1,
    status: 'draft'
  }
});

// Browse namespace
const chapters = await book.blobs.list({
  path: 'chapters/',
  recursive: true
});

// Trigger processing within namespace
await book.process('generate-outline');
```

## Migration Path

1. **Phase 1**: Add namespace support to existing providers
2. **Phase 2**: Create namespace-only providers for organization
3. **Phase 3**: Implement hybrid providers
4. **Phase 4**: Add sub-namespace support
5. **Phase 5**: Implement namespace-specific schemas and validation

This namespace-based provider architecture transforms Memmie Studio into a comprehensive content management and processing system where organization and transformation are seamlessly integrated.