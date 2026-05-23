CREATE TABLE IF NOT EXISTS user_urls (
<<<<<<< HEAD
    user_id TEXT NOT NULL,
    short_id VARCHAR(16) NOT NULL REFERENCES urls(short_id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, short_id)
);
=======
    user_id  VARCHAR(128) NOT NULL,
    short_id VARCHAR(32) NOT NULL,
    PRIMARY KEY (user_id, short_id)
);

CREATE INDEX IF NOT EXISTS user_urls_user_id_idx ON user_urls(user_id);
>>>>>>> 23478c9 (apply previous review fixes)
