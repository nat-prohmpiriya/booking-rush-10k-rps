-- Saga instances table for orchestrator pattern
CREATE TABLE IF NOT EXISTS saga_instances (
    id UUID PRIMARY KEY,
    definition_id VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    data JSONB NOT NULL DEFAULT '{}',
    step_results JSONB NOT NULL DEFAULT '[]',
    current_step INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Index for status queries (important for GetByStatus and GetPendingCompensations)
CREATE INDEX IF NOT EXISTS idx_saga_instances_status ON saga_instances(status);

-- Index for definition_id queries
CREATE INDEX IF NOT EXISTS idx_saga_instances_definition_id ON saga_instances(definition_id);

-- Index for finding incomplete sagas
CREATE INDEX IF NOT EXISTS idx_saga_instances_incomplete
    ON saga_instances(status)
    WHERE status IN ('pending', 'running', 'failed', 'compensating');

-- Index for created_at for ordering and cleanup
CREATE INDEX IF NOT EXISTS idx_saga_instances_created_at ON saga_instances(created_at);

-- Saga transitions table for audit trail
CREATE TABLE IF NOT EXISTS saga_transitions (
    id UUID PRIMARY KEY,
    saga_id UUID NOT NULL REFERENCES saga_instances(id) ON DELETE CASCADE,
    from_status VARCHAR(20) NOT NULL,
    to_status VARCHAR(20) NOT NULL,
    step_name VARCHAR(100),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for saga_id queries
CREATE INDEX IF NOT EXISTS idx_saga_transitions_saga_id ON saga_transitions(saga_id);

-- Dead letter queue table for failed messages
CREATE TABLE IF NOT EXISTS saga_dead_letters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    saga_id UUID,
    topic VARCHAR(255) NOT NULL,
    message_key VARCHAR(255),
    message_value JSONB NOT NULL,
    error_message TEXT NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    processed BOOLEAN NOT NULL DEFAULT FALSE
);

-- Index for unprocessed dead letters
CREATE INDEX IF NOT EXISTS idx_saga_dead_letters_unprocessed
    ON saga_dead_letters(processed, created_at)
    WHERE processed = FALSE;
