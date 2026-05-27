CREATE TABLE IF NOT EXISTS user_urls (
    user_id TEXT NOT NULL,
    short_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (user_id, short_id),
    CONSTRAINT user_urls_short_id_fk
        FOREIGN KEY (short_id)
        REFERENCES urls (short_id)
        ON DELETE CASCADE
);