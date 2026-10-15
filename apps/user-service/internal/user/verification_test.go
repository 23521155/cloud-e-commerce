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

// linkToken returns the token in the confirmation link of an outbox email.
func linkToken(t *testing.T, m outbox.Message) string {
	t.Helper()

	var mail outbox.Email
	if err := json.Unmarshal([]byte(m.Payload), &mail); err != nil {
		t.Fatal(err)
	}
	const prefix = "/verify-email?token="
	i := strings.Index(mail.Text, prefix)
	if i < 0 {
		t.Fatalf("no link in %q", mail.Text)
	}
	return strings.Fields(mail.Text[i+len(prefix):])[0]
}

func TestVerifyEmailConfirmsOnce(t *testing.T) {

	store := &fakeStore{}
	svc := NewService(store, &fakeIssuer{}, testOptions)
	ctx := context.Background()

	u, err := svc.Register(ctx, RegisterInput{Name: "Reader", Email: "reader@example.com", Password: "12345678"})
	if err != nil {
		t.Fatal(err)
	}
	link := linkToken(t, store.outbox[0])

	got, err := svc.VerifyEmail(ctx, link)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID || got.EmailVerifiedAt == nil {
		t.Errorf("verified %+v", got)
	}

	// The order service is told, with the confirmed address.
	last := store.outbox[len(store.outbox)-1]
	var event outbox.EmailVerified
	if err := json.Unmarshal([]byte(last.Payload), &event); err != nil {
		t.Fatal(err)
	}
	if last.Queue != outbox.QueueUserEmailVerified || event.Type != "user.email_verified" || event.UserID != u.ID || event.Email != "reader@example.com" || event.VerifiedAt.IsZero() {
		t.Errorf("event %s = %+v", last.Queue, event)
	}
	events := len(store.outbox)

	if _, err := svc.VerifyEmail(ctx, link); !errors.Is(err, ErrInvalidEmailToken) {
		t.Errorf("second use error = %v, want ErrInvalidEmailToken", err)
	}
	if _, err := svc.VerifyEmail(ctx, "made-up"); !errors.Is(err, ErrInvalidEmailToken) {
		t.Errorf("unknown token error = %v, want ErrInvalidEmailToken", err)
	}
	if len(store.outbox) != events {
		t.Error("a refused link sent an event")
	}
}

func TestResendVerification(t *testing.T) {

	store := &fakeStore{}
	svc := NewService(store, &fakeIssuer{}, testOptions)
	ctx := context.Background()

	u, err := svc.Register(ctx, RegisterInput{Name: "Reader", Email: "reader@example.com", Password: "12345678"})
	if err != nil {
		t.Fatal(err)
	}

	// Right after registering, the first email has just gone out.
	if err := svc.ResendVerification(ctx, u.ID); !errors.Is(err, ErrTooSoon) {
		t.Fatalf("immediate resend error = %v, want ErrTooSoon", err)
	}

	at(svc, resendInterval+time.Second)
	if err := svc.ResendVerification(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if len(store.outbox) != 2 || len(store.emailTokens) != 2 {
		t.Fatalf("%d emails, %d tokens; want 2 and 2", len(store.outbox), len(store.emailTokens))
	}

	// Either link confirms the address; then nothing more is sent.
	if _, err := svc.VerifyEmail(ctx, linkToken(t, store.outbox[1])); err != nil {
		t.Fatal(err)
	}
	at(svc, 3*resendInterval)
	if err := svc.ResendVerification(ctx, u.ID); !errors.Is(err, ErrAlreadyVerified) {
		t.Errorf("resend after confirming error = %v, want ErrAlreadyVerified", err)
	}
}
