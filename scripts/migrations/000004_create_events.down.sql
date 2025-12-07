-- 000004_create_events.down.sql

DROP TRIGGER IF EXISTS update_events_updated_at ON events;
DROP TABLE IF EXISTS events;
DROP TYPE IF EXISTS event_status;
