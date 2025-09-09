# Next Questions for ReYNa Studio MVP Refinement

## Critical Architecture Decisions

### 1. Database Strategy
**Question:** Should we use a single MongoDB database for all blob/bucket storage, or separate databases per service?

**Considerations:**
- Single DB: Easier transactions, simpler deployment
- Multiple DBs: Better isolation, independent scaling
- **Recommendation:** Single MongoDB with proper collections and indexes for MVP

### 2. Schema Service Database
**Question:** PostgreSQL vs MongoDB for schema storage?

**Considerations:**
- PostgreSQL: ACID compliance, better for relational data, JSON support
- MongoDB: Consistency with State Service, flexible schema evolution
- **Recommendation:** PostgreSQL for schemas (immutable, versioned, relational)

### 3. WebSocket Architecture
**Question:** Should WebSocket connections go through Gateway or direct to Studio API?

**Considerations:**
- Through Gateway: Centralized auth, load balancing
- Direct to Studio: Lower latency, simpler implementation
- **Recommendation:** Through Gateway with sticky sessions for MVP

### 4. Event Bus Configuration
**Question:** How should we structure NATS topics for optimal routing?

**Considerations:**
- Hierarchical: `blob.created.{schema-id}.{user-id}`
- Flat: `blob-created-{schema-id}`
- **Recommendation:** Hierarchical for flexible subscription patterns

### 5. Processor Deployment Model
**Question:** Should processors run as separate services or workers within Processor Service?

**Considerations:**
- Separate services: Independent scaling, isolation
- Workers: Simpler deployment, shared resources
- **Recommendation:** Workers for MVP, services for scale

## Implementation Clarifications

### 6. User Authentication Flow
**Question:** How exactly does auth work with WebSocket connections?

**Current Understanding:**
- Initial HTTP request includes JWT
- Upgrade to WebSocket maintains auth context
- **Need to confirm:** Token refresh mechanism for long-lived connections

### 7. Bucket Permissions
**Question:** How granular should bucket permissions be?

**Options:**
- Simple: Owner-only for MVP
- Medium: Owner + explicit shares
- Complex: Role-based with inheritance
- **Recommendation:** Owner + explicit shares for MVP

### 8. Blob Size Limits
**Question:** What's the maximum blob size we should support?

**Considerations:**
- MongoDB document limit: 16MB
- Network transfer considerations
- Storage costs
- **Recommendation:** 10MB for MVP, use GridFS for larger

### 9. Schema Evolution Strategy
**Question:** How do we handle schema updates for existing blobs?

**Options:**
- Lazy migration: Update on read
- Eager migration: Batch update all blobs
- Versioned reading: Support multiple versions
- **Recommendation:** Versioned reading with lazy migration

### 10. Processor Configuration
**Question:** How do users configure processor settings?

**Options:**
- Global defaults only
- Per-user settings
- Per-bucket settings
- **Recommendation:** Per-user settings stored in Processor Service

## Technical Implementation

### 11. Caching Strategy
**Question:** What should we cache and where?

**Candidates:**
- Schemas: Redis, 1-hour TTL
- User auth: Redis, 15-min TTL
- Bucket structure: Redis, 5-min TTL
- **Need to decide:** Cache blob content or just metadata?

### 12. Search Implementation
**Question:** Do we need search in MVP?

**Options:**
- No search: Browse only
- Basic search: MongoDB text indexes
- Advanced search: Elasticsearch later
- **Recommendation:** MongoDB text indexes for title/preview search

### 13. Batch Operations
**Question:** Should we support batch blob/bucket operations?

**Use cases:**
- Moving multiple blobs between buckets
- Bulk delete
- Mass organization
- **Recommendation:** Yes, with limits (100 items max)

### 14. Rate Limiting
**Question:** What rate limits should we enforce?

**Suggested Limits:**
- API calls: 100/minute per user
- WebSocket messages: 10/second per connection
- Blob creation: 30/minute per user
- **Need to confirm:** Are these reasonable?

### 15. Monitoring & Logging
**Question:** What metrics are critical for MVP?

**Proposed Metrics:**
- Blob creation latency
- WebSocket connection count
- Schema validation failures
- Processor execution time
- **Missing:** What about error tracking?

## Business Logic

### 16. User Quotas
**Question:** What limits should we impose on users?

**Proposed for MVP:**
- Blobs: 10,000 per user
- Buckets: 1,000 per user
- Storage: 10GB per user
- **Need to decide:** How to handle quota exceeded?

### 17. Processor Marketplace
**Question:** Should MVP include any marketplace features?

**Options:**
- No marketplace: Built-in processors only
- Read-only: Browse available processors
- Full marketplace: Share and monetize
- **Recommendation:** Built-in only for MVP

### 18. Data Export
**Question:** How do users export their data?

**Options:**
- No export in MVP
- JSON export of buckets/blobs
- Full backup with media
- **Recommendation:** JSON export for MVP

### 19. Collaboration Features
**Question:** Any collaboration in MVP?

**Current Plan:** No collaboration in MVP
**Concern:** Users might expect basic sharing
**Recommendation:** Read-only bucket sharing via public links?

### 20. Mobile Support
**Question:** Should MVP include mobile web support?

**Considerations:**
- Responsive web: Yes, minimal effort
- Native apps: No, too much work
- PWA: Maybe, if time permits
- **Recommendation:** Responsive web only

## Deployment & Operations

### 21. Environment Strategy
**Question:** How many environments do we need?

**Proposed:**
- Local: Docker Compose
- Staging: Kubernetes
- Production: Kubernetes
- **Need to decide:** CI/CD pipeline?

### 22. Database Backups
**Question:** What's our backup strategy?

**Options:**
- MongoDB Atlas: Automated backups
- Self-managed: Mongodump cron jobs
- **Recommendation:** MongoDB Atlas for simplicity

### 23. Secret Management
**Question:** How do we manage secrets?

**Options:**
- Environment variables
- Kubernetes secrets
- HashiCorp Vault
- **Recommendation:** K8s secrets for MVP

### 24. Domain & SSL
**Question:** What domain structure should we use?

**Proposed:**
- API: api.reyna.studio
- WebSocket: ws.reyna.studio
- Web: app.reyna.studio
- **Need to confirm:** Domain ownership?

### 25. Error Handling
**Question:** How do we handle and report errors?

**Proposed:**
- User-facing: Generic error messages
- Logging: Structured logs to stdout
- Monitoring: Prometheus metrics
- **Missing:** Error reporting service (Sentry)?

## Risk Assessment

### 26. What if Schema Service is down?
**Mitigation:** Cache schemas aggressively, fail open for reads?

### 27. What if NATS loses messages?
**Mitigation:** Use NATS Streaming for persistence?

### 28. What if a processor hangs?
**Mitigation:** Timeouts, circuit breakers, health checks

### 29. What if MongoDB runs out of space?
**Mitigation:** Monitor disk usage, alert at 80%

### 30. What if we hit scalability limits?
**Mitigation:** Design for horizontal scaling from day 1

## Final Considerations

### 31. Testing Strategy
**Question:** What's the minimum viable test coverage?

**Proposed:**
- Unit tests: Critical paths only (auth, blob creation)
- Integration tests: Full flow test
- E2E tests: One happy path
- **Target:** 60% coverage for MVP?

### 32. Documentation
**Question:** What documentation is essential?

**Must Have:**
- API documentation (OpenAPI)
- WebSocket protocol
- Deployment guide
- **Nice to Have:** Video tutorials?

### 33. Launch Strategy
**Question:** How do we roll out the MVP?

**Options:**
- Closed beta: 10 users
- Open beta: 100 users
- Public launch: Anyone
- **Recommendation:** Closed beta first

### 34. Success Metrics
**Question:** How do we measure MVP success?

**Proposed Metrics:**
- 100 active users in first month
- 1000 blobs created daily
- <1% error rate
- **Missing:** User satisfaction metric?

### 35. Timeline Reality Check
**Question:** Is 25 days realistic for MVP?

**Concerns:**
- Might need 5-10 more days for testing
- UI might take longer than estimated
- Integration always has surprises
- **Recommendation:** Plan for 30-35 days?

## Decisions Needed Before Starting

1. **Confirm database strategy** (single vs multiple)
2. **Approve quota limits** for MVP users
3. **Decide on error tracking service** (Sentry? Rollbar?)
4. **Confirm domain and hosting** details
5. **Approve simplified feature set** (no collaboration? no search?)
6. **Set Go/No-Go criteria** for launch
7. **Establish rollback plan** if issues arise
8. **Define success metrics** for beta
9. **Confirm team availability** for 30-35 days
10. **Approve budget** for cloud services

## Recommended Next Actions

1. **Architecture Review:** 2-hour session to finalize decisions
2. **Prototype Critical Path:** Build blob creation flow end-to-end
3. **Validate Performance:** Test WebSocket with 100 connections
4. **Security Audit:** Review auth flow and data isolation
5. **Create Project Board:** Break down into specific tasks
6. **Set Up Infrastructure:** Provision databases and services
7. **Begin Schema Service:** Most critical dependency

**Ready to proceed with these clarifications?**