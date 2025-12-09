// Generate test data JSON for k6 load testing
// Run: node generate_test_data.js > seed_data.json

const fs = require('fs');

const NUM_USERS = 10000;
const NUM_EVENTS = 3;
const SHOWS_PER_EVENT = 3;
const ZONES_PER_SHOW = 5;

// Generate user IDs
const userIds = Array.from({ length: NUM_USERS }, (_, i) => `load-test-user-${i + 1}`);

// Generate event IDs
const eventIds = Array.from({ length: NUM_EVENTS }, (_, i) => `load-test-event-${i + 1}`);

// Generate show IDs
const showIds = [];
for (let e = 1; e <= NUM_EVENTS; e++) {
    for (let s = 1; s <= SHOWS_PER_EVENT; s++) {
        showIds.push(`load-test-show-${e}-${s}`);
    }
}

// Generate zone IDs (simplified for load test)
const zoneIds = [];
for (let s = 1; s <= SHOWS_PER_EVENT; s++) {
    for (let z = 1; z <= ZONES_PER_SHOW; z++) {
        zoneIds.push(`load-test-zone-${s}-${z}`);
    }
}

const testData = {
    eventIds,
    showIds,
    zoneIds,
    userIds
};

// Output to stdout or file
const output = JSON.stringify(testData, null, 2);

if (process.argv[2]) {
    fs.writeFileSync(process.argv[2], output);
    console.error(`Written to ${process.argv[2]}`);
} else {
    console.log(output);
}
