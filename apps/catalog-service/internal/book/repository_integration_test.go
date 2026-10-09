//go:build integration

package book

import (
	"context"
	"os"
	"testing"

	"github.com/fizzisme/catalog-service/internal/database"
	"github.com/fizzisme/catalog-service/internal/seeddata"
)

// Run with the seeded database up:
//
//	DATABASE_URL=sqlserver://... go test -tags integration ./internal/book/
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

func TestListPagingAndTotal(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	p1, total, err := repo.List(ctx, Filter{Sort: "new", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	p2, _, err := repo.List(ctx, Filter{Sort: "new", Page: 2})
	if err != nil {
		t.Fatal(err)
	}

	if total < 9000 || total > 9172 {
		t.Errorf("total = %d, want the seeded catalogue", total)
	}
	if len(p1) != PageSize || len(p2) != PageSize {
		t.Fatalf("page sizes = %d, %d; want %d", len(p1), len(p2), PageSize)
	}
	if p1[PageSize-1].ID <= p2[0].ID {
		t.Errorf("pages overlap or are out of order: %d then %d", p1[PageSize-1].ID, p2[0].ID)
	}
}

func TestListSorts(t *testing.T) {
	repo := newTestRepo(t)

	asc, _, err := repo.List(context.Background(), Filter{Sort: "price-asc", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(asc); i++ {
		if asc[i].PriceVND < asc[i-1].PriceVND {
			t.Errorf("price-asc not ascending at %d", i)
		}
	}

	oldest, _, err := repo.List(context.Background(), Filter{Sort: "year", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if oldest[0].Year < seeddata.MinYear || oldest[0].Year > 1850 {
		t.Errorf("oldest book is %d, want a plausible early 19th-century year", oldest[0].Year)
	}
	for i := 1; i < len(oldest); i++ {
		if oldest[i].Year < oldest[i-1].Year {
			t.Errorf("year sort not ascending at %d", i)
		}
	}
}

func TestListFilters(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	hist, total, err := repo.List(ctx, Filter{Subject: "history", Sort: "new", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total == 0 {
		t.Fatal("no history books")
	}
	for _, b := range hist {
		if b.SubjectSlug != "history" {
			t.Errorf("%s has subject %s", b.Slug, b.SubjectSlug)
		}
	}

	rare, _, err := repo.List(ctx, Filter{Rare: true, Sort: "new", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range rare {
		if !b.Rare || b.Year >= 1900 {
			t.Errorf("%s is not rare (year %d)", b.Slug, b.Year)
		}
	}

	found, _, err := repo.List(ctx, Filter{Terms: []string{"sherlock", "holmes"}, Sort: "new", Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Error("search found nothing for sherlock holmes")
	}
}

func TestListEscapesLikeWildcards(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	_, all, err := repo.List(ctx, Filter{Sort: "new", Page: 1})
	if err != nil {
		t.Fatal(err)
	}

	for _, term := range []string{"%", "_", "[a-z]"} {
		_, total, err := repo.List(ctx, Filter{Terms: []string{term}, Sort: "new", Page: 1})
		if err != nil {
			t.Fatal(err)
		}
		if total >= all {
			t.Errorf("term %q matched everything (%d of %d): wildcard not escaped", term, total, all)
		}
	}
}

func TestGetBySlugAndRelated(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	const slug = "the-complete-sherlock-holmes-b00005vo0t"

	b, err := repo.GetBySlug(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "The Complete Sherlock Holmes" || b.SubjectSlug == "" {
		t.Errorf("unexpected book: %+v", b)
	}

	if _, err := repo.GetBySlug(ctx, "does-not-exist"); err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}

	related, err := repo.Related(ctx, slug, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(related) != 4 {
		t.Fatalf("related = %d, want 4", len(related))
	}
	for _, r := range related {
		if r.Slug == slug {
			t.Error("related includes the book itself")
		}
		if r.SubjectSlug != b.SubjectSlug {
			t.Errorf("related %s has subject %s, want %s first", r.Slug, r.SubjectSlug, b.SubjectSlug)
		}
	}
}

func TestBySlugs(t *testing.T) {
	repo := newTestRepo(t)

	books, err := repo.BySlugs(context.Background(), []string{
		"the-complete-sherlock-holmes-b00005vo0t",
		"khong-co",
		"the-complete-sherlock-holmes-b000k04qd2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 2 {
		t.Fatalf("got %d books, want 2 (the unknown slug is left out)", len(books))
	}

	none, err := repo.BySlugs(context.Background(), nil)
	if err != nil || len(none) != 0 {
		t.Errorf("empty input: %v, %v", none, err)
	}
}
