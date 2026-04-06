ALTER TABLE orders_snapshot DROP COLUMN IF EXISTS order_at;
ALTER TABLE orders_snapshot ADD COLUMN order_date VARCHAR(20) DEFAULT '';
ALTER TABLE orders_snapshot ADD COLUMN order_time VARCHAR(20) DEFAULT '';