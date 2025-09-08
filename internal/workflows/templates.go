package workflows

import (
	"fmt"
	"time"
)

// WorkflowTemplate represents a reusable workflow template
type WorkflowTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	Variables   []TemplateVariable     `json:"variables"`
	Workflow    *BlobProcessingWorkflow `json:"workflow"`
	Tags        []string               `json:"tags"`
	CreatedAt   time.Time              `json:"created_at"`
}

// TemplateVariable represents a configurable variable in a template
type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, number, boolean, array, object
	Description  string      `json:"description"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Required     bool        `json:"required"`
	Options      []string    `json:"options,omitempty"` // For enum types
}

// CreateBookWritingWorkflow creates a workflow for book writing assistance
func CreateBookWritingWorkflow(bookID, authorID string) *BlobProcessingWorkflow {
	workflow := &BlobProcessingWorkflow{
		ID:          fmt.Sprintf("book_%s_workflow", bookID),
		ProviderID:  fmt.Sprintf("book:%s", bookID),
		Name:        "Book Writing Assistant",
		Description: "Processes chapters and generates expansions, summaries, and consistency checks",
		Type:        WorkflowTypeProcessBlob,
		Steps: []BlobProcessingStep{
			{
				ID:         "validate_chapter",
				Name:       "Validate Chapter Structure",
				ProviderID: "validator",
				Type:       "validate",
				InputMap: map[string]interface{}{
					"content":       "$.blob.content",
					"chapter_number": "$.blob.metadata.chapter_number",
					"expected_schema": "chapter_schema_v1",
				},
				Config: StepConfig{
					Timeout:    30,
					MaxRetries: 2,
				},
				OnFailure: "fail",
			},
			{
				ID:         "expand_content",
				Name:       "Expand Chapter Content",
				ProviderID: "ai-expander",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"content": "$.blob.content",
					"style":   "$.provider.config.writing_style",
					"prompt":  "Expand this chapter section with more descriptive details while maintaining the author's voice",
				},
				Dependencies: []string{"validate_chapter"},
				Config: StepConfig{
					Timeout:      60,
					MaxRetries:   3,
					CacheResults: true,
					CacheTTL:     3600,
					Parameters: map[string]interface{}{
						"model":       "gpt-4",
						"temperature": 0.7,
						"max_tokens":  2000,
					},
				},
				OnFailure: "skip",
			},
			{
				ID:         "check_consistency",
				Name:       "Check Character Consistency",
				ProviderID: "consistency-checker",
				Type:       "validate",
				InputMap: map[string]interface{}{
					"content":     "$.blob.content",
					"book_id":     bookID,
					"chapter_num": "$.blob.metadata.chapter_number",
				},
				Dependencies: []string{"validate_chapter"},
				Config: StepConfig{
					Timeout:           45,
					ParallelExecution: true,
					Parameters: map[string]interface{}{
						"check_characters": true,
						"check_timeline":   true,
						"check_locations":  true,
					},
				},
			},
			{
				ID:         "generate_summary",
				Name:       "Generate Chapter Summary",
				ProviderID: "summarizer",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"content": "$.blob.content",
					"type":    "chapter_summary",
					"length":  "medium",
				},
				Dependencies: []string{"expand_content"},
				Config: StepConfig{
					Timeout:      30,
					CacheResults: true,
					CacheTTL:     7200,
				},
			},
			{
				ID:         "update_outline",
				Name:       "Update Book Outline",
				ProviderID: "outline-manager",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"book_id":         bookID,
					"chapter_number":  "$.blob.metadata.chapter_number",
					"chapter_summary": "$.steps.generate_summary.output",
					"word_count":      "$.blob.metadata.word_count",
				},
				Dependencies: []string{"generate_summary"},
				Config: StepConfig{
					Timeout: 15,
				},
			},
		},
		Config: ProcessingConfig{
			MaxConcurrency:   3,
			StopOnError:      false,
			EnableRollback:   true,
			TrackLineage:     true,
			EmitEvents:       true,
			AutoRetry:        true,
			RetryDelay:       5,
			MaxExecutionTime: 300,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	return workflow
}

// CreateResearchWorkflow creates a workflow for research document processing
func CreateResearchWorkflow(topicID string) *BlobProcessingWorkflow {
	workflow := &BlobProcessingWorkflow{
		ID:          fmt.Sprintf("research_%s_workflow", topicID),
		ProviderID:  fmt.Sprintf("research:%s", topicID),
		Name:        "Research Document Processor",
		Description: "Extracts citations, key points, and finds related papers",
		Type:        WorkflowTypeProcessBlob,
		Steps: []BlobProcessingStep{
			{
				ID:         "extract_metadata",
				Name:       "Extract Document Metadata",
				ProviderID: "metadata-extractor",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"content": "$.blob.content",
					"type":    "$.blob.metadata.document_type",
				},
				Config: StepConfig{
					Timeout:    30,
					MaxRetries: 2,
				},
			},
			{
				ID:         "extract_citations",
				Name:       "Extract Citations",
				ProviderID: "citation-extractor",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"content": "$.blob.content",
					"format":  "$.blob.metadata.citation_format",
				},
				Dependencies: []string{"extract_metadata"},
				Config: StepConfig{
					Timeout:           60,
					ParallelExecution: true,
					CacheResults:      true,
					CacheTTL:          86400,
				},
			},
			{
				ID:         "extract_key_points",
				Name:       "Extract Key Points",
				ProviderID: "key-points-extractor",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"content":    "$.blob.content",
					"max_points": 10,
					"detail":     "high",
				},
				Dependencies: []string{"extract_metadata"},
				Config: StepConfig{
					Timeout:           45,
					ParallelExecution: true,
					Parameters: map[string]interface{}{
						"algorithm": "textrank",
						"language":  "en",
					},
				},
			},
			{
				ID:         "find_related",
				Name:       "Find Related Papers",
				ProviderID: "paper-finder",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"title":      "$.steps.extract_metadata.output.title",
					"abstract":   "$.steps.extract_metadata.output.abstract",
					"keywords":   "$.steps.extract_key_points.output.keywords",
					"limit":      20,
				},
				Dependencies: []string{"extract_metadata", "extract_key_points"},
				Config: StepConfig{
					Timeout:    90,
					MaxRetries: 3,
					Parameters: map[string]interface{}{
						"search_engines": []string{"arxiv", "pubmed", "semantic_scholar"},
						"min_relevance":  0.7,
					},
				},
			},
			{
				ID:         "generate_summary",
				Name:       "Generate Research Summary",
				ProviderID: "research-summarizer",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"content":     "$.blob.content",
					"key_points":  "$.steps.extract_key_points.output",
					"citations":   "$.steps.extract_citations.output",
					"summary_type": "academic",
				},
				Dependencies: []string{"extract_key_points", "extract_citations"},
				Config: StepConfig{
					Timeout:      60,
					CacheResults: true,
					CacheTTL:     7200,
				},
			},
		},
		Config: ProcessingConfig{
			MaxConcurrency:   5,
			StopOnError:      false,
			EnableRollback:   true,
			TrackLineage:     true,
			EmitEvents:       true,
			AutoRetry:        true,
			RetryDelay:       10,
			MaxExecutionTime: 600,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	return workflow
}

// CreateCodeDocumentationWorkflow creates a workflow for code documentation
func CreateCodeDocumentationWorkflow(projectID string) *BlobProcessingWorkflow {
	workflow := &BlobProcessingWorkflow{
		ID:          fmt.Sprintf("code_doc_%s_workflow", projectID),
		ProviderID:  fmt.Sprintf("project:%s", projectID),
		Name:        "Code Documentation Generator",
		Description: "Analyzes code and generates comprehensive documentation",
		Type:        WorkflowTypeProcessBlob,
		Steps: []BlobProcessingStep{
			{
				ID:         "parse_code",
				Name:       "Parse Code Structure",
				ProviderID: "code-parser",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"content":  "$.blob.content",
					"language": "$.blob.metadata.language",
					"file_path": "$.blob.metadata.file_path",
				},
				Config: StepConfig{
					Timeout:    30,
					MaxRetries: 2,
				},
			},
			{
				ID:         "analyze_complexity",
				Name:       "Analyze Code Complexity",
				ProviderID: "complexity-analyzer",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"ast":      "$.steps.parse_code.output.ast",
					"language": "$.blob.metadata.language",
				},
				Dependencies: []string{"parse_code"},
				Config: StepConfig{
					Timeout:           20,
					ParallelExecution: true,
				},
			},
			{
				ID:         "generate_docs",
				Name:       "Generate Documentation",
				ProviderID: "doc-generator",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"ast":        "$.steps.parse_code.output.ast",
					"complexity": "$.steps.analyze_complexity.output",
					"style":      "$.provider.config.doc_style",
				},
				Dependencies: []string{"parse_code", "analyze_complexity"},
				Config: StepConfig{
					Timeout:      60,
					CacheResults: true,
					CacheTTL:     3600,
					Parameters: map[string]interface{}{
						"format":          "markdown",
						"include_examples": true,
						"generate_diagrams": true,
					},
				},
			},
			{
				ID:         "generate_tests",
				Name:       "Generate Test Cases",
				ProviderID: "test-generator",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"ast":      "$.steps.parse_code.output.ast",
					"language": "$.blob.metadata.language",
					"framework": "$.provider.config.test_framework",
				},
				Dependencies: []string{"parse_code"},
				Config: StepConfig{
					Timeout:           45,
					ParallelExecution: true,
					Parameters: map[string]interface{}{
						"coverage_target": 80,
						"include_edge_cases": true,
					},
				},
			},
			{
				ID:         "create_api_spec",
				Name:       "Create API Specification",
				ProviderID: "api-spec-generator",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"ast":        "$.steps.parse_code.output.ast",
					"docs":       "$.steps.generate_docs.output",
					"spec_format": "openapi",
				},
				Dependencies: []string{"parse_code", "generate_docs"},
				Condition:    "$.blob.metadata.is_api == true",
				Config: StepConfig{
					Timeout: 30,
					Parameters: map[string]interface{}{
						"version": "3.0",
						"include_schemas": true,
					},
				},
			},
		},
		Config: ProcessingConfig{
			MaxConcurrency:   4,
			StopOnError:      false,
			EnableRollback:   false,
			TrackLineage:     true,
			EmitEvents:       true,
			AutoRetry:        true,
			RetryDelay:       5,
			MaxExecutionTime: 300,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	return workflow
}

// CreateDataProcessingWorkflow creates a workflow for data transformation
func CreateDataProcessingWorkflow(datasetID string) *BlobProcessingWorkflow {
	workflow := &BlobProcessingWorkflow{
		ID:          fmt.Sprintf("data_%s_workflow", datasetID),
		ProviderID:  fmt.Sprintf("dataset:%s", datasetID),
		Name:        "Data Processing Pipeline",
		Description: "Validates, transforms, and enriches data",
		Type:        WorkflowTypeProcessBlob,
		Steps: []BlobProcessingStep{
			{
				ID:         "validate_schema",
				Name:       "Validate Data Schema",
				ProviderID: "schema-validator",
				Type:       "validate",
				InputMap: map[string]interface{}{
					"data":      "$.blob.content",
					"schema_id": "$.blob.metadata.schema_id",
				},
				Config: StepConfig{
					Timeout:    20,
					MaxRetries: 1,
				},
				OnFailure: "fail",
			},
			{
				ID:         "clean_data",
				Name:       "Clean and Normalize Data",
				ProviderID: "data-cleaner",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"data":   "$.blob.content",
					"rules":  "$.provider.config.cleaning_rules",
					"format": "$.blob.metadata.format",
				},
				Dependencies: []string{"validate_schema"},
				Config: StepConfig{
					Timeout: 60,
					Parameters: map[string]interface{}{
						"remove_duplicates": true,
						"handle_nulls":      "interpolate",
						"normalize_dates":   true,
					},
				},
			},
			{
				ID:         "enrich_data",
				Name:       "Enrich with External Data",
				ProviderID: "data-enricher",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"data":    "$.steps.clean_data.output",
					"sources": "$.provider.config.enrichment_sources",
				},
				Dependencies: []string{"clean_data"},
				Config: StepConfig{
					Timeout:    90,
					MaxRetries: 3,
					Parameters: map[string]interface{}{
						"match_threshold": 0.8,
						"max_enrichments": 100,
					},
				},
				OnFailure: "continue",
			},
			{
				ID:         "transform_format",
				Name:       "Transform Data Format",
				ProviderID: "format-transformer",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"data":         "$.steps.enrich_data.output",
					"source_format": "$.blob.metadata.format",
					"target_format": "$.provider.config.target_format",
				},
				Dependencies: []string{"enrich_data"},
				Config: StepConfig{
					Timeout:      30,
					CacheResults: true,
					CacheTTL:     1800,
				},
			},
			{
				ID:         "generate_report",
				Name:       "Generate Data Quality Report",
				ProviderID: "report-generator",
				Type:       "transform",
				InputMap: map[string]interface{}{
					"original_data":   "$.blob.content",
					"processed_data":  "$.steps.transform_format.output",
					"validation_results": "$.steps.validate_schema.output",
				},
				Dependencies: []string{"transform_format"},
				Config: StepConfig{
					Timeout: 45,
					Parameters: map[string]interface{}{
						"include_statistics": true,
						"include_visualizations": true,
					},
				},
			},
		},
		Config: ProcessingConfig{
			MaxConcurrency:   3,
			StopOnError:      true,
			EnableRollback:   true,
			TrackLineage:     true,
			EmitEvents:       true,
			AutoRetry:        true,
			RetryDelay:       10,
			MaxExecutionTime: 600,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	return workflow
}

// GetWorkflowTemplates returns all available workflow templates
func GetWorkflowTemplates() []WorkflowTemplate {
	return []WorkflowTemplate{
		{
			ID:          "book_writing",
			Name:        "Book Writing Assistant",
			Category:    "creative",
			Description: "Complete workflow for book writing including expansion, consistency checking, and outline management",
			Variables: []TemplateVariable{
				{
					Name:        "book_id",
					Type:        "string",
					Description: "Unique identifier for the book",
					Required:    true,
				},
				{
					Name:        "author_id",
					Type:        "string",
					Description: "Author's unique identifier",
					Required:    true,
				},
				{
					Name:         "writing_style",
					Type:         "string",
					Description:  "Writing style preference",
					DefaultValue: "descriptive",
					Options:      []string{"descriptive", "concise", "poetic", "technical"},
				},
			},
			Tags:      []string{"writing", "book", "creative", "ai-assisted"},
			CreatedAt: time.Now(),
		},
		{
			ID:          "research_processor",
			Name:        "Research Document Processor",
			Category:    "academic",
			Description: "Extracts citations, key points, and finds related research papers",
			Variables: []TemplateVariable{
				{
					Name:        "topic_id",
					Type:        "string",
					Description: "Research topic identifier",
					Required:    true,
				},
				{
					Name:         "citation_format",
					Type:         "string",
					Description:  "Citation format to use",
					DefaultValue: "apa",
					Options:      []string{"apa", "mla", "chicago", "ieee"},
				},
			},
			Tags:      []string{"research", "academic", "citations", "papers"},
			CreatedAt: time.Now(),
		},
		{
			ID:          "code_documentation",
			Name:        "Code Documentation Generator",
			Category:    "development",
			Description: "Analyzes code and generates comprehensive documentation with tests",
			Variables: []TemplateVariable{
				{
					Name:        "project_id",
					Type:        "string",
					Description: "Project identifier",
					Required:    true,
				},
				{
					Name:         "doc_style",
					Type:         "string",
					Description:  "Documentation style",
					DefaultValue: "detailed",
					Options:      []string{"minimal", "standard", "detailed"},
				},
				{
					Name:         "test_framework",
					Type:         "string",
					Description:  "Test framework to use",
					DefaultValue: "jest",
					Options:      []string{"jest", "mocha", "pytest", "junit"},
				},
			},
			Tags:      []string{"code", "documentation", "testing", "api"},
			CreatedAt: time.Now(),
		},
		{
			ID:          "data_processing",
			Name:        "Data Processing Pipeline",
			Category:    "data",
			Description: "Validates, cleans, enriches, and transforms data",
			Variables: []TemplateVariable{
				{
					Name:        "dataset_id",
					Type:        "string",
					Description: "Dataset identifier",
					Required:    true,
				},
				{
					Name:         "target_format",
					Type:         "string",
					Description:  "Target data format",
					DefaultValue: "json",
					Options:      []string{"json", "csv", "parquet", "avro"},
				},
			},
			Tags:      []string{"data", "etl", "transformation", "validation"},
			CreatedAt: time.Now(),
		},
	}
}