//go:build integration

package user

import (
	"context"
	"errors"
	"os"
	"testing"

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
