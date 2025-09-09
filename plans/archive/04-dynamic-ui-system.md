# Dynamic UI System Design

## Overview

The Dynamic UI System allows providers to define their optimal interface layout using JSON, which is then rendered across all platforms (web, mobile, AR). This enables a "no-code" approach where new providers can be added without changing the frontend code.

## Core Concepts

### 1. UI Layout Definition
Providers define their UI as a JSON structure that describes:
- Layout type (split, tabs, grid, stack, canvas)
- Component tree with data bindings
- Actions and interactions
- Responsive behavior

### 2. Data Binding
Components bind to data using JSONPath expressions:
- `$.current_blob` - Current blob being edited
- `$.processed_blob` - Result from provider
- `$.dag.children[0]` - First child in DAG
- `$.provider.config.style` - Provider configuration

### 3. Cross-Platform Rendering
Same JSON renders differently based on platform:
- **Web**: React components
- **Mobile**: React Native components  
- **AR**: 3D spatial layouts

## UI Layout Schema

```typescript
interface UILayout {
  type: LayoutType;
  orientation?: "horizontal" | "vertical";
  children: UIComponent[];
  responsive?: ResponsiveConfig;
  metadata?: Record<string, any>;
}

type LayoutType = 
  | "split"    // Side-by-side panels
  | "tabs"     // Tabbed interface
  | "grid"     // Grid layout
  | "stack"    // Vertical stack
  | "canvas"   // Free positioning
  | "modal"    // Overlay modal
  | "drawer"   // Slide-out drawer

interface UIComponent {
  id: string;
  type: ComponentType;
  dataSource: string;  // JSONPath expression
  props: ComponentProps;
  actions?: UIAction[];
  visibility?: string;  // Condition expression
  style?: StyleConfig;
  children?: UIComponent[];  // For container components
}

type ComponentType =
  // Content Components
  | "blob-editor"       // Text editor with syntax highlighting
  | "blob-viewer"       // Read-only content viewer
  | "markdown-preview"  // Rendered markdown
  | "code-editor"      // Monaco code editor
  | "rich-text-editor" // WYSIWYG editor
  
  // Media Components
  | "audio-player"     // Audio playback with controls
  | "video-player"     // Video playback
  | "image-viewer"     // Image with zoom/pan
  | "waveform-viz"     // Audio waveform visualization
  
  // Data Visualization
  | "dag-visualizer"   // Interactive DAG graph
  | "chart"           // Charts (line, bar, pie)
  | "metrics-panel"   // Key metrics display
  | "timeline"        // Temporal visualization
  
  // Input Components
  | "ramble-button"   // Voice input button
  | "file-upload"     // File/drag-drop upload
  | "form"           // Dynamic form
  | "search-box"     // Search with suggestions
  
  // Layout Components
  | "container"      // Generic container
  | "card"          // Card wrapper
  | "accordion"     // Collapsible sections
  | "toolbar"       // Action toolbar

interface UIAction {
  id: string;
  type: "transform" | "create" | "delete" | "navigate" | "custom";
  label: string;
  icon?: string;
  provider?: string;
  params?: Record<string, any>;
  confirmation?: string;
  hotkey?: string;
}
```

## Component Library

### Text Editor Component
```typescript
// blob-editor component
{
  "id": "main-editor",
  "type": "blob-editor",
  "dataSource": "$.current_blob.content",
  "props": {
    "language": "markdown",
    "theme": "vs-dark",
    "lineNumbers": true,
    "wordWrap": true,
    "fontSize": 14,
    "placeholder": "Start writing...",
    "autoSave": true,
    "autoSaveDelay": 1000
  },
  "actions": [
    {
      "id": "expand",
      "type": "transform",
      "label": "Expand",
      "icon": "expand",
      "provider": "text-expander",
      "hotkey": "cmd+e"
    }
  ]
}
```

### DAG Visualizer Component
```typescript
// dag-visualizer component
{
  "id": "dag-view",
  "type": "dag-visualizer",
  "dataSource": "$.dag",
  "props": {
    "layout": "hierarchical",
    "direction": "TB",  // Top to Bottom
    "nodeSize": {
      "width": 150,
      "height": 80
    },
    "interactive": true,
    "showLabels": true,
    "showMinimap": true,
    "nodeRenderer": "blob-summary",  // Custom node renderer
    "edgeStyle": "bezier",
    "zoom": {
      "min": 0.2,
      "max": 2.0,
      "fit": true
    }
  },
  "actions": [
    {
      "id": "focus-node",
      "type": "navigate",
      "label": "Open",
      "params": {
        "target": "blob-detail"
      }
    }
  ]
}
```

### Voice Input Component
```typescript
// ramble-button component
{
  "id": "ramble",
  "type": "ramble-button",
  "dataSource": null,
  "props": {
    "size": "large",
    "position": "bottom-right",
    "floating": true,
    "pulseAnimation": true,
    "maxDuration": 300,  // 5 minutes max
    "autoTranscribe": true,
    "targetProvider": "$.current_provider"
  },
  "actions": [
    {
      "id": "start-ramble",
      "type": "create",
      "label": "Start Recording",
      "params": {
        "blob_type": "ramble",
        "auto_process": true
      }
    }
  ]
}
```

## Provider UI Examples

### Book Writer Layout
```json
{
  "type": "split",
  "orientation": "horizontal",
  "responsive": {
    "breakpoint": 768,
    "mobileLayout": "tabs"
  },
  "children": [
    {
      "id": "editor-panel",
      "type": "container",
      "props": {
        "flex": 1,
        "padding": 20
      },
      "children": [
        {
          "id": "chapter-title",
          "type": "blob-viewer",
          "dataSource": "$.current_blob.metadata.title",
          "props": {
            "variant": "h2"
          }
        },
        {
          "id": "editor",
          "type": "blob-editor",
          "dataSource": "$.current_blob.content",
          "props": {
            "language": "markdown",
            "autoSave": true
          }
        },
        {
          "id": "word-count",
          "type": "metrics-panel",
          "dataSource": "$.current_blob.metrics",
          "props": {
            "metrics": ["word_count", "reading_time"]
          }
        }
      ]
    },
    {
      "id": "preview-panel",
      "type": "container",
      "props": {
        "flex": 1,
        "padding": 20,
        "background": "#f5f5f5"
      },
      "children": [
        {
          "id": "expanded-title",
          "type": "blob-viewer",
          "dataSource": "$.processed_blob.metadata.title",
          "props": {
            "variant": "h2"
          }
        },
        {
          "id": "expanded-content",
          "type": "markdown-preview",
          "dataSource": "$.processed_blob.content",
          "props": {
            "showToc": true
          }
        }
      ]
    }
  ]
}
```

### Music Generator Layout (Strudel)
```json
{
  "type": "stack",
  "children": [
    {
      "id": "description-input",
      "type": "blob-editor",
      "dataSource": "$.input_blob.content",
      "props": {
        "height": 100,
        "placeholder": "Describe the music you want...",
        "language": "text"
      },
      "actions": [
        {
          "id": "generate",
          "type": "transform",
          "label": "Generate Music",
          "icon": "music",
          "provider": "music-generator"
        }
      ]
    },
    {
      "id": "code-output",
      "type": "code-editor",
      "dataSource": "$.processed_blob.strudel_code",
      "props": {
        "language": "javascript",
        "theme": "monokai",
        "height": 300,
        "readOnly": false
      }
    },
    {
      "id": "player",
      "type": "audio-player",
      "dataSource": "$.processed_blob.audio_url",
      "props": {
        "controls": true,
        "waveform": true,
        "loop": true
      },
      "visibility": "$.processed_blob.audio_url != null"
    }
  ]
}
```

### Research Assistant Layout
```json
{
  "type": "grid",
  "props": {
    "columns": 2,
    "gap": 20,
    "responsive": {
      "mobile": {
        "columns": 1
      }
    }
  },
  "children": [
    {
      "id": "sources",
      "type": "card",
      "props": {
        "title": "Sources",
        "collapsible": true
      },
      "children": [
        {
          "id": "source-list",
          "type": "blob-viewer",
          "dataSource": "$.sources",
          "props": {
            "variant": "list"
          }
        },
        {
          "id": "upload",
          "type": "file-upload",
          "props": {
            "accept": [".pdf", ".txt", ".md"],
            "multiple": true
          }
        }
      ]
    },
    {
      "id": "graph",
      "type": "card",
      "props": {
        "title": "Knowledge Graph",
        "span": 2
      },
      "children": [
        {
          "id": "dag",
          "type": "dag-visualizer",
          "dataSource": "$.knowledge_graph",
          "props": {
            "layout": "force",
            "height": 400
          }
        }
      ]
    },
    {
      "id": "summary",
      "type": "card",
      "props": {
        "title": "Summary"
      },
      "children": [
        {
          "id": "summary-content",
          "type": "markdown-preview",
          "dataSource": "$.summary.content"
        }
      ]
    },
    {
      "id": "citations",
      "type": "card",
      "props": {
        "title": "Citations"
      },
      "children": [
        {
          "id": "citation-list",
          "type": "blob-viewer",
          "dataSource": "$.citations",
          "props": {
            "format": "apa"
          }
        }
      ]
    }
  ]
}
```

## React Implementation

### Dynamic Component Renderer
```typescript
// DynamicUI.tsx
import React from 'react';
import { useBlobs, useProvider } from '@/hooks';
import * as Components from '@/components/ui';
import JSONPath from 'jsonpath';

interface DynamicUIProps {
  layout: UILayout;
  data: Record<string, any>;
}

export const DynamicUI: React.FC<DynamicUIProps> = ({ layout, data }) => {
  return <LayoutRenderer layout={layout} data={data} />;
};

const LayoutRenderer: React.FC<{ layout: UILayout; data: any }> = ({ 
  layout, 
  data 
}) => {
  const getLayoutComponent = () => {
    switch (layout.type) {
      case 'split':
        return (
          <SplitLayout orientation={layout.orientation}>
            {layout.children.map(child => (
              <ComponentRenderer 
                key={child.id} 
                component={child} 
                data={data} 
              />
            ))}
          </SplitLayout>
        );
      case 'tabs':
        return (
          <TabLayout>
            {layout.children.map(child => (
              <Tab key={child.id} label={child.props?.title}>
                <ComponentRenderer component={child} data={data} />
              </Tab>
            ))}
          </TabLayout>
        );
      // ... other layout types
    }
  };

  return getLayoutComponent();
};

const ComponentRenderer: React.FC<{ 
  component: UIComponent; 
  data: any 
}> = ({ component, data }) => {
  // Resolve data binding
  const boundData = component.dataSource 
    ? JSONPath.query(data, component.dataSource)[0]
    : null;
    
  // Check visibility condition
  if (component.visibility) {
    const isVisible = evaluateExpression(component.visibility, data);
    if (!isVisible) return null;
  }
  
  // Get component from library
  const Component = Components[component.type];
  if (!Component) {
    console.warn(`Unknown component type: ${component.type}`);
    return null;
  }
  
  return (
    <Component
      id={component.id}
      data={boundData}
      {...component.props}
      actions={component.actions}
      style={component.style}
    >
      {component.children?.map(child => (
        <ComponentRenderer 
          key={child.id} 
          component={child} 
          data={data} 
        />
      ))}
    </Component>
  );
};
```

### Component Implementation Example
```typescript
// components/ui/BlobEditor.tsx
import React, { useState, useCallback } from 'react';
import MonacoEditor from '@monaco-editor/react';
import { useDebounce } from '@/hooks';

interface BlobEditorProps {
  data: string;
  language?: string;
  theme?: string;
  autoSave?: boolean;
  autoSaveDelay?: number;
  onChange?: (value: string) => void;
  actions?: UIAction[];
}

export const BlobEditor: React.FC<BlobEditorProps> = ({
  data,
  language = 'plaintext',
  theme = 'vs-light',
  autoSave = false,
  autoSaveDelay = 1000,
  onChange,
  actions,
  ...props
}) => {
  const [content, setContent] = useState(data);
  const debouncedContent = useDebounce(content, autoSaveDelay);
  
  React.useEffect(() => {
    if (autoSave && debouncedContent !== data) {
      onChange?.(debouncedContent);
    }
  }, [debouncedContent]);
  
  const handleAction = useCallback((action: UIAction) => {
    // Execute action through action system
    ActionSystem.execute(action, { content });
  }, [content]);
  
  return (
    <div className="blob-editor">
      <MonacoEditor
        value={content}
        language={language}
        theme={theme}
        onChange={setContent}
        {...props}
      />
      {actions && (
        <ActionBar actions={actions} onAction={handleAction} />
      )}
    </div>
  );
};
```

## Mobile (React Native) Implementation

```typescript
// DynamicUI.native.tsx
import React from 'react';
import { View, ScrollView } from 'react-native';
import * as Components from '@/components/ui/native';

const ComponentRenderer: React.FC<{ component: UIComponent }> = ({ 
  component 
}) => {
  // Map web components to React Native equivalents
  const getNativeComponent = () => {
    switch (component.type) {
      case 'blob-editor':
        return Components.TextInput;
      case 'blob-viewer':
        return Components.TextView;
      case 'dag-visualizer':
        return Components.GraphView;
      case 'ramble-button':
        return Components.VoiceButton;
      // ... other mappings
    }
  };
  
  const Component = getNativeComponent();
  return <Component {...component.props} />;
};
```

## AR (Vision Pro) Implementation

```swift
// DynamicUI.swift
import SwiftUI
import RealityKit

struct DynamicUIView: View {
    let layout: UILayout
    @StateObject var dataStore: BlobDataStore
    
    var body: some View {
        switch layout.type {
        case .split:
            HStack {
                ForEach(layout.children) { component in
                    ComponentView(component: component)
                        .frame(depth: 100)  // 3D depth
                }
            }
        case .canvas:
            ZStack {
                ForEach(layout.children) { component in
                    ComponentView(component: component)
                        .position3D(component.position)
                }
            }
        }
    }
}

struct BlobEditor3D: View {
    @Binding var content: String
    
    var body: some View {
        VStack {
            // Floating text editor in 3D space
            TextEditor(text: $content)
                .frame(width: 400, height: 300)
                .glassBackgroundEffect()
            
            // Voice input gesture
            SpatialTapGesture()
                .onEnded { _ in
                    startVoiceInput()
                }
        }
    }
}
```

## Responsive Design

```typescript
interface ResponsiveConfig {
  breakpoints?: {
    mobile?: number;   // Default: 768
    tablet?: number;   // Default: 1024
    desktop?: number;  // Default: 1440
  };
  mobileLayout?: UILayout;
  tabletLayout?: UILayout;
}

// Usage
{
  "type": "split",
  "orientation": "horizontal",
  "responsive": {
    "breakpoints": {
      "mobile": 768
    },
    "mobileLayout": {
      "type": "tabs",
      "children": [...] // Same children, different layout
    }
  }
}
```

## Action System

```typescript
class ActionSystem {
  static async execute(action: UIAction, context: ActionContext) {
    switch (action.type) {
      case 'transform':
        return this.executeTransform(action, context);
      case 'create':
        return this.executeCreate(action, context);
      case 'navigate':
        return this.executeNavigate(action, context);
      // ...
    }
  }
  
  private static async executeTransform(action: UIAction, context: ActionContext) {
    const { provider, params } = action;
    
    // Call provider service
    const result = await providerClient.execute({
      provider_id: provider,
      blob_id: context.blobId,
      params
    });
    
    // Update UI with result
    context.updateData({
      processed_blob: result.blob
    });
  }
}
```

## Benefits

1. **No Frontend Changes**: New providers work immediately
2. **Cross-Platform**: One definition, multiple renderings
3. **Customizable**: Users can modify layouts
4. **Consistent UX**: Standard component library
5. **Performance**: Optimized rendering and data binding
6. **Accessibility**: Built into component library

This dynamic UI system enables infinite flexibility while maintaining consistency across the platform.