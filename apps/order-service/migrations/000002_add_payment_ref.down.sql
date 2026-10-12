ALTER TABLE orders DROP CONSTRAINT IF EXISTS ck_orders_paid_at;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_ref, paid_at;
