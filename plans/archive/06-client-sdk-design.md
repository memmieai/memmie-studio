# Client SDK Design

## Overview

The Memmie Studio SDK provides client libraries for multiple languages to interact with the Studio API. Each SDK follows language-specific best practices while maintaining consistency across implementations.

## JavaScript/TypeScript SDK

### Installation
```bash
npm install @memmieai/studio-sdk
# or
yarn add @memmieai/studio-sdk
```

### Core Client
```typescript
// src/client.ts
import { EventEmitter } from 'events';

export interface StudioConfig {
  apiKey: string;
  baseUrl?: string;
  timeout?: number;
  retryConfig?: RetryConfig;
  webhookSecret?: string;
}

export class StudioClient extends EventEmitter {
  private config: StudioConfig;
  private http: HttpClient;
  private ws?: WebSocket;
  
  // Sub-clients
  public blobs: BlobClient;
  public providers: ProviderClient;
  public deltas: DeltaClient;
  public dag: DAGClient;
  public analytics: AnalyticsClient;
  
  constructor(config: StudioConfig) {
    super();
    this.config = {
      baseUrl: 'https://api.memmie.ai/studio/v1',
      timeout: 30000,
      ...config
    };
    
    this.http = new HttpClient(this.config);
    
    // Initialize sub-clients
    this.blobs = new BlobClient(this.http);
    this.providers = new ProviderClient(this.http);
    this.deltas = new DeltaClient(this.http);
    this.dag = new DAGClient(this.http);
    this.analytics = new AnalyticsClient(this.http);
  }
  
  // WebSocket connection for real-time updates
  async connect(): Promise<void> {
    const wsUrl = this.config.baseUrl.replace('http', 'ws') + '/ws';
    this.ws = new WebSocket(wsUrl);
    
    this.ws.on('open', () => {
      this.ws?.send(JSON.stringify({
        type: 'auth',
        token: `Bearer ${this.config.apiKey}`
      }));
      this.emit('connected');
    });
    
    this.ws.on('message', (data: string) => {
      const message = JSON.parse(data);
      this.emit(message.type, message.data);
    });
    
    this.ws.on('error', (error) => {
      this.emit('error', error);
    });
    
    this.ws.on('close', () => {
      this.emit('disconnected');
      // Auto-reconnect logic
      setTimeout(() => this.connect(), 5000);
    });
  }
  
  // Subscribe to real-time updates
  subscribe(channels: string[]): void {
    if (!this.ws) {
      throw new Error('WebSocket not connected. Call connect() first.');
    }
    
    this.ws.send(JSON.stringify({
      type: 'subscribe',
      channels
    }));
  }
  
  disconnect(): void {
    this.ws?.close();
  }
}
```

### Blob Operations
```typescript
// src/blobs.ts
export interface Blob {
  id: string;
  userId: string;
  content?: string | Buffer;
  contentUrl?: string;
  contentType: string;
  size: number;
  version: number;
  parentId?: string;
  metadata: Record<string, any>;
  processingStatus: Record<string, ProcessingStatus>;
  createdAt: Date;
  updatedAt: Date;
}

export interface CreateBlobInput {
  content: string | Buffer | File;
  contentType?: string;
  metadata?: Record<string, any>;
  parentId?: string;
  providers?: string[];
}

export class BlobClient {
  constructor(private http: HttpClient) {}
  
  async create(input: CreateBlobInput): Promise<Blob> {
    const formData = new FormData();
    
    if (input.content instanceof File) {
      formData.append('file', input.content);
    } else {
      formData.append('content', 
        typeof input.content === 'string' 
          ? input.content 
          : input.content.toString('base64')
      );
    }
    
    if (input.metadata) {
      formData.append('metadata', JSON.stringify(input.metadata));
    }
    
    if (input.providers) {
      formData.append('providers', input.providers.join(','));
    }
    
    const response = await this.http.post('/blobs', formData);
    return this.parseBlob(response.data);
  }
  
  async get(id: string, options?: GetBlobOptions): Promise<Blob> {
    const response = await this.http.get(`/blobs/${id}`, { params: options });
    return this.parseBlob(response.data);
  }
  
  async update(id: string, delta: Delta): Promise<Blob> {
    const response = await this.http.patch(`/blobs/${id}`, { delta });
    return this.parseBlob(response.data);
  }
  
  async delete(id: string, cascade = false): Promise<void> {
    await this.http.delete(`/blobs/${id}`, { params: { cascade } });
  }
  
  async list(options?: ListOptions): Promise<PaginatedResponse<Blob>> {
    const response = await this.http.get('/blobs', { params: options });
    return {
      data: response.data.map(this.parseBlob),
      pagination: response.pagination
    };
  }
  
  async *iterate(options?: ListOptions): AsyncGenerator<Blob> {
    let page = 1;
    let hasMore = true;
    
    while (hasMore) {
      const response = await this.list({ ...options, page });
      
      for (const blob of response.data) {
        yield blob;
      }
      
      hasMore = page < response.pagination.pages;
      page++;
    }
  }
  
  // Stream blob content
  async stream(id: string): Promise<ReadableStream> {
    const response = await this.http.get(`/blobs/${id}/content`, {
      responseType: 'stream'
    });
    return response.data;
  }
  
  // Watch for changes
  watch(id: string, callback: (blob: Blob) => void): () => void {
    const handler = (data: any) => {
      if (data.blobId === id) {
        callback(this.parseBlob(data.blob));
      }
    };
    
    this.http.client.on('blob.updated', handler);
    
    // Return unsubscribe function
    return () => {
      this.http.client.off('blob.updated', handler);
    };
  }
  
  private parseBlob(data: any): Blob {
    return {
      ...data,
      createdAt: new Date(data.createdAt),
      updatedAt: new Date(data.updatedAt)
    };
  }
}
```

### Provider Operations
```typescript
// src/providers.ts
export interface Provider {
  id: string;
  name: string;
  description: string;
  category: string;
  version: string;
  capabilities: ProviderCapabilities;
  status: ProviderStatus;
  config?: Record<string, any>;
}

export interface ProcessingResult {
  jobId: string;
  status: JobStatus;
  output?: Blob;
  error?: string;
  metadata?: Record<string, any>;
}

export class ProviderClient {
  constructor(private http: HttpClient) {}
  
  async list(filter?: ProviderFilter): Promise<Provider[]> {
    const response = await this.http.get('/providers', { params: filter });
    return response.data;
  }
  
  async get(id: string): Promise<Provider> {
    const response = await this.http.get(`/providers/${id}`);
    return response.data;
  }
  
  async process(
    providerId: string, 
    blobId: string, 
    config?: Record<string, any>
  ): Promise<ProcessingResult> {
    const response = await this.http.post(`/providers/${providerId}/process`, {
      blobId,
      configuration: config,
      async: true
    });
    
    // Poll for result if async
    if (response.data.status === 'queued') {
      return this.pollJob(response.data.jobId);
    }
    
    return response.data;
  }
  
  async batchProcess(
    providerId: string,
    blobIds: string[],
    config?: Record<string, any>
  ): Promise<BatchResult> {
    const response = await this.http.post(`/providers/${providerId}/batch`, {
      blobIds,
      configuration: config
    });
    
    return this.pollBatch(response.data.batchId);
  }
  
  private async pollJob(jobId: string): Promise<ProcessingResult> {
    return new Promise((resolve, reject) => {
      const interval = setInterval(async () => {
        try {
          const response = await this.http.get(`/jobs/${jobId}`);
          const job = response.data;
          
          if (job.status === 'completed') {
            clearInterval(interval);
            resolve(job);
          } else if (job.status === 'failed') {
            clearInterval(interval);
            reject(new Error(job.error));
          }
        } catch (error) {
          clearInterval(interval);
          reject(error);
        }
      }, 1000);
    });
  }
}
```

### DAG Operations
```typescript
// src/dag.ts
export interface DAGNode {
  id: string;
  type: 'root' | 'derived';
  level: number;
  providerId?: string;
  metadata: Record<string, any>;
}

export interface DAGEdge {
  from: string;
  to: string;
  providerId: string;
  transform: string;
}

export interface DAG {
  rootId: string;
  nodes: DAGNode[];
  edges: DAGEdge[];
  statistics: DAGStatistics;
}

export class DAGClient {
  constructor(private http: HttpClient) {}
  
  async getDAG(blobId: string, options?: DAGOptions): Promise<DAG> {
    const response = await this.http.get(`/blobs/${blobId}/dag`, {
      params: options
    });
    return response.data;
  }
  
  async getAncestors(blobId: string): Promise<BlobRelation[]> {
    const response = await this.http.get(`/blobs/${blobId}/ancestors`);
    return response.data;
  }
  
  async getDescendants(
    blobId: string, 
    filter?: DescendantFilter
  ): Promise<BlobRelation[]> {
    const response = await this.http.get(`/blobs/${blobId}/descendants`, {
      params: filter
    });
    return response.data;
  }
  
  // Visualize DAG as DOT graph
  toDot(dag: DAG): string {
    const lines = ['digraph G {'];
    
    // Add nodes
    for (const node of dag.nodes) {
      const label = node.metadata.title || node.id.slice(0, 8);
      const color = node.type === 'root' ? 'green' : 'blue';
      lines.push(`  "${node.id}" [label="${label}", color="${color}"];`);
    }
    
    // Add edges
    for (const edge of dag.edges) {
      lines.push(`  "${edge.from}" -> "${edge.to}" [label="${edge.transform}"];`);
    }
    
    lines.push('}');
    return lines.join('\n');
  }
  
  // Find path between nodes
  findPath(dag: DAG, fromId: string, toId: string): string[] | null {
    const adjacency = new Map<string, string[]>();
    
    for (const edge of dag.edges) {
      if (!adjacency.has(edge.from)) {
        adjacency.set(edge.from, []);
      }
      adjacency.get(edge.from)!.push(edge.to);
    }
    
    // BFS to find path
    const queue: Array<{id: string, path: string[]}> = [{
      id: fromId,
      path: [fromId]
    }];
    const visited = new Set<string>();
    
    while (queue.length > 0) {
      const {id, path} = queue.shift()!;
      
      if (id === toId) {
        return path;
      }
      
      if (visited.has(id)) continue;
      visited.add(id);
      
      const neighbors = adjacency.get(id) || [];
      for (const neighbor of neighbors) {
        queue.push({
          id: neighbor,
          path: [...path, neighbor]
        });
      }
    }
    
    return null;
  }
}
```

### React Hooks
```typescript
// src/react/hooks.ts
import { useState, useEffect, useCallback } from 'react';
import { StudioClient, Blob, Provider } from '../index';

export function useBlob(client: StudioClient, blobId: string) {
  const [blob, setBlob] = useState<Blob | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  
  useEffect(() => {
    let cancelled = false;
    
    const fetchBlob = async () => {
      try {
        setLoading(true);
        const data = await client.blobs.get(blobId);
        if (!cancelled) {
          setBlob(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err as Error);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };
    
    fetchBlob();
    
    // Subscribe to updates
    const unsubscribe = client.blobs.watch(blobId, (updated) => {
      if (!cancelled) {
        setBlob(updated);
      }
    });
    
    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, [client, blobId]);
  
  const update = useCallback(async (delta: any) => {
    setLoading(true);
    try {
      const updated = await client.blobs.update(blobId, delta);
      setBlob(updated);
      return updated;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [client, blobId]);
  
  return { blob, loading, error, update };
}

export function useProvider(
  client: StudioClient, 
  providerId: string, 
  blobId: string
) {
  const [processing, setProcessing] = useState(false);
  const [result, setResult] = useState<ProcessingResult | null>(null);
  const [error, setError] = useState<Error | null>(null);
  
  const process = useCallback(async (config?: any) => {
    setProcessing(true);
    setError(null);
    
    try {
      const res = await client.providers.process(providerId, blobId, config);
      setResult(res);
      return res;
    } catch (err) {
      setError(err as Error);
      throw err;
    } finally {
      setProcessing(false);
    }
  }, [client, providerId, blobId]);
  
  return { process, processing, result, error };
}

export function useDAG(client: StudioClient, blobId: string) {
  const [dag, setDAG] = useState<DAG | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  
  useEffect(() => {
    let cancelled = false;
    
    const fetchDAG = async () => {
      try {
        setLoading(true);
        const data = await client.dag.getDAG(blobId);
        if (!cancelled) {
          setDAG(data);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err as Error);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };
    
    fetchDAG();
    
    return () => {
      cancelled = true;
    };
  }, [client, blobId]);
  
  return { dag, loading, error };
}
```

## Python SDK

### Installation
```bash
pip install memmie-studio
```

### Core Client
```python
# memmie_studio/client.py
from typing import Optional, Dict, Any, List, AsyncGenerator
import asyncio
import aiohttp
import websockets
from dataclasses import dataclass
from datetime import datetime

@dataclass
class StudioConfig:
    api_key: str
    base_url: str = "https://api.memmie.ai/studio/v1"
    timeout: int = 30
    max_retries: int = 3

class StudioClient:
    def __init__(self, config: StudioConfig):
        self.config = config
        self.session = None
        self.ws = None
        
        # Sub-clients
        self.blobs = BlobClient(self)
        self.providers = ProviderClient(self)
        self.deltas = DeltaClient(self)
        self.dag = DAGClient(self)
        self.analytics = AnalyticsClient(self)
    
    async def __aenter__(self):
        self.session = aiohttp.ClientSession(
            headers={
                "Authorization": f"Bearer {self.config.api_key}",
                "Content-Type": "application/json"
            },
            timeout=aiohttp.ClientTimeout(total=self.config.timeout)
        )
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
        if self.ws:
            await self.ws.close()
    
    async def connect_websocket(self):
        """Connect to WebSocket for real-time updates"""
        ws_url = self.config.base_url.replace("http", "ws") + "/ws"
        self.ws = await websockets.connect(ws_url)
        
        # Authenticate
        await self.ws.send(json.dumps({
            "type": "auth",
            "token": f"Bearer {self.config.api_key}"
        }))
        
        # Start listening
        asyncio.create_task(self._listen_websocket())
    
    async def _listen_websocket(self):
        """Listen for WebSocket messages"""
        async for message in self.ws:
            data = json.loads(message)
            # Handle different message types
            await self._handle_ws_message(data)
    
    async def subscribe(self, channels: List[str]):
        """Subscribe to real-time channels"""
        if not self.ws:
            await self.connect_websocket()
        
        await self.ws.send(json.dumps({
            "type": "subscribe",
            "channels": channels
        }))
    
    async def request(
        self, 
        method: str, 
        path: str, 
        **kwargs
    ) -> Dict[str, Any]:
        """Make HTTP request with retry logic"""
        url = f"{self.config.base_url}{path}"
        
        for attempt in range(self.config.max_retries):
            try:
                async with self.session.request(method, url, **kwargs) as response:
                    response.raise_for_status()
                    return await response.json()
            except aiohttp.ClientError as e:
                if attempt == self.config.max_retries - 1:
                    raise
                await asyncio.sleep(2 ** attempt)
```

### Blob Operations
```python
# memmie_studio/blobs.py
from typing import Optional, Dict, Any, List, AsyncGenerator
from dataclasses import dataclass
from datetime import datetime

@dataclass
class Blob:
    id: str
    user_id: str
    content: Optional[bytes]
    content_url: Optional[str]
    content_type: str
    size: int
    version: int
    parent_id: Optional[str]
    metadata: Dict[str, Any]
    processing_status: Dict[str, str]
    created_at: datetime
    updated_at: datetime

class BlobClient:
    def __init__(self, client: 'StudioClient'):
        self.client = client
    
    async def create(
        self,
        content: Union[str, bytes, BinaryIO],
        content_type: str = "text/plain",
        metadata: Optional[Dict[str, Any]] = None,
        parent_id: Optional[str] = None,
        providers: Optional[List[str]] = None
    ) -> Blob:
        """Create a new blob"""
        if isinstance(content, str):
            content = content.encode('utf-8')
        elif hasattr(content, 'read'):
            content = content.read()
        
        data = {
            "content": base64.b64encode(content).decode('utf-8'),
            "content_type": content_type,
            "metadata": metadata or {},
            "parent_id": parent_id,
            "providers": providers or []
        }
        
        response = await self.client.request("POST", "/blobs", json=data)
        return self._parse_blob(response["data"])
    
    async def get(
        self, 
        blob_id: str,
        version: Optional[int] = None,
        include_children: bool = False,
        include_deltas: bool = False
    ) -> Blob:
        """Get a blob by ID"""
        params = {}
        if version:
            params["version"] = version
        if include_children:
            params["include_children"] = True
        if include_deltas:
            params["include_deltas"] = True
        
        response = await self.client.request(
            "GET", 
            f"/blobs/{blob_id}", 
            params=params
        )
        return self._parse_blob(response["data"])
    
    async def update(self, blob_id: str, delta: Dict[str, Any]) -> Blob:
        """Update a blob with a delta"""
        response = await self.client.request(
            "PATCH",
            f"/blobs/{blob_id}",
            json={"delta": delta}
        )
        return self._parse_blob(response["data"])
    
    async def delete(self, blob_id: str, cascade: bool = False):
        """Delete a blob"""
        params = {"cascade": cascade} if cascade else {}
        await self.client.request(
            "DELETE",
            f"/blobs/{blob_id}",
            params=params
        )
    
    async def list(
        self,
        page: int = 1,
        limit: int = 20,
        sort: str = "-created_at",
        **filters
    ) -> List[Blob]:
        """List user's blobs with filtering"""
        params = {
            "page": page,
            "limit": limit,
            "sort": sort,
            **{f"filter[{k}]": v for k, v in filters.items()}
        }
        
        response = await self.client.request("GET", "/blobs", params=params)
        return [self._parse_blob(b) for b in response["data"]]
    
    async def iterate(self, **filters) -> AsyncGenerator[Blob, None]:
        """Iterate through all blobs"""
        page = 1
        while True:
            blobs = await self.list(page=page, **filters)
            if not blobs:
                break
            
            for blob in blobs:
                yield blob
            
            page += 1
    
    async def stream(self, blob_id: str) -> AsyncGenerator[bytes, None]:
        """Stream blob content"""
        url = f"{self.client.config.base_url}/blobs/{blob_id}/content"
        
        async with self.client.session.get(url) as response:
            async for chunk in response.content.iter_chunked(1024):
                yield chunk
    
    def _parse_blob(self, data: Dict[str, Any]) -> Blob:
        """Parse blob from API response"""
        return Blob(
            id=data["id"],
            user_id=data["user_id"],
            content=data.get("content"),
            content_url=data.get("content_url"),
            content_type=data["content_type"],
            size=data["size"],
            version=data["version"],
            parent_id=data.get("parent_id"),
            metadata=data.get("metadata", {}),
            processing_status=data.get("processing_status", {}),
            created_at=datetime.fromisoformat(data["created_at"]),
            updated_at=datetime.fromisoformat(data["updated_at"])
        )
```

### Context Manager Pattern
```python
# memmie_studio/context.py
from contextlib import asynccontextmanager

@asynccontextmanager
async def studio_session(api_key: str, **config):
    """Context manager for Studio operations"""
    client = StudioClient(StudioConfig(api_key=api_key, **config))
    
    async with client:
        yield client

# Usage
async def main():
    async with studio_session("your-api-key") as studio:
        # Create blob
        blob = await studio.blobs.create(
            "Hello, World!",
            metadata={"title": "Greeting"}
        )
        
        # Process with provider
        result = await studio.providers.process(
            "text-expander",
            blob.id,
            config={"target_length": 100}
        )
        
        # Get DAG
        dag = await studio.dag.get_dag(blob.id)
```

## Go SDK

### Installation
```bash
go get github.com/memmieai/studio-sdk-go
```

### Core Client
```go
// client.go
package studio

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Config struct {
    APIKey     string
    BaseURL    string
    Timeout    time.Duration
    MaxRetries int
}

type Client struct {
    config     Config
    httpClient *http.Client
    
    // Sub-clients
    Blobs      *BlobClient
    Providers  *ProviderClient
    Deltas     *DeltaClient
    DAG        *DAGClient
    Analytics  *AnalyticsClient
}

func NewClient(config Config) *Client {
    if config.BaseURL == "" {
        config.BaseURL = "https://api.memmie.ai/studio/v1"
    }
    if config.Timeout == 0 {
        config.Timeout = 30 * time.Second
    }
    if config.MaxRetries == 0 {
        config.MaxRetries = 3
    }
    
    httpClient := &http.Client{
        Timeout: config.Timeout,
    }
    
    client := &Client{
        config:     config,
        httpClient: httpClient,
    }
    
    // Initialize sub-clients
    client.Blobs = &BlobClient{client: client}
    client.Providers = &ProviderClient{client: client}
    client.Deltas = &DeltaClient{client: client}
    client.DAG = &DAGClient{client: client}
    client.Analytics = &AnalyticsClient{client: client}
    
    return client
}

func (c *Client) request(
    ctx context.Context,
    method string,
    path string,
    body interface{},
) (*http.Response, error) {
    url := fmt.Sprintf("%s%s", c.config.BaseURL, path)
    
    var bodyReader io.Reader
    if body != nil {
        data, err := json.Marshal(body)
        if err != nil {
            return nil, err
        }
        bodyReader = bytes.NewReader(data)
    }
    
    req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
    req.Header.Set("Content-Type", "application/json")
    
    // Retry logic
    var resp *http.Response
    for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
        resp, err = c.httpClient.Do(req)
        if err == nil && resp.StatusCode < 500 {
            break
        }
        
        if attempt < c.config.MaxRetries-1 {
            time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
        }
    }
    
    return resp, err
}
```

### Blob Operations
```go
// blobs.go
package studio

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "time"
)

type Blob struct {
    ID               string                 `json:"id"`
    UserID           string                 `json:"user_id"`
    Content          []byte                 `json:"content,omitempty"`
    ContentURL       string                 `json:"content_url,omitempty"`
    ContentType      string                 `json:"content_type"`
    Size             int64                  `json:"size"`
    Version          int                    `json:"version"`
    ParentID         *string                `json:"parent_id,omitempty"`
    Metadata         map[string]interface{} `json:"metadata"`
    ProcessingStatus map[string]string      `json:"processing_status"`
    CreatedAt        time.Time              `json:"created_at"`
    UpdatedAt        time.Time              `json:"updated_at"`
}

type CreateBlobInput struct {
    Content     []byte                 `json:"content"`
    ContentType string                 `json:"content_type"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
    ParentID    *string                `json:"parent_id,omitempty"`
    Providers   []string               `json:"providers,omitempty"`
}

type BlobClient struct {
    client *Client
}

func (c *BlobClient) Create(ctx context.Context, input CreateBlobInput) (*Blob, error) {
    resp, err := c.client.request(ctx, "POST", "/blobs", input)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        return nil, parseError(resp)
    }
    
    var result struct {
        Data Blob `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result.Data, nil
}

func (c *BlobClient) Get(ctx context.Context, id string) (*Blob, error) {
    resp, err := c.client.request(ctx, "GET", fmt.Sprintf("/blobs/%s", id), nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }
    
    var result struct {
        Data Blob `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result.Data, nil
}

func (c *BlobClient) Update(ctx context.Context, id string, delta Delta) (*Blob, error) {
    body := map[string]interface{}{
        "delta": delta,
    }
    
    resp, err := c.client.request(ctx, "PATCH", fmt.Sprintf("/blobs/%s", id), body)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, parseError(resp)
    }
    
    var result struct {
        Data Blob `json:"data"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result.Data, nil
}

func (c *BlobClient) Delete(ctx context.Context, id string, cascade bool) error {
    path := fmt.Sprintf("/blobs/%s", id)
    if cascade {
        path += "?cascade=true"
    }
    
    resp, err := c.client.request(ctx, "DELETE", path, nil)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusNoContent {
        return parseError(resp)
    }
    
    return nil
}

// Stream blob content
func (c *BlobClient) Stream(ctx context.Context, id string) (io.ReadCloser, error) {
    resp, err := c.client.request(ctx, "GET", fmt.Sprintf("/blobs/%s/content", id), nil)
    if err != nil {
        return nil, err
    }
    
    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, parseError(resp)
    }
    
    return resp.Body, nil
}

// Iterate through blobs
func (c *BlobClient) Iterate(ctx context.Context, options ListOptions) *BlobIterator {
    return &BlobIterator{
        client:  c,
        ctx:     ctx,
        options: options,
        page:    1,
    }
}

type BlobIterator struct {
    client  *BlobClient
    ctx     context.Context
    options ListOptions
    page    int
    items   []Blob
    index   int
    done    bool
}

func (i *BlobIterator) Next() bool {
    if i.done {
        return false
    }
    
    if i.index < len(i.items) {
        i.index++
        return true
    }
    
    // Fetch next page
    i.options.Page = i.page
    result, err := i.client.List(i.ctx, i.options)
    if err != nil || len(result.Data) == 0 {
        i.done = true
        return false
    }
    
    i.items = result.Data
    i.index = 0
    i.page++
    
    return len(i.items) > 0
}

func (i *BlobIterator) Blob() *Blob {
    if i.index > 0 && i.index <= len(i.items) {
        return &i.items[i.index-1]
    }
    return nil
}
```

## CLI Tool

### Installation
```bash
# Global installation
npm install -g @memmieai/studio-cli

# Or use directly with npx
npx @memmieai/studio-cli
```

### Command Structure
```bash
# Initialize configuration
studio init

# Blob operations
studio blob create <file> [--metadata key=value] [--providers provider1,provider2]
studio blob get <id> [--version 2]
studio blob update <id> --delta delta.json
studio blob delete <id> [--cascade]
studio blob list [--filter key=value]

# Provider operations
studio provider list [--category transformation]
studio provider get <provider-id>
studio provider process <provider-id> <blob-id> [--config key=value]

# DAG operations
studio dag show <blob-id> [--depth 3]
studio dag ancestors <blob-id>
studio dag descendants <blob-id>

# Watch for changes
studio watch <blob-id>

# Interactive mode
studio repl
```

### CLI Implementation
```typescript
#!/usr/bin/env node
// src/cli.ts

import { Command } from 'commander';
import { StudioClient } from '@memmieai/studio-sdk';
import * as fs from 'fs/promises';
import * as path from 'path';

const program = new Command();

program
  .name('studio')
  .description('Memmie Studio CLI')
  .version('1.0.0');

// Initialize configuration
program
  .command('init')
  .description('Initialize Studio configuration')
  .action(async () => {
    const configPath = path.join(process.env.HOME!, '.studio', 'config.json');
    
    // Prompt for API key
    const apiKey = await prompt('Enter your API key: ');
    
    await fs.mkdir(path.dirname(configPath), { recursive: true });
    await fs.writeFile(configPath, JSON.stringify({
      apiKey,
      baseUrl: 'https://api.memmie.ai/studio/v1'
    }, null, 2));
    
    console.log('Configuration saved to', configPath);
  });

// Blob commands
const blob = program.command('blob');

blob
  .command('create <file>')
  .description('Create a new blob from file')
  .option('-m, --metadata <items...>', 'Metadata key=value pairs')
  .option('-p, --providers <providers>', 'Comma-separated provider IDs')
  .action(async (file, options) => {
    const client = await getClient();
    const content = await fs.readFile(file);
    
    const metadata = parseMetadata(options.metadata);
    const providers = options.providers?.split(',');
    
    const blob = await client.blobs.create({
      content,
      metadata,
      providers
    });
    
    console.log('Created blob:', blob.id);
  });

blob
  .command('get <id>')
  .description('Get blob by ID')
  .option('-v, --version <version>', 'Specific version')
  .option('-o, --output <file>', 'Save content to file')
  .action(async (id, options) => {
    const client = await getClient();
    const blob = await client.blobs.get(id, {
      version: options.version
    });
    
    if (options.output) {
      await fs.writeFile(options.output, blob.content);
      console.log('Content saved to', options.output);
    } else {
      console.log(JSON.stringify(blob, null, 2));
    }
  });

// Provider commands
const provider = program.command('provider');

provider
  .command('process <provider-id> <blob-id>')
  .description('Process blob with provider')
  .option('-c, --config <items...>', 'Configuration key=value pairs')
  .action(async (providerId, blobId, options) => {
    const client = await getClient();
    const config = parseMetadata(options.config);
    
    console.log(`Processing blob ${blobId} with ${providerId}...`);
    
    const result = await client.providers.process(providerId, blobId, config);
    
    console.log('Processing complete:', result);
  });

// Watch command
program
  .command('watch <blob-id>')
  .description('Watch blob for changes')
  .action(async (blobId) => {
    const client = await getClient();
    
    await client.connect();
    await client.subscribe([`blob:${blobId}`]);
    
    console.log(`Watching blob ${blobId} for changes...`);
    
    client.on('blob.updated', (data) => {
      console.log('Blob updated:', data);
    });
    
    client.on('provider.completed', (data) => {
      console.log('Provider completed:', data);
    });
    
    // Keep process alive
    process.stdin.resume();
  });

// REPL mode
program
  .command('repl')
  .description('Start interactive REPL')
  .action(async () => {
    const repl = require('repl');
    const client = await getClient();
    
    const replServer = repl.start({
      prompt: 'studio> ',
    });
    
    replServer.context.studio = client;
    replServer.context.help = () => {
      console.log(`
Available objects:
  studio - StudioClient instance
  
Examples:
  await studio.blobs.list()
  await studio.blobs.create({ content: "Hello" })
  await studio.providers.list()
      `);
    };
  });

program.parse();

async function getClient(): Promise<StudioClient> {
  const configPath = path.join(process.env.HOME!, '.studio', 'config.json');
  const config = JSON.parse(await fs.readFile(configPath, 'utf-8'));
  
  return new StudioClient(config);
}

function parseMetadata(items?: string[]): Record<string, string> {
  if (!items) return {};
  
  return items.reduce((acc, item) => {
    const [key, value] = item.split('=');
    acc[key] = value;
    return acc;
  }, {} as Record<string, string>);
}
```

This comprehensive client SDK design provides full-featured libraries for JavaScript/TypeScript, Python, and Go, along with a powerful CLI tool for interacting with the Memmie Studio system.