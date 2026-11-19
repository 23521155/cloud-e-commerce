package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fizzisme/user-service/internal/google"
	"github.com/fizzisme/user-service/internal/outbox"
	"github.com/google/uuid"
)

// ProviderGoogle is the provider of identities linked with Google.
const ProviderGoogle = "google"

// AMRGoogle is a session signed in with Google (the access token's amr).
const AMRGoogle = "google"

var (
	// ErrGoogleOff means no Google OAuth client is configured.
	ErrGoogleOff = errors.New("google sign-in is not set up")

	// ErrGoogleEmailUnverified means Google has not confirmed the account's
	// address, so it cannot be trusted to sign in to an account with it.
	ErrGoogleEmailUnverified = errors.New("google account email is not verified")
)

// Identity is an account at another provider, linked to a user.
type Identity struct {
	Provider string
	Subject  string
}

// googleAuth is what the service needs from Google; *google.Client outside
// tests.
type googleAuth interface {
	Start(ctx context.Context) (google.Start, error)
	Exchange(ctx context.Context, code, verifier, nonce string) (google.Identity, error)
}

var _ googleAuth = (*google.Client)(nil)

// identityStore is the part of the repository Google sign-in uses.
type identityStore interface {
	FindByIdentity(ctx context.Context, provider, subject string) (User, error)
	CreateWithIdentity(ctx context.Context, u User, id Identity, messages ...outbox.Message) (User, error)
	LinkIdentity(ctx context.Context, userID string, id Identity, events func(User) ([]outbox.Message, error)) (User, error)
}

// GoogleStart begins a Google sign-in: the address to send the reader to,
// and the state, nonce and verifier the web keeps until they come back.
func (s *Service) GoogleStart(ctx context.Context) (google.Start, error) {
	if s.google == nil {
		return google.Start{}, ErrGoogleOff
	}
	return s.google.Start(ctx)
}

// GoogleSignIn finishes a Google sign-in with the code Google sent back, and
// starts a session on device. The account is, in order:
//
//  1. the one this Google account is already linked to;
//  2. the one with the same email, which gets linked (Google has confirmed
//     the address, so it is the owner's) and has its email confirmed;
//  3. a new account without a password, its email confirmed.
//
// Confirming an email reports it (outbox.NewEmailVerified), so the orders
// placed as a guest with it join the account. With two-step sign-in on, the
// code step follows as after a password.
func (s *Service) GoogleSignIn(ctx context.Context, code, verifier, nonce, device string) (User, Tokens, error) {

	if s.google == nil {
		return User{}, Tokens{}, ErrGoogleOff
	}

	g, err := s.google.Exchange(ctx, code, verifier, nonce)
	if err != nil {
		return User{}, Tokens{}, err
	}
	if !g.EmailVerified {
		return User{}, Tokens{}, ErrGoogleEmailUnverified
	}

	id := Identity{Provider: ProviderGoogle, Subject: g.Subject}

	u, err := s.store.FindByIdentity(ctx, id.Provider, id.Subject)
	if errors.Is(err, ErrNotFound) {
		u, err = s.linkOrCreate(ctx, id, g)
	}
	if err != nil {
		return User{}, Tokens{}, err
	}

	if u.TOTPEnabledAt != nil {
		return User{}, Tokens{}, s.challenge(ctx, u, device)
	}

	tokens, err := s.startSession(ctx, u, device, AMRGoogle)
	if err != nil {
		return User{}, Tokens{}, err
	}

	return u, tokens, nil
}

// linkOrCreate links id to the account with Google's email, or makes one.
func (s *Service) linkOrCreate(ctx context.Context, id Identity, g google.Identity) (User, error) {

	email := normalizeEmail(g.Email)
	if err := validateEmail(email); err != nil {
		return User{}, err
	}

	existing, err := s.store.FindByEmail(ctx, email)
	switch {
	case err == nil:
		wasVerified := existing.EmailVerifiedAt != nil
		return s.store.LinkIdentity(ctx, existing.ID, id, func(u User) ([]outbox.Message, error) {
			if wasVerified {
				return nil, nil
			}
			return verifiedEvent(u)
		})
	case !errors.Is(err, ErrNotFound):
		return User{}, err
	}

	uid, err := uuid.NewV7()
	if err != nil {
		return User{}, err
	}

	u := User{
		ID:    uid.String(),
		Email: email,
		// No password: a password sign-in never matches an empty hash.
		PasswordHash: "",
		Name:         googleName(g.Name, email),
		Role:         RoleCustomer,
	}

	// The event needs the time the address was confirmed, which the database
	// sets; it is within the same transaction, so "now" is close enough.
	now := s.now()
	u.EmailVerifiedAt = &now
	msg, err := verifiedEvent(u)
	if err != nil {
		return User{}, err
	}

	return s.store.CreateWithIdentity(ctx, u, id, msg...)
}

// verifiedEvent reports that u's address is confirmed.
func verifiedEvent(u User) ([]outbox.Message, error) {
	if u.EmailVerifiedAt == nil {
		return nil, fmt.Errorf("account %s has no confirmed email", u.ID)
	}
	msg, err := outbox.NewEmailVerified(u.ID, u.Email, *u.EmailVerifiedAt)
	if err != nil {
		return nil, err
	}
	return []outbox.Message{msg}, nil
}

// googleName is the name Google gives, or the address before the @, cut to
// the column's length.
func googleName(name, email string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if r := []rune(name); len(r) > maxNameLen {
		name = string(r[:maxNameLen])
	}
	return name
}
