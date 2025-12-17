#!/bin/bash
# Seed all test data using Docker containers
# This script doesn't require local psql or redis-cli installation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Load .env file if exists
if [ -f "${PROJECT_ROOT}/.env" ]; then
    echo "Loading configuration from .env file..."
    set -a
    source "${PROJECT_ROOT}/.env"
    set +a
fi

# Configuration (use .env values or defaults)
POSTGRES_HOST=${POSTGRES_HOST:-"100.104.0.42"}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-"postgres"}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-""}
POSTGRES_DB=${POSTGRES_DB:-"booking_rush"}

REDIS_HOST=${REDIS_HOST:-"100.104.0.42"}
REDIS_PORT=${REDIS_PORT:-6379}
REDIS_PASSWORD=${REDIS_PASSWORD:-""}

echo "=========================================="
echo "Booking Rush - Load Test Data Seeder"
echo "=========================================="

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not in PATH"
    echo "Please install Docker or use native psql/redis-cli"
    exit 1
fi

echo ""
echo "1. Seeding PostgreSQL..."
echo "   Host: ${POSTGRES_HOST}:${POSTGRES_PORT}"
echo "   Database: ${POSTGRES_DB}"

docker run --rm -i \
    -e PGPASSWORD="${POSTGRES_PASSWORD}" \
    postgres:15-alpine \
    psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" < "${SCRIPT_DIR}/seed_data.sql"

echo "   PostgreSQL seeding complete!"

echo ""
echo "2. Seeding Redis..."
echo "   Host: ${REDIS_HOST}:${REDIS_PORT}"

# Zone IDs
ZONES=(
    "load-test-zone-1-1"
    "load-test-zone-1-2"
    "load-test-zone-1-3"
    "load-test-zone-1-4"
    "load-test-zone-1-5"
    "load-test-zone-2-1"
    "load-test-zone-2-2"
    "load-test-zone-2-3"
    "load-test-zone-2-4"
    "load-test-zone-2-5"
    "load-test-zone-3-1"
    "load-test-zone-3-2"
    "load-test-zone-3-3"
    "load-test-zone-3-4"
    "load-test-zone-3-5"
)

SEATS_PER_ZONE=20000

# Build Redis commands
REDIS_COMMANDS=""
for zone in "${ZONES[@]}"; do
    REDIS_COMMANDS="${REDIS_COMMANDS}SET zone:availability:${zone} ${SEATS_PER_ZONE}\n"
done

# Execute Redis commands
if [ -n "${REDIS_PASSWORD}" ]; then
    echo -e "${REDIS_COMMANDS}" | docker run --rm -i redis:7-alpine \
        redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" -a "${REDIS_PASSWORD}" --no-auth-warning
else
    echo -e "${REDIS_COMMANDS}" | docker run --rm -i redis:7-alpine \
        redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}"
fi

echo "   Redis seeding complete!"

echo ""
echo "=========================================="
echo "Summary"
echo "=========================================="
echo "PostgreSQL:"
echo "  - 1 test tenant"
echo "  - 10,000 test users"
echo "  - 3 events, 9 shows, 45 zones"
echo ""
echo "Redis:"
echo "  - ${#ZONES[@]} zone availability keys"
echo "  - ${SEATS_PER_ZONE} seats per zone"
echo "  - Total: $((${#ZONES[@]} * SEATS_PER_ZONE)) available seats"
echo ""
echo "Ready for load testing!"
