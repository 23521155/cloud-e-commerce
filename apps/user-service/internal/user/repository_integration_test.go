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
