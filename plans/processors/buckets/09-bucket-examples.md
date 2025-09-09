# Bucket System Examples - Various Use Cases

## Overview

This document provides comprehensive examples of how the dynamic bucket system can be used to organize blobs for different creative and professional use cases. Each example shows the bucket hierarchy, metadata structure, and typical blob organization.

## 1. Book Writing Use Case

### Bucket Structure
```yaml
Novel Project:
  id: bucket-novel-001
  type: book
  name: "The Quantum Mirror"
  metadata:
    genre: "Science Fiction"
    target_word_count: 80000
    deadline: "2024-12-31"
    publisher: "Pending"
  
  Children:
    - Part 1:
        id: bucket-part-001
        type: book-section
        name: "The Discovery"
        
        Children:
          - Chapter 1:
              id: bucket-ch-001
              type: chapter
              name: "Unexpected Signal"
              blob_ids:
                - blob-draft-001 (text-input-v1)
                - blob-expanded-001 (expanded-text-v1)
                - blob-edited-001 (edited-text-v1)
                - blob-notes-001 (chapter-notes-v1)
          
          - Chapter 2:
              id: bucket-ch-002
              type: chapter
              name: "The Team Assembles"
    
    - Research:
        id: bucket-research-001
        type: research
        name: "Quantum Physics Research"
        blob_ids:
          - blob-article-001 (research-article-v1)
          - blob-notes-002 (research-notes-v1)
    
    - Characters:
        id: bucket-chars-001
        type: character-profiles
        name: "Character Development"
        
        Children:
          - Dr. Sarah Chen:
              id: bucket-char-001
              type: character
              blob_ids:
                - blob-profile-001 (character-profile-v1)
                - blob-backstory-001 (character-backstory-v1)
```

### Example API Calls
```javascript
// Create the novel bucket
POST /api/v1/users/{user_id}/buckets
{
  "name": "The Quantum Mirror",
  "type": "book",
  "metadata": {
    "genre": "Science Fiction",
    "target_word_count": 80000
  }
}

// Add a chapter bucket
POST /api/v1/users/{user_id}/buckets
{
  "name": "Unexpected Signal",
  "type": "chapter",
  "parent_bucket_id": "bucket-part-001",
  "metadata": {
    "chapter_number": 1,
    "target_words": 3500
  }
}

// Create draft blob in chapter
POST /api/v1/users/{user_id}/blobs
{
  "processor_id": "user-input",
  "schema_id": "text-input-v1",
  "data": {
    "content": "The signal arrived at 3:47 AM...",
    "style": "creative"
  },
  "bucket_ids": ["bucket-ch-001"],
  "metadata": {
    "title": "Chapter 1 Draft",
    "version": "draft"
  }
}
```

## 2. Music Album Production

### Bucket Structure
```yaml
Album Project:
  id: bucket-album-001
  type: album
  name: "Echoes of Tomorrow"
  metadata:
    genre: "Electronic/Ambient"
    release_date: "2024-08-15"
    label: "Independent"
    total_tracks: 12
  
  Children:
    - Track 01:
        id: bucket-track-001
        type: track
        name: "Dawn Breaking"
        metadata:
          duration: "4:32"
          bpm: 120
          key: "C minor"
        
        Children:
          - Stems:
              id: bucket-stems-001
              type: audio-stems
              blob_ids:
                - blob-drums-001 (audio-stem-v1)
                - blob-bass-001 (audio-stem-v1)
                - blob-synth-001 (audio-stem-v1)
          
          - Versions:
              id: bucket-versions-001
              type: track-versions
              blob_ids:
                - blob-demo-001 (audio-track-v1)
                - blob-mix-v1-001 (audio-track-v1)
                - blob-master-001 (audio-track-v1)
          
          - Lyrics:
              id: bucket-lyrics-001
              type: lyrics
              blob_ids:
                - blob-lyrics-001 (lyrics-v1)
                - blob-lyrics-trans-001 (lyrics-translation-v1)
    
    - Album Art:
        id: bucket-art-001
        type: album-artwork
        name: "Visual Assets"
        
        Children:
          - Cover Art:
              id: bucket-cover-001
              type: artwork
              blob_ids:
                - blob-cover-draft-001 (image-draft-v1)
                - blob-cover-final-001 (image-final-v1)
    
    - Marketing:
        id: bucket-marketing-001
        type: marketing
        blob_ids:
          - blob-press-release-001 (press-release-v1)
          - blob-social-posts-001 (social-media-pack-v1)
```

## 3. Academic Research Project

### Bucket Structure
```yaml
Research Project:
  id: bucket-research-001
  type: research-project
  name: "Climate Change Impact on Coral Reefs"
  metadata:
    institution: "Marine Biology Institute"
    grant_number: "NSF-2024-789"
    start_date: "2024-01-15"
    end_date: "2025-12-31"
  
  Children:
    - Literature Review:
        id: bucket-lit-review-001
        type: literature
        
        Children:
          - Primary Sources:
              id: bucket-primary-001
              type: sources
              blob_ids:
                - blob-paper-001 (research-paper-v1)
                - blob-paper-002 (research-paper-v1)
          
          - Annotations:
              id: bucket-annotations-001
              type: annotations
              blob_ids:
                - blob-notes-001 (research-notes-v1)
                - blob-synthesis-001 (literature-synthesis-v1)
    
    - Data Collection:
        id: bucket-data-001
        type: data
        
        Children:
          - Field Data:
              id: bucket-field-001
              type: field-data
              metadata:
                location: "Great Barrier Reef"
                dates: ["2024-03-15", "2024-03-22"]
              blob_ids:
                - blob-measurements-001 (data-csv-v1)
                - blob-photos-001 (field-photos-v1)
          
          - Analysis:
              id: bucket-analysis-001
              type: analysis
              blob_ids:
                - blob-stats-001 (statistical-analysis-v1)
                - blob-viz-001 (data-visualization-v1)
    
    - Manuscript:
        id: bucket-manuscript-001
        type: manuscript
        
        Children:
          - Drafts:
              id: bucket-drafts-001
              type: drafts
              blob_ids:
                - blob-draft-v1 (manuscript-draft-v1)
                - blob-draft-v2 (manuscript-draft-v1)
          
          - Peer Review:
              id: bucket-peer-001
              type: peer-review
              blob_ids:
                - blob-review-001 (peer-review-v1)
                - blob-response-001 (review-response-v1)
```

## 4. Business Pitch Deck

### Bucket Structure
```yaml
Pitch Deck:
  id: bucket-pitch-001
  type: pitch-deck
  name: "Series A Pitch - TechStart"
  metadata:
    stage: "Series A"
    target_raise: "$5M"
    investor_meeting: "2024-07-15"
  
  Children:
    - Executive Summary:
        id: bucket-exec-001
        type: pitch-section
        metadata:
          order: 1
        blob_ids:
          - blob-summary-draft (pitch-text-v1)
          - blob-summary-expanded (expanded-pitch-v1)
          - blob-summary-final (pitch-slide-v1)
    
    - Problem Statement:
        id: bucket-problem-001
        type: pitch-section
        metadata:
          order: 2
        blob_ids:
          - blob-problem-text (pitch-text-v1)
          - blob-market-data (market-research-v1)
    
    - Solution:
        id: bucket-solution-001
        type: pitch-section
        metadata:
          order: 3
        
        Children:
          - Product Demo:
              id: bucket-demo-001
              type: demo
              blob_ids:
                - blob-demo-script (demo-script-v1)
                - blob-demo-video (demo-video-v1)
    
    - Financials:
        id: bucket-financials-001
        type: pitch-section
        metadata:
          order: 4
        blob_ids:
          - blob-projections (financial-model-v1)
          - blob-charts (financial-charts-v1)
    
    - Supporting Documents:
        id: bucket-support-001
        type: supporting-docs
        blob_ids:
          - blob-cap-table (cap-table-v1)
          - blob-customer-letters (testimonials-v1)
```

## 5. Personal Journal/Blog

### Bucket Structure
```yaml
Personal Blog:
  id: bucket-blog-001
  type: blog
  name: "Digital Nomad Chronicles"
  metadata:
    author: "Alex Chen"
    url: "digitalnomadchronicles.com"
  
  Children:
    - 2024:
        id: bucket-year-2024
        type: year-archive
        
        Children:
          - January:
              id: bucket-jan-2024
              type: month-archive
              
              Children:
                - Bali Adventures:
                    id: bucket-post-001
                    type: blog-post
                    metadata:
                      published: "2024-01-15"
                      tags: ["travel", "bali", "remote-work"]
                    blob_ids:
                      - blob-draft-001 (blog-draft-v1)
                      - blob-expanded-001 (expanded-blog-v1)
                      - blob-images-001 (blog-images-v1)
                      - blob-published-001 (blog-published-v1)
    
    - Drafts:
        id: bucket-drafts-001
        type: drafts
        blob_ids:
          - blob-idea-001 (blog-idea-v1)
          - blob-outline-001 (blog-outline-v1)
    
    - Resources:
        id: bucket-resources-001
        type: resources
        
        Children:
          - Photos:
              id: bucket-photos-001
              type: photo-library
              blob_ids:
                - blob-photo-001 (photo-v1)
          
          - Templates:
              id: bucket-templates-001
              type: templates
              blob_ids:
                - blob-template-001 (blog-template-v1)
```

## 6. Collaborative Workspace

### Bucket Structure
```yaml
Team Project:
  id: bucket-team-001
  type: team-workspace
  name: "Product Launch Q3"
  metadata:
    team: "Product Team Alpha"
    deadline: "2024-09-30"
    members: ["user-001", "user-002", "user-003"]
  
  Children:
    - Planning:
        id: bucket-planning-001
        type: planning
        shared_with: ["user-001", "user-002", "user-003"]
        
        Children:
          - Requirements:
              id: bucket-req-001
              type: requirements
              blob_ids:
                - blob-prd-001 (product-requirements-v1)
                - blob-specs-001 (technical-specs-v1)
          
          - Timeline:
              id: bucket-timeline-001
              type: timeline
              blob_ids:
                - blob-gantt-001 (project-timeline-v1)
    
    - Design:
        id: bucket-design-001
        type: design
        shared_with: ["user-002"]
        
        Children:
          - Mockups:
              id: bucket-mockups-001
              type: mockups
              blob_ids:
                - blob-wireframe-001 (design-wireframe-v1)
                - blob-hifi-001 (design-hifi-v1)
    
    - Development:
        id: bucket-dev-001
        type: development
        shared_with: ["user-001", "user-003"]
        blob_ids:
          - blob-arch-001 (architecture-doc-v1)
          - blob-api-001 (api-spec-v1)
    
    - Marketing:
        id: bucket-marketing-001
        type: marketing
        shared_with: ["user-002", "user-003"]
        blob_ids:
          - blob-campaign-001 (marketing-campaign-v1)
          - blob-copy-001 (marketing-copy-v1)
```

## 7. Learning Course Creation

### Bucket Structure
```yaml
Online Course:
  id: bucket-course-001
  type: course
  name: "Introduction to Machine Learning"
  metadata:
    duration: "8 weeks"
    level: "beginner"
    platform: "online"
  
  Children:
    - Module 1:
        id: bucket-module-001
        type: course-module
        name: "Foundations of ML"
        metadata:
          week: 1
          duration: "2 hours"
        
        Children:
          - Lesson 1.1:
              id: bucket-lesson-001
              type: lesson
              name: "What is Machine Learning?"
              blob_ids:
                - blob-script-001 (lesson-script-v1)
                - blob-slides-001 (lesson-slides-v1)
                - blob-video-001 (lesson-video-v1)
          
          - Exercises:
              id: bucket-exercises-001
              type: exercises
              blob_ids:
                - blob-quiz-001 (quiz-v1)
                - blob-assignment-001 (assignment-v1)
    
    - Resources:
        id: bucket-resources-001
        type: course-resources
        blob_ids:
          - blob-reading-001 (reading-list-v1)
          - blob-datasets-001 (sample-datasets-v1)
```

## API Usage Patterns

### Creating Nested Bucket Structure
```javascript
// Create root bucket
const rootBucket = await createBucket({
  name: "My Project",
  type: "project"
});

// Create child buckets
const childBucket = await createBucket({
  name: "Chapter 1",
  type: "chapter",
  parent_bucket_id: rootBucket.id
});

// Add blob to multiple buckets
const blob = await createBlob({
  processor_id: "text-expansion",
  schema_id: "expanded-text-v1",
  data: { /* ... */ },
  bucket_ids: [rootBucket.id, childBucket.id]
});
```

### Querying Buckets
```javascript
// Get all buckets of a type
const books = await getBuckets({ type: "book" });

// Get bucket with full hierarchy
const bucketTree = await getBucketTree(bucketId);

// Get all blobs in a bucket
const blobs = await getBlobsByBucket(bucketId);

// Get buckets shared with user
const sharedBuckets = await getSharedBuckets(userId);
```

### Moving and Reorganizing
```javascript
// Move bucket to new parent
await moveBucket(bucketId, newParentId);

// Add existing blob to bucket
await addBlobToBucket(blobId, bucketId);

// Remove blob from bucket
await removeBlobFromBucket(blobId, bucketId);
```

## Bucket Metadata Schemas

### Book Bucket Metadata
```json
{
  "genre": "string",
  "target_word_count": "number",
  "isbn": "string",
  "publisher": "string",
  "deadline": "date"
}
```

### Music Track Metadata
```json
{
  "duration": "string",
  "bpm": "number",
  "key": "string",
  "tempo": "string",
  "instruments": ["array", "of", "strings"]
}
```

### Research Project Metadata
```json
{
  "institution": "string",
  "grant_number": "string",
  "pi_name": "string",
  "keywords": ["array"],
  "discipline": "string"
}
```

## Benefits of Dynamic Buckets

1. **Flexibility**: Support any organizational structure without code changes
2. **Hierarchy**: Natural parent-child relationships for complex projects
3. **Sharing**: Granular access control at bucket level
4. **Discovery**: Easy to browse and find related content
5. **Portability**: Move entire bucket trees between projects
6. **Templates**: Create bucket templates for common structures
7. **Analytics**: Track usage and patterns at bucket level
8. **Collaboration**: Share specific buckets with team members

## Migration from Fixed Fields

For existing systems with fixed fields like `book_id` or `conversation_id`:

```javascript
// Old approach
blob.book_id = "book-123";
blob.conversation_id = "conv-456";

// New approach with buckets
blob.bucket_ids = ["bucket-book-123", "bucket-conv-456"];

// Migration script
async function migrateToBuckets() {
  const blobs = await getAllBlobs();
  
  for (const blob of blobs) {
    const bucketIds = [];
    
    // Create bucket for book if needed
    if (blob.book_id) {
      const bookBucket = await findOrCreateBucket({
        type: "book",
        legacy_id: blob.book_id
      });
      bucketIds.push(bookBucket.id);
    }
    
    // Create bucket for conversation if needed
    if (blob.conversation_id) {
      const convBucket = await findOrCreateBucket({
        type: "conversation",
        legacy_id: blob.conversation_id
      });
      bucketIds.push(convBucket.id);
    }
    
    // Update blob with bucket IDs
    await updateBlob(blob.id, { bucket_ids: bucketIds });
  }
}
```

This dynamic bucket system provides unlimited flexibility for organizing blobs while maintaining structure and relationships.