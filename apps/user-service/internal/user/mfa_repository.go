package user

import (
	"context"
	"database/sql"
	"errors"
)

// SaveTOTPSecret keeps a new sealed secret while two-step sign-in is off.
func (r *Repository) SaveTOTPSecret(ctx context.Context, userID string, sealed []byte) error {

	res, err := r.db.ExecContext(ctx, `
UPDATE users SET totp_secret_enc = @sealed, totp_last_step = NULL, updated_at = SYSUTCDATETIME()
WHERE id = @id AND totp_enabled_at IS NULL`,
		sql.Named("id", userID), sql.Named("sealed", sealed))
	if err != nil {
		return err
	}

	if n, err := res.RowsAffected(); err != nil || n == 0 {
		if err == nil {
			err = ErrMFAAlreadyOn
		}
		return err
	}
	return nil
}

// EnableTOTP turns two-step sign-in on with new recovery codes.
func (r *Repository) EnableTOTP(ctx context.Context, userID string, step int64, recoveryHashes [][]byte) error {

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE users SET totp_enabled_at = SYSUTCDATETIME(), totp_last_step = @step, updated_at = SYSUTCDATETIME()
WHERE id = @id AND totp_secret_enc IS NOT NULL`,
		sql.Named("id", userID), sql.Named("step", step)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id = @id`, sql.Named("id", userID)); err != nil {
		return err
	}
	for _, h := range recoveryHashes {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO recovery_codes (user_id, code_hash) VALUES (@id, @hash)`,
			sql.Named("id", userID), sql.Named("hash", h)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DisableTOTP turns two-step sign-in off and forgets its secret and codes.
func (r *Repository) DisableTOTP(ctx context.Context, userID string) error {

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
UPDATE users SET totp_secret_enc = NULL, totp_enabled_at = NULL, totp_last_step = NULL, updated_at = SYSUTCDATETIME()
WHERE id = @id`, sql.Named("id", userID)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recovery_codes WHERE user_id = @id`, sql.Named("id", userID)); err != nil {
		return err
	}

	return tx.Commit()
}

// UseRecoveryCode marks an unused recovery code as used.
func (r *Repository) UseRecoveryCode(ctx context.Context, userID string, hash []byte) (bool, error) {

	res, err := r.db.ExecContext(ctx, `
UPDATE recovery_codes SET used_at = SYSUTCDATETIME()
WHERE user_id = @id AND code_hash = @hash AND used_at IS NULL`,
		sql.Named("id", userID), sql.Named("hash", hash))
	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()
	return n == 1, err
}

// UseTOTPStep records step as used, unless it or a later one was.
func (r *Repository) UseTOTPStep(ctx context.Context, userID string, step int64) (bool, error) {

	res, err := r.db.ExecContext(ctx, `
UPDATE users SET totp_last_step = @step
WHERE id = @id AND (totp_last_step IS NULL OR totp_last_step < @step)`,
		sql.Named("id", userID), sql.Named("step", step))
	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()
	return n == 1, err
}

// CreateMFAChallenge stores the code step of a sign-in.
func (r *Repository) CreateMFAChallenge(ctx context.Context, c MFAChallenge) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO mfa_challenges (user_id, token_hash, device_name, expires_at)
VALUES (@user_id, @token_hash, @device_name, @expires_at)`,
		sql.Named("user_id", c.UserID),
		sql.Named("token_hash", c.TokenHash),
		sql.Named("device_name", c.DeviceName),
		sql.Named("expires_at", c.ExpiresAt.UTC()),
	)
	return err
}

// FindMFAChallenge returns the challenge with a token hash, or ErrNotFound.
func (r *Repository) FindMFAChallenge(ctx context.Context, hash []byte) (MFAChallenge, error) {

	var c MFAChallenge
	err := r.db.GetContext(ctx, &c, `
SELECT id, user_id, token_hash, device_name, created_at, expires_at, used_at, attempts
FROM mfa_challenges WHERE token_hash = @hash`, sql.Named("hash", hash))
	if errors.Is(err, sql.ErrNoRows) {
		return MFAChallenge{}, ErrNotFound
	}
	return c, err
}

// CountMFAAttempt adds one try to an unused challenge and returns the tries.
func (r *Repository) CountMFAAttempt(ctx context.Context, id int64) (int, error) {

	var attempts int
	err := r.db.GetContext(ctx, &attempts, `
UPDATE mfa_challenges SET attempts = attempts + 1
OUTPUT inserted.attempts
WHERE id = @id AND used_at IS NULL`, sql.Named("id", id))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidChallenge
	}
	return attempts, err
}

// UseMFAChallenge marks a challenge done, unless it was already.
func (r *Repository) UseMFAChallenge(ctx context.Context, id int64) (bool, error) {

	res, err := r.db.ExecContext(ctx, `
UPDATE mfa_challenges SET used_at = SYSUTCDATETIME() WHERE id = @id AND used_at IS NULL`,
		sql.Named("id", id))
	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()
	return n == 1, err
}
