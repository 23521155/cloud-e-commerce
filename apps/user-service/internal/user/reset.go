package user

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/fizzisme/user-service/internal/email"
	"github.com/fizzisme/user-service/internal/outbox"
)

// PurposeResetPassword is the email token purpose of a password reset link.
const PurposeResetPassword = "reset-password"

// resetPasswordTTL is how long a reset link works: short, as it is as good
// as the password.
const resetPasswordTTL = 30 * time.Minute

// resetStore is the password reset part of the repository.
type resetStore interface {
	// ResetPassword uses an unused, unexpired reset-password token with hash,
	// sets the account's password hash and revokes all its sessions, in one
	// transaction. ErrInvalidEmailToken when there is no such token.
	ResetPassword(ctx context.Context, hash []byte, passwordHash string) error
}

// RequestPasswordReset emails a reset link to the account with address, if
// there is one. The caller gets the same answer either way, so the form
// cannot tell who has an account; a second request within resendInterval
// sends nothing.
func (s *Service) RequestPasswordReset(ctx context.Context, address string) error {

	u, err := s.store.FindByEmail(ctx, normalizeEmail(address))
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	last, ok, err := s.store.LastEmailTokenAt(ctx, u.ID, PurposeResetPassword)
	if err != nil {
		return err
	}
	if ok && s.now().Sub(last) < resendInterval {
		return nil
	}

	raw, hash, err := newSecret()
	if err != nil {
		return err
	}

	link := s.webURL + "/reset-password?" + url.Values{"token": {raw}}.Encode()
	mail, err := email.ResetPassword(u.Email, u.Name, link)
	if err != nil {
		return err
	}
	msg, err := outbox.NewEmail(mail)
	if err != nil {
		return err
	}

	return s.store.CreateEmailToken(ctx, EmailToken{
		UserID:    u.ID,
		Purpose:   PurposeResetPassword,
		Email:     u.Email,
		TokenHash: hash,
		ExpiresAt: s.now().Add(resetPasswordTTL),
	}, msg)
}

// ResetPassword sets a new password with a reset link. Every session of the
// account ends: whoever knew the old password is signed out.
func (s *Service) ResetPassword(ctx context.Context, raw, password string) error {

	if err := validatePassword(password); err != nil {
		return err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	return s.store.ResetPassword(ctx, hashToken(raw), hash)
}
