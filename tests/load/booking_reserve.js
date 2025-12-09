// k6 Load Test Script for Booking Reserve Endpoint
// Target: 10,000 RPS for /bookings/reserve

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const reserveSuccessRate = new Rate('reserve_success_rate');
const reserveFailRate = new Rate('reserve_fail_rate');
const reserveDuration = new Trend('reserve_duration');
const insufficientSeatsErrors = new Counter('insufficient_seats_errors');
const serverErrors = new Counter('server_errors');

// Load test data from environment or use defaults
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8083';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'test-token';

// Test data - loaded from shared array for efficiency
const testData = new SharedArray('test_data', function () {
    // This will be loaded from seed_data.json if exists
    // Otherwise use default test data
    try {
        return JSON.parse(open('./seed_data.json'));
    } catch (e) {
        // Default test data
        return {
            eventIds: ['event-1', 'event-2', 'event-3'],
            zoneIds: ['zone-1', 'zone-2', 'zone-3', 'zone-4', 'zone-5'],
            userIds: Array.from({ length: 10000 }, (_, i) => `user-${i + 1}`)
        };
    }
});

// Get test data
const eventIds = testData.eventIds || testData[0]?.eventIds || ['event-1'];
const zoneIds = testData.zoneIds || testData[0]?.zoneIds || ['zone-1'];
const userIds = testData.userIds || testData[0]?.userIds || ['user-1'];

// Test scenarios configuration
export const options = {
    scenarios: {
        // Scenario 1: Smoke test - verify basic functionality
        smoke: {
            executor: 'constant-vus',
            vus: 1,
            duration: '30s',
            tags: { scenario: 'smoke' },
            exec: 'reserveSeats',
        },

        // Scenario 2: Ramp-up test - gradually increase load
        ramp_up: {
            executor: 'ramping-vus',
            startVUs: 0,
            stages: [
                { duration: '1m', target: 100 },   // Ramp to 100 VUs
                { duration: '2m', target: 500 },   // Ramp to 500 VUs
                { duration: '3m', target: 1000 },  // Ramp to 1000 VUs
                { duration: '2m', target: 500 },   // Scale down
                { duration: '1m', target: 0 },     // Ramp down to 0
            ],
            tags: { scenario: 'ramp_up' },
            exec: 'reserveSeats',
            startTime: '35s', // Start after smoke test
        },

        // Scenario 3: Sustained load - maintain high load
        sustained: {
            executor: 'constant-arrival-rate',
            rate: 5000,           // 5000 iterations per timeUnit
            timeUnit: '1s',       // = 5000 RPS
            duration: '5m',
            preAllocatedVUs: 1000,
            maxVUs: 2000,
            tags: { scenario: 'sustained' },
            exec: 'reserveSeats',
            startTime: '10m', // Start after ramp_up
        },

        // Scenario 4: Spike test - sudden traffic spike
        spike: {
            executor: 'ramping-arrival-rate',
            startRate: 1000,
            timeUnit: '1s',
            stages: [
                { duration: '30s', target: 1000 },  // Stay at 1000 RPS
                { duration: '10s', target: 10000 }, // Spike to 10k RPS
                { duration: '1m', target: 10000 },  // Stay at 10k RPS
                { duration: '10s', target: 1000 },  // Drop back to 1000 RPS
                { duration: '1m', target: 1000 },   // Stay at 1000 RPS
            ],
            preAllocatedVUs: 2000,
            maxVUs: 5000,
            tags: { scenario: 'spike' },
            exec: 'reserveSeats',
            startTime: '16m', // Start after sustained
        },

        // Scenario 5: 10k RPS stress test - target performance
        stress_10k: {
            executor: 'constant-arrival-rate',
            rate: 10000,          // 10000 iterations per timeUnit
            timeUnit: '1s',       // = 10,000 RPS
            duration: '5m',
            preAllocatedVUs: 2000,
            maxVUs: 5000,
            tags: { scenario: 'stress_10k' },
            exec: 'reserveSeats',
            startTime: '20m', // Start after spike
        },
    },

    thresholds: {
        // HTTP request duration
        'http_req_duration': ['p(95)<500', 'p(99)<1000'], // 95th < 500ms, 99th < 1s
        'http_req_duration{scenario:smoke}': ['p(95)<200'],
        'http_req_duration{scenario:sustained}': ['p(95)<500'],
        'http_req_duration{scenario:stress_10k}': ['p(95)<1000'],

        // Success rate
        'reserve_success_rate': ['rate>0.95'], // 95% success rate
        'reserve_success_rate{scenario:smoke}': ['rate>0.99'],

        // Error rates
        'http_req_failed': ['rate<0.05'], // Less than 5% failure

        // Custom metrics
        'reserve_duration': ['p(95)<500', 'avg<200'],
    },
};

// Default function - reserve seats
export default function () {
    reserveSeats();
}

// Reserve seats function
export function reserveSeats() {
    const userId = randomItem(userIds);
    const eventId = randomItem(eventIds);
    const zoneId = randomItem(zoneIds);
    const quantity = randomIntBetween(1, 4);
    const idempotencyKey = `${userId}-${eventId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const payload = JSON.stringify({
        event_id: eventId,
        zone_id: zoneId,
        quantity: quantity,
        unit_price: 100.00,
        idempotency_key: idempotencyKey,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${AUTH_TOKEN}`,
            'X-User-ID': userId,
        },
        tags: { name: 'ReserveSeats' },
    };

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/bookings/reserve`, payload, params);
    const duration = Date.now() - startTime;

    // Record custom metrics
    reserveDuration.add(duration);

    // Check response
    const success = check(response, {
        'status is 201': (r) => r.status === 201,
        'has booking_id': (r) => {
            try {
                const body = JSON.parse(r.body);
                return body.booking_id !== undefined;
            } catch (e) {
                return false;
            }
        },
        'response time OK': (r) => r.timings.duration < 1000,
    });

    // Record success/failure rates
    reserveSuccessRate.add(success);
    reserveFailRate.add(!success);

    // Track specific error types
    if (!success) {
        if (response.status === 409) {
            insufficientSeatsErrors.add(1);
        } else if (response.status >= 500) {
            serverErrors.add(1);
        }
    }

    // Small sleep to prevent overwhelming during VU-based scenarios
    sleep(randomIntBetween(1, 10) / 1000); // 1-10ms
}

// Lifecycle functions
export function setup() {
    console.log(`Starting load test against ${BASE_URL}`);
    console.log(`Test data: ${eventIds.length} events, ${zoneIds.length} zones, ${userIds.length} users`);

    // Verify API is reachable
    const healthCheck = http.get(`${BASE_URL}/health`, {
        timeout: '10s',
    });

    if (healthCheck.status !== 200) {
        console.warn(`Health check failed: ${healthCheck.status}`);
    }

    return {
        startTime: new Date().toISOString(),
    };
}

export function teardown(data) {
    console.log(`Load test completed. Started at: ${data.startTime}`);
}
