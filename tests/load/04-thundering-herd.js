// k6 Load Test Script for Thundering Herd Rejection Efficiency
// Tests: Rate limiting speed, sold out rejection speed, resource stability
//
// Run: k6 run thundering_herd.js --env SCENARIO=<scenario>
// Scenarios: rate_limit_429, sold_out_rejection, spike_20k, all

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const rejection429Rate = new Rate('rejection_429_rate');
const rejection429Duration = new Trend('rejection_429_duration_ms');
const rejection429Fast = new Rate('rejection_429_under_5ms');
const soldOutDuration = new Trend('sold_out_duration_ms');
const soldOutFast = new Rate('sold_out_under_5ms');
const successfulRequests = new Counter('successful_requests');
const rateLimitedRequests = new Counter('rate_limited_requests');
const soldOutRequests = new Counter('sold_out_requests');
const serverErrors = new Counter('server_errors');
const retryAfterPresent = new Rate('retry_after_header_present');
const rateLimitHeadersPresent = new Rate('rate_limit_headers_present');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8083';
const GATEWAY_URL = __ENV.GATEWAY_URL || 'http://localhost:8080';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'test-token';
const SCENARIO = __ENV.SCENARIO || 'all';

// Test data
let testDataConfig;
try {
    testDataConfig = JSON.parse(open('./thundering_herd_seed.json'));
} catch (e) {
    testDataConfig = {
        events: ['thundering-herd-event-1'],
        zones: {
            soldOut: ['sold-out-zone-1'],    // Zone with 0 seats
            limited: ['limited-zone-1'],      // Zone with 10 seats
            normal: ['normal-zone-1'],        // Zone with 10000 seats
        },
    };
}

// User pool
const NUM_USERS = 10000;
const userIds = new SharedArray('user_ids', function () {
    return Array.from({ length: NUM_USERS }, (_, i) => `thundering-herd-user-${i + 1}`);
});

// Scenario configurations
const scenarios = {
    // Scenario 1: Test 429 response speed under rate limiting
    rate_limit_429: {
        executor: 'constant-arrival-rate',
        rate: 5000,           // Try 5000 RPS to trigger rate limits
        timeUnit: '1s',
        duration: '30s',
        preAllocatedVUs: 500,
        maxVUs: 1000,
        tags: { scenario: 'rate_limit_429' },
        exec: 'testRateLimiting',
    },

    // Scenario 2: Test sold out rejection speed
    sold_out_rejection: {
        executor: 'constant-arrival-rate',
        rate: 2000,           // 2000 RPS on sold out zone
        timeUnit: '1s',
        duration: '30s',
        preAllocatedVUs: 300,
        maxVUs: 500,
        tags: { scenario: 'sold_out_rejection' },
        exec: 'testSoldOutRejection',
    },

    // Scenario 3: 20k RPS spike test
    spike_20k: {
        executor: 'ramping-arrival-rate',
        startRate: 1000,
        timeUnit: '1s',
        stages: [
            { duration: '10s', target: 5000 },   // Ramp to 5k
            { duration: '10s', target: 10000 },  // Ramp to 10k
            { duration: '10s', target: 20000 },  // Spike to 20k
            { duration: '30s', target: 20000 },  // Hold at 20k
            { duration: '10s', target: 1000 },   // Ramp down
        ],
        preAllocatedVUs: 2000,
        maxVUs: 5000,
        tags: { scenario: 'spike_20k' },
        exec: 'testSpikeLoad',
    },

    // Scenario 4: Concurrency test - thundering herd on limited inventory
    thundering_herd: {
        executor: 'per-vu-iterations',
        vus: 100,             // 100 concurrent users
        iterations: 1,         // Each tries once
        maxDuration: '10s',
        tags: { scenario: 'thundering_herd' },
        exec: 'testThunderingHerd',
    },
};

// Select scenarios based on SCENARIO env var
function getScenarios() {
    if (SCENARIO === 'all') {
        return scenarios;
    }
    if (scenarios[SCENARIO]) {
        return { [SCENARIO]: scenarios[SCENARIO] };
    }
    // Default to rate limiting test
    return { rate_limit_429: scenarios.rate_limit_429 };
}

export const options = {
    scenarios: getScenarios(),
    thresholds: {
        // 429 responses should be fast (< 5ms)
        'rejection_429_duration_ms': ['p(95)<5', 'p(99)<10'],
        'rejection_429_under_5ms': ['rate>0.95'],

        // Sold out rejections should be fast (< 5ms)
        'sold_out_duration_ms': ['p(95)<5', 'p(99)<10'],
        'sold_out_under_5ms': ['rate>0.95'],

        // Rate limit headers should always be present
        'rate_limit_headers_present': ['rate>0.99'],
        'retry_after_header_present': ['rate>0.90'],

        // General HTTP thresholds
        'http_req_failed': ['rate<0.50'],  // Allow up to 50% failures (expected for 429s)
    },
};

// Test 1: Rate Limiting 429 Response Speed
export function testRateLimiting() {
    const userIndex = randomIntBetween(0, userIds.length - 1);
    const userId = userIds[userIndex];
    const zoneId = randomItem(testDataConfig.zones.normal);
    const idempotencyKey = `rate-limit-test-${userId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const payload = JSON.stringify({
        event_id: testDataConfig.events[0],
        zone_id: zoneId,
        quantity: 1,
        unit_price: 100.00,
        idempotency_key: idempotencyKey,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${AUTH_TOKEN}`,
            'X-User-ID': userId,
        },
        tags: { name: 'RateLimitTest' },
    };

    const startTime = Date.now();
    const response = http.post(`${GATEWAY_URL}/bookings/reserve`, payload, params);
    const duration = Date.now() - startTime;

    // Check for rate limit headers
    const hasRateLimitHeaders =
        response.headers['X-Ratelimit-Limit'] !== undefined ||
        response.headers['X-RateLimit-Limit'] !== undefined;
    rateLimitHeadersPresent.add(hasRateLimitHeaders);

    if (response.status === 429) {
        // Rate limited
        rejection429Rate.add(true);
        rejection429Duration.add(duration);
        rejection429Fast.add(duration < 5);
        rateLimitedRequests.add(1);

        // Check Retry-After header
        const hasRetryAfter = response.headers['Retry-After'] !== undefined;
        retryAfterPresent.add(hasRetryAfter);

        check(response, {
            '429 response fast': (r) => duration < 5,
            'has Retry-After header': (r) => r.headers['Retry-After'] !== undefined,
            'has X-RateLimit headers': () => hasRateLimitHeaders,
            'error message clear': (r) => {
                try {
                    const body = JSON.parse(r.body);
                    return body.error && body.error.message && body.error.message.includes('retry');
                } catch {
                    return false;
                }
            },
        });
    } else if (response.status === 201) {
        successfulRequests.add(1);
        rejection429Rate.add(false);
    } else if (response.status >= 500) {
        serverErrors.add(1);
    }
}

// Test 2: Sold Out Rejection Speed
export function testSoldOutRejection() {
    const userIndex = randomIntBetween(0, userIds.length - 1);
    const userId = userIds[userIndex];
    const zoneId = randomItem(testDataConfig.zones.soldOut);
    const idempotencyKey = `sold-out-test-${userId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const payload = JSON.stringify({
        event_id: testDataConfig.events[0],
        zone_id: zoneId,
        quantity: 1,
        unit_price: 100.00,
        idempotency_key: idempotencyKey,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${AUTH_TOKEN}`,
            'X-User-ID': userId,
        },
        tags: { name: 'SoldOutTest' },
    };

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/bookings/reserve`, payload, params);
    const duration = Date.now() - startTime;

    if (response.status === 409) {
        // Sold out (INSUFFICIENT_SEATS returns 409)
        soldOutDuration.add(duration);
        soldOutFast.add(duration < 5);
        soldOutRequests.add(1);

        check(response, {
            'sold out response fast (< 5ms)': () => duration < 5,
            'has INSUFFICIENT_SEATS error code': (r) => {
                try {
                    const body = JSON.parse(r.body);
                    return body.code === 'INSUFFICIENT_SEATS';
                } catch {
                    return false;
                }
            },
            'error message helpful': (r) => {
                try {
                    const body = JSON.parse(r.body);
                    return body.error && body.error.toLowerCase().includes('seat');
                } catch {
                    return false;
                }
            },
        });
    } else if (response.status === 201) {
        // Should not happen if zone is sold out
        successfulRequests.add(1);
    } else if (response.status >= 500) {
        serverErrors.add(1);
    }
}

// Test 3: 20k RPS Spike Load
export function testSpikeLoad() {
    const userIndex = randomIntBetween(0, userIds.length - 1);
    const userId = userIds[userIndex];
    const zoneId = randomItem(testDataConfig.zones.normal);
    const idempotencyKey = `spike-test-${userId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const payload = JSON.stringify({
        event_id: testDataConfig.events[0],
        zone_id: zoneId,
        quantity: 1,
        unit_price: 100.00,
        idempotency_key: idempotencyKey,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${AUTH_TOKEN}`,
            'X-User-ID': userId,
        },
        tags: { name: 'SpikeTest' },
        timeout: '10s',
    };

    const startTime = Date.now();
    const response = http.post(`${GATEWAY_URL}/bookings/reserve`, payload, params);
    const duration = Date.now() - startTime;

    if (response.status === 429) {
        rejection429Rate.add(true);
        rejection429Duration.add(duration);
        rejection429Fast.add(duration < 5);
        rateLimitedRequests.add(1);
        retryAfterPresent.add(response.headers['Retry-After'] !== undefined);
    } else if (response.status === 201) {
        successfulRequests.add(1);
        rejection429Rate.add(false);
    } else if (response.status === 409) {
        soldOutRequests.add(1);
    } else if (response.status >= 500) {
        serverErrors.add(1);
    }

    check(response, {
        'no resource exhaustion (no 503)': (r) => r.status !== 503,
        'response received (no timeout)': (r) => r.status !== 0,
    });
}

// Test 4: Thundering Herd on Limited Inventory
export function testThunderingHerd() {
    const userId = `thundering-herd-user-${__VU}`;
    const zoneId = testDataConfig.zones.limited[0];
    const idempotencyKey = `herd-test-${userId}-${Date.now()}`;

    const payload = JSON.stringify({
        event_id: testDataConfig.events[0],
        zone_id: zoneId,
        quantity: 1,
        unit_price: 100.00,
        idempotency_key: idempotencyKey,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${AUTH_TOKEN}`,
            'X-User-ID': userId,
        },
        tags: { name: 'ThunderingHerdTest' },
    };

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/bookings/reserve`, payload, params);
    const duration = Date.now() - startTime;

    if (response.status === 201) {
        successfulRequests.add(1);
        console.log(`VU ${__VU} got seat in ${duration}ms`);
    } else if (response.status === 409) {
        soldOutRequests.add(1);
        soldOutDuration.add(duration);
        soldOutFast.add(duration < 5);
        console.log(`VU ${__VU} rejected (sold out) in ${duration}ms`);
    } else if (response.status === 429) {
        rateLimitedRequests.add(1);
        rejection429Duration.add(duration);
    }

    check(response, {
        'response is success or sold out': (r) => r.status === 201 || r.status === 409 || r.status === 429,
        'rejection fast for losers': () => response.status === 201 || duration < 10,
    });
}

// Default function
export default function () {
    if (SCENARIO === 'rate_limit_429') {
        testRateLimiting();
    } else if (SCENARIO === 'sold_out_rejection') {
        testSoldOutRejection();
    } else if (SCENARIO === 'spike_20k') {
        testSpikeLoad();
    } else if (SCENARIO === 'thundering_herd') {
        testThunderingHerd();
    } else {
        testRateLimiting();
    }
}

// Setup function
export function setup() {
    console.log(`Starting Thundering Herd test against:`);
    console.log(`  - Booking Service: ${BASE_URL}`);
    console.log(`  - API Gateway: ${GATEWAY_URL}`);
    console.log(`  - Scenario: ${SCENARIO}`);

    // Health check
    const bookingHealth = http.get(`${BASE_URL}/health`, { timeout: '10s' });
    if (bookingHealth.status !== 200) {
        console.warn(`Booking Service health check failed: ${bookingHealth.status}`);
    }

    const gatewayHealth = http.get(`${GATEWAY_URL}/health`, { timeout: '10s' });
    if (gatewayHealth.status !== 200) {
        console.warn(`API Gateway health check failed: ${gatewayHealth.status}`);
    }

    return {
        startTime: new Date().toISOString(),
    };
}

// Teardown function
export function teardown(data) {
    console.log(`\n=== Thundering Herd Test Complete ===`);
    console.log(`Started: ${data.startTime}`);
    console.log(`Ended: ${new Date().toISOString()}`);
}

// Handle summary
export function handleSummary(data) {
    const summary = {
        testName: 'Thundering Herd Rejection Efficiency',
        scenario: SCENARIO,
        timestamp: new Date().toISOString(),
        metrics: {
            rejection429: {
                p95: data.metrics.rejection_429_duration_ms?.values?.['p(95)'],
                p99: data.metrics.rejection_429_duration_ms?.values?.['p(99)'],
                under5ms: data.metrics.rejection_429_under_5ms?.values?.rate,
            },
            soldOut: {
                p95: data.metrics.sold_out_duration_ms?.values?.['p(95)'],
                p99: data.metrics.sold_out_duration_ms?.values?.['p(99)'],
                under5ms: data.metrics.sold_out_under_5ms?.values?.rate,
            },
            counts: {
                successful: data.metrics.successful_requests?.values?.count,
                rateLimited: data.metrics.rate_limited_requests?.values?.count,
                soldOut: data.metrics.sold_out_requests?.values?.count,
                serverErrors: data.metrics.server_errors?.values?.count,
            },
            headers: {
                retryAfterPresent: data.metrics.retry_after_header_present?.values?.rate,
                rateLimitHeadersPresent: data.metrics.rate_limit_headers_present?.values?.rate,
            },
        },
        thresholds: data.thresholds,
    };

    return {
        'thundering_herd_results.json': JSON.stringify(summary, null, 2),
        stdout: generateTextReport(summary),
    };
}

function generateTextReport(summary) {
    let report = `
╔════════════════════════════════════════════════════════════════════╗
║           THUNDERING HERD REJECTION EFFICIENCY REPORT              ║
╠════════════════════════════════════════════════════════════════════╣
║ Scenario: ${summary.scenario.padEnd(54)}║
║ Timestamp: ${summary.timestamp.padEnd(53)}║
╠════════════════════════════════════════════════════════════════════╣
║ 429 RATE LIMIT RESPONSES                                           ║
╠────────────────────────────────────────────────────────────────────╣
║ P95 Latency:      ${formatMs(summary.metrics.rejection429.p95).padEnd(46)}║
║ P99 Latency:      ${formatMs(summary.metrics.rejection429.p99).padEnd(46)}║
║ Under 5ms:        ${formatPercent(summary.metrics.rejection429.under5ms).padEnd(46)}║
║ Target:           < 5ms (P95)                                      ║
╠════════════════════════════════════════════════════════════════════╣
║ SOLD OUT RESPONSES                                                 ║
╠────────────────────────────────────────────────────────────────────╣
║ P95 Latency:      ${formatMs(summary.metrics.soldOut.p95).padEnd(46)}║
║ P99 Latency:      ${formatMs(summary.metrics.soldOut.p99).padEnd(46)}║
║ Under 5ms:        ${formatPercent(summary.metrics.soldOut.under5ms).padEnd(46)}║
║ Target:           < 5ms (Lua script immediate return)              ║
╠════════════════════════════════════════════════════════════════════╣
║ REQUEST COUNTS                                                     ║
╠────────────────────────────────────────────────────────────────────╣
║ Successful:       ${formatCount(summary.metrics.counts.successful).padEnd(46)}║
║ Rate Limited:     ${formatCount(summary.metrics.counts.rateLimited).padEnd(46)}║
║ Sold Out:         ${formatCount(summary.metrics.counts.soldOut).padEnd(46)}║
║ Server Errors:    ${formatCount(summary.metrics.counts.serverErrors).padEnd(46)}║
╠════════════════════════════════════════════════════════════════════╣
║ HEADER COMPLIANCE                                                  ║
╠────────────────────────────────────────────────────────────────────╣
║ Retry-After:      ${formatPercent(summary.metrics.headers.retryAfterPresent).padEnd(46)}║
║ X-RateLimit-*:    ${formatPercent(summary.metrics.headers.rateLimitHeadersPresent).padEnd(46)}║
╚════════════════════════════════════════════════════════════════════╝
`;
    return report;
}

function formatMs(value) {
    if (value === undefined) return 'N/A';
    return `${value.toFixed(2)}ms`;
}

function formatPercent(value) {
    if (value === undefined) return 'N/A';
    return `${(value * 100).toFixed(2)}%`;
}

function formatCount(value) {
    if (value === undefined) return '0';
    return value.toLocaleString();
}
