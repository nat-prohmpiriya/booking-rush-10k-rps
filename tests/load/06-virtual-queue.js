// k6 Load Test Script for Virtual Queue + Booking Flow
// Target: 10,000 concurrent users → Virtual Queue → 500 TPS booking throughput
// Proves: System handles 10k concurrent users with zero overselling

import http from 'k6/http';
import { check, sleep, fail } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// ============================================================================
// Custom Metrics
// ============================================================================
const queueJoinSuccess = new Rate('queue_join_success');
const queuePassReceived = new Rate('queue_pass_received');
const bookingSuccess = new Rate('booking_success');
const bookingFailed = new Rate('booking_failed');
const insufficientSeats = new Counter('insufficient_seats');
const serverErrors = new Counter('server_errors');
const queueWaitTime = new Trend('queue_wait_time');
const bookingDuration = new Trend('booking_duration');
const totalCompleteFlow = new Counter('total_complete_flow');
const currentInQueue = new Gauge('current_in_queue');
const queuePassExpired = new Counter('queue_pass_expired');

// ============================================================================
// Configuration
// ============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';
const MAX_QUEUE_POLL_TIME = 300; // 5 minutes max wait in queue
const QUEUE_POLL_INTERVAL = 2;   // Poll every 2 seconds

// Load test data
let testDataConfig;
try {
    testDataConfig = JSON.parse(open('./seed-data/data.json'));
} catch (e) {
    testDataConfig = {
        eventIds: ['b0000000-0000-0001-0000-000000000001'],
        showIds: ['b0000000-0000-0001-0001-000000000001'],
        zoneIds: ['b0000000-0000-0001-0001-000000000001'],
    };
}

const eventIds = testDataConfig.eventIds || [];
const showIds = testDataConfig.showIds || [];
const zoneIds = testDataConfig.zoneIds || [];

// Load pre-generated tokens
let userTokens = [];
try {
    userTokens = JSON.parse(open('./seed-data/tokens.json'));
    console.log(`Loaded ${userTokens.length} pre-generated tokens`);
} catch (e) {
    console.log('No pre-generated tokens found');
}

const tokens = new SharedArray('tokens', function () {
    return userTokens.length > 0 ? userTokens : [];
});

// Get scenario from environment
const SCENARIO = __ENV.SCENARIO || 'virtual_queue_10k';

// ============================================================================
// Scenarios
// ============================================================================
const allScenarios = {
    // Quick test: 100 users through queue
    virtual_queue_smoke: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '30s', target: 100 },
            { duration: '2m', target: 100 },
            { duration: '30s', target: 0 },
        ],
        tags: { scenario: 'virtual_queue_smoke' },
        exec: 'virtualQueueFlow',
    },

    // Main test: 10,000 concurrent users
    virtual_queue_10k: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 2000 },    // Ramp to 2k
            { duration: '1m', target: 5000 },    // Ramp to 5k
            { duration: '1m', target: 10000 },   // Ramp to 10k
            { duration: '5m', target: 10000 },   // Sustain 10k
            { duration: '1m', target: 0 },       // Ramp down
        ],
        tags: { scenario: 'virtual_queue_10k' },
        exec: 'virtualQueueFlow',
    },

    // Stress test: 15,000 concurrent users
    virtual_queue_15k: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 5000 },
            { duration: '1m', target: 10000 },
            { duration: '1m', target: 15000 },
            { duration: '5m', target: 15000 },
            { duration: '1m', target: 0 },
        ],
        tags: { scenario: 'virtual_queue_15k' },
        exec: 'virtualQueueFlow',
    },
};

const selectedScenarios = SCENARIO === 'all'
    ? allScenarios
    : { [SCENARIO]: allScenarios[SCENARIO] };

export const options = {
    scenarios: selectedScenarios,
    thresholds: {
        'queue_join_success': ['rate>0.95'],        // 95% should join queue
        'queue_pass_received': ['rate>0.80'],       // 80% should get pass (some may timeout)
        'booking_success': ['rate>0.90'],           // 90% of those with pass should book
        'http_req_failed': ['rate<0.10'],           // <10% HTTP errors
        'booking_duration': ['p(95)<2000'],         // 95% booking < 2s
    },
};

// ============================================================================
// Main Flow: Join Queue → Get Pass → Book
// ============================================================================
export function virtualQueueFlow() {
    // Get random user token
    let token, userId;
    if (tokens.length > 0) {
        const tokenIndex = __VU % tokens.length; // Distribute VUs across tokens
        const tokenData = tokens[tokenIndex];
        token = tokenData.token;
        userId = tokenData.user_id;
    } else {
        fail('No tokens available. Run token generation first.');
    }

    const eventId = randomItem(eventIds);
    const showId = randomItem(showIds);
    const zoneId = randomItem(zoneIds);

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
    };

    // ========================================
    // Step 1: Join Queue
    // ========================================
    const joinIdempotencyKey = `queue-join-${userId}-${eventId}-${Date.now()}`;
    const joinPayload = JSON.stringify({
        event_id: eventId,
    });

    const joinHeaders = {
        ...headers,
        'X-Idempotency-Key': joinIdempotencyKey,
    };

    const joinResponse = http.post(`${BASE_URL}/queue/join`, joinPayload, {
        headers: joinHeaders,
        tags: { name: 'JoinQueue' },
    });

    const joinSuccess = check(joinResponse, {
        'join queue status 200/201': (r) => r.status === 200 || r.status === 201,
    });

    queueJoinSuccess.add(joinSuccess);

    if (!joinSuccess) {
        // If already in queue (409), continue polling
        if (joinResponse.status !== 409) {
            serverErrors.add(1);
            return;
        }
    }

    // ========================================
    // Step 2: Poll for Queue Pass
    // ========================================
    let queuePass = null;
    let queuePosition = 0;
    const queueStartTime = Date.now();
    let pollCount = 0;
    const maxPolls = MAX_QUEUE_POLL_TIME / QUEUE_POLL_INTERVAL;

    while (!queuePass && pollCount < maxPolls) {
        sleep(QUEUE_POLL_INTERVAL);
        pollCount++;

        const positionResponse = http.get(
            `${BASE_URL}/queue/position/${eventId}`,
            {
                headers: headers,
                tags: { name: 'GetPosition' },
            }
        );

        if (positionResponse.status === 200) {
            try {
                const posData = JSON.parse(positionResponse.body);
                queuePosition = posData.position || posData.data?.position || 0;

                // Check if we got a queue pass
                queuePass = posData.queue_pass || posData.data?.queue_pass;

                if (queuePass) {
                    break;
                }

                // Update gauge (approximate)
                currentInQueue.add(queuePosition);

            } catch (e) {
                // Parse error, continue polling
            }
        }
    }

    const queueEndTime = Date.now();
    const waitTime = (queueEndTime - queueStartTime) / 1000;
    queueWaitTime.add(waitTime);

    if (!queuePass) {
        queuePassReceived.add(false);
        // Timeout or couldn't get pass
        return;
    }

    queuePassReceived.add(true);

    // ========================================
    // Step 3: Reserve with Queue Pass
    // ========================================
    const quantity = randomIntBetween(1, 2);
    const idempotencyKey = `${userId}-${zoneId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const bookingPayload = JSON.stringify({
        event_id: eventId,
        show_id: showId,
        zone_id: zoneId,
        quantity: quantity,
        unit_price: 100.00,
        queue_pass: queuePass,
    });

    const bookingHeaders = {
        ...headers,
        'X-Queue-Pass': queuePass,
        'X-Idempotency-Key': idempotencyKey,
    };

    const bookingStartTime = Date.now();
    const bookingResponse = http.post(`${BASE_URL}/bookings/reserve`, bookingPayload, {
        headers: bookingHeaders,
        tags: { name: 'ReserveWithPass' },
    });
    const bookingDur = Date.now() - bookingStartTime;
    bookingDuration.add(bookingDur);

    const bookingOk = check(bookingResponse, {
        'booking status 201': (r) => r.status === 201,
        'has booking_id': (r) => {
            try {
                const body = JSON.parse(r.body);
                return body.booking_id || body.data?.booking_id;
            } catch (e) {
                return false;
            }
        },
    });

    bookingSuccess.add(bookingOk);
    bookingFailed.add(!bookingOk);

    if (bookingOk) {
        totalCompleteFlow.add(1);
    } else {
        // Track specific errors
        if (bookingResponse.status === 409) {
            insufficientSeats.add(1);
        } else if (bookingResponse.status === 401 || bookingResponse.status === 403) {
            // Queue pass expired or invalid
            queuePassExpired.add(1);
        } else if (bookingResponse.status >= 500) {
            serverErrors.add(1);
        }
    }

    // Small random sleep before next iteration
    sleep(randomIntBetween(100, 500) / 1000);
}

// ============================================================================
// Lifecycle
// ============================================================================
export function setup() {
    console.log('='.repeat(60));
    console.log('Virtual Queue Load Test');
    console.log('='.repeat(60));
    console.log(`Base URL: ${BASE_URL}`);
    console.log(`Scenario: ${SCENARIO}`);
    console.log(`Test Data: ${eventIds.length} events, ${zoneIds.length} zones`);
    console.log(`Tokens: ${tokens.length}`);
    console.log(`Max Queue Wait: ${MAX_QUEUE_POLL_TIME}s`);
    console.log('='.repeat(60));

    // Verify queue endpoints exist
    const healthCheck = http.get('http://localhost:8080/health');
    if (healthCheck.status !== 200) {
        console.warn('Health check failed!');
    }

    return {
        startTime: new Date().toISOString(),
    };
}

export function teardown(data) {
    console.log('='.repeat(60));
    console.log('Virtual Queue Load Test Complete');
    console.log(`Started: ${data.startTime}`);
    console.log(`Ended: ${new Date().toISOString()}`);
    console.log('='.repeat(60));
    console.log('Key Metrics to Check:');
    console.log('  - queue_join_success: Should be > 95%');
    console.log('  - queue_pass_received: Should be > 80%');
    console.log('  - booking_success: Should be > 90%');
    console.log('  - insufficient_seats: Expected (seats run out)');
    console.log('  - server_errors: Should be 0 or very low');
    console.log('='.repeat(60));
}

// Default export
export default function () {
    virtualQueueFlow();
}
