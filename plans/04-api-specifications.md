# Memmie Studio API Specifications

## API Overview

Base URL: `https://api.memmie.ai/studio/v1`

Authentication: Bearer token (JWT from Auth Service)

Content Types:
- Request: `application/json`
- Response: `application/json`
- Blob uploads: `multipart/form-data`

## Core Endpoints

### Blob Management

#### Create Blob
```http
POST /blobs
Authorization: Bearer {token}
Content-Type: application/json

{
  "content": "base64_encoded_content",
  "content_type": "text/plain",
  "metadata": {
    "title": "My Document",
    "tags": ["draft", "chapter-1"]
  },
  "parent_id": "550e8400-e29b-41d4-a716-446655440000",
  "providers": ["text-expander", "grammar-checker"]
}

Response: 201 Created
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "user_123",
    "content_url": "https://storage.memmie.ai/blobs/550e8400...",
    "content_type": "text/plain",
    "size": 1024,
    "version": 1,
    "parent_id": "550e8400-e29b-41d4-a716-446655440000",
    "metadata": {
      "title": "My Document",
      "tags": ["draft", "chapter-1"]
    },
    "processing_status": {
      "text-expander": "pending",
      "grammar-checker": "pending"
    },
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

#### Get Blob
```http
GET /blobs/{blob_id}
Authorization: Bearer {token}

Query Parameters:
- version (optional): Specific version number
- include_children (optional): Include child blobs
- include_deltas (optional): Include delta history

Response: 200 OK
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "user_123",
    "content": "The actual content...",
    "content_type": "text/plain",
    "size": 1024,
    "version": 3,
    "parent_id": "550e8400-e29b-41d4-a716-446655440000",
    "children": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440002",
        "provider_id": "text-expander",
        "created_at": "2024-01-15T10:31:00Z"
      }
    ],
    "deltas": [
      {
        "id": "delta_1",
        "operation": "create",
        "version": 1,
        "created_at": "2024-01-15T10:30:00Z"
      },
      {
        "id": "delta_2",
        "operation": "update",
        "version": 2,
        "created_at": "2024-01-15T10:35:00Z"
      }
    ],
    "metadata": {
      "title": "My Document",
      "tags": ["draft", "chapter-1"],
      "word_count": 150,
      "reading_time": "1 min"
    },
    "processing_status": {
      "text-expander": "completed",
      "grammar-checker": "completed"
    },
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:40:00Z"
  }
}
```

#### Update Blob
```http
PATCH /blobs/{blob_id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "delta": {
    "operation": "update",
    "patch": {
      "content": "Updated content...",
      "metadata": {
        "tags": ["draft", "chapter-1", "revised"]
      }
    }
  },
  "trigger_providers": true
}

Response: 200 OK
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "version": 4,
    "delta_id": "delta_3",
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

#### Delete Blob
```http
DELETE /blobs/{blob_id}
Authorization: Bearer {token}

Query Parameters:
- cascade (optional): Delete child blobs
- soft (optional): Soft delete (mark as deleted)

Response: 204 No Content
```

#### List User Blobs
```http
GET /blobs
Authorization: Bearer {token}

Query Parameters:
- page (default: 1): Page number
- limit (default: 20): Items per page
- sort (default: -created_at): Sort field and order
- filter[parent_id]: Filter by parent
- filter[content_type]: Filter by content type
- filter[tags]: Filter by tags (comma-separated)
- filter[provider]: Filter by processing provider
- search: Full-text search

Response: 200 OK
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "title": "Chapter 1",
      "content_type": "text/markdown",
      "size": 2048,
      "version": 2,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "pages": 8
  }
}
```

### Delta Operations

#### Get Delta History
```http
GET /blobs/{blob_id}/deltas
Authorization: Bearer {token}

Query Parameters:
- from_version: Start version
- to_version: End version
- provider_id: Filter by provider

Response: 200 OK
{
  "data": [
    {
      "id": "delta_1",
      "blob_id": "550e8400-e29b-41d4-a716-446655440001",
      "provider_id": "user",
      "operation": "create",
      "patch": {
        "content": "Initial content"
      },
      "from_version": 0,
      "to_version": 1,
      "applied_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": "delta_2",
      "blob_id": "550e8400-e29b-41d4-a716-446655440001",
      "provider_id": "grammar-checker",
      "operation": "update",
      "patch": {
        "op": "replace",
        "path": "/content",
        "value": "Corrected content"
      },
      "from_version": 1,
      "to_version": 2,
      "applied_at": "2024-01-15T10:35:00Z"
    }
  ]
}
```

#### Apply Delta
```http
POST /deltas
Authorization: Bearer {token}
Content-Type: application/json

{
  "blob_id": "550e8400-e29b-41d4-a716-446655440001",
  "operation": "update",
  "patch": {
    "op": "add",
    "path": "/metadata/reviewed",
    "value": true
  }
}

Response: 201 Created
{
  "data": {
    "id": "delta_3",
    "blob_id": "550e8400-e29b-41d4-a716-446655440001",
    "status": "applied",
    "new_version": 3,
    "applied_at": "2024-01-15T11:00:00Z"
  }
}
```

#### Revert to Version
```http
POST /blobs/{blob_id}/revert
Authorization: Bearer {token}
Content-Type: application/json

{
  "target_version": 2,
  "create_backup": true
}

Response: 200 OK
{
  "data": {
    "blob_id": "550e8400-e29b-41d4-a716-446655440001",
    "reverted_to": 2,
    "current_version": 5,
    "backup_blob_id": "550e8400-e29b-41d4-a716-446655440003"
  }
}
```

### DAG Operations

#### Get Blob DAG
```http
GET /blobs/{blob_id}/dag
Authorization: Bearer {token}

Query Parameters:
- depth (default: -1): Maximum depth (-1 for unlimited)
- direction (default: both): ancestors|descendants|both

Response: 200 OK
{
  "data": {
    "root_id": "550e8400-e29b-41d4-a716-446655440000",
    "nodes": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "type": "root",
        "level": 0,
        "metadata": {
          "title": "Original Document"
        }
      },
      {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "type": "derived",
        "level": 1,
        "provider_id": "text-expander",
        "metadata": {
          "title": "Expanded Version"
        }
      }
    ],
    "edges": [
      {
        "from": "550e8400-e29b-41d4-a716-446655440000",
        "to": "550e8400-e29b-41d4-a716-446655440001",
        "provider_id": "text-expander",
        "transform": "expansion"
      }
    ],
    "statistics": {
      "total_nodes": 5,
      "max_depth": 3,
      "providers_involved": ["text-expander", "summarizer"]
    }
  }
}
```

#### Get Ancestors
```http
GET /blobs/{blob_id}/ancestors
Authorization: Bearer {token}

Response: 200 OK
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "relation": "parent",
      "distance": 1
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440099",
      "relation": "grandparent",
      "distance": 2
    }
  ]
}
```

#### Get Descendants
```http
GET /blobs/{blob_id}/descendants
Authorization: Bearer {token}

Query Parameters:
- provider_id: Filter by provider
- max_depth: Maximum depth

Response: 200 OK
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "provider_id": "text-expander",
      "depth": 1,
      "created_at": "2024-01-15T10:31:00Z"
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440003",
      "provider_id": "summarizer",
      "depth": 2,
      "created_at": "2024-01-15T10:32:00Z"
    }
  ]
}
```

### Provider Operations

#### List Providers
```http
GET /providers
Authorization: Bearer {token}

Query Parameters:
- category: Filter by category
- status: active|inactive|all
- capabilities: Filter by capabilities

Response: 200 OK
{
  "data": [
    {
      "id": "text-expander",
      "name": "Text Expander",
      "description": "Expands brief text into detailed content",
      "category": "transformation",
      "version": "1.2.0",
      "capabilities": {
        "input_types": ["text/plain", "text/markdown"],
        "output_types": ["text/markdown"],
        "max_input_size": 10485760,
        "supports_streaming": false
      },
      "pricing": {
        "model": "per_use",
        "cost": 0.001
      },
      "status": "active",
      "rating": 4.8,
      "usage_count": 15234
    }
  ]
}
```

#### Get Provider Details
```http
GET /providers/{provider_id}
Authorization: Bearer {token}

Response: 200 OK
{
  "data": {
    "id": "text-expander",
    "name": "Text Expander",
    "description": "Expands brief text into detailed content",
    "long_description": "This provider uses advanced AI...",
    "author": "MemmieAI Labs",
    "version": "1.2.0",
    "schema": {
      "input": {
        "type": "object",
        "properties": {
          "content": {"type": "string"},
          "target_length": {"type": "integer"}
        }
      },
      "output": {
        "type": "object",
        "properties": {
          "expanded_content": {"type": "string"},
          "expansion_ratio": {"type": "number"}
        }
      }
    },
    "configuration": {
      "model": {
        "type": "string",
        "enum": ["gpt-3.5", "gpt-4"],
        "default": "gpt-4"
      },
      "style": {
        "type": "string",
        "enum": ["professional", "casual", "academic"],
        "default": "professional"
      }
    },
    "metrics": {
      "average_latency": 2.5,
      "success_rate": 0.99,
      "total_processed": 150234
    }
  }
}
```

#### Register Provider
```http
POST /providers
Authorization: Bearer {token}
Content-Type: application/json

{
  "id": "custom-analyzer",
  "name": "Custom Analyzer",
  "description": "Analyzes custom data",
  "workflow_id": "custom-analyzer-workflow",
  "trigger_events": ["onCreate", "onEdit"],
  "supported_types": ["application/json"],
  "configuration": {
    "api_key": "xxx",
    "endpoint": "https://api.custom.com"
  }
}

Response: 201 Created
{
  "data": {
    "id": "custom-analyzer",
    "status": "pending_validation",
    "validation_id": "val_123"
  }
}
```

#### Trigger Provider Processing
```http
POST /providers/{provider_id}/process
Authorization: Bearer {token}
Content-Type: application/json

{
  "blob_id": "550e8400-e29b-41d4-a716-446655440001",
  "configuration": {
    "target_length": 500,
    "style": "academic"
  },
  "async": true
}

Response: 202 Accepted
{
  "data": {
    "job_id": "job_123",
    "status": "queued",
    "estimated_completion": "2024-01-15T10:35:00Z",
    "status_url": "/jobs/job_123"
  }
}
```

#### Get Provider Processing Status
```http
GET /providers/{provider_id}/status/{blob_id}
Authorization: Bearer {token}

Response: 200 OK
{
  "data": {
    "blob_id": "550e8400-e29b-41d4-a716-446655440001",
    "provider_id": "text-expander",
    "status": "completed",
    "last_processed_version": 3,
    "processed_at": "2024-01-15T10:31:00Z",
    "output_blob_id": "550e8400-e29b-41d4-a716-446655440002",
    "metadata": {
      "expansion_ratio": 3.5,
      "processing_time": 1.2
    }
  }
}
```

### Batch Operations

#### Batch Create Blobs
```http
POST /blobs/batch
Authorization: Bearer {token}
Content-Type: application/json

{
  "blobs": [
    {
      "content": "Content 1",
      "content_type": "text/plain",
      "metadata": {"chapter": 1}
    },
    {
      "content": "Content 2",
      "content_type": "text/plain",
      "metadata": {"chapter": 2}
    }
  ],
  "common_metadata": {
    "project": "book",
    "author": "user_123"
  },
  "providers": ["text-expander"]
}

Response: 207 Multi-Status
{
  "data": {
    "created": [
      {
        "temp_id": 0,
        "blob_id": "550e8400-e29b-41d4-a716-446655440001",
        "status": "created"
      },
      {
        "temp_id": 1,
        "blob_id": "550e8400-e29b-41d4-a716-446655440002",
        "status": "created"
      }
    ],
    "failed": [],
    "summary": {
      "total": 2,
      "succeeded": 2,
      "failed": 0
    }
  }
}
```

#### Batch Process with Provider
```http
POST /providers/{provider_id}/batch
Authorization: Bearer {token}
Content-Type: application/json

{
  "blob_ids": [
    "550e8400-e29b-41d4-a716-446655440001",
    "550e8400-e29b-41d4-a716-446655440002"
  ],
  "configuration": {
    "style": "professional"
  }
}

Response: 202 Accepted
{
  "data": {
    "batch_id": "batch_123",
    "total_items": 2,
    "status": "processing",
    "status_url": "/batches/batch_123"
  }
}
```

### WebSocket API

#### Real-time Updates
```javascript
// Connect to WebSocket
const ws = new WebSocket('wss://api.memmie.ai/studio/v1/ws');

// Authenticate
ws.send(JSON.stringify({
  type: 'auth',
  token: 'Bearer {token}'
}));

// Subscribe to blob updates
ws.send(JSON.stringify({
  type: 'subscribe',
  channels: [
    'blob:550e8400-e29b-41d4-a716-446655440001',
    'user:blobs',
    'provider:text-expander'
  ]
}));

// Receive updates
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  switch(message.type) {
    case 'blob.updated':
      console.log('Blob updated:', message.data);
      break;
    case 'provider.completed':
      console.log('Provider completed:', message.data);
      break;
    case 'delta.applied':
      console.log('Delta applied:', message.data);
      break;
  }
};
```

### Search and Query

#### Search Blobs
```http
POST /search
Authorization: Bearer {token}
Content-Type: application/json

{
  "query": "machine learning",
  "filters": {
    "content_type": ["text/plain", "text/markdown"],
    "created_after": "2024-01-01T00:00:00Z",
    "tags": ["ai", "ml"]
  },
  "sort": {
    "field": "relevance",
    "order": "desc"
  },
  "highlight": true,
  "page": 1,
  "limit": 20
}

Response: 200 OK
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "title": "Introduction to Machine Learning",
      "excerpt": "...fundamentals of <mark>machine learning</mark>...",
      "relevance_score": 0.95,
      "content_type": "text/markdown",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45
  },
  "facets": {
    "content_types": {
      "text/markdown": 30,
      "text/plain": 15
    },
    "tags": {
      "ai": 40,
      "ml": 35,
      "deep-learning": 20
    }
  }
}
```

#### Advanced Query
```http
POST /query
Authorization: Bearer {token}
Content-Type: application/json

{
  "select": ["id", "title", "content_type", "metadata.word_count"],
  "from": "blobs",
  "where": {
    "and": [
      {"content_type": {"in": ["text/plain", "text/markdown"]}},
      {"metadata.word_count": {"gte": 100}},
      {"or": [
        {"metadata.tags": {"contains": "draft"}},
        {"metadata.status": "review"}
      ]}
    ]
  },
  "order_by": [
    {"field": "metadata.word_count", "direction": "desc"}
  ],
  "limit": 50
}

Response: 200 OK
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "title": "Chapter 1",
      "content_type": "text/markdown",
      "metadata": {
        "word_count": 1500
      }
    }
  ],
  "metadata": {
    "query_time": 0.125,
    "total_results": 23
  }
}
```

### Analytics

#### Get Usage Statistics
```http
GET /analytics/usage
Authorization: Bearer {token}

Query Parameters:
- period: day|week|month|year
- from: Start date
- to: End date

Response: 200 OK
{
  "data": {
    "period": "month",
    "from": "2024-01-01",
    "to": "2024-01-31",
    "statistics": {
      "blobs_created": 1234,
      "blobs_processed": 3456,
      "storage_used": 10485760,
      "providers_used": {
        "text-expander": 234,
        "grammar-checker": 456,
        "summarizer": 123
      },
      "deltas_applied": 5678,
      "api_calls": 12345
    },
    "trends": {
      "blobs_created": "+15%",
      "storage_used": "+8%"
    }
  }
}
```

#### Get Provider Analytics
```http
GET /analytics/providers/{provider_id}
Authorization: Bearer {token}

Response: 200 OK
{
  "data": {
    "provider_id": "text-expander",
    "period": "last_30_days",
    "metrics": {
      "total_invocations": 1234,
      "success_rate": 0.99,
      "average_latency": 2.5,
      "p95_latency": 4.2,
      "p99_latency": 6.8,
      "errors": {
        "timeout": 5,
        "invalid_input": 3,
        "internal_error": 2
      }
    },
    "cost_analysis": {
      "total_cost": 1.234,
      "average_cost_per_invocation": 0.001
    }
  }
}
```

## Error Responses

### Standard Error Format
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input provided",
    "details": {
      "field": "content",
      "reason": "Content exceeds maximum size of 10MB"
    },
    "request_id": "req_123456",
    "documentation": "https://docs.memmie.ai/studio/errors#VALIDATION_ERROR"
  }
}
```

### Error Codes
- `400` - Bad Request
  - `VALIDATION_ERROR`: Input validation failed
  - `INVALID_DELTA`: Delta cannot be applied
  - `VERSION_MISMATCH`: Version conflict

- `401` - Unauthorized
  - `INVALID_TOKEN`: Authentication token invalid
  - `TOKEN_EXPIRED`: Authentication token expired

- `403` - Forbidden
  - `PERMISSION_DENIED`: No permission for resource
  - `QUOTA_EXCEEDED`: Usage quota exceeded

- `404` - Not Found
  - `BLOB_NOT_FOUND`: Blob does not exist
  - `PROVIDER_NOT_FOUND`: Provider not registered

- `409` - Conflict
  - `CONCURRENT_MODIFICATION`: Resource modified by another request
  - `DUPLICATE_RESOURCE`: Resource already exists

- `429` - Too Many Requests
  - `RATE_LIMITED`: Rate limit exceeded

- `500` - Internal Server Error
  - `INTERNAL_ERROR`: Unexpected server error
  - `PROVIDER_ERROR`: Provider processing failed

- `503` - Service Unavailable
  - `SERVICE_UNAVAILABLE`: Service temporarily unavailable
  - `MAINTENANCE_MODE`: System under maintenance

## Rate Limiting

Rate limits are enforced per user:
- 1000 requests per minute for read operations
- 100 requests per minute for write operations
- 10 requests per minute for batch operations

Headers included in responses:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining
- `X-RateLimit-Reset`: Unix timestamp when limit resets

## Versioning

API version is specified in the URL path: `/studio/v1/`

Breaking changes will result in a new version: `/studio/v2/`

Deprecation notices will be provided 6 months in advance via:
- `X-API-Deprecation-Date` header
- API documentation
- Email notifications

This comprehensive API specification provides a complete interface for interacting with the Memmie Studio system.