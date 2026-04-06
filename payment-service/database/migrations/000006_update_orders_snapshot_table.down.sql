ALTER TABLE orders_snapshot DROP COLUMN IF EXISTS order_date;
ALTER TABLE orders_snapshot DROP COLUMN IF EXISTS order_time;
ALTER TABLE orders_snapshot ADD COLUMN order_at TIMESTAMP;