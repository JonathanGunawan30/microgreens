ALTER TABLE order_items DROP COLUMN product_name VARCHAR(255) DEFAULT '';
ALTER TABLE order_items DROP COLUMN product_image VARCHAR(255) DEFAULT '';
ALTER TABLE order_items DROP COLUMN price BIGINT DEFAULT 0;
ALTER TABLE order_items DROP COLUMN product_unit VARCHAR(50) DEFAULT '';
ALTER TABLE order_items DROP COLUMN product_weight BIGINT DEFAULT 0;

DROP TABLE IF EXISTS products_snapshot;