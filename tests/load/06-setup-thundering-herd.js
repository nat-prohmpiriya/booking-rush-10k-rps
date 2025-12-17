// Setup test data for Thundering Herd Testing
// This script initializes Redis with test zones for rate limiting and sold out tests
//
// Run: node setup_thundering_herd.js
// Reset: node setup_thundering_herd.js reset
//
// Environment variables:
// - REDIS_HOST: Redis host (default: localhost)
// - REDIS_PORT: Redis port (default: 6379)
// - REDIS_PASSWORD: Redis password (optional)

const Redis = require('ioredis');
const fs = require('fs');

// Configuration
const REDIS_HOST = process.env.REDIS_HOST || 'localhost';
const REDIS_PORT = process.env.REDIS_PORT || 6379;
const REDIS_PASSWORD = process.env.REDIS_PASSWORD || '';

// Test data configuration
const TEST_CONFIG = {
    // Event for all thundering herd tests
    events: ['thundering-herd-event-1'],

    // Zone types for different test scenarios
    zones: {
        // Sold out zones (0 seats) - for testing fast rejection
        soldOut: [
            { id: 'sold-out-zone-1', seats: 0 },
            { id: 'sold-out-zone-2', seats: 0 },
            { id: 'sold-out-zone-3', seats: 0 },
        ],

        // Limited inventory zones (few seats) - for thundering herd test
        limited: [
            { id: 'limited-zone-1', seats: 10 },     // 100 VUs compete for 10 seats
            { id: 'limited-zone-2', seats: 5 },      // Even more contention
            { id: 'limited-zone-3', seats: 1 },      // Last seat race
        ],

        // Normal zones (many seats) - for rate limit testing
        normal: [
            { id: 'normal-zone-1', seats: 100000 },
            { id: 'normal-zone-2', seats: 100000 },
            { id: 'normal-zone-3', seats: 100000 },
        ],
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

        // Setup sold out zones
        console.log('\nSetting up Sold Out zones:');
        for (const zone of TEST_CONFIG.zones.soldOut) {
            pipeline.set(`zone:availability:${zone.id}`, zone.seats);
            console.log(`  ${zone.id}: ${zone.seats} seats`);
            zoneCount++;
        }

        // Setup limited zones
        console.log('\nSetting up Limited zones:');
        for (const zone of TEST_CONFIG.zones.limited) {
            pipeline.set(`zone:availability:${zone.id}`, zone.seats);
            console.log(`  ${zone.id}: ${zone.seats} seats`);
            zoneCount++;
        }

        // Setup normal zones
        console.log('\nSetting up Normal zones:');
        for (const zone of TEST_CONFIG.zones.normal) {
            pipeline.set(`zone:availability:${zone.id}`, zone.seats);
            console.log(`  ${zone.id}: ${zone.seats} seats`);
            zoneCount++;
        }

        console.log(`\nExecuting pipeline for ${zoneCount} zones...`);
        await pipeline.exec();
        console.log('Zone availability initialized successfully');

        // Verify critical zones
        console.log('\nVerifying zone availability:');
        for (const zone of TEST_CONFIG.zones.soldOut) {
            const seats = await redis.get(`zone:availability:${zone.id}`);
            console.log(`  ${zone.id}: ${seats} seats (expected: ${zone.seats})`);
        }
        for (const zone of TEST_CONFIG.zones.limited) {
            const seats = await redis.get(`zone:availability:${zone.id}`);
            console.log(`  ${zone.id}: ${seats} seats (expected: ${zone.seats})`);
        }

        // Generate test data file for k6
        const testData = {
            events: TEST_CONFIG.events,
            zones: {
                soldOut: TEST_CONFIG.zones.soldOut.map(z => z.id),
                limited: TEST_CONFIG.zones.limited.map(z => z.id),
                normal: TEST_CONFIG.zones.normal.map(z => z.id),
            },
            seats: {
                soldOut: TEST_CONFIG.zones.soldOut.map(z => ({ id: z.id, seats: z.seats })),
                limited: TEST_CONFIG.zones.limited.map(z => ({ id: z.id, seats: z.seats })),
                normal: TEST_CONFIG.zones.normal.map(z => ({ id: z.id, seats: z.seats })),
            },
        };

        fs.writeFileSync('./thundering_herd_seed.json', JSON.stringify(testData, null, 2));
        console.log('\nTest data written to thundering_herd_seed.json');

    } catch (error) {
        console.error('Error setting up Redis:', error);
        process.exit(1);
    } finally {
        await redis.quit();
        console.log('\nRedis connection closed');
    }
}

async function resetThunderingHerdData() {
    const redis = new Redis({
        host: REDIS_HOST,
        port: REDIS_PORT,
        password: REDIS_PASSWORD || undefined,
    });

    try {
        await redis.ping();
        console.log('Resetting thundering herd test data...');

        // Get all zone availability keys for thundering herd tests
        const patterns = [
            'zone:availability:sold-out-*',
            'zone:availability:limited-*',
            'zone:availability:normal-*',
            'zone:availability:thundering-*',
            'user:reservations:thundering-herd-*',
            'reservation:*',
        ];

        let deletedCount = 0;
        for (const pattern of patterns) {
            const keys = await redis.keys(pattern);
            if (keys.length > 0) {
                await redis.del(...keys);
                deletedCount += keys.length;
                console.log(`  Deleted ${keys.length} keys matching ${pattern}`);
            }
        }

        console.log(`Total deleted: ${deletedCount} keys`);
    } catch (error) {
        console.error('Error resetting data:', error);
    } finally {
        await redis.quit();
    }
}

// CLI handling
const command = process.argv[2];

if (command === 'reset') {
    resetThunderingHerdData().then(() => {
        console.log('\nReset complete. Run setup again to initialize test data.');
    });
} else {
    setupRedis().then(() => {
        console.log('\n=== Setup Complete ===');
        console.log('Run tests with:');
        console.log('  k6 run thundering_herd.js --env SCENARIO=rate_limit_429');
        console.log('  k6 run thundering_herd.js --env SCENARIO=sold_out_rejection');
        console.log('  k6 run thundering_herd.js --env SCENARIO=spike_20k');
        console.log('  k6 run thundering_herd.js --env SCENARIO=thundering_herd');
        console.log('  k6 run thundering_herd.js --env SCENARIO=all');
    });
}
