-- Text columns are NVARCHAR because go-mssqldb sends Go strings as
-- nvarchar; comparing them with VARCHAR columns would force an implicit
-- conversion and skip the index.

-- One row per account.
CREATE TABLE users (
    -- UUIDv7 made by the service: the JWT subject, and orders.user_id in the
    -- order service. Time-ordered, so inserts append to the index.
    id                NVARCHAR(36)  NOT NULL CONSTRAINT pk_users PRIMARY KEY,
    -- Stored lowercased by the service, so the unique key is case-blind
    -- whatever the collation.
    email             NVARCHAR(254) NOT NULL CONSTRAINT uq_users_email UNIQUE,
    -- argon2id in the PHC string format; the parameters travel with the hash.
    password_hash     NVARCHAR(200) NOT NULL,
    name              NVARCHAR(100) NOT NULL,
    role              NVARCHAR(20)  NOT NULL CONSTRAINT df_users_role DEFAULT N'customer'
                                    CONSTRAINT ck_users_role CHECK (role IN (N'customer', N'staff')),
    -- NULL until the owner of the address confirms it.
    email_verified_at DATETIME2     NULL,
    created_at        DATETIME2     NOT NULL CONSTRAINT df_users_created_at DEFAULT SYSUTCDATETIME(),
    updated_at        DATETIME2     NOT NULL CONSTRAINT df_users_updated_at DEFAULT SYSUTCDATETIME()
);
