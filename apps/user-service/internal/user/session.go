package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/fizzisme/user-service/internal/token"
	"github.com/google/uuid"
)

// ErrInvalidRefreshToken means a refresh token is unknown, expired, revoked
// or was used before: the client has to sign in again.
var ErrInvalidRefreshToken = errors.New("invalid refresh token")

// reuseGrace is how long a used refresh token is still accepted. Two
// requests from one browser can carry the same token at the same time (two
// tabs, a prefetch); after the grace period a used token means it was
// copied.
const reuseGrace = 30 * time.Second

// RefreshToken is one refresh_tokens row. Only the hash of the token is
// stored.
type RefreshToken struct {
	ID         int64      `db:"id"`
	UserID     string     `db:"user_id"`
	FamilyID   string     `db:"family_id"`
	TokenHash  []byte     `db:"token_hash"`
	DeviceName string     `db:"device_name"`
	CreatedAt  time.Time  `db:"created_at"`
	ExpiresAt  time.Time  `db:"expires_at"`
	UsedAt     *time.Time `db:"used_at"`
	RevokedAt  *time.Time `db:"revoked_at"`
}

// Tokens are what signing in or refreshing gives the client.
type Tokens struct {
	Access token.Access

	// Refresh is exchanged at POST /auth/refresh for new tokens.
	Refresh          string
	RefreshExpiresAt time.Time
}

// sessionStore is the refresh token part of the repository.
type sessionStore interface {
	CreateRefreshToken(ctx context.Context, t RefreshToken) error
	FindRefreshToken(ctx context.Context, hash []byte) (RefreshToken, error)
	// MarkRefreshTokenUsed reports false when the token was already used.
	MarkRefreshTokenUsed(ctx context.Context, id int64) (bool, error)
	RevokeFamily(ctx context.Context, familyID string) error
	RevokeAllSessions(ctx context.Context, userID string) error
}

// newSecret returns a random token for a link or a cookie, and its hash.
func newSecret() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(b)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// issue signs an access token for u and adds a refresh token to family.
func (s *Service) issue(ctx context.Context, u User, familyID, device string) (Tokens, error) {

	access, err := s.tokens.Issue(token.Subject{ID: u.ID, Email: u.Email, Name: u.Name, Roles: []string{u.Role}})
	if err != nil {
		return Tokens{}, err
	}

	raw, hash, err := newSecret()
	if err != nil {
		return Tokens{}, err
	}

	expires := s.now().Add(s.refreshTTL)
	if err := s.store.CreateRefreshToken(ctx, RefreshToken{
		UserID:     u.ID,
		FamilyID:   familyID,
		TokenHash:  hash,
		DeviceName: truncate(device, 200),
		ExpiresAt:  expires,
	}); err != nil {
		return Tokens{}, err
	}

	return Tokens{Access: access, Refresh: raw, RefreshExpiresAt: expires}, nil
}

// startSession begins a new family: one signed-in device.
func (s *Service) startSession(ctx context.Context, u User, device string) (Tokens, error) {
	return s.issue(ctx, u, uuid.NewString(), device)
}

// Refresh exchanges a refresh token for new tokens in the same session. The
// access token is signed from the account as it is now, so a changed name
// or role shows up.
func (s *Service) Refresh(ctx context.Context, raw, device string) (User, Tokens, error) {

	t, err := s.store.FindRefreshToken(ctx, hashToken(raw))
	if errors.Is(err, ErrNotFound) {
		return User{}, Tokens{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return User{}, Tokens{}, err
	}

	now := s.now()
	if t.RevokedAt != nil || !now.Before(t.ExpiresAt) {
		return User{}, Tokens{}, ErrInvalidRefreshToken
	}

	usedBefore := t.UsedAt != nil
	if !usedBefore {
		marked, err := s.store.MarkRefreshTokenUsed(ctx, t.ID)
		if err != nil {
			return User{}, Tokens{}, err
		}
		// Another request used it a moment ago: same as a reuse within
		// the grace period.
		usedBefore = !marked
		if usedBefore {
			t.UsedAt = &now
		}
	}

	if usedBefore && now.Sub(*t.UsedAt) > reuseGrace {
		// A used token came back: someone else may hold a copy. End the
		// session on every holder.
		if err := s.store.RevokeFamily(ctx, t.FamilyID); err != nil {
			return User{}, Tokens{}, err
		}
		return User{}, Tokens{}, ErrInvalidRefreshToken
	}

	u, err := s.store.FindByID(ctx, t.UserID)
	if errors.Is(err, ErrNotFound) {
		return User{}, Tokens{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return User{}, Tokens{}, err
	}

	tokens, err := s.issue(ctx, u, t.FamilyID, device)
	if err != nil {
		return User{}, Tokens{}, err
	}

	return u, tokens, nil
}

// Logout ends the session a refresh token belongs to. An unknown token is
// not an error: the session is over either way.
func (s *Service) Logout(ctx context.Context, raw string) error {

	t, err := s.store.FindRefreshToken(ctx, hashToken(raw))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	return s.store.RevokeFamily(ctx, t.FamilyID)
}

// truncate cuts s to at most n characters.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
