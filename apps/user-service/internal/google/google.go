// Package google signs readers in with their Google account: OpenID Connect's
// authorization code flow with PKCE. The web keeps the state, nonce and code
// verifier of a sign-in in a cookie between the two calls; this package builds
// the address that sends the reader to Google, then exchanges the code Google
// sends back and checks the ID token that comes with it.
package google

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// issuer is Google's OpenID Connect issuer; discovery reads its endpoints and
// signing keys from there.
const issuer = "https://accounts.google.com"

// ErrRefused means Google did not sign the reader in: the code was wrong,
// used or expired, or the ID token did not check out.
var ErrRefused = errors.New("google sign-in refused")

// Config is the OAuth client made in the Google Cloud console.
type Config struct {
	ClientID     string
	ClientSecret string

	// RedirectURL is the web's callback; it must be listed on the client.
	RedirectURL string
}

// Start is one sign-in about to leave for Google. State, Nonce and Verifier
// stay with the browser until Google sends it back.
type Start struct {
	URL      string `json:"url"`
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
}

// Identity is who Google says the reader is.
type Identity struct {
	// Subject is the Google account's id: it never changes, unlike the email.
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
}

// Client talks to Google. Discovery happens on first use, so the service
// starts even when Google cannot be reached.
type Client struct {
	cfg Config

	mu       sync.Mutex
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// ready runs discovery once it succeeds; a failure is tried again next call.
func (c *Client) ready(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.oauth != nil {
		return c.oauth, c.verifier, nil
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("google discovery: %w", err)
	}

	c.oauth = &oauth2.Config{
		ClientID:     c.cfg.ClientID,
		ClientSecret: c.cfg.ClientSecret,
		RedirectURL:  c.cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
	}
	c.verifier = provider.Verifier(&oidc.Config{ClientID: c.cfg.ClientID})

	return c.oauth, c.verifier, nil
}

// Start makes a new state, nonce and PKCE verifier and the Google address
// that asks the reader to sign in.
func (c *Client) Start(ctx context.Context) (Start, error) {

	oauth, _, err := c.ready(ctx)
	if err != nil {
		return Start{}, err
	}

	state, err := random()
	if err != nil {
		return Start{}, err
	}
	nonce, err := random()
	if err != nil {
		return Start{}, err
	}
	verifier := oauth2.GenerateVerifier()

	url := oauth.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oidc.Nonce(nonce),
		// Always show the account chooser, so a shared computer does not
		// sign in as whoever used Google last.
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)

	return Start{URL: url, State: state, Nonce: nonce, Verifier: verifier}, nil
}

// Exchange trades the code Google sent back for the reader's identity. The
// verifier and nonce are the ones Start made for this sign-in. A refusal by
// Google is ErrRefused; other errors mean Google could not be reached.
func (c *Client) Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error) {

	oauth, idVerifier, err := c.ready(ctx)
	if err != nil {
		return Identity{}, err
	}

	tok, err := oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		var refused *oauth2.RetrieveError
		if errors.As(err, &refused) {
			return Identity{}, fmt.Errorf("%w: %s", ErrRefused, refused.ErrorCode)
		}
		return Identity{}, fmt.Errorf("google token exchange: %w", err)
	}

	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		return Identity{}, fmt.Errorf("%w: no id_token", ErrRefused)
	}

	// Checks the signature, issuer, audience and expiry.
	idToken, err := idVerifier.Verify(ctx, raw)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrRefused, err)
	}

	// The nonce ties the token to this browser's sign-in.
	if nonce == "" || idToken.Nonce != nonce {
		return Identity{}, fmt.Errorf("%w: nonce does not match", ErrRefused)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrRefused, err)
	}

	return Identity{Subject: idToken.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified, Name: claims.Name}, nil
}

// random is 32 random bytes, URL-safe.
func random() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
