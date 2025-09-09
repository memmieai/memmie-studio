# Memmie Studio MVP Implementation Tasks

This folder contains detailed, sequential task breakdowns for implementing the Memmie Studio MVP. Each task is designed to be completed independently by an agent or developer in one session.

## MVP Goal
Build a beta-testable platform for 100 users with:
- Text-based book writing assistant
- Business pitch creator
- Dark mode UI
- Simple authentication
- MongoDB blob storage

## Task Sequence

### Backend Services (Week 1)

#### Task 01: State Service Setup ✅
- **File**: `01-state-service-setup.md`
- **Purpose**: MongoDB blob storage for user content
- **Port**: 8006
- **Key Features**: Create, read, update blobs with metadata

#### Task 02: Provider Service MVP ✅
- **File**: `02-provider-service-mvp.md`
- **Purpose**: Manage book and pitch providers with AI processing
- **Port**: 8007
- **Key Features**: Provider templates, OpenAI integration

#### Task 03: Studio API Setup ✅
- **File**: `03-studio-api-setup.md`
- **Purpose**: Orchestrate services and serve frontend
- **Port**: 8010
- **Key Features**: Auth integration, service orchestration

### Frontend Implementation (Week 2)

#### Task 04: Frontend Setup ✅
- **File**: `04-frontend-setup.md`
- **Purpose**: React app with dark theme and routing
- **Port**: 3000 (dev)
- **Key Features**: Login, dashboard, navigation

#### Task 05: Book Writer Interface ✅
- **File**: `05-book-writer-interface.md`
- **Purpose**: Split-pane editor for book writing
- **Key Features**: Live AI expansion, auto-save

#### Task 06: Pitch Creator Interface ✅
- **File**: `06-pitch-creator-interface.md`
- **Purpose**: Structured pitch/business plan creator
- **Key Features**: Section-based editing, AI enhancement

### Testing & Deployment (Week 3)

#### Task 07: Integration Testing ✅
- **File**: `07-integration-testing.md`
- **Purpose**: End-to-end testing and deployment prep
- **Key Features**: Test scripts, load testing, deployment checklist

## Quick Start

1. **Start MongoDB and Auth Service**
   ```bash
   # Ensure MongoDB is running on localhost:27017
   # Ensure Auth service is running on localhost:8001
   ```

2. **Implement Backend Services (Tasks 01-03)**
   ```bash
   # Follow each task file sequentially
   # Each creates a working service
   ```

3. **Build Frontend (Tasks 04-06)**
   ```bash
   # Create React app
   # Add components for each provider
   ```

4. **Test Everything (Task 07)**
   ```bash
   cd /home/uneid/iter3/memmieai/memmie-studio
   ./scripts/start-mvp.sh
   ./tests/integration_test.sh
   ```

## Service Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Browser   │────▶│  Studio API  │────▶│Auth Service │
│  Port 3000  │     │  Port 8010   │     │ Port 8001   │
└─────────────┘     └──────────────┘     └─────────────┘
                            │
                    ┌───────┴────────┐
                    ▼                ▼
            ┌──────────────┐  ┌──────────────┐
            │State Service │  │Provider Svc  │
            │  Port 8006   │  │  Port 8007   │
            └──────────────┘  └──────────────┘
                    │                │
                    ▼                ▼
            ┌──────────────┐  ┌──────────────┐
            │   MongoDB    │  │  OpenAI API  │
            └──────────────┘  └──────────────┘
```

## Key Technologies

- **Backend**: Go, Gin framework
- **Frontend**: React, TypeScript
- **Database**: MongoDB (blob storage)
- **AI**: OpenAI GPT-4/GPT-3.5
- **Auth**: JWT tokens via Auth service

## Environment Variables Required

```bash
# OpenAI (for Provider Service)
OPENAI_API_KEY=sk-...

# Service URLs
AUTH_SERVICE_URL=http://localhost:8001
STATE_SERVICE_URL=http://localhost:8006
PROVIDER_SERVICE_URL=http://localhost:8007

# MongoDB
MONGO_URI=mongodb://memmie:memmiepass@localhost:27017/db?authSource=admin
```

## Success Metrics

- ✅ All services compile and run
- ✅ Auth flow works end-to-end
- ✅ Can create and save documents
- ✅ AI processing works (with API key)
- ✅ UI is responsive and dark themed
- ✅ System handles 100 concurrent users
- ✅ < 500ms average response time

## Common Issues & Solutions

1. **MongoDB Connection Failed**
   - Ensure MongoDB is running with auth enabled
   - Check credentials in connection string

2. **Auth Service Not Found**
   - Start memmie-auth service first
   - Verify it's running on port 8001

3. **OpenAI API Errors**
   - Set valid OPENAI_API_KEY environment variable
   - Check API quota and billing

4. **CORS Errors**
   - Studio API includes CORS middleware
   - Check allowed origins match frontend URL

## Next Steps After MVP

1. Add voice input ("Ramble" feature)
2. Implement WebSocket for real-time updates
3. Add more providers (research, music, etc.)
4. Build mobile apps
5. Add collaboration features
6. Implement full delta/versioning system

## Support

For issues or questions about implementation:
1. Check the individual task files for detailed steps
2. Review error logs in each service
3. Run integration tests to identify failures
4. Ensure all prerequisites are met

Each task file contains complete, copy-paste ready code that should work when implemented sequentially.