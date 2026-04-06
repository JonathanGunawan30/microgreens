ALTER TABLE order_items ADD COLUMN IF NOT EXISTS product_name VARCHAR(255) DEFAULT '';
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS product_image VARCHAR(255) DEFAULT '';
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS price BIGINT DEFAULT 0;
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS product_unit VARCHAR(50) DEFAULT '';
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS product_weight BIGINT DEFAULT 0;

CREATE TABLE products_snapshot (
    product_id BIGINT UNIQUE NOT NULL,
    name VARCHAR(255),
    image VARCHAR(255),
    sale_price BIGINT,
    unit VARCHAR(50),
    weight BIGINT,
    is_active BOOLEAN DEFAULT true,
    updated_at TIMESTAMP DEFAULT NOW()
)