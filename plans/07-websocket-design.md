# WebSocket Real-time Communication Design

## Overview

The WebSocket system provides real-time, bidirectional communication between ReYNa Studio clients and the backend. It delivers instant updates when blobs are created, processed, or modified, enabling a responsive user experience.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                    Clients                           │
│         (Web, Mobile, AR - Multiple Devices)         │
└──────────────────────────────────────────────────────┘
                         │
                    WebSocket
                         │
┌──────────────────────────────────────────────────────┐
│              Studio API WebSocket Server             │
│                    Port 8010/ws                      │
│                                                      │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │Connection  │  │   Event    │  │   Auth     │   │
│  │  Manager   │  │   Router   │  │ Validator  │   │
│  └────────────┘  └────────────┘  └────────────┘   │
└──────────────────────────────────────────────────────┘
                         │
                    NATS Events
                         │
┌──────────────────────────────────────────────────────┐
│              Backend Services Events                 │
│  State Service | Processor Service | Schema Service  │
└──────────────────────────────────────────────────────┘
```

## WebSocket Implementation

### Connection Manager
```go
package websocket

import (
    "sync"
    "github.com/gorilla/websocket"
)

type Hub struct {
    // Registered clients by user ID
    clients    map[string]map[*Client]bool // userID -> set of clients
    
    // Inbound messages from clients
    broadcast  chan Message
    
    // Register requests from clients
    register   chan *Client
    
    // Unregister requests from clients
    unregister chan *Client
    
    // Mutex for concurrent access
    mu         sync.RWMutex
}

type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    
    // Client identification
    userID   string
    deviceID string
    
    // Subscriptions
    subscriptions map[string]bool // event patterns
    
    // Rate limiting
    limiter  *rate.Limiter
}

type Message struct {
    Type      string                 `json:"type"`
    UserID    string                 `json:"user_id,omitempty"`
    EventType string                 `json:"event_type,omitempty"`
    Data      interface{}            `json:"data"`
    Timestamp int64                  `json:"timestamp"`
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[string]map[*Client]bool),
        broadcast:  make(chan Message, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.registerClient(client)
            
        case client := <-h.unregister:
            h.unregisterClient(client)
            
        case message := <-h.broadcast:
            h.broadcastMessage(message)
        }
    }
}

func (h *Hub) registerClient(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    if h.clients[client.userID] == nil {
        h.clients[client.userID] = make(map[*Client]bool)
    }
    h.clients[client.userID][client] = true
    
    // Send connection success
    client.send <- []byte(`{"type":"connected","data":{"status":"ready"}}`)
}

func (h *Hub) broadcastMessage(message Message) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    // Send to all clients of the target user
    if clients, ok := h.clients[message.UserID]; ok {
        data, _ := json.Marshal(message)
        for client := range clients {
            select {
            case client.send <- data:
            default:
                // Client's send channel is full, close it
                h.unregisterClient(client)
            }
        }
    }
}
```

### NATS Event Subscriber
```go
type EventSubscriber struct {
    nc    *nats.Conn
    hub   *Hub
}

func NewEventSubscriber(natsURL string, hub *Hub) (*EventSubscriber, error) {
    nc, err := nats.Connect(natsURL)
    if err != nil {
        return nil, err
    }
    
    subscriber := &EventSubscriber{
        nc:  nc,
        hub: hub,
    }
    
    // Subscribe to all blob events
    subscriber.subscribeToEvents()
    
    return subscriber, nil
}

func (s *EventSubscriber) subscribeToEvents() {
    // Blob events
    s.nc.Subscribe("blob.>", s.handleBlobEvent)
    
    // Processor events
    s.nc.Subscribe("processor.>", s.handleProcessorEvent)
    
    // Book/conversation events
    s.nc.Subscribe("book.>", s.handleBookEvent)
    s.nc.Subscribe("conversation.>", s.handleConversationEvent)
}

func (s *EventSubscriber) handleBlobEvent(msg *nats.Msg) {
    var event BlobEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        return
    }
    
    // Route to appropriate user's WebSocket
    s.hub.broadcast <- Message{
        Type:      "blob_event",
        UserID:    event.UserID,
        EventType: msg.Subject,
        Data:      event,
        Timestamp: time.Now().Unix(),
    }
}
```

## WebSocket Protocol

### Connection Flow
```yaml
1. Client connects:
   WS: /api/v1/ws?token=<jwt_token>

2. Server authenticates:
   - Validate JWT token
   - Extract userID
   - Create client instance

3. Server sends ready:
   {
     "type": "connected",
     "data": {
       "status": "ready",
       "user_id": "user_123",
       "session_id": "session_456"
     }
   }

4. Client subscribes to events:
   {
     "type": "subscribe",
     "events": ["blob.created", "blob.updated", "processor.completed"]
   }

5. Server confirms:
   {
     "type": "subscribed",
     "data": {
       "events": ["blob.created", "blob.updated", "processor.completed"]
     }
   }
```

### Message Types

#### From Server to Client
```typescript
// Blob created
{
  "type": "blob.created",
  "data": {
    "blob_id": "blob_123",
    "user_id": "user_456",
    "processor_id": "user-input",
    "schema_id": "text-input-v1",
    "preview": "First 200 characters...",
    "book_id": "my-novel",
    "created_at": "2024-01-01T00:00:00Z"
  }
}

// Blob derived (processor created new blob)
{
  "type": "blob.derived",
  "data": {
    "parent_id": "blob_123",
    "derived_id": "blob_789",
    "processor_id": "text-expansion",
    "expansion_ratio": 3.5,
    "preview": "Expanded text preview..."
  }
}

// Processing status
{
  "type": "processor.status",
  "data": {
    "blob_id": "blob_123",
    "processor_id": "text-expansion",
    "status": "processing", // processing, completed, failed
    "progress": 0.75,
    "message": "Expanding text..."
  }
}

// Book updated
{
  "type": "book.updated",
  "data": {
    "book_id": "my-novel",
    "chapter_added": {
      "chapter_num": 5,
      "blob_id": "blob_999",
      "title": "Chapter 5: The Revelation"
    }
  }
}

// Error
{
  "type": "error",
  "data": {
    "code": "PROCESSOR_FAILED",
    "message": "Text expansion failed: API quota exceeded",
    "blob_id": "blob_123",
    "retry_after": 3600
  }
}
```

#### From Client to Server
```typescript
// Subscribe to events
{
  "type": "subscribe",
  "events": ["blob.*", "processor.*"]
}

// Unsubscribe
{
  "type": "unsubscribe",
  "events": ["processor.status"]
}

// Create blob via WebSocket
{
  "type": "create_blob",
  "data": {
    "content": "Chapter text...",
    "schema_id": "text-input-v1",
    "metadata": {
      "book_id": "my-novel",
      "chapter": 3
    }
  }
}

// Request processing
{
  "type": "process_blob",
  "data": {
    "blob_id": "blob_123",
    "processor_id": "text-expansion"
  }
}

// Ping/keepalive
{
  "type": "ping"
}
```

## Client Implementation

### JavaScript/TypeScript Client
```typescript
class ReYNaWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private eventHandlers: Map<string, Set<Function>> = new Map();
  private messageQueue: any[] = [];

  constructor(private token: string) {}

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const wsUrl = `${WS_BASE_URL}/api/v1/ws?token=${this.token}`;
      
      this.ws = new WebSocket(wsUrl);
      
      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        this.flushMessageQueue();
        resolve();
      };
      
      this.ws.onmessage = (event) => {
        this.handleMessage(JSON.parse(event.data));
      };
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(error);
      };
      
      this.ws.onclose = () => {
        console.log('WebSocket disconnected');
        this.attemptReconnect();
      };
    });
  }

  private handleMessage(message: any) {
    const { type, data } = message;
    
    // Emit to all handlers for this event type
    const handlers = this.eventHandlers.get(type);
    if (handlers) {
      handlers.forEach(handler => handler(data));
    }
    
    // Also emit to wildcard handlers
    const wildcardHandlers = this.eventHandlers.get('*');
    if (wildcardHandlers) {
      wildcardHandlers.forEach(handler => handler(message));
    }
  }

  on(event: string, handler: Function) {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)!.add(handler);
    
    // Subscribe on server
    this.send({
      type: 'subscribe',
      events: [event]
    });
  }

  off(event: string, handler: Function) {
    const handlers = this.eventHandlers.get(event);
    if (handlers) {
      handlers.delete(handler);
      if (handlers.size === 0) {
        this.eventHandlers.delete(event);
        // Unsubscribe on server
        this.send({
          type: 'unsubscribe',
          events: [event]
        });
      }
    }
  }

  send(message: any) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      // Queue message for later
      this.messageQueue.push(message);
    }
  }

  private flushMessageQueue() {
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      this.send(message);
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }
    
    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
    
    setTimeout(() => {
      this.connect().catch(console.error);
    }, delay);
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// Usage in React
const useReYNaWebSocket = () => {
  const [ws, setWs] = useState<ReYNaWebSocket | null>(null);
  const [connected, setConnected] = useState(false);
  
  useEffect(() => {
    const token = localStorage.getItem('auth_token');
    if (!token) return;
    
    const websocket = new ReYNaWebSocket(token);
    
    websocket.connect()
      .then(() => {
        setConnected(true);
        setWs(websocket);
        
        // Subscribe to events
        websocket.on('blob.created', (data) => {
          console.log('New blob created:', data);
          // Update UI
        });
        
        websocket.on('blob.derived', (data) => {
          console.log('Blob processed:', data);
          // Update UI
        });
      })
      .catch(console.error);
    
    return () => {
      websocket.disconnect();
    };
  }, []);
  
  return { ws, connected };
};
```

## Event Filtering

### User-Specific Routing
```go
func (h *Hub) routeEventToUser(event Event) {
    // Extract user ID from event
    userID := event.GetUserID()
    if userID == "" {
        return // Skip events without user context
    }
    
    // Check if user has active connections
    h.mu.RLock()
    clients, exists := h.clients[userID]
    h.mu.RUnlock()
    
    if !exists || len(clients) == 0 {
        return // No active connections for this user
    }
    
    // Check if event matches user's subscriptions
    message := Message{
        Type:      event.Type,
        UserID:    userID,
        EventType: event.FullType,
        Data:      event.Data,
        Timestamp: time.Now().Unix(),
    }
    
    // Send to user's clients
    h.broadcast <- message
}
```

### Subscription Patterns
```go
type SubscriptionManager struct {
    subscriptions map[string][]string // clientID -> patterns
    mu            sync.RWMutex
}

func (sm *SubscriptionManager) Subscribe(clientID string, patterns []string) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    sm.subscriptions[clientID] = patterns
}

func (sm *SubscriptionManager) Matches(clientID string, eventType string) bool {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    patterns, exists := sm.subscriptions[clientID]
    if !exists {
        return false
    }
    
    for _, pattern := range patterns {
        if matchPattern(pattern, eventType) {
            return true
        }
    }
    
    return false
}

func matchPattern(pattern, eventType string) bool {
    // Support wildcards: blob.* matches blob.created, blob.updated, etc.
    if strings.HasSuffix(pattern, "*") {
        prefix := strings.TrimSuffix(pattern, "*")
        return strings.HasPrefix(eventType, prefix)
    }
    return pattern == eventType
}
```

## Performance Optimizations

### 1. Connection Pooling
- Limit connections per user (max 5 devices)
- Reuse connections across browser tabs
- Implement connection sharing via SharedWorker

### 2. Message Batching
```go
type MessageBatcher struct {
    messages  []Message
    ticker    *time.Ticker
    batchSize int
    send      func([]Message)
}

func (mb *MessageBatcher) Add(msg Message) {
    mb.messages = append(mb.messages, msg)
    
    if len(mb.messages) >= mb.batchSize {
        mb.flush()
    }
}

func (mb *MessageBatcher) flush() {
    if len(mb.messages) > 0 {
        mb.send(mb.messages)
        mb.messages = nil
    }
}
```

### 3. Compression
```go
func (c *Client) writeMessage(messageType int, data []byte) error {
    c.conn.SetWriteDeadline(time.Now().Add(writeWait))
    
    // Enable compression for messages > 1KB
    if len(data) > 1024 {
        c.conn.EnableWriteCompression(true)
    }
    
    return c.conn.WriteMessage(messageType, data)
}
```

### 4. Rate Limiting
```go
func (c *Client) rateLimitedSend(message []byte) error {
    if !c.limiter.Allow() {
        return ErrRateLimitExceeded
    }
    
    select {
    case c.send <- message:
        return nil
    case <-time.After(time.Second):
        return ErrSendTimeout
    }
}
```

## Security Considerations

### 1. Authentication
- Validate JWT token on connection
- Refresh token periodically
- Disconnect on token expiration

### 2. Authorization
- Verify user owns requested resources
- Filter events by user permissions
- Prevent cross-user data leakage

### 3. Input Validation
- Sanitize all client messages
- Limit message size (max 64KB)
- Validate event subscription patterns

### 4. DoS Protection
- Rate limit connections per IP
- Limit subscriptions per client
- Implement backpressure

## Monitoring

### Metrics
- Active connections count
- Messages per second
- Average latency
- Connection duration
- Error rate by type

### Health Checks
```go
func (h *Hub) HealthCheck() HealthStatus {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    totalClients := 0
    for _, clients := range h.clients {
        totalClients += len(clients)
    }
    
    return HealthStatus{
        ActiveConnections: totalClients,
        ActiveUsers:       len(h.clients),
        MessageQueueSize:  len(h.broadcast),
        Status:            "healthy",
    }
}
```

## Testing

### Load Testing
```javascript
// WebSocket load test with k6
import ws from 'k6/ws';
import { check } from 'k6';

export default function() {
  const url = 'ws://localhost:8010/api/v1/ws?token=test_token';
  
  const res = ws.connect(url, {}, function(socket) {
    socket.on('open', () => {
      socket.send(JSON.stringify({
        type: 'subscribe',
        events: ['blob.*']
      }));
    });
    
    socket.on('message', (data) => {
      const message = JSON.parse(data);
      check(message, {
        'message has type': (m) => m.type !== undefined,
        'message has data': (m) => m.data !== undefined,
      });
    });
    
    socket.setTimeout(() => {
      socket.close();
    }, 10000);
  });
  
  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
```

## Implementation Timeline

### Phase 1: Basic WebSocket (MVP)
- Connection management
- Authentication
- Simple event routing
- Text message support

### Phase 2: Advanced Features
- Event filtering
- Message batching
- Compression
- Rate limiting

### Phase 3: Optimization
- Connection pooling
- Horizontal scaling
- Redis pub/sub
- Monitoring dashboard