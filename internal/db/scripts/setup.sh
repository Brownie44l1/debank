#!/bin/bash
# ============================================
# Database Setup Script
# ============================================
# Sets up the complete database with schema, functions, and test data
# Usage: ./setup.sh [--prod]
#   --prod: Skip test data (for production)
# ============================================

set -e  # Exit on any error

# Configuration
DB_USER=${DB_USER:-postgres}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-bank_ledger}

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Parse arguments
SKIP_TEST_DATA=false
if [ "$1" == "--prod" ]; then
    SKIP_TEST_DATA=true
    echo -e "${YELLOW}⚠️  Production mode: Skipping test data${NC}"
fi

echo -e "${GREEN}Starting database setup...${NC}"
echo "Database: $DB_NAME"
echo "Host: $DB_HOST:$DB_PORT"
echo "User: $DB_USER"
echo ""

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DB_DIR="$SCRIPT_DIR/.."

# Function to run SQL file
run_sql() {
    local file=$1
    local description=$2
    
    echo -e "${GREEN}Running: $description${NC}"
    PGPASSWORD=$DB_PASSWORD psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -f "$file"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $description completed${NC}\n"
    else
        echo -e "${RED}✗ $description failed${NC}"
        exit 1
    fi
}

# 1. Create schema
run_sql "$DB_DIR/schema/001_schema.sql" "Schema (tables and indexes)"

# 2. Create functions and triggers
run_sql "$DB_DIR/schema/002_functions.sql" "Functions and triggers"

# 3. Seed system accounts (always run)
run_sql "$DB_DIR/seeds/001_system_accounts.sql" "System accounts"

# 4. Seed test data (skip in production)
if [ "$SKIP_TEST_DATA" = false ]; then
    run_sql "$DB_DIR/seeds/002_test_data.sql" "Test data"
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ Database setup complete!${NC}"
echo -e "${GREEN}========================================${NC}"

# Show final state
echo -e "\n${GREEN}Current database state:${NC}"
PGPASSWORD=$DB_PASSWORD psql -U $DB_USER -h $DB_HOST -p $DB_PORT -d $DB_NAME -c "\dt"