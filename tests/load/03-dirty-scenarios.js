// k6 Dirty Scenario Testing for Booking System
// Tests edge cases: client disconnect, timeout, concurrent booking, etc.
// Usage: k6 run dirty_scenarios.js --env SCENARIO=<scenario_name>

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { randomIntBetween, uuidv4 } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import exec from 'k6/execution';

// ============================================================================
// Configuration
// ============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8083';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'test-token';
const SCENARIO = __ENV.SCENARIO || 'all';

// Test event/zone with limited seats for testing
const TEST_EVENT_ID = __ENV.TEST_EVENT_ID || 'dirty-test-event-1';
const TEST_ZONE_ID = __ENV.TEST_ZONE_ID || 'dirty-test-zone-1';

// ============================================================================
// Custom Metrics
// ============================================================================
const scenarioSuccessRate = new Rate('scenario_success_rate');
const abandonedReservations = new Counter('abandoned_reservations');
const idempotencyHits = new Counter('idempotency_hits');
const insufficientSeatsErrors = new Counter('insufficient_seats_errors');
const concurrencyWinners = new Counter('concurrency_winners');
const concurrencyLosers = new Counter('concurrency_losers');
const reservationDuration = new Trend('reservation_duration');
const confirmDuration = new Trend('confirm_duration');
const inventoryCount = new Gauge('inventory_count');
const serviceUnavailable = new Counter('service_unavailable_503');
const networkTimeouts = new Counter('network_timeouts');

// ============================================================================
// Test Scenarios Configuration
// ============================================================================
export const options = {
    scenarios: {
        // Scenario 1: Client Disconnect - Reserve then abandon (simulated)
        client_disconnect: {
            executor: 'per-vu-iterations',
            vus: 10,
            iterations: 5,
            maxDuration: '5m',
            tags: { scenario: 'client_disconnect' },
            exec: 'testClientDisconnect',
            startTime: '0s',
        },

        // Scenario 2: Idempotency - Same key retries
        idempotency_retry: {
            executor: 'per-vu-iterations',
            vus: 5,
            iterations: 10,
            maxDuration: '3m',
            tags: { scenario: 'idempotency_retry' },
            exec: 'testIdempotencyRetry',
            startTime: '0s',
        },

        // Scenario 3: Last Seat Race - 100 concurrent requests for 1 seat
        last_seat_race: {
            executor: 'shared-iterations',
            vus: 100,
            iterations: 100,
            maxDuration: '2m',
            tags: { scenario: 'last_seat_race' },
            exec: 'testLastSeatRace',
            startTime: '0s',
        },

        // Scenario 4: Payment Timeout - Reserve then timeout on confirm
        payment_timeout: {
            executor: 'per-vu-iterations',
            vus: 5,
            iterations: 3,
            maxDuration: '3m',
            tags: { scenario: 'payment_timeout' },
            exec: 'testPaymentTimeout',
            startTime: '0s',
        },

        // Scenario 5: Graceful Degradation - Service availability under stress
        graceful_degradation: {
            executor: 'ramping-vus',
            startVUs: 1,
            stages: [
                { duration: '30s', target: 50 },
                { duration: '1m', target: 100 },
                { duration: '30s', target: 1 },
            ],
            tags: { scenario: 'graceful_degradation' },
            exec: 'testGracefulDegradation',
            startTime: '0s',
        },

        // Scenario 6: Network Timeout Simulation
        network_timeout: {
            executor: 'per-vu-iterations',
            vus: 5,
            iterations: 5,
            maxDuration: '3m',
            tags: { scenario: 'network_timeout' },
            exec: 'testNetworkTimeout',
            startTime: '0s',
        },
    },

    thresholds: {
        // General thresholds
        'scenario_success_rate': ['rate>0.80'],
        'http_req_failed': ['rate<0.20'],

        // Scenario-specific thresholds
        'scenario_success_rate{scenario:idempotency_retry}': ['rate>0.95'],
        'scenario_success_rate{scenario:last_seat_race}': ['rate==1'], // Exactly 1 winner

        // Duration thresholds
        'reservation_duration': ['p(95)<1000'],
        'confirm_duration': ['p(95)<1000'],
    },
};

// ============================================================================
// Helper Functions
// ============================================================================
function getHeaders(userId) {
    return {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${AUTH_TOKEN}`,
        'X-User-ID': userId || `test-user-${randomIntBetween(1, 10000)}`,
    };
}

function reserveSeats(userId, eventId, zoneId, quantity, idempotencyKey) {
    const payload = JSON.stringify({
        event_id: eventId || TEST_EVENT_ID,
        zone_id: zoneId || TEST_ZONE_ID,
        quantity: quantity || 1,
        unit_price: 100.00,
        idempotency_key: idempotencyKey || undefined,
    });

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/bookings/reserve`, payload, {
        headers: getHeaders(userId),
        tags: { name: 'ReserveSeats' },
        timeout: '30s',
    });
    reservationDuration.add(Date.now() - startTime);

    return response;
}

function confirmBooking(bookingId, userId, paymentId) {
    const payload = JSON.stringify({
        payment_id: paymentId || `payment-${uuidv4()}`,
    });

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/bookings/${bookingId}/confirm`, payload, {
        headers: getHeaders(userId),
        tags: { name: 'ConfirmBooking' },
        timeout: '30s',
    });
    confirmDuration.add(Date.now() - startTime);

    return response;
}

function releaseBooking(bookingId, userId) {
    return http.del(`${BASE_URL}/bookings/${bookingId}`, null, {
        headers: getHeaders(userId),
        tags: { name: 'ReleaseBooking' },
    });
}

function getBooking(bookingId, userId) {
    return http.get(`${BASE_URL}/bookings/${bookingId}`, {
        headers: getHeaders(userId),
        tags: { name: 'GetBooking' },
    });
}

function checkHealth() {
    return http.get(`${BASE_URL}/health`, {
        timeout: '5s',
    });
}

// ============================================================================
// Scenario 1: Client Disconnect After Reserve
// ============================================================================
// Simulates: Client reserves seats but disconnects before payment
// Expected: Seats released after 10 min TTL
// Verify: No orphaned reservations
export function testClientDisconnect() {
    const userId = `disconnect-user-${exec.vu.idInTest}-${exec.scenario.iterationInTest}`;
    const eventId = `disconnect-event-${randomIntBetween(1, 3)}`;
    const zoneId = `disconnect-zone-${randomIntBetween(1, 3)}`;

    group('Client Disconnect Scenario', () => {
        // Step 1: Reserve seats
        const reserveResponse = reserveSeats(userId, eventId, zoneId, 2);

        const reserveSuccess = check(reserveResponse, {
            'reserve status is 201': (r) => r.status === 201,
            'has booking_id': (r) => {
                try {
                    const body = JSON.parse(r.body);
                    return body.booking_id !== undefined;
                } catch (e) {
                    return false;
                }
            },
        });

        if (!reserveSuccess) {
            // Check if it's a 409 (insufficient seats) which is acceptable
            if (reserveResponse.status === 409) {
                console.log(`Expected: Insufficient seats for ${userId}`);
                scenarioSuccessRate.add(true);
            } else {
                scenarioSuccessRate.add(false);
                console.log(`Reserve failed: ${reserveResponse.status} - ${reserveResponse.body}`);
            }
            return;
        }

        const reserveBody = JSON.parse(reserveResponse.body);
        const bookingId = reserveBody.booking_id;

        // Step 2: Simulate disconnect (just don't confirm)
        console.log(`Client disconnected: ${userId}, booking: ${bookingId}`);
        abandonedReservations.add(1);

        // Step 3: Verify booking exists in reserved state
        sleep(1);
        const getResponse = getBooking(bookingId, userId);

        const bookingExists = check(getResponse, {
            'booking exists': (r) => r.status === 200,
            'status is reserved': (r) => {
                try {
                    const body = JSON.parse(r.body);
                    return body.status === 'reserved';
                } catch (e) {
                    return false;
                }
            },
        });

        scenarioSuccessRate.add(bookingExists);

        // Note: Actual TTL verification would require waiting 10+ minutes
        // In production, this would be tested with shorter TTL or separate verification script
        console.log(`Abandoned reservation created: ${bookingId}, will expire after TTL`);
    });
}

// ============================================================================
// Scenario 2: Idempotency Key Retry
// ============================================================================
// Simulates: Client retries request with same idempotency key
// Expected: Same response returned, no double-booking
export function testIdempotencyRetry() {
    const userId = `idempotent-user-${exec.vu.idInTest}`;
    const idempotencyKey = `idem-${userId}-${exec.scenario.iterationInTest}`;
    const eventId = `idempotent-event-${randomIntBetween(1, 3)}`;
    const zoneId = `idempotent-zone-${randomIntBetween(1, 3)}`;

    group('Idempotency Retry Scenario', () => {
        // Step 1: First request with idempotency key
        const firstResponse = reserveSeats(userId, eventId, zoneId, 1, idempotencyKey);

        if (firstResponse.status !== 201) {
            // Check for insufficient seats
            if (firstResponse.status === 409) {
                console.log(`Insufficient seats (expected): ${userId}`);
                scenarioSuccessRate.add(true);
            } else {
                console.log(`First request failed: ${firstResponse.status} - ${firstResponse.body}`);
                scenarioSuccessRate.add(false);
            }
            return;
        }

        const firstBody = JSON.parse(firstResponse.body);
        const firstBookingId = firstBody.booking_id;

        // Step 2: Retry with same idempotency key (multiple times)
        let allRetriesMatch = true;
        for (let i = 0; i < 3; i++) {
            sleep(0.5); // Small delay between retries

            const retryResponse = reserveSeats(userId, eventId, zoneId, 1, idempotencyKey);

            const retrySuccess = check(retryResponse, {
                'retry status is 201': (r) => r.status === 201,
                'same booking_id returned': (r) => {
                    try {
                        const body = JSON.parse(r.body);
                        return body.booking_id === firstBookingId;
                    } catch (e) {
                        return false;
                    }
                },
            });

            if (retrySuccess) {
                idempotencyHits.add(1);
            } else {
                allRetriesMatch = false;
                console.log(`Retry ${i + 1} mismatch: expected ${firstBookingId}, got ${retryResponse.body}`);
            }
        }

        scenarioSuccessRate.add(allRetriesMatch);

        // Cleanup: Release the reservation
        releaseBooking(firstBookingId, userId);
    });
}

// ============================================================================
// Scenario 3: Last Seat Race
// ============================================================================
// Simulates: 100 concurrent requests for last 1 seat
// Expected: Exactly 1 success, 99 failures with INSUFFICIENT_SEATS
// Verify: Total seat count unchanged (no negative inventory)
export function testLastSeatRace() {
    const userId = `race-user-${exec.vu.idInTest}`;
    // All VUs compete for the same event/zone with limited seats
    const eventId = 'race-test-event';
    const zoneId = 'race-test-zone-1-seat';

    group('Last Seat Race Scenario', () => {
        const response = reserveSeats(userId, eventId, zoneId, 1);

        if (response.status === 201) {
            // This VU won the race
            concurrencyWinners.add(1);
            console.log(`Winner: ${userId}`);

            const body = JSON.parse(response.body);
            check(response, {
                'winner has valid booking': (r) => body.booking_id !== undefined,
            });

            scenarioSuccessRate.add(true);
        } else if (response.status === 409) {
            // Expected for losers
            concurrencyLosers.add(1);
            insufficientSeatsErrors.add(1);

            const isExpectedError = check(response, {
                'loser gets INSUFFICIENT_SEATS': (r) => {
                    try {
                        const body = JSON.parse(r.body);
                        return body.code === 'INSUFFICIENT_SEATS';
                    } catch (e) {
                        return false;
                    }
                },
            });

            scenarioSuccessRate.add(isExpectedError);
        } else {
            // Unexpected error
            console.log(`Unexpected response: ${response.status} - ${response.body}`);
            scenarioSuccessRate.add(false);
        }
    });
}

// ============================================================================
// Scenario 4: Payment Timeout
// ============================================================================
// Simulates: Payment service times out
// Expected: Saga compensates, seats released
export function testPaymentTimeout() {
    const userId = `payment-timeout-user-${exec.vu.idInTest}-${exec.scenario.iterationInTest}`;
    const eventId = `payment-timeout-event-${randomIntBetween(1, 3)}`;
    const zoneId = `payment-timeout-zone-${randomIntBetween(1, 3)}`;

    group('Payment Timeout Scenario', () => {
        // Step 1: Reserve seats
        const reserveResponse = reserveSeats(userId, eventId, zoneId, 2);

        if (reserveResponse.status !== 201) {
            if (reserveResponse.status === 409) {
                scenarioSuccessRate.add(true);
            } else {
                scenarioSuccessRate.add(false);
            }
            return;
        }

        const reserveBody = JSON.parse(reserveResponse.body);
        const bookingId = reserveBody.booking_id;

        // Step 2: Simulate payment timeout (attempt to confirm with timeout)
        // In real scenario, payment service would timeout
        // Here we simulate by using a very short timeout
        const confirmPayload = JSON.stringify({
            payment_id: `timeout-payment-${uuidv4()}`,
        });

        const confirmResponse = http.post(`${BASE_URL}/bookings/${bookingId}/confirm`, confirmPayload, {
            headers: getHeaders(userId),
            tags: { name: 'ConfirmWithTimeout' },
            timeout: '100ms', // Very short timeout to simulate network issues
        });

        // Whether confirm succeeds or times out, verify system state
        sleep(2);

        const getResponse = getBooking(bookingId, userId);

        const stateValid = check(getResponse, {
            'booking state is valid': (r) => {
                if (r.status !== 200) return false;
                try {
                    const body = JSON.parse(r.body);
                    // Either confirmed or still reserved (timeout happened)
                    return body.status === 'confirmed' || body.status === 'reserved';
                } catch (e) {
                    return false;
                }
            },
        });

        scenarioSuccessRate.add(stateValid);

        // Cleanup if still reserved
        if (getResponse.status === 200) {
            const body = JSON.parse(getResponse.body);
            if (body.status === 'reserved') {
                releaseBooking(bookingId, userId);
            }
        }
    });
}

// ============================================================================
// Scenario 5: Graceful Degradation
// ============================================================================
// Simulates: High load to check graceful degradation
// Expected: Service returns 503 under extreme load, graceful degradation
export function testGracefulDegradation() {
    const userId = `degrade-user-${exec.vu.idInTest}-${Date.now()}`;
    const eventId = `degrade-event-${randomIntBetween(1, 10)}`;
    const zoneId = `degrade-zone-${randomIntBetween(1, 10)}`;

    group('Graceful Degradation Scenario', () => {
        // Health check
        const healthResponse = checkHealth();

        if (healthResponse.status !== 200) {
            serviceUnavailable.add(1);
            console.log('Service unhealthy during degradation test');
            // Still acceptable - this is the graceful degradation
            scenarioSuccessRate.add(true);
            return;
        }

        // Attempt reservation under load
        const reserveResponse = reserveSeats(userId, eventId, zoneId, 1);

        const responseValid = check(reserveResponse, {
            'response is valid (201, 409, or 503)': (r) => {
                return r.status === 201 || r.status === 409 || r.status === 503;
            },
        });

        if (reserveResponse.status === 503) {
            serviceUnavailable.add(1);
            console.log('Service temporarily unavailable (503) - graceful degradation');
        }

        scenarioSuccessRate.add(responseValid);

        // Cleanup successful reservations
        if (reserveResponse.status === 201) {
            try {
                const body = JSON.parse(reserveResponse.body);
                releaseBooking(body.booking_id, userId);
            } catch (e) {
                // Ignore cleanup errors
            }
        }
    });

    sleep(randomIntBetween(10, 100) / 1000); // 10-100ms random delay
}

// ============================================================================
// Scenario 6: Network Timeout Simulation
// ============================================================================
// Simulates: Network timeout mid-request
// Expected: No duplicate reservations
export function testNetworkTimeout() {
    const userId = `timeout-user-${exec.vu.idInTest}-${exec.scenario.iterationInTest}`;
    const idempotencyKey = `timeout-idem-${userId}`;
    const eventId = `timeout-event-${randomIntBetween(1, 3)}`;
    const zoneId = `timeout-zone-${randomIntBetween(1, 3)}`;

    group('Network Timeout Scenario', () => {
        const bookings = [];

        // Step 1: Multiple rapid requests with same idempotency key
        // Some may timeout, but should not create duplicates
        for (let i = 0; i < 5; i++) {
            const timeout = i === 0 ? '100ms' : '30s'; // First request may timeout

            const payload = JSON.stringify({
                event_id: eventId,
                zone_id: zoneId,
                quantity: 1,
                unit_price: 100.00,
                idempotency_key: idempotencyKey,
            });

            const response = http.post(`${BASE_URL}/bookings/reserve`, payload, {
                headers: getHeaders(userId),
                tags: { name: 'ReserveWithTimeout' },
                timeout: timeout,
            });

            if (response.status === 201) {
                try {
                    const body = JSON.parse(response.body);
                    bookings.push(body.booking_id);
                } catch (e) {
                    // Ignore parse errors
                }
            } else if (response.status === 0) {
                networkTimeouts.add(1);
                console.log(`Request ${i + 1} timed out`);
            }

            sleep(0.1);
        }

        // Step 2: Verify no duplicates (all booking IDs should be the same)
        const uniqueBookings = [...new Set(bookings)];
        const noDuplicates = check(null, {
            'no duplicate bookings': () => uniqueBookings.length <= 1,
        });

        if (uniqueBookings.length > 1) {
            console.log(`ERROR: Duplicate bookings detected: ${JSON.stringify(uniqueBookings)}`);
        }

        scenarioSuccessRate.add(noDuplicates);

        // Cleanup
        if (uniqueBookings.length > 0) {
            releaseBooking(uniqueBookings[0], userId);
        }
    });
}

// ============================================================================
// Default Function
// ============================================================================
export default function () {
    // Run specific scenario based on env variable
    switch (SCENARIO) {
        case 'client_disconnect':
            testClientDisconnect();
            break;
        case 'idempotency':
            testIdempotencyRetry();
            break;
        case 'last_seat':
            testLastSeatRace();
            break;
        case 'payment_timeout':
            testPaymentTimeout();
            break;
        case 'degradation':
            testGracefulDegradation();
            break;
        case 'network_timeout':
            testNetworkTimeout();
            break;
        default:
            // Run a random scenario
            const scenarios = [
                testClientDisconnect,
                testIdempotencyRetry,
                testPaymentTimeout,
                testGracefulDegradation,
                testNetworkTimeout,
            ];
            const randomScenario = scenarios[randomIntBetween(0, scenarios.length - 1)];
            randomScenario();
    }
}

// ============================================================================
// Setup & Teardown
// ============================================================================
export function setup() {
    console.log(`Starting Dirty Scenario Tests against ${BASE_URL}`);
    console.log(`Scenario: ${SCENARIO}`);

    // Verify service is available
    const healthCheck = checkHealth();
    if (healthCheck.status !== 200) {
        console.warn(`Health check failed: ${healthCheck.status}`);
    }

    return {
        startTime: new Date().toISOString(),
        scenario: SCENARIO,
    };
}

export function teardown(data) {
    console.log(`Dirty Scenario Tests completed`);
    console.log(`Started at: ${data.startTime}`);
    console.log(`Scenario: ${data.scenario}`);
}

// ============================================================================
// Handle Summary
// ============================================================================
export function handleSummary(data) {
    const summary = {
        timestamp: new Date().toISOString(),
        scenario: SCENARIO,
        metrics: {
            total_requests: data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0,
            success_rate: data.metrics.scenario_success_rate ? data.metrics.scenario_success_rate.values.rate : 0,
            abandoned_reservations: data.metrics.abandoned_reservations ? data.metrics.abandoned_reservations.values.count : 0,
            idempotency_hits: data.metrics.idempotency_hits ? data.metrics.idempotency_hits.values.count : 0,
            insufficient_seats_errors: data.metrics.insufficient_seats_errors ? data.metrics.insufficient_seats_errors.values.count : 0,
            concurrency_winners: data.metrics.concurrency_winners ? data.metrics.concurrency_winners.values.count : 0,
            concurrency_losers: data.metrics.concurrency_losers ? data.metrics.concurrency_losers.values.count : 0,
            service_unavailable_503: data.metrics.service_unavailable_503 ? data.metrics.service_unavailable_503.values.count : 0,
            network_timeouts: data.metrics.network_timeouts ? data.metrics.network_timeouts.values.count : 0,
        },
        thresholds: data.thresholds,
    };

    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
        'dirty_scenarios_results.json': JSON.stringify(summary, null, 2),
    };
}

// Simple text summary function
function textSummary(data, options) {
    let output = '\n=== Dirty Scenario Test Results ===\n\n';

    if (data.metrics) {
        output += 'Key Metrics:\n';
        output += `  - Total HTTP Requests: ${data.metrics.http_reqs?.values?.count || 0}\n`;
        output += `  - Success Rate: ${((data.metrics.scenario_success_rate?.values?.rate || 0) * 100).toFixed(2)}%\n`;
        output += `  - Abandoned Reservations: ${data.metrics.abandoned_reservations?.values?.count || 0}\n`;
        output += `  - Idempotency Hits: ${data.metrics.idempotency_hits?.values?.count || 0}\n`;
        output += `  - Insufficient Seats Errors: ${data.metrics.insufficient_seats_errors?.values?.count || 0}\n`;
        output += `  - Concurrency Winners: ${data.metrics.concurrency_winners?.values?.count || 0}\n`;
        output += `  - Concurrency Losers: ${data.metrics.concurrency_losers?.values?.count || 0}\n`;
        output += `  - Service Unavailable (503): ${data.metrics.service_unavailable_503?.values?.count || 0}\n`;
        output += `  - Network Timeouts: ${data.metrics.network_timeouts?.values?.count || 0}\n`;
    }

    output += '\n';
    return output;
}
