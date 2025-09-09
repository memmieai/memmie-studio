# Task 06: Pitch Creator Interface

## Objective
Create the Pitch Creator interface for business plans and pitch decks with structured sections and AI enhancement.

## Prerequisites
- Book Writer interface (Task 05) completed
- React app running
- All backend services operational

## Task Steps

### Step 1: Create Pitch Creator Component
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/PitchCreator.tsx`

```typescript
import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, DocumentResponse } from '../api/client';
import './PitchCreator.css';

interface PitchSection {
  id: string;
  title: string;
  placeholder: string;
  content: string;
}

export const PitchCreator: React.FC = () => {
  const navigate = useNavigate();
  const [pitchTitle, setPitchTitle] = useState('');
  const [companyName, setCompanyName] = useState('');
  const [currentSection, setCurrentSection] = useState(0);
  const [processedPitch, setProcessedPitch] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [lastSaved, setLastSaved] = useState<DocumentResponse | null>(null);

  const [sections, setSections] = useState<PitchSection[]>([
    {
      id: 'problem',
      title: 'Problem',
      placeholder: 'What problem are you solving? Who experiences this problem? How big is the problem?',
      content: '',
    },
    {
      id: 'solution',
      title: 'Solution',
      placeholder: 'How does your product/service solve the problem? What makes it unique?',
      content: '',
    },
    {
      id: 'market',
      title: 'Market',
      placeholder: 'Who is your target market? What is the market size? What is your go-to-market strategy?',
      content: '',
    },
    {
      id: 'business-model',
      title: 'Business Model',
      placeholder: 'How will you make money? What are your revenue streams? What is your pricing strategy?',
      content: '',
    },
    {
      id: 'competition',
      title: 'Competition',
      placeholder: 'Who are your competitors? What is your competitive advantage?',
      content: '',
    },
    {
      id: 'team',
      title: 'Team',
      placeholder: 'Who is on your team? What are their backgrounds and expertise?',
      content: '',
    },
    {
      id: 'traction',
      title: 'Traction',
      placeholder: 'What progress have you made? Any customers, revenue, partnerships?',
      content: '',
    },
    {
      id: 'funding',
      title: 'Funding Ask',
      placeholder: 'How much are you raising? What will you use the funds for? What milestones will you achieve?',
      content: '',
    },
  ]);

  const updateSectionContent = (index: number, content: string) => {
    const newSections = [...sections];
    newSections[index].content = content;
    setSections(newSections);
  };

  const getAllContent = () => {
    return sections
      .filter(s => s.content)
      .map(s => `## ${s.title}\n${s.content}`)
      .join('\n\n');
  };

  const isComplete = () => {
    return sections.filter(s => s.content.trim()).length >= 4; // At least 4 sections filled
  };

  const processPitch = async () => {
    if (!isComplete() || isProcessing) return;

    setIsProcessing(true);
    try {
      const fullContent = getAllContent();
      const response = await api.createDocument({
        provider_id: 'pitch',
        content: fullContent,
        process_content: true,
        metadata: {
          title: pitchTitle || 'Untitled Pitch',
          company: companyName,
          type: 'pitch',
        },
      });

      if (response.processed) {
        setProcessedPitch(response.processed);
      }
      setLastSaved(response);
    } catch (error) {
      console.error('Failed to process pitch:', error);
    } finally {
      setIsProcessing(false);
    }
  };

  const handleSave = async () => {
    if (!isComplete()) {
      alert('Please fill out at least 4 sections before saving');
      return;
    }

    setIsSaving(true);
    try {
      const fullContent = getAllContent();
      const response = await api.createDocument({
        provider_id: 'pitch',
        content: fullContent,
        process_content: true,
        metadata: {
          title: pitchTitle || 'Untitled Pitch',
          company: companyName,
          type: 'pitch',
          status: 'draft',
          sections: sections.map(s => ({
            id: s.id,
            title: s.title,
            hasContent: !!s.content,
          })),
        },
      });

      if (response.processed) {
        setProcessedPitch(response.processed);
      }
      setLastSaved(response);
      alert('Pitch saved successfully!');
    } catch (error) {
      console.error('Failed to save:', error);
      alert('Failed to save pitch');
    } finally {
      setIsSaving(false);
    }
  };

  const handleBack = () => {
    const hasContent = sections.some(s => s.content);
    if (hasContent && !lastSaved) {
      if (!window.confirm('You have unsaved changes. Are you sure you want to leave?')) {
        return;
      }
    }
    navigate('/');
  };

  const completedSections = sections.filter(s => s.content.trim()).length;
  const progress = (completedSections / sections.length) * 100;

  return (
    <div className="pitch-creator">
      <header className="pitch-header">
        <button onClick={handleBack} className="back-btn">
          ← Back
        </button>
        <div className="pitch-meta">
          <input
            type="text"
            placeholder="Pitch Title"
            value={pitchTitle}
            onChange={(e) => setPitchTitle(e.target.value)}
            className="pitch-title-input"
          />
          <input
            type="text"
            placeholder="Company Name"
            value={companyName}
            onChange={(e) => setCompanyName(e.target.value)}
            className="company-input"
          />
        </div>
        <div className="header-actions">
          <div className="progress-indicator">
            <span>{completedSections}/{sections.length} sections</span>
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${progress}%` }}></div>
            </div>
          </div>
          <button
            onClick={handleSave}
            disabled={isSaving || !isComplete()}
            className="save-btn"
          >
            {isSaving ? 'Saving...' : 'Save Pitch'}
          </button>
        </div>
      </header>

      <div className="pitch-content">
        <aside className="sections-nav">
          <h3>Sections</h3>
          <div className="section-list">
            {sections.map((section, index) => (
              <button
                key={section.id}
                className={`section-nav-item ${currentSection === index ? 'active' : ''} ${section.content ? 'completed' : ''}`}
                onClick={() => setCurrentSection(index)}
              >
                <span className="section-number">{index + 1}</span>
                <span className="section-title">{section.title}</span>
                {section.content && <span className="check-mark">✓</span>}
              </button>
            ))}
          </div>
        </aside>

        <div className="editor-area">
          <div className="section-editor">
            <h2>{sections[currentSection].title}</h2>
            <textarea
              value={sections[currentSection].content}
              onChange={(e) => updateSectionContent(currentSection, e.target.value)}
              placeholder={sections[currentSection].placeholder}
              className="section-textarea"
            />
            <div className="section-nav-buttons">
              <button
                onClick={() => setCurrentSection(Math.max(0, currentSection - 1))}
                disabled={currentSection === 0}
                className="nav-btn"
              >
                ← Previous
              </button>
              <button
                onClick={() => setCurrentSection(Math.min(sections.length - 1, currentSection + 1))}
                disabled={currentSection === sections.length - 1}
                className="nav-btn"
              >
                Next →
              </button>
            </div>
          </div>
        </div>

        <div className="preview-area">
          <div className="preview-header">
            <h3>AI-Enhanced Pitch</h3>
            {isComplete() && !processedPitch && (
              <button
                onClick={processPitch}
                disabled={isProcessing}
                className="process-btn"
              >
                {isProcessing ? 'Processing...' : 'Generate Full Pitch'}
              </button>
            )}
          </div>
          <div className="preview-content">
            {isProcessing ? (
              <div className="processing-indicator">
                <div className="spinner"></div>
                <p>Creating your professional pitch...</p>
              </div>
            ) : processedPitch ? (
              <div className="processed-pitch">
                {processedPitch.split('\n').map((line, index) => {
                  if (line.startsWith('##')) {
                    return <h3 key={index}>{line.replace('##', '').trim()}</h3>;
                  } else if (line.startsWith('#')) {
                    return <h2 key={index}>{line.replace('#', '').trim()}</h2>;
                  } else if (line.trim()) {
                    return <p key={index}>{line}</p>;
                  }
                  return null;
                })}
              </div>
            ) : (
              <div className="empty-preview">
                <p>Your professional pitch will appear here</p>
                <p className="hint">
                  Fill out at least 4 sections and click "Generate Full Pitch" to see the AI-enhanced version
                </p>
              </div>
            )}
          </div>
        </div>
      </div>

      <footer className="pitch-footer">
        <div className="footer-info">
          {lastSaved && (
            <span className="saved-indicator">
              Last saved: {new Date(lastSaved.created_at).toLocaleTimeString()}
            </span>
          )}
        </div>
      </footer>
    </div>
  );
};
```

### Step 2: Create Pitch Creator Styles
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/PitchCreator.css`

```css
.pitch-creator {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #0a0a0a;
}

.pitch-header {
  background: #1a1a1a;
  padding: 15px 20px;
  display: flex;
  align-items: center;
  gap: 20px;
  border-bottom: 1px solid #333;
}

.pitch-meta {
  flex: 1;
  display: flex;
  gap: 15px;
}

.pitch-title-input,
.company-input {
  flex: 1;
  padding: 8px 12px;
  background: #0a0a0a;
  color: #fff;
  border: 1px solid #333;
  border-radius: 6px;
  font-size: 16px;
}

.pitch-title-input:focus,
.company-input:focus {
  outline: none;
  border-color: #4a9eff;
}

.progress-indicator {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #888;
  font-size: 14px;
}

.progress-bar {
  width: 100px;
  height: 6px;
  background: #333;
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: #4a9eff;
  transition: width 0.3s ease;
}

.pitch-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.sections-nav {
  width: 250px;
  background: #1a1a1a;
  padding: 20px;
  border-right: 1px solid #333;
  overflow-y: auto;
}

.sections-nav h3 {
  margin: 0 0 20px 0;
  font-size: 14px;
  text-transform: uppercase;
  color: #666;
}

.section-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.section-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: transparent;
  color: #888;
  border: 1px solid transparent;
  border-radius: 6px;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s;
}

.section-nav-item:hover {
  background: #2a2a2a;
  color: #fff;
}

.section-nav-item.active {
  background: #2a2a2a;
  border-color: #4a9eff;
  color: #4a9eff;
}

.section-nav-item.completed {
  color: #4a9eff;
}

.section-number {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #333;
  border-radius: 50%;
  font-size: 12px;
}

.section-nav-item.active .section-number {
  background: #4a9eff;
  color: #000;
}

.section-title {
  flex: 1;
}

.check-mark {
  color: #4a9eff;
}

.editor-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 30px;
  overflow-y: auto;
}

.section-editor {
  max-width: 800px;
  width: 100%;
  margin: 0 auto;
}

.section-editor h2 {
  margin: 0 0 20px 0;
  color: #fff;
}

.section-textarea {
  width: 100%;
  min-height: 300px;
  padding: 20px;
  background: #1a1a1a;
  color: #fff;
  border: 1px solid #333;
  border-radius: 8px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  font-size: 16px;
  line-height: 1.6;
  resize: vertical;
}

.section-textarea:focus {
  outline: none;
  border-color: #4a9eff;
}

.section-textarea::placeholder {
  color: #444;
}

.section-nav-buttons {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
}

.nav-btn {
  padding: 10px 20px;
  background: #1a1a1a;
  color: #888;
  border: 1px solid #333;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.nav-btn:hover:not(:disabled) {
  background: #2a2a2a;
  color: #fff;
  border-color: #666;
}

.nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.preview-area {
  width: 400px;
  background: #1a1a1a;
  border-left: 1px solid #333;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.preview-header {
  padding: 20px;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.preview-header h3 {
  margin: 0;
  font-size: 14px;
  text-transform: uppercase;
  color: #666;
}

.process-btn {
  padding: 8px 16px;
  background: #4a9eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
  font-size: 14px;
}

.process-btn:hover:not(:disabled) {
  background: #3a8eef;
}

.process-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.preview-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
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

.processed-pitch {
  color: #e0e0e0;
  line-height: 1.6;
}

.processed-pitch h2 {
  color: #fff;
  margin: 20px 0 10px 0;
  font-size: 20px;
}

.processed-pitch h3 {
  color: #4a9eff;
  margin: 15px 0 10px 0;
  font-size: 16px;
}

.processed-pitch p {
  margin: 0 0 12px 0;
}

.empty-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  text-align: center;
  padding: 20px;
}

.empty-preview p {
  margin: 10px 0;
}

.empty-preview .hint {
  font-size: 14px;
  color: #444;
}

.pitch-footer {
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

/* Responsive adjustments */
@media (max-width: 1200px) {
  .preview-area {
    display: none;
  }
  
  .editor-area {
    border-right: none;
  }
}

@media (max-width: 768px) {
  .pitch-content {
    flex-direction: column;
  }
  
  .sections-nav {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #333;
    padding: 15px;
  }
  
  .section-list {
    flex-direction: row;
    overflow-x: auto;
    gap: 10px;
  }
  
  .pitch-header {
    flex-wrap: wrap;
  }
  
  .pitch-meta {
    width: 100%;
    order: 3;
  }
}
```

### Step 3: Update Router for Pitch Creator
Update file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/App.tsx`

```typescript
import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { Login } from './components/Login';
import { Dashboard } from './components/Dashboard';
import { BookWriter } from './components/BookWriter';
import { PitchCreator } from './components/PitchCreator';
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
          <Route
            path="/create/pitch"
            element={
              <PrivateRoute>
                <PitchCreator />
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

### Step 4: Create Pitch Templates
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/samples/pitch-templates.ts`

```typescript
export const pitchTemplates = {
  saas: {
    title: 'SaaS B2B Platform',
    company: 'TechCo',
    sections: {
      problem: 'Small businesses struggle with managing customer relationships across multiple channels. Current CRM solutions are too complex and expensive.',
      solution: 'AI-powered CRM that automatically captures and organizes customer interactions from email, chat, and social media.',
      market: 'Target: SMBs with 10-100 employees. TAM: $50B. SAM: $5B. SOM: $500M in 5 years.',
      'business-model': 'SaaS subscription model. $50-500/month based on team size. Annual contracts with 20% discount.',
    },
  },
  marketplace: {
    title: 'Marketplace Platform',
    company: 'MarketPlace Inc',
    sections: {
      problem: 'Freelance professionals struggle to find quality clients. Clients struggle to find vetted talent.',
      solution: 'AI-matched marketplace that connects pre-vetted professionals with enterprise clients.',
      market: 'Target: Fortune 500 companies and top 1% freelancers. TAM: $400B freelance economy.',
      'business-model': '20% commission on transactions. Premium subscriptions for enhanced features.',
    },
  },
  consumer: {
    title: 'Consumer App',
    company: 'ConsumerTech',
    sections: {
      problem: 'People waste hours planning meals and grocery shopping. Food waste costs families $1500/year.',
      solution: 'AI meal planner that creates personalized menus and automated grocery lists based on preferences and budget.',
      market: 'Target: Busy families in urban areas. 50M households in US. Global expansion potential.',
      'business-model': 'Freemium model. Premium features $9.99/month. Affiliate revenue from grocery partners.',
    },
  },
};
```

### Step 5: Test the Pitch Creator

```bash
# Make sure all backend services are running

# Terminal 1: Start React development server
cd /home/uneid/iter3/memmieai/memmie-studio/web
npm start

# Browser: Navigate to http://localhost:3000
# 1. Login with test credentials
# 2. Click on "Pitch Creator" provider
# 3. Click "New Document"
# 4. Fill out sections one by one
# 5. Click "Generate Full Pitch" after filling 4+ sections
# 6. View AI-enhanced pitch on the right
```

## Expected Output
- Section-based navigation on the left
- Central editor for current section
- Progress indicator showing completion
- AI-enhanced pitch preview on the right
- Professional formatting for pitch output
- Guided workflow through all sections

## Success Criteria
✅ Pitch Creator component loads without errors
✅ Can navigate between sections
✅ Section completion tracked with checkmarks
✅ Progress bar updates correctly
✅ Requires minimum 4 sections before processing
✅ AI enhancement creates professional pitch (requires OpenAI key)
✅ Save functionality works
✅ Navigation and unsaved changes warning work

## Notes
- Structured approach guides users through pitch creation
- Each section has helpful placeholder text
- AI transforms bullet points into professional narrative
- Progress tracking motivates completion
- Mobile responsive with collapsible navigation