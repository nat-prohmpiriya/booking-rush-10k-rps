-- 000010_create_audit_logs.up.sql
-- Audit logs table with monthly partitioning for efficient storage and queries

CREATE TYPE audit_action AS ENUM (
    'create',
    'update', 
    'delete',
    'login',
    'logout',
    'reserve',
    'confirm',
    'cancel',
    'refund',
    'view'
);

-- Create partitioned parent table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID DEFAULT uuid_generate_v4(),
    
    -- Actor information
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_email VARCHAR(255),
    user_role VARCHAR(50),
    
    -- Action details
    action audit_action NOT NULL,
    resource_type VARCHAR(100) NOT NULL, -- e.g., 'booking', 'event', 'user'
    resource_id UUID,
    
    -- Request context
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    trace_id VARCHAR(255),
    
    -- Change details
    old_values JSONB,
    new_values JSONB,
    changes JSONB, -- Computed diff
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    -- Timestamp (used for partitioning)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    
    -- Primary key includes partition key
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create indexes on parent table (will be inherited by partitions)
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);
CREATE INDEX idx_audit_logs_trace_id ON audit_logs(trace_id);

-- Function to create monthly partition
CREATE OR REPLACE FUNCTION create_audit_logs_partition(partition_date DATE)
RETURNS TEXT AS $$
DECLARE
    partition_start DATE;
    partition_end DATE;
    partition_name TEXT;
BEGIN
    partition_start := DATE_TRUNC('month', partition_date);
    partition_end := partition_start + INTERVAL '1 month';
    partition_name := 'audit_logs_' || TO_CHAR(partition_start, 'YYYY_MM');
    
    -- Check if partition already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname = partition_name
        AND n.nspname = 'public'
    ) THEN
        EXECUTE format(
            'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs
            FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            partition_start,
            partition_end
        );
        RETURN partition_name || ' created';
    ELSE
        RETURN partition_name || ' already exists';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create partitions for current month and next 3 months
SELECT create_audit_logs_partition(NOW()::DATE);
SELECT create_audit_logs_partition((NOW() + INTERVAL '1 month')::DATE);
SELECT create_audit_logs_partition((NOW() + INTERVAL '2 months')::DATE);
SELECT create_audit_logs_partition((NOW() + INTERVAL '3 months')::DATE);

-- Function to automatically create future partitions (run via cron/scheduler)
CREATE OR REPLACE FUNCTION ensure_audit_logs_partitions()
RETURNS TEXT AS $$
DECLARE
    result TEXT := '';
BEGIN
    -- Create partition for next month if not exists
    result := result || create_audit_logs_partition((NOW() + INTERVAL '1 month')::DATE) || '; ';
    result := result || create_audit_logs_partition((NOW() + INTERVAL '2 months')::DATE) || '; ';
    result := result || create_audit_logs_partition((NOW() + INTERVAL '3 months')::DATE);
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Comment for maintenance
COMMENT ON TABLE audit_logs IS 'Partitioned by month. Run ensure_audit_logs_partitions() monthly to create future partitions. Old partitions can be dropped after retention period.';
