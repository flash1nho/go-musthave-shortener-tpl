ALTER TABLE shorten_urls ADD COLUMN user_id VARCHAR(255);
CREATE INDEX idx_shorten_urls_user_id ON shorten_urls(user_id);
