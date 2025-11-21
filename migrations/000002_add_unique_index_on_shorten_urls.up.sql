DROP INDEX IF EXISTS idx_shorten_urls_original_url_and_short_url;

CREATE UNIQUE INDEX idx_shorten_urls_original_url ON shorten_urls(original_url);
