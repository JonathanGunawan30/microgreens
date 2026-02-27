CREATE TABLE payment_logs (
    id SERIAL PRIMARY KEY,
    payment_id BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL
);

CREATE INDEX idx_payment_logs_payment_id ON payment_logs(payment_id);