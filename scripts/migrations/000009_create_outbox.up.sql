-- 000009_create_outbox.up.sql
-- Transactional Outbox pattern for reliable event publishing

CREATE TYPE outbox_status AS ENUM (
    'pending',      -- Waiting to be published
    'published',    -- Successfully published to Kafka
    'failed'        -- Failed to publish (will retry)
);

CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Event details
    aggregate_type VARCHAR(100) NOT NULL, -- e.g., 'booking', 'payment'
    aggregate_id UUID NOT NULL,           -- ID of the related entity
    event_type VARCHAR(100) NOT NULL,     -- e.g., 'booking.created', 'payment.succeeded'
    
    -- Payload
    payload JSONB NOT NULL,
    
    -- Publishing metadata
    topic VARCHAR(100) NOT NULL,          -- Kafka topic
    partition_key VARCHAR(255),           -- For Kafka partitioning
    
    -- Status tracking
    status outbox_status DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 5,
    last_error TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    published_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for outbox processing
CREATE INDEX idx_outbox_status_pending ON outbox(created_at) 
    WHERE status = 'pending';

CREATE INDEX idx_outbox_status_failed ON outbox(created_at) 
    WHERE status = 'failed' AND retry_count < max_retries;

CREATE INDEX idx_outbox_aggregate ON outbox(aggregate_type, aggregate_id);

-- Index for cleanup of old published events
CREATE INDEX idx_outbox_published ON outbox(published_at) 
    WHERE status = 'published';
