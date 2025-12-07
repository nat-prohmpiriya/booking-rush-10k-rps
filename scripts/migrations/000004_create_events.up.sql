-- 000004_create_events.up.sql
-- Events table - the main entity for ticket booking

CREATE TYPE event_status AS ENUM ('draft', 'published', 'cancelled', 'completed');

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    organizer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    
    -- Basic info
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    short_description VARCHAR(500),
    
    -- Media
    poster_url TEXT,
    banner_url TEXT,
    gallery JSONB DEFAULT '[]', -- array of image URLs
    
    -- Location
    venue_name VARCHAR(255),
    venue_address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Booking settings
    max_tickets_per_user INT DEFAULT 10,
    booking_start_at TIMESTAMP WITH TIME ZONE,
    booking_end_at TIMESTAMP WITH TIME ZONE,
    
    -- Status and visibility
    status event_status DEFAULT 'draft',
    is_featured BOOLEAN DEFAULT false,
    is_public BOOLEAN DEFAULT true,
    
    -- SEO
    meta_title VARCHAR(255),
    meta_description VARCHAR(500),
    
    -- Additional settings
    settings JSONB DEFAULT '{}',
    
    -- Timestamps
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Unique slug per tenant
    CONSTRAINT unique_event_slug_per_tenant UNIQUE (tenant_id, slug)
);

-- Indexes for common queries
CREATE INDEX idx_events_tenant_id ON events(tenant_id);
CREATE INDEX idx_events_organizer_id ON events(organizer_id);
CREATE INDEX idx_events_category_id ON events(category_id);
CREATE INDEX idx_events_slug ON events(slug);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_is_featured ON events(is_featured) WHERE is_featured = true;
CREATE INDEX idx_events_is_public ON events(is_public) WHERE is_public = true;
CREATE INDEX idx_events_booking_dates ON events(booking_start_at, booking_end_at);
CREATE INDEX idx_events_city ON events(city);
CREATE INDEX idx_events_deleted_at ON events(deleted_at) WHERE deleted_at IS NULL;

-- Full text search index for name and description
CREATE INDEX idx_events_search ON events USING gin(to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- Trigger for updated_at
CREATE TRIGGER update_events_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
