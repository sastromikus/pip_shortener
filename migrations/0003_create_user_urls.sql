CREATE TABLE IF NOT EXISTS user_urls (
    user_id TEXT NOT NULL,
    short_id VARCHAR(16) NOT NULL REFERENCES urls(short_id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, short_id)
);
