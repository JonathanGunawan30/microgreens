ALTER TABLE payments ADD COLUMN order_code       VARCHAR(64)  DEFAULT '';
ALTER TABLE payments ADD COLUMN shipping_type    VARCHAR(20)  DEFAULT '';
ALTER TABLE payments ADD COLUMN order_at         TIMESTAMP;
ALTER TABLE payments ADD COLUMN order_remarks    TEXT         DEFAULT '';
ALTER TABLE payments ADD COLUMN customer_name    VARCHAR(255) DEFAULT '';
ALTER TABLE payments ADD COLUMN customer_email   VARCHAR(255) DEFAULT '';
ALTER TABLE payments ADD COLUMN customer_address TEXT         DEFAULT '';