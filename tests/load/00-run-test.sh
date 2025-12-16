#!/bin/bash

# k6 Load Test Runner
# Usage: ./04-run-test.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

AUTH_URL="http://localhost:8080/api/v1/auth/login"
EMAIL="loadtest1@test.com"
PASSWORD='Test123!'

# DB connection
DB_CONTAINER="booking-rush-postgres"
DB_USER="postgres"
DB_PASSWORD="postgres"

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

  # Get admin token for sync
  echo "[4/4] Syncing inventory to Redis..."
  ADMIN_TOKEN=$(curl -s http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"organizer@test.com\",\"password\":\"Test123!\"}" | jq -r '.data.access_token')

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
  echo "Getting auth token..."
  TOKEN=$(curl -s "$AUTH_URL" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | jq -r '.data.access_token')

  if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "ERROR: Failed to get token"
    return 1
  fi

  echo "Token: ${TOKEN:0:30}..."
  echo ""

  echo "Running scenario: $scenario"
  K6_WEB_DASHBOARD=true k6 run --env AUTH_TOKEN="$TOKEN" --env SCENARIO="$scenario" 01-booking-reserve.js
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
echo "  0) exit"
echo ""
read -p "Enter choice [0-7]: " choice

case $choice in
  1) reset_data && run_test "smoke" ;;
  2) reset_data && run_test "ramp_up" ;;
  3) reset_data && run_test "sustained" ;;
  4) reset_data && run_test "spike" ;;
  5) reset_data && run_test "stress_10k" ;;
  6) reset_data && run_test "all" ;;
  7) reset_data ;;
  0) echo "Bye!"; exit 0 ;;
  *) echo "Invalid choice"; exit 1 ;;
esac
