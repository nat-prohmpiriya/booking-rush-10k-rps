#!/usr/bin/env node

/**
 * Seed Extra Events Script
 * Creates 20 additional events with shows and zones via API
 */

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080/api/v1';
const ADMIN_EMAIL = process.env.ADMIN_EMAIL || 'test1@test.com';
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || '#Ttest1234';

let ACCESS_TOKEN = '';

const colors = {
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
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
  log('green', `  Show: ${showData.name}`);
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
  if (!data.success) return null;
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

const events = [
  // === OPEN Events (10) ===
  {
    status: 'OPEN',
    event: {
      name: 'Taylor Swift The Eras Tour',
      description: 'Taylor Swift brings her record-breaking Eras Tour to Bangkok for an unforgettable night spanning her entire musical catalog.',
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
      description: 'The Ultimate Fighting Championship returns to Bangkok with an action-packed fight card.',
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
      { name: 'Cageside', price: 25000, total_seats: 100, description: 'VIP cageside seats' },
      { name: 'Floor', price: 12000, total_seats: 500, description: 'Floor seating' },
      { name: 'Lower Tier', price: 6000, total_seats: 1500, description: 'Lower tier seating' },
      { name: 'Upper Tier', price: 3000, total_seats: 3000, description: 'Upper tier seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Disney On Ice: Frozen Adventures',
      description: 'Join Anna, Elsa, and friends in this magical ice skating spectacular.',
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
      { name: 'VIP', price: 4500, total_seats: 300, description: 'Best view with gift pack' },
      { name: 'Premium', price: 3000, total_seats: 800, description: 'Great viewing angle' },
      { name: 'Standard', price: 1800, total_seats: 1500, description: 'Standard seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Bruno Mars 24K Magic World Tour',
      description: 'Bruno Mars brings his electrifying 24K Magic World Tour to Bangkok.',
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
      { name: 'Golden Circle', price: 12000, total_seats: 500, description: 'Standing closest to stage' },
      { name: 'CAT 1', price: 8500, total_seats: 1000, description: 'Premium reserved' },
      { name: 'CAT 2', price: 5500, total_seats: 2000, description: 'Standard reserved' },
      { name: 'CAT 3', price: 3500, total_seats: 2500, description: 'Value seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Cirque du Soleil: Alegria',
      description: 'Experience the wonder of Cirque du Soleil with their iconic show Alegria.',
      short_description: 'Cirque du Soleil Alegria Bangkok',
      venue_name: 'Royal Paragon Hall',
      venue_address: 'Siam Paragon, Rama I Rd',
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
      { name: 'Weekend Show 1', show_date: '2026-02-01', start_time: '20:00:00+07:00', end_time: '22:30:00+07:00' }
    ],
    zones: [
      { name: 'Platinum', price: 8900, total_seats: 200, description: 'Best seats in house' },
      { name: 'Gold', price: 6500, total_seats: 500, description: 'Premium viewing' },
      { name: 'Silver', price: 4500, total_seats: 800, description: 'Great value' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'ONE Championship: Bangkok Battleground',
      description: 'Asia premier martial arts organization brings world-class MMA action to Bangkok.',
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
      { name: 'Ringside VIP', price: 15000, total_seats: 100, description: 'Ringside VIP experience' },
      { name: 'Ringside', price: 8000, total_seats: 300, description: 'Ringside seating' },
      { name: 'Lower Bowl', price: 4000, total_seats: 1000, description: 'Lower bowl seating' },
      { name: 'Upper Bowl', price: 1500, total_seats: 2000, description: 'Upper bowl seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Wine and Dine Festival Bangkok',
      description: 'Premium wine tasting event featuring over 200 wines from around the world with gourmet food pairings.',
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
      description: 'Watch your favorite Marvel superheroes battle villains in this action-packed live arena show.',
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
      { name: 'VIP Floor', price: 5500, total_seats: 400, description: 'Floor seating with gift' },
      { name: 'Premium', price: 3500, total_seats: 1000, description: 'Premium seating' },
      { name: 'Standard', price: 2000, total_seats: 2000, description: 'Standard seating' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Bangkok Art Biennale Gala',
      description: 'Exclusive gala evening celebrating contemporary art with live performances and exhibitions.',
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
      { name: 'Benefactor', price: 12000, total_seats: 150, description: 'Gala dinner + cocktails' },
      { name: 'Supporter', price: 5000, total_seats: 300, description: 'Cocktails + exhibition' }
    ]
  },
  {
    status: 'OPEN',
    event: {
      name: 'Yoga & Wellness Retreat',
      description: 'A day of mindfulness, yoga sessions, and wellness workshops with renowned instructors.',
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

  // === UPCOMING Events (5) ===
  {
    status: 'UPCOMING',
    event: {
      name: 'Blackpink World Tour: Born Pink',
      description: 'K-pop queens Blackpink bring their Born Pink World Tour to Bangkok.',
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
      { name: 'Seated A', price: 6500, total_seats: 3000, description: 'Lower bowl seats' },
      { name: 'Seated B', price: 4000, total_seats: 4000, description: 'Upper bowl seats' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Formula E Bangkok E-Prix',
      description: 'Electric racing comes to Bangkok streets with the Formula E championship.',
      short_description: 'Formula E Bangkok E-Prix 2026',
      venue_name: 'Bangkok Street Circuit',
      venue_address: 'Royal Thai Navy Convention Hall Area',
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
      { name: 'Paddock Club', price: 35000, total_seats: 100, description: 'Paddock access + hospitality' },
      { name: 'Grandstand A', price: 8500, total_seats: 500, description: 'Prime grandstand' },
      { name: 'Grandstand B', price: 5000, total_seats: 1000, description: 'Secondary grandstand' },
      { name: 'General Admission', price: 2000, total_seats: 5000, description: 'Roaming access' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Anime Festival Asia Bangkok',
      description: 'Southeast Asia largest anime convention featuring cosplay, merchandise, and guest appearances.',
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
      { name: 'VIP Pass', price: 3500, total_seats: 500, description: '2-day + exclusive merch' },
      { name: 'Day Pass', price: 1200, total_seats: 3000, description: 'Single day access' },
      { name: 'Concert Only', price: 2500, total_seats: 1000, description: 'Evening concert only' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'World Barista Championship',
      description: 'The world best baristas compete for the championship title in Bangkok.',
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
      { name: 'VIP Tasting', price: 4500, total_seats: 100, description: 'Tasting sessions + finals' },
      { name: 'Full Access', price: 2500, total_seats: 500, description: 'All competitions access' },
      { name: 'Finals Only', price: 1200, total_seats: 800, description: 'Finals viewing' }
    ]
  },
  {
    status: 'UPCOMING',
    event: {
      name: 'Dua Lipa Future Nostalgia Tour',
      description: 'Pop superstar Dua Lipa brings her electrifying Future Nostalgia Tour to Bangkok.',
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
      { name: 'VIP Standing', price: 9500, total_seats: 800, description: 'Front of stage standing' },
      { name: 'CAT 1', price: 7000, total_seats: 1500, description: 'Premium seating' },
      { name: 'CAT 2', price: 5000, total_seats: 2000, description: 'Standard seating' },
      { name: 'CAT 3', price: 3000, total_seats: 2500, description: 'Value seating' }
    ]
  },

  // === ENDED Events (5) ===
  {
    status: 'ENDED',
    event: {
      name: 'Harry Styles Love On Tour',
      description: 'Harry Styles brought his Love On Tour to Bangkok for two magical nights.',
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
      { name: 'Golden Circle', price: 9500, total_seats: 500, description: 'Standing front stage' },
      { name: 'CAT 1', price: 6500, total_seats: 1500, description: 'Premium seating' },
      { name: 'CAT 2', price: 4000, total_seats: 2500, description: 'Standard seating' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'World Travel Fair Bangkok',
      description: 'Annual travel exhibition featuring deals from airlines, hotels, and tour operators.',
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
      description: 'Annual Bangkok Marathon with full marathon, half marathon, and fun run categories.',
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
      { name: 'Full Marathon', price: 1800, total_seats: 5000, description: '42.195km race' },
      { name: 'Half Marathon', price: 1500, total_seats: 8000, description: '21.1km race' },
      { name: 'Fun Run 10K', price: 800, total_seats: 10000, description: '10km fun run' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Thailand Game Show 2025',
      description: 'Southeast Asia biggest gaming expo featuring new game releases, esports, and cosplay.',
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
      { name: 'VIP Pass', price: 2500, total_seats: 500, description: '2-day + exclusive booth' },
      { name: 'Day Pass', price: 450, total_seats: 10000, description: 'Single day access' }
    ]
  },
  {
    status: 'ENDED',
    event: {
      name: 'Oktoberfest Bangkok 2025',
      description: 'German beer festival celebration with authentic Bavarian food, music, and entertainment.',
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
      { name: 'Premium', price: 1500, total_seats: 500, description: 'Premium area + 2 beers' },
      { name: 'General', price: 500, total_seats: 2000, description: 'General admission' }
    ]
  }
];

async function main() {
  console.log('=== Extra Events Seed Script (20 Events) ===');
  console.log(`API URL: ${API_BASE_URL}`);
  console.log(`User: ${ADMIN_EMAIL}`);
  console.log('');

  await login();
  console.log('');
  console.log('=== Creating 20 Events ===');
  console.log('');

  let created = 0;

  for (let i = 0; i < events.length; i++) {
    const { status, event, shows, zones } = events[i];
    const statusColor = status === 'OPEN' ? 'green' : status === 'UPCOMING' ? 'blue' : 'red';

    log(statusColor, `[${i + 1}/20] ${status} - ${event.name}`);

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
  log('green', '  OPEN (10): Taylor Swift, UFC, Disney On Ice, Bruno Mars, Cirque du Soleil,');
  log('green', '             ONE Championship, Wine Festival, Marvel Live, Art Gala, Yoga Retreat');
  log('blue', '  UPCOMING (5): Blackpink, Formula E, Anime Festival, Barista Championship, Dua Lipa');
  log('red', '  ENDED (5): Harry Styles, Travel Fair, Marathon, Game Show, Oktoberfest');
}

main().catch(err => {
  console.error('Error:', err);
  process.exit(1);
});
