-- 000010_create_audit_logs.down.sql

-- Drop partitions (list current month and next 3 months by pattern)
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT tablename FROM pg_tables 
             WHERE schemaname = 'public' 
             AND tablename LIKE 'audit_logs_%'
    LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename);
    END LOOP;
END $$;

DROP FUNCTION IF EXISTS ensure_audit_logs_partitions();
DROP FUNCTION IF EXISTS create_audit_logs_partition(DATE);
DROP TABLE IF EXISTS audit_logs;
DROP TYPE IF EXISTS audit_action;
