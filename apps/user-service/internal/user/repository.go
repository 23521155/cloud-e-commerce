package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fizzisme/user-service/internal/outbox"
	"github.com/jmoiron/sqlx"
	mssql "github.com/microsoft/go-mssqldb"
)

// SQL Server error numbers for a duplicate key.
const (
	sqlUniqueConstraint = 2627
	sqlUniqueIndex      = 2601
)

// columns are the columns read into a User.
var columns = []string{
	"id", "email", "password_hash", "name", "role", "email_verified_at", "created_at", "updated_at",
	"phone", "address", "city", "postal_code", "country",
}

// userColumns is columns for a SELECT.
var userColumns = strings.Join(columns, ", ")

// outputColumns is columns for an OUTPUT clause, e.g. INSERTED.id.
func outputColumns(table string) string {
	prefixed := make([]string, len(columns))
	for i, c := range columns {
		prefixed[i] = table + "." + c
	}
	return strings.Join(prefixed, ", ")
}

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts u with the link that confirms its email, and the outbox
// messages that send it, in one transaction. It returns the stored row,
// with the timestamps the database set. A taken email gives ErrEmailTaken.
func (r *Repository) Create(ctx context.Context, u User, verify EmailToken, messages ...outbox.Message) (User, error) {

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()

	var created User
	err = tx.GetContext(ctx, &created, `
INSERT INTO users (id, email, password_hash, name, role)
OUTPUT `+outputColumns("INSERTED")+`
VALUES (@p1, @p2, @p3, @p4, @p5)`,
		u.ID, u.Email, u.PasswordHash, u.Name, u.Role,
	)
	if isUniqueViolation(err) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}

	if err := insertEmailToken(ctx, tx, verify); err != nil {
		return User{}, err
	}
	if err := outbox.Add(ctx, tx, messages...); err != nil {
		return User{}, err
	}

	return created, tx.Commit()
}

func insertEmailToken(ctx context.Context, tx *sqlx.Tx, t EmailToken) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO email_tokens (user_id, purpose, email, token_hash, expires_at)
VALUES (@user_id, @purpose, @email, @token_hash, @expires_at)`,
		sql.Named("user_id", t.UserID),
		sql.Named("purpose", t.Purpose),
		sql.Named("email", t.Email),
		sql.Named("token_hash", t.TokenHash),
		sql.Named("expires_at", t.ExpiresAt.UTC()),
	)
	return err
}

// UpdateProfile sets the name and profile of the account with id and
// returns the stored row, or ErrNotFound.
func (r *Repository) UpdateProfile(ctx context.Context, id, name string, p Profile) (User, error) {
	return r.findOne(ctx, `
UPDATE users
SET name = @name, phone = @phone, address = @address, city = @city,
    postal_code = @postal_code, country = @country, updated_at = SYSUTCDATETIME()
OUTPUT `+outputColumns("INSERTED")+`
WHERE id = @id`,
		sql.Named("id", id), sql.Named("name", name),
		sql.Named("phone", p.Phone), sql.Named("address", p.Address), sql.Named("city", p.City),
		sql.Named("postal_code", p.PostalCode), sql.Named("country", p.Country),
	)
}

// UpdatePassword stores a new password hash for the account with id, or
// returns ErrNotFound.
func (r *Repository) UpdatePassword(ctx context.Context, id, hash string) error {

	res, err := r.db.ExecContext(ctx, `
UPDATE users SET password_hash = @hash, updated_at = SYSUTCDATETIME() WHERE id = @id`,
		sql.Named("id", id), sql.Named("hash", hash))
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		err = ErrNotFound
	}
	return err
}

// FindByEmail returns the account with a normalized email, or ErrNotFound.
func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	return r.findOne(ctx, `SELECT `+userColumns+` FROM users WHERE email = @p1`, email)
}

// FindByID returns the account with an id, or ErrNotFound.
func (r *Repository) FindByID(ctx context.Context, id string) (User, error) {
	return r.findOne(ctx, `SELECT `+userColumns+` FROM users WHERE id = @p1`, id)
}

func (r *Repository) findOne(ctx context.Context, query string, args ...any) (User, error) {
	var u User
	err := r.db.GetContext(ctx, &u, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// isUniqueViolation reports whether err is SQL Server refusing a duplicate
// key. The only unique key a client can collide on is the email.
func isUniqueViolation(err error) bool {
	var sqlErr mssql.Error
	return errors.As(err, &sqlErr) &&
		(sqlErr.Number == sqlUniqueConstraint || sqlErr.Number == sqlUniqueIndex)
}
