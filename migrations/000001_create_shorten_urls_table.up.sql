CREATE TABLE shorten_urls (
    id SERIAL PRIMARY KEY,
    original_url VARCHAR(255) NOT NULL,
    short_url VARCHAR(255) NOT NULL
);

CREATE INDEX idx_shorten_urls_original_url_and_short_url ON shorten_urls(original_url, short_url);
