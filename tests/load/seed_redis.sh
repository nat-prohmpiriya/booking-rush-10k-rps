#!/bin/bash
# Seed Redis with zone availability data for load testing

set -e

REDIS_HOST=${REDIS_HOST:-"100.104.0.42"}
REDIS_PORT=${REDIS_PORT:-6379}
REDIS_PASSWORD=${REDIS_PASSWORD:-""}

echo "Seeding Redis at ${REDIS_HOST}:${REDIS_PORT}"

# Zone IDs from seed_data.json
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

# Build redis-cli command with auth if password is set
REDIS_CLI="redis-cli -h ${REDIS_HOST} -p ${REDIS_PORT}"
if [ -n "${REDIS_PASSWORD}" ]; then
    REDIS_CLI="${REDIS_CLI} -a ${REDIS_PASSWORD}"
fi

# Set zone availability in Redis
for zone in "${ZONES[@]}"; do
    key="zone:availability:${zone}"
    echo "Setting ${key} = ${SEATS_PER_ZONE}"
    ${REDIS_CLI} SET "${key}" "${SEATS_PER_ZONE}" > /dev/null
done

echo ""
echo "Redis seeding complete!"
echo "Total zones: ${#ZONES[@]}"
echo "Seats per zone: ${SEATS_PER_ZONE}"
echo "Total available seats: $((${#ZONES[@]} * SEATS_PER_ZONE))"

# Verify seeding
echo ""
echo "Verification:"
${REDIS_CLI} KEYS "zone:availability:load-test-*" | head -5
