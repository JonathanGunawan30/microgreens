CREATE TABLE users_snapshot
(
    id         SERIAL PRIMARY KEY,
    user_id    BIGINT UNIQUE NOT NULL,
    name       VARCHAR(255),
    email      VARCHAR(255),
    address    TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);