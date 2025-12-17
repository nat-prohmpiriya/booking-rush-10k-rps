// Generate JWT tokens for load test users
// Usage: NUM_TOKENS=500 node generate-tokens.js

const fs = require('fs');

const BASE_URL = process.env.BASE_URL || 'http://localhost:8080/api/v1';
const NUM_TOKENS = parseInt(process.env.NUM_TOKENS) || 1000;
const BATCH_SIZE = parseInt(process.env.BATCH_SIZE) || 5;
const OUTPUT_FILE = './tokens.json';

async function login(email, password) {
    const response = await fetch(`${BASE_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
    });

    if (!response.ok) {
        throw new Error(`Login failed for ${email}: ${response.status}`);
    }

    const data = await response.json();
    return data.data?.access_token || data.access_token;
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

async function loginWithRetry(i, password, maxRetries = 2) {
    const email = `loadtest${i}@test.com`;
    const userId = `a0000000-0000-0000-0000-${String(i).padStart(12, '0')}`;

    for (let attempt = 1; attempt <= maxRetries; attempt++) {
        try {
            const token = await login(email, password);
            return { user_id: userId, email, token };
        } catch (err) {
            if (attempt < maxRetries && err.message.includes('429')) {
                await sleep(1000 * attempt); // Exponential backoff
            } else if (attempt === maxRetries) {
                console.error(`\nFailed: ${email} - ${err.message}`);
                return null;
            }
        }
    }
    return null;
}

async function main() {
    console.log(`Generating ${NUM_TOKENS} tokens (batch size: ${BATCH_SIZE})...`);
    console.log(`API: ${BASE_URL}`);

    const tokens = [];
    const password = 'Test123!';
    const startTime = Date.now();

    // Process in batches
    for (let batchStart = 1; batchStart <= NUM_TOKENS; batchStart += BATCH_SIZE) {
        const batchEnd = Math.min(batchStart + BATCH_SIZE - 1, NUM_TOKENS);
        const batchPromises = [];

        // Create batch of login promises
        for (let i = batchStart; i <= batchEnd; i++) {
            batchPromises.push(loginWithRetry(i, password));
        }

        // Execute batch in parallel
        const results = await Promise.all(batchPromises);

        // Collect successful results
        for (const result of results) {
            if (result) {
                tokens.push(result);
            }
        }

        process.stdout.write(`\r${tokens.length}/${NUM_TOKENS} tokens generated`);

        // Delay between batches to avoid rate limiting
        if (batchEnd < NUM_TOKENS) {
            await sleep(200);
        }
    }

    const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
    console.log(`\n\nSaving to ${OUTPUT_FILE}...`);
    fs.writeFileSync(OUTPUT_FILE, JSON.stringify(tokens, null, 2));
    console.log(`Done! Generated ${tokens.length} tokens in ${elapsed}s`);
}

main().catch(console.error);
