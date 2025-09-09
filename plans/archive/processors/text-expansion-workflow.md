# Text Expansion Processor - Complete Workflow Implementation

## Overview

The Text Expansion Processor is the flagship processor for ReYNa Studio's book writing feature. It takes short text inputs and expands them into detailed, engaging prose using AI, while maintaining the author's style and intent.

## Processor Registration

### Processor Configuration
```yaml
id: text-expansion
name: Text Expansion Processor
description: Expands brief text into detailed prose for creative writing

# Schema definitions
input_schema_id: text-input-v1
output_schema_id: expanded-text-v1

# Event subscriptions
subscribe_events:
  - blob.created.text-input-v1
  - blob.updated.text-input-v1
  - expansion.requested

emit_events:
  - blob.created.expanded-text-v1
  - expansion.completed
  - expansion.failed

# Processing configuration
max_concurrency: 10
timeout_seconds: 30
retry_policy:
  max_attempts: 3
  backoff_ms: 1000
  max_backoff_ms: 10000

# AI configuration
ai_model: gpt-4
ai_temperature: 0.7
ai_max_tokens: 2000
```

## Schema Definitions

### Input Schema: text-input-v1
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "text-input-v1",
  "title": "Text Input for Expansion",
  "type": "object",
  "required": ["content"],
  "properties": {
    "content": {
      "type": "string",
      "description": "The text to expand",
      "minLength": 10,
      "maxLength": 10000
    },
    "style": {
      "type": "string",
      "description": "Writing style preference",
      "enum": ["formal", "casual", "creative", "technical", "poetic"],
      "default": "creative"
    },
    "expansion_level": {
      "type": "integer",
      "description": "How much to expand (1-5)",
      "minimum": 1,
      "maximum": 5,
      "default": 3
    },
    "context": {
      "type": "object",
      "description": "Additional context for expansion",
      "properties": {
        "book_id": {"type": "string"},
        "chapter_number": {"type": "integer"},
        "genre": {"type": "string"},
        "previous_text": {"type": "string", "maxLength": 1000},
        "character_names": {
          "type": "array",
          "items": {"type": "string"}
        },
        "setting": {"type": "string"},
        "tone": {"type": "string"}
      }
    },
    "constraints": {
      "type": "object",
      "description": "Expansion constraints",
      "properties": {
        "target_words": {"type": "integer", "minimum": 50, "maximum": 5000},
        "preserve_dialogue": {"type": "boolean", "default": true},
        "maintain_tense": {"type": "boolean", "default": true},
        "avoid_words": {
          "type": "array",
          "items": {"type": "string"}
        }
      }
    }
  }
}
```

### Output Schema: expanded-text-v1
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "expanded-text-v1",
  "title": "Expanded Text Output",
  "type": "object",
  "required": ["original", "expanded", "metrics"],
  "properties": {
    "original": {
      "type": "string",
      "description": "Original input text"
    },
    "expanded": {
      "type": "string",
      "description": "Expanded version of the text"
    },
    "sections": {
      "type": "array",
      "description": "Expanded text broken into sections",
      "items": {
        "type": "object",
        "properties": {
          "type": {
            "type": "string",
            "enum": ["narrative", "dialogue", "description", "action"]
          },
          "content": {"type": "string"},
          "expansion_ratio": {"type": "number"}
        }
      }
    },
    "metrics": {
      "type": "object",
      "required": ["expansion_ratio", "word_count_original", "word_count_expanded"],
      "properties": {
        "expansion_ratio": {"type": "number", "minimum": 1.0},
        "word_count_original": {"type": "integer"},
        "word_count_expanded": {"type": "integer"},
        "character_count_original": {"type": "integer"},
        "character_count_expanded": {"type": "integer"},
        "sentence_count_original": {"type": "integer"},
        "sentence_count_expanded": {"type": "integer"},
        "readability_score": {"type": "number"},
        "processing_time_ms": {"type": "integer"}
      }
    },
    "style_analysis": {
      "type": "object",
      "properties": {
        "detected_style": {"type": "string"},
        "tone": {"type": "string"},
        "voice": {"type": "string", "enum": ["first-person", "second-person", "third-person"]},
        "tense": {"type": "string", "enum": ["past", "present", "future", "mixed"]},
        "complexity_level": {"type": "integer", "minimum": 1, "maximum": 10},
        "vocabulary_richness": {"type": "number"}
      }
    },
    "suggestions": {
      "type": "array",
      "description": "Additional suggestions for improvement",
      "items": {
        "type": "object",
        "properties": {
          "type": {"type": "string"},
          "message": {"type": "string"},
          "location": {"type": "string"}
        }
      }
    },
    "processing_metadata": {
      "type": "object",
      "properties": {
        "processor_version": {"type": "string"},
        "model_used": {"type": "string"},
        "temperature": {"type": "number"},
        "tokens_used": {"type": "integer"},
        "cost_estimate": {"type": "number"}
      }
    }
  }
}
```

## Workflow Definition (YAML)

```yaml
id: wf-text-expansion
name: Text Expansion Workflow
version: 1.0.0
processor_id: text-expansion

triggers:
  - event: blob.created.text-input-v1
  - event: expansion.requested
  - api: POST /processors/text-expansion/process

input_schema: text-input-v1
output_schema: expanded-text-v1

steps:
  # Step 1: Validate and prepare input
  - id: validate-input
    name: Validate Input
    type: validation
    config:
      schema_id: text-input-v1
      strict_mode: true
    on_failure: fail
    
  # Step 2: Fetch context if book_id provided
  - id: fetch-context
    name: Fetch Book Context
    type: conditional
    condition: "$.context.book_id != null"
    steps:
      - type: api_call
        service: state-service
        method: GET
        endpoint: "/api/v1/users/{{$.user_id}}/books/{{$.context.book_id}}/context"
        output_map:
          previous_chapters: "$.chapters"
          book_style: "$.detected_style"
    on_failure: continue
    
  # Step 3: Analyze style
  - id: analyze-style
    name: Analyze Writing Style
    type: api_call
    service: ai-service
    method: POST
    endpoint: "/api/v1/analyze/style"
    input_map:
      text: "$.data.content"
      context: "$.context"
    output_map:
      detected_style: "$.style"
      tone: "$.tone"
      voice: "$.voice"
      tense: "$.tense"
    timeout_seconds: 10
    retry:
      max_attempts: 2
      backoff_ms: 1000
      
  # Step 4: Prepare expansion prompt
  - id: prepare-prompt
    name: Prepare AI Prompt
    type: transformation
    operations:
      - set: "$.prompt"
        value: |
          Expand the following text while maintaining the author's style and voice.
          
          Original text: {{$.data.content}}
          
          Style: {{$.data.style || $.detected_style}}
          Expansion level: {{$.data.expansion_level || 3}}/5
          
          Context:
          - Genre: {{$.context.genre || 'general fiction'}}
          - Setting: {{$.context.setting || 'unspecified'}}
          - Tone: {{$.tone || 'neutral'}}
          - Voice: {{$.voice || 'third-person'}}
          - Tense: {{$.tense || 'past'}}
          
          Constraints:
          - Target words: {{$.constraints.target_words || 'auto'}}
          - Preserve dialogue: {{$.constraints.preserve_dialogue || true}}
          - Maintain tense: {{$.constraints.maintain_tense || true}}
          
          Previous text for continuity:
          {{$.context.previous_text || $.previous_chapters[-1].preview || ''}}
          
          Instructions:
          1. Expand with rich descriptions and sensory details
          2. Develop character emotions and motivations
          3. Add atmospheric elements and world-building
          4. Maintain narrative flow and pacing
          5. Keep the original meaning and intent
          
  # Step 5: Call AI for expansion
  - id: expand-text
    name: Expand Text with AI
    type: api_call
    service: ai-service
    method: POST
    endpoint: "/api/v1/generate/completion"
    input_map:
      model: "{{$.processor_config.ai_model || 'gpt-4'}}"
      prompt: "$.prompt"
      temperature: "{{$.processor_config.ai_temperature || 0.7}}"
      max_tokens: "{{$.processor_config.ai_max_tokens || 2000}}"
      stop_sequences: ["[END]", "Chapter", "###"]
    output_map:
      expanded_text: "$.completion"
      tokens_used: "$.usage.total_tokens"
      model_used: "$.model"
    timeout_seconds: 30
    retry:
      max_attempts: 3
      backoff_ms: 2000
      max_backoff_ms: 10000
    on_failure: compensate
    
  # Step 6: Post-process expanded text
  - id: post-process
    name: Post-Process Expanded Text
    type: transformation
    operations:
      # Clean up any artifacts
      - replace: "$.expanded_text"
        pattern: "\\[END\\]|###"
        with: ""
      # Ensure proper formatting
      - trim: "$.expanded_text"
      # Fix dialogue punctuation
      - regex_replace: "$.expanded_text"
        pattern: '([.!?])"(\s+)([A-Z])'
        with: '$1"$2$3'
        
  # Step 7: Break into sections
  - id: analyze-sections
    name: Analyze Text Sections
    type: computation
    operations:
      - split_sections:
          input: "$.expanded_text"
          output: "$.sections"
          rules:
            - type: "dialogue"
              pattern: '"[^"]*"'
            - type: "description"
              pattern: '(The|A|An)\\s+\\w+.*\\.'
            - type: "action"
              pattern: '\\w+ed\\s+.*\\.'
            - type: "narrative"
              pattern: '.*'
              
  # Step 8: Calculate metrics
  - id: calculate-metrics
    name: Calculate Expansion Metrics
    type: computation
    operations:
      - word_count:
          input: "$.data.content"
          output: "$.metrics.word_count_original"
      - word_count:
          input: "$.expanded_text"
          output: "$.metrics.word_count_expanded"
      - calculate:
          expression: "$.metrics.word_count_expanded / $.metrics.word_count_original"
          output: "$.metrics.expansion_ratio"
      - character_count:
          input: "$.data.content"
          output: "$.metrics.character_count_original"
      - character_count:
          input: "$.expanded_text"
          output: "$.metrics.character_count_expanded"
      - sentence_count:
          input: "$.data.content"
          output: "$.metrics.sentence_count_original"
      - sentence_count:
          input: "$.expanded_text"
          output: "$.metrics.sentence_count_expanded"
      - readability_score:
          input: "$.expanded_text"
          output: "$.metrics.readability_score"
          algorithm: "flesch-kincaid"
          
  # Step 9: Generate suggestions
  - id: generate-suggestions
    name: Generate Improvement Suggestions
    type: api_call
    service: ai-service
    method: POST
    endpoint: "/api/v1/analyze/suggestions"
    input_map:
      original: "$.data.content"
      expanded: "$.expanded_text"
      style: "$.detected_style"
    output_map:
      suggestions: "$.suggestions"
    timeout_seconds: 10
    on_failure: continue  # Suggestions are optional
    
  # Step 10: Create output blob
  - id: create-blob
    name: Create Output Blob
    type: api_call
    service: state-service
    method: POST
    endpoint: "/api/v1/blobs"
    input_map:
      user_id: "$.user_id"
      processor_id: "text-expansion"
      schema_id: "expanded-text-v1"
      parent_id: "$.input_blob_id"
      data:
        original: "$.data.content"
        expanded: "$.expanded_text"
        sections: "$.sections"
        metrics: "$.metrics"
        style_analysis:
          detected_style: "$.detected_style"
          tone: "$.tone"
          voice: "$.voice"
          tense: "$.tense"
        suggestions: "$.suggestions"
        processing_metadata:
          processor_version: "1.0.0"
          model_used: "$.model_used"
          temperature: "{{$.processor_config.ai_temperature}}"
          tokens_used: "$.tokens_used"
    output_map:
      blob_id: "$.id"
      created_at: "$.created_at"
      
  # Step 11: Emit completion event
  - id: emit-event
    name: Emit Completion Event
    type: event
    config:
      event_type: "expansion.completed"
      payload:
        user_id: "$.user_id"
        input_blob_id: "$.input_blob_id"
        output_blob_id: "$.blob_id"
        expansion_ratio: "$.metrics.expansion_ratio"
        processing_time_ms: "$.execution_time"

# Error handling
error_handlers:
  - error_type: "validation_error"
    steps:
      - type: event
        event_type: "expansion.failed"
        payload:
          reason: "validation_failed"
          errors: "$.validation_errors"
          
  - error_type: "ai_service_error"
    steps:
      - type: notification
        message: "AI service temporarily unavailable, retrying..."
      - type: delay
        duration_ms: 5000
      - type: retry
        max_attempts: 3
        
  - error_type: "quota_exceeded"
    steps:
      - type: event
        event_type: "expansion.failed"
        payload:
          reason: "quota_exceeded"
          message: "Monthly AI quota exceeded"

# Compensation (rollback) actions
compensation:
  - step_id: "create-blob"
    action:
      type: api_call
      service: state-service
      method: DELETE
      endpoint: "/api/v1/blobs/{{$.blob_id}}"
```

## Implementation Code

### Processor Worker Implementation
```go
package processor

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/nats-io/nats.go"
    "github.com/memmieai/text-expansion/internal/workflow"
    "github.com/memmieai/common/pkg/events"
)

type TextExpansionProcessor struct {
    nats          *nats.Conn
    workflowEngine *workflow.Engine
    schemaClient   SchemaClient
    stateClient    StateClient
    aiClient       AIClient
    config         ProcessorConfig
}

func NewTextExpansionProcessor(config ProcessorConfig) (*TextExpansionProcessor, error) {
    // Initialize NATS connection
    nc, err := nats.Connect(config.NATSUrl)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to NATS: %w", err)
    }
    
    // Initialize workflow engine
    engine, err := workflow.NewEngine(workflow.Config{
        WorkflowPath: "./workflows/text-expansion.yaml",
        MaxWorkers:   config.MaxConcurrency,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to initialize workflow: %w", err)
    }
    
    processor := &TextExpansionProcessor{
        nats:           nc,
        workflowEngine: engine,
        schemaClient:   NewSchemaClient(config.SchemaServiceURL),
        stateClient:    NewStateClient(config.StateServiceURL),
        aiClient:       NewAIClient(config.AIServiceURL),
        config:         config,
    }
    
    // Subscribe to events
    if err := processor.subscribeToEvents(); err != nil {
        return nil, fmt.Errorf("failed to subscribe to events: %w", err)
    }
    
    return processor, nil
}

func (p *TextExpansionProcessor) subscribeToEvents() error {
    // Subscribe to blob creation events
    _, err := p.nats.Subscribe("blob.created.text-input-v1", p.handleBlobCreated)
    if err != nil {
        return err
    }
    
    // Subscribe to direct expansion requests
    _, err = p.nats.Subscribe("expansion.requested", p.handleExpansionRequest)
    if err != nil {
        return err
    }
    
    return nil
}

func (p *TextExpansionProcessor) handleBlobCreated(msg *nats.Msg) {
    var event events.BlobCreatedEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        p.logError("failed to unmarshal event", err)
        return
    }
    
    ctx := context.Background()
    if err := p.processBlob(ctx, event.BlobID, event.UserID); err != nil {
        p.logError("failed to process blob", err)
        p.emitFailureEvent(event.BlobID, err)
    }
}

func (p *TextExpansionProcessor) processBlob(ctx context.Context, blobID, userID string) error {
    // Fetch blob from State Service
    blob, err := p.stateClient.GetBlob(ctx, blobID)
    if err != nil {
        return fmt.Errorf("failed to fetch blob: %w", err)
    }
    
    // Validate against input schema
    validation, err := p.schemaClient.ValidateData(ctx, p.config.InputSchemaID, blob.Data)
    if err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    if !validation.Valid {
        return fmt.Errorf("input validation failed: %v", validation.Errors)
    }
    
    // Prepare workflow input
    workflowInput := map[string]interface{}{
        "user_id":        userID,
        "input_blob_id":  blobID,
        "data":           blob.Data,
        "processor_config": map[string]interface{}{
            "ai_model":       p.config.AIModel,
            "ai_temperature": p.config.AITemperature,
            "ai_max_tokens":  p.config.AIMaxTokens,
        },
    }
    
    // Execute workflow
    result, err := p.workflowEngine.Execute(ctx, workflowInput)
    if err != nil {
        return fmt.Errorf("workflow execution failed: %w", err)
    }
    
    // Validate output against schema
    outputValidation, err := p.schemaClient.ValidateData(ctx, p.config.OutputSchemaID, result)
    if err != nil {
        return fmt.Errorf("output validation failed: %w", err)
    }
    
    if !outputValidation.Valid {
        return fmt.Errorf("output validation failed: %v", outputValidation.Errors)
    }
    
    // Emit success event
    p.emitSuccessEvent(blobID, result["blob_id"].(string))
    
    return nil
}

func (p *TextExpansionProcessor) emitSuccessEvent(inputBlobID, outputBlobID string) {
    event := events.ExpansionCompletedEvent{
        InputBlobID:  inputBlobID,
        OutputBlobID: outputBlobID,
        ProcessorID:  "text-expansion",
        Timestamp:    time.Now(),
    }
    
    data, _ := json.Marshal(event)
    p.nats.Publish("expansion.completed", data)
}

func (p *TextExpansionProcessor) emitFailureEvent(blobID string, err error) {
    event := events.ExpansionFailedEvent{
        BlobID:      blobID,
        ProcessorID: "text-expansion",
        Error:       err.Error(),
        Timestamp:   time.Now(),
    }
    
    data, _ := json.Marshal(event)
    p.nats.Publish("expansion.failed", data)
}
```

## Testing Scenarios

### Test Case 1: Basic Expansion
```json
{
  "input": {
    "content": "The knight entered the castle.",
    "style": "creative"
  },
  "expected_output": {
    "expanded": "The armored knight, his steel plates clanking with each deliberate step, pushed through the massive oak doors and entered the ancient castle. Shadows danced across weathered stone walls as torchlight flickered in the drafty corridors, and the musty scent of centuries-old secrets filled his nostrils.",
    "metrics": {
      "expansion_ratio": 3.5,
      "word_count_original": 5,
      "word_count_expanded": 42
    }
  }
}
```

### Test Case 2: Book Context
```json
{
  "input": {
    "content": "Sarah discovered the hidden door.",
    "context": {
      "book_id": "mystery-novel",
      "chapter_number": 3,
      "genre": "mystery",
      "previous_text": "The old library held many secrets.",
      "character_names": ["Sarah", "Professor Blackwood"],
      "setting": "Victorian mansion"
    }
  },
  "expected_behavior": "Expansion should maintain mystery genre conventions, reference the library setting, and be consistent with Victorian era."
}
```

### Test Case 3: Dialogue Preservation
```json
{
  "input": {
    "content": "\"I can't believe it,\" she said. The room fell silent.",
    "constraints": {
      "preserve_dialogue": true
    }
  },
  "expected_behavior": "The dialogue should remain unchanged while the narrative around it is expanded."
}
```

## Performance Requirements

- **Latency**: < 3 seconds for 500-word input
- **Throughput**: 100 concurrent expansions
- **Success Rate**: > 99% for valid inputs
- **Expansion Quality**: 4.5/5 user satisfaction rating

## Monitoring and Metrics

### Key Metrics
- Average expansion ratio by style
- Processing time by input length
- AI token usage and costs
- User satisfaction ratings
- Error rate by error type

### Dashboard Queries
```sql
-- Daily expansion statistics
SELECT 
    DATE(created_at) as date,
    COUNT(*) as total_expansions,
    AVG((data->>'metrics'->>'expansion_ratio')::float) as avg_ratio,
    AVG((data->>'metrics'->>'processing_time_ms')::int) as avg_time_ms,
    SUM((data->>'processing_metadata'->>'tokens_used')::int) as total_tokens
FROM blobs
WHERE processor_id = 'text-expansion'
    AND created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at)
ORDER BY date DESC;

-- User satisfaction by style
SELECT 
    data->>'style' as style,
    COUNT(*) as count,
    AVG(user_rating) as avg_rating
FROM blobs b
JOIN user_ratings r ON b.id = r.blob_id
WHERE processor_id = 'text-expansion'
GROUP BY data->>'style';
```

## Cost Optimization

### Token Usage Optimization
1. Cache frequently expanded phrases
2. Use GPT-3.5-turbo for simple expansions
3. Batch similar requests
4. Implement user quotas

### Performance Optimization
1. Pre-warm AI connections
2. Cache style analysis results
3. Use connection pooling
4. Implement circuit breakers

## Future Enhancements

1. **Multi-language Support**: Expand text in 10+ languages
2. **Style Transfer**: Convert between writing styles
3. **Character Voice**: Maintain unique character voices
4. **Scene Transitions**: Intelligent scene bridging
5. **Plot Consistency**: Check for plot holes and inconsistencies
6. **Collaborative Expansion**: Multiple users editing same document
7. **Version Control**: Track all expansion versions
8. **Custom Models**: Fine-tuned models per genre