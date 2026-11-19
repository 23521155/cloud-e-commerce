package user

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fizzisme/user-service/internal/outbox"
)

// resetLink returns the token in the reset link of the newest outbox email.
func resetLink(t *testing.T, store *fakeStore) string {
	t.Helper()

	var mail outbox.Email
	if err := json.Unmarshal([]byte(store.outbox[len(store.outbox)-1].Payload), &mail); err != nil {
		t.Fatal(err)
	}
	const prefix = "/reset-password?token="
	i := strings.Index(mail.Text, prefix)
	if i < 0 {
		t.Fatalf("no reset link in %q", mail.Text)
	}
	return strings.Fields(mail.Text[i+len(prefix):])[0]
}

func TestPasswordResetEndToEnd(t *testing.T) {

	svc, store, _, laptop := signedIn(t)
	ctx := context.Background()
	emails := len(store.outbox)

	if err := svc.RequestPasswordReset(ctx, " READER@example.com "); err != nil {
		t.Fatal(err)
	}
	if len(store.outbox) != emails+1 {
		t.Fatalf("%d emails, want one more", len(store.outbox))
	}
	tok := store.emailTokens[len(store.emailTokens)-1]
	if tok.Purpose != PurposeResetPassword || time.Until(tok.ExpiresAt) > resetPasswordTTL {
		t.Errorf("token = %+v", tok)
	}

	link := resetLink(t, store)
	if err := svc.ResetPassword(ctx, link, "short"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("short password error = %v, want ErrInvalidInput", err)
	}
	if err := svc.ResetPassword(ctx, link, "a brand new password"); err != nil {
		t.Fatal(err)
	}

	if _, _, err := svc.Login(ctx, "reader@example.com", "a brand new password", ""); err != nil {
		t.Errorf("new password refused: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, laptop.Refresh, ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Errorf("old session still works after a reset: %v", err)
	}
	if err := svc.ResetPassword(ctx, link, "another new password"); !errors.Is(err, ErrInvalidEmailToken) {
		t.Errorf("second use error = %v, want ErrInvalidEmailToken", err)
	}
}

func TestPasswordResetSaysNothingAboutUnknownAddresses(t *testing.T) {

	svc, store, _, _ := signedIn(t)
	emails := len(store.outbox)

	if err := svc.RequestPasswordReset(context.Background(), "nobody@example.com"); err != nil {
		t.Errorf("unknown address error = %v, want nil", err)
	}
	if len(store.outbox) != emails {
		t.Error("an email was queued for an address with no account")
	}
}

func TestPasswordResetAtMostOnceAMinute(t *testing.T) {

	svc, store, _, _ := signedIn(t)
	ctx := context.Background()
	emails := len(store.outbox)

	for range 3 {
		if err := svc.RequestPasswordReset(ctx, "reader@example.com"); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.outbox) != emails+1 {
		t.Errorf("%d reset emails, want 1", len(store.outbox)-emails)
	}

	at(svc, resendInterval+time.Second)
	if err := svc.RequestPasswordReset(ctx, "reader@example.com"); err != nil {
		t.Fatal(err)
	}
	if len(store.outbox) != emails+2 {
		t.Errorf("%d reset emails after a minute, want 2", len(store.outbox)-emails)
	}
}
