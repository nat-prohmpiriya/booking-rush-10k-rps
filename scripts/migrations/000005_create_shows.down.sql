-- 000005_create_shows.down.sql

DROP TRIGGER IF EXISTS update_shows_updated_at ON shows;
DROP TABLE IF EXISTS shows;
DROP TYPE IF EXISTS show_status;
