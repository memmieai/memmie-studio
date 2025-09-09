# Gateway WebSocket Implementation Tasks

## Prerequisites
- Existing memmie-gateway service running on port 8000
- Studio API with WebSocket support on port 8010
- Auth Service on port 8001

## Task 1: Add WebSocket Dependencies
**File**: `memmie-gateway/go.mod`
```bash
go get github.com/gorilla/websocket
go get github.com/hashicorp/golang-lru
```

## Task 2: Create WebSocket Proxy Models
**File**: `internal/proxy/models.go`
```go
package proxy

import (
    "sync"
    "time"
    
    "github.com/gorilla/websocket"
)

type WebSocketProxy struct {
    upgrader    websocket.Upgrader
    backends    []string // Studio API WebSocket URLs
    connections map[string]*ProxyConnection
    mu          sync.RWMutex
    logger      logger.Logger
}

type ProxyConnection struct {
    ID           string
    UserID       string
    ClientConn   *websocket.Conn
    BackendConn  *websocket.Conn
    LastActivity time.Time
    mu           sync.Mutex
}

type ConnectionPool struct {
    connections map[string][]*ProxyConnection // userID -> connections
    mu          sync.RWMutex
}

// Message types for routing
type WSMessage struct {
    Type      string          `json:"type"`
    Data      json.RawMessage `json:"data"`
    Timestamp int64           `json:"timestamp"`
}
```

## Task 3: Implement WebSocket Proxy
**File**: `internal/proxy/websocket_proxy.go`
```go
package proxy

import (
    "context"
    "fmt"
    "net/http"
    "net/url"
    "sync"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/memmieai/memmie-common/pkg/logger"
)

type WebSocketProxy struct {
    upgrader websocket.Upgrader
    backend  string // Studio API WebSocket URL
    pool     *ConnectionPool
    auth     AuthValidator
    logger   logger.Logger
}

func NewWebSocketProxy(backendURL string, auth AuthValidator, logger logger.Logger) *WebSocketProxy {
    return &WebSocketProxy{
        upgrader: websocket.Upgrader{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
            CheckOrigin: func(r *http.Request) bool {
                // Configure CORS properly
                origin := r.Header.Get("Origin")
                // Allow specific origins in production
                return true // For development
            },
        },
        backend: backendURL,
        pool: &ConnectionPool{
            connections: make(map[string][]*ProxyConnection),
        },
        auth:   auth,
        logger: logger,
    }
}

func (p *WebSocketProxy) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Extract and validate token
    token := r.URL.Query().Get("token")
    if token == "" {
        // Try to get from Authorization header
        token = r.Header.Get("Sec-WebSocket-Protocol")
    }
    
    if token == "" {
        http.Error(w, "Missing authentication", http.StatusUnauthorized)
        return
    }
    
    // Validate token
    userInfo, err := p.auth.ValidateToken(r.Context(), token)
    if err != nil || !userInfo.Valid {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
    }
    
    // Upgrade client connection
    clientConn, err := p.upgrader.Upgrade(w, r, nil)
    if err != nil {
        p.logger.Error("Failed to upgrade client connection", "error", err)
        return
    }
    
    // Connect to backend Studio API
    backendURL := fmt.Sprintf("%s?user_id=%s", p.backend, userInfo.UserID)
    backendConn, _, err := websocket.DefaultDialer.Dial(backendURL, nil)
    if err != nil {
        p.logger.Error("Failed to connect to backend", "error", err)
        clientConn.Close()
        return
    }
    
    // Create proxy connection
    conn := &ProxyConnection{
        ID:           generateConnectionID(),
        UserID:       userInfo.UserID,
        ClientConn:   clientConn,
        BackendConn:  backendConn,
        LastActivity: time.Now(),
    }
    
    // Add to pool
    p.pool.AddConnection(userInfo.UserID, conn)
    defer p.pool.RemoveConnection(userInfo.UserID, conn.ID)
    
    // Start proxying
    p.startProxy(conn)
}

func (p *WebSocketProxy) startProxy(conn *ProxyConnection) {
    var wg sync.WaitGroup
    wg.Add(2)
    
    // Client to Backend
    go func() {
        defer wg.Done()
        p.proxyMessages(conn.ClientConn, conn.BackendConn, "client->backend", conn)
    }()
    
    // Backend to Client
    go func() {
        defer wg.Done()
        p.proxyMessages(conn.BackendConn, conn.ClientConn, "backend->client", conn)
    }()
    
    wg.Wait()
    
    // Close both connections
    conn.ClientConn.Close()
    conn.BackendConn.Close()
    
    p.logger.Info("WebSocket proxy connection closed", "user_id", conn.UserID)
}

func (p *WebSocketProxy) proxyMessages(from, to *websocket.Conn, direction string, conn *ProxyConnection) {
    for {
        messageType, data, err := from.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                p.logger.Error("WebSocket read error", "direction", direction, "error", err)
            }
            break
        }
        
        // Update activity
        conn.LastActivity = time.Now()
        
        // Optional: Log or modify messages here
        if p.shouldLogMessage(messageType, data) {
            p.logger.Debug("Proxying message", 
                "direction", direction,
                "user_id", conn.UserID,
                "type", messageType,
                "size", len(data),
            )
        }
        
        // Optional: Apply rate limiting
        if err := p.checkRateLimit(conn.UserID); err != nil {
            p.logger.Warn("Rate limit exceeded", "user_id", conn.UserID)
            continue
        }
        
        // Forward message
        if err := to.WriteMessage(messageType, data); err != nil {
            p.logger.Error("WebSocket write error", "direction", direction, "error", err)
            break
        }
    }
}

func (p *WebSocketProxy) checkRateLimit(userID string) error {
    // Implement rate limiting per user
    // Use Redis or in-memory counter
    return nil
}

func (p *WebSocketProxy) shouldLogMessage(messageType int, data []byte) bool {
    // Don't log ping/pong messages
    return messageType == websocket.TextMessage && len(data) > 0
}
```

## Task 4: Implement Connection Pool Management
**File**: `internal/proxy/connection_pool.go`
```go
package proxy

import (
    "sync"
    "time"
    
    "github.com/google/uuid"
)

func (p *ConnectionPool) AddConnection(userID string, conn *ProxyConnection) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.connections[userID] == nil {
        p.connections[userID] = make([]*ProxyConnection, 0)
    }
    
    p.connections[userID] = append(p.connections[userID], conn)
}

func (p *ConnectionPool) RemoveConnection(userID, connID string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if conns, ok := p.connections[userID]; ok {
        for i, c := range conns {
            if c.ID == connID {
                // Remove from slice
                p.connections[userID] = append(conns[:i], conns[i+1:]...)
                break
            }
        }
        
        // Clean up empty entries
        if len(p.connections[userID]) == 0 {
            delete(p.connections, userID)
        }
    }
}

func (p *ConnectionPool) GetUserConnections(userID string) []*ProxyConnection {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    return p.connections[userID]
}

func (p *ConnectionPool) CleanupInactive(timeout time.Duration) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    now := time.Now()
    for userID, conns := range p.connections {
        activeConns := make([]*ProxyConnection, 0)
        for _, conn := range conns {
            if now.Sub(conn.LastActivity) < timeout {
                activeConns = append(activeConns, conn)
            } else {
                // Close inactive connection
                conn.ClientConn.Close()
                conn.BackendConn.Close()
            }
        }
        
        if len(activeConns) > 0 {
            p.connections[userID] = activeConns
        } else {
            delete(p.connections, userID)
        }
    }
}

func generateConnectionID() string {
    return uuid.New().String()
}
```

## Task 5: Create Load Balancer for Multiple Studio Instances
**File**: `internal/proxy/load_balancer.go`
```go
package proxy

import (
    "fmt"
    "sync/atomic"
)

type LoadBalancer struct {
    backends []string
    current  uint64
}

func NewLoadBalancer(backends []string) *LoadBalancer {
    return &LoadBalancer{
        backends: backends,
        current:  0,
    }
}

// RoundRobin selects the next backend
func (lb *LoadBalancer) NextBackend() string {
    if len(lb.backends) == 0 {
        return ""
    }
    
    if len(lb.backends) == 1 {
        return lb.backends[0]
    }
    
    // Atomic increment and modulo
    n := atomic.AddUint64(&lb.current, 1)
    return lb.backends[n%uint64(len(lb.backends))]
}

// StickySession returns the same backend for a user
func (lb *LoadBalancer) BackendForUser(userID string) string {
    if len(lb.backends) == 0 {
        return ""
    }
    
    if len(lb.backends) == 1 {
        return lb.backends[0]
    }
    
    // Simple hash-based sticky session
    hash := 0
    for _, ch := range userID {
        hash = hash*31 + int(ch)
    }
    
    if hash < 0 {
        hash = -hash
    }
    
    return lb.backends[hash%len(lb.backends)]
}
```

## Task 6: Add WebSocket Health Check
**File**: `internal/proxy/health_check.go`
```go
package proxy

import (
    "context"
    "fmt"
    "time"
    
    "github.com/gorilla/websocket"
)

type HealthChecker struct {
    backends []string
    healthy  map[string]bool
    mu       sync.RWMutex
    logger   logger.Logger
}

func NewHealthChecker(backends []string, logger logger.Logger) *HealthChecker {
    hc := &HealthChecker{
        backends: backends,
        healthy:  make(map[string]bool),
        logger:   logger,
    }
    
    // Mark all as healthy initially
    for _, backend := range backends {
        hc.healthy[backend] = true
    }
    
    return hc
}

func (hc *HealthChecker) Start(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            hc.checkAll()
        }
    }
}

func (hc *HealthChecker) checkAll() {
    for _, backend := range hc.backends {
        go hc.checkBackend(backend)
    }
}

func (hc *HealthChecker) checkBackend(backend string) {
    // Try to establish WebSocket connection
    healthURL := fmt.Sprintf("%s/health", backend)
    
    conn, _, err := websocket.DefaultDialer.Dial(healthURL, nil)
    if err != nil {
        hc.setHealth(backend, false)
        hc.logger.Warn("Backend unhealthy", "backend", backend, "error", err)
        return
    }
    defer conn.Close()
    
    // Send ping
    if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
        hc.setHealth(backend, false)
        return
    }
    
    // Wait for pong
    conn.SetReadDeadline(time.Now().Add(5 * time.Second))
    _, _, err = conn.ReadMessage()
    
    hc.setHealth(backend, err == nil)
}

func (hc *HealthChecker) setHealth(backend string, healthy bool) {
    hc.mu.Lock()
    defer hc.mu.Unlock()
    
    hc.healthy[backend] = healthy
}

func (hc *HealthChecker) IsHealthy(backend string) bool {
    hc.mu.RLock()
    defer hc.mu.RUnlock()
    
    return hc.healthy[backend]
}

func (hc *HealthChecker) GetHealthyBackends() []string {
    hc.mu.RLock()
    defer hc.mu.RUnlock()
    
    healthy := []string{}
    for backend, isHealthy := range hc.healthy {
        if isHealthy {
            healthy = append(healthy, backend)
        }
    }
    
    return healthy
}
```

## Task 7: Update Gateway Router
**File**: `internal/router/router.go` (update existing)
```go
package router

import (
    "github.com/gorilla/mux"
    "github.com/memmieai/memmie-gateway/internal/proxy"
    "github.com/memmieai/memmie-gateway/internal/middleware"
)

func SetupRoutes(
    router *mux.Router,
    httpProxy *proxy.HTTPProxy,
    wsProxy *proxy.WebSocketProxy,
    authMiddleware *middleware.AuthMiddleware,
) {
    // Existing HTTP routes...
    
    // WebSocket route - special handling, no HTTP auth middleware
    router.HandleFunc("/ws", wsProxy.HandleWebSocket)
    
    // Alternative WebSocket routes for different clients
    router.HandleFunc("/websocket", wsProxy.HandleWebSocket)
    router.HandleFunc("/socket.io/", wsProxy.HandleWebSocket) // For Socket.IO compatibility
}
```

## Task 8: Create WebSocket Middleware
**File**: `internal/middleware/websocket.go`
```go
package middleware

import (
    "net/http"
    "strings"
    "time"
    
    "github.com/memmieai/memmie-common/pkg/logger"
)

type WebSocketMiddleware struct {
    rateLimiter RateLimiter
    logger      logger.Logger
}

func NewWebSocketMiddleware(rateLimiter RateLimiter, logger logger.Logger) *WebSocketMiddleware {
    return &WebSocketMiddleware{
        rateLimiter: rateLimiter,
        logger:      logger,
    }
}

func (m *WebSocketMiddleware) RateLimit(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get client IP
        ip := m.getClientIP(r)
        
        // Check rate limit
        if !m.rateLimiter.Allow(ip) {
            http.Error(w, "Too many connections", http.StatusTooManyRequests)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func (m *WebSocketMiddleware) getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        parts := strings.Split(xff, ",")
        return strings.TrimSpace(parts[0])
    }
    
    // Check X-Real-IP header
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    
    // Fall back to RemoteAddr
    return strings.Split(r.RemoteAddr, ":")[0]
}

// LogConnections logs WebSocket connections
func (m *WebSocketMiddleware) LogConnections(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Log connection attempt
        m.logger.Info("WebSocket connection attempt",
            "method", r.Method,
            "path", r.URL.Path,
            "remote_addr", r.RemoteAddr,
            "user_agent", r.UserAgent(),
        )
        
        // Wrap response writer to capture status
        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        
        next.ServeHTTP(wrapped, r)
        
        // Log connection result
        m.logger.Info("WebSocket connection result",
            "status", wrapped.statusCode,
            "duration", time.Since(start),
        )
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

## Task 9: Implement Rate Limiter
**File**: `internal/middleware/rate_limiter.go`
```go
package middleware

import (
    "sync"
    "time"
    
    "golang.org/x/time/rate"
)

type RateLimiter interface {
    Allow(key string) bool
}

type IPRateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
    return &IPRateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     r,
        burst:    b,
    }
}

func (rl *IPRateLimiter) Allow(ip string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    limiter, exists := rl.limiters[ip]
    if !exists {
        limiter = rate.NewLimiter(rl.rate, rl.burst)
        rl.limiters[ip] = limiter
    }
    
    return limiter.Allow()
}

// Cleanup removes old entries
func (rl *IPRateLimiter) Cleanup() {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    
    // Simple cleanup - in production, track last used time
    if len(rl.limiters) > 10000 {
        // Reset if too many entries
        rl.limiters = make(map[string]*rate.Limiter)
    }
}
```

## Task 10: Update Gateway Configuration
**File**: `internal/config/config.go` (update)
```go
package config

type Config struct {
    // Existing fields...
    
    // WebSocket configuration
    WebSocketEnabled     bool     `env:"WEBSOCKET_ENABLED" default:"true"`
    StudioWebSocketURLs  []string `env:"STUDIO_WS_URLS" default:"ws://localhost:8010/ws"`
    WebSocketTimeout     int      `env:"WS_TIMEOUT_SECONDS" default:"300"`
    MaxConnectionsPerIP  int      `env:"MAX_CONNECTIONS_PER_IP" default:"10"`
    MaxConnectionsPerUser int     `env:"MAX_CONNECTIONS_PER_USER" default:"5"`
}
```

## Task 11: Update Main Server
**File**: `cmd/server/main.go` (update)
```go
package main

import (
    "context"
    
    "github.com/memmieai/memmie-gateway/internal/proxy"
    "github.com/memmieai/memmie-gateway/internal/middleware"
    "golang.org/x/time/rate"
)

func main() {
    // Existing initialization...
    
    // Initialize WebSocket proxy if enabled
    if cfg.WebSocketEnabled {
        // Create load balancer
        loadBalancer := proxy.NewLoadBalancer(cfg.StudioWebSocketURLs)
        
        // Create health checker
        healthChecker := proxy.NewHealthChecker(cfg.StudioWebSocketURLs, log)
        go healthChecker.Start(context.Background())
        
        // Create WebSocket proxy
        wsProxy := proxy.NewWebSocketProxy(
            loadBalancer,
            authClient,
            healthChecker,
            log,
        )
        
        // Create rate limiter
        rateLimiter := middleware.NewIPRateLimiter(
            rate.Limit(cfg.MaxConnectionsPerIP),
            cfg.MaxConnectionsPerIP,
        )
        
        // Create WebSocket middleware
        wsMiddleware := middleware.NewWebSocketMiddleware(rateLimiter, log)
        
        // Add WebSocket route with middleware
        router.Handle("/ws", 
            wsMiddleware.LogConnections(
                wsMiddleware.RateLimit(
                    http.HandlerFunc(wsProxy.HandleWebSocket),
                ),
            ),
        )
        
        log.Info("WebSocket proxy enabled", "backends", cfg.StudioWebSocketURLs)
    }
    
    // Rest of server setup...
}
```

## Task 12: Create WebSocket Client for Testing
**File**: `test/websocket_client_test.go`
```go
package test

import (
    "testing"
    "time"
    
    "github.com/gorilla/websocket"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWebSocketProxy(t *testing.T) {
    // Get auth token
    token := getTestToken(t)
    
    // Connect through gateway
    url := fmt.Sprintf("ws://localhost:8000/ws?token=%s", token)
    conn, _, err := websocket.DefaultDialer.Dial(url, nil)
    require.NoError(t, err)
    defer conn.Close()
    
    // Send test message
    testMsg := map[string]interface{}{
        "type": "heartbeat",
        "timestamp": time.Now().Unix(),
    }
    
    err = conn.WriteJSON(testMsg)
    require.NoError(t, err)
    
    // Read response
    var response map[string]interface{}
    err = conn.ReadJSON(&response)
    require.NoError(t, err)
    
    assert.Equal(t, "ack", response["type"])
}

func TestWebSocketLoadBalancing(t *testing.T) {
    // Test that multiple connections are distributed
    connections := make([]*websocket.Conn, 10)
    
    for i := 0; i < 10; i++ {
        token := getTestToken(t)
        url := fmt.Sprintf("ws://localhost:8000/ws?token=%s", token)
        conn, _, err := websocket.DefaultDialer.Dial(url, nil)
        require.NoError(t, err)
        connections[i] = conn
    }
    
    // Clean up
    for _, conn := range connections {
        conn.Close()
    }
}

func TestWebSocketRateLimiting(t *testing.T) {
    // Try to create too many connections from same IP
    connections := []*websocket.Conn{}
    maxAllowed := 10
    
    for i := 0; i < maxAllowed+5; i++ {
        token := getTestToken(t)
        url := fmt.Sprintf("ws://localhost:8000/ws?token=%s", token)
        conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
        
        if i < maxAllowed {
            require.NoError(t, err)
            connections = append(connections, conn)
        } else {
            // Should be rate limited
            assert.Error(t, err)
            if resp != nil {
                assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
            }
        }
    }
    
    // Clean up
    for _, conn := range connections {
        conn.Close()
    }
}
```

## Task 13: Add Monitoring and Metrics
**File**: `internal/proxy/metrics.go`
```go
package proxy

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    wsConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "gateway_websocket_connections",
        Help: "Current number of WebSocket connections",
    }, []string{"backend"})
    
    wsMessages = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "gateway_websocket_messages_total",
        Help: "Total number of WebSocket messages proxied",
    }, []string{"direction", "backend"})
    
    wsErrors = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "gateway_websocket_errors_total",
        Help: "Total number of WebSocket errors",
    }, []string{"type", "backend"})
    
    wsLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "gateway_websocket_latency_seconds",
        Help: "WebSocket message latency",
    }, []string{"backend"})
)

func recordConnection(backend string, delta float64) {
    wsConnections.WithLabelValues(backend).Add(delta)
}

func recordMessage(direction, backend string) {
    wsMessages.WithLabelValues(direction, backend).Inc()
}

func recordError(errorType, backend string) {
    wsErrors.WithLabelValues(errorType, backend).Inc()
}
```

## Testing Checklist
- [ ] WebSocket connection through gateway works
- [ ] Authentication via token works
- [ ] Messages proxy correctly both directions
- [ ] Load balancing distributes connections
- [ ] Rate limiting prevents abuse
- [ ] Health checks detect backend failures
- [ ] Graceful handling of backend disconnection
- [ ] Metrics are recorded correctly

## Success Criteria
- [ ] WebSocket connections establish successfully
- [ ] <10ms additional latency from proxying
- [ ] Handles 1000 concurrent connections
- [ ] Automatic failover on backend failure
- [ ] Rate limiting protects against DoS
- [ ] Zero message loss during normal operation