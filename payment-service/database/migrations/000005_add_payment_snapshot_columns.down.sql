ALTER TABLE payments DROP COLUMN IF EXISTS order_code;
ALTER TABLE payments DROP COLUMN IF EXISTS shipping_type;
ALTER TABLE payments DROP COLUMN IF EXISTS order_at;
ALTER TABLE payments DROP COLUMN IF EXISTS order_remarks;
ALTER TABLE payments DROP COLUMN IF EXISTS customer_name;
ALTER TABLE payments DROP COLUMN IF EXISTS customer_email;
ALTER TABLE payments DROP COLUMN IF EXISTS customer_address;