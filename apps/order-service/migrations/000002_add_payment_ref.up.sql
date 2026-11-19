-- What the gateway tells us about a paid order. Both stay NULL for cash on
-- delivery and for online orders that were never paid.
ALTER TABLE orders ADD
    -- The gateway's own id of the payment (VNPay: vnp_TransactionNo), needed
    -- to look the payment up or refund it from the merchant portal.
    payment_ref NVARCHAR(50) NULL,
    -- When the gateway confirmed the payment.
    paid_at     DATETIME2    NULL;

-- golang-migrate sends this file as one batch, and SQL Server compiles the
-- whole batch before running it, so the new columns must be named inside EXEC.
-- A paid order always says when it was paid.
EXEC(N'ALTER TABLE orders ADD CONSTRAINT ck_orders_paid_at
    CHECK (payment_status <> N''PAID'' OR paid_at IS NOT NULL)');
