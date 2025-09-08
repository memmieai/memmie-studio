# Database Schema Design

## Overview

The database design uses PostgreSQL as the primary store with the following principles:
- Event sourcing via deltas table
- Materialized views for performance
- JSONB for flexible metadata
- Partitioning for scalability
- Proper indexing for query patterns

## Core Tables

### Users Context
```sql
-- Users are managed by auth service, but we need local context
CREATE TABLE user_contexts (
    user_id UUID PRIMARY KEY,
    settings JSONB DEFAULT '{}',
    quota_limits JSONB DEFAULT '{}',
    storage_used BIGINT DEFAULT 0,
    blob_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_contexts_storage ON user_contexts(storage_used);
CREATE INDEX idx_user_contexts_created ON user_contexts(created_at);
```

### Blobs Table
```sql
CREATE TABLE blobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    
    -- Content reference
    content_hash VARCHAR(64) NOT NULL, -- SHA-256 hash
    content_size BIGINT NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    
    -- Versioning
    version INTEGER NOT NULL DEFAULT 1,
    is_latest BOOLEAN DEFAULT TRUE,
    
    -- DAG relationships
    parent_blob_id UUID REFERENCES blobs(id) ON DELETE CASCADE,
    root_blob_id UUID, -- Original blob in chain
    depth INTEGER DEFAULT 0, -- Distance from root
    
    -- Provider tracking
    created_by VARCHAR(255) NOT NULL DEFAULT 'user', -- Provider ID or 'user'
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT fk_user FOREIGN KEY (user_id) 
        REFERENCES user_contexts(user_id) ON DELETE CASCADE,
    CONSTRAINT unique_latest_version 
        UNIQUE(id, version) WHERE is_latest = TRUE
);

-- Indexes for query performance
CREATE INDEX idx_blobs_user_id ON blobs(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_blobs_parent ON blobs(parent_blob_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_blobs_root ON blobs(root_blob_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_blobs_content_hash ON blobs(content_hash);
CREATE INDEX idx_blobs_created_by ON blobs(created_by);
CREATE INDEX idx_blobs_tags ON blobs USING GIN(tags);
CREATE INDEX idx_blobs_metadata ON blobs USING GIN(metadata);
CREATE INDEX idx_blobs_created_at ON blobs(created_at DESC);
CREATE INDEX idx_blobs_updated_at ON blobs(updated_at DESC);

-- Full-text search index
CREATE INDEX idx_blobs_search ON blobs 
    USING GIN(to_tsvector('english', metadata->>'title' || ' ' || metadata->>'description'));
```

### Blob Versions Table
```sql
-- Store all versions of blobs for history
CREATE TABLE blob_versions (
    blob_id UUID NOT NULL,
    version INTEGER NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    content_size BIGINT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    
    PRIMARY KEY (blob_id, version),
    CONSTRAINT fk_blob FOREIGN KEY (blob_id) 
        REFERENCES blobs(id) ON DELETE CASCADE
);

CREATE INDEX idx_blob_versions_created ON blob_versions(created_at DESC);
```

### Deltas Table (Event Sourcing)
```sql
CREATE TABLE deltas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blob_id UUID NOT NULL,
    user_id UUID NOT NULL,
    
    -- Operation details
    operation VARCHAR(50) NOT NULL, -- create, update, transform, delete, revert
    provider_id VARCHAR(255) NOT NULL DEFAULT 'user',
    
    -- Change data
    patch JSONB NOT NULL, -- JSON Patch or custom format
    
    -- Versioning
    from_version INTEGER NOT NULL DEFAULT 0,
    to_version INTEGER NOT NULL,
    
    -- Causality tracking
    previous_delta_id UUID REFERENCES deltas(id),
    caused_by_delta_id UUID REFERENCES deltas(id), -- Parent delta that triggered this
    caused_by_event JSONB, -- Event that triggered this delta
    
    -- Status
    status VARCHAR(50) DEFAULT 'pending', -- pending, applying, applied, failed
    error_message TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    applied_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT fk_blob FOREIGN KEY (blob_id) 
        REFERENCES blobs(id) ON DELETE CASCADE,
    CONSTRAINT fk_user FOREIGN KEY (user_id) 
        REFERENCES user_contexts(user_id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_deltas_blob_id ON deltas(blob_id);
CREATE INDEX idx_deltas_user_id ON deltas(user_id);
CREATE INDEX idx_deltas_status ON deltas(status) WHERE status != 'applied';
CREATE INDEX idx_deltas_provider ON deltas(provider_id);
CREATE INDEX idx_deltas_created ON deltas(created_at DESC);
CREATE INDEX idx_deltas_applied ON deltas(applied_at DESC) WHERE applied_at IS NOT NULL;
CREATE INDEX idx_deltas_causality ON deltas(caused_by_delta_id) WHERE caused_by_delta_id IS NOT NULL;

-- Partitioning by month for scalability
CREATE TABLE deltas_2024_01 PARTITION OF deltas
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

### DAG Edges Table
```sql
CREATE TABLE blob_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_blob_id UUID NOT NULL,
    child_blob_id UUID NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    transform_type VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT fk_parent FOREIGN KEY (parent_blob_id) 
        REFERENCES blobs(id) ON DELETE CASCADE,
    CONSTRAINT fk_child FOREIGN KEY (child_blob_id) 
        REFERENCES blobs(id) ON DELETE CASCADE,
    CONSTRAINT unique_edge UNIQUE(parent_blob_id, child_blob_id, provider_id)
);

CREATE INDEX idx_edges_parent ON blob_edges(parent_blob_id);
CREATE INDEX idx_edges_child ON blob_edges(child_blob_id);
CREATE INDEX idx_edges_provider ON blob_edges(provider_id);
```

### Content Storage Table
```sql
-- Content-addressed storage
CREATE TABLE content_storage (
    content_hash VARCHAR(64) PRIMARY KEY,
    content BYTEA, -- NULL if stored externally
    storage_backend VARCHAR(50) DEFAULT 'postgres', -- postgres, s3, filesystem
    storage_path TEXT, -- External storage path if applicable
    compression VARCHAR(20), -- none, gzip, lz4, zstd
    original_size BIGINT NOT NULL,
    compressed_size BIGINT,
    mime_type VARCHAR(255),
    reference_count INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_content_storage_accessed ON content_storage(last_accessed);
CREATE INDEX idx_content_storage_refs ON content_storage(reference_count);
```

### Providers Table
```sql
CREATE TABLE providers (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    version VARCHAR(50) NOT NULL,
    
    -- Configuration
    workflow_id VARCHAR(255),
    config JSONB DEFAULT '{}',
    
    -- Capabilities
    trigger_events TEXT[] DEFAULT '{}',
    supported_types TEXT[] DEFAULT '{}',
    max_input_size BIGINT,
    timeout_seconds INTEGER DEFAULT 30,
    
    -- Schema definitions
    input_schema JSONB,
    output_schema JSONB,
    config_schema JSONB,
    
    -- Rules
    processing_rules JSONB DEFAULT '{}',
    
    -- Status
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, deprecated
    
    -- Metadata
    author VARCHAR(255),
    license VARCHAR(100),
    documentation_url TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_provider_version UNIQUE(id, version)
);

CREATE INDEX idx_providers_status ON providers(status);
CREATE INDEX idx_providers_category ON providers(category);
CREATE INDEX idx_providers_events ON providers USING GIN(trigger_events);
CREATE INDEX idx_providers_types ON providers USING GIN(supported_types);
```

### Provider Processing State
```sql
CREATE TABLE provider_processing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blob_id UUID NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    
    -- Processing status
    status VARCHAR(50) NOT NULL, -- pending, processing, completed, failed, skipped
    
    -- Version tracking
    processed_version INTEGER,
    current_blob_version INTEGER,
    
    -- Results
    output_blob_id UUID REFERENCES blobs(id),
    output_delta_id UUID REFERENCES deltas(id),
    
    -- Execution details
    job_id VARCHAR(255), -- Temporal workflow ID
    attempt_count INTEGER DEFAULT 0,
    error_message TEXT,
    
    -- Metrics
    processing_time_ms INTEGER,
    input_size BIGINT,
    output_size BIGINT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    queued_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT fk_blob FOREIGN KEY (blob_id) 
        REFERENCES blobs(id) ON DELETE CASCADE,
    CONSTRAINT fk_provider FOREIGN KEY (provider_id) 
        REFERENCES providers(id),
    CONSTRAINT fk_user FOREIGN KEY (user_id) 
        REFERENCES user_contexts(user_id) ON DELETE CASCADE,
    CONSTRAINT unique_processing UNIQUE(blob_id, provider_id, processed_version)
);

CREATE INDEX idx_processing_blob ON provider_processing(blob_id);
CREATE INDEX idx_processing_provider ON provider_processing(provider_id);
CREATE INDEX idx_processing_status ON provider_processing(status) WHERE status IN ('pending', 'processing');
CREATE INDEX idx_processing_completed ON provider_processing(completed_at DESC) WHERE completed_at IS NOT NULL;
```

### Events Table
```sql
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL, -- Blob ID usually
    aggregate_type VARCHAR(50) NOT NULL, -- blob, provider, etc.
    user_id UUID,
    
    -- Event data
    payload JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    
    -- Processing
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP WITH TIME ZONE,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT fk_user FOREIGN KEY (user_id) 
        REFERENCES user_contexts(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_aggregate ON events(aggregate_id, aggregate_type);
CREATE INDEX idx_events_unprocessed ON events(processed, created_at) WHERE processed = FALSE;
CREATE INDEX idx_events_created ON events(created_at DESC);

-- Partitioning by day for high volume
CREATE TABLE events_2024_01_15 PARTITION OF events
    FOR VALUES FROM ('2024-01-15') TO ('2024-01-16');
```

### Jobs Table (Async Processing)
```sql
CREATE TABLE jobs (
    id VARCHAR(255) PRIMARY KEY, -- Temporal workflow ID
    job_type VARCHAR(100) NOT NULL,
    user_id UUID NOT NULL,
    
    -- Job details
    input JSONB NOT NULL,
    output JSONB,
    
    -- Status
    status VARCHAR(50) NOT NULL, -- queued, running, completed, failed, cancelled
    progress INTEGER DEFAULT 0, -- 0-100
    
    -- Error handling
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT fk_user FOREIGN KEY (user_id) 
        REFERENCES user_contexts(user_id) ON DELETE CASCADE
);

CREATE INDEX idx_jobs_status ON jobs(status) WHERE status IN ('queued', 'running');
CREATE INDEX idx_jobs_user ON jobs(user_id);
CREATE INDEX idx_jobs_created ON jobs(created_at DESC);
CREATE INDEX idx_jobs_expires ON jobs(expires_at) WHERE expires_at IS NOT NULL;
```

## Analytics Tables

### Usage Metrics
```sql
CREATE TABLE usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    metric_date DATE NOT NULL,
    
    -- Counters
    blobs_created INTEGER DEFAULT 0,
    blobs_processed INTEGER DEFAULT 0,
    deltas_applied INTEGER DEFAULT 0,
    storage_bytes_used BIGINT DEFAULT 0,
    
    -- Provider usage
    provider_invocations JSONB DEFAULT '{}', -- {provider_id: count}
    
    -- API usage
    api_calls INTEGER DEFAULT 0,
    api_errors INTEGER DEFAULT 0,
    
    -- Computed metrics
    total_processing_time_ms BIGINT DEFAULT 0,
    average_response_time_ms INTEGER,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT fk_user FOREIGN KEY (user_id) 
        REFERENCES user_contexts(user_id) ON DELETE CASCADE,
    CONSTRAINT unique_user_date UNIQUE(user_id, metric_date)
);

CREATE INDEX idx_metrics_user_date ON usage_metrics(user_id, metric_date DESC);
CREATE INDEX idx_metrics_date ON usage_metrics(metric_date DESC);
```

### Provider Metrics
```sql
CREATE TABLE provider_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id VARCHAR(255) NOT NULL,
    metric_date DATE NOT NULL,
    
    -- Performance metrics
    total_invocations INTEGER DEFAULT 0,
    successful_invocations INTEGER DEFAULT 0,
    failed_invocations INTEGER DEFAULT 0,
    
    -- Latency percentiles (milliseconds)
    latency_p50 INTEGER,
    latency_p95 INTEGER,
    latency_p99 INTEGER,
    
    -- Resource usage
    total_input_bytes BIGINT DEFAULT 0,
    total_output_bytes BIGINT DEFAULT 0,
    total_processing_time_ms BIGINT DEFAULT 0,
    
    -- Error breakdown
    error_counts JSONB DEFAULT '{}', -- {error_type: count}
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT fk_provider FOREIGN KEY (provider_id) 
        REFERENCES providers(id),
    CONSTRAINT unique_provider_date UNIQUE(provider_id, metric_date)
);

CREATE INDEX idx_provider_metrics_date ON provider_metrics(provider_id, metric_date DESC);
```

## Materialized Views

### User Blob Summary
```sql
CREATE MATERIALIZED VIEW user_blob_summary AS
SELECT 
    user_id,
    COUNT(*) as total_blobs,
    COUNT(DISTINCT root_blob_id) as root_blobs,
    SUM(content_size) as total_storage,
    COUNT(DISTINCT created_by) as providers_used,
    MAX(created_at) as last_activity,
    AVG(depth) as avg_dag_depth
FROM blobs
WHERE deleted_at IS NULL
GROUP BY user_id;

CREATE UNIQUE INDEX idx_user_blob_summary ON user_blob_summary(user_id);

-- Refresh every hour
CREATE OR REPLACE FUNCTION refresh_user_blob_summary()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_blob_summary;
END;
$$ LANGUAGE plpgsql;
```

### DAG Statistics
```sql
CREATE MATERIALIZED VIEW dag_statistics AS
WITH RECURSIVE dag_tree AS (
    -- Base case: root nodes
    SELECT 
        id as root_id,
        id as node_id,
        0 as level,
        ARRAY[id] as path
    FROM blobs
    WHERE parent_blob_id IS NULL
    
    UNION ALL
    
    -- Recursive case
    SELECT 
        dt.root_id,
        b.id as node_id,
        dt.level + 1,
        dt.path || b.id
    FROM dag_tree dt
    JOIN blobs b ON b.parent_blob_id = dt.node_id
    WHERE NOT b.id = ANY(dt.path) -- Prevent cycles
)
SELECT 
    root_id,
    COUNT(*) as total_nodes,
    MAX(level) as max_depth,
    COUNT(DISTINCT level) as unique_levels
FROM dag_tree
GROUP BY root_id;

CREATE UNIQUE INDEX idx_dag_statistics ON dag_statistics(root_id);
```

## Functions and Triggers

### Update Timestamps
```sql
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_blobs_updated_at
    BEFORE UPDATE ON blobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_providers_updated_at
    BEFORE UPDATE ON providers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

### Content Reference Counting
```sql
CREATE OR REPLACE FUNCTION update_content_references()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE content_storage 
        SET reference_count = reference_count + 1,
            last_accessed = CURRENT_TIMESTAMP
        WHERE content_hash = NEW.content_hash;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE content_storage 
        SET reference_count = reference_count - 1
        WHERE content_hash = OLD.content_hash;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER manage_content_references
    AFTER INSERT OR DELETE ON blobs
    FOR EACH ROW
    EXECUTE FUNCTION update_content_references();
```

### Event Emission
```sql
CREATE OR REPLACE FUNCTION emit_blob_event()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO events (
        event_type,
        aggregate_id,
        aggregate_type,
        user_id,
        payload
    ) VALUES (
        CASE 
            WHEN TG_OP = 'INSERT' THEN 'blob.created'
            WHEN TG_OP = 'UPDATE' THEN 'blob.updated'
            WHEN TG_OP = 'DELETE' THEN 'blob.deleted'
        END,
        COALESCE(NEW.id, OLD.id),
        'blob',
        COALESCE(NEW.user_id, OLD.user_id),
        jsonb_build_object(
            'blob_id', COALESCE(NEW.id, OLD.id),
            'operation', TG_OP,
            'version', COALESCE(NEW.version, OLD.version)
        )
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER blob_events
    AFTER INSERT OR UPDATE OR DELETE ON blobs
    FOR EACH ROW
    EXECUTE FUNCTION emit_blob_event();
```

## Indexes for Common Queries

```sql
-- Find blobs by user with pagination
CREATE INDEX idx_blobs_user_created ON blobs(user_id, created_at DESC) 
WHERE deleted_at IS NULL;

-- Find child blobs of a parent
CREATE INDEX idx_blobs_parent_created ON blobs(parent_blob_id, created_at DESC) 
WHERE deleted_at IS NULL;

-- Find blobs by provider
CREATE INDEX idx_blobs_provider_created ON blobs(created_by, created_at DESC) 
WHERE deleted_at IS NULL;

-- Find pending deltas
CREATE INDEX idx_deltas_pending ON deltas(created_at) 
WHERE status = 'pending';

-- Find processing jobs by provider
CREATE INDEX idx_processing_provider_status ON provider_processing(provider_id, status) 
WHERE status IN ('pending', 'processing');

-- Content deduplication lookup
CREATE INDEX idx_content_hash_size ON content_storage(content_hash, original_size);
```

## Partitioning Strategy

```sql
-- Partition deltas by month
CREATE TABLE deltas_2024_02 PARTITION OF deltas
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Partition events by day
CREATE TABLE events_2024_01_16 PARTITION OF events
    FOR VALUES FROM ('2024-01-16') TO ('2024-01-17');

-- Automated partition creation
CREATE OR REPLACE FUNCTION create_monthly_partitions()
RETURNS void AS $$
DECLARE
    start_date date;
    end_date date;
    partition_name text;
BEGIN
    start_date := date_trunc('month', CURRENT_DATE);
    end_date := start_date + interval '1 month';
    partition_name := 'deltas_' || to_char(start_date, 'YYYY_MM');
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF deltas FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;

-- Schedule monthly
SELECT cron.schedule('create_partitions', '0 0 1 * *', 'SELECT create_monthly_partitions()');
```

## Performance Optimizations

### Connection Pooling
```sql
-- Recommended PostgreSQL configuration
max_connections = 200
shared_buffers = 4GB
effective_cache_size = 12GB
work_mem = 16MB
maintenance_work_mem = 1GB
random_page_cost = 1.1
effective_io_concurrency = 200
```

### Vacuum and Analyze
```sql
-- Auto-vacuum settings for high-update tables
ALTER TABLE blobs SET (autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE deltas SET (autovacuum_vacuum_scale_factor = 0.1);
ALTER TABLE events SET (autovacuum_vacuum_scale_factor = 0.05);

-- Regular maintenance
CREATE OR REPLACE FUNCTION maintenance_routine()
RETURNS void AS $$
BEGIN
    -- Analyze frequently queried tables
    ANALYZE blobs;
    ANALYZE deltas;
    ANALYZE provider_processing;
    
    -- Refresh materialized views
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_blob_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY dag_statistics;
    
    -- Clean up old events
    DELETE FROM events 
    WHERE processed = TRUE 
    AND created_at < CURRENT_TIMESTAMP - interval '30 days';
    
    -- Clean up expired jobs
    DELETE FROM jobs 
    WHERE expires_at < CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- Schedule daily
SELECT cron.schedule('maintenance', '0 2 * * *', 'SELECT maintenance_routine()');
```

This comprehensive database schema provides a solid foundation for the Memmie Studio system with proper indexing, partitioning, and performance optimizations.