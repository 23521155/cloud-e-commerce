//go:build integration

package user

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/fizzisme/user-service/internal/database"
	"github.com/fizzisme/user-service/internal/outbox"
	"github.com/google/uuid"
)

// Run with the migrated database up:
//
//	DATABASE_URL=sqlserver://... go test -tags integration ./internal/user/
func newTestRepo(t *testing.T) *Repository {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL is not set")
	}

	db, err := database.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	return NewRepository(db)
}

// newUser returns a user with a fresh id and email, removed after the test.
func newUser(t *testing.T, repo *Repository) User {
	t.Helper()

	id := uuid.Must(uuid.NewV7()).String()
	email := "it-" + id + "@example.com"

	t.Cleanup(func() {
		repo.db.ExecContext(context.Background(), `DELETE FROM users WHERE email = @p1`, email)
	})

	return User{ID: id, Email: email, PasswordHash: "$argon2id$test", Name: "Integration", Role: RoleCustomer}
}

func TestCreateReturnsStoredRow(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)

	got, err := repo.Create(context.Background(), u, testVerification(u))
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != u.ID || got.Email != u.Email || got.Role != RoleCustomer {
		t.Errorf("got %+v, want the inserted values", got)
	}
	if got.CreatedAt.IsZero() || got.EmailVerifiedAt != nil {
		t.Errorf("created_at = %v, email_verified_at = %v; want set, NULL", got.CreatedAt, got.EmailVerifiedAt)
	}
}

func TestCreateRefusesDuplicateEmail(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)

	if _, err := repo.Create(context.Background(), u, testVerification(u)); err != nil {
		t.Fatal(err)
	}

	again := u
	again.ID = uuid.Must(uuid.NewV7()).String()

	if _, err := repo.Create(context.Background(), again, testVerification(again)); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

func TestFindByEmailAndID(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u, testVerification(u)); err != nil {
		t.Fatal(err)
	}

	byEmail, err := repo.FindByEmail(ctx, u.Email)
	if err != nil || byEmail.ID != u.ID || byEmail.PasswordHash != u.PasswordHash {
		t.Errorf("FindByEmail = %+v, %v", byEmail, err)
	}

	byID, err := repo.FindByID(ctx, u.ID)
	if err != nil || byID.Email != u.Email {
		t.Errorf("FindByID = %+v, %v", byID, err)
	}

	if _, err := repo.FindByEmail(ctx, "missing-"+u.Email); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing email error = %v, want ErrNotFound", err)
	}
	if _, err := repo.FindByID(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing id error = %v, want ErrNotFound", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	created, err := repo.Create(ctx, u, testVerification(u))
	if err != nil {
		t.Fatal(err)
	}
	if created.Profile != (Profile{}) {
		t.Errorf("new account profile = %+v, want empty", created.Profile)
	}

	want := Profile{Phone: "0901234567", Address: "12 Lê Lợi", City: "Huế", PostalCode: "530000", Country: "Vietnam"}
	got, err := repo.UpdateProfile(ctx, u.ID, "Trần Thị Bình", want)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Trần Thị Bình" || got.Profile != want || got.UpdatedAt.Before(created.UpdatedAt) {
		t.Errorf("updated = %+v", got)
	}

	if _, err := repo.UpdateProfile(ctx, "missing", "R", Profile{}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing id error = %v, want ErrNotFound", err)
	}
}

func TestRefreshTokenLifecycle(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u, testVerification(u)); err != nil {
		t.Fatal(err)
	}

	family := uuid.NewString()
	_, hash, err := newSecret()
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.CreateRefreshToken(ctx, RefreshToken{UserID: u.ID, FamilyID: family, TokenHash: hash, DeviceName: "Firefox on Windows", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindRefreshToken(ctx, hash)
	if err != nil || got.UserID != u.ID || got.FamilyID != family || got.UsedAt != nil || got.RevokedAt != nil {
		t.Fatalf("FindRefreshToken = %+v, %v", got, err)
	}

	marked, err := repo.MarkRefreshTokenUsed(ctx, got.ID)
	if err != nil || !marked {
		t.Errorf("first MarkRefreshTokenUsed = %v, %v; want true", marked, err)
	}
	marked, err = repo.MarkRefreshTokenUsed(ctx, got.ID)
	if err != nil || marked {
		t.Errorf("second MarkRefreshTokenUsed = %v, %v; want false", marked, err)
	}

	if err := repo.RevokeFamily(ctx, family); err != nil {
		t.Fatal(err)
	}
	if got, _ = repo.FindRefreshToken(ctx, hash); got.RevokedAt == nil || got.UsedAt == nil {
		t.Errorf("after revoke: %+v", got)
	}

	if _, err := repo.FindRefreshToken(ctx, hashToken("unknown")); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown token error = %v, want ErrNotFound", err)
	}
}

func TestUpdatePasswordAndRevokeAllSessions(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u, testVerification(u)); err != nil {
		t.Fatal(err)
	}

	if err := repo.UpdatePassword(ctx, u.ID, "$argon2id$new"); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.FindByID(ctx, u.ID); got.PasswordHash != "$argon2id$new" {
		t.Errorf("hash = %q, want the new one", got.PasswordHash)
	}
	if err := repo.UpdatePassword(ctx, "missing", "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing id error = %v, want ErrNotFound", err)
	}

	hashes := make([][]byte, 2)
	for i := range hashes {
		_, hashes[i], _ = newSecret()
		if err := repo.CreateRefreshToken(ctx, RefreshToken{UserID: u.ID, FamilyID: uuid.NewString(), TokenHash: hashes[i], ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.RevokeAllSessions(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	for _, h := range hashes {
		if got, _ := repo.FindRefreshToken(ctx, h); got.RevokedAt == nil {
			t.Errorf("token of family %s not revoked", got.FamilyID)
		}
	}
}

// testVerification returns a confirmation link token for u.
func testVerification(u User) EmailToken {
	_, hash, _ := newSecret()
	return EmailToken{UserID: u.ID, Purpose: PurposeVerifyEmail, Email: u.Email, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}
}

func TestCreateWritesTheLinkAndTheEmailWithTheAccount(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	msg, err := outbox.NewEmail(outbox.Email{To: u.Email, Subject: "it", Text: "it"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, u, testVerification(u), msg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.db.ExecContext(ctx, `DELETE FROM outbox WHERE payload LIKE @p1`, "%"+u.Email+"%") })

	var tokens, messages int
	repo.db.GetContext(ctx, &tokens, `SELECT COUNT(*) FROM email_tokens WHERE user_id = @p1`, u.ID)
	repo.db.GetContext(ctx, &messages, `SELECT COUNT(*) FROM outbox WHERE payload LIKE @p1`, "%"+u.Email+"%")
	if tokens != 1 || messages != 1 {
		t.Errorf("tokens %d, messages %d; want 1 and 1", tokens, messages)
	}

	// A refused account leaves neither behind.
	again := u
	again.ID = uuid.Must(uuid.NewV7()).String()
	if _, err := repo.Create(ctx, again, testVerification(again), msg); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
	repo.db.GetContext(ctx, &messages, `SELECT COUNT(*) FROM outbox WHERE payload LIKE @p1`, "%"+u.Email+"%")
	if messages != 1 {
		t.Errorf("messages = %d after a refused account, want still 1", messages)
	}
}

func TestVerifyEmailUsesTheLinkOnce(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	raw, hash, _ := newSecret()
	verify := EmailToken{UserID: u.ID, Purpose: PurposeVerifyEmail, Email: u.Email, TokenHash: hash, ExpiresAt: time.Now().Add(time.Hour)}
	if _, err := repo.Create(ctx, u, verify); err != nil {
		t.Fatal(err)
	}

	noEvents := func(User) ([]outbox.Message, error) { return nil, nil }

	got, err := repo.VerifyEmail(ctx, hashToken(raw), noEvents)
	if err != nil || got.EmailVerifiedAt == nil {
		t.Fatalf("VerifyEmail = %+v, %v", got, err)
	}
	if _, err := repo.VerifyEmail(ctx, hashToken(raw), noEvents); !errors.Is(err, ErrInvalidEmailToken) {
		t.Errorf("second use error = %v, want ErrInvalidEmailToken", err)
	}

	at, ok, err := repo.LastEmailTokenAt(ctx, u.ID, PurposeVerifyEmail)
	if err != nil || !ok || at.IsZero() {
		t.Errorf("LastEmailTokenAt = %v, %v, %v", at, ok, err)
	}
	if _, ok, _ := repo.LastEmailTokenAt(ctx, "missing", PurposeVerifyEmail); ok {
		t.Error("LastEmailTokenAt found a token for no account")
	}
}

func TestVerifyEmailExpiredLink(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	raw, hash, _ := newSecret()
	expired := EmailToken{UserID: u.ID, Purpose: PurposeVerifyEmail, Email: u.Email, TokenHash: hash, ExpiresAt: time.Now().Add(-time.Minute)}
	if _, err := repo.Create(ctx, u, expired); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.VerifyEmail(ctx, hashToken(raw), func(User) ([]outbox.Message, error) { return nil, nil }); !errors.Is(err, ErrInvalidEmailToken) {
		t.Errorf("expired link error = %v, want ErrInvalidEmailToken", err)
	}
	if got, _ := repo.FindByID(ctx, u.ID); got.EmailVerifiedAt != nil {
		t.Error("an expired link confirmed the address")
	}
}

func TestResetPassword(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u, testVerification(u)); err != nil {
		t.Fatal(err)
	}

	raw, hash, _ := newSecret()
	_, other, _ := newSecret()
	for _, h := range [][]byte{hash, other} {
		if err := repo.CreateEmailToken(ctx, EmailToken{UserID: u.ID, Purpose: PurposeResetPassword, Email: u.Email, TokenHash: h, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	_, session, _ := newSecret()
	if err := repo.CreateRefreshToken(ctx, RefreshToken{UserID: u.ID, FamilyID: uuid.NewString(), TokenHash: session, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	if err := repo.ResetPassword(ctx, hashToken(raw), "$argon2id$reset"); err != nil {
		t.Fatal(err)
	}

	if got, _ := repo.FindByID(ctx, u.ID); got.PasswordHash != "$argon2id$reset" {
		t.Errorf("password hash = %q", got.PasswordHash)
	}
	if got, _ := repo.FindRefreshToken(ctx, session); got.RevokedAt == nil {
		t.Error("session not revoked by the reset")
	}
	// The link, and the other link sent earlier, are both used up.
	if err := repo.ResetPassword(ctx, hashToken(raw), "x"); !errors.Is(err, ErrInvalidEmailToken) {
		t.Errorf("second use error = %v", err)
	}
	if err := repo.ResetPassword(ctx, other, "x"); !errors.Is(err, ErrInvalidEmailToken) {
		t.Errorf("older link error = %v", err)
	}
}

func TestTOTPAndChallenges(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u, testVerification(u)); err != nil {
		t.Fatal(err)
	}

	if err := repo.SaveTOTPSecret(ctx, u.ID, []byte("sealed")); err != nil {
		t.Fatal(err)
	}
	_, code1, _ := newSecret()
	_, code2, _ := newSecret()
	if err := repo.EnableTOTP(ctx, u.ID, 100, [][]byte{code1, code2}); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.FindByID(ctx, u.ID)
	if string(got.TOTPSecretSealed) != "sealed" || got.TOTPEnabledAt == nil || got.TOTPLastStep == nil || *got.TOTPLastStep != 100 {
		t.Errorf("after enabling: %+v", got)
	}
	if err := repo.SaveTOTPSecret(ctx, u.ID, []byte("other")); !errors.Is(err, ErrMFAAlreadyOn) {
		t.Errorf("new secret while on: %v, want ErrMFAAlreadyOn", err)
	}

	for _, tc := range []struct {
		step int64
		want bool
	}{{100, false}, {99, false}, {101, true}, {101, false}} {
		if ok, err := repo.UseTOTPStep(ctx, u.ID, tc.step); err != nil || ok != tc.want {
			t.Errorf("UseTOTPStep(%d) = %v, %v; want %v", tc.step, ok, err, tc.want)
		}
	}

	_, hash, _ := newSecret()
	if err := repo.CreateMFAChallenge(ctx, MFAChallenge{UserID: u.ID, TokenHash: hash, DeviceName: "Firefox", ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	c, err := repo.FindMFAChallenge(ctx, hash)
	if err != nil || c.UserID != u.ID || c.Attempts != 0 {
		t.Fatalf("FindMFAChallenge = %+v, %v", c, err)
	}
	if n, err := repo.CountMFAAttempt(ctx, c.ID); err != nil || n != 1 {
		t.Errorf("CountMFAAttempt = %d, %v", n, err)
	}
	if ok, _ := repo.UseMFAChallenge(ctx, c.ID); !ok {
		t.Error("first use refused")
	}
	if ok, _ := repo.UseMFAChallenge(ctx, c.ID); ok {
		t.Error("second use accepted")
	}
	if _, err := repo.CountMFAAttempt(ctx, c.ID); !errors.Is(err, ErrInvalidChallenge) {
		t.Errorf("attempt on a used step: %v, want ErrInvalidChallenge", err)
	}
}

func TestRecoveryCodesAndDisable(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u, testVerification(u)); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveTOTPSecret(ctx, u.ID, []byte("sealed")); err != nil {
		t.Fatal(err)
	}

	_, code1, _ := newSecret()
	if err := repo.EnableTOTP(ctx, u.ID, 1, [][]byte{code1}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := repo.UseRecoveryCode(ctx, u.ID, code1); !ok {
		t.Error("first use refused")
	}
	if ok, _ := repo.UseRecoveryCode(ctx, u.ID, code1); ok {
		t.Error("second use accepted")
	}

	if err := repo.DisableTOTP(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.FindByID(ctx, u.ID)
	if got.TOTPEnabledAt != nil || got.TOTPSecretSealed != nil {
		t.Errorf("after disabling: %+v", got)
	}
	var left int
	repo.db.GetContext(ctx, &left, `SELECT COUNT(*) FROM recovery_codes WHERE user_id = @p1`, u.ID)
	if left != 0 {
		t.Errorf("%d recovery codes left after disabling", left)
	}
}

func TestSetRole(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u, testVerification(u)); err != nil {
		t.Fatal(err)
	}
	_, session, _ := newSecret()
	if err := repo.CreateRefreshToken(ctx, RefreshToken{UserID: u.ID, FamilyID: uuid.NewString(), TokenHash: session, ExpiresAt: time.Now().Add(time.Hour), AMR: AMRPassword}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.SetRole(ctx, u.Email, RoleStaff)
	if err != nil || got.Role != RoleStaff {
		t.Fatalf("SetRole = %+v, %v", got, err)
	}
	if rt, _ := repo.FindRefreshToken(ctx, session); rt.RevokedAt == nil || rt.AMR != AMRPassword {
		t.Errorf("session after the role change: %+v", rt)
	}
	if _, err := repo.SetRole(ctx, "missing-"+u.Email, RoleStaff); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown email: %v, want ErrNotFound", err)
	}
}
