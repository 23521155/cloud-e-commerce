-- Sign-in accounts from other providers linked to a user (Google for now).
-- An account made with Google alone has an empty password_hash: a password
-- sign-in never matches it, and "forgot password" can set one.
CREATE TABLE user_identities (
    -- "google"
    provider   NVARCHAR(20)  NOT NULL,
    -- The provider's id for the account (the ID token's sub): it never
    -- changes, unlike the email.
    subject    NVARCHAR(255) NOT NULL,
    user_id    NVARCHAR(36)  NOT NULL CONSTRAINT fk_user_identities_users REFERENCES users (id) ON DELETE CASCADE,
    created_at DATETIME2     NOT NULL CONSTRAINT df_user_identities_created_at DEFAULT SYSUTCDATETIME(),
    CONSTRAINT pk_user_identities PRIMARY KEY (provider, subject)
);

CREATE INDEX ix_user_identities_user_id ON user_identities (user_id);
