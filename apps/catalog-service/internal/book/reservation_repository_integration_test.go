//go:build integration

package book

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
)

// testBook is a book the tests borrow: its id (to hold and sell with) and its
// slug (to reserve with).
type testBook struct {
	ID   int    `db:"id"`
	Slug string `db:"slug"`
}

// freeBooks returns n books nobody holds, so the test does not depend on
// which books the seed produced.
func freeBooks(t *testing.T, repo *Repository, n int) []testBook {
	t.Helper()

	var books []testBook
	err := repo.db.SelectContext(context.Background(), &books,
		"SELECT TOP (@n) id, slug FROM books WHERE status = N'AVAILABLE' ORDER BY id DESC", sql.Named("n", n))
	if err != nil {
		t.Fatal(err)
	}
	if len(books) < n {
		t.Fatalf("need %d free books, found %d", n, len(books))
	}

	return books
}

func idsOf(books []testBook) []int {
	ids := make([]int, len(books))
	for i, b := range books {
		ids[i] = b.ID
	}
	return normalise(ids)
}

// slugsOf returns the slugs sorted, as Repository.Reserve expects.
func slugsOf(books []testBook) []string {
	slugs := make([]string, len(books))
	for i, b := range books {
		slugs[i] = b.Slug
	}
	return normalise(slugs)
}

// freeAll puts the test books back, whatever state a failed run left them in.
func freeAll(t *testing.T, repo *Repository, books []testBook) {
	t.Helper()

	list, args := idList(idsOf(books))
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

	books := freeBooks(t, repo, 5) // a b c d e
	t.Cleanup(func() { freeAll(t, repo, books) })

	shared := books[2]
	cartA := []testBook{books[0], books[1], books[2]}
	cartB := []testBook{books[2], books[3], books[4]}

	for round := 0; round < 20; round++ {
		freeAll(t, repo, books)

		var wg sync.WaitGroup
		errs := make([]error, 2)

		for i, order := range []struct {
			holder string
			cart   []testBook
		}{{"RACE-A", cartA}, {"RACE-B", cartB}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = repo.Reserve(ctx, order.holder, 60, slugsOf(order.cart))
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
		if len(unavailable.Books) != 1 ||
			unavailable.Books[0].ID != shared.ID ||
			unavailable.Books[0].Slug != shared.Slug ||
			unavailable.Books[0].Status != StatusReserved {
			t.Fatalf("round %d: unavailable = %+v, want only the shared book, RESERVED", round, unavailable.Books)
		}

		// The loser's own books (not shared) must still be free.
		for _, b := range loserCart {
			if b.ID == shared.ID {
				continue
			}
			if got := statusOf(t, repo, b.ID); got != StatusAvailable {
				t.Fatalf("round %d: book %d is %s after a failed reserve, want AVAILABLE", round, b.ID, got)
			}
		}
	}
}

func TestHoldLifecycle(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	books := freeBooks(t, repo, 2)
	t.Cleanup(func() { freeAll(t, repo, books) })

	ids, slugs := idsOf(books), slugsOf(books)

	held, err := repo.Reserve(ctx, "LIFE-1", 60, slugs)
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 2 || held[0].PriceVND <= 0 || held[0].Slug == "" || held[0].ID == 0 {
		t.Fatalf("held = %+v, want a priced snapshot of both books, with id and slug", held)
	}

	// The same holder may repeat the request; nobody else may take the books.
	if _, err := repo.Reserve(ctx, "LIFE-1", 60, slugs); err != nil {
		t.Errorf("same holder reserving again: %v", err)
	}
	if _, err := repo.Reserve(ctx, "LIFE-2", 60, slugs[:1]); err == nil {
		t.Error("another holder reserved a held book")
	}

	// A stranger can neither release nor sell them, and the refusal names the
	// books by slug as well as by id.
	if n, err := repo.Release(ctx, "LIFE-2", ids); err != nil || n != 0 {
		t.Errorf("stranger release = %d, %v; want 0, nil", n, err)
	}
	var unavailable *UnavailableError
	if err := repo.Sell(ctx, "LIFE-2", ids); !errors.As(err, &unavailable) {
		t.Errorf("a stranger sold held books: %v", err)
	} else if len(unavailable.Books) != 2 || unavailable.Books[0].Slug == "" || unavailable.Books[0].ID == 0 {
		t.Errorf("sell refusal = %+v, want both books with id and slug", unavailable.Books)
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

	_, err = repo.Reserve(ctx, "LIFE-3", 60, slugs)
	if !errors.As(err, &unavailable) || unavailable.Books[0].Status != StatusSold {
		t.Errorf("reserving sold books: %v, want UnavailableError with SOLD", err)
	}
}

// A slug that is not in the catalogue is reported by slug (its id is not
// known), and nothing else in the request is held.
func TestReserveReportsAMissingSlug(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	books := freeBooks(t, repo, 1)
	t.Cleanup(func() { freeAll(t, repo, books) })

	slugs := normalise([]string{books[0].Slug, "no-such-book-zzzz"})

	_, err := repo.Reserve(ctx, "MISS-1", 60, slugs)

	var unavailable *UnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("err = %v, want UnavailableError", err)
	}
	want := Unavailable{Slug: "no-such-book-zzzz", Status: StatusMissing}
	if len(unavailable.Books) != 1 || unavailable.Books[0] != want {
		t.Fatalf("unavailable = %+v, want only %+v", unavailable.Books, want)
	}
	if got := statusOf(t, repo, books[0].ID); got != StatusAvailable {
		t.Fatalf("the existing book is %s after a failed reserve, want AVAILABLE", got)
	}
}

// The database ignores case in a slug, and so must a request: asking for the
// same book twice in different spellings is one book.
func TestReserveMatchesSlugsWithoutRegardToCase(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	books := freeBooks(t, repo, 1)
	t.Cleanup(func() { freeAll(t, repo, books) })

	svc := NewReservationService(repo)

	res, err := svc.Reserve(ctx, "CASE-1", 60, []string{strings.ToUpper(books[0].Slug), books[0].Slug})
	if err != nil {
		t.Fatalf("the same book in two spellings was refused: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != books[0].ID {
		t.Fatalf("items = %+v, want the one book", res.Items)
	}
}

func TestExpiredHoldsAreFreed(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	books := freeBooks(t, repo, 2)
	t.Cleanup(func() { freeAll(t, repo, books) })

	ids, slugs := idsOf(books), slugsOf(books)

	if _, err := repo.Reserve(ctx, "EXP-1", 60, slugs); err != nil {
		t.Fatal(err)
	}

	// Age the hold instead of sleeping.
	list, args := idList(ids)
	if _, err := repo.db.ExecContext(ctx,
		"UPDATE books SET reserved_until = DATEADD(MINUTE, -1, SYSUTCDATETIME()) WHERE id IN ("+list+")", args...); err != nil {
		t.Fatal(err)
	}

	// Another order may take an expired hold straight away ...
	taken, err := repo.Reserve(ctx, "EXP-2", 60, slugs[:1])
	if err != nil {
		t.Fatalf("reserving an expired hold: %v", err)
	}

	// ... and the sweep frees the rest, leaving the live hold alone.
	if _, err := repo.ReleaseExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if got := statusOf(t, repo, taken[0].ID); got != StatusReserved {
		t.Errorf("live hold is %s, want RESERVED", got)
	}
	for _, b := range books {
		if b.ID == taken[0].ID {
			continue
		}
		if got := statusOf(t, repo, b.ID); got != StatusAvailable {
			t.Errorf("expired hold is %s, want AVAILABLE", got)
		}
	}
}
