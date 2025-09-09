# MVP Roadmap - Memmie Studio Beta

## Goal
Launch a minimal, functional MVP supporting 100 beta users with text-based book writing and business plan creation features. Dead simple dark mode UI. No voice features. Focus on core functionality.

## Timeline: 3 Weeks to Beta

### Week 1: Core Backend (Days 1-7)

#### Day 1-2: Simplified State Service
```go
// Minimal blob storage - just MongoDB
type Blob struct {
    ID         string    `bson:"_id"`
    UserID     string    `bson:"user_id"`
    ProviderID string    `bson:"provider_id"` // "book" or "pitch"
    Content    string    `bson:"content"`
    ParentID   *string   `bson:"parent_id"`
    CreatedAt  time.Time `bson:"created_at"`
}
```
- [ ] Setup MongoDB connection
- [ ] Create/Read blob endpoints
- [ ] Parent-child relationships only (no complex DAG)
- [ ] No deltas/versioning for MVP

#### Day 3-4: Simplified Provider Service
```go
// Just two hardcoded providers for MVP
const (
    BookWriterProvider = "book"
    PitchCreatorProvider = "pitch"
)
```
- [ ] Hardcode two provider templates
- [ ] No provider instances - just templates
- [ ] Fixed UI layouts for each
- [ ] Direct GPT-4 integration

#### Day 5-6: Studio API Service
```go
// Minimal orchestration
func (s *StudioService) ExpandText(ctx context.Context, req ExpandRequest) (*ExpandResponse, error) {
    // 1. Get blob from State
    // 2. Call GPT-4 directly
    // 3. Save expanded version
    // 4. Return both versions
}
```
- [ ] Authentication with existing Auth service
- [ ] Simple expand endpoint
- [ ] Get blobs endpoint
- [ ] No WebSocket - just polling

#### Day 7: Integration Testing
- [ ] End-to-end flow test
- [ ] Load test with 100 concurrent users
- [ ] Fix critical bugs only

### Week 2: Minimal Frontend (Days 8-14)

#### Day 8-9: React Setup
```typescript
// Ultra-minimal component set
- App.tsx         // Main layout
- Editor.tsx      // Text input
- Viewer.tsx      // Text display  
- LoginForm.tsx   // Auth
```
- [ ] Create React App with TypeScript
- [ ] Tailwind CSS dark theme only
- [ ] No component library - pure HTML/CSS

#### Day 10-11: Book Writer Interface
```typescript
// Dead simple split view
<div className="flex h-screen bg-black text-green-400 font-mono">
  <div className="w-1/2 border-r border-gray-800">
    <textarea 
      className="w-full h-full p-4 bg-black"
      placeholder="Write your chapter..."
    />
  </div>
  <div className="w-1/2 p-4 overflow-auto">
    {expandedText || "Expanded text will appear here..."}
  </div>
</div>
```
- [ ] Split pane layout
- [ ] Auto-save to backend
- [ ] Expand button
- [ ] Chapter list sidebar

#### Day 12-13: Business Pitch Interface
```typescript
// Structured sections
const pitchSections = [
  'Problem',
  'Solution', 
  'Market',
  'Business Model',
  'Team',
  'Ask'
];
```
- [ ] Section-based editor
- [ ] Expand each section individually
- [ ] Export to markdown
- [ ] Simple preview

#### Day 14: Polish & Deploy
- [ ] Error handling
- [ ] Loading states
- [ ] Deploy to Vercel/Netlify
- [ ] Basic mobile responsiveness

### Week 3: Testing & Launch (Days 15-21)

#### Day 15-16: Infrastructure
```yaml
# docker-compose for easy deployment
services:
  studio:
    image: memmie-studio:mvp
    ports:
      - "8010:8010"
  state:
    image: memmie-state:mvp
    ports:
      - "8006:8006"
  mongodb:
    image: mongo:6
```
- [ ] Dockerize services
- [ ] Deploy to cloud (DigitalOcean/AWS)
- [ ] Setup monitoring (Sentry)
- [ ] Configure rate limiting

#### Day 17-18: User Onboarding
- [ ] Landing page with waitlist
- [ ] Beta access codes
- [ ] Simple onboarding flow
- [ ] Example content/templates

#### Day 19-20: Beta Testing
- [ ] Invite first 10 users
- [ ] Gather feedback
- [ ] Fix critical bugs
- [ ] Performance optimization

#### Day 21: Beta Launch
- [ ] Open to 100 users
- [ ] Discord/Slack community
- [ ] Feedback form
- [ ] Analytics setup

## MVP Features

### ✅ Included
1. **Authentication**: Login/logout
2. **Book Writer**: Chapter writing with AI expansion
3. **Pitch Creator**: Section-based business plan builder
4. **Auto-save**: Every 5 seconds
5. **Export**: Download as text/markdown
6. **Dark Mode**: Single theme, no toggle

### ❌ Not Included (Post-MVP)
1. Voice input/Ramble
2. Multiple providers
3. DAG visualization
4. Real-time collaboration
5. Mobile app
6. Provider marketplace
7. Versioning/history
8. Custom UI layouts
9. File uploads
10. WebSocket updates

## Technical Stack (Simplified)

### Backend
```
- Go services (reuse existing)
- MongoDB (single database)
- Redis (caching only)
- GPT-4 API direct calls
- JWT auth (existing)
```

### Frontend
```
- React 18 (no Next.js)
- TypeScript
- Tailwind CSS
- Axios for API calls
- Zustand for state
- No UI library
```

## API Endpoints (MVP Only)

```
POST /api/v1/auth/login
POST /api/v1/auth/logout

GET  /api/v1/blobs?provider=book
POST /api/v1/blobs
PUT  /api/v1/blobs/:id

POST /api/v1/expand
{
  "content": "text to expand",
  "provider": "book|pitch",
  "context": "optional context"
}
```

## Database Schema (Simplified)

```javascript
// MongoDB collections
blobs: {
  _id: ObjectId,
  user_id: string,
  provider_id: "book" | "pitch",
  content: string,
  parent_id: ObjectId | null,
  metadata: {
    title?: string,
    section?: string,  // for pitch
    chapter?: number,  // for book
  },
  created_at: Date,
  updated_at: Date
}

users: {
  _id: ObjectId,
  email: string,
  beta_code: string,
  created_at: Date
}
```

## UI Mockup (ASCII)

### Book Writer
```
┌─────────────────────────────────────────────────────────┐
│ MEMMIE STUDIO v0.1 | Book Writer          [Export] [←→] │
├─────────────┬───────────────────────────────────────────┤
│ CHAPTERS    │ DRAFT                  │ EXPANDED         │
│             │                        │                  │
│ > Chapter 1 │ The hero begins their │ The hero, a     │
│   Chapter 2 │ journey in a small    │ young farmer    │
│   Chapter 3 │ village.              │ named Marcus,   │
│ + New       │                       │ begins their    │
│             │ [Write more...]       │ epic journey in │
│             │                       │ the small       │
│             │                       │ village of...   │
│             │                       │                 │
│             │              [EXPAND] │                 │
└─────────────┴────────────────────────┴─────────────────┘
```

### Pitch Creator
```
┌─────────────────────────────────────────────────────────┐
│ MEMMIE STUDIO v0.1 | Pitch Creator       [Export] [←→]  │
├─────────────────────────────────────────────────────────┤
│ SECTIONS           │ CONTENT                            │
│                    │                                    │
│ ✓ Problem          │ ┌─ Problem ──────────────────────┐ │
│ > Solution         │ │ Businesses struggle to create   │ │
│   Market           │ │ compelling pitch decks quickly. │ │
│   Business Model   │ │                                 │ │
│   Team             │ │ [Expand This Section]           │ │
│   Ask              │ └─────────────────────────────────┘ │
│                    │                                    │
│                    │ ┌─ Solution (Expanded) ──────────┐ │
│                    │ │ Our AI-powered platform helps   │ │
│                    │ │ entrepreneurs create investor-  │ │
│                    │ │ ready pitch decks in minutes... │ │
│                    │ └─────────────────────────────────┘ │
└────────────────────┴────────────────────────────────────┘
```

## Success Metrics

### Week 1 Goals
- [ ] All 3 services running
- [ ] Can create/retrieve blobs
- [ ] GPT-4 expansion working
- [ ] 100 user load test passing

### Week 2 Goals
- [ ] Frontend deployed
- [ ] Both providers functional
- [ ] <2s response time
- [ ] Auto-save working

### Week 3 Goals  
- [ ] 100 beta users invited
- [ ] <1% error rate
- [ ] 90% user retention (day 1)
- [ ] 50+ pieces of content created

## Launch Checklist

### Technical
- [ ] Services deployed
- [ ] Database backed up
- [ ] Monitoring active
- [ ] Error logging configured
- [ ] Rate limiting enabled
- [ ] SSL certificates

### Product
- [ ] Landing page live
- [ ] Beta codes distributed
- [ ] Example content ready
- [ ] Help documentation
- [ ] Feedback form

### Community
- [ ] Discord server
- [ ] Welcome email
- [ ] Daily check-ins planned
- [ ] Feedback loops established

## Risk Mitigation

1. **GPT-4 Rate Limits**: Cache expansions, queue requests
2. **Database Overload**: Simple indexes, connection pooling
3. **UI Bugs**: Minimal features, extensive testing
4. **User Confusion**: Clear onboarding, examples
5. **Scale Issues**: Start with 10 users, gradually increase

## Post-MVP Priorities

1. WebSocket for real-time updates
2. Voice input (Ramble)
3. More providers (research, code)
4. Mobile app
5. Collaboration features
6. Version history
7. Advanced UI customization
8. Provider marketplace

---

**Target Launch Date**: 3 weeks from now
**Beta Users**: 100 max
**Success Criteria**: 50% daily active users after 1 week

This MVP focuses on proving the core value proposition: AI-enhanced content creation with a simple, fast interface.