package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fizzisme/user-service/internal/token"
	"github.com/google/uuid"
)

var (
	// ErrInvalidInput means the request is malformed; the message says which
	// field and is written for the user.
	ErrInvalidInput = errors.New("invalid input")

	// ErrEmailTaken means another account already uses the email.
	ErrEmailTaken = errors.New("email already registered")

	// ErrInvalidCredentials means the email or the password is wrong. Which
	// one is never told, so the answer cannot be used to find accounts.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrNotFound means no account has the id or email.
	ErrNotFound = errors.New("user not found")
)

// Field limits, matched by the columns in migrations. The profile limits are
// also the order service's, so a saved profile is accepted at checkout.
const (
	maxNameLen  = 100
	maxEmailLen = 254

	maxPhoneLen   = 20
	maxAddressLen = 300
	maxCityLen    = 100
	maxPostalLen  = 20
	maxCountryLen = 60

	minPasswordLen = 8
	// Long enough for any passphrase, short enough that hashing a huge
	// body cannot be used to load the service.
	maxPasswordLen = 128
)

// store is what the service needs from the repository; tests replace it
// with a fake.
type store interface {
	Create(ctx context.Context, u User) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id string) (User, error)
	UpdateProfile(ctx context.Context, id, name string, p Profile) (User, error)

	sessionStore
}

// issuer signs access tokens; *token.Issuer outside tests.
type issuer interface {
	Issue(sub token.Subject) (token.Access, error)
}

type Service struct {
	store  store
	tokens issuer

	// refreshTTL is how long a refresh token is accepted, and so how long
	// a device stays signed in without being used.
	refreshTTL time.Duration

	// now is time.Now outside tests.
	now func() time.Time
}

func NewService(store store, tokens issuer, refreshTTL time.Duration) *Service {
	return &Service{store: store, tokens: tokens, refreshTTL: refreshTTL, now: time.Now}
}

// RegisterInput is a new account as the user typed it.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// Register creates a customer account.
func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {

	name := strings.TrimSpace(in.Name)
	email := normalizeEmail(in.Email)

	if err := validateName(name); err != nil {
		return User{}, err
	}
	if err := validateEmail(email); err != nil {
		return User{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return User{}, err
	}

	hash, err := hashPassword(in.Password)
	if err != nil {
		return User{}, err
	}

	// UUIDv7 starts with the time, so new rows land at the end of the
	// primary key index instead of splitting pages at random.
	id, err := uuid.NewV7()
	if err != nil {
		return User{}, err
	}

	return s.store.Create(ctx, User{
		ID:           id.String(),
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         RoleCustomer,
	})
}

// Login checks an email and password and starts a session for the account
// on device. A wrong email and a wrong password give the same error.
func (s *Service) Login(ctx context.Context, email, password, device string) (User, Tokens, error) {

	u, err := s.store.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ErrNotFound) {
		// Hash anyway, so an unknown email takes as long to refuse as a
		// wrong password and timing does not reveal which accounts exist.
		verifyPassword(password, dummyHash())
		return User{}, Tokens{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, Tokens{}, err
	}

	ok, err := verifyPassword(password, u.PasswordHash)
	if err != nil {
		return User{}, Tokens{}, err
	}
	if !ok {
		return User{}, Tokens{}, ErrInvalidCredentials
	}

	tokens, err := s.startSession(ctx, u, device)
	if err != nil {
		return User{}, Tokens{}, err
	}

	return u, tokens, nil
}

// Me returns the account an access token was issued for.
func (s *Service) Me(ctx context.Context, id string) (User, error) {
	return s.store.FindByID(ctx, id)
}

// UpdateProfile sets the name and the contact and delivery details of the
// account with id. Every detail but the name may be left empty.
func (s *Service) UpdateProfile(ctx context.Context, id, name string, p Profile) (User, error) {

	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return User{}, err
	}

	fields := []struct {
		label string
		value *string
		max   int
	}{
		{"phone", &p.Phone, maxPhoneLen},
		{"address", &p.Address, maxAddressLen},
		{"city", &p.City, maxCityLen},
		{"postal code", &p.PostalCode, maxPostalLen},
		{"country", &p.Country, maxCountryLen},
	}
	for _, f := range fields {
		*f.value = strings.TrimSpace(*f.value)
		if utf8.RuneCountInString(*f.value) > f.max {
			return User{}, fmt.Errorf("%w: %s must be at most %d characters", ErrInvalidInput, f.label, f.max)
		}
	}

	return s.store.UpdateProfile(ctx, id, name, p)
}

// dummyHash is a hash of a random password with the current cost, for
// Login to check against when the email has no account.
var dummyHash = sync.OnceValue(func() string {
	hash, err := hashPassword(uuid.NewString())
	if err != nil {
		panic(err)
	}
	return hash
})

// normalizeEmail makes one address match however it was typed. Only case
// and surrounding spaces are folded; dots and +tags are kept.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateName(name string) error {
	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return fmt.Errorf("%w: name must be 1 to %d characters", ErrInvalidInput, maxNameLen)
	}
	return nil
}

func validateEmail(email string) error {
	// ParseAddress also accepts "Name <a@b>"; only a bare address is wanted.
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > maxEmailLen {
		return fmt.Errorf("%w: email is not a valid address", ErrInvalidInput)
	}
	return nil
}

func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordLen || n > maxPasswordLen {
		return fmt.Errorf("%w: password must be %d to %d characters", ErrInvalidInput, minPasswordLen, maxPasswordLen)
	}
	return nil
}
