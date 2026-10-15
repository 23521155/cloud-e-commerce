package user

import (
	"context"
	"errors"
	"testing"
	"time"
)

// signedIn returns a service with one account signed in once, its store
// and the tokens of that sign-in.
func signedIn(t *testing.T) (*Service, *fakeStore, User, Tokens) {
	t.Helper()

	store := &fakeStore{}
	svc := NewService(store, &fakeIssuer{}, testRefreshTTL)

	ctx := context.Background()
	if _, err := svc.Register(ctx, RegisterInput{Name: "Reader", Email: "reader@example.com", Password: "right password"}); err != nil {
		t.Fatal(err)
	}

	u, tokens, err := svc.Login(ctx, "reader@example.com", "right password", "Firefox on Windows")
	if err != nil {
		t.Fatal(err)
	}
	return svc, store, u, tokens
}

// at makes the service's clock read now+d.
func at(svc *Service, d time.Duration) {
	svc.now = func() time.Time { return time.Now().Add(d) }
}

func TestLoginStoresOnlyTheHashOfTheRefreshToken(t *testing.T) {

	_, store, u, tokens := signedIn(t)

	if len(store.refresh) != 1 {
		t.Fatalf("stored %d refresh tokens, want 1", len(store.refresh))
	}
	stored := store.refresh[0]

	if string(stored.TokenHash) == tokens.Refresh || string(stored.TokenHash) != string(hashToken(tokens.Refresh)) {
		t.Error("the stored value is not the hash of the token")
	}
	if stored.UserID != u.ID || stored.DeviceName != "Firefox on Windows" || stored.FamilyID == "" {
		t.Errorf("stored %+v", stored)
	}
	if d := time.Until(tokens.RefreshExpiresAt); d < testRefreshTTL-time.Minute || d > testRefreshTTL {
		t.Errorf("refresh token expires in %v, want %v", d, testRefreshTTL)
	}
}

func TestRefreshRotatesWithinTheSession(t *testing.T) {

	svc, store, u, first := signedIn(t)

	got, second, err := svc.Refresh(context.Background(), first.Refresh, "Firefox on Windows")
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != u.ID || second.Refresh == first.Refresh || second.Access.Token == "" {
		t.Errorf("refreshed as %q, tokens %+v", got.ID, second)
	}
	if len(store.refresh) != 2 || store.refresh[0].UsedAt == nil || store.refresh[1].FamilyID != store.refresh[0].FamilyID {
		t.Errorf("tokens after refresh: %+v %+v", store.refresh[0], store.refresh[1])
	}

	// The new token works in turn.
	if _, _, err := svc.Refresh(context.Background(), second.Refresh, ""); err != nil {
		t.Errorf("second refresh: %v", err)
	}
}

func TestRefreshSignsTheAccountAsItIsNow(t *testing.T) {

	svc, _, u, tokens := signedIn(t)
	issuer := svc.tokens.(*fakeIssuer)

	if _, err := svc.UpdateProfile(context.Background(), u.ID, "New Name", Profile{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Refresh(context.Background(), tokens.Refresh, ""); err != nil {
		t.Fatal(err)
	}

	if issuer.gotName != "New Name" {
		t.Errorf("access token name = %q, want the new name", issuer.gotName)
	}
}

func TestRefreshAllowsAUsedTokenWithinTheGracePeriod(t *testing.T) {

	svc, _, _, tokens := signedIn(t)

	// Two tabs refresh with the same token at once.
	if _, _, err := svc.Refresh(context.Background(), tokens.Refresh, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Refresh(context.Background(), tokens.Refresh, ""); err != nil {
		t.Errorf("second use within %v refused: %v", reuseGrace, err)
	}
}

func TestRefreshReuseAfterTheGracePeriodEndsTheSession(t *testing.T) {

	svc, store, _, first := signedIn(t)

	_, second, err := svc.Refresh(context.Background(), first.Refresh, "")
	if err != nil {
		t.Fatal(err)
	}

	// The old token comes back later: it was copied.
	at(svc, reuseGrace+time.Second)
	if _, _, err := svc.Refresh(context.Background(), first.Refresh, ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("reused token error = %v, want ErrInvalidRefreshToken", err)
	}

	for _, rt := range store.refresh {
		if rt.RevokedAt == nil {
			t.Errorf("token %d not revoked after reuse", rt.ID)
		}
	}
	// So the legitimate holder's newer token is refused too.
	if _, _, err := svc.Refresh(context.Background(), second.Refresh, ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("newer token error = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestRefreshRefusesUnknownAndExpiredTokens(t *testing.T) {

	svc, _, _, tokens := signedIn(t)

	if _, _, err := svc.Refresh(context.Background(), "not-a-token", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("unknown token error = %v", err)
	}

	at(svc, testRefreshTTL+time.Minute)
	if _, _, err := svc.Refresh(context.Background(), tokens.Refresh, ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("expired token error = %v", err)
	}
}

func TestLogoutEndsOnlyThatSession(t *testing.T) {

	svc, _, _, laptop := signedIn(t)
	_, phone, err := svc.Login(context.Background(), "reader@example.com", "right password", "Safari on iOS")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Logout(context.Background(), laptop.Refresh); err != nil {
		t.Fatal(err)
	}

	if _, _, err := svc.Refresh(context.Background(), laptop.Refresh, ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("signed-out session refreshed: %v", err)
	}
	if _, _, err := svc.Refresh(context.Background(), phone.Refresh, ""); err != nil {
		t.Errorf("other session refused: %v", err)
	}

	if err := svc.Logout(context.Background(), "unknown"); err != nil {
		t.Errorf("logout with an unknown token: %v", err)
	}
}
