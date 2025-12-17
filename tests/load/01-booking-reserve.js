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
const BYPASS_GATEWAY = __ENV.BYPASS_GATEWAY === 'true';
const GATEWAY_URL = __ENV.GATEWAY_URL || 'http://localhost:8080/api/v1';
const BOOKING_URL = __ENV.BOOKING_URL || 'http://localhost:8083/api/v1';
const BASE_URL = BYPASS_GATEWAY ? BOOKING_URL : (__ENV.BASE_URL || GATEWAY_URL);
const AUTH_URL = GATEWAY_URL; // Always use gateway for auth
const AUTH_EMAIL = __ENV.AUTH_EMAIL || 'loadtest1@test.com';
const AUTH_PASSWORD = __ENV.AUTH_PASSWORD || 'Test123!';

// Token will be set during setup (fallback)
let AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

// Load pre-generated tokens from JSON file (for multi-user testing)
let userTokens = [];
try {
    userTokens = JSON.parse(open('./seed-data/tokens.json'));
    console.log(`Loaded ${userTokens.length} pre-generated tokens`);
} catch (e) {
    console.log('No pre-generated tokens found, will use single token');
}

// SharedArray for tokens (memory efficient)
const tokens = new SharedArray('tokens', function () {
    return userTokens.length > 0 ? userTokens : [];
});

// Load test data from JSON file
let testDataConfig;
try {
    testDataConfig = JSON.parse(open('./seed-data/data.json'));
} catch (e) {
    // Default test data
    testDataConfig = {
        eventIds: ['load-test-event-1', 'load-test-event-2', 'load-test-event-3'],
        zoneIds: [
            'load-test-zone-1-1', 'load-test-zone-1-2', 'load-test-zone-1-3',
            'load-test-zone-2-1', 'load-test-zone-2-2', 'load-test-zone-2-3',
            'load-test-zone-3-1', 'load-test-zone-3-2', 'load-test-zone-3-3'
        ],
        userIds: []
    };
}

// Direct arrays for event, show, and zone IDs
const eventIds = testDataConfig.eventIds;
const showIds = testDataConfig.showIds || [];
const zoneIds = testDataConfig.zoneIds;

// Remove old userIds (now using tokens instead)

// Get scenario from environment (default: run all)
const SCENARIO = __ENV.SCENARIO || 'all';

// All scenarios definition
const allScenarios = {
    smoke: {
        executor: 'constant-vus',
        vus: 1,
        duration: '30s',
        tags: { scenario: 'smoke' },
        exec: 'reserveSeats',
    },
    ramp_up: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 100 },
            { duration: '2m', target: 500 },
            { duration: '3m', target: 1000 },
            { duration: '2m', target: 500 },
            { duration: '1m', target: 0 },
        ],
        tags: { scenario: 'ramp_up' },
        exec: 'reserveSeats',
    },
    sustained: {
        executor: 'constant-arrival-rate',
        rate: 5000,
        timeUnit: '1s',
        duration: '5m',
        preAllocatedVUs: 2000,
        maxVUs: 5000,
        tags: { scenario: 'sustained' },
        exec: 'reserveSeats',
    },
    spike: {
        executor: 'ramping-arrival-rate',
        startRate: 1000,
        timeUnit: '1s',
        stages: [
            { duration: '30s', target: 1000 },
            { duration: '10s', target: 10000 },
            { duration: '1m', target: 10000 },
            { duration: '10s', target: 1000 },
            { duration: '1m', target: 1000 },
        ],
        preAllocatedVUs: 2000,
        maxVUs: 5000,
        tags: { scenario: 'spike' },
        exec: 'reserveSeats',
    },
    stress_10k: {
        executor: 'constant-arrival-rate',
        rate: 10000,
        timeUnit: '1s',
        duration: '5m',
        preAllocatedVUs: 2000,
        maxVUs: 5000,
        tags: { scenario: 'stress_10k' },
        exec: 'reserveSeats',
    },
};

// Select scenario based on ENV
const selectedScenarios = SCENARIO === 'all'
    ? allScenarios
    : { [SCENARIO]: allScenarios[SCENARIO] };

// Test configuration
export const options = {
    scenarios: selectedScenarios,
    thresholds: {
        'http_req_duration': ['p(95)<500', 'p(99)<1000'],
        'reserve_success_rate': ['rate>0.95'],
        'http_req_failed': ['rate<0.05'],
        'reserve_duration': ['p(95)<500', 'avg<200'],
    },
};

// Default function - reserve seats (receives data from setup)
export default function (data) {
    reserveSeats(data);
}

// Reserve seats function - receives data from setup()
export function reserveSeats(data) {
    // Use pre-generated tokens if available, otherwise use single token from setup
    let token, userId;
    if (tokens.length > 0) {
        const tokenIndex = randomIntBetween(0, tokens.length - 1);
        const tokenData = tokens[tokenIndex];
        token = tokenData.token;
        userId = tokenData.user_id;
    } else {
        token = data?.token || AUTH_TOKEN;
        userId = `a0000000-0000-0000-0000-000000000001`;
    }

    const eventId = randomItem(eventIds);
    const showId = randomItem(showIds);
    const zoneId = randomItem(zoneIds);
    const quantity = randomIntBetween(1, 4);
    const idempotencyKey = `${userId}-${zoneId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const payload = JSON.stringify({
        event_id: eventId,
        show_id: showId,
        zone_id: zoneId,
        quantity: quantity,
        unit_price: 100.00,
    });

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        'X-Idempotency-Key': idempotencyKey,
    };

    // When bypassing gateway, add X-User-ID and X-Tenant-ID headers directly
    if (BYPASS_GATEWAY) {
        headers['X-User-ID'] = userId;
        headers['X-Tenant-ID'] = '00000000-0000-0000-0000-000000000001'; // Default tenant for load test
    }

    const params = {
        headers: headers,
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
    console.log(`Bypass Gateway: ${BYPASS_GATEWAY}`);
    console.log(`Test data: ${eventIds.length} events, ${zoneIds.length} zones, ${tokens.length} tokens`);

    // Login to get auth token if not provided via ENV
    let token = AUTH_TOKEN;
    if (!token) {
        console.log(`Logging in as ${AUTH_EMAIL}...`);
        const loginResponse = http.post(`${AUTH_URL}/auth/login`, JSON.stringify({
            email: AUTH_EMAIL,
            password: AUTH_PASSWORD,
        }), {
            headers: { 'Content-Type': 'application/json' },
            timeout: '10s',
        });

        if (loginResponse.status === 200) {
            try {
                const body = JSON.parse(loginResponse.body);
                token = body.token || body.data?.token || body.access_token;
                console.log(`Login successful, got token`);
            } catch (e) {
                console.error(`Failed to parse login response: ${e}`);
            }
        } else {
            console.error(`Login failed: ${loginResponse.status} - ${loginResponse.body}`);
        }
    }

    // Verify API is reachable
    const healthCheck = http.get('http://localhost:8080/health', {
        timeout: '10s',
    });

    if (healthCheck.status !== 200) {
        console.warn(`Health check failed: ${healthCheck.status}`);
    }

    return {
        startTime: new Date().toISOString(),
        token: token,
    };
}

export function teardown(data) {
    console.log(`Load test completed. Started at: ${data.startTime}`);
}
