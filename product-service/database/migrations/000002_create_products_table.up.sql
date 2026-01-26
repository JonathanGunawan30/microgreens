CREATE TYPE product_status_enum AS ENUM ('draft', 'active', 'inactive');

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    parent_id BIGINT NULL,
    category_slug VARCHAR(100) NOT NULL REFERENCES categories(slug),
    name VARCHAR(100) NOT NULL,
    image VARCHAR(255) NOT NULL,
    description TEXT NULL,
    reguler_price BIGINT DEFAULT 0,
    sale_price BIGINT DEFAULT 0,
    unit VARCHAR(100) DEFAULT 'gram',
    weight BIGINT DEFAULT 0,
    variant INT DEFAULT 1,
    status product_status_enum NOT NULL DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_status_products ON products(status);
CREATE INDEX idx_category_slug_products ON products(category_slug);
