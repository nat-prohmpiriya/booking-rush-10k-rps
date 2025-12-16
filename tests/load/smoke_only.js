import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const reserveSuccessRate = new Rate('reserve_success_rate');
const reserveDuration = new Trend('reserve_duration');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';
const AUTH_TOKEN = __ENV.AUTH_TOKEN;

let testDataConfig;
try {
    testDataConfig = JSON.parse(open('./seed_data.json'));
} catch (e) {
    testDataConfig = { eventIds: [], showIds: [], zoneIds: [] };
}

export const options = {
    scenarios: {
        smoke: {
            executor: 'constant-vus',
            vus: 10,
            duration: '30s',
        },
    },
    thresholds: {
        'http_req_duration': ['p(95)<500'],
        'reserve_success_rate': ['rate>0.90'],
    },
};

export default function() {
    const eventId = randomItem(testDataConfig.eventIds);
    const showId = randomItem(testDataConfig.showIds);
    const zoneId = randomItem(testDataConfig.zoneIds);
    const quantity = randomIntBetween(1, 2);
    const idempotencyKey = `smoke-${__VU}-${__ITER}-${Date.now()}`;

    const payload = JSON.stringify({
        event_id: eventId,
        show_id: showId,
        zone_id: zoneId,
        quantity: quantity,
        unit_price: 100.00,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${AUTH_TOKEN}`,
            'X-Idempotency-Key': idempotencyKey,
        },
    };

    const startTime = Date.now();
    const response = http.post(`${BASE_URL}/bookings/reserve`, payload, params);
    const duration = Date.now() - startTime;

    reserveDuration.add(duration);

    const success = check(response, {
        'status is 201': (r) => r.status === 201,
        'response time OK': (r) => r.timings.duration < 1000,
    });

    reserveSuccessRate.add(success);
    
    if (!success && response.status !== 409) {
        console.log(`Error: ${response.status} - ${response.body}`);
    }

    sleep(0.01);
}
