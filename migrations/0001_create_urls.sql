CREATE TABLE IF NOT EXISTS urls (
<<<<<<< HEAD
    short_id VARCHAR(16) PRIMARY KEY,
=======
    id SERIAL PRIMARY KEY,
    short_id VARCHAR(32) NOT NULL UNIQUE,
>>>>>>> 23478c9 (apply previous review fixes)
    original_url TEXT NOT NULL
);
