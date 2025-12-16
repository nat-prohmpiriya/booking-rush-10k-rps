#!/bin/bash
# Seed script for microservices load testing
# Seeds auth_db, ticket_db, and syncs inventory to Redis

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Load .env.local if exists (same as docker-compose)
if [ -f "$PROJECT_ROOT/.env.local" ]; then
    echo ">> Loading .env.local"
    set -a
    source "$PROJECT_ROOT/.env.local"
    set +a
fi

# Default values matching docker-compose.db.yml
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"

REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-redis123}"

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"

echo "=== Load Test Seed Script for Microservices ==="
echo "PostgreSQL: $POSTGRES_HOST:$POSTGRES_PORT"
echo "Redis: $REDIS_HOST:$REDIS_PORT"
echo "Gateway: $GATEWAY_URL"
echo ""

# Function to run SQL
run_sql() {
    local db=$1
    local file=$2
    echo ">> Seeding $db from $file..."
    PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$db" -f "$file"
}

# Function to run SQL via docker
run_sql_docker() {
    local db=$1
    local file=$2
    echo ">> Seeding $db from $file (via docker)..."
    docker exec -i booking-rush-postgres psql -U postgres -d "$db" < "$file"
}

# Detect if running locally or via docker
if command -v psql &> /dev/null && [ "$POSTGRES_HOST" = "localhost" ]; then
    RUN_SQL="run_sql"
else
    RUN_SQL="run_sql_docker"
fi

# Step 1: Seed auth_db
echo ""
echo "=== Step 1: Seeding auth_db (10,000 users) ==="
$RUN_SQL "auth_db" "$SCRIPT_DIR/seed_auth.sql"

# Step 2: Seed ticket_db
echo ""
echo "=== Step 2: Seeding ticket_db (events, shows, zones) ==="
$RUN_SQL "ticket_db" "$SCRIPT_DIR/seed_ticket.sql"

# Step 3: Sync inventory to Redis
echo ""
echo "=== Step 3: Syncing inventory to Redis ==="

# Get auth token first
echo ">> Getting auth token..."
AUTH_RESPONSE=$(curl -s -X POST "$GATEWAY_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"organizer@test.com","password":"Test123!"}' 2>/dev/null || echo '{}')

TOKEN=$(echo "$AUTH_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo ">> Warning: Could not get auth token, trying admin sync directly..."
    # Try syncing via direct Redis commands
    echo ">> Syncing zones to Redis directly..."

    # Get zone data from ticket_db and sync to Redis
    ZONES=$(docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -A -c \
        "SELECT id || '|' || available_seats FROM seat_zones WHERE id::text LIKE 'b0000000-%' AND is_active = true;")

    while IFS='|' read -r zone_id seats; do
        if [ -n "$zone_id" ]; then
            docker exec booking-rush-redis redis-cli -a "$REDIS_PASSWORD" --no-auth-warning \
                SET "zone:availability:$zone_id" "$seats" > /dev/null 2>&1
        fi
    done <<< "$ZONES"

    SYNCED_COUNT=$(docker exec booking-rush-redis redis-cli -a "$REDIS_PASSWORD" --no-auth-warning \
        KEYS "zone:availability:b0000000-*" 2>/dev/null | wc -l)
    echo ">> Synced $SYNCED_COUNT zones to Redis"
else
    echo ">> Calling sync-inventory API..."
    SYNC_RESPONSE=$(curl -s -X POST "$GATEWAY_URL/api/v1/admin/sync-inventory" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json")
    echo ">> $SYNC_RESPONSE"
fi

# Step 4: Generate seed_data.json for k6
echo ""
echo "=== Step 4: Generating seed_data.json for k6 ==="

# Get user IDs
USER_IDS=$(docker exec booking-rush-postgres psql -U postgres -d auth_db -t -A -c \
    "SELECT json_agg(id) FROM users WHERE email LIKE 'loadtest%@test.com' LIMIT 10000;")

# Get event IDs
EVENT_IDS=$(docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -A -c \
    "SELECT json_agg(id) FROM events WHERE id::text LIKE 'd0000000-%';")

# Get show IDs
SHOW_IDS=$(docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -A -c \
    "SELECT json_agg(id) FROM shows WHERE id::text LIKE 'c0000000-%';")

# Get zone IDs
ZONE_IDS=$(docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -A -c \
    "SELECT json_agg(id) FROM seat_zones WHERE id::text LIKE 'b0000000-%' AND is_active = true;")

# Create seed_data.json
cat > "$SCRIPT_DIR/seed_data.json" << EOF
{
    "userIds": $USER_IDS,
    "eventIds": $EVENT_IDS,
    "showIds": $SHOW_IDS,
    "zoneIds": $ZONE_IDS
}
EOF

echo ">> Generated seed_data.json"

# Summary
echo ""
echo "=== Seed Complete ==="
echo ""
echo "Data Summary:"
docker exec booking-rush-postgres psql -U postgres -d auth_db -t -c \
    "SELECT 'Users: ' || COUNT(*) FROM users WHERE email LIKE 'loadtest%@test.com';"
docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -c \
    "SELECT 'Events: ' || COUNT(*) FROM events WHERE id::text LIKE 'd0000000-%';"
docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -c \
    "SELECT 'Shows: ' || COUNT(*) FROM shows WHERE id::text LIKE 'c0000000-%';"
docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -c \
    "SELECT 'Zones: ' || COUNT(*) FROM seat_zones WHERE id::text LIKE 'b0000000-%';"
docker exec booking-rush-postgres psql -U postgres -d ticket_db -t -c \
    "SELECT 'Total Seats: ' || SUM(available_seats) FROM seat_zones WHERE id::text LIKE 'b0000000-%';"

REDIS_ZONES=$(docker exec booking-rush-redis redis-cli -a "$REDIS_PASSWORD" --no-auth-warning \
    KEYS "zone:availability:b0000000-*" 2>/dev/null | wc -l | tr -d ' ')
echo " Redis Zones: $REDIS_ZONES"

echo ""
echo "Ready for load testing!"
echo "Run: make load-smoke"
