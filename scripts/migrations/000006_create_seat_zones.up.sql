-- 000006_create_seat_zones.up.sql
-- Seat zones/ticket types for a show

CREATE TABLE IF NOT EXISTS seat_zones (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    
    -- Zone info
    name VARCHAR(100) NOT NULL, -- e.g., "VIP", "Standard", "Standing"
    description TEXT,
    color VARCHAR(7), -- hex color for UI display
    
    -- Pricing
    price DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'THB',
    
    -- Capacity tracking (source of truth in Redis, synced here)
    total_seats INT NOT NULL,
    available_seats INT NOT NULL,
    reserved_seats INT DEFAULT 0,
    sold_seats INT DEFAULT 0,
    
    -- Limits
    min_per_order INT DEFAULT 1,
    max_per_order INT DEFAULT 10,
    
    -- Visibility and status
    is_active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    
    -- Sales period (can override show's sale dates)
    sale_start_at TIMESTAMP WITH TIME ZONE,
    sale_end_at TIMESTAMP WITH TIME ZONE,
    
    -- Additional attributes (e.g., seat map configuration)
    attributes JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_seat_zones_show_id ON seat_zones(show_id);
CREATE INDEX idx_seat_zones_is_active ON seat_zones(is_active) WHERE is_active = true;
CREATE INDEX idx_seat_zones_sort_order ON seat_zones(show_id, sort_order);
CREATE INDEX idx_seat_zones_price ON seat_zones(price);
CREATE INDEX idx_seat_zones_deleted_at ON seat_zones(deleted_at) WHERE deleted_at IS NULL;

-- Index for available zones query
CREATE INDEX idx_seat_zones_available ON seat_zones(show_id, available_seats) 
    WHERE is_active = true AND available_seats > 0 AND deleted_at IS NULL;

-- Trigger for updated_at
CREATE TRIGGER update_seat_zones_updated_at
    BEFORE UPDATE ON seat_zones
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to update show's total_capacity when zones are added/modified
CREATE OR REPLACE FUNCTION update_show_capacity()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        UPDATE shows 
        SET total_capacity = (
            SELECT COALESCE(SUM(total_seats), 0) 
            FROM seat_zones 
            WHERE show_id = OLD.show_id AND deleted_at IS NULL
        )
        WHERE id = OLD.show_id;
        RETURN OLD;
    ELSE
        UPDATE shows 
        SET total_capacity = (
            SELECT COALESCE(SUM(total_seats), 0) 
            FROM seat_zones 
            WHERE show_id = NEW.show_id AND deleted_at IS NULL
        )
        WHERE id = NEW.show_id;
        RETURN NEW;
    END IF;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_show_capacity_trigger
    AFTER INSERT OR UPDATE OR DELETE ON seat_zones
    FOR EACH ROW
    EXECUTE FUNCTION update_show_capacity();
