CREATE TABLE IF NOT EXISTS user_urls (
    user_id  VARCHAR(128) NOT NULL,
    short_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (user_id, short_id)
);

CREATE INDEX IF NOT EXISTS user_urls_user_id_idx ON user_urls(user_id);
