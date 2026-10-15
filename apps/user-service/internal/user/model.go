package user

import "time"

// Roles a user can hold. The gateway forwards them as X-User-Roles.
const (
	RoleCustomer = "customer"
	RoleStaff    = "staff"
)

// User is one account. PasswordHash never leaves the service.
type User struct {
	// ID is a UUIDv7 string: the JWT subject, and orders.user_id in the
	// order service.
	ID              string     `db:"id" json:"id"`
	Email           string     `db:"email" json:"email"`
	PasswordHash    string     `db:"password_hash" json:"-"`
	Name            string     `db:"name" json:"name"`
	Role            string     `db:"role" json:"role"`
	EmailVerifiedAt *time.Time `db:"email_verified_at" json:"-"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"-"`
}
