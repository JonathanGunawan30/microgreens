DROP TABLE IF EXISTS products_snapshot;
CREATE TABLE IF NOT EXISTS products_snapshot
(
    id         SERIAL PRIMARY KEY,
    product_id BIGINT UNIQUE NOT NULL,
    name       VARCHAR(255),
    image      VARCHAR(255),
    sale_price BIGINT    DEFAULT 0,
    unit       VARCHAR(50),
    weight     BIGINT    DEFAULT 0,
    is_active  BOOLEAN   DEFAULT true,
    updated_at TIMESTAMP DEFAULT NOW()
);