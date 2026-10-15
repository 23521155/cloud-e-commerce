package user

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fizzisme/user-service/internal/token"
	"github.com/google/uuid"
)

// testRefreshTTL is the refresh token lifetime of test services.
const testRefreshTTL = 720 * time.Hour

// fakeStore keeps created users and refresh tokens in memory and refuses a
// second account with the same email, as the unique key does.
type fakeStore struct {
	users map[string]User
	err   error

	refresh []*RefreshToken
}

func (f *fakeStore) CreateRefreshToken(ctx context.Context, t RefreshToken) error {
	t.ID = int64(len(f.refresh) + 1)
	f.refresh = append(f.refresh, &t)
	return nil
}

func (f *fakeStore) FindRefreshToken(ctx context.Context, hash []byte) (RefreshToken, error) {
	for _, t := range f.refresh {
		if string(t.TokenHash) == string(hash) {
			return *t, nil
		}
	}
	return RefreshToken{}, ErrNotFound
}

func (f *fakeStore) MarkRefreshTokenUsed(ctx context.Context, id int64) (bool, error) {
	t := f.refresh[id-1]
	if t.UsedAt != nil {
		return false, nil
	}
	now := time.Now()
	t.UsedAt = &now
	return true, nil
}

func (f *fakeStore) UpdatePassword(ctx context.Context, id, hash string) error {
	for email, u := range f.users {
		if u.ID == id {
			u.PasswordHash = hash
			f.users[email] = u
			return nil
		}
	}
	return ErrNotFound
}

func (f *fakeStore) RevokeAllSessions(ctx context.Context, userID string) error {
	now := time.Now()
	for _, t := range f.refresh {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeStore) RevokeFamily(ctx context.Context, familyID string) error {
	now := time.Now()
	for _, t := range f.refresh {
		if t.FamilyID == familyID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
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

func (f *fakeStore) UpdateProfile(ctx context.Context, id, name string, p Profile) (User, error) {
	for email, u := range f.users {
		if u.ID == id {
			u.Name, u.Profile = name, p
			f.users[email] = u
			return u, nil
		}
	}
	return User{}, ErrNotFound
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
	gotName  string
}

func (f *fakeIssuer) Issue(sub token.Subject) (token.Access, error) {
	f.gotID, f.gotEmail, f.gotRoles, f.gotName = sub.ID, sub.Email, sub.Roles, sub.Name
	return token.Access{Token: "token-for-" + sub.ID}, nil
}

func TestRegisterCreatesCustomer(t *testing.T) {

	store := &fakeStore{}
	svc := NewService(store, &fakeIssuer{}, testRefreshTTL)

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

	svc := NewService(&fakeStore{}, &fakeIssuer{}, testRefreshTTL)
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
			_, err := NewService(store, &fakeIssuer{}, testRefreshTTL).Register(context.Background(), in)

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
	_, err := NewService(&fakeStore{}, &fakeIssuer{}, testRefreshTTL).Register(context.Background(), RegisterInput{
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
	svc := NewService(&fakeStore{}, tokens, testRefreshTTL)

	u, err := svc.Register(context.Background(), RegisterInput{Name: "Reader", Email: "reader@example.com", Password: "right password"})
	if err != nil {
		t.Fatal(err)
	}
	return svc, tokens, u
}

func TestLoginIssuesTokenForTheAccount(t *testing.T) {

	svc, tokens, want := registered(t)

	// The email is matched however it is typed.
	u, tokens2, err := svc.Login(context.Background(), " Reader@Example.com", "right password", "Firefox on Windows")
	if err != nil {
		t.Fatal(err)
	}

	if u.ID != want.ID || tokens2.Access.Token != "token-for-"+want.ID || tokens2.Refresh == "" {
		t.Errorf("logged in as %q with %+v", u.ID, tokens2)
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
		_, _, err := svc.Login(context.Background(), tc.email, tc.password, "")
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

func TestUpdateProfileTrimsAndStores(t *testing.T) {

	svc, _, u := registered(t)

	got, err := svc.UpdateProfile(context.Background(), u.ID, "  Trần Thị Bình ", Profile{
		Phone: " 0901234567 ", Address: "12 Lê Lợi", City: "Huế", PostalCode: "", Country: "Vietnam",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Trần Thị Bình" || got.Phone != "0901234567" || got.City != "Huế" || got.PostalCode != "" {
		t.Errorf("stored %+v", got)
	}
}

func TestUpdateProfileValidates(t *testing.T) {

	svc, _, u := registered(t)

	tests := []struct {
		name    string
		user    string
		profile Profile
		field   string
	}{
		{"empty name", " ", Profile{}, "name"},
		{"long phone", "R", Profile{Phone: strings.Repeat("9", maxPhoneLen+1)}, "phone"},
		{"long address", "R", Profile{Address: strings.Repeat("a", maxAddressLen+1)}, "address"},
		{"long postal code", "R", Profile{PostalCode: strings.Repeat("1", maxPostalLen+1)}, "postal code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateProfile(context.Background(), u.ID, tt.user, tt.profile)
			if !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), tt.field) {
				t.Errorf("error = %v, want ErrInvalidInput naming %q", err, tt.field)
			}
		})
	}
}

func TestUpdateProfileUnknownAccount(t *testing.T) {

	svc, _, _ := registered(t)

	if _, err := svc.UpdateProfile(context.Background(), "no-such-id", "R", Profile{}); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}
