package user

import (
	"net/url"
	"time"

	"github.com/fizzisme/user-service/internal/email"
	"github.com/fizzisme/user-service/internal/outbox"
)

// Purposes of an email token (email_tokens.purpose).
const (
	PurposeVerifyEmail = "verify-email"
)

// verifyEmailTTL is how long a confirmation link works.
const verifyEmailTTL = 24 * time.Hour

// EmailToken is one email_tokens row: a one-time link sent to an address.
// Only the hash of the token is stored.
type EmailToken struct {
	ID        int64      `db:"id"`
	UserID    string     `db:"user_id"`
	Purpose   string     `db:"purpose"`
	Email     string     `db:"email"`
	TokenHash []byte     `db:"token_hash"`
	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
}

// verification returns a new confirmation link for u's address: the token
// to store and the email that carries it.
func (s *Service) verification(u User) (EmailToken, outbox.Message, error) {

	raw, hash, err := newSecret()
	if err != nil {
		return EmailToken{}, outbox.Message{}, err
	}

	link := s.webURL + "/verify-email?" + url.Values{"token": {raw}}.Encode()

	mail, err := email.VerifyEmail(u.Email, u.Name, link)
	if err != nil {
		return EmailToken{}, outbox.Message{}, err
	}
	msg, err := outbox.NewEmail(mail)
	if err != nil {
		return EmailToken{}, outbox.Message{}, err
	}

	return EmailToken{
		UserID:    u.ID,
		Purpose:   PurposeVerifyEmail,
		Email:     u.Email,
		TokenHash: hash,
		ExpiresAt: s.now().Add(verifyEmailTTL),
	}, msg, nil
}
