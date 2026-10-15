package user

import (
	"context"
	"database/sql"
	"errors"
)

// refreshTokenColumns are the columns read into a RefreshToken.
const refreshTokenColumns = `
id, user_id, family_id, token_hash, device_name, created_at, expires_at, used_at, revoked_at, amr`

// CreateRefreshToken stores a new refresh token.
func (r *Repository) CreateRefreshToken(ctx context.Context, t RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO refresh_tokens (user_id, family_id, token_hash, device_name, expires_at, amr)
VALUES (@user_id, @family_id, @token_hash, @device_name, @expires_at, @amr)`,
		sql.Named("user_id", t.UserID),
		sql.Named("family_id", t.FamilyID),
		sql.Named("token_hash", t.TokenHash),
		sql.Named("device_name", t.DeviceName),
		sql.Named("expires_at", t.ExpiresAt.UTC()),
		sql.Named("amr", t.AMR),
	)
	return err
}

// FindRefreshToken returns the token with a hash, or ErrNotFound.
func (r *Repository) FindRefreshToken(ctx context.Context, hash []byte) (RefreshToken, error) {

	var t RefreshToken
	err := r.db.GetContext(ctx, &t,
		`SELECT `+refreshTokenColumns+` FROM refresh_tokens WHERE token_hash = @hash`,
		sql.Named("hash", hash))
	if errors.Is(err, sql.ErrNoRows) {
		return RefreshToken{}, ErrNotFound
	}
	return t, err
}

// MarkRefreshTokenUsed sets used_at on an unused token. It reports false
// when another request used the token first.
func (r *Repository) MarkRefreshTokenUsed(ctx context.Context, id int64) (bool, error) {

	res, err := r.db.ExecContext(ctx, `
UPDATE refresh_tokens SET used_at = SYSUTCDATETIME()
WHERE id = @id AND used_at IS NULL`, sql.Named("id", id))
	if err != nil {
		return false, err
	}

	n, err := res.RowsAffected()
	return n == 1, err
}

// RevokeFamily revokes every token of one session.
func (r *Repository) RevokeFamily(ctx context.Context, familyID string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE refresh_tokens SET revoked_at = SYSUTCDATETIME()
WHERE family_id = @family_id AND revoked_at IS NULL`, sql.Named("family_id", familyID))
	return err
}

// RevokeAllSessions revokes every refresh token of one account.
func (r *Repository) RevokeAllSessions(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE refresh_tokens SET revoked_at = SYSUTCDATETIME()
WHERE user_id = @user_id AND revoked_at IS NULL`, sql.Named("user_id", userID))
	return err
}
