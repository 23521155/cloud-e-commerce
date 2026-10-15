package user

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	mssql "github.com/microsoft/go-mssqldb"
)

// SQL Server error numbers for a duplicate key.
const (
	sqlUniqueConstraint = 2627
	sqlUniqueIndex      = 2601
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts u and returns the stored row, with the timestamps the
// database set. A taken email gives ErrEmailTaken.
func (r *Repository) Create(ctx context.Context, u User) (User, error) {

	var created User
	err := r.db.GetContext(ctx, &created, `
INSERT INTO users (id, email, password_hash, name, role)
OUTPUT INSERTED.id, INSERTED.email, INSERTED.password_hash, INSERTED.name, INSERTED.role,
       INSERTED.email_verified_at, INSERTED.created_at, INSERTED.updated_at
VALUES (@p1, @p2, @p3, @p4, @p5)`,
		u.ID, u.Email, u.PasswordHash, u.Name, u.Role,
	)
	if isUniqueViolation(err) {
		return User{}, ErrEmailTaken
	}

	return created, err
}

// isUniqueViolation reports whether err is SQL Server refusing a duplicate
// key. The only unique key a client can collide on is the email.
func isUniqueViolation(err error) bool {
	var sqlErr mssql.Error
	return errors.As(err, &sqlErr) &&
		(sqlErr.Number == sqlUniqueConstraint || sqlErr.Number == sqlUniqueIndex)
}
