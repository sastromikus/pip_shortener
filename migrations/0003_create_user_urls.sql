-- +goose Up
CREATE TABLE IF NOT EXISTS user_urls (
    user_id TEXT NOT NULL,
    short_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (user_id, short_id)
);

CREATE INDEX IF NOT EXISTS user_urls_user_id_idx ON user_urls(user_id);

-- +goose Down
DROP TABLE IF EXISTS user_urls;
