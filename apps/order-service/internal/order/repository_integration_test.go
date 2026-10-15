//go:build integration

package order

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fizzisme/order-service/internal/database"
)

// Run with the migrated database up:
//
//	DATABASE_URL=sqlserver://... go test -tags integration ./internal/order/
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

// newKey returns a random UUID (version 4) for an idempotency key.
func newKey() string {

	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// newTestOrder returns a guest cash-on-delivery order with a code no other
// order uses, so tests can run against a database that already has orders.
func newTestOrder(t *testing.T, repo *Repository) *Order {
	t.Helper()

	for range 5 {
		code := newCode()
		taken, err := repo.CodeExists(context.Background(), code)
		if err != nil {
			t.Fatal(err)
		}
		if taken {
			continue
		}

		author := "Paulo Coelho"
		return &Order{
			Code:          code,
			Status:        StatusConfirmed,
			PaymentMethod: PaymentCOD,
			PaymentStatus: PaymentUnpaid,
			Email:         "guest@example.com",
			Phone:         "0901234567",
			FullName:      "Nguyễn Văn A",
			Address:       "12 Lê Lợi",
			City:          "Hồ Chí Minh",
			Country:       "Vietnam",
			SubtotalVND:   200000,
			Items: []Item{
				{BookID: 1, Slug: "nha-gia-kim", Title: "Nhà giả kim", Author: &author, PriceVND: 120000},
				{BookID: 2, Slug: "dac-nhan-tam", Title: "Đắc nhân tâm", PriceVND: 80000},
			},
		}
	}

	t.Fatal("no free order code after 5 tries")
	return nil
}

func TestInsertThenFindByIdempotencyKey(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	key := newKey()
	o := newTestOrder(t, repo)

	if err := repo.Insert(ctx, key, o); err != nil {
		t.Fatal(err)
	}
	if o.ID == 0 || o.CreatedAt.IsZero() {
		t.Fatalf("Insert did not fill in ID and CreatedAt: %d, %v", o.ID, o.CreatedAt)
	}

	got, err := repo.FindByIdempotencyKey(ctx, key)
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != o.ID || got.Code != o.Code || got.UserID != nil || got.SubtotalVND != o.SubtotalVND {
		t.Errorf("got %+v, want the inserted order %+v", got, o)
	}
	if len(got.Items) != 2 || got.Items[0].Slug != "nha-gia-kim" || got.Items[1].Author != nil {
		t.Errorf("items = %+v, want the two inserted items in order", got.Items)
	}

	taken, err := repo.CodeExists(ctx, o.Code)
	if err != nil {
		t.Fatal(err)
	}
	if !taken {
		t.Errorf("CodeExists(%q) = false after the insert", o.Code)
	}
}

func TestFindByIdempotencyKeyNotFound(t *testing.T) {
	repo := newTestRepo(t)

	if _, err := repo.FindByIdempotencyKey(context.Background(), newKey()); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestInsertDuplicates(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	first := newTestOrder(t, repo)
	key := newKey()
	if err := repo.Insert(ctx, key, first); err != nil {
		t.Fatal(err)
	}

	sameKey := newTestOrder(t, repo)
	if err := repo.Insert(ctx, key, sameKey); !errors.Is(err, ErrDuplicateKey) {
		t.Errorf("same key: err = %v, want ErrDuplicateKey", err)
	}

	sameCode := newTestOrder(t, repo)
	sameCode.Code = first.Code
	if err := repo.Insert(ctx, newKey(), sameCode); !errors.Is(err, ErrDuplicateCode) {
		t.Errorf("same code: err = %v, want ErrDuplicateCode", err)
	}
}

func TestCancel(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	key := newKey()
	o := newTestOrder(t, repo)
	if err := repo.Insert(ctx, key, o); err != nil {
		t.Fatal(err)
	}

	if err := repo.Cancel(ctx, o.ID, "books could not be sold"); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByIdempotencyKey(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusCancelled {
		t.Errorf("status = %q, want %q", got.Status, StatusCancelled)
	}
}

// insertAwaiting writes an online order that waits for its payment.
func insertAwaiting(t *testing.T, repo *Repository) *Order {
	t.Helper()

	o := newTestOrder(t, repo)
	expires := time.Now().UTC().Add(15 * time.Minute)
	o.Status = StatusAwaitingPayment
	o.PaymentMethod = PaymentVietQR
	o.ExpiresAt = &expires

	if err := repo.Insert(context.Background(), newKey(), o); err != nil {
		t.Fatal(err)
	}

	return o
}

func TestFindByCode(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	o := insertAwaiting(t, repo)

	got, err := repo.FindByCode(ctx, o.Code)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != o.ID || got.Status != StatusAwaitingPayment || got.ExpiresAt == nil {
		t.Errorf("got %+v, want the inserted order %+v", got, o)
	}
	if len(got.Items) != 2 || got.Items[0].Slug != "nha-gia-kim" {
		t.Errorf("items = %+v, want the two inserted items in order", got.Items)
	}
	if got.PaymentRef != nil || got.PaidAt != nil {
		t.Errorf("an unpaid order has payment_ref %v and paid_at %v", got.PaymentRef, got.PaidAt)
	}
}

func TestFindByCodeNotFound(t *testing.T) {
	repo := newTestRepo(t)

	// Codes are upper case; this one cannot be drawn.
	if _, err := repo.FindByCode(context.Background(), "no-such-code"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestMarkPaid(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	o := insertAwaiting(t, repo)

	if err := repo.MarkPaid(ctx, o.ID, "14567890"); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByCode(ctx, o.Code)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusConfirmed || got.PaymentStatus != PaymentPaid {
		t.Errorf("status/payment = %q/%q, want %q/%q", got.Status, got.PaymentStatus, StatusConfirmed, PaymentPaid)
	}
	if got.PaymentRef == nil || *got.PaymentRef != "14567890" {
		t.Errorf("payment_ref = %v, want 14567890", got.PaymentRef)
	}
	if got.PaidAt == nil {
		t.Error("paid_at is not set")
	}

	// A second confirmation of the same payment finds nothing to do.
	if err := repo.MarkPaid(ctx, o.ID, "14567890"); !errors.Is(err, ErrNotAwaitingPayment) {
		t.Errorf("second MarkPaid: err = %v, want ErrNotAwaitingPayment", err)
	}
}

func TestMarkPaidRefusesAnOrderThatWasNotAwaiting(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// A cash-on-delivery order is CONFIRMED from the start.
	o := newTestOrder(t, repo)
	if err := repo.Insert(ctx, newKey(), o); err != nil {
		t.Fatal(err)
	}

	if err := repo.MarkPaid(ctx, o.ID, "14567890"); !errors.Is(err, ErrNotAwaitingPayment) {
		t.Errorf("err = %v, want ErrNotAwaitingPayment", err)
	}
}

// The IPN call and the return redirect may confirm the same payment at once:
// exactly one of them must win.
func TestMarkPaidOnlyOneOfParallelCallsWins(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	o := insertAwaiting(t, repo)

	const calls = 8
	errs := make([]error, calls)

	var wg sync.WaitGroup
	for i := range calls {
		wg.Go(func() {
			errs[i] = repo.MarkPaid(ctx, o.ID, "14567890")
		})
	}
	wg.Wait()

	won := 0
	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, ErrNotAwaitingPayment):
			t.Errorf("unexpected err = %v", err)
		}
	}
	if won != 1 {
		t.Errorf("%d calls won, want exactly 1", won)
	}
}

func TestCancelAwaiting(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	o := insertAwaiting(t, repo)

	if err := repo.CancelAwaiting(ctx, o.ID, "payment failed"); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByCode(ctx, o.Code)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusCancelled || got.PaymentStatus != PaymentUnpaid {
		t.Errorf("status/payment = %q/%q, want %q/%q", got.Status, got.PaymentStatus, StatusCancelled, PaymentUnpaid)
	}
}

func TestListExpiredAwaiting(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	ended := insertAwaiting(t, repo)
	if _, err := repo.db.ExecContext(ctx,
		"UPDATE orders SET expires_at = DATEADD(MINUTE, -1, SYSUTCDATETIME()) WHERE id = @id",
		sql.Named("id", ended.ID)); err != nil {
		t.Fatal(err)
	}
	onHold := insertAwaiting(t, repo)

	// Other tests leave expired orders behind, so look for ours among many.
	orders, err := repo.ListExpiredAwaiting(ctx, 10000)
	if err != nil {
		t.Fatal(err)
	}

	var found *Order
	for i := range orders {
		switch orders[i].ID {
		case ended.ID:
			found = &orders[i]
		case onHold.ID:
			t.Error("an order still on hold was listed")
		}
		if orders[i].Status != StatusAwaitingPayment {
			t.Errorf("order %s is %s, want only AWAITING_PAYMENT", orders[i].Code, orders[i].Status)
		}
	}
	if found == nil {
		t.Fatal("the order whose hold ended was not listed")
	}
	if len(found.Items) != 2 || found.Items[0].BookID != 1 {
		t.Errorf("items = %+v, want the two inserted items", found.Items)
	}
}

func TestListExpiredAwaitingHonoursTheLimit(t *testing.T) {
	repo := newTestRepo(t)

	orders, err := repo.ListExpiredAwaiting(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) > 1 {
		t.Errorf("%d orders, want at most 1", len(orders))
	}
}

// A late failure must not undo a payment that a parallel request confirmed.
func TestCancelAwaitingLeavesAPaidOrderAlone(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	o := insertAwaiting(t, repo)
	if err := repo.MarkPaid(ctx, o.ID, "14567890"); err != nil {
		t.Fatal(err)
	}

	if err := repo.CancelAwaiting(ctx, o.ID, "payment failed"); !errors.Is(err, ErrNotAwaitingPayment) {
		t.Errorf("err = %v, want ErrNotAwaitingPayment", err)
	}

	got, err := repo.FindByCode(ctx, o.Code)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusConfirmed || got.PaymentStatus != PaymentPaid {
		t.Errorf("status/payment = %q/%q: the paid order was changed", got.Status, got.PaymentStatus)
	}
}

func TestListByUserShowsOnlyTheCustomersOrders(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := "it-user-" + newKey()
	other := "it-user-" + newKey()

	insert := func(owner *string, status, method, paymentStatus string) *Order {
		t.Helper()
		o := newTestOrder(t, repo)
		o.UserID, o.Status, o.PaymentMethod, o.PaymentStatus = owner, status, method, paymentStatus
		if status == StatusAwaitingPayment {
			until := time.Now().Add(time.Hour)
			o.ExpiresAt = &until
		}
		if err := repo.Insert(ctx, newKey(), o); err != nil {
			t.Fatal(err)
		}
		return o
	}

	older := insert(&user, StatusConfirmed, PaymentCOD, PaymentUnpaid)
	// A paid order is written awaiting payment and then paid, as in checkout.
	paid := insert(&user, StatusAwaitingPayment, PaymentCard, PaymentUnpaid)
	if err := repo.MarkPaid(ctx, paid.ID, "it-ref-"+paid.Code); err != nil {
		t.Fatal(err)
	}
	insert(&user, StatusAwaitingPayment, PaymentVietQR, PaymentUnpaid) // not paid yet
	insert(&user, StatusCancelled, PaymentCard, PaymentUnpaid)         // abandoned payment
	insert(&other, StatusConfirmed, PaymentCOD, PaymentUnpaid)
	insert(nil, StatusConfirmed, PaymentCOD, PaymentUnpaid)

	got, err := repo.ListByUser(ctx, user, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0].Code != paid.Code || got[1].Code != older.Code {
		codes := make([]string, len(got))
		for i, o := range got {
			codes[i] = o.Code
		}
		t.Fatalf("listed %v, want [%s %s] (newest first, only orders the customer got)", codes, paid.Code, older.Code)
	}
	if len(got[0].Items) != 2 {
		t.Errorf("items = %+v, want the two books", got[0].Items)
	}

	none, err := repo.ListByUser(ctx, "it-user-nobody", 10)
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("no orders = %v, %v; want an empty list", none, err)
	}
}

func TestLinkGuestOrders(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	email := "IT-" + newKey() + "@Example.com"
	user, other := "it-user-"+newKey(), "it-user-"+newKey()

	insert := func(owner *string) *Order {
		t.Helper()
		o := newTestOrder(t, repo)
		o.Email, o.UserID = email, owner
		if err := repo.Insert(ctx, newKey(), o); err != nil {
			t.Fatal(err)
		}
		return o
	}

	guest := insert(nil)
	theirs := insert(&other)

	n, err := repo.LinkGuestOrders(ctx, user, strings.ToLower(email))
	if err != nil || n != 1 {
		t.Fatalf("LinkGuestOrders = %d, %v; want 1", n, err)
	}

	if got, _ := repo.FindByCode(ctx, guest.Code); got.UserID == nil || *got.UserID != user {
		t.Errorf("guest order user = %v, want %s", got.UserID, user)
	}
	if got, _ := repo.FindByCode(ctx, theirs.Code); got.UserID == nil || *got.UserID != other {
		t.Errorf("someone else's order moved to %v", got.UserID)
	}

	if n, _ := repo.LinkGuestOrders(ctx, user, email); n != 0 {
		t.Errorf("second call linked %d, want 0", n)
	}
}
