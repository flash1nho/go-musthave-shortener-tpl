DROP INDEX IF EXISTS idx_shorten_urls_original_url;

CREATE INDEX idx_shorten_urls_original_url_and_short_url ON shorten_urls(original_url, short_url);
