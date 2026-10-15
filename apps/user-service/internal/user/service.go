package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	// ErrInvalidInput means the request is malformed; the message says which
	// field and is written for the user.
	ErrInvalidInput = errors.New("invalid input")

	// ErrEmailTaken means another account already uses the email.
	ErrEmailTaken = errors.New("email already registered")
)

// Field limits, matched by the columns in migrations/000001_create_users.
const (
	maxNameLen  = 100
	maxEmailLen = 254

	minPasswordLen = 8
	// Long enough for any passphrase, short enough that hashing a huge
	// body cannot be used to load the service.
	maxPasswordLen = 128
)

// store is what the service needs from the repository; tests replace it
// with a fake.
type store interface {
	Create(ctx context.Context, u User) (User, error)
}

type Service struct {
	store store
}

func NewService(store store) *Service {
	return &Service{store: store}
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
