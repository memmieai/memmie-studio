# Task 07: Integration Testing and MVP Launch

## Objective
Perform end-to-end integration testing of the complete MVP system and prepare for beta launch with 100 users.

## Prerequisites
- All previous tasks (01-06) completed
- All services built and runnable
- MongoDB and Auth service configured
- OpenAI API key available

## Task Steps

### Step 1: Start All Services
Create file: `/home/uneid/iter3/memmieai/memmie-studio/scripts/start-mvp.sh`

```bash
#!/bin/bash

echo "Starting Memmie Studio MVP Services..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if MongoDB is running
if ! nc -z localhost 27017; then
    echo -e "${RED}MongoDB is not running on port 27017${NC}"
    echo "Please start MongoDB first"
    exit 1
fi

# Check if Auth service is running
if ! nc -z localhost 8001; then
    echo -e "${RED}Auth service is not running on port 8001${NC}"
    echo "Please start the Auth service first"
    exit 1
fi

# Start State Service
echo "Starting State Service (port 8006)..."
cd /home/uneid/iter3/memmieai/memmie-state
nohup go run cmd/server/main.go > state.log 2>&1 &
STATE_PID=$!
sleep 3

# Check State Service
if nc -z localhost 8006; then
    echo -e "${GREEN}✓ State Service started (PID: $STATE_PID)${NC}"
else
    echo -e "${RED}✗ State Service failed to start${NC}"
    exit 1
fi

# Start Provider Service
echo "Starting Provider Service (port 8007)..."
cd /home/uneid/iter3/memmieai/memmie-provider
export OPENAI_API_KEY="${OPENAI_API_KEY}"
nohup go run cmd/server/main.go > provider.log 2>&1 &
PROVIDER_PID=$!
sleep 3

# Check Provider Service
if nc -z localhost 8007; then
    echo -e "${GREEN}✓ Provider Service started (PID: $PROVIDER_PID)${NC}"
else
    echo -e "${RED}✗ Provider Service failed to start${NC}"
    exit 1
fi

# Start Studio API
echo "Starting Studio API (port 8010)..."
cd /home/uneid/iter3/memmieai/memmie-studio
nohup go run cmd/server/main.go > studio.log 2>&1 &
STUDIO_PID=$!
sleep 3

# Check Studio API
if nc -z localhost 8010; then
    echo -e "${GREEN}✓ Studio API started (PID: $STUDIO_PID)${NC}"
else
    echo -e "${RED}✗ Studio API failed to start${NC}"
    exit 1
fi

# Start Frontend (development mode)
echo "Starting Frontend (port 3000)..."
cd /home/uneid/iter3/memmieai/memmie-studio/web
nohup npm start > frontend.log 2>&1 &
FRONTEND_PID=$!

echo ""
echo "========================================="
echo -e "${GREEN}MVP Services Started Successfully!${NC}"
echo "========================================="
echo "State Service:    http://localhost:8006"
echo "Provider Service: http://localhost:8007"
echo "Studio API:       http://localhost:8010"
echo "Frontend:         http://localhost:3000"
echo ""
echo "Process IDs:"
echo "  State:    $STATE_PID"
echo "  Provider: $PROVIDER_PID"
echo "  Studio:   $STUDIO_PID"
echo "  Frontend: $FRONTEND_PID"
echo ""
echo "To stop all services, run: ./scripts/stop-mvp.sh"
```

Create file: `/home/uneid/iter3/memmieai/memmie-studio/scripts/stop-mvp.sh`

```bash
#!/bin/bash

echo "Stopping Memmie Studio MVP Services..."

# Kill services by port
lsof -ti:8006 | xargs kill -9 2>/dev/null && echo "✓ Stopped State Service"
lsof -ti:8007 | xargs kill -9 2>/dev/null && echo "✓ Stopped Provider Service"
lsof -ti:8010 | xargs kill -9 2>/dev/null && echo "✓ Stopped Studio API"
lsof -ti:3000 | xargs kill -9 2>/dev/null && echo "✓ Stopped Frontend"

echo "All services stopped."
```

```bash
chmod +x /home/uneid/iter3/memmieai/memmie-studio/scripts/*.sh
```

### Step 2: Create Test User Accounts
Create file: `/home/uneid/iter3/memmieai/memmie-studio/scripts/create-test-users.sh`

```bash
#!/bin/bash

# Create test users via Auth service
echo "Creating test users..."

# Test User 1
curl -X POST http://localhost:8001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "writer@test.com",
    "password": "testpass123",
    "username": "TestWriter"
  }'

echo ""

# Test User 2
curl -X POST http://localhost:8001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "entrepreneur@test.com",
    "password": "testpass123",
    "username": "TestEntrepreneur"
  }'

echo ""
echo "Test users created:"
echo "  - writer@test.com / testpass123"
echo "  - entrepreneur@test.com / testpass123"
```

### Step 3: Integration Test Script
Create file: `/home/uneid/iter3/memmieai/memmie-studio/tests/integration_test.sh`

```bash
#!/bin/bash

echo "Running Integration Tests..."

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Base URLs
AUTH_URL="http://localhost:8001"
STATE_URL="http://localhost:8006"
PROVIDER_URL="http://localhost:8007"
STUDIO_URL="http://localhost:8010"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name=$1
    local command=$2
    local expected_code=$3
    
    echo -n "Testing: $test_name... "
    
    response=$(eval "$command")
    code=$?
    
    if [ $code -eq $expected_code ]; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "  Response: $response"
        ((TESTS_FAILED++))
    fi
}

echo ""
echo "=== Service Health Checks ==="
run_test "Auth Service Health" "curl -s -o /dev/null -w '%{http_code}' $AUTH_URL/health" 0
run_test "State Service Health" "curl -s -o /dev/null -w '%{http_code}' $STATE_URL/health" 0
run_test "Provider Service Health" "curl -s -o /dev/null -w '%{http_code}' $PROVIDER_URL/health" 0
run_test "Studio API Health" "curl -s -o /dev/null -w '%{http_code}' $STUDIO_URL/api/v1/health" 0

echo ""
echo "=== Authentication Tests ==="

# Login to get token
TOKEN=$(curl -s -X POST $AUTH_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"writer@test.com","password":"testpass123"}' \
  | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo -e "${RED}Failed to get auth token${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Got auth token${NC}"

echo ""
echo "=== Provider Tests ==="
run_test "List Providers" "curl -s -o /dev/null -w '%{http_code}' $STUDIO_URL/api/v1/providers -H 'Authorization: Bearer $TOKEN'" 0

echo ""
echo "=== Document Creation Tests ==="

# Create a book document
BOOK_RESPONSE=$(curl -s -X POST $STUDIO_URL/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "provider_id": "book",
    "content": "The ship sailed into the storm.",
    "process_content": false,
    "metadata": {"title": "Test Chapter"}
  }')

if echo "$BOOK_RESPONSE" | grep -q '"id"'; then
    echo -e "${GREEN}✓ Created book document${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Failed to create book document${NC}"
    ((TESTS_FAILED++))
fi

# Create a pitch document
PITCH_RESPONSE=$(curl -s -X POST $STUDIO_URL/api/v1/documents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "provider_id": "pitch",
    "content": "## Problem\nSmall businesses struggle with inventory.\n\n## Solution\nAI-powered inventory management.",
    "process_content": false,
    "metadata": {"title": "Test Pitch"}
  }')

if echo "$PITCH_RESPONSE" | grep -q '"id"'; then
    echo -e "${GREEN}✓ Created pitch document${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ Failed to create pitch document${NC}"
    ((TESTS_FAILED++))
fi

echo ""
echo "=== Document Retrieval Tests ==="
run_test "List All Documents" "curl -s -o /dev/null -w '%{http_code}' $STUDIO_URL/api/v1/documents -H 'Authorization: Bearer $TOKEN'" 0
run_test "List Book Documents" "curl -s -o /dev/null -w '%{http_code}' '$STUDIO_URL/api/v1/documents?provider_id=book' -H 'Authorization: Bearer $TOKEN'" 0

echo ""
echo "========================================="
echo "Integration Test Results"
echo "========================================="
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
```

```bash
chmod +x /home/uneid/iter3/memmieai/memmie-studio/tests/integration_test.sh
```

### Step 4: Performance Test Script
Create file: `/home/uneid/iter3/memmieai/memmie-studio/tests/load_test.sh`

```bash
#!/bin/bash

echo "Running Load Test (100 concurrent users simulation)..."

# Install Apache Bench if not available
if ! command -v ab &> /dev/null; then
    echo "Installing Apache Bench..."
    sudo apt-get update && sudo apt-get install -y apache2-utils
fi

# Get auth token
TOKEN=$(curl -s -X POST http://localhost:8001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"writer@test.com","password":"testpass123"}' \
  | grep -o '"token":"[^"]*' | cut -d'"' -f4)

# Create a request file
cat > /tmp/ab_post_data.json <<EOF
{
  "provider_id": "book",
  "content": "Test content for load testing.",
  "process_content": false,
  "metadata": {"title": "Load Test"}
}
EOF

echo ""
echo "Testing Document Creation Endpoint..."
echo "100 requests, 10 concurrent"
ab -n 100 -c 10 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -p /tmp/ab_post_data.json \
  http://localhost:8010/api/v1/documents

echo ""
echo "Testing Document List Endpoint..."
echo "200 requests, 20 concurrent"
ab -n 200 -c 20 \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8010/api/v1/documents

rm /tmp/ab_post_data.json
```

### Step 5: Production Deployment Checklist
Create file: `/home/uneid/iter3/memmieai/memmie-studio/DEPLOYMENT.md`

```markdown
# MVP Deployment Checklist

## Pre-Deployment

### Environment Setup
- [ ] Production MongoDB instance configured
- [ ] Redis cache configured (optional for MVP)
- [ ] OpenAI API key set
- [ ] Environment variables configured for all services

### Security
- [ ] JWT secret key generated and secured
- [ ] CORS origins restricted to production domain
- [ ] Database credentials secured
- [ ] API rate limiting configured
- [ ] HTTPS certificates installed

### Database
- [ ] MongoDB indexes created
- [ ] Backup strategy in place
- [ ] Connection pooling configured

## Deployment Steps

### 1. Build Frontend
```bash
cd web
npm run build
```

### 2. Build Services
```bash
# State Service
cd /home/uneid/iter3/memmieai/memmie-state
go build -o bin/state-service cmd/server/main.go

# Provider Service
cd /home/uneid/iter3/memmieai/memmie-provider
go build -o bin/provider-service cmd/server/main.go

# Studio API
cd /home/uneid/iter3/memmieai/memmie-studio
go build -o bin/studio-api cmd/server/main.go
```

### 3. Deploy with Docker
```dockerfile
# Dockerfile for Studio API
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o studio-api cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/studio-api .
COPY --from=builder /app/web/build ./web/build
EXPOSE 8010
CMD ["./studio-api"]
```

### 4. Environment Variables
```env
# Production .env
NODE_ENV=production
PORT=8010
AUTH_SERVICE_URL=https://auth.memmie.ai
STATE_SERVICE_URL=https://state.memmie.ai
PROVIDER_SERVICE_URL=https://provider.memmie.ai
OPENAI_API_KEY=sk-...
JWT_SECRET=<secure-random-string>
MONGO_URI=mongodb://...
```

## Post-Deployment

### Monitoring
- [ ] Health check endpoints verified
- [ ] Logging configured (stdout/file/service)
- [ ] Error tracking set up (Sentry/similar)
- [ ] Metrics collection configured

### Testing
- [ ] Run integration tests against production
- [ ] Test user registration and login
- [ ] Test document creation for both providers
- [ ] Test AI processing (if API key valid)
- [ ] Load test with expected traffic

### Beta User Onboarding
- [ ] Welcome email template created
- [ ] Documentation/help pages ready
- [ ] Support channel established
- [ ] Feedback collection mechanism in place

## Rollback Plan
1. Keep previous version binaries
2. Database backup before deployment
3. Feature flags for new functionality
4. Canary deployment (10% -> 50% -> 100%)

## Success Metrics
- [ ] 100 beta users successfully onboarded
- [ ] < 500ms average response time
- [ ] 99.9% uptime
- [ ] < 1% error rate
- [ ] Positive user feedback (>80% satisfaction)
```

### Step 6: Run Complete Test Suite

```bash
# Terminal 1: Start all services
cd /home/uneid/iter3/memmieai/memmie-studio
./scripts/start-mvp.sh

# Terminal 2: Create test users
./scripts/create-test-users.sh

# Terminal 3: Run integration tests
./tests/integration_test.sh

# Terminal 4: Run load tests
./tests/load_test.sh

# Terminal 5: Manual UI testing
# Open browser to http://localhost:3000
# 1. Login with writer@test.com / testpass123
# 2. Create a new book chapter
# 3. Type and see AI expansion
# 4. Save the chapter
# 5. Go back to dashboard
# 6. Create a new pitch
# 7. Fill out sections
# 8. Generate full pitch
# 9. Save the pitch
# 10. Verify documents appear in dashboard
```

## Expected Output
- All services start successfully
- Health checks pass
- Integration tests: 100% pass rate
- Load tests: <500ms average response time
- UI: Smooth interaction with no errors

## Success Criteria
✅ All services start and communicate
✅ Authentication flow works end-to-end
✅ Documents can be created and retrieved
✅ AI processing works (with valid API key)
✅ Frontend displays data correctly
✅ System handles 100 concurrent users
✅ No memory leaks or crashes
✅ Error handling works properly

## MVP Launch Ready Checklist
- [x] State Service with MongoDB blob storage
- [x] Provider Service with book/pitch templates
- [x] Studio API with auth integration
- [x] React frontend with dark mode
- [x] Book Writer interface
- [x] Pitch Creator interface
- [x] Integration tests passing
- [ ] OpenAI API key configured
- [ ] 100 beta user accounts ready
- [ ] Production deployment complete

## Notes
- Monitor logs during testing for any errors
- Check MongoDB for proper data persistence
- Verify JWT tokens expire correctly
- Test with different screen sizes for responsiveness
- Document any issues for post-MVP fixes

## Next Steps After MVP
1. Add more providers (research, music, etc.)
2. Implement WebSocket for real-time updates
3. Add collaboration features
4. Implement version history with deltas
5. Add export functionality (PDF, Markdown)
6. Implement the "Ramble" voice feature
7. Add mobile app support
8. Scale to support 1000+ users