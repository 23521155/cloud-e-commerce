-- Two-step sign-in with an authenticator app (TOTP, RFC 6238).
ALTER TABLE users ADD
    -- The shared secret, encrypted with AES-256-GCM (MFA_ENCRYPTION_KEY): a
    -- copy of the database alone cannot produce codes. Set when setup starts.
    totp_secret_enc VARBINARY(128) NULL,
    -- Set once a first code has proved the app holds the secret; sign-in asks
    -- for a code from then on.
    totp_enabled_at DATETIME2      NULL,
    -- The 30-second step of the last code accepted: a code is good once.
    totp_last_step  BIGINT         NULL;

-- The second step of a sign-in: the password was right, a code is due.
CREATE TABLE mfa_challenges (
    id          BIGINT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_mfa_challenges PRIMARY KEY,
    user_id     NVARCHAR(36)  NOT NULL CONSTRAINT fk_mfa_challenges_user REFERENCES users (id) ON DELETE CASCADE,
    token_hash  BINARY(32)    NOT NULL CONSTRAINT uq_mfa_challenges_token_hash UNIQUE,
    device_name NVARCHAR(200) NOT NULL CONSTRAINT df_mfa_challenges_device_name DEFAULT N'',
    created_at  DATETIME2     NOT NULL CONSTRAINT df_mfa_challenges_created_at DEFAULT SYSUTCDATETIME(),
    expires_at  DATETIME2     NOT NULL,
    used_at     DATETIME2     NULL,
    -- Codes tried; the challenge stops working after a few wrong ones.
    attempts    INT           NOT NULL CONSTRAINT df_mfa_challenges_attempts DEFAULT 0
);
