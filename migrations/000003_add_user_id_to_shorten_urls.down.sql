ALTER TABLE shorten_urls DROP COLUMN user_id;
DROP INDEX IF EXISTS idx_shorten_urls_user_id;
