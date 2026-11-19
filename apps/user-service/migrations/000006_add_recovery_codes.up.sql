-- One-time codes that stand in for the authenticator app when it is lost.
-- Shown once when two-step sign-in is turned on; only the SHA-256 is kept.
CREATE TABLE recovery_codes (
    id        BIGINT IDENTITY(1, 1) NOT NULL CONSTRAINT pk_recovery_codes PRIMARY KEY,
    user_id   NVARCHAR(36) NOT NULL CONSTRAINT fk_recovery_codes_user REFERENCES users (id) ON DELETE CASCADE,
    code_hash BINARY(32)   NOT NULL,
    used_at   DATETIME2    NULL,
    CONSTRAINT uq_recovery_codes_user_code UNIQUE (user_id, code_hash)
);
