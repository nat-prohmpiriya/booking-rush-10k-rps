#!/usr/bin/env node

/**
 * 02-seed-events.mjs
 * Creates 30 events with shows and zones via API
 *
 * Usage: ADMIN_EMAIL="organizer@test.com" ADMIN_PASSWORD="Test123!" node scripts/02-seed-events.mjs
 *
 * Distribution:
 * - OPEN (15): Events currently on sale
 * - UPCOMING (8): Events opening soon
 * - ENDED (7): Past events
 */

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080/api/v1';
const ADMIN_EMAIL = process.env.ADMIN_EMAIL || 'organizer@test.com';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || 'Test123!';

let ACCESS_TOKEN = '';

const colors = {
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
  reset: '\x1b[0m'
};

// Delay function to avoid rate limiting
const delay = (ms) => new Promise(resolve => setTimeout(resolve, ms));
const DELAY_MS = 2000; // 2 seconds between API calls

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
    log('yellow', '\nMake sure you have run: node scripts/01-seed-users.mjs');
    process.exit(1);
  }

  ACCESS_TOKEN = data.data.access_token;
  log('green', `Login successful! Role: ${data.data.user.role}, Tenant: ${data.data.user.tenant_id || 'none'}`);
}

async function createEvent(eventData) {
  await delay(DELAY_MS);

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
    log('red', `  Failed: ${JSON.stringify(data.error)}`);
    return null;
  }

  return data.data.id;
}

async function createShow(eventId, showData) {
  await delay(DELAY_MS);

  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/shows`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${ACCESS_TOKEN}`
      },
      body: JSON.stringify(showData)
    });

    const text = await res.text();
    try {
      const data = JSON.parse(text);
      if (!data.success) return null;
      return data.data.id;
    } catch (e) {
      log('red', `  Show error: Invalid JSON response`);
      return null;
    }
  } catch (e) {
    log('red', `  Show error: ${e.message}`);
    return null;
  }
}

async function createZone(showId, zoneData) {
  await delay(DELAY_MS);

  try {
    const res = await fetch(`${API_BASE_URL}/shows/${showId}/zones`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${ACCESS_TOKEN}`
      },
      body: JSON.stringify(zoneData)
    });

    const text = await res.text();
    try {
      const data = JSON.parse(text);
      if (!data.success) return null;
      return data.data.id;
    } catch (e) {
      log('red', `  Zone error: Invalid JSON response`);
      return null;
    }
  } catch (e) {
    log('red', `  Zone error: ${e.message}`);
    return null;
  }
}

async function publishEvent(eventId) {
  const res = await fetch(`${API_BASE_URL}/events/${eventId}/publish`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${ACCESS_TOKEN}` }
  });

  const data = await res.json();
  return data.success;
}

// ============================================================================
// EVENT DEFINITIONS (30 Events)
// ============================================================================

const events = [
  // ============================================================================
  // OPEN EVENTS (15)
  // ============================================================================
  {
    status: 'OPEN',
    event: {
      name: 'BTS World Tour: Love Yourself',
      description: 'Experience the global phenomenon BTS live in Bangkok!',
      short_description: 'BTS Live in Bangkok - World Tour 2025',
      venue_name: 'Rajamangala National Stadium',
      venue_address: '286 Ramkhamhaeng Rd',
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
      { name: 'VVIP Standing', price: 12000, total_seats: 500, description: 'Front stage standing' },
      { name: 'VIP Standing', price: 8500, total_seats: 1000, description: 'Premium standing' },
      { name: 'Gold Seated', price: 5500, total_seats: 2000, description: 'Lower bowl' },
      { name: 'Silver Seated', price: 3500, total_seats: 3000, description: 'Upper bowl' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Coldplay Music of the Spheres',
      description: 'Coldplay returns to Bangkok with their spectacular World Tour.',
      short_description: 'Coldplay World Tour 2026 - Bangkok',
      venue_name: 'Rajamangala National Stadium',
      venue_address: '286 Ramkhamhaeng Rd',
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
      { name: 'Infinity Ticket', price: 9500, total_seats: 1000, description: 'GA Standing' },
      { name: 'A Reserve', price: 7500, total_seats: 2000, description: 'Premium seating' },
      { name: 'B Reserve', price: 5500, total_seats: 3000, description: 'Standard seating' },
      { name: 'C Reserve', price: 3500, total_seats: 4000, description: 'Value seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Muay Thai Super Fight Night',
      description: 'Witness the best Muay Thai fighters from around the world.',
      short_description: 'World Championship Muay Thai',
      venue_name: 'Lumpinee Boxing Stadium',
      venue_address: '6 Ramintra Rd',
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
      { name: 'Standard', price: 1500, total_seats: 500, description: 'Standard seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Royal Bangkok Symphony Orchestra',
      description: 'An evening of classical masterpieces.',
      short_description: 'Classical Night - Beethoven and Tchaikovsky',
      venue_name: 'Thailand Cultural Centre',
      venue_address: '14 Thiam Ruam Mit Rd',
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
      { name: 'Orchestra', price: 2500, total_seats: 200, description: 'Best acoustic' },
      { name: 'Mezzanine', price: 1800, total_seats: 300, description: 'Elevated view' },
      { name: 'Balcony', price: 1200, total_seats: 400, description: 'Upper level' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Bangkok Street Food Festival',
      description: 'Celebrate Thailand culinary heritage.',
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
      { name: 'VIP All-You-Can-Eat', price: 1500, total_seats: 500, description: 'Unlimited food' },
      { name: 'Premium Pass', price: 800, total_seats: 1000, description: '10 vouchers' },
      { name: 'General Entry', price: 299, total_seats: 2000, description: 'Entry only' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Taylor Swift The Eras Tour',
      description: 'Taylor Swift brings her record-breaking Eras Tour to Bangkok.',
      short_description: 'Taylor Swift Eras Tour Bangkok 2026',
      venue_name: 'Rajamangala National Stadium',
      venue_address: '286 Ramkhamhaeng Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1501281668745-f7f57925c3b4?w=800',
      banner_url: 'https://images.unsplash.com/photo-1514525253161-7a46d19cd819?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-12-01T10:00:00+07:00',
      booking_end_at: '2026-03-14T23:59:59+07:00'
    },
    shows: [
      { name: 'Night 1', show_date: '2026-03-15', start_time: '19:00:00+07:00', end_time: '23:00:00+07:00' },
      { name: 'Night 2', show_date: '2026-03-16', start_time: '19:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'Floor Standing', price: 15000, total_seats: 800, description: 'Closest to stage' },
      { name: 'Lower Bowl', price: 9500, total_seats: 2000, description: 'Premium seating' },
      { name: 'Upper Bowl', price: 5500, total_seats: 4000, description: 'Standard seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'UFC Fight Night Bangkok',
      description: 'UFC returns to Bangkok with an action-packed fight card.',
      short_description: 'UFC Fight Night - Thailand',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1552072092-7f9b8d63efcb?w=800',
      banner_url: 'https://images.unsplash.com/photo-1579373903781-fd5c0c30c4cd?w=1600',
      max_tickets_per_user: 6,
      booking_start_at: '2025-12-01T10:00:00+07:00',
      booking_end_at: '2026-02-27T23:59:59+07:00'
    },
    shows: [
      { name: 'Main Event', show_date: '2026-02-28', start_time: '17:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'Cageside', price: 25000, total_seats: 100, description: 'VIP cageside' },
      { name: 'Floor', price: 12000, total_seats: 500, description: 'Floor seating' },
      { name: 'Lower Tier', price: 6000, total_seats: 1500, description: 'Lower tier' },
      { name: 'Upper Tier', price: 3000, total_seats: 3000, description: 'Upper tier' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Disney On Ice: Frozen Adventures',
      description: 'Join Anna, Elsa, and friends in this magical ice spectacular.',
      short_description: 'Disney On Ice presents Frozen',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1607344645866-009c320b63e0?w=800',
      banner_url: 'https://images.unsplash.com/photo-1533929736458-ca588d08c8be?w=1600',
      max_tickets_per_user: 8,
      booking_start_at: '2025-11-15T10:00:00+07:00',
      booking_end_at: '2026-01-24T23:59:59+07:00'
    },
    shows: [
      { name: 'Saturday Matinee', show_date: '2026-01-25', start_time: '14:00:00+07:00', end_time: '17:00:00+07:00' },
      { name: 'Saturday Evening', show_date: '2026-01-25', start_time: '19:00:00+07:00', end_time: '22:00:00+07:00' },
      { name: 'Sunday Matinee', show_date: '2026-01-26', start_time: '14:00:00+07:00', end_time: '17:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP', price: 4500, total_seats: 300, description: 'Best view + gift' },
      { name: 'Premium', price: 3000, total_seats: 800, description: 'Great view' },
      { name: 'Standard', price: 1800, total_seats: 1500, description: 'Standard' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Bruno Mars 24K Magic World Tour',
      description: 'Bruno Mars brings his electrifying tour to Bangkok.',
      short_description: 'Bruno Mars Live in Bangkok',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1516450360452-9312f5e86fc7?w=800',
      banner_url: 'https://images.unsplash.com/photo-1492684223066-81342ee5ff30?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-12-05T10:00:00+07:00',
      booking_end_at: '2026-02-13T23:59:59+07:00'
    },
    shows: [
      { name: 'Bangkok Show', show_date: '2026-02-14', start_time: '20:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'Golden Circle', price: 12000, total_seats: 500, description: 'Standing front' },
      { name: 'CAT 1', price: 8500, total_seats: 1000, description: 'Premium' },
      { name: 'CAT 2', price: 5500, total_seats: 2000, description: 'Standard' },
      { name: 'CAT 3', price: 3500, total_seats: 2500, description: 'Value' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Cirque du Soleil: Alegria',
      description: 'Experience the wonder of Cirque du Soleil.',
      short_description: 'Cirque du Soleil Alegria Bangkok',
      venue_name: 'Royal Paragon Hall',
      venue_address: 'Siam Paragon',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=800',
      banner_url: 'https://images.unsplash.com/photo-1518834107812-67b0b7c58434?w=1600',
      max_tickets_per_user: 6,
      booking_start_at: '2025-11-20T10:00:00+07:00',
      booking_end_at: '2026-01-30T23:59:59+07:00'
    },
    shows: [
      { name: 'Opening Night', show_date: '2026-01-31', start_time: '20:00:00+07:00', end_time: '22:30:00+07:00' },
      { name: 'Weekend Show', show_date: '2026-02-01', start_time: '20:00:00+07:00', end_time: '22:30:00+07:00' }
    ],
    zones: [
      { name: 'Platinum', price: 8900, total_seats: 200, description: 'Best seats' },
      { name: 'Gold', price: 6500, total_seats: 500, description: 'Premium' },
      { name: 'Silver', price: 4500, total_seats: 800, description: 'Great value' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'ONE Championship: Bangkok Battleground',
      description: 'Asia premier martial arts organization.',
      short_description: 'ONE Championship Bangkok',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1555597673-b21d5c3c8232?w=800',
      banner_url: 'https://images.unsplash.com/photo-1517438476312-10d79c077509?w=1600',
      max_tickets_per_user: 6,
      booking_start_at: '2025-12-01T10:00:00+07:00',
      booking_end_at: '2026-01-16T23:59:59+07:00'
    },
    shows: [
      { name: 'Fight Night', show_date: '2026-01-17', start_time: '18:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'Ringside VIP', price: 15000, total_seats: 100, description: 'Ringside VIP' },
      { name: 'Ringside', price: 8000, total_seats: 300, description: 'Ringside' },
      { name: 'Lower Bowl', price: 4000, total_seats: 1000, description: 'Lower bowl' },
      { name: 'Upper Bowl', price: 1500, total_seats: 2000, description: 'Upper bowl' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Wine and Dine Festival Bangkok',
      description: 'Premium wine tasting with 200+ wines.',
      short_description: 'Wine and Dine Festival 2026',
      venue_name: 'Centara Grand',
      venue_address: '999/99 Rama I Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1510812431401-41d2bd2722f3?w=800',
      banner_url: 'https://images.unsplash.com/photo-1558642452-9d2a7deb7f62?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-12-01T10:00:00+07:00',
      booking_end_at: '2026-02-06T23:59:59+07:00'
    },
    shows: [
      { name: 'VIP Evening', show_date: '2026-02-07', start_time: '18:00:00+07:00', end_time: '22:00:00+07:00' },
      { name: 'Grand Tasting', show_date: '2026-02-08', start_time: '14:00:00+07:00', end_time: '20:00:00+07:00' }
    ],
    zones: [
      { name: 'Platinum', price: 5500, total_seats: 100, description: 'All wines + premium food' },
      { name: 'Gold', price: 3500, total_seats: 300, description: 'Selected wines + food' },
      { name: 'Silver', price: 1800, total_seats: 500, description: 'Entry + 10 tastings' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Marvel Universe Live!',
      description: 'Marvel superheroes battle villains in this live arena show.',
      short_description: 'Marvel Universe Live Bangkok',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1612036782180-6f0b6cd846fe?w=800',
      banner_url: 'https://images.unsplash.com/photo-1608889825103-eb5ed706fc64?w=1600',
      max_tickets_per_user: 8,
      booking_start_at: '2025-11-25T10:00:00+07:00',
      booking_end_at: '2026-02-20T23:59:59+07:00'
    },
    shows: [
      { name: 'Saturday Show', show_date: '2026-02-21', start_time: '15:00:00+07:00', end_time: '18:00:00+07:00' },
      { name: 'Sunday Show', show_date: '2026-02-22', start_time: '15:00:00+07:00', end_time: '18:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Floor', price: 5500, total_seats: 400, description: 'Floor + gift' },
      { name: 'Premium', price: 3500, total_seats: 1000, description: 'Premium' },
      { name: 'Standard', price: 2000, total_seats: 2000, description: 'Standard' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Bangkok Art Biennale Gala',
      description: 'Exclusive gala celebrating contemporary art.',
      short_description: 'BAB Gala Night 2026',
      venue_name: 'Bangkok Art and Culture Centre',
      venue_address: '939 Rama I Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1531243269054-5ebf6f34081e?w=800',
      banner_url: 'https://images.unsplash.com/photo-1460661419201-fd4cecdf8a8b?w=1600',
      max_tickets_per_user: 2,
      booking_start_at: '2025-12-01T10:00:00+07:00',
      booking_end_at: '2026-01-14T23:59:59+07:00'
    },
    shows: [
      { name: 'Gala Night', show_date: '2026-01-15', start_time: '18:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'Patron', price: 25000, total_seats: 50, description: 'VIP dinner + art piece' },
      { name: 'Benefactor', price: 12000, total_seats: 150, description: 'Gala dinner' },
      { name: 'Supporter', price: 5000, total_seats: 300, description: 'Cocktails + exhibition' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Yoga & Wellness Retreat',
      description: 'A day of mindfulness, yoga, and wellness workshops.',
      short_description: 'Bangkok Wellness Retreat 2026',
      venue_name: 'Centara Grand Beach Resort',
      venue_address: 'Hua Hin',
      city: 'Prachuap Khiri Khan',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1545389336-cf090694435e?w=800',
      banner_url: 'https://images.unsplash.com/photo-1506126613408-eca07ce68773?w=1600',
      max_tickets_per_user: 2,
      booking_start_at: '2025-12-01T10:00:00+07:00',
      booking_end_at: '2026-02-14T23:59:59+07:00'
    },
    shows: [
      { name: 'Full Day Retreat', show_date: '2026-02-15', start_time: '06:00:00+07:00', end_time: '18:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Package', price: 8500, total_seats: 30, description: 'All sessions + spa + lunch' },
      { name: 'Full Access', price: 4500, total_seats: 100, description: 'All sessions + lunch' },
      { name: 'Morning Only', price: 2000, total_seats: 50, description: 'Morning sessions' }
    ]
  },

  // ============================================================================
  // UPCOMING EVENTS (8)
  // ============================================================================
  {
    status: 'UPCOMING',
    event: {
      name: 'Ed Sheeran Mathematics Tour',
      description: 'Grammy Award winner Ed Sheeran brings his tour to Bangkok.',
      short_description: 'Ed Sheeran Live - Mathematics Tour',
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
      { name: 'CAT 1', price: 8900, total_seats: 800, description: 'Best seats' },
      { name: 'CAT 2', price: 6500, total_seats: 1500, description: 'Premium' },
      { name: 'CAT 3', price: 4500, total_seats: 2000, description: 'Great value' },
      { name: 'CAT 4', price: 2500, total_seats: 2500, description: 'Budget' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Bangkok International Jazz Festival',
      description: 'Three days of world-class jazz.',
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
      { name: 'VIP Table', price: 3500, total_seats: 50, description: 'Reserved table' },
      { name: 'Premium GA', price: 1800, total_seats: 500, description: 'Premium area' },
      { name: 'General Admission', price: 900, total_seats: 1000, description: 'Standard' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'TechCrunch Bangkok 2026',
      description: 'Southeast Asia premier technology conference.',
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
      { name: 'VIP All-Access', price: 15000, total_seats: 100, description: 'All sessions + dinner' },
      { name: 'Conference Pass', price: 8500, total_seats: 500, description: 'All sessions' },
      { name: 'Startup Pass', price: 3500, total_seats: 300, description: 'Startup discount' },
      { name: 'Student Pass', price: 1500, total_seats: 200, description: 'Student discount' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Blackpink World Tour: Born Pink',
      description: 'K-pop queens Blackpink bring their tour to Bangkok.',
      short_description: 'Blackpink Born Pink Tour Bangkok',
      venue_name: 'Rajamangala National Stadium',
      venue_address: '286 Ramkhamhaeng Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1493225457124-a3eb161ffa5f?w=800',
      banner_url: 'https://images.unsplash.com/photo-1470229722913-7c0e2dbbafd3?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2026-01-15T10:00:00+07:00',
      booking_end_at: '2026-04-09T23:59:59+07:00'
    },
    shows: [
      { name: 'Night 1', show_date: '2026-04-10', start_time: '19:00:00+07:00', end_time: '22:00:00+07:00' },
      { name: 'Night 2', show_date: '2026-04-11', start_time: '19:00:00+07:00', end_time: '22:00:00+07:00' }
    ],
    zones: [
      { name: 'Blink Zone', price: 15000, total_seats: 500, description: 'Closest to stage' },
      { name: 'Pink Zone', price: 9500, total_seats: 1500, description: 'Premium standing' },
      { name: 'Seated A', price: 6500, total_seats: 3000, description: 'Lower bowl' },
      { name: 'Seated B', price: 4000, total_seats: 4000, description: 'Upper bowl' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Formula E Bangkok E-Prix',
      description: 'Electric racing comes to Bangkok streets.',
      short_description: 'Formula E Bangkok E-Prix 2026',
      venue_name: 'Bangkok Street Circuit',
      venue_address: 'Royal Thai Navy Area',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1568605117036-5fe5e7bab0b7?w=800',
      banner_url: 'https://images.unsplash.com/photo-1541348263662-e068662d82af?w=1600',
      max_tickets_per_user: 6,
      booking_start_at: '2026-02-01T10:00:00+07:00',
      booking_end_at: '2026-05-14T23:59:59+07:00'
    },
    shows: [
      { name: 'Race Day', show_date: '2026-05-15', start_time: '10:00:00+07:00', end_time: '18:00:00+07:00' }
    ],
    zones: [
      { name: 'Paddock Club', price: 35000, total_seats: 100, description: 'Paddock access' },
      { name: 'Grandstand A', price: 8500, total_seats: 500, description: 'Prime grandstand' },
      { name: 'Grandstand B', price: 5000, total_seats: 1000, description: 'Secondary grandstand' },
      { name: 'General Admission', price: 2000, total_seats: 5000, description: 'Roaming access' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Anime Festival Asia Bangkok',
      description: 'Southeast Asia largest anime convention.',
      short_description: 'AFA Bangkok 2026',
      venue_name: 'BITEC Bangna',
      venue_address: '88 Bangna-Trad Road',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1578632767115-351597cf2477?w=800',
      banner_url: 'https://images.unsplash.com/photo-1613376023733-0a73315d9b06?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2026-01-20T10:00:00+07:00',
      booking_end_at: '2026-03-19T23:59:59+07:00'
    },
    shows: [
      { name: 'Day 1', show_date: '2026-03-20', start_time: '10:00:00+07:00', end_time: '21:00:00+07:00' },
      { name: 'Day 2', show_date: '2026-03-21', start_time: '10:00:00+07:00', end_time: '21:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Pass', price: 3500, total_seats: 500, description: '2-day + merch' },
      { name: 'Day Pass', price: 1200, total_seats: 3000, description: 'Single day' },
      { name: 'Concert Only', price: 2500, total_seats: 1000, description: 'Evening concert' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'World Barista Championship',
      description: 'The world best baristas compete in Bangkok.',
      short_description: 'WBC Bangkok 2026',
      venue_name: 'Queen Sirikit National Convention Center',
      venue_address: '60 New Rachadapisek Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?w=800',
      banner_url: 'https://images.unsplash.com/photo-1442512595331-e89e73853f31?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2026-02-15T10:00:00+07:00',
      booking_end_at: '2026-04-24T23:59:59+07:00'
    },
    shows: [
      { name: 'Finals Day', show_date: '2026-04-25', start_time: '09:00:00+07:00', end_time: '18:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Tasting', price: 4500, total_seats: 100, description: 'Tastings + finals' },
      { name: 'Full Access', price: 2500, total_seats: 500, description: 'All competitions' },
      { name: 'Finals Only', price: 1200, total_seats: 800, description: 'Finals viewing' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Dua Lipa Future Nostalgia Tour',
      description: 'Pop superstar Dua Lipa brings her tour to Bangkok.',
      short_description: 'Dua Lipa Live in Bangkok',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1619229666372-3c26c399a4cb?w=800',
      banner_url: 'https://images.unsplash.com/photo-1501386761578-eac5c94b800a?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2026-02-01T10:00:00+07:00',
      booking_end_at: '2026-05-01T23:59:59+07:00'
    },
    shows: [
      { name: 'Bangkok Show', show_date: '2026-05-02', start_time: '20:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Standing', price: 9500, total_seats: 800, description: 'Front standing' },
      { name: 'CAT 1', price: 7000, total_seats: 1500, description: 'Premium' },
      { name: 'CAT 2', price: 5000, total_seats: 2000, description: 'Standard' },
      { name: 'CAT 3', price: 3000, total_seats: 2500, description: 'Value' }
    ]
  },

  // ============================================================================
  // ENDED EVENTS (7)
  // ============================================================================
  {
    status: 'ENDED',
    event: {
      name: 'Summer Sonic Bangkok',
      description: 'Thailand premier summer music festival.',
      short_description: 'Summer Sonic Festival 2025',
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
      { name: 'VIP', price: 5500, total_seats: 500, description: 'VIP area' },
      { name: 'General Admission', price: 2500, total_seats: 5000, description: 'Full access' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Stand-up Comedy Night: Thai Edition',
      description: 'A night of laughter featuring Thailand top comedians.',
      short_description: 'Comedy Night Bangkok',
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
      { name: 'VIP', price: 1500, total_seats: 100, description: 'Front row + meet and greet' },
      { name: 'Standard', price: 800, total_seats: 300, description: 'Standard' },
      { name: 'Economy', price: 500, total_seats: 200, description: 'Back section' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Harry Styles Love On Tour',
      description: 'Harry Styles brought his tour to Bangkok.',
      short_description: 'Harry Styles Bangkok 2025',
      venue_name: 'Impact Arena',
      venue_address: '99 Popular Rd, Pak Kret',
      city: 'Nonthaburi',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1516450360452-9312f5e86fc7?w=800',
      banner_url: 'https://images.unsplash.com/photo-1459749411175-04bf5292ceea?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-08-01T10:00:00+07:00',
      booking_end_at: '2025-10-19T23:59:59+07:00'
    },
    shows: [
      { name: 'Night 1', show_date: '2025-10-20', start_time: '20:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'Golden Circle', price: 9500, total_seats: 500, description: 'Standing front' },
      { name: 'CAT 1', price: 6500, total_seats: 1500, description: 'Premium' },
      { name: 'CAT 2', price: 4000, total_seats: 2500, description: 'Standard' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'World Travel Fair Bangkok',
      description: 'Annual travel exhibition with amazing deals.',
      short_description: 'Travel Fair Bangkok 2025',
      venue_name: 'BITEC Bangna',
      venue_address: '88 Bangna-Trad Road',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1436491865332-7a61a109cc05?w=800',
      banner_url: 'https://images.unsplash.com/photo-1488646953014-85cb44e25828?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-09-01T10:00:00+07:00',
      booking_end_at: '2025-11-07T23:59:59+07:00'
    },
    shows: [
      { name: 'Weekend', show_date: '2025-11-08', start_time: '10:00:00+07:00', end_time: '20:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Early Access', price: 500, total_seats: 500, description: 'Early entry + vouchers' },
      { name: 'General', price: 100, total_seats: 5000, description: 'Standard entry' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Bangkok Marathon 2025',
      description: 'Annual Bangkok Marathon with multiple categories.',
      short_description: 'Bangkok Marathon 2025',
      venue_name: 'Sanam Chai Road',
      venue_address: 'Grand Palace Area',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1452626038306-9aae5e071dd3?w=800',
      banner_url: 'https://images.unsplash.com/photo-1571008887538-b36bb32f4571?w=1600',
      max_tickets_per_user: 1,
      booking_start_at: '2025-07-01T10:00:00+07:00',
      booking_end_at: '2025-11-15T23:59:59+07:00'
    },
    shows: [
      { name: 'Race Day', show_date: '2025-11-16', start_time: '04:00:00+07:00', end_time: '12:00:00+07:00' }
    ],
    zones: [
      { name: 'Full Marathon', price: 1800, total_seats: 5000, description: '42.195km' },
      { name: 'Half Marathon', price: 1500, total_seats: 8000, description: '21.1km' },
      { name: 'Fun Run 10K', price: 800, total_seats: 10000, description: '10km' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Thailand Game Show 2025',
      description: 'Southeast Asia biggest gaming expo.',
      short_description: 'TGS 2025',
      venue_name: 'Queen Sirikit National Convention Center',
      venue_address: '60 New Rachadapisek Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1542751371-adc38448a05e?w=800',
      banner_url: 'https://images.unsplash.com/photo-1511512578047-dfb367046420?w=1600',
      max_tickets_per_user: 4,
      booking_start_at: '2025-09-15T10:00:00+07:00',
      booking_end_at: '2025-10-24T23:59:59+07:00'
    },
    shows: [
      { name: 'Day 1', show_date: '2025-10-25', start_time: '10:00:00+07:00', end_time: '21:00:00+07:00' },
      { name: 'Day 2', show_date: '2025-10-26', start_time: '10:00:00+07:00', end_time: '21:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Pass', price: 2500, total_seats: 500, description: '2-day + exclusive' },
      { name: 'Day Pass', price: 450, total_seats: 10000, description: 'Single day' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Oktoberfest Bangkok 2025',
      description: 'German beer festival celebration.',
      short_description: 'Oktoberfest Bangkok 2025',
      venue_name: 'Watergate Pavillion',
      venue_address: 'Ratchaprarop Rd',
      city: 'Bangkok',
      country: 'Thailand',
      poster_url: 'https://images.unsplash.com/photo-1567696911980-2eed69a46042?w=800',
      banner_url: 'https://images.unsplash.com/photo-1572894086901-1a3da4de4917?w=1600',
      max_tickets_per_user: 6,
      booking_start_at: '2025-09-01T10:00:00+07:00',
      booking_end_at: '2025-10-17T23:59:59+07:00'
    },
    shows: [
      { name: 'Opening Weekend', show_date: '2025-10-18', start_time: '17:00:00+07:00', end_time: '23:00:00+07:00' }
    ],
    zones: [
      { name: 'VIP Table', price: 5000, total_seats: 100, description: 'Reserved table + beers' },
      { name: 'Premium', price: 1500, total_seats: 500, description: 'Premium + 2 beers' },
      { name: 'General', price: 500, total_seats: 2000, description: 'General admission' }
    ]
  }
];

// ============================================================================
// MAIN
// ============================================================================

async function main() {
  console.log('');
  log('cyan', '='.repeat(60));
  log('cyan', '  Booking Rush - Seed Events Script (30 Events)');
  log('cyan', '='.repeat(60));
  console.log('');
  log('blue', `API URL: ${API_BASE_URL}`);
  log('blue', `User: ${ADMIN_EMAIL}`);
  console.log('');

  await login();
  console.log('');
  log('yellow', '[Step 2] Creating events...');
  console.log('');

  let created = 0;
  const total = events.length;

  for (let i = 0; i < events.length; i++) {
    const { status, event, shows, zones } = events[i];
    const statusColor = status === 'OPEN' ? 'green' : status === 'UPCOMING' ? 'blue' : 'red';
    const shortName = event.name.length > 35 ? event.name.substring(0, 35) + '...' : event.name;

    process.stdout.write(`  [${String(i + 1).padStart(2)}/${total}] `);
    log(statusColor, `${status.padEnd(8)} ${shortName}`);

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
  }

  // Summary
  console.log('');
  log('cyan', '='.repeat(60));
  log('green', `  Created ${created}/${total} events successfully!`);
  log('cyan', '='.repeat(60));
  console.log('');
  log('blue', 'Summary:');
  log('green', '  OPEN (15):     BTS, Coldplay, Taylor Swift, Bruno Mars, etc.');
  log('blue', '  UPCOMING (8):  Ed Sheeran, Blackpink, Dua Lipa, Formula E, etc.');
  log('red', '  ENDED (7):     Summer Sonic, Harry Styles, Oktoberfest, etc.');
  console.log('');
}

main().catch(err => {
  console.error('Error:', err);
  process.exit(1);
});
