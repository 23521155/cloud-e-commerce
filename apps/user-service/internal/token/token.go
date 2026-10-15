// Package token issues the access tokens the API gateway verifies.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims extend the gateway's auth.Claims: it reads sub, email and roles and
// forwards them as X-User-ID, X-User-Email and X-User-Roles. Name is for
// display only (the web shows it without asking this service).
type Claims struct {
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`

	jwt.RegisteredClaims
}

// Issuer signs access tokens with the secret shared with the gateway.
type Issuer struct {
	secret []byte
	issuer string
	ttl    time.Duration

	// now is time.Now outside tests.
	now func() time.Time
}

// NewIssuer refuses an empty secret: tokens signed with it would be
// accepted by anyone who also left it empty.
func NewIssuer(secret, issuer string, ttl time.Duration) (*Issuer, error) {
	if secret == "" {
		return nil, errors.New("JWT secret is empty")
	}
	if ttl <= 0 {
		return nil, errors.New("access token TTL must be positive")
	}
	return &Issuer{secret: []byte(secret), issuer: issuer, ttl: ttl, now: time.Now}, nil
}

// Access is a signed access token and when it stops being accepted.
type Access struct {
	Token     string
	ExpiresAt time.Time
}

// Subject is the user an access token is issued for.
type Subject struct {
	ID    string
	Email string
	Name  string
	Roles []string
}

// Issue signs an HS256 access token for one user. The gateway checks the
// algorithm, signature, issuer and expiry.
func (i *Issuer) Issue(sub Subject) (Access, error) {

	now := i.now()
	expires := now.Add(i.ttl)

	claims := Claims{
		Email: sub.Email,
		Name:  sub.Name,
		Roles: sub.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   sub.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return Access{}, err
	}

	return Access{Token: signed, ExpiresAt: expires}, nil
}
