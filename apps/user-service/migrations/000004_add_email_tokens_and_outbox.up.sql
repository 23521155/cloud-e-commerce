-- One-time links sent by email: confirming an address, resetting a password.
-- Only the SHA-256 of the token is kept; the token is in the email.
CREATE TABLE email_tokens (
    id         BIGINT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_email_tokens PRIMARY KEY,
    user_id    NVARCHAR(36)  NOT NULL CONSTRAINT fk_email_tokens_user REFERENCES users (id) ON DELETE CASCADE,
    purpose    NVARCHAR(20)  NOT NULL
                             CONSTRAINT ck_email_tokens_purpose CHECK (purpose IN (N'verify-email', N'reset-password')),
    -- The address the link was sent to: confirming it confirms this address.
    email      NVARCHAR(254) NOT NULL,
    token_hash BINARY(32)    NOT NULL CONSTRAINT uq_email_tokens_token_hash UNIQUE,
    created_at DATETIME2     NOT NULL CONSTRAINT df_email_tokens_created_at DEFAULT SYSUTCDATETIME(),
    expires_at DATETIME2     NOT NULL,
    used_at    DATETIME2     NULL
);

CREATE INDEX ix_email_tokens_user_id ON email_tokens (user_id, purpose);

-- Messages for Azure Storage queues, written in the same transaction as the
-- change they report and deleted once sent (internal/outbox).
CREATE TABLE outbox (
    id           BIGINT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_outbox PRIMARY KEY,
    queue        NVARCHAR(63)  NOT NULL,
    payload      NVARCHAR(MAX) NOT NULL,
    created_at   DATETIME2     NOT NULL CONSTRAINT df_outbox_created_at DEFAULT SYSUTCDATETIME(),
    -- Set while a relay is sending the row; a row past it is claimed again.
    locked_until DATETIME2     NULL,
    attempts     INT           NOT NULL CONSTRAINT df_outbox_attempts DEFAULT 0
);
