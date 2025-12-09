# Load Testing with k6

This directory contains k6 load testing scripts for the Booking Rush system, targeting 10,000 RPS for the `/bookings/reserve` endpoint.

## Prerequisites

### Install k6

**macOS:**
```bash
brew install k6
```

**Linux (Debian/Ubuntu):**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Docker:**
```bash
docker pull grafana/k6
```

## Test Data Setup

### 1. Seed PostgreSQL

Run the SQL seed script to create test data in the database:

```bash
# Using psql
psql -h 100.104.0.42 -U booking_user -d booking_db -f seed_data.sql

# Or using make (if configured)
make seed-load-test
```

This creates:
- 1 test tenant
- 10,000 test users
- 3 events with 3 shows each (9 shows total)
- 5 seat zones per show (45 zones total)
- 20,000 seats per zone (900,000 total seats)

### 2. Seed Redis

Run the Redis seed script to initialize zone availability:

```bash
# Make executable
chmod +x seed_redis.sh

# Run with environment variables
REDIS_HOST=100.104.0.42 REDIS_PASSWORD=yourpassword ./seed_redis.sh
```

### 3. Generate Full Test Data JSON (Optional)

If you need user IDs in the JSON file:

```bash
node generate_test_data.js seed_data.json
```

## Running Tests

### Quick Start (Smoke Test Only)

Test basic functionality with minimal load:

```bash
k6 run --env BASE_URL=http://localhost:8083 booking_reserve.js
```

### Run Specific Scenarios

```bash
# Smoke test only
k6 run --env BASE_URL=http://localhost:8083 --tag scenario:smoke booking_reserve.js

# Ramp-up test
k6 run --env BASE_URL=http://localhost:8083 --tag scenario:ramp_up booking_reserve.js

# Sustained load test (5000 RPS)
k6 run --env BASE_URL=http://localhost:8083 --tag scenario:sustained booking_reserve.js

# Spike test
k6 run --env BASE_URL=http://localhost:8083 --tag scenario:spike booking_reserve.js

# 10k RPS stress test
k6 run --env BASE_URL=http://localhost:8083 --tag scenario:stress_10k booking_reserve.js
```

### Full Test Suite

Run all scenarios sequentially:

```bash
k6 run --env BASE_URL=http://localhost:8083 booking_reserve.js
```

**Timeline:**
- 0-35s: Smoke test (1 VU)
- 35s-10m: Ramp-up test (0→1000 VUs)
- 10m-15m: Sustained load (5000 RPS)
- 16m-19m: Spike test (1000→10000 RPS)
- 20m-25m: Stress test (10000 RPS)

Total duration: ~25 minutes

### Using Docker

```bash
docker run --rm -i grafana/k6 run \
  --env BASE_URL=http://host.docker.internal:8083 \
  - < booking_reserve.js
```

### With Custom Authentication

```bash
k6 run \
  --env BASE_URL=http://localhost:8083 \
  --env AUTH_TOKEN=your-jwt-token \
  booking_reserve.js
```

## Test Scenarios

| Scenario | Type | Target | Duration | Description |
|----------|------|--------|----------|-------------|
| smoke | VU-based | 1 VU | 30s | Basic functionality check |
| ramp_up | VU-based | 0→1000 VUs | 9m | Gradual load increase |
| sustained | RPS-based | 5000 RPS | 5m | Maintain high throughput |
| spike | RPS-based | 1000→10000 RPS | 2.5m | Sudden traffic spike |
| stress_10k | RPS-based | 10000 RPS | 5m | Target performance test |

## Thresholds

The test is configured with these performance thresholds:

| Metric | Threshold | Description |
|--------|-----------|-------------|
| http_req_duration p(95) | < 500ms | 95th percentile response time |
| http_req_duration p(99) | < 1000ms | 99th percentile response time |
| reserve_success_rate | > 95% | Successful reservations |
| http_req_failed | < 5% | HTTP failure rate |

## Custom Metrics

| Metric | Type | Description |
|--------|------|-------------|
| reserve_success_rate | Rate | Successful reserve operations |
| reserve_fail_rate | Rate | Failed reserve operations |
| reserve_duration | Trend | Custom timing for reserve |
| insufficient_seats_errors | Counter | 409 Conflict errors |
| server_errors | Counter | 5xx errors |

## Output Options

### JSON Output
```bash
k6 run --out json=results.json booking_reserve.js
```

### InfluxDB (for Grafana dashboards)
```bash
k6 run --out influxdb=http://localhost:8086/k6 booking_reserve.js
```

### Cloud (k6 Cloud)
```bash
k6 cloud booking_reserve.js
```

## Troubleshooting

### "Connection refused" errors
- Ensure the booking service is running
- Check firewall settings
- Verify BASE_URL is correct

### High error rates
- Check Redis connectivity and seeded data
- Verify PostgreSQL has test data
- Check service logs for errors

### Performance issues
- Ensure services have adequate resources
- Check database connection pool settings
- Monitor Redis memory usage

### Insufficient VUs
For 10k RPS, you may need:
- Increase `maxVUs` in scenarios
- Run k6 distributed: `k6 run --execution-segment 0:1/2` and `k6 run --execution-segment 1/2:1`

## Clean Up

Remove test data after testing:

```sql
-- PostgreSQL cleanup
DELETE FROM bookings WHERE user_id LIKE 'load-test-user-%';
DELETE FROM seat_zones WHERE id LIKE 'load-test-%';
DELETE FROM shows WHERE id LIKE 'load-test-%';
DELETE FROM events WHERE id LIKE 'load-test-%';
DELETE FROM users WHERE id LIKE 'load-test-%';
DELETE FROM tenants WHERE id = 'load-test-tenant';
```

```bash
# Redis cleanup
redis-cli -h 100.104.0.42 KEYS "zone:availability:load-test-*" | xargs redis-cli -h 100.104.0.42 DEL
```
