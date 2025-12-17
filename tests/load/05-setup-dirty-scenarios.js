// Setup test data for Dirty Scenario Testing
// This script initializes Redis with test zones and their availability
// Run: node setup_dirty_scenarios.js
//
// Environment variables:
// - REDIS_HOST: Redis host (default: localhost)
// - REDIS_PORT: Redis port (default: 6379)
// - REDIS_PASSWORD: Redis password (optional)

const Redis = require('ioredis');

// Configuration
const REDIS_HOST = process.env.REDIS_HOST || 'localhost';
const REDIS_PORT = process.env.REDIS_PORT || 6379;
const REDIS_PASSWORD = process.env.REDIS_PASSWORD || '';

// Test data configuration
const TEST_CONFIG = {
    // Scenario 1: Client Disconnect test zones (plenty of seats)
    disconnect: {
        events: ['disconnect-event-1', 'disconnect-event-2', 'disconnect-event-3'],
        zonesPerEvent: 3,
        seatsPerZone: 1000,
    },

    // Scenario 2: Idempotency test zones (plenty of seats)
    idempotent: {
        events: ['idempotent-event-1', 'idempotent-event-2', 'idempotent-event-3'],
        zonesPerEvent: 3,
        seatsPerZone: 1000,
    },

    // Scenario 3: Last Seat Race (single zone with exactly 1 seat)
    lastSeatRace: {
        events: ['race-test-event'],
        zones: ['race-test-zone-1-seat'],
        seatsPerZone: 1, // Only 1 seat for race condition test
    },

    // Scenario 4: Payment Timeout test zones
    paymentTimeout: {
        events: ['payment-timeout-event-1', 'payment-timeout-event-2', 'payment-timeout-event-3'],
        zonesPerEvent: 3,
        seatsPerZone: 500,
    },

    // Scenario 5: Graceful Degradation test zones
    degradation: {
        events: Array.from({ length: 10 }, (_, i) => `degrade-event-${i + 1}`),
        zonesPerEvent: 10,
        seatsPerZone: 100,
    },

    // Scenario 6: Network Timeout test zones
    networkTimeout: {
        events: ['timeout-event-1', 'timeout-event-2', 'timeout-event-3'],
        zonesPerEvent: 3,
        seatsPerZone: 500,
    },

    // General dirty test zones
    dirty: {
        events: ['dirty-test-event-1', 'dirty-test-event-2', 'dirty-test-event-3'],
        zonesPerEvent: 3,
        seatsPerZone: 500,
    },
};

async function setupRedis() {
    const redis = new Redis({
        host: REDIS_HOST,
        port: REDIS_PORT,
        password: REDIS_PASSWORD || undefined,
        retryDelayOnFailover: 100,
        maxRetriesPerRequest: 3,
    });

    console.log(`Connecting to Redis at ${REDIS_HOST}:${REDIS_PORT}...`);

    try {
        await redis.ping();
        console.log('Connected to Redis successfully');

        const pipeline = redis.pipeline();
        let zoneCount = 0;

        // Setup Disconnect scenario zones
        for (const event of TEST_CONFIG.disconnect.events) {
            for (let z = 1; z <= TEST_CONFIG.disconnect.zonesPerEvent; z++) {
                const zoneId = `disconnect-zone-${z}`;
                pipeline.set(`zone:availability:${zoneId}`, TEST_CONFIG.disconnect.seatsPerZone);
                zoneCount++;
            }
        }

        // Setup Idempotency scenario zones
        for (const event of TEST_CONFIG.idempotent.events) {
            for (let z = 1; z <= TEST_CONFIG.idempotent.zonesPerEvent; z++) {
                const zoneId = `idempotent-zone-${z}`;
                pipeline.set(`zone:availability:${zoneId}`, TEST_CONFIG.idempotent.seatsPerZone);
                zoneCount++;
            }
        }

        // Setup Last Seat Race zone (critical - only 1 seat!)
        for (const zone of TEST_CONFIG.lastSeatRace.zones) {
            pipeline.set(`zone:availability:${zone}`, TEST_CONFIG.lastSeatRace.seatsPerZone);
            zoneCount++;
            console.log(`  Race zone: ${zone} with ${TEST_CONFIG.lastSeatRace.seatsPerZone} seat(s)`);
        }

        // Setup Payment Timeout scenario zones
        for (const event of TEST_CONFIG.paymentTimeout.events) {
            for (let z = 1; z <= TEST_CONFIG.paymentTimeout.zonesPerEvent; z++) {
                const zoneId = `payment-timeout-zone-${z}`;
                pipeline.set(`zone:availability:${zoneId}`, TEST_CONFIG.paymentTimeout.seatsPerZone);
                zoneCount++;
            }
        }

        // Setup Graceful Degradation scenario zones
        for (let e = 1; e <= TEST_CONFIG.degradation.events.length; e++) {
            for (let z = 1; z <= TEST_CONFIG.degradation.zonesPerEvent; z++) {
                const zoneId = `degrade-zone-${z}`;
                // Only set once per zone (zones are shared across events)
                if (e === 1) {
                    pipeline.set(`zone:availability:${zoneId}`, TEST_CONFIG.degradation.seatsPerZone);
                    zoneCount++;
                }
            }
        }

        // Setup Network Timeout scenario zones
        for (const event of TEST_CONFIG.networkTimeout.events) {
            for (let z = 1; z <= TEST_CONFIG.networkTimeout.zonesPerEvent; z++) {
                const zoneId = `timeout-zone-${z}`;
                pipeline.set(`zone:availability:${zoneId}`, TEST_CONFIG.networkTimeout.seatsPerZone);
                zoneCount++;
            }
        }

        // Setup Dirty test zones
        for (const event of TEST_CONFIG.dirty.events) {
            for (let z = 1; z <= TEST_CONFIG.dirty.zonesPerEvent; z++) {
                const zoneId = `dirty-test-zone-${z}`;
                pipeline.set(`zone:availability:${zoneId}`, TEST_CONFIG.dirty.seatsPerZone);
                zoneCount++;
            }
        }

        console.log(`Setting up ${zoneCount} zones in Redis...`);
        await pipeline.exec();
        console.log('Zone availability initialized successfully');

        // Verify critical zones
        console.log('\nVerifying critical zones:');
        for (const zone of TEST_CONFIG.lastSeatRace.zones) {
            const seats = await redis.get(`zone:availability:${zone}`);
            console.log(`  ${zone}: ${seats} seat(s)`);
        }

        // Output test data for k6
        const testData = {
            events: {
                disconnect: TEST_CONFIG.disconnect.events,
                idempotent: TEST_CONFIG.idempotent.events,
                race: TEST_CONFIG.lastSeatRace.events,
                paymentTimeout: TEST_CONFIG.paymentTimeout.events,
                degradation: TEST_CONFIG.degradation.events,
                networkTimeout: TEST_CONFIG.networkTimeout.events,
                dirty: TEST_CONFIG.dirty.events,
            },
            zones: {
                disconnect: Array.from({ length: TEST_CONFIG.disconnect.zonesPerEvent }, (_, i) => `disconnect-zone-${i + 1}`),
                idempotent: Array.from({ length: TEST_CONFIG.idempotent.zonesPerEvent }, (_, i) => `idempotent-zone-${i + 1}`),
                race: TEST_CONFIG.lastSeatRace.zones,
                paymentTimeout: Array.from({ length: TEST_CONFIG.paymentTimeout.zonesPerEvent }, (_, i) => `payment-timeout-zone-${i + 1}`),
                degradation: Array.from({ length: TEST_CONFIG.degradation.zonesPerEvent }, (_, i) => `degrade-zone-${i + 1}`),
                networkTimeout: Array.from({ length: TEST_CONFIG.networkTimeout.zonesPerEvent }, (_, i) => `timeout-zone-${i + 1}`),
                dirty: Array.from({ length: TEST_CONFIG.dirty.zonesPerEvent }, (_, i) => `dirty-test-zone-${i + 1}`),
            },
            seats: {
                disconnect: TEST_CONFIG.disconnect.seatsPerZone,
                idempotent: TEST_CONFIG.idempotent.seatsPerZone,
                race: TEST_CONFIG.lastSeatRace.seatsPerZone,
                paymentTimeout: TEST_CONFIG.paymentTimeout.seatsPerZone,
                degradation: TEST_CONFIG.degradation.seatsPerZone,
                networkTimeout: TEST_CONFIG.networkTimeout.seatsPerZone,
                dirty: TEST_CONFIG.dirty.seatsPerZone,
            },
        };

        // Write test data to file
        const fs = require('fs');
        fs.writeFileSync('./dirty_scenarios_seed.json', JSON.stringify(testData, null, 2));
        console.log('\nTest data written to dirty_scenarios_seed.json');

    } catch (error) {
        console.error('Error setting up Redis:', error);
        process.exit(1);
    } finally {
        await redis.quit();
        console.log('\nRedis connection closed');
    }
}

// Reset function to clear all dirty test data
async function resetDirtyTestData() {
    const redis = new Redis({
        host: REDIS_HOST,
        port: REDIS_PORT,
        password: REDIS_PASSWORD || undefined,
    });

    try {
        await redis.ping();
        console.log('Resetting dirty test data...');

        // Get all zone availability keys for dirty tests
        const patterns = [
            'zone:availability:disconnect-*',
            'zone:availability:idempotent-*',
            'zone:availability:race-*',
            'zone:availability:payment-timeout-*',
            'zone:availability:degrade-*',
            'zone:availability:timeout-*',
            'zone:availability:dirty-*',
            'user:reservations:disconnect-*',
            'user:reservations:idempotent-*',
            'user:reservations:race-*',
            'user:reservations:payment-timeout-*',
            'user:reservations:degrade-*',
            'user:reservations:timeout-*',
            'user:reservations:dirty-*',
            'reservation:*',
        ];

        let deletedCount = 0;
        for (const pattern of patterns) {
            const keys = await redis.keys(pattern);
            if (keys.length > 0) {
                await redis.del(...keys);
                deletedCount += keys.length;
            }
        }

        console.log(`Deleted ${deletedCount} keys`);
    } catch (error) {
        console.error('Error resetting data:', error);
    } finally {
        await redis.quit();
    }
}

// CLI handling
const command = process.argv[2];

if (command === 'reset') {
    resetDirtyTestData().then(() => {
        console.log('Reset complete. Run setup again to initialize test data.');
    });
} else {
    setupRedis().then(() => {
        console.log('\nSetup complete. Run dirty_scenarios.js with k6 to test.');
        console.log('Example: k6 run dirty_scenarios.js --env SCENARIO=last_seat_race');
    });
}
