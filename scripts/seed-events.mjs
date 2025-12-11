#!/usr/bin/env node

/**
 * Seed Events Script
 * Creates 10 events with shows and zones via API
 * Mixed statuses: OPEN, UPCOMING, and ENDED events
 */

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080/api/v1';
const ADMIN_EMAIL = process.env.ADMIN_EMAIL || 'test1@test.com';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || '#Ttest1234';

let ACCESS_TOKEN = '';

// Colors for console
const colors = {
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  reset: '\x1b[0m'
};

function log(color, message) {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

async function login() {
  log('yellow', '[Step 1] Logging in...');

  const res = await fetch(`${API_BASE_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD })
  });

  const data = await res.json();

  if (!data.success || !data.data?.access_token) {
    log('red', `Login failed: ${JSON.stringify(data)}`);
    process.exit(1);
  }

  ACCESS_TOKEN = data.data.access_token;
  log('green', `Login successful! Role: ${data.data.user.role}`);
}

async function createEvent(eventData) {
  const res = await fetch(`${API_BASE_URL}/events`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ACCESS_TOKEN}`
    },
    body: JSON.stringify(eventData)
  });

  const data = await res.json();

  if (!data.success) {
    log('red', `  Failed to create event: ${JSON.stringify(data.error)}`);
    return null;
  }

  log('green', `  Event created: ${data.data.id}`);
  return data.data.id;
}

async function createShow(eventId, showData) {
  const res = await fetch(`${API_BASE_URL}/events/${eventId}/shows`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ACCESS_TOKEN}`
    },
    body: JSON.stringify(showData)
  });

  const data = await res.json();

  if (!data.success) {
    log('red', `  Failed to create show: ${JSON.stringify(data.error)}`);
    return null;
  }

  log('green', `  Show created: ${showData.name}`);
  return data.data.id;
}

async function createZone(showId, zoneData) {
  const res = await fetch(`${API_BASE_URL}/shows/${showId}/zones`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${ACCESS_TOKEN}`
    },
    body: JSON.stringify(zoneData)
  });

  const data = await res.json();

  if (!data.success) {
    log('red', `  Failed to create zone: ${JSON.stringify(data.error)}`);
    return null;
  }

  return data.data.id;
}

async function publishEvent(eventId) {
  const res = await fetch(`${API_BASE_URL}/events/${eventId}/publish`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${ACCESS_TOKEN}` }
  });

  const data = await res.json();

  if (!data.success) {
    log('red', `  Failed to publish: ${JSON.stringify(data.error)}`);
    return false;
  }

  log('green', `  Published!`);
  return true;
}

// Event definitions
const events = [
  {
    status: 'OPEN',
    event: {
      name: 'BTS World Tour: Love Yourself',
      description: 'Experience the global phenomenon BTS live in Bangkok! Join millions of ARMYs for an unforgettable night of music, dance, and connection.',
      short_description: 'BTS Live in Bangkok - World Tour 2025',
      venue_name: 'Rajamangala National Stadium',
      venue_address: '286 Ramkhamhaeng Rd, Hua Mak',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1540039155733-5bb30b53aa14?w=800',
      banner_url: 'https://images.unsplash.com/photo-1459749411175-04bf5292ceea?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-12-01T10:00:00+07:00',
      booking_end_at: '2025-12-30T23:59:59+07:00'
    },
    shows: [
      { name: 'Night 1', show_date: '2025-12-31', start_time: '19:00:00+07:00', end_time: '23:00:00+07:00' },
      { name: 'Night 2', show_date: '2026-01-01', start_time: '19:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'VVIP Standing', price: 12000, total_seats: 500, description: 'Front stage standing area' },
      { name: 'VIP Standing', price: 8500, total_seats: 1000, description: 'Premium standing area' },
      { name: 'Gold Seated', price: 5500, total_seats: 2000, description: 'Reserved seating lower bowl' },
      { name: 'Silver Seated', price: 3500, total_seats: 3000, description: 'Reserved seating upper bowl' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Ed Sheeran Mathematics Tour',
      description: 'Grammy Award winner Ed Sheeran brings his Mathematics Tour to Bangkok.',
      short_description: 'Ed Sheeran Live - Mathematics Tour Bangkok',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=800',
      banner_url: 'https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-12-20T10:00:00+07:00',
      booking_end_at: '2026-01-28T23:59:59+07:00'
    },
    shows: [
      { name: 'Bangkok Show', show_date: '2026-01-29', start_time: '20:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'CAT 1', price: 8900, total_seats: 800, description: 'Best seats - center stage' },
      { name: 'CAT 2', price: 6500, total_seats: 1500, description: 'Premium side stage' },
      { name: 'CAT 3', price: 4500, total_seats: 2000, description: 'Great value seating' },
      { name: 'CAT 4', price: 2500, total_seats: 2500, description: 'Budget-friendly seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Coldplay Music of the Spheres',
      description: 'Coldplay returns to Bangkok with their spectacular Music of the Spheres World Tour.',
      short_description: 'Coldplay World Tour 2026 - Bangkok',
      venue_name: 'Rajamangala National Stadium',
      venue_address: '286 Ramkhamhaeng Rd, Hua Mak',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1470229722913-7c0e2dbbafd3?w=800',
      banner_url: 'https://images.unsplash.com/photo-1429962714451-bb934ecdc4ec?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-11-15T10:00:00+07:00',
      booking_end_at: '2026-02-13T23:59:59+07:00'
    },
    shows: [
      { name: 'Bangkok Show', show_date: '2026-02-14', start_time: '19:30:00+07:00', end_time: '22:30:00+07:00' }
    ],
    zones: [
      { name: 'Infinity Ticket', price: 9500, total_seats: 1000, description: 'GA Standing closest to stage' },
      { name: 'A Reserve', price: 7500, total_seats: 2000, description: 'Premium reserved seating' },
      { name: 'B Reserve', price: 5500, total_seats: 3000, description: 'Standard reserved seating' },
      { name: 'C Reserve', price: 3500, total_seats: 4000, description: 'Value seating' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Summer Sonic Bangkok',
      description: 'Thailand premier summer music festival featuring top international and local artists.',
      short_description: 'Summer Sonic Festival 2025 - Bangkok Edition',
      venue_name: 'BITEC Bangna',
      venue_address: '88 Bangna-Trad Road',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1533174072545-7a4b6ad7a6c3?w=800',
      banner_url: 'https://images.unsplash.com/photo-1506157786151-b8491531f063?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-10-01T10:00:00+07:00',
      booking_end_at: '2025-11-14T23:59:59+07:00'
    },
    shows: [
      { name: 'Festival Day', show_date: '2025-11-15', start_time: '12:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP', price: 5500, total_seats: 500, description: 'VIP viewing area' },
      { name: 'General Admission', price: 2500, total_seats: 5000, description: 'Full festival access' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Muay Thai Super Fight Night',
      description: 'Witness the best Muay Thai fighters from around the world compete.',
      short_description: 'World Championship Muay Thai - Bangkok',
      venue_name: 'Lumpinee Boxing Stadium',
      venue_address: '6 Ramintra Rd, Anusawari',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1549719386-74dfcbf7dbed?w=800',
      banner_url: 'https://images.unsplash.com/photo-1544367567-0f2fcb009e0b?w=1600',
      max_tickets_per_user: 6,
      booking_start_at: '2025-11-01T10:00:00+07:00',
      booking_end_at: '2026-01-08T23:59:59+07:00'
    },
    shows: [
      { name: 'Championship Night', show_date: '2026-01-09', start_time: '18:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'Ringside', price: 5000, total_seats: 100, description: 'Ringside seats' },
      { name: 'VIP', price: 3000, total_seats: 200, description: 'VIP seating' },
      { name: 'Standard', price: 1500, total_seats: 500, description: 'Standard seating' },
      { name: 'Standing', price: 800, total_seats: 300, description: 'Standing area' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Bangkok International Jazz Festival',
      description: 'Three days of world-class jazz featuring international and local artists.',
      short_description: 'Bangkok Jazz Festival 2026',
      venue_name: 'Lumpini Park',
      venue_address: 'Rama IV Road',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1415201364774-f6f0bb35f28f?w=800',
      banner_url: 'https://images.unsplash.com/photo-1511192336575-5a79af67a629?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-12-25T10:00:00+07:00',
      booking_end_at: '2026-01-18T23:59:59+07:00'
    },
    shows: [
      { name: 'Day 1 - Opening Night', show_date: '2026-01-19', start_time: '17:00:00+07:00', end_time: '23:00:00+07:00' },
      { name: 'Day 2 - Main Event', show_date: '2026-01-20', start_time: '17:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Table', price: 3500, total_seats: 50, description: 'Reserved table for 4' },
      { name: 'Premium GA', price: 1800, total_seats: 500, description: 'Premium area' },
      { name: 'General Admission', price: 900, total_seats: 1000, description: 'Standard access' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Royal Bangkok Symphony Orchestra',
      description: 'An evening of classical masterpieces performed by the Royal Bangkok Symphony Orchestra.',
      short_description: 'Classical Night - Beethoven and Tchaikovsky',
      venue_name: 'Thailand Cultural Centre',
      venue_address: '14 Thiam Ruam Mit Rd, Huai Khwang',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1465847899084-d164df4dedc6?w=800',
      banner_url: 'https://images.unsplash.com/photo-1507838153414-b4b713384a76?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-11-01T10:00:00+07:00',
      booking_end_at: '2025-12-29T23:59:59+07:00'
    },
    shows: [
      { name: 'Evening Performance', show_date: '2025-12-30', start_time: '19:30:00+07:00', end_time: '22:00:00+07:00' }
    ],
    zones: [
      { name: 'Orchestra', price: 2500, total_seats: 200, description: 'Best acoustic experience' },
      { name: 'Mezzanine', price: 1800, total_seats: 300, description: 'Elevated view' },
      { name: 'Balcony', price: 1200, total_seats: 400, description: 'Upper level seating' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Stand-up Comedy Night: Thai Edition',
      description: 'A night of laughter featuring Thailand top comedians.',
      short_description: 'Comedy Night Bangkok - Thai Comedians Special',
      venue_name: 'Scala Theater',
      venue_address: 'Siam Square Soi 1',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1585699324551-f6c309eedeca?w=800',
      banner_url: 'https://images.unsplash.com/photo-1527224538127-2104bb71c51b?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-11-01T10:00:00+07:00',
      booking_end_at: '2025-11-27T23:59:59+07:00'
    },
    shows: [
      { name: 'Evening Show', show_date: '2025-11-28', start_time: '20:00:00+07:00', end_time: '22:30:00+07:00' }
    ],
    zones: [
      { name: 'VIP', price: 1500, total_seats: 100, description: 'Front row with meet and greet' },
      { name: 'Standard', price: 800, total_seats: 300, description: 'Standard seating' },
      { name: 'Economy', price: 500, total_seats: 200, description: 'Back section' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Bangkok Street Food Festival',
      description: 'Celebrate Thailand culinary heritage at this 3-day food festival.',
      short_description: 'Street Food Festival - Taste of Thailand',
      venue_name: 'Central World Square',
      venue_address: '999/9 Rama I Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1555939594-58d7cb561ad1?w=800',
      banner_url: 'https://images.unsplash.com/photo-1504674900247-0877df9cc836?w=1600',
      max_tickets_per_user: 6,
      booking_start_at: '2025-11-01T10:00:00+07:00',
      booking_end_at: '2026-01-10T23:59:59+07:00'
    },
    shows: [
      { name: 'Weekend Pass', show_date: '2026-01-11', start_time: '11:00:00+07:00', end_time: '22:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP All-You-Can-Eat', price: 1500, total_seats: 500, description: 'Unlimited food sampling' },
      { name: 'Premium Pass', price: 800, total_seats: 1000, description: '10 food vouchers included' },
      { name: 'General Entry', price: 299, total_seats: 2000, description: 'Festival entry only' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'TechCrunch Bangkok 2026',
      description: 'Southeast Asia premier technology conference featuring keynotes from tech leaders.',
      short_description: 'TechCrunch Conference - Innovation Summit',
      venue_name: 'Queen Sirikit National Convention Center',
      venue_address: '60 New Rachadapisek Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=800',
      banner_url: 'https://images.unsplash.com/photo-1505373877841-8d25f7d46678?w=1600',
      max_tickets_per_user: 2,
      booking_start_at: '2026-01-01T10:00:00+07:00',
      booking_end_at: '2026-02-17T23:59:59+07:00'
    },
    shows: [
      { name: 'Day 1 - Keynotes', show_date: '2026-02-18', start_time: '09:00:00+07:00', end_time: '18:00:00+07:00' },
      { name: 'Day 2 - Workshops', show_date: '2026-02-19', start_time: '09:00:00+07:00', end_time: '18:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP All-Access', price: 15000, total_seats: 100, description: 'All sessions + networking dinner' },
      { name: 'Conference Pass', price: 8500, total_seats: 500, description: 'All sessions access' },
      { name: 'Startup Pass', price: 3500, total_seats: 300, description: 'Discounted for startups' },
      { name: 'Student Pass', price: 1500, total_seats: 200, description: 'Student discount' }
    ]
  }
];

async function main() {
  console.log('=== Event Seed Script (JavaScript) ===');
  console.log(`API URL: ${API_BASE_URL}`);
  console.log(`User: ${ADMIN_EMAIL}`);
  console.log('');

  await login();
  console.log('');
  console.log('=== Creating 10 Events ===');
  console.log('');

  let created = 0;

  for (let i = 0; i < events.length; i++) {
    const { status, event, shows, zones } = events[i];
    const statusColor = status === 'OPEN' ? 'green' : status === 'UPCOMING' ? 'blue' : 'red';

    log(statusColor, `[${i + 1}/10] ${status} - ${event.name}`);

    const eventId = await createEvent(event);
    if (!eventId) continue;

    for (const show of shows) {
      const showId = await createShow(eventId, show);
      if (!showId) continue;

      for (const zone of zones) {
        await createZone(showId, { ...zone, sort_order: 0 });
      }
    }

    await publishEvent(eventId);
    created++;
    console.log('');
  }

  console.log('=== Seed Complete ===');
  console.log('');
  log('green', `Successfully created ${created} events!`);
  console.log('');
  console.log('Summary:');
  log('green', '  OPEN (5): BTS, Coldplay, Muay Thai, Symphony, Food Festival');
  log('blue', '  UPCOMING (3): Ed Sheeran, Jazz Festival, TechCrunch');
  log('red', '  ENDED (2): Summer Sonic, Comedy Night');
}

main().catch(err => {
  console.error('Error:', err);
  process.exit(1);
});
