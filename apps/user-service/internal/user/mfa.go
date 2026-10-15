package user

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Two-step sign-in with an authenticator app (TOTP, RFC 6238): 6 digits,
// a new code every 30 seconds, SHA-1 (what Google and Microsoft
// Authenticator read from a QR code).
const (
	totpIssuer = "Marginalleya"
	totpPeriod = 30 * time.Second

	// mfaChallengeTTL is how long the code step of a sign-in stays open.
	mfaChallengeTTL = 5 * time.Minute

	// maxMFAAttempts is how many codes one sign-in may try.
	maxMFAAttempts = 5

	// recoveryCodeCount is how many recovery codes turning it on gives.
	recoveryCodeCount = 10
)

var totpOpts = totp.ValidateOpts{Period: uint(totpPeriod.Seconds()), Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}

var (
	// ErrMFAUnavailable means the service has no key to keep TOTP secrets.
	ErrMFAUnavailable = errors.New("two-step sign-in is not available")

	// ErrMFAAlreadyOn means two-step sign-in is on already.
	ErrMFAAlreadyOn = errors.New("two-step sign-in is already on")

	// ErrMFANotSetUp means a code was sent before setup started.
	ErrMFANotSetUp = errors.New("two-step sign-in is not set up")

	// ErrWrongCode means an authenticator code is wrong, or was used already.
	ErrWrongCode = errors.New("code is incorrect")

	// ErrInvalidChallenge means the code step of a sign-in is unknown,
	// expired, done, or has had too many wrong codes: sign in again.
	ErrInvalidChallenge = errors.New("sign-in step expired")
)

// MFARequiredError is Login's answer when the password was right and the
// account asks for a code: send Token and a code to POST /auth/mfa/verify.
type MFARequiredError struct {
	Token     string
	ExpiresAt time.Time
}

func (e *MFARequiredError) Error() string { return "an authenticator code is required" }

// MFAChallenge is one mfa_challenges row.
type MFAChallenge struct {
	ID         int64      `db:"id"`
	UserID     string     `db:"user_id"`
	TokenHash  []byte     `db:"token_hash"`
	DeviceName string     `db:"device_name"`
	CreatedAt  time.Time  `db:"created_at"`
	ExpiresAt  time.Time  `db:"expires_at"`
	UsedAt     *time.Time `db:"used_at"`
	Attempts   int        `db:"attempts"`
}

// sealer keeps TOTP secrets encrypted; *secretbox.Box outside tests.
type sealer interface {
	Seal(plaintext []byte) ([]byte, error)
	Open(sealed []byte) ([]byte, error)
}

// mfaStore is the two-step sign-in part of the repository.
type mfaStore interface {
	// SaveTOTPSecret starts setup: it keeps the sealed secret, unless
	// two-step sign-in is on already (ErrMFAAlreadyOn).
	SaveTOTPSecret(ctx context.Context, userID string, sealed []byte) error
	// EnableTOTP turns two-step sign-in on, with the step of the code that
	// proved the setup and the hashes of new recovery codes, replacing any
	// older ones, in one transaction.
	EnableTOTP(ctx context.Context, userID string, step int64, recoveryHashes [][]byte) error
	// DisableTOTP turns two-step sign-in off and forgets the secret and the
	// recovery codes.
	DisableTOTP(ctx context.Context, userID string) error
	// UseRecoveryCode marks an unused recovery code of the account as used,
	// and reports false when there is none with that hash.
	UseRecoveryCode(ctx context.Context, userID string, hash []byte) (bool, error)
	// UseTOTPStep records a code's step and reports false when that step,
	// or a later one, was used already.
	UseTOTPStep(ctx context.Context, userID string, step int64) (bool, error)

	CreateMFAChallenge(ctx context.Context, c MFAChallenge) error
	FindMFAChallenge(ctx context.Context, hash []byte) (MFAChallenge, error)
	// CountMFAAttempt adds one try to an unused challenge and returns the
	// tries so far; ErrInvalidChallenge if it is used.
	CountMFAAttempt(ctx context.Context, id int64) (int, error)
	// UseMFAChallenge reports false when the challenge was used already.
	UseMFAChallenge(ctx context.Context, id int64) (bool, error)
}

// TOTPSetup is what the reader adds to their authenticator app.
type TOTPSetup struct {
	// Secret is base32, for typing in by hand.
	Secret string
	// URL is the otpauth:// link a QR code carries.
	URL string
}

// SetupTOTP starts two-step sign-in: a new secret, kept until a first code
// confirms it. Starting again replaces an unconfirmed secret.
func (s *Service) SetupTOTP(ctx context.Context, id string) (TOTPSetup, error) {

	if s.box == nil {
		return TOTPSetup{}, ErrMFAUnavailable
	}

	u, err := s.store.FindByID(ctx, id)
	if err != nil {
		return TOTPSetup{}, err
	}
	if u.TOTPEnabledAt != nil {
		return TOTPSetup{}, ErrMFAAlreadyOn
	}

	key, err := totp.Generate(totp.GenerateOpts{Issuer: totpIssuer, AccountName: u.Email})
	if err != nil {
		return TOTPSetup{}, err
	}

	sealed, err := s.box.Seal([]byte(key.Secret()))
	if err != nil {
		return TOTPSetup{}, err
	}
	if err := s.store.SaveTOTPSecret(ctx, id, sealed); err != nil {
		return TOTPSetup{}, err
	}

	return TOTPSetup{Secret: key.Secret(), URL: key.URL()}, nil
}

// ConfirmTOTP turns two-step sign-in on once a code shows the app has the
// secret, and returns recovery codes: the only time they are shown.
func (s *Service) ConfirmTOTP(ctx context.Context, id, code string) ([]string, error) {

	u, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.TOTPEnabledAt != nil {
		return nil, ErrMFAAlreadyOn
	}
	if u.TOTPSecretSealed == nil {
		return nil, ErrMFANotSetUp
	}

	step, err := s.checkCode(u, code)
	if err != nil {
		return nil, err
	}

	codes := make([]string, recoveryCodeCount)
	hashes := make([][]byte, recoveryCodeCount)
	for i := range codes {
		if codes[i], err = newRecoveryCode(); err != nil {
			return nil, err
		}
		hashes[i] = hashToken(normalizeRecoveryCode(codes[i]))
	}

	if err := s.store.EnableTOTP(ctx, id, step, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

// DisableTOTP turns two-step sign-in off. It asks for the password and a
// code (or recovery code), so a device left signed in is not enough.
func (s *Service) DisableTOTP(ctx context.Context, id, password, code string) error {

	u, err := s.store.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if u.TOTPEnabledAt == nil {
		return ErrMFANotSetUp
	}

	ok, err := verifyPassword(password, u.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return ErrWrongPassword
	}

	if err := s.secondFactor(ctx, u, code); err != nil {
		return err
	}

	return s.store.DisableTOTP(ctx, id)
}

// secondFactor accepts an authenticator code once, or an unused recovery
// code.
func (s *Service) secondFactor(ctx context.Context, u User, code string) error {

	if isRecoveryCode(code) {
		used, err := s.store.UseRecoveryCode(ctx, u.ID, hashToken(normalizeRecoveryCode(code)))
		if err != nil {
			return err
		}
		if !used {
			return ErrWrongCode
		}
		return nil
	}

	step, err := s.checkCode(u, code)
	if err != nil {
		return err
	}

	fresh, err := s.store.UseTOTPStep(ctx, u.ID, step)
	if err != nil {
		return err
	}
	if !fresh {
		// The code was right but has been used: it may have been seen.
		return ErrWrongCode
	}
	return nil
}

// Recovery codes are 12 base32 letters shown as xxxx-xxxx-xxxx: 60 random
// bits each, easy to copy by hand.
var recoveryEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func newRecoveryCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	c := strings.ToLower(recoveryEncoding.EncodeToString(b))[:12]
	return c[:4] + "-" + c[4:8] + "-" + c[8:], nil
}

// normalizeRecoveryCode ignores case, spaces and dashes, as typed.
func normalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.NewReplacer("-", "", " ", "").Replace(code))
}

// isRecoveryCode tells a recovery code (letters) from an app code (digits).
func isRecoveryCode(code string) bool {
	return len(normalizeRecoveryCode(code)) == 12
}

// challenge opens the code step of a sign-in for u on device.
func (s *Service) challenge(ctx context.Context, u User, device string) error {

	raw, hash, err := newSecret()
	if err != nil {
		return err
	}

	expires := s.now().Add(mfaChallengeTTL)
	if err := s.store.CreateMFAChallenge(ctx, MFAChallenge{
		UserID:     u.ID,
		TokenHash:  hash,
		DeviceName: truncate(device, 200),
		ExpiresAt:  expires,
	}); err != nil {
		return err
	}

	return &MFARequiredError{Token: raw, ExpiresAt: expires}
}

// VerifyMFA finishes a sign-in with an authenticator code, or a recovery
// code, and starts the session on the device the password was given on.
func (s *Service) VerifyMFA(ctx context.Context, token, code string) (User, Tokens, error) {

	c, err := s.store.FindMFAChallenge(ctx, hashToken(token))
	if errors.Is(err, ErrNotFound) {
		return User{}, Tokens{}, ErrInvalidChallenge
	}
	if err != nil {
		return User{}, Tokens{}, err
	}
	if c.UsedAt != nil || !s.now().Before(c.ExpiresAt) {
		return User{}, Tokens{}, ErrInvalidChallenge
	}

	// Counted before checking, so guessing is capped even when every guess
	// arrives at once.
	attempts, err := s.store.CountMFAAttempt(ctx, c.ID)
	if err != nil {
		return User{}, Tokens{}, err
	}
	if attempts > maxMFAAttempts {
		return User{}, Tokens{}, ErrInvalidChallenge
	}

	u, err := s.store.FindByID(ctx, c.UserID)
	if errors.Is(err, ErrNotFound) {
		return User{}, Tokens{}, ErrInvalidChallenge
	}
	if err != nil {
		return User{}, Tokens{}, err
	}

	if err := s.secondFactor(ctx, u, code); err != nil {
		return User{}, Tokens{}, err
	}

	if ok, err := s.store.UseMFAChallenge(ctx, c.ID); err != nil || !ok {
		if err == nil {
			err = ErrInvalidChallenge
		}
		return User{}, Tokens{}, err
	}

	tokens, err := s.startSession(ctx, u, c.DeviceName, AMRPasswordAndCode)
	if err != nil {
		return User{}, Tokens{}, err
	}

	return u, tokens, nil
}

// checkCode compares code with the codes of u's secret for now and one step
// either side (a phone clock a little off), and returns the step it matched.
func (s *Service) checkCode(u User, code string) (int64, error) {

	if s.box == nil {
		return 0, ErrMFAUnavailable
	}

	secret, err := s.box.Open(u.TOTPSecretSealed)
	if err != nil {
		return 0, fmt.Errorf("open TOTP secret: %w", err)
	}

	now := s.now()
	for _, skew := range []int64{0, -1, 1} {
		at := now.Add(time.Duration(skew) * totpPeriod)
		want, err := totp.GenerateCodeCustom(string(secret), at, totpOpts)
		if err != nil {
			return 0, err
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return at.Unix() / int64(totpPeriod.Seconds()), nil
		}
	}

	return 0, ErrWrongCode
}
