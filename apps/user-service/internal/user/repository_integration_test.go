//go:build integration

package user

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/fizzisme/user-service/internal/database"
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

	got, err := repo.Create(context.Background(), u)
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

	if _, err := repo.Create(context.Background(), u); err != nil {
		t.Fatal(err)
	}

	again := u
	again.ID = uuid.Must(uuid.NewV7()).String()

	if _, err := repo.Create(context.Background(), again); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

func TestFindByEmailAndID(t *testing.T) {
	repo := newTestRepo(t)
	u := newUser(t, repo)
	ctx := context.Background()

	if _, err := repo.Create(ctx, u); err != nil {
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

	created, err := repo.Create(ctx, u)
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

	if _, err := repo.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	family := uuid.NewString()
	_, hash, err := newRefreshToken()
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

	if _, err := repo.Create(ctx, u); err != nil {
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
		_, hashes[i], _ = newRefreshToken()
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
