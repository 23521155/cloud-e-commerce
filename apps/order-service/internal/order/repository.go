package order

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
	mssql "github.com/microsoft/go-mssqldb"
)

var (
	// ErrNotFound means no order matched.
	ErrNotFound = errors.New("order not found")

	// ErrDuplicateKey means an order with this idempotency key already
	// exists: a repeated request got there first.
	ErrDuplicateKey = errors.New("idempotency key already used")

	// ErrDuplicateCode means another order drew the same code between the
	// check and the insert.
	ErrDuplicateCode = errors.New("order code already used")

	// ErrNotAwaitingPayment means the order is no longer AWAITING_PAYMENT:
	// it was already paid or cancelled, possibly by a parallel request.
	ErrNotAwaitingPayment = errors.New("order is not awaiting payment")
)

// SQL Server error numbers of a unique constraint and a unique index.
const (
	sqlUniqueConstraint = 2627
	sqlUniqueIndex      = 2601
)

// orderColumns are the columns read into an Order.
const orderColumns = `
id, code, user_id, status, payment_method, payment_status,
email, phone, full_name, address, city, postal_code, country,
subtotal_vnd, shipping_fee_vnd, expires_at, payment_ref, paid_at, created_at`

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// CodeExists reports whether an order already uses code.
func (r *Repository) CodeExists(ctx context.Context, code string) (bool, error) {

	var n int
	err := r.db.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM orders WHERE code = @code", sql.Named("code", code))

	return n > 0, err
}

// FindByIdempotencyKey returns the order placed with key, with its items,
// or ErrNotFound. key must be a UUID.
func (r *Repository) FindByIdempotencyKey(ctx context.Context, key string) (Order, error) {
	return r.findOne(ctx, "idempotency_key", key)
}

// FindByCode returns the order with code, with its items, or ErrNotFound.
func (r *Repository) FindByCode(ctx context.Context, code string) (Order, error) {
	return r.findOne(ctx, "code", code)
}

// findOne returns the order whose column equals value, with its items, or
// ErrNotFound. column must be a unique column named by this file, never
// text from a request: it is put into the query as is.
func (r *Repository) findOne(ctx context.Context, column string, value any) (Order, error) {

	var o Order
	err := r.db.GetContext(ctx, &o,
		"SELECT "+orderColumns+" FROM orders WHERE "+column+" = @value",
		sql.Named("value", value))
	if err == sql.ErrNoRows {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}

	if o.Items, err = r.items(ctx, o.ID); err != nil {
		return Order{}, err
	}

	return o, nil
}

// ListExpiredAwaiting returns up to limit AWAITING_PAYMENT orders whose hold
// has ended, oldest first, with their items.
func (r *Repository) ListExpiredAwaiting(ctx context.Context, limit int) ([]Order, error) {

	// Reads ix_orders_status_expires_at.
	var orders []Order
	if err := r.db.SelectContext(ctx, &orders, `
SELECT TOP (@limit) `+orderColumns+`
FROM orders
WHERE status = N'AWAITING_PAYMENT' AND expires_at < SYSUTCDATETIME()
ORDER BY expires_at`, sql.Named("limit", limit)); err != nil {
		return nil, err
	}

	for i := range orders {
		var err error
		if orders[i].Items, err = r.items(ctx, orders[i].ID); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

// ListByUser returns up to limit orders of one customer, newest first, with
// their items. Online orders that are still awaiting payment, or were
// cancelled without being paid, are left out: the customer never got them.
func (r *Repository) ListByUser(ctx context.Context, userID string, limit int) ([]Order, error) {

	// Reads ix_orders_user_id_created_at.
	orders := []Order{}
	if err := r.db.SelectContext(ctx, &orders, `
SELECT TOP (@limit) `+orderColumns+`
FROM orders
WHERE user_id = @user_id
  AND status <> N'AWAITING_PAYMENT'
  AND NOT (status = N'CANCELLED' AND payment_method <> N'cod' AND payment_status = N'UNPAID')
ORDER BY created_at DESC`,
		sql.Named("limit", limit), sql.Named("user_id", userID)); err != nil {
		return nil, err
	}

	for i := range orders {
		var err error
		if orders[i].Items, err = r.items(ctx, orders[i].ID); err != nil {
			return nil, err
		}
	}

	return orders, nil
}

// items returns the items of the order with id, in the order they were added.
func (r *Repository) items(ctx context.Context, id int64) ([]Item, error) {

	items := []Item{}
	err := r.db.SelectContext(ctx, &items, `
SELECT book_id, book_slug, title, author, cover_url, price_vnd
FROM order_items WHERE order_id = @id ORDER BY id`, sql.Named("id", id))

	return items, err
}

// Insert writes o and its items in one transaction, under the idempotency
// key key, and fills in o.ID and o.CreatedAt. It returns ErrDuplicateKey or
// ErrDuplicateCode when another order already has that key or code.
func (r *Repository) Insert(ctx context.Context, key string, o *Order) error {

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // a no-op once committed

	row := tx.QueryRowxContext(ctx, `
INSERT INTO orders (
    code, idempotency_key, user_id, status, payment_method, payment_status,
    email, phone, full_name, address, city, postal_code, country,
    subtotal_vnd, shipping_fee_vnd, expires_at)
OUTPUT inserted.id, inserted.created_at
VALUES (
    @code, @key, @user_id, @status, @payment_method, @payment_status,
    @email, @phone, @full_name, @address, @city, @postal_code, @country,
    @subtotal_vnd, @shipping_fee_vnd, @expires_at)`,
		sql.Named("code", o.Code),
		sql.Named("key", key),
		sql.Named("user_id", o.UserID),
		sql.Named("status", o.Status),
		sql.Named("payment_method", o.PaymentMethod),
		sql.Named("payment_status", o.PaymentStatus),
		sql.Named("email", o.Email),
		sql.Named("phone", o.Phone),
		sql.Named("full_name", o.FullName),
		sql.Named("address", o.Address),
		sql.Named("city", o.City),
		sql.Named("postal_code", o.PostalCode),
		sql.Named("country", o.Country),
		sql.Named("subtotal_vnd", o.SubtotalVND),
		sql.Named("shipping_fee_vnd", o.ShippingFeeVND),
		sql.Named("expires_at", o.ExpiresAt),
	)
	if err := row.Scan(&o.ID, &o.CreatedAt); err != nil {
		return duplicateOr(err)
	}

	for _, it := range o.Items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO order_items (order_id, book_id, book_slug, title, author, cover_url, price_vnd)
VALUES (@order_id, @book_id, @book_slug, @title, @author, @cover_url, @price_vnd)`,
			sql.Named("order_id", o.ID),
			sql.Named("book_id", it.BookID),
			sql.Named("book_slug", it.Slug),
			sql.Named("title", it.Title),
			sql.Named("author", it.Author),
			sql.Named("cover_url", it.CoverURL),
			sql.Named("price_vnd", it.PriceVND),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Cancel marks an order CANCELLED with reason. Cancelling it again only
// rewrites the reason.
func (r *Repository) Cancel(ctx context.Context, id int64, reason string) error {

	_, err := r.db.ExecContext(ctx, `
UPDATE orders
SET status = N'CANCELLED', cancel_reason = @reason, updated_at = SYSUTCDATETIME()
WHERE id = @id`, sql.Named("id", id), sql.Named("reason", reason))

	return err
}

// MarkPaid records that the gateway confirmed the payment of an
// AWAITING_PAYMENT order: CONFIRMED, PAID, with ref and the time.
// ErrNotAwaitingPayment when the order is no longer awaiting payment.
//
// The status is checked by the UPDATE itself, not by a SELECT before it: the
// IPN call and the return redirect can arrive at the same moment, and SQL
// Server runs the two UPDATEs one after the other, so only one finds the
// order still awaiting payment.
func (r *Repository) MarkPaid(ctx context.Context, id int64, ref string) error {

	res, err := r.db.ExecContext(ctx, `
UPDATE orders
SET status = N'CONFIRMED', payment_status = N'PAID', payment_ref = @ref,
    paid_at = SYSUTCDATETIME(), updated_at = SYSUTCDATETIME()
WHERE id = @id AND status = N'AWAITING_PAYMENT'`,
		sql.Named("id", id), sql.Named("ref", ref))
	if err != nil {
		return err
	}

	return oneRowOr(res, ErrNotAwaitingPayment)
}

// CancelAwaiting marks an AWAITING_PAYMENT order CANCELLED with reason: the
// payment failed or its hold ran out. ErrNotAwaitingPayment when the order
// is no longer awaiting payment; it is then left as it is. Unlike Cancel, it
// never undoes a payment a parallel request has just confirmed.
func (r *Repository) CancelAwaiting(ctx context.Context, id int64, reason string) error {

	res, err := r.db.ExecContext(ctx, `
UPDATE orders
SET status = N'CANCELLED', cancel_reason = @reason, updated_at = SYSUTCDATETIME()
WHERE id = @id AND status = N'AWAITING_PAYMENT'`,
		sql.Named("id", id), sql.Named("reason", reason))
	if err != nil {
		return err
	}

	return oneRowOr(res, ErrNotAwaitingPayment)
}

// oneRowOr returns notFound when res changed no row.
func oneRowOr(res sql.Result, notFound error) error {

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}

	return nil
}

// duplicateOr turns a unique violation on the idempotency key or the code
// into ErrDuplicateKey or ErrDuplicateCode, and returns any other err as is.
func duplicateOr(err error) error {

	var sqlErr mssql.Error
	if !errors.As(err, &sqlErr) ||
		(sqlErr.Number != sqlUniqueConstraint && sqlErr.Number != sqlUniqueIndex) {
		return err
	}

	// The message names the constraint that was violated.
	switch {
	case strings.Contains(sqlErr.Message, "uq_orders_idempotency_key"):
		return ErrDuplicateKey
	case strings.Contains(sqlErr.Message, "uq_orders_code"):
		return ErrDuplicateCode
	}

	return err
}
