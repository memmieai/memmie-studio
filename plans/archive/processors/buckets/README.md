# Bucket System Documentation

## Overview

This folder contains all documentation related to the dynamic bucket system that replaces fixed organizational fields (like `book_id`, `conversation_id`) with flexible, hierarchical containers.

## Document Structure

### Core Design Documents

1. **[08-bucket-system-design.md](./08-bucket-system-design.md)**
   - Complete bucket system architecture
   - Data models and relationships
   - API specifications
   - Migration strategy from fixed fields

2. **[09-bucket-examples.md](./09-bucket-examples.md)**
   - Comprehensive use case examples
   - 7 different project types illustrated
   - API usage patterns
   - Migration scripts

## Key Concepts

### What are Buckets?

Buckets are flexible, hierarchical containers that organize blobs in any structure the user needs. They replace rigid, predefined fields with a dynamic system that can represent:

- Books with chapters
- Albums with tracks
- Research projects with data
- Conversations with messages
- Any future content type

### Core Features

1. **Hierarchical Structure**: Buckets can contain other buckets, forming trees
2. **Type-Agnostic**: Any string can be a bucket type
3. **Metadata Flexibility**: Each bucket type can have custom metadata
4. **Multi-Association**: Blobs can belong to multiple buckets
5. **Granular Sharing**: Control access at the bucket level

### System Integration

The bucket system integrates with:

- **State Service**: Stores buckets and manages blob associations
- **Schema Service**: Validates bucket metadata against type-specific schemas
- **Processor Service**: Uses buckets to organize processor inputs/outputs
- **Studio API**: Provides WebSocket updates for bucket changes

## Quick Start

### Creating a Book Project

```javascript
// 1. Create root bucket for the book
const bookBucket = await createBucket({
  name: "My Novel",
  type: "book",
  metadata: {
    genre: "Science Fiction",
    target_word_count: 80000
  }
});

// 2. Create chapter buckets
const chapter1 = await createBucket({
  name: "Chapter 1: The Beginning",
  type: "chapter",
  parent_bucket_id: bookBucket.id
});

// 3. Add blobs to chapter
const blob = await createBlob({
  processor_id: "user-input",
  schema_id: "text-input-v1",
  data: { content: "It was a dark and stormy night..." },
  bucket_ids: [chapter1.id]
});
```

## Architecture Decisions

### Why Buckets?

1. **Flexibility**: Users can organize content however they want
2. **Future-Proof**: No code changes needed for new content types
3. **Simplicity**: One unified system instead of multiple fixed fields
4. **Scalability**: Hierarchical structure scales naturally

### Design Principles

1. **User-Defined Types**: The system doesn't enforce bucket types
2. **Optional Validation**: Metadata schemas are optional
3. **Loose Coupling**: Buckets don't depend on specific processors
4. **Event-Driven**: All bucket changes emit events for real-time updates

## Related Documentation

- [State Service Design](../../01-state-service-design.md) - Updated with bucket support
- [Schema Service Design](../../06-schema-service-design.md) - Includes bucket schemas
- [Processor Recommendation](../REC.md) - Uses buckets instead of fixed fields

## Implementation Status

- ✅ Bucket system design complete
- ✅ State Service integration planned
- ✅ Schema Service bucket schemas defined
- ✅ Use case examples documented
- ⏳ Implementation pending

## Future Enhancements

1. **Bucket Templates**: Pre-defined structures for common use cases
2. **Bucket Analytics**: Usage patterns and insights
3. **Bucket Marketplace**: Share bucket structures with community
4. **Smart Organization**: AI-suggested bucket structures
5. **Bulk Operations**: Move/copy multiple blobs between buckets