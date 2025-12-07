-- 000003_create_categories.down.sql

DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
DROP TABLE IF EXISTS categories;
