// Command set-role gives an account the staff role, or takes it back. Staff
// sign in to the shop office (/admin) and must turn on two-step sign-in
// first. The account is made the usual way, by registering on the shop.
//
//	set-role -email someone@example.com -role staff
//	set-role -email someone@example.com -role customer
//
// The account's sessions end, so the next sign-in carries the new role. It
// reads DATABASE_URL like the service.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fizzisme/user-service/internal/config"
	"github.com/fizzisme/user-service/internal/database"
	"github.com/fizzisme/user-service/internal/user"
)

func main() {
	email := flag.String("email", "", "email of the account (it must exist)")
	role := flag.String("role", user.RoleStaff, "staff or customer")
	flag.Parse()

	if err := run(strings.ToLower(strings.TrimSpace(*email)), *role); err != nil {
		fmt.Fprintln(os.Stderr, "set-role:", err)
		os.Exit(1)
	}
}

func run(email, role string) error {

	if email == "" {
		return errors.New("-email is required")
	}
	if role != user.RoleStaff && role != user.RoleCustomer {
		return fmt.Errorf("-role must be %s or %s", user.RoleStaff, user.RoleCustomer)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	u, err := user.NewRepository(db).SetRole(ctx, email, role)
	if errors.Is(err, user.ErrNotFound) {
		return fmt.Errorf("no account with email %s: register it on the shop first", email)
	}
	if err != nil {
		return err
	}

	fmt.Printf("%s is now %s; their sessions have ended.\n", u.Email, u.Role)
	if role == user.RoleStaff && u.TOTPEnabledAt == nil {
		fmt.Println("They must turn on two-step sign-in (account page) before the shop office lets them in.")
	}
	return nil
}
