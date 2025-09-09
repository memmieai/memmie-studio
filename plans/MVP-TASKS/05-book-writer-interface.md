# Task 05: Book Writer Interface

## Objective
Create the Book Writer interface with a split-pane view showing original text on the left and AI-expanded version on the right.

## Prerequisites
- Frontend setup (Task 04) completed
- React app running
- Studio API and all backend services running

## Task Steps

### Step 1: Create Book Writer Component
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/BookWriter.tsx`

```typescript
import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, DocumentResponse } from '../api/client';
import './BookWriter.css';

export const BookWriter: React.FC = () => {
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [processedContent, setProcessedContent] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [autoProcess, setAutoProcess] = useState(true);
  const [lastSaved, setLastSaved] = useState<DocumentResponse | null>(null);
  const [debounceTimer, setDebounceTimer] = useState<NodeJS.Timeout | null>(null);
  const navigate = useNavigate();

  // Auto-process content when user stops typing
  useEffect(() => {
    if (!autoProcess || !content || content.length < 20) {
      return;
    }

    // Clear existing timer
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    // Set new timer
    const timer = setTimeout(() => {
      processContent();
    }, 1500); // Wait 1.5 seconds after user stops typing

    setDebounceTimer(timer);

    return () => {
      if (timer) {
        clearTimeout(timer);
      }
    };
  }, [content, autoProcess]);

  const processContent = async () => {
    if (!content || isProcessing) return;

    setIsProcessing(true);
    try {
      const response = await api.createDocument({
        provider_id: 'book',
        content: content,
        process_content: true,
        metadata: {
          title: title || 'Untitled Chapter',
          type: 'chapter',
        },
      });

      if (response.processed) {
        setProcessedContent(response.processed);
      }
      setLastSaved(response);
    } catch (error) {
      console.error('Failed to process content:', error);
    } finally {
      setIsProcessing(false);
    }
  };

  const handleSave = async () => {
    if (!content) return;

    setIsSaving(true);
    try {
      const response = await api.createDocument({
        provider_id: 'book',
        content: content,
        process_content: autoProcess,
        metadata: {
          title: title || 'Untitled Chapter',
          type: 'chapter',
          status: 'draft',
        },
      });

      if (response.processed) {
        setProcessedContent(response.processed);
      }
      setLastSaved(response);
      
      // Show success message
      alert('Chapter saved successfully!');
    } catch (error) {
      console.error('Failed to save:', error);
      alert('Failed to save chapter');
    } finally {
      setIsSaving(false);
    }
  };

  const handleBack = () => {
    if (content && !lastSaved) {
      if (!window.confirm('You have unsaved changes. Are you sure you want to leave?')) {
        return;
      }
    }
    navigate('/');
  };

  const wordCount = content.split(/\s+/).filter(word => word.length > 0).length;
  const expandedWordCount = processedContent.split(/\s+/).filter(word => word.length > 0).length;

  return (
    <div className="book-writer">
      <header className="writer-header">
        <button onClick={handleBack} className="back-btn">
          ← Back
        </button>
        <input
          type="text"
          placeholder="Chapter Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="title-input"
        />
        <div className="header-actions">
          <label className="auto-process-toggle">
            <input
              type="checkbox"
              checked={autoProcess}
              onChange={(e) => setAutoProcess(e.target.checked)}
            />
            <span>Auto-expand</span>
          </label>
          <button 
            onClick={handleSave} 
            disabled={isSaving || !content}
            className="save-btn"
          >
            {isSaving ? 'Saving...' : 'Save Chapter'}
          </button>
        </div>
      </header>

      <div className="writer-content">
        <div className="editor-pane">
          <div className="pane-header">
            <h3>Your Writing</h3>
            <span className="word-count">{wordCount} words</span>
          </div>
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="Start writing your chapter here. The AI will automatically expand your text with more descriptive details, character development, and engaging narrative..."
            className="editor-textarea"
          />
        </div>

        <div className="divider"></div>

        <div className="preview-pane">
          <div className="pane-header">
            <h3>AI Expanded Version</h3>
            {processedContent && (
              <span className="word-count">{expandedWordCount} words</span>
            )}
          </div>
          <div className="preview-content">
            {isProcessing ? (
              <div className="processing-indicator">
                <div className="spinner"></div>
                <p>Expanding your text...</p>
              </div>
            ) : processedContent ? (
              <div className="processed-text">
                {processedContent.split('\n').map((paragraph, index) => (
                  <p key={index}>{paragraph}</p>
                ))}
              </div>
            ) : (
              <div className="empty-preview">
                <p>Your expanded text will appear here</p>
                <p className="hint">
                  {autoProcess 
                    ? 'Start typing and the AI will automatically expand your text'
                    : 'Click "Process" to expand your text with AI'}
                </p>
              </div>
            )}
          </div>
          {!autoProcess && content && (
            <button 
              onClick={processContent} 
              disabled={isProcessing}
              className="process-btn"
            >
              {isProcessing ? 'Processing...' : 'Expand Text'}
            </button>
          )}
        </div>
      </div>

      <footer className="writer-footer">
        <div className="footer-info">
          {lastSaved && (
            <span className="saved-indicator">
              Last saved: {new Date(lastSaved.created_at).toLocaleTimeString()}
            </span>
          )}
        </div>
        <div className="expansion-stats">
          {processedContent && (
            <span className="expansion-ratio">
              Expansion: {Math.round((expandedWordCount / wordCount) * 100)}%
            </span>
          )}
        </div>
      </footer>
    </div>
  );
};
```

### Step 2: Create Book Writer Styles
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/BookWriter.css`

```css
.book-writer {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #0a0a0a;
}

.writer-header {
  background: #1a1a1a;
  padding: 15px 20px;
  display: flex;
  align-items: center;
  gap: 20px;
  border-bottom: 1px solid #333;
}

.back-btn {
  padding: 8px 12px;
  background: transparent;
  color: #888;
  border: 1px solid #333;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.back-btn:hover {
  color: #fff;
  border-color: #666;
}

.title-input {
  flex: 1;
  padding: 8px 12px;
  background: #0a0a0a;
  color: #fff;
  border: 1px solid #333;
  border-radius: 6px;
  font-size: 16px;
}

.title-input:focus {
  outline: none;
  border-color: #4a9eff;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 20px;
}

.auto-process-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #888;
  cursor: pointer;
}

.auto-process-toggle input[type="checkbox"] {
  cursor: pointer;
}

.auto-process-toggle:hover {
  color: #fff;
}

.save-btn {
  padding: 8px 20px;
  background: #4a9eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
  white-space: nowrap;
}

.save-btn:hover:not(:disabled) {
  background: #3a8eef;
}

.save-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.writer-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.editor-pane,
.preview-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.pane-header {
  padding: 15px 20px;
  background: #1a1a1a;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pane-header h3 {
  margin: 0;
  font-size: 14px;
  text-transform: uppercase;
  color: #666;
}

.word-count {
  font-size: 12px;
  color: #666;
}

.editor-textarea {
  flex: 1;
  padding: 20px;
  background: #0a0a0a;
  color: #fff;
  border: none;
  resize: none;
  font-family: 'Georgia', serif;
  font-size: 16px;
  line-height: 1.8;
}

.editor-textarea:focus {
  outline: none;
}

.editor-textarea::placeholder {
  color: #444;
}

.divider {
  width: 1px;
  background: #333;
}

.preview-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  background: #0a0a0a;
}

.processing-indicator {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #333;
  border-top-color: #4a9eff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.processed-text {
  font-family: 'Georgia', serif;
  font-size: 16px;
  line-height: 1.8;
  color: #e0e0e0;
}

.processed-text p {
  margin: 0 0 1em 0;
}

.processed-text p:last-child {
  margin-bottom: 0;
}

.empty-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  text-align: center;
}

.empty-preview p {
  margin: 10px 0;
}

.empty-preview .hint {
  font-size: 14px;
  color: #444;
}

.process-btn {
  margin: 20px;
  padding: 10px 20px;
  background: #4a9eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.process-btn:hover:not(:disabled) {
  background: #3a8eef;
}

.process-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.writer-footer {
  background: #1a1a1a;
  padding: 10px 20px;
  border-top: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-info {
  color: #666;
  font-size: 12px;
}

.saved-indicator {
  color: #4a9eff;
}

.expansion-stats {
  color: #666;
  font-size: 12px;
}

.expansion-ratio {
  color: #4a9eff;
}

/* Responsive adjustments */
@media (max-width: 768px) {
  .writer-content {
    flex-direction: column;
  }
  
  .divider {
    width: 100%;
    height: 1px;
  }
  
  .writer-header {
    flex-wrap: wrap;
  }
  
  .title-input {
    width: 100%;
    order: 3;
  }
}
```

### Step 3: Update Router in App.tsx
Update file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/App.tsx`

```typescript
import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { Login } from './components/Login';
import { Dashboard } from './components/Dashboard';
import { BookWriter } from './components/BookWriter';
import './App.css';

const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};

function App() {
  return (
    <Router>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <PrivateRoute>
                <Dashboard />
              </PrivateRoute>
            }
          />
          <Route
            path="/create/book"
            element={
              <PrivateRoute>
                <BookWriter />
              </PrivateRoute>
            }
          />
        </Routes>
      </AuthProvider>
    </Router>
  );
}

export default App;
```

### Step 4: Create Sample Content for Testing
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/samples/book-samples.ts`

```typescript
export const bookSamples = [
  {
    title: "The Mysterious Island",
    content: "The storm hit without warning. Captain Morrison gripped the wheel as waves crashed over the bow.",
  },
  {
    title: "The Last Colony",
    content: "Earth was dying. The colony ships were humanity's last hope, but only one would make it to Alpha Centauri.",
  },
  {
    title: "The Detective's Case",
    content: "The body was found at midnight. Detective Sarah Chen examined the scene, noting the unusual pattern of blood drops.",
  },
];

export const writingPrompts = [
  "A character discovers a hidden door in their basement",
  "Two strangers meet during a blackout",
  "A time traveler arrives with a warning",
  "The last bookstore on Earth",
  "A letter arrives 50 years late",
];
```

### Step 5: Add Quick Start Guide
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/QuickStart.tsx`

```typescript
import React from 'react';
import { bookSamples, writingPrompts } from '../samples/book-samples';
import './QuickStart.css';

interface QuickStartProps {
  onSelectSample: (title: string, content: string) => void;
}

export const QuickStart: React.FC<QuickStartProps> = ({ onSelectSample }) => {
  return (
    <div className="quick-start">
      <h3>Quick Start</h3>
      
      <div className="section">
        <h4>Sample Chapters</h4>
        <div className="samples">
          {bookSamples.map((sample, index) => (
            <button
              key={index}
              className="sample-btn"
              onClick={() => onSelectSample(sample.title, sample.content)}
            >
              {sample.title}
            </button>
          ))}
        </div>
      </div>

      <div className="section">
        <h4>Writing Prompts</h4>
        <div className="prompts">
          {writingPrompts.map((prompt, index) => (
            <button
              key={index}
              className="prompt-btn"
              onClick={() => onSelectSample('', prompt)}
            >
              {prompt}
            </button>
          ))}
        </div>
      </div>

      <div className="tips">
        <h4>Tips</h4>
        <ul>
          <li>Write 2-3 sentences and watch the AI expand them</li>
          <li>The AI maintains your writing style</li>
          <li>Auto-expand processes after you stop typing</li>
          <li>Save regularly to preserve your work</li>
        </ul>
      </div>
    </div>
  );
};
```

### Step 6: Create QuickStart Styles
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/QuickStart.css`

```css
.quick-start {
  position: fixed;
  right: 20px;
  top: 80px;
  width: 300px;
  background: #1a1a1a;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 20px;
  max-height: calc(100vh - 100px);
  overflow-y: auto;
  z-index: 100;
}

.quick-start h3 {
  margin: 0 0 20px 0;
  color: #fff;
  font-size: 18px;
}

.quick-start .section {
  margin-bottom: 25px;
}

.quick-start h4 {
  margin: 0 0 10px 0;
  color: #666;
  font-size: 12px;
  text-transform: uppercase;
}

.samples,
.prompts {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sample-btn,
.prompt-btn {
  padding: 8px 12px;
  background: #0a0a0a;
  color: #888;
  border: 1px solid #333;
  border-radius: 4px;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
}

.sample-btn:hover,
.prompt-btn:hover {
  background: #2a2a2a;
  color: #fff;
  border-color: #4a9eff;
}

.tips {
  padding-top: 20px;
  border-top: 1px solid #333;
}

.tips ul {
  margin: 0;
  padding-left: 20px;
}

.tips li {
  color: #666;
  font-size: 13px;
  margin-bottom: 8px;
  line-height: 1.5;
}
```

### Step 7: Test the Book Writer

```bash
# Make sure all backend services are running
# State Service (8006), Provider Service (8007), Auth Service (8001), Studio API (8010)

# Terminal 1: Start React development server
cd /home/uneid/iter3/memmieai/memmie-studio/web
npm start

# Browser: Navigate to http://localhost:3000
# 1. Login with test credentials
# 2. Click on "Book Writer" provider
# 3. Click "New Document"
# 4. Start typing in the left pane
# 5. Watch AI expansion appear in right pane
```

## Expected Output
- Split-pane interface with writing area on left
- AI-expanded text appears on right after 1.5 seconds of no typing
- Word count for both original and expanded text
- Auto-expand toggle for manual/automatic processing
- Save button to persist the chapter
- Responsive design for mobile devices

## Success Criteria
✅ Book Writer component loads without errors
✅ Can type text in the left editor pane
✅ AI expansion appears in right pane (requires OpenAI key)
✅ Word counts update correctly
✅ Auto-expand feature works with debouncing
✅ Save functionality persists to backend
✅ Navigation back to dashboard works
✅ Unsaved changes warning appears when needed

## Notes
- Requires valid OpenAI API key in Provider Service
- Auto-expand waits 1.5 seconds after typing stops
- Minimum 20 characters required for auto-processing
- Uses Georgia serif font for better readability
- Expansion ratio shows how much AI expanded the text