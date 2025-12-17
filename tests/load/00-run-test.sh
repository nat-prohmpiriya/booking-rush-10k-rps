#!/bin/bash

# k6 Load Test Runner
# Usage: ./04-run-test.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

AUTH_URL="http://localhost:8080/api/v1/auth/login"

# DB connection
DB_CONTAINER="booking-rush-postgres"
DB_USER="postgres"
DB_PASSWORD="postgres"

# Function to generate tokens
generate_tokens() {
  echo ""
  echo "=== Generating JWT Tokens ==="

  read -p "Number of tokens to generate [500]: " num_tokens
  num_tokens=${num_tokens:-500}

  echo "Generating $num_tokens tokens..."
  cd "$SCRIPT_DIR/seed-data"

  NUM_TOKENS=$num_tokens node generate-tokens.js

  if [ $? -eq 0 ]; then
    TOKEN_COUNT=$(jq 'length' tokens.json 2>/dev/null)
    echo ""
    echo "=== Token Generation Complete ==="
    echo "  File: seed-data/tokens.json"
    echo "  Count: $TOKEN_COUNT tokens"
    echo "  Note: Tokens expire in ~15 minutes"
  else
    echo "ERROR: Token generation failed"
    return 1
  fi

  cd "$SCRIPT_DIR"
  echo ""
}

# Function to reset data
reset_data() {
  echo ""
  echo "=== Resetting Data ==="

  # Clear Redis
  echo "[1/4] Clearing Redis..."
  docker exec booking-rush-redis redis-cli -a redis123 --no-auth-warning FLUSHDB > /dev/null
  echo "Redis cleared"

  # Clear load test bookings from DB
  echo "[2/4] Clearing load test bookings from DB..."
  docker exec $DB_CONTAINER psql -U $DB_USER -d booking_db -c \
    "DELETE FROM bookings WHERE user_id::text LIKE 'a0000000-%';" > /dev/null 2>&1
  docker exec $DB_CONTAINER psql -U $DB_USER -d booking_db -c \
    "DELETE FROM saga_instances WHERE booking_id::text LIKE 'a0000000-%';" > /dev/null 2>&1
  echo "DB cleared"

  # Reset zone available_seats
  echo "[3/4] Resetting zone seats in DB..."
  docker exec $DB_CONTAINER psql -U $DB_USER -d ticket_db -c \
    "UPDATE seat_zones SET available_seats = total_seats WHERE id::text LIKE 'b0000000-%';" > /dev/null 2>&1
  echo "Zones reset"

  # Get admin token for sync (using heredoc to avoid shell escaping issues with !)
  echo "[4/4] Syncing inventory to Redis..."
  ADMIN_TOKEN=$(cat << 'EOF' | curl -s http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d @- | jq -r '.data.access_token'
{"email":"organizer@test.com","password":"Test123!"}
EOF
)

  if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
    echo "ERROR: Failed to get admin token"
    return 1
  fi

  SYNC_RESULT=$(curl -s -X POST http://localhost:8080/api/v1/admin/sync-inventory \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json")
  echo "$SYNC_RESULT" | jq -r '"Synced: \(.zones_synced) zones"'

  echo "=== Reset Complete ==="
  echo ""
}

# Function to run test
run_test() {
  local scenario=$1

  echo ""
  # Ask about bypass gateway
  read -p "Bypass Gateway? (y/n) [n]: " bypass_choice
  BYPASS_GATEWAY="false"
  if [ "$bypass_choice" = "y" ] || [ "$bypass_choice" = "Y" ]; then
    BYPASS_GATEWAY="true"
    echo "  → Testing directly to booking:8083 (bypass gateway)"
  else
    echo "  → Testing via gateway:8080"
  fi
  echo ""

  echo "Getting auth token..."
  TOKEN=$(cat << 'EOF' | curl -s "$AUTH_URL" -H "Content-Type: application/json" -d @- | jq -r '.data.access_token'
{"email":"loadtest1@test.com","password":"Test123!"}
EOF
)

  if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "ERROR: Failed to get token"
    return 1
  fi

  echo "Token: ${TOKEN:0:30}..."
  echo ""

  # Create results folder
  mkdir -p results

  # Generate filename with timestamp
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  SUFFIX=""
  if [ "$BYPASS_GATEWAY" = "true" ]; then
    SUFFIX="-direct"
  fi
  RESULT_FILE="results/${scenario}${SUFFIX}-${TIMESTAMP}"

  echo "Running scenario: $scenario"
  echo "Bypass Gateway: $BYPASS_GATEWAY"
  echo "Results will be saved to: ${RESULT_FILE}.json"
  echo ""

  K6_WEB_DASHBOARD=true k6 run \
    --env AUTH_TOKEN="$TOKEN" \
    --env SCENARIO="$scenario" \
    --env BYPASS_GATEWAY="$BYPASS_GATEWAY" \
    --out json="${RESULT_FILE}.json" \
    --summary-export="${RESULT_FILE}-summary.json" \
    01-booking-reserve.js

  echo ""
  echo "=== Results saved ==="
  echo "  Full:    ${RESULT_FILE}.json"
  echo "  Summary: ${RESULT_FILE}-summary.json"
}

# Function to run Virtual Queue test
run_queue_test() {
  local scenario=$1

  echo ""
  echo "=== Virtual Queue Load Test ==="
  echo "This test simulates:"
  echo "  - 10,000 concurrent users joining queue"
  echo "  - Queue releases 500 users at a time"
  echo "  - Users with queue pass can book"
  echo ""

  # Check if REQUIRE_QUEUE_PASS is enabled
  echo "Checking REQUIRE_QUEUE_PASS setting..."
  QUEUE_CHECK=$(curl -s http://localhost:8080/health | jq -r '.queue_pass_required // "unknown"')
  echo "  Queue Pass Required: $QUEUE_CHECK"
  echo ""

  # Create results folder
  mkdir -p results

  # Generate filename with timestamp
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  RESULT_FILE="results/${scenario}-${TIMESTAMP}"

  echo "Running scenario: $scenario"
  echo "Results will be saved to: ${RESULT_FILE}.json"
  echo ""

  K6_WEB_DASHBOARD=true k6 run \
    --env SCENARIO="$scenario" \
    --out json="${RESULT_FILE}.json" \
    --summary-export="${RESULT_FILE}-summary.json" \
    06-virtual-queue.js

  echo ""
  echo "=== Results saved ==="
  echo "  Full:    ${RESULT_FILE}.json"
  echo "  Summary: ${RESULT_FILE}-summary.json"
  echo ""
  echo "=== Key Metrics to Verify ==="
  echo "  - queue_join_success > 95%"
  echo "  - queue_pass_received > 80%"
  echo "  - booking_success > 90%"
  echo "  - Zero overselling (check DB)"
}

# Main menu
echo "=== k6 Load Test Runner ==="
echo ""
echo "Select option:"
echo "  1) smoke       - 1 VU, 30s (quick test)"
echo "  2) ramp_up     - 0→1000 VUs, 9 min"
echo "  3) sustained   - 5000 RPS, 5 min"
echo "  4) spike       - 1k→10k RPS, 3 min"
echo "  5) stress_10k  - 10000 RPS, 5 min"
echo "  6) all         - Run all scenarios (~25 min)"
echo "  ---"
echo "  7) reset       - Reset all (Redis + DB bookings + zones)"
echo "  8) tokens      - Generate JWT tokens (run before test!)"
echo "  ---"
echo "  9) vq_smoke    - Virtual Queue: 100 users (quick test)"
echo "  10) vq_10k     - Virtual Queue: 10,000 concurrent users"
echo "  11) vq_15k     - Virtual Queue: 15,000 concurrent users (stress)"
echo "  0) exit"
echo ""
read -p "Enter choice [0-11]: " choice

case $choice in
  1) reset_data && run_test "smoke" ;;
  2) reset_data && run_test "ramp_up" ;;
  3) reset_data && run_test "sustained" ;;
  4) reset_data && run_test "spike" ;;
  5) reset_data && run_test "stress_10k" ;;
  6) reset_data && run_test "all" ;;
  7) reset_data ;;
  8) generate_tokens ;;
  9) reset_data && run_queue_test "virtual_queue_smoke" ;;
  10) reset_data && run_queue_test "virtual_queue_10k" ;;
  11) reset_data && run_queue_test "virtual_queue_15k" ;;
  0) echo "Bye!"; exit 0 ;;
  *) echo "Invalid choice"; exit 1 ;;
esac
