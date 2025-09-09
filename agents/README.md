# ReYNa Studio Agent Documentation

## Overview

This `agents/` folder contains the complete architectural documentation and implementation plans for ReYNa Studio, a revolutionary creative platform built on a dynamic bucket system for content organization.

## Documentation Structure

```
agents/
├── README.md                     # This file
└── plans_v1/                     # Version 1 MVP Plans
    ├── 00-COMPLETE-SYSTEM-ARCHITECTURE.md   # Full system vision
    ├── 01-MVP-IMPLEMENTATION-ROADMAP.md     # Step-by-step implementation
    ├── 02-BUCKET-SYSTEM-DESIGN.md           # Core bucket architecture
    ├── 03-BUCKET-USE-CASES.md               # 7 real-world examples
    ├── 04-CONVERSATION-AS-BUCKETS.md        # Chat system proof
    ├── 05-BUCKET-SYSTEM-README.md           # Bucket system overview
    ├── NEXT_QUESTIONS.md                    # Decisions needed
    └── VERIFICATION-NOTES.md                # Code review findings
```

## Quick Navigation

### For Understanding the Vision
Start with: `00-COMPLETE-SYSTEM-ARCHITECTURE.md`

### For Implementation
Start with: `01-MVP-IMPLEMENTATION-ROADMAP.md`

### For Technical Details
- Bucket System: `02-BUCKET-SYSTEM-DESIGN.md`
- Use Cases: `03-BUCKET-USE-CASES.md`
- Chat Proof: `04-CONVERSATION-AS-BUCKETS.md`

### For Decision Making
Review: `NEXT_QUESTIONS.md`

## Key Concepts

### 1. Dynamic Buckets
Replace fixed fields (book_id, conversation_id) with flexible, hierarchical containers that can represent ANY organizational structure.

### 2. Schema-Driven
Every piece of data validates against a versioned JSON Schema, ensuring type safety and enabling evolution.

### 3. Processor-Centric
All content transformations happen through registered processors that consume and produce schema-validated blobs.

### 4. Event-Driven
NATS messaging enables loose coupling between services and real-time updates via WebSocket.

## Service Architecture

### Services to Keep/Transform
- **memmie-auth** (8001): Keep as-is ✅
- **memmie-state** (8006): Transform to blob/bucket storage
- **memmie-gateway** (8000): Add WebSocket support
- **memmie-provider** (8007): Rename to memmie-processor

### Services to Create
- **memmie-schema** (8011): Schema registry and validation
- **memmie-studio** (8010): Studio API with WebSocket

### Services to Replace/Remove
- **memmie-conversation** (8002): Replace with buckets
- **memmie-core** (8004): Split into processors
- **memmie-workflow** (8005): Simplify for MVP
- **memmie-memory** (8003): Defer to Phase 2
- **memmie-media** (8009): Defer to Phase 2
- **memmie-notification** (8008): Defer to Phase 2

## Implementation Timeline

**28-Day MVP Sprint:**

- **Week 1**: Foundation (Schema Service, State transformation)
- **Week 2**: Core Systems (Studio API, WebSocket, Processors)
- **Week 3**: Features (Text Processor, Basic UI)
- **Week 4**: Polish (Testing, Documentation, Deployment)

## Critical Path

1. Schema Service (enables everything else)
2. State Service transformation (blob/bucket storage)
3. Studio API with WebSocket (user interface)
4. Text Expansion Processor (first feature)
5. Basic React UI (user interaction)

## Success Metrics

- 100 beta users active
- 1000 blobs created daily
- <500ms blob creation time
- <100ms WebSocket latency
- <1% error rate

## Next Steps

1. Review `NEXT_QUESTIONS.md` for pending decisions
2. Approve 28-day timeline in `VERIFICATION-NOTES.md`
3. Create memmie-schema and memmie-studio repositories
4. Begin Schema Service implementation (Day 1)

## Why This Architecture?

### For Users
- Organize content YOUR way (not our way)
- Real-time updates across all devices
- AI-powered content enhancement
- Never lose work with versioning

### For Developers
- Clean service boundaries
- Easy to extend with new processors
- Type-safe with schema validation
- Scales horizontally

### For Business
- Future-proof (new content types = new bucket types)
- Marketplace ready (processors can be monetized)
- Multi-tenant capable
- Cost-effective (process only what's needed)

## Contact

For questions about this architecture, refer to the detailed documentation in each file or review the verification notes for the latest findings from code analysis.