-- Text columns are NVARCHAR because go-mssqldb sends Go strings as
-- nvarchar; comparing them with VARCHAR columns would force an implicit
-- conversion and skip the index.

-- One row per checkout. Online payments start as AWAITING_PAYMENT (books
-- reserved in the catalogue, hidden from the customer) and only become an
-- order once the gateway confirms; cash on delivery is CONFIRMED at once.
CREATE TABLE orders (
    id               BIGINT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_orders PRIMARY KEY,
    -- Random public code; guests look an order up by code + email.
    code             NVARCHAR(16)          NOT NULL CONSTRAINT uq_orders_code UNIQUE,
    -- Sent by the client so a repeated "Place order" returns the same row.
    idempotency_key  UNIQUEIDENTIFIER      NOT NULL CONSTRAINT uq_orders_idempotency_key UNIQUE,
    -- JWT subject forwarded by the gateway as X-User-ID; NULL for guests.
    user_id          NVARCHAR(64)          NULL,
    status           NVARCHAR(20)          NOT NULL
                                           CONSTRAINT ck_orders_status CHECK (status IN (N'AWAITING_PAYMENT', N'CONFIRMED', N'CANCELLED')),
    payment_method   NVARCHAR(10)          NOT NULL
                                           CONSTRAINT ck_orders_payment_method CHECK (payment_method IN (N'vietqr', N'momo', N'cod', N'card', N'paypal')),
    payment_status   NVARCHAR(10)          NOT NULL CONSTRAINT df_orders_payment_status DEFAULT N'UNPAID'
                                           CONSTRAINT ck_orders_payment_status CHECK (payment_status IN (N'UNPAID', N'PAID', N'REFUNDED')),
    email            NVARCHAR(254)         NOT NULL,
    phone            NVARCHAR(20)          NOT NULL,
    full_name        NVARCHAR(150)         NOT NULL,
    address          NVARCHAR(300)         NOT NULL,
    city             NVARCHAR(100)         NOT NULL,
    postal_code      NVARCHAR(20)          NULL,
    country          NVARCHAR(60)          NOT NULL,
    -- Read from the catalogue when the order is placed, never from the client.
    subtotal_vnd     BIGINT                NOT NULL CONSTRAINT ck_orders_subtotal_vnd CHECK (subtotal_vnd > 0),
    -- NULL until shipping is confirmed.
    shipping_fee_vnd BIGINT                NULL CONSTRAINT ck_orders_shipping_fee_vnd CHECK (shipping_fee_vnd >= 0),
    -- End of the payment hold; the reserved books are released after it.
    expires_at       DATETIME2             NULL,
    cancel_reason    NVARCHAR(200)         NULL,
    created_at       DATETIME2             NOT NULL CONSTRAINT df_orders_created_at DEFAULT SYSUTCDATETIME(),
    updated_at       DATETIME2             NOT NULL CONSTRAINT df_orders_updated_at DEFAULT SYSUTCDATETIME(),

    -- Cash on delivery never waits for a gateway.
    CONSTRAINT ck_orders_cod_not_awaiting CHECK (NOT (payment_method = N'cod' AND status = N'AWAITING_PAYMENT')),
    -- A payment hold always has an end.
    CONSTRAINT ck_orders_awaiting_expires CHECK (status <> N'AWAITING_PAYMENT' OR expires_at IS NOT NULL)
);

-- "My orders", newest first.
CREATE INDEX ix_orders_user_id_created_at ON orders (user_id, created_at DESC);
-- Finding payment holds that have run out.
CREATE INDEX ix_orders_status_expires_at ON orders (status, expires_at);

-- Every listing is a single copy, so there is no quantity. Book details are
-- a snapshot taken when the order is placed; book_id points to the catalogue
-- database, so there is no foreign key.
CREATE TABLE order_items (
    id        BIGINT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_order_items PRIMARY KEY,
    order_id  BIGINT                NOT NULL CONSTRAINT fk_order_items_order REFERENCES orders (id) ON DELETE CASCADE,
    book_id   INT                   NOT NULL,
    book_slug NVARCHAR(100)         NOT NULL,
    title     NVARCHAR(600)         NOT NULL,
    author    NVARCHAR(150)         NULL,
    cover_url NVARCHAR(300)         NULL,
    price_vnd BIGINT                NOT NULL CONSTRAINT ck_order_items_price_vnd CHECK (price_vnd > 0),

    CONSTRAINT uq_order_items_order_book UNIQUE (order_id, book_id)
);

CREATE INDEX ix_order_items_book_id ON order_items (book_id);
