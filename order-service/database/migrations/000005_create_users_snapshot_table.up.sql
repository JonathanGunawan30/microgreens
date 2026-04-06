CREATE TABLE users_snapshot
(
    user_id    BIGINT PRIMARY KEY,
    name       VARCHAR(255),
    email      VARCHAR(255),
    phone      VARCHAR(17),
    address    TEXT,
    updated_at TIMESTAMP
);