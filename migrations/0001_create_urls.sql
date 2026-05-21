CREATE TABLE IF NOT EXISTS urls (
    short_id VARCHAR(16) PRIMARY KEY,
    original_url TEXT NOT NULL
);