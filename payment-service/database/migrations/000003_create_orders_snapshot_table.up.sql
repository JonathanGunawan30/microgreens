CREATE TABLE orders_snapshot
(
    id            SERIAL PRIMARY KEY,
    order_id      BIGINT UNIQUE NOT NULL,
    order_code    VARCHAR(64),
    total_amount  NUMERIC(10, 2) DEFAULT 0,
    shipping_type VARCHAR(20),
    remarks       TEXT,
    order_at      TIMESTAMP,
    updated_at    TIMESTAMP      DEFAULT NOW()
);
