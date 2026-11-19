package user

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// fakeBox "seals" by prefixing, so a test can see what is stored.
type fakeBox struct{}

func (fakeBox) Seal(p []byte) ([]byte, error) { return append([]byte("sealed:"), p...), nil }

func (fakeBox) Open(s []byte) ([]byte, error) {
	if !strings.HasPrefix(string(s), "sealed:") {
		return nil, errors.New("not sealed here")
	}
	return s[len("sealed:"):], nil
}

// withMFA returns a signed-in account with two-step sign-in on, and its
// secret. The clock is pinned so codes are predictable.
func withMFA(t *testing.T) (*Service, *fakeStore, User, string) {
	t.Helper()

	svc, store, u, _ := signedIn(t)
	svc.box = fakeBox{}
	start := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return start }

	ctx := context.Background()
	setup, err := svc.SetupTOTP(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(setup.URL, "otpauth://totp/Marginalleya:reader@example.com?") {
		t.Errorf("URL = %q", setup.URL)
	}

	stored := store.users["reader@example.com"].TOTPSecretSealed
	if string(stored) != "sealed:"+setup.Secret {
		t.Errorf("stored %q, want the sealed secret", stored)
	}

	codes, err := svc.ConfirmTOTP(ctx, u.ID, code(t, setup.Secret, start))
	if err != nil {
		t.Fatal(err)
	}
	recoveryCodes = codes
	return svc, store, u, setup.Secret
}

// recoveryCodes are those of the last withMFA.
var recoveryCodes []string

func code(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	c, err := totp.GenerateCodeCustom(secret, at, totpOpts)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// signInWithCode signs in with the password, then with a code for at.
func signInWithCode(svc *Service, secret string, at time.Time, t *testing.T) (Tokens, error) {
	t.Helper()

	_, _, err := svc.Login(context.Background(), "reader@example.com", "right password", "Firefox on Windows")
	var mfa *MFARequiredError
	if !errors.As(err, &mfa) {
		t.Fatalf("Login error = %v, want MFARequiredError", err)
	}
	_, tokens, err := svc.VerifyMFA(context.Background(), mfa.Token, code(t, secret, at))
	return tokens, err
}

func TestConfirmTOTPNeedsTheRightCode(t *testing.T) {

	svc, store, u, _ := signedIn(t)
	svc.box = fakeBox{}
	ctx := context.Background()

	if _, err := svc.ConfirmTOTP(ctx, u.ID, "123456"); !errors.Is(err, ErrMFANotSetUp) {
		t.Errorf("before setup: %v, want ErrMFANotSetUp", err)
	}
	if _, err := svc.SetupTOTP(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmTOTP(ctx, u.ID, "000000"); !errors.Is(err, ErrWrongCode) {
		t.Errorf("wrong code: %v, want ErrWrongCode", err)
	}
	if store.users["reader@example.com"].TOTPEnabledAt != nil {
		t.Error("two-step sign-in turned on by a wrong code")
	}
}

func TestSignInAsksForACodeOnceOn(t *testing.T) {

	svc, _, _, secret := withMFA(t)
	now := svc.now()

	// The code of the confirmation step was used; the next step works.
	tokens, err := signInWithCode(svc, secret, now.Add(totpPeriod), t)
	if err != nil || tokens.Access.Token == "" || tokens.Refresh == "" {
		t.Fatalf("sign-in with a code = %+v, %v", tokens, err)
	}
}

func TestACodeIsGoodOnce(t *testing.T) {

	svc, _, _, secret := withMFA(t)
	next := svc.now().Add(totpPeriod)

	if _, err := signInWithCode(svc, secret, next, t); err != nil {
		t.Fatal(err)
	}
	if _, err := signInWithCode(svc, secret, next, t); !errors.Is(err, ErrWrongCode) {
		t.Errorf("same code again: %v, want ErrWrongCode", err)
	}
}

func TestPhoneClockOffByOneStep(t *testing.T) {

	svc, _, _, secret := withMFA(t)
	now := svc.now()
	svc.now = func() time.Time { return now.Add(3 * totpPeriod) }

	// A phone 30 seconds behind still signs in; 60 seconds behind does not.
	if _, err := signInWithCode(svc, secret, now.Add(2*totpPeriod), t); err != nil {
		t.Errorf("one step behind: %v", err)
	}
	svc.now = func() time.Time { return now.Add(6 * totpPeriod) }
	if _, err := signInWithCode(svc, secret, now.Add(4*totpPeriod), t); !errors.Is(err, ErrWrongCode) {
		t.Errorf("two steps behind: %v, want ErrWrongCode", err)
	}
}

func TestTooManyWrongCodesEndTheStep(t *testing.T) {

	svc, _, _, secret := withMFA(t)
	ctx := context.Background()

	_, _, err := svc.Login(ctx, "reader@example.com", "right password", "")
	var mfa *MFARequiredError
	if !errors.As(err, &mfa) {
		t.Fatal(err)
	}

	for range maxMFAAttempts {
		if _, _, err := svc.VerifyMFA(ctx, mfa.Token, "000000"); !errors.Is(err, ErrWrongCode) {
			t.Fatalf("wrong code: %v", err)
		}
	}
	right := code(t, secret, svc.now().Add(totpPeriod))
	svc.now = func() time.Time { return time.Date(2026, 10, 16, 9, 0, 30, 0, time.UTC) }
	if _, _, err := svc.VerifyMFA(ctx, mfa.Token, right); !errors.Is(err, ErrInvalidChallenge) {
		t.Errorf("right code after %d wrong ones: %v, want ErrInvalidChallenge", maxMFAAttempts, err)
	}
}

func TestTheCodeStepExpires(t *testing.T) {

	svc, _, _, secret := withMFA(t)
	ctx := context.Background()

	_, _, err := svc.Login(ctx, "reader@example.com", "right password", "")
	var mfa *MFARequiredError
	errors.As(err, &mfa)

	later := svc.now().Add(mfaChallengeTTL + time.Second)
	svc.now = func() time.Time { return later }
	if _, _, err := svc.VerifyMFA(ctx, mfa.Token, code(t, secret, later)); !errors.Is(err, ErrInvalidChallenge) {
		t.Errorf("after %v: %v, want ErrInvalidChallenge", mfaChallengeTTL, err)
	}
}

func TestSetupTOTPOnceOn(t *testing.T) {

	svc, _, u, _ := withMFA(t)

	if _, err := svc.SetupTOTP(context.Background(), u.ID); !errors.Is(err, ErrMFAAlreadyOn) {
		t.Errorf("setup again: %v, want ErrMFAAlreadyOn", err)
	}
}

func TestMFAUnavailableWithoutAKey(t *testing.T) {

	svc, _, u, _ := signedIn(t)

	if _, err := svc.SetupTOTP(context.Background(), u.ID); !errors.Is(err, ErrMFAUnavailable) {
		t.Errorf("setup without a key: %v, want ErrMFAUnavailable", err)
	}
}

func TestTurningItOnGivesTenRecoveryCodes(t *testing.T) {

	_, store, _, _ := withMFA(t)

	if len(recoveryCodes) != recoveryCodeCount || len(store.recovery) != recoveryCodeCount {
		t.Fatalf("%d codes shown, %d stored; want %d", len(recoveryCodes), len(store.recovery), recoveryCodeCount)
	}
	seen := map[string]bool{}
	for _, c := range recoveryCodes {
		if len(c) != 14 || c[4] != '-' || c[9] != '-' || seen[c] {
			t.Errorf("code %q: want unique xxxx-xxxx-xxxx", c)
		}
		seen[c] = true
		if _, stored := store.recovery[c]; stored {
			t.Error("a recovery code is stored as is")
		}
	}
}

func TestSignInWithARecoveryCodeOnce(t *testing.T) {

	svc, _, _, _ := withMFA(t)
	ctx := context.Background()

	signIn := func(code string) error {
		_, _, err := svc.Login(ctx, "reader@example.com", "right password", "")
		var mfa *MFARequiredError
		if !errors.As(err, &mfa) {
			t.Fatalf("Login: %v", err)
		}
		_, _, err = svc.VerifyMFA(ctx, mfa.Token, code)
		return err
	}

	// Typed in capitals with spaces still counts.
	typed := strings.ToUpper(strings.ReplaceAll(recoveryCodes[3], "-", " "))
	if err := signIn(typed); err != nil {
		t.Fatalf("recovery code refused: %v", err)
	}
	if err := signIn(recoveryCodes[3]); !errors.Is(err, ErrWrongCode) {
		t.Errorf("same recovery code again: %v, want ErrWrongCode", err)
	}
	if err := signIn("aaaa-bbbb-cccc"); !errors.Is(err, ErrWrongCode) {
		t.Errorf("made-up recovery code: %v, want ErrWrongCode", err)
	}
}

func TestDisableTOTP(t *testing.T) {

	svc, store, u, secret := withMFA(t)
	ctx := context.Background()
	next := code(t, secret, svc.now().Add(totpPeriod))

	if err := svc.DisableTOTP(ctx, u.ID, "wrong password", next); !errors.Is(err, ErrWrongPassword) {
		t.Errorf("wrong password: %v, want ErrWrongPassword", err)
	}
	if err := svc.DisableTOTP(ctx, u.ID, "right password", "000000"); !errors.Is(err, ErrWrongCode) {
		t.Errorf("wrong code: %v, want ErrWrongCode", err)
	}
	svc.now = func() time.Time { return time.Date(2026, 10, 16, 9, 0, 30, 0, time.UTC) }
	if err := svc.DisableTOTP(ctx, u.ID, "right password", next); err != nil {
		t.Fatal(err)
	}

	got := store.users["reader@example.com"]
	if got.TOTPEnabledAt != nil || got.TOTPSecretSealed != nil || store.recovery != nil {
		t.Errorf("after turning off: %+v, recovery %v", got, store.recovery)
	}
	if _, _, err := svc.Login(ctx, "reader@example.com", "right password", ""); err != nil {
		t.Errorf("sign-in after turning off: %v, want no code step", err)
	}
}

func TestTheTokenSaysHowTheSessionSignedIn(t *testing.T) {

	svc, _, _, secret := withMFA(t)
	issuer := svc.tokens.(*fakeIssuer)

	tokens, err := signInWithCode(svc, secret, svc.now().Add(totpPeriod), t)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(issuer.gotAMR, " ") != AMRPasswordAndCode {
		t.Errorf("amr after the code = %v", issuer.gotAMR)
	}

	// A refresh keeps it.
	issuer.gotAMR = nil
	if _, _, err := svc.Refresh(context.Background(), tokens.Refresh, ""); err != nil {
		t.Fatal(err)
	}
	if strings.Join(issuer.gotAMR, " ") != AMRPasswordAndCode {
		t.Errorf("amr after a refresh = %v", issuer.gotAMR)
	}
}
