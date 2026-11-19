package user

import (
	"context"
	"errors"
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

// resendInterval is the least time between two confirmation emails to one
// account, so the button cannot be used to flood an inbox.
const resendInterval = time.Minute

var (
	// ErrInvalidEmailToken means a link is unknown, used or expired.
	ErrInvalidEmailToken = errors.New("invalid or expired link")

	// ErrAlreadyVerified means the account's address is confirmed already.
	ErrAlreadyVerified = errors.New("email already verified")

	// ErrTooSoon means an email of the same kind went out a moment ago.
	ErrTooSoon = errors.New("an email was sent a moment ago")
)

// verificationStore is the email token part of the repository.
type verificationStore interface {
	// CreateEmailToken stores t with the messages that send it, in one
	// transaction.
	CreateEmailToken(ctx context.Context, t EmailToken, messages ...outbox.Message) error

	// LastEmailTokenAt is when the newest token of a purpose was made for
	// an account; ok is false when there is none.
	LastEmailTokenAt(ctx context.Context, userID, purpose string) (at time.Time, ok bool, err error)

	// VerifyEmail uses an unused, unexpired verify-email token with hash and
	// marks the address it was sent to as confirmed, if the account still
	// has that address. events gives the outbox messages to write in the
	// same transaction, for the confirmed account. ErrInvalidEmailToken
	// when there is no such token.
	VerifyEmail(ctx context.Context, hash []byte, events func(User) ([]outbox.Message, error)) (User, error)
}

// VerifyEmail confirms the address a link was sent to, and tells the order
// service in the same transaction, so the orders placed as a guest with this
// address can join the account. The address is only trusted once its owner
// has confirmed it: otherwise anyone could register someone else's email
// and see their orders.
func (s *Service) VerifyEmail(ctx context.Context, raw string) (User, error) {
	return s.store.VerifyEmail(ctx, hashToken(raw), func(u User) ([]outbox.Message, error) {
		msg, err := outbox.NewEmailVerified(u.ID, u.Email, *u.EmailVerifiedAt)
		if err != nil {
			return nil, err
		}
		return []outbox.Message{msg}, nil
	})
}

// ResendVerification sends a new confirmation link to the account's address.
// Earlier links keep working until they expire.
func (s *Service) ResendVerification(ctx context.Context, id string) error {

	u, err := s.store.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if u.EmailVerifiedAt != nil {
		return ErrAlreadyVerified
	}

	last, ok, err := s.store.LastEmailTokenAt(ctx, id, PurposeVerifyEmail)
	if err != nil {
		return err
	}
	if ok && s.now().Sub(last) < resendInterval {
		return ErrTooSoon
	}

	verify, msg, err := s.verification(u)
	if err != nil {
		return err
	}

	return s.store.CreateEmailToken(ctx, verify, msg)
}

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
