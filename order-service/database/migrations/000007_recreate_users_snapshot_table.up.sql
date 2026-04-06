DROP TABLE IF EXISTS users_snapshot;
CREATE TABLE IF NOT EXISTS users_snapshot
(
    id         SERIAL PRIMARY KEY,
    user_id    BIGINT UNIQUE NOT NULL,
    name       VARCHAR(255),
    email      VARCHAR(255),
    phone      VARCHAR(17),
    address    TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);