-- One row per refresh token. Signing in starts a family (one device's
-- session); every refresh uses up its token and adds the next one to the
-- family. A used token shown again after a short grace period means it was
-- copied, and the whole family is revoked.
CREATE TABLE refresh_tokens (
    id          BIGINT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_refresh_tokens PRIMARY KEY,
    user_id     NVARCHAR(36)  NOT NULL CONSTRAINT fk_refresh_tokens_user REFERENCES users (id) ON DELETE CASCADE,
    family_id   NVARCHAR(36)  NOT NULL,
    -- SHA-256 of the token; the token itself is only ever in the client's cookie.
    token_hash  BINARY(32)    NOT NULL CONSTRAINT uq_refresh_tokens_token_hash UNIQUE,
    -- From the gateway's X-Device-Name (browser and OS), to tell sessions apart.
    device_name NVARCHAR(200) NOT NULL CONSTRAINT df_refresh_tokens_device_name DEFAULT N'',
    created_at  DATETIME2     NOT NULL CONSTRAINT df_refresh_tokens_created_at DEFAULT SYSUTCDATETIME(),
    expires_at  DATETIME2     NOT NULL,
    -- Set when the token is exchanged for the next one.
    used_at     DATETIME2     NULL,
    -- Set on sign-out, password change or detected reuse, for the whole family.
    revoked_at  DATETIME2     NULL
);

-- Revoking one session, and every session of an account.
CREATE INDEX ix_refresh_tokens_family_id ON refresh_tokens (family_id);
CREATE INDEX ix_refresh_tokens_user_id ON refresh_tokens (user_id);
