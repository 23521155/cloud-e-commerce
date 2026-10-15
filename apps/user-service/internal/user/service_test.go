package user

import (
	"context"
	"errors"
	"strings"
	"testing"

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

func TestRegisterCreatesCustomer(t *testing.T) {

	store := &fakeStore{}
	svc := NewService(store)

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

	svc := NewService(&fakeStore{})
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
			_, err := NewService(store).Register(context.Background(), in)

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
	_, err := NewService(&fakeStore{}).Register(context.Background(), RegisterInput{
		Name: "Reader", Email: "reader@example.com", Password: "mậtkhẩuu",
	})
	if err != nil {
		t.Fatalf("8-character password refused: %v", err)
	}
}
