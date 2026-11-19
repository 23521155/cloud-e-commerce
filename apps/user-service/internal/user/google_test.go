package user

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fizzisme/user-service/internal/google"
	"github.com/fizzisme/user-service/internal/outbox"
)

func (f *fakeStore) FindByIdentity(ctx context.Context, provider, subject string) (User, error) {
	email, ok := f.identities[Identity{provider, subject}]
	if !ok {
		return User{}, ErrNotFound
	}
	return f.users[email], nil
}

func (f *fakeStore) CreateWithIdentity(ctx context.Context, u User, id Identity, messages ...outbox.Message) (User, error) {
	if f.users == nil {
		f.users = map[string]User{}
	}
	if _, taken := f.users[u.Email]; taken {
		return User{}, ErrEmailTaken
	}
	f.users[u.Email] = u
	f.link(id, u.Email)
	f.outbox = append(f.outbox, messages...)
	return u, nil
}

func (f *fakeStore) LinkIdentity(ctx context.Context, userID string, id Identity, events func(User) ([]outbox.Message, error)) (User, error) {
	for email, u := range f.users {
		if u.ID != userID {
			continue
		}
		if u.EmailVerifiedAt == nil {
			now := time.Now()
			u.EmailVerifiedAt = &now
			f.users[email] = u
		}
		f.link(id, email)
		messages, err := events(u)
		if err != nil {
			return User{}, err
		}
		f.outbox = append(f.outbox, messages...)
		return u, nil
	}
	return User{}, ErrNotFound
}

func (f *fakeStore) link(id Identity, email string) {
	if f.identities == nil {
		f.identities = map[Identity]string{}
	}
	f.identities[id] = email
}

// fakeGoogle answers Exchange with the identity the test set.
type fakeGoogle struct {
	identity google.Identity
	err      error

	gotCode, gotVerifier, gotNonce string
}

func (g *fakeGoogle) Start(ctx context.Context) (google.Start, error) {
	return google.Start{URL: "https://accounts.google.com/auth", State: "s", Nonce: "n", Verifier: "v"}, g.err
}

func (g *fakeGoogle) Exchange(ctx context.Context, code, verifier, nonce string) (google.Identity, error) {
	g.gotCode, g.gotVerifier, g.gotNonce = code, verifier, nonce
	return g.identity, g.err
}

var anGoogleReader = google.Identity{Subject: "1098765", Email: "Reader@Gmail.com", EmailVerified: true, Name: "Google Reader"}

func googleService(store *fakeStore, g *fakeGoogle) (*Service, *fakeIssuer) {
	o := testOptions
	o.Google = g
	tokens := &fakeIssuer{}
	return NewService(store, tokens, o), tokens
}

// verifiedEvents are the email-verified events in the outbox.
func verifiedEvents(t *testing.T, store *fakeStore) []outbox.EmailVerified {
	t.Helper()
	var out []outbox.EmailVerified
	for _, m := range store.outbox {
		if m.Queue != outbox.QueueUserEmailVerified {
			continue
		}
		var e outbox.EmailVerified
		if err := json.Unmarshal([]byte(m.Payload), &e); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

func TestGoogleSignInCreatesAnAccount(t *testing.T) {

	store, g := &fakeStore{}, &fakeGoogle{identity: anGoogleReader}
	svc, tokens := googleService(store, g)

	u, got, err := svc.GoogleSignIn(context.Background(), "code", "verifier", "nonce", "Chrome on Windows")
	if err != nil {
		t.Fatal(err)
	}

	if g.gotCode != "code" || g.gotVerifier != "verifier" || g.gotNonce != "nonce" {
		t.Errorf("exchanged %q %q %q", g.gotCode, g.gotVerifier, g.gotNonce)
	}
	if u.Email != "reader@gmail.com" || u.Name != "Google Reader" || u.Role != RoleCustomer || u.EmailVerifiedAt == nil {
		t.Errorf("account = %+v", u)
	}
	if u.PasswordHash != "" {
		t.Error("an account made with Google has a password")
	}
	if got.Refresh == "" || strings.Join(tokens.gotAMR, " ") != AMRGoogle {
		t.Errorf("tokens %+v, amr %v", got, tokens.gotAMR)
	}

	// The confirmed address is reported, so guest orders with it join.
	if ev := verifiedEvents(t, store); len(ev) != 1 || ev[0].UserID != u.ID || ev[0].Email != "reader@gmail.com" {
		t.Errorf("events = %+v", ev)
	}

	// Signing in again finds the same account and reports nothing new.
	again, _, err := svc.GoogleSignIn(context.Background(), "code2", "v2", "n2", "")
	if err != nil || again.ID != u.ID || len(store.users) != 1 || len(verifiedEvents(t, store)) != 1 {
		t.Errorf("second sign-in: %+v, %v; %d users, %d events", again, err, len(store.users), len(verifiedEvents(t, store)))
	}
}

func TestGoogleSignInLinksTheAccountWithTheSameEmail(t *testing.T) {

	store := &fakeStore{}
	svc, _ := googleService(store, &fakeGoogle{identity: anGoogleReader})

	// Registered with a password, address not confirmed yet.
	existing, err := svc.Register(context.Background(), RegisterInput{Name: "Reader", Email: "reader@gmail.com", Password: "right password"})
	if err != nil {
		t.Fatal(err)
	}

	u, _, err := svc.GoogleSignIn(context.Background(), "code", "v", "n", "")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != existing.ID || u.Name != "Reader" || u.EmailVerifiedAt == nil || len(store.users) != 1 {
		t.Errorf("signed in as %+v, want the registered account, confirmed", u)
	}
	if ev := verifiedEvents(t, store); len(ev) != 1 || ev[0].UserID != existing.ID {
		t.Errorf("events = %+v", ev)
	}

	// The password still signs in too.
	if _, _, err := svc.Login(context.Background(), "reader@gmail.com", "right password", ""); err != nil {
		t.Errorf("password sign-in after linking: %v", err)
	}
}

func TestGoogleSignInDoesNotReportAnAddressConfirmedBefore(t *testing.T) {

	now := time.Now()
	store := &fakeStore{users: map[string]User{
		"reader@gmail.com": {ID: "u1", Email: "reader@gmail.com", Name: "Reader", Role: RoleCustomer, EmailVerifiedAt: &now},
	}}
	svc, _ := googleService(store, &fakeGoogle{identity: anGoogleReader})

	if _, _, err := svc.GoogleSignIn(context.Background(), "code", "v", "n", ""); err != nil {
		t.Fatal(err)
	}
	if ev := verifiedEvents(t, store); len(ev) != 0 {
		t.Errorf("events = %+v, want none", ev)
	}
}

func TestGoogleSignInRefusesAnUnverifiedEmail(t *testing.T) {

	id := anGoogleReader
	id.EmailVerified = false

	store := &fakeStore{}
	svc, _ := googleService(store, &fakeGoogle{identity: id})

	if _, _, err := svc.GoogleSignIn(context.Background(), "code", "v", "n", ""); !errors.Is(err, ErrGoogleEmailUnverified) {
		t.Errorf("err = %v, want ErrGoogleEmailUnverified", err)
	}
	if len(store.users) != 0 || len(store.identities) != 0 {
		t.Error("an account was stored")
	}
}

func TestGoogleSignInAsksForTheCode(t *testing.T) {

	now := time.Now()
	store := &fakeStore{users: map[string]User{
		"reader@gmail.com": {ID: "u1", Email: "reader@gmail.com", Name: "Reader", Role: RoleCustomer, EmailVerifiedAt: &now, TOTPEnabledAt: &now},
	}}
	svc, _ := googleService(store, &fakeGoogle{identity: anGoogleReader})

	_, tokens, err := svc.GoogleSignIn(context.Background(), "code", "v", "n", "")

	var mfa *MFARequiredError
	if !errors.As(err, &mfa) || tokens.Refresh != "" {
		t.Errorf("err = %v, tokens %+v; want the code step and no tokens", err, tokens)
	}
}

func TestGoogleSignInErrors(t *testing.T) {

	svc, _ := googleService(&fakeStore{}, &fakeGoogle{err: google.ErrRefused})
	if _, _, err := svc.GoogleSignIn(context.Background(), "code", "v", "n", ""); !errors.Is(err, google.ErrRefused) {
		t.Errorf("refused: err = %v", err)
	}

	off := NewService(&fakeStore{}, &fakeIssuer{}, testOptions)
	if _, _, err := off.GoogleSignIn(context.Background(), "code", "v", "n", ""); !errors.Is(err, ErrGoogleOff) {
		t.Errorf("off: sign-in err = %v", err)
	}
	if _, err := off.GoogleStart(context.Background()); !errors.Is(err, ErrGoogleOff) {
		t.Errorf("off: start err = %v", err)
	}
}

func TestPasswordSignInToAGoogleAccount(t *testing.T) {

	store := &fakeStore{}
	svc, _ := googleService(store, &fakeGoogle{identity: anGoogleReader})

	if _, _, err := svc.GoogleSignIn(context.Background(), "code", "v", "n", ""); err != nil {
		t.Fatal(err)
	}

	// No password: refused like a wrong one, not an internal error.
	for _, password := range []string{"", "anything at all"} {
		if _, _, err := svc.Login(context.Background(), "reader@gmail.com", password, ""); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("password %q: err = %v, want ErrInvalidCredentials", password, err)
		}
	}
}

func TestGoogleName(t *testing.T) {

	if got := googleName("  An  ", "an@gmail.com"); got != "An" {
		t.Errorf("got %q", got)
	}
	if got := googleName("", "an.nguyen@gmail.com"); got != "an.nguyen" {
		t.Errorf("no name: got %q", got)
	}
	if got := googleName(strings.Repeat("á", maxNameLen+5), "a@b.co"); len([]rune(got)) != maxNameLen {
		t.Errorf("long name: %d runes", len([]rune(got)))
	}
}
