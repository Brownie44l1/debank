#!/bin/bash
set -e
BASE_DIR="$(dirname "$0")/.."

psql -U postgres -d bank_ledger -f "$BASE_DIR/schema/001_schema.sql"
psql -U postgres -d bank_ledger -f "$BASE_DIR/schema/002_functions.sql"
psql -U postgres -d bank_ledger -f "$BASE_DIR/seeds/001_system_accounts.sql"
psql -U postgres -d bank_ledger -f "$BASE_DIR/seeds/002_test_data.sql"