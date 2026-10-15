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
	// EmailVerifiedAt is when the owner of the address confirmed it; nil
	// until then.
	EmailVerifiedAt *time.Time `db:"email_verified_at" json:"emailVerifiedAt"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"-"`

	Profile
}

// Profile is the contact and delivery details kept on the account, used to
// fill in checkout. Empty strings are details not given.
type Profile struct {
	Phone      string `db:"phone" json:"phone"`
	Address    string `db:"address" json:"address"`
	City       string `db:"city" json:"city"`
	PostalCode string `db:"postal_code" json:"postalCode"`
	Country    string `db:"country" json:"country"`
}
