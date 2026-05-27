CREATE TABLE IF NOT EXISTS user_urls (
    user_id TEXT NOT NULL,
    short_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (user_id, short_id)
);