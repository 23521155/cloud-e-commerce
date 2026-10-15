package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fizzisme/user-service/internal/token"
	"github.com/google/uuid"
)

// fakeStore keeps created users in memory and refuses a second account
// with the same email, as the unique key does.
type fakeStore struct {
	users map[string]User
	err   error
}

func (f *fakeStore) Create(ctx context.Context, u User) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	if f.users == nil {
		f.users = map[string]User{}
	}
	if _, taken := f.users[u.Email]; taken {
		return User{}, ErrEmailTaken
	}
	f.users[u.Email] = u
	return u, nil
}

func (f *fakeStore) FindByEmail(ctx context.Context, email string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	u, ok := f.users[email]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (f *fakeStore) FindByID(ctx context.Context, id string) (User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

// fakeIssuer signs nothing; it records who a token was asked for.
type fakeIssuer struct {
	gotID    string
	gotEmail string
	gotRoles []string
}

func (f *fakeIssuer) Issue(sub token.Subject) (token.Access, error) {
	f.gotID, f.gotEmail, f.gotRoles = sub.ID, sub.Email, sub.Roles
	return token.Access{Token: "token-for-" + sub.ID}, nil
}

func TestRegisterCreatesCustomer(t *testing.T) {

	store := &fakeStore{}
	svc := NewService(store, &fakeIssuer{})

	u, err := svc.Register(context.Background(), RegisterInput{
		Name:     "  Nguyễn Văn An ",
		Email:    " An.Nguyen@Example.com ",
		Password: "mật khẩu dài",
	})
	if err != nil {
		t.Fatal(err)
	}

	if u.Name != "Nguyễn Văn An" {
		t.Errorf("name = %q, want trimmed", u.Name)
	}
	if u.Email != "an.nguyen@example.com" {
		t.Errorf("email = %q, want trimmed and lowercased", u.Email)
	}
	if u.Role != RoleCustomer {
		t.Errorf("role = %q, want %q", u.Role, RoleCustomer)
	}

	id, err := uuid.Parse(u.ID)
	if err != nil || id.Version() != 7 {
		t.Errorf("id = %q, want a UUIDv7", u.ID)
	}

	if ok, err := verifyPassword("mật khẩu dài", u.PasswordHash); err != nil || !ok {
		t.Errorf("stored hash does not verify the password: %v, %v", ok, err)
	}
}

func TestRegisterRejectsTakenEmailWhateverTheCase(t *testing.T) {

	svc := NewService(&fakeStore{}, &fakeIssuer{})
	ctx := context.Background()

	if _, err := svc.Register(ctx, RegisterInput{Name: "A", Email: "reader@example.com", Password: "12345678"}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Register(ctx, RegisterInput{Name: "B", Email: "READER@example.com", Password: "12345678"})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

func TestRegisterValidates(t *testing.T) {

	valid := RegisterInput{Name: "Reader", Email: "reader@example.com", Password: "12345678"}

	tests := []struct {
		name  string
		edit  func(*RegisterInput)
		field string
	}{
		{"empty name", func(in *RegisterInput) { in.Name = "   " }, "name"},
		{"long name", func(in *RegisterInput) { in.Name = strings.Repeat("a", maxNameLen+1) }, "name"},
		{"no at sign", func(in *RegisterInput) { in.Email = "reader.example.com" }, "email"},
		{"display name form", func(in *RegisterInput) { in.Email = "Reader <reader@example.com>" }, "email"},
		{"long email", func(in *RegisterInput) { in.Email = strings.Repeat("a", 250) + "@example.com" }, "email"},
		{"short password", func(in *RegisterInput) { in.Password = "1234567" }, "password"},
		{"long password", func(in *RegisterInput) { in.Password = strings.Repeat("a", maxPasswordLen+1) }, "password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.edit(&in)

			store := &fakeStore{}
			_, err := NewService(store, &fakeIssuer{}).Register(context.Background(), in)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
			if !strings.Contains(err.Error(), tt.field) {
				t.Errorf("error %q does not name the field %q", err, tt.field)
			}
			if len(store.users) != 0 {
				t.Error("an invalid account was stored")
			}
		})
	}
}

func TestRegisterCountsPasswordInCharacters(t *testing.T) {

	// Eight characters, but more than eight bytes.
	_, err := NewService(&fakeStore{}, &fakeIssuer{}).Register(context.Background(), RegisterInput{
		Name: "Reader", Email: "reader@example.com", Password: "mậtkhẩuu",
	})
	if err != nil {
		t.Fatalf("8-character password refused: %v", err)
	}
}

// registered returns a service with one account in it.
func registered(t *testing.T) (*Service, *fakeIssuer, User) {
	t.Helper()

	tokens := &fakeIssuer{}
	svc := NewService(&fakeStore{}, tokens)

	u, err := svc.Register(context.Background(), RegisterInput{Name: "Reader", Email: "reader@example.com", Password: "right password"})
	if err != nil {
		t.Fatal(err)
	}
	return svc, tokens, u
}

func TestLoginIssuesTokenForTheAccount(t *testing.T) {

	svc, tokens, want := registered(t)

	// The email is matched however it is typed.
	u, access, err := svc.Login(context.Background(), " Reader@Example.com", "right password")
	if err != nil {
		t.Fatal(err)
	}

	if u.ID != want.ID || access.Token != "token-for-"+want.ID {
		t.Errorf("logged in as %q with %q", u.ID, access.Token)
	}
	if tokens.gotEmail != "reader@example.com" || len(tokens.gotRoles) != 1 || tokens.gotRoles[0] != RoleCustomer {
		t.Errorf("token asked for %q %v", tokens.gotEmail, tokens.gotRoles)
	}
}

func TestLoginRefusesWithOneErrorForBothMistakes(t *testing.T) {

	svc, tokens, _ := registered(t)

	for _, tc := range []struct{ email, password string }{
		{"reader@example.com", "wrong password"},
		{"nobody@example.com", "right password"},
	} {
		_, _, err := svc.Login(context.Background(), tc.email, tc.password)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("Login(%q) error = %v, want ErrInvalidCredentials", tc.email, err)
		}
	}
	if tokens.gotID != "" {
		t.Error("a token was issued for a refused login")
	}
}

func TestMeFindsTheAccount(t *testing.T) {

	svc, _, want := registered(t)

	u, err := svc.Me(context.Background(), want.ID)
	if err != nil || u.Email != want.Email {
		t.Fatalf("Me = %+v, %v", u, err)
	}

	if _, err := svc.Me(context.Background(), "no-such-id"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown id error = %v, want ErrNotFound", err)
	}
}
