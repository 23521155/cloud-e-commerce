//go:build integration

package order

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"testing"

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
