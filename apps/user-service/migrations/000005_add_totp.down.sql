DROP TABLE IF EXISTS mfa_challenges;
ALTER TABLE users DROP COLUMN totp_secret_enc, totp_enabled_at, totp_last_step;
