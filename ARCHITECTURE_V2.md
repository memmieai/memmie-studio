# Memmie Studio - Complete Architecture V2

## Vision
A universal productivity platform that combines personal knowledge management with AI-powered content transformation, supporting dynamic interfaces across web, mobile, and AR platforms.

## Core Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Client Layer                                │
├──────────────┬──────────────┬──────────────┬───────────────────────┤
│  Web (React) │ Mobile (RN)  │ AR (Vision)  │  CLI/Voice/API       │
└──────────────┴──────────────┴──────────────┴───────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Studio Service (Port 8010)                        │
│                      [API Gateway/Proxy]                             │
│  • WebSocket for real-time updates                                  │
│  • Serves React frontend                                            │
│  • Routes to backend services                                       │
│  • Handles authentication                                           │
└─────────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────┬───────┴────────┬────────────────┐
        ▼               ▼                ▼                ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│State Service │ │Provider Svc  │ │Workflow Svc  │ │Core AI Svc   │
│  Port 8006   │ │  Port 8007   │ │  Port 8005   │ │  Port 8004   │
│              │ │              │ │              │ │              │
│ User Blob    │ │ Provider     │ │ Workflow     │ │ LLM/AI       │
│ Storage      │ │ Registry     │ │ Execution    │ │ Processing   │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
        │               │                │                │
        └───────────────┴────────────────┴────────────────┘
                                │
                        ┌───────▼────────┐
                        │      NATS      │
                        │  Event Bus     │
                        └────────────────┘
```

## Service Responsibilities

### 1. Studio Service (Port 8010) - API Gateway & Frontend Server
**Primary Role**: API facade and React app server

```go
// Serves React frontend
GET  /                           # React app
GET  /static/*                   # Static assets

// API Gateway endpoints
POST /api/v1/blobs               # Create blob → routes to State Service
GET  /api/v1/blobs/dag           # Get user's DAG → State Service
POST /api/v1/blobs/ramble        # Speech to blob → Media + State
POST /api/v1/providers/execute   # Execute provider → Provider Service
GET  /api/v1/ui/layout           # Get dynamic UI layout
WS   /api/v1/ws                  # WebSocket for real-time updates
```

### 2. State Service (Port 8006) - User Blob Storage
**Primary Role**: Manage per-user blob storage and DAGs

```go
// MongoDB Collections
users_blobs: {
  user_id: UUID,
  blobs: [{
    id: UUID,
    content: string,
    metadata: {},
    provider_id: string,  // e.g., "book:my-novel"
    parent_id: UUID,      // For DAG relationships
    children: [UUID],
    deltas: [Delta],
    created_at: timestamp,
    updated_at: timestamp
  }]
}

// API
POST   /api/v1/users/{user_id}/blobs
GET    /api/v1/users/{user_id}/blobs
GET    /api/v1/users/{user_id}/dag
DELETE /api/v1/users/{user_id}/blobs/{blob_id}
POST   /api/v1/users/{user_id}/deltas
```

### 3. Provider Service (Port 8007) - Provider Registry & Execution
**Primary Role**: Manage providers and orchestrate transformations

```go
// Provider Types
type Provider struct {
    ID          string      // e.g., "book:my-novel", "music:symphony-1"
    UserID      string      // Owner of this provider instance
    Type        string      // "text-expander", "music-generator", "research-assistant"
    Template    string      // Base template this was created from
    Config      Config      // User-specific configuration
    UILayout    UILayout    // Dynamic UI configuration
}

// API
POST /api/v1/providers           # Create provider instance
GET  /api/v1/providers/templates # Get available templates
POST /api/v1/providers/{id}/execute
GET  /api/v1/providers/{id}/ui-layout
```

## Dynamic UI System

### UI Layout Definition
```typescript
interface UILayout {
  type: "split" | "tabs" | "stack" | "grid" | "canvas";
  orientation?: "horizontal" | "vertical";
  children: UIComponent[];
  metadata: {
    provider_id: string;
    responsive: ResponsiveConfig;
  };
}

interface UIComponent {
  id: string;
  type: "blob-viewer" | "blob-editor" | "dag-visualizer" | 
        "audio-player" | "code-editor" | "markdown" | "ramble-button";
  dataSource: string;  // Path to data in DAG
  props: Record<string, any>;
  actions: UIAction[];
}

interface UIAction {
  type: "transform" | "create" | "delete" | "ramble";
  label: string;
  provider?: string;
  params?: Record<string, any>;
}
```

### Example: Book Writer UI
```json
{
  "type": "split",
  "orientation": "horizontal",
  "children": [
    {
      "id": "raw-input",
      "type": "blob-editor",
      "dataSource": "$.blobs[?(@.metadata.type=='raw')]",
      "props": {
        "title": "Chapter Draft",
        "editable": true
      },
      "actions": [
        {
          "type": "transform",
          "label": "Expand",
          "provider": "text-expander"
        }
      ]
    },
    {
      "id": "expanded-output",
      "type": "blob-viewer",
      "dataSource": "$.blobs[?(@.metadata.type=='expanded')]",
      "props": {
        "title": "Expanded Version",
        "showWordCount": true
      }
    }
  ]
}
```

### Example: Music Creator UI
```json
{
  "type": "stack",
  "children": [
    {
      "id": "strudel-editor",
      "type": "code-editor",
      "dataSource": "$.blobs[?(@.metadata.type=='strudel-code')]",
      "props": {
        "language": "javascript",
        "theme": "music"
      }
    },
    {
      "id": "audio-preview",
      "type": "audio-player",
      "dataSource": "$.blobs[?(@.metadata.type=='audio-render')]",
      "props": {
        "controls": true,
        "visualization": "waveform"
      }
    }
  ]
}
```

## Speech Input System ("Ramble" Feature)

### Implementation Flow
1. **Client**: Records audio via Web Audio API / Native APIs
2. **Studio Service**: Receives audio stream
3. **Media Service**: Processes audio → text (Whisper API)
4. **State Service**: Creates blob from transcription
5. **Provider Service**: Optionally triggers providers

### API
```typescript
// WebSocket for streaming audio
ws.send({
  type: "ramble_start",
  provider_id: "book:my-novel",  // Optional context
  metadata: {
    chapter: 3,
    type: "notes"
  }
});

ws.send(audioChunk);  // Stream audio chunks

ws.send({
  type: "ramble_end"
});

// Receive transcription and created blob
ws.receive({
  type: "ramble_complete",
  blob_id: "...",
  transcription: "..."
});
```

## Cross-Platform Support

### 1. Web (Primary)
- **Framework**: React 18 with TypeScript
- **State**: Zustand for client state
- **Real-time**: WebSocket for live updates
- **UI**: Tailwind CSS + Radix UI
- **Speech**: Web Audio API + MediaRecorder

### 2. Mobile (React Native)
- **Shared Logic**: 90% code reuse with web
- **Native Features**:
  - Voice recording with native modules
  - Offline blob storage with SQLite
  - Background sync

### 3. AR (Vision Pro)
- **Framework**: SwiftUI with RealityKit
- **Interaction**:
  - Spatial blob arrangement
  - Hand gesture manipulation
  - Voice-first input
  - 3D DAG visualization

### 4. CLI
```bash
memmie blob create --ramble
memmie blob list --provider "book:my-novel"
memmie provider execute text-expander --input "blob-id"
memmie dag visualize
```

## Use Cases

### 1. Book Writing Assistant
```yaml
Provider: book-writer
Features:
  - Chapter expansion
  - Character consistency
  - Plot tracking
  - Style maintenance
UI: Split view with draft/expanded
```

### 2. Music Composition (Strudel)
```yaml
Provider: music-generator
Features:
  - Text to Strudel code
  - Pattern generation
  - Rhythm suggestions
  - Live playback
UI: Code editor + audio player
```

### 3. Research Assistant (Notion/Obsidian-like)
```yaml
Provider: research-assistant
Features:
  - Source extraction
  - Citation management
  - Concept linking
  - Summary generation
UI: Graph view + document editor
```

### 4. Code Documentation
```yaml
Provider: code-documenter
Features:
  - Auto-documentation
  - API spec generation
  - Example creation
  - Test generation
UI: Code editor + markdown preview
```

## Data Flow Example: Book Chapter Expansion

1. **User Input** (via UI or Ramble)
   ```
   Studio → State Service: Create blob
   State → NATS: blob.created event
   ```

2. **Provider Execution**
   ```
   Provider Service → Workflow Service: Execute expansion workflow
   Workflow → Core AI: Generate expanded text
   Core AI → Workflow: Return expansion
   ```

3. **Delta Application**
   ```
   Workflow → State Service: Apply deltas
   State → Studio: Updated DAG
   Studio → Client: WebSocket update
   ```

4. **UI Update**
   ```
   Client: Re-render based on new DAG
   UI: Show expanded text in right pane
   ```

## Frontend Architecture

### React App Structure
```
src/
├── components/
│   ├── DynamicUI/        # Renders UI from layout JSON
│   ├── BlobEditor/       # Text editing component
│   ├── DAGVisualizer/    # Graph visualization
│   ├── RambleButton/     # Speech input
│   └── ProviderPanel/    # Provider controls
├── hooks/
│   ├── useWebSocket.ts   # Real-time updates
│   ├── useBlobs.ts       # Blob management
│   └── useProviders.ts   # Provider interaction
├── stores/
│   ├── blobStore.ts      # Zustand store for blobs
│   └── uiStore.ts        # UI state management
└── api/
    └── client.ts         # API client for Studio Service
```

### Studio Service Serving React
```go
// main.go
func main() {
    // API routes
    api := router.Group("/api/v1")
    setupAPIRoutes(api)
    
    // WebSocket
    router.GET("/ws", handleWebSocket)
    
    // Serve React app
    router.Static("/static", "./web/build/static")
    router.StaticFile("/", "./web/build/index.html")
    
    // Catch-all for React routing
    router.NoRoute(func(c *gin.Context) {
        c.File("./web/build/index.html")
    })
}
```

## Security & Performance

### Authentication Flow
1. Studio Service validates JWT from Auth Service
2. Creates session with user context
3. All backend calls include user context
4. Blob access filtered by user_id

### Caching Strategy
- **Redis**: Cache frequently accessed blobs
- **CDN**: Static assets and React app
- **Local Storage**: Offline blob cache
- **Service Worker**: Background sync

### Scalability
- **Horizontal Scaling**: All services stateless
- **Database Sharding**: By user_id
- **Event Streaming**: NATS for decoupling
- **Rate Limiting**: Per user and provider

## Deployment

### Docker Compose (Development)
```yaml
services:
  studio:
    build: ./memmie-studio
    ports:
      - "8010:8010"
    volumes:
      - ./web/build:/app/web/build
    environment:
      - STATE_SERVICE_URL=http://state:8006
      - PROVIDER_SERVICE_URL=http://provider:8007
```

### Kubernetes (Production)
- Separate deployments per service
- HPA for auto-scaling
- Ingress for routing
- Persistent volumes for blob storage

## Next Steps

1. **Week 1**: Implement State Service blob storage
2. **Week 2**: Build Provider Service with templates
3. **Week 3**: Create Studio Service API gateway
4. **Week 4**: Develop React frontend with dynamic UI
5. **Week 5**: Add speech input and WebSocket
6. **Week 6**: Mobile app with React Native
7. **Week 7**: Testing and optimization
8. **Week 8**: AR prototype for Vision Pro

This architecture provides:
- **Flexibility**: Dynamic UI adapts to any provider type
- **Scalability**: Microservices can scale independently
- **Extensibility**: New providers easily added
- **Cross-platform**: Unified API for all clients
- **Real-time**: WebSocket for instant updates
- **AI-Native**: Every interaction can be enhanced with AI