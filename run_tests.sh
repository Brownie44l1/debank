#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "🚀 Starting Wallet API Test Suite"

# 1. Check database connection
echo -e "\n${GREEN}[1/4] Checking database connection...${NC}"
if psql "$DATABASE_URL" -c "SELECT 1" > /dev/null 2>&1; then
    echo "✅ Database connected"
else
    echo -e "${RED}❌ Database connection failed${NC}"
    exit 1
fi

# 2. Setup test data
echo -e "\n${GREEN}[2/4] Setting up test data...${NC}"
psql $DATABASE_URL -f setup_test_data.sql
echo "✅ Test data loaded"

# 3. Run Go tests
echo -e "\n${GREEN}[3/4] Running Go tests...${NC}"
# Run all tests
go test ./internal/repository -v
go test ./internal/service -v

# Run specific test
go test ./internal/repository -v -run TestGetAccountByUserID_Success

# Run with coverage
go test ./internal/repository -v -cover
go test ./internal/service -v -cover
echo "✅ Go tests completed"

# 4. Start API server (optional)
echo -e "\n${GREEN}[4/4] Starting API server...${NC}"
echo "Run 'go run cmd/server/main.go' in another terminal"
echo "Then test with Postman!"

echo -e "\n${GREEN}✅ Test setup complete!${NC}"