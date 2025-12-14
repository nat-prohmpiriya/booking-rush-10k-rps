#!/bin/bash

# Script to run all saga workers for testing

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Load environment variables
if [ -f "$PROJECT_ROOT/.env.local" ]; then
    source "$PROJECT_ROOT/.env.local"
fi

# Export common env vars
export APP_ENVIRONMENT=development
export KAFKA_BROKERS=localhost:19092
export REDIS_HOST=localhost
export REDIS_PORT=6379

# Booking database
export BOOKING_DATABASE_HOST=localhost
export BOOKING_DATABASE_PORT=5432
export BOOKING_DATABASE_USER=postgres
export BOOKING_DATABASE_PASSWORD=postgres
export BOOKING_DATABASE_DBNAME=booking_db
export BOOKING_DATABASE_SSLMODE=disable

# Payment database
export PAYMENT_DATABASE_HOST=localhost
export PAYMENT_DATABASE_PORT=5432
export PAYMENT_DATABASE_USER=postgres
export PAYMENT_DATABASE_PASSWORD=postgres
export PAYMENT_DATABASE_DBNAME=payment_db
export PAYMENT_DATABASE_SSLMODE=disable

echo "=== Building Saga Workers ==="
cd "$PROJECT_ROOT/backend-booking"
go build -o "$PROJECT_ROOT/bin/saga-orchestrator" ./cmd/saga-orchestrator/main.go
go build -o "$PROJECT_ROOT/bin/saga-step-worker" ./cmd/saga-step-worker/main.go

cd "$PROJECT_ROOT/backend-payment"
go build -o "$PROJECT_ROOT/bin/saga-payment-worker" ./cmd/saga-payment-worker/main.go

echo ""
echo "=== Starting Saga Workers ==="
echo "Press Ctrl+C to stop all workers"
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Stopping all workers..."
    kill $(jobs -p) 2>/dev/null
    exit 0
}
trap cleanup SIGINT SIGTERM

# Start workers in background
echo "[1/3] Starting saga-orchestrator..."
"$PROJECT_ROOT/bin/saga-orchestrator" &
sleep 1

echo "[2/3] Starting saga-step-worker..."
"$PROJECT_ROOT/bin/saga-step-worker" &
sleep 1

echo "[3/3] Starting saga-payment-worker..."
"$PROJECT_ROOT/bin/saga-payment-worker" &
sleep 1

echo ""
echo "=== All Workers Started ==="
echo "You can now test the saga flow by calling:"
echo ""
echo 'curl -X POST http://localhost:8083/api/v1/saga/bookings \'
echo '  -H "Content-Type: application/json" \'
echo '  -H "X-User-ID: test-user-1" \'
echo '  -H "X-Idempotency-Key: test-saga-$(date +%s)" \'
echo '  -d '"'"'{"event_id":"<EVENT_ID>","zone_id":"<ZONE_ID>","quantity":1,"total_price":1500}'"'"
echo ""

# Wait for all background jobs
wait
