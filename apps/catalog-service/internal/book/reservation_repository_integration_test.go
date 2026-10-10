//go:build integration

package book

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
)

// freeBookIDs returns n ascending ids of books nobody holds, so the test
// does not depend on which ids the seed produced.
func freeBookIDs(t *testing.T, repo *Repository, n int) []int {
	t.Helper()

	var ids []int
	err := repo.db.SelectContext(context.Background(), &ids,
		"SELECT TOP (@n) id FROM books WHERE status = N'AVAILABLE' ORDER BY id DESC", sql.Named("n", n))
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) < n {
		t.Fatalf("need %d free books, found %d", n, len(ids))
	}

	return normalise(ids)
}

// freeAll puts the test books back, whatever state a failed run left them in.
func freeAll(t *testing.T, repo *Repository, ids []int) {
	t.Helper()

	list, args := idList(ids)
	_, err := repo.db.ExecContext(context.Background(),
		"UPDATE books SET status = N'AVAILABLE', reserved_by = NULL, reserved_until = NULL WHERE id IN ("+list+")", args...)
	if err != nil {
		t.Fatal(err)
	}
}

func statusOf(t *testing.T, repo *Repository, id int) string {
	t.Helper()

	var status string
	err := repo.db.GetContext(context.Background(), &status,
		"SELECT status FROM books WHERE id = @id", sql.Named("id", id))
	if err != nil {
		t.Fatal(err)
	}

	return status
}

// Two carts that share one book, ordered at the same instant: exactly one
// wins, and the loser must not keep the books it did not share.
func TestOverlappingCartsOnlyOneWins(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	ids := freeBookIDs(t, repo, 5) // a b c d e
	t.Cleanup(func() { freeAll(t, repo, ids) })

	cartA := []int{ids[0], ids[1], ids[2]}
	cartB := []int{ids[2], ids[3], ids[4]}

	for round := 0; round < 20; round++ {
		freeAll(t, repo, ids)

		var wg sync.WaitGroup
		errs := make([]error, 2)

		for i, order := range []struct {
			holder string
			cart   []int
		}{{"RACE-A", cartA}, {"RACE-B", cartB}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = repo.Reserve(ctx, order.holder, 60, order.cart)
			}()
		}
		wg.Wait()

		wins := 0
		for _, err := range errs {
			if err == nil {
				wins++
			}
		}
		if wins != 1 {
			t.Fatalf("round %d: %d orders won, want exactly 1 (errors: %v)", round, wins, errs)
		}

		loser, loserCart := 0, cartA
		if errs[0] == nil {
			loser, loserCart = 1, cartB
		}

		var unavailable *UnavailableError
		if !errors.As(errs[loser], &unavailable) {
			t.Fatalf("round %d: loser error = %v, want UnavailableError", round, errs[loser])
		}
		if len(unavailable.Books) != 1 || unavailable.Books[0].ID != ids[2] || unavailable.Books[0].Status != StatusReserved {
			t.Fatalf("round %d: unavailable = %+v, want only the shared book, RESERVED", round, unavailable.Books)
		}

		// The loser's own books (not shared) must still be free.
		for _, id := range loserCart {
			if id == ids[2] {
				continue
			}
			if got := statusOf(t, repo, id); got != StatusAvailable {
				t.Fatalf("round %d: book %d is %s after a failed reserve, want AVAILABLE", round, id, got)
			}
		}
	}
}

func TestHoldLifecycle(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	ids := freeBookIDs(t, repo, 2)
	t.Cleanup(func() { freeAll(t, repo, ids) })

	held, err := repo.Reserve(ctx, "LIFE-1", 60, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 2 || held[0].PriceVND <= 0 || held[0].Slug == "" {
		t.Fatalf("held = %+v, want a priced snapshot of both books", held)
	}

	// The same holder may repeat the request; nobody else may take the books.
	if _, err := repo.Reserve(ctx, "LIFE-1", 60, ids); err != nil {
		t.Errorf("same holder reserving again: %v", err)
	}
	if _, err := repo.Reserve(ctx, "LIFE-2", 60, ids[:1]); err == nil {
		t.Error("another holder reserved a held book")
	}

	// A stranger can neither release nor sell them.
	if n, err := repo.Release(ctx, "LIFE-2", ids); err != nil || n != 0 {
		t.Errorf("stranger release = %d, %v; want 0, nil", n, err)
	}
	if err := repo.Sell(ctx, "LIFE-2", ids); err == nil {
		t.Error("a stranger sold held books")
	}

	// The holder sells; repeating the sale is fine, releasing a sold book is not possible.
	if err := repo.Sell(ctx, "LIFE-1", ids); err != nil {
		t.Fatal(err)
	}
	if err := repo.Sell(ctx, "LIFE-1", ids); err != nil {
		t.Errorf("selling twice: %v", err)
	}
	if n, err := repo.Release(ctx, "LIFE-1", ids); err != nil || n != 0 {
		t.Errorf("release of sold books = %d, %v; want 0, nil", n, err)
	}
	if got := statusOf(t, repo, ids[0]); got != StatusSold {
		t.Errorf("status = %s, want SOLD", got)
	}

	var unavailable *UnavailableError
	_, err = repo.Reserve(ctx, "LIFE-3", 60, ids)
	if !errors.As(err, &unavailable) || unavailable.Books[0].Status != StatusSold {
		t.Errorf("reserving sold books: %v, want UnavailableError with SOLD", err)
	}
}

func TestExpiredHoldsAreFreed(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	ids := freeBookIDs(t, repo, 2)
	t.Cleanup(func() { freeAll(t, repo, ids) })

	if _, err := repo.Reserve(ctx, "EXP-1", 60, ids); err != nil {
		t.Fatal(err)
	}

	// Age the hold instead of sleeping.
	list, args := idList(ids)
	if _, err := repo.db.ExecContext(ctx,
		"UPDATE books SET reserved_until = DATEADD(MINUTE, -1, SYSUTCDATETIME()) WHERE id IN ("+list+")", args...); err != nil {
		t.Fatal(err)
	}

	// Another order may take an expired hold straight away ...
	if _, err := repo.Reserve(ctx, "EXP-2", 60, ids[:1]); err != nil {
		t.Errorf("reserving an expired hold: %v", err)
	}

	// ... and the sweep frees the rest, leaving the live hold alone.
	if _, err := repo.ReleaseExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if got := statusOf(t, repo, ids[0]); got != StatusReserved {
		t.Errorf("live hold is %s, want RESERVED", got)
	}
	if got := statusOf(t, repo, ids[1]); got != StatusAvailable {
		t.Errorf("expired hold is %s, want AVAILABLE", got)
	}
}
