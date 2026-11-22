CREATE TABLE sell_requests (
    id BIGINT IDENTITY(1, 1) NOT NULL
        CONSTRAINT pk_sell_requests PRIMARY KEY,

    title NVARCHAR(600) NOT NULL,
    author NVARCHAR(150) NOT NULL,
    year SMALLINT NULL,
    edition NVARCHAR(200) NULL,
    binding NVARCHAR(50) NOT NULL,
    grade NVARCHAR(30) NOT NULL,
    faults NVARCHAR(4000) NOT NULL,
    virtues NVARCHAR(4000) NULL,
    asking_price_vnd BIGINT NULL,

    seller_name NVARCHAR(150) NOT NULL,
    seller_email NVARCHAR(254) NOT NULL,
    seller_phone NVARCHAR(30) NULL,

    status NVARCHAR(20) NOT NULL
        CONSTRAINT df_sell_requests_status DEFAULT N'PENDING_REVIEW'
        CONSTRAINT ck_sell_requests_status
            CHECK (status IN (
                N'PENDING_REVIEW',
                N'APPROVED',
                N'REJECTED'
            )),

    created_at DATETIME2 NOT NULL
        CONSTRAINT df_sell_requests_created_at
        DEFAULT SYSUTCDATETIME(),

    updated_at DATETIME2 NOT NULL
        CONSTRAINT df_sell_requests_updated_at
        DEFAULT SYSUTCDATETIME(),

    CONSTRAINT ck_sell_requests_asking_price
        CHECK (asking_price_vnd IS NULL OR asking_price_vnd > 0)
);

CREATE INDEX ix_sell_requests_status_created_at
    ON sell_requests (status, created_at DESC);