CREATE TABLE notifications(
    id SERIAL PRIMARY KEY,
    notification_type VARCHAR(50) NOT NULL,
    receiver_id INT NULL,
    receiver_email VARCHAR(255) NULL,
    subject VARCHAR(255) NULL,
    message TEXT NOT NULL,
    status VARCHAR(50) NULL,
    send_at TIMESTAMP NULL,
    read_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
)